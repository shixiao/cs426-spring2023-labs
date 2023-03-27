package kvtest

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"cs426.yale.edu/lab4/kv"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client tests use a mocked out implementation to make sure your
// Kv (client.go) implementation works correctly, even if the server
// is not implemented.

func TestClientGetSingleNode(t *testing.T) {
	// Simplest test of Kv in client.go -- `Get` calls need to send a gRPC Get()
	// call. We mock out the server implementation here, so this only tests your
	// client.go logic -- this setup only has a single server and a single shard.
	setup := MakeTestSetupWithoutServers(MakeBasicOneShard())
	setup.clientPool.OverrideGetResponse("n1", "found!", true)

	val, wasFound, err := setup.Get("abc123")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "found!", val)

	setup.clientPool.OverrideGetResponse("n1", "", false)
	_, wasFound, err = setup.Get("abc123")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	setup.clientPool.OverrideRpcError("n1", status.Errorf(codes.Aborted, "oh no!"))
	_, _, err = setup.Get("abc123")
	assert.NotNil(t, err)
}

func TestClientSetDeleteSingleNode(t *testing.T) {
	// Similar to TestClientGetSingleNode: one node, one shard,
	// testing that Set/Delete RPCs are sent.
	setup := MakeTestSetupWithoutServers(MakeBasicOneShard())
	setup.clientPool.OverrideDeleteResponse("n1")
	setup.clientPool.OverrideSetResponse("n1")

	err := setup.Set("abc", "123", 10*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, 1, setup.clientPool.GetRequestsSent("n1"))

	err = setup.Delete("abc")
	assert.Nil(t, err)
	assert.Equal(t, 2, setup.clientPool.GetRequestsSent("n1"))
}

func TestClientGetClientError(t *testing.T) {
	// Tests basic error handling -- if ClientPool.GetClient() returns
	// an error your client code should return that error back to the user.
	setup := MakeTestSetupWithoutServers(MakeBasicOneShard())
	setup.clientPool.OverrideGetClientError("n1", errors.New("oh no!"))

	_, _, err := setup.Get("abc123")
	assert.NotNil(t, err)

	err = setup.Set("abc123", "val", 10*time.Second)
	assert.NotNil(t, err)

	err = setup.Delete("abc123")
	assert.NotNil(t, err)
}

func TestClientNoShardsAssigned(t *testing.T) {
	// If the shard for the key in question is not assigned to any server
	// you must return an error (and not crash). This could happen in a production
	// setting due to operator error or servers being drained too quickly.
	setup := MakeTestSetupWithoutServers(MakeNoShardAssigned())

	_, _, err := setup.Get("abc123")
	assert.NotNil(t, err)

	err = setup.Set("abc123", "a", 1*time.Second)
	assert.NotNil(t, err)

	err = setup.Delete("abc123")
	assert.NotNil(t, err)
}

func TestClientMultiShardSingleNode(t *testing.T) {
	// Starting to test the logic of the client: we setup multiple
	// shards on a single node and spread the keys among them.
	//
	// This test does not check particular sharding logic because
	// you are free to implement your own.
	setup := MakeTestSetupWithoutServers(MakeMultiShardSingleNode())

	keys := RandomKeys(100, 5)
	setup.clientPool.OverrideSetResponse("n1")
	for _, key := range keys {
		err := setup.Set(key, "", 1*time.Second)
		assert.Nil(t, err)
	}
	setup.clientPool.OverrideGetResponse("n1", "found!", true)
	for _, key := range keys {
		val, wasFound, err := setup.Get(key)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, "found!", val)
	}
	// should be 100 sets + 100 gets
	assert.Equal(t, 200, setup.clientPool.GetRequestsSent("n1"))
}

func TestClientMultiNodeGet(t *testing.T) {
	// With multiple nodes hosting the same shard, your implementation
	// has some freedom to pick. In this case we only do a single Get, so
	// reading from either n1 or n2 is valid.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeBothAssignedSingleShard())

	setup.clientPool.OverrideGetResponse("n1", "val", true)
	setup.clientPool.OverrideGetResponse("n2", "val", true)

	val, wasFound, err := setup.Get("abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "val", val)
}

func TestClientMultiNodeSetDelete(t *testing.T) {
	// Tests fan-out of Set and Delete calls. If multiple nodes
	// host the same shard, your client logic must send the RPCs
	// to all nodes instead of just one.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeBothAssignedSingleShard())
	setup.clientPool.OverrideSetResponse("n1")
	setup.clientPool.OverrideSetResponse("n2")

	err := setup.Set("abc", "123", 1*time.Second)
	assert.Nil(t, err)

	// Set requests should go to both replicas
	assert.Equal(t, 1, setup.clientPool.GetRequestsSent("n1"))
	assert.Equal(t, 1, setup.clientPool.GetRequestsSent("n2"))

	setup.clientPool.OverrideDeleteResponse("n1")
	setup.clientPool.OverrideDeleteResponse("n2")
	err = setup.Delete("abc")
	assert.Nil(t, err)

	// Same for deletes
	assert.Equal(t, 2, setup.clientPool.GetRequestsSent("n1"))
	assert.Equal(t, 2, setup.clientPool.GetRequestsSent("n2"))
}

func TestClientMultiNodeGetLoadBalance(t *testing.T) {
	// Tests load balancing of Get() calls -- if we have two nodes
	// and send several requests to the same key, your client
	// should attempt to spread the load of those Get() calls among
	// the two replicas.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeBothAssignedSingleShard())

	setup.clientPool.OverrideGetResponse("n1", "val", true)
	setup.clientPool.OverrideGetResponse("n2", "val", true)

	for i := 0; i < 100; i++ {
		val, wasFound, err := setup.Get("abc")
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, "val", val)
	}
	n1Rpc := setup.clientPool.GetRequestsSent("n1")
	assert.Less(t, 35, n1Rpc)
	assert.Greater(t, 65, n1Rpc)

	n2Rpc := setup.clientPool.GetRequestsSent("n2")
	assert.Less(t, 35, n2Rpc)
	assert.Greater(t, 65, n2Rpc)
}

func TestClientMultiNodeGetFailover(t *testing.T) {
	// Similar to MultiNodeGetLoadBalance -- if we have two nodes
	// with the same shard, your client should retry failed requests
	// on further nodes.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeBothAssignedSingleShard())

	setup.clientPool.OverrideRpcError("n1", status.Errorf(codes.Aborted, "oh no!"))
	setup.clientPool.OverrideGetResponse("n2", "ok!", true)

	for i := 0; i < 10; i++ {
		val, wasFound, err := setup.Get("abc")
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, "ok!", val)
	}
	// At least one attempt should be made to N1
	assert.Less(t, 1, setup.clientPool.GetRequestsSent("n1"))
	// All attempts eventually go to N2 to succeed
	assert.Equal(t, 10, setup.clientPool.GetRequestsSent("n2"))

	// Now flip, so that N1 succeeds and N2 errors
	setup.clientPool.ClearRpcOverrides("n1")
	setup.clientPool.ClearRpcOverrides("n2")
	setup.clientPool.OverrideGetResponse("n1", "ok!", true)
	setup.clientPool.OverrideRpcError("n2", status.Errorf(codes.Aborted, "oh no!"))

	for i := 0; i < 10; i++ {
		val, wasFound, err := setup.Get("abc")
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, "ok!", val)
	}
}

func TestClientMultiNodeGetFullFailure(t *testing.T) {
	// If all nodes return an error, return an error to the client.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeBothAssignedSingleShard())
	setup.clientPool.OverrideRpcError("n1", status.Errorf(codes.Aborted, "oh no!"))
	setup.clientPool.OverrideRpcError("n2", status.Errorf(codes.Aborted, "oh no!"))
	_, _, err := setup.Get("abc")
	assert.NotNil(t, err)

	// RPCs sent to both nodes
	assert.Equal(t, 1, setup.clientPool.GetRequestsSent("n1"))
	assert.Equal(t, 1, setup.clientPool.GetRequestsSent("n2"))
}

func TestClientMultiNodeSetDeletePartialFailure(t *testing.T) {
	// If multiple nodes host the same shard and only some of them
	// fail the RPC, you must report that failure to a user. Set/Delete
	// calls are only successful if *all* nodes return success.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeBothAssignedSingleShard())
	setup.clientPool.OverrideSetResponse("n1")
	setup.clientPool.OverrideRpcError("n2", status.Errorf(codes.Aborted, "oh no!"))

	for i := 0; i < 10; i++ {
		err := setup.Set(fmt.Sprintf("key:%d", i), "abc", 1*time.Second)
		assert.NotNil(t, err)
	}
	assert.Equal(t, 10, setup.clientPool.GetRequestsSent("n1"))
	assert.Equal(t, 10, setup.clientPool.GetRequestsSent("n2"))

	setup.clientPool.OverrideDeleteResponse("n1")
	for i := 0; i < 10; i++ {
		err := setup.Delete(fmt.Sprintf("key:%d", i))
		assert.NotNil(t, err)
	}
	assert.Equal(t, 20, setup.clientPool.GetRequestsSent("n1"))
	assert.Equal(t, 20, setup.clientPool.GetRequestsSent("n2"))
}

func TestClientMultiNodeSetDeleteDoneConcurrently(t *testing.T) {
	// Your client should fan out to all nodes for a shard *concurrently*,
	// not in serial.
	//
	// We naively test this with latency injection -- we make all calls
	// take 300ms, and ensure the overall latency does not stack up
	// when your client must fan out to 10 nodes.
	setup := MakeTestSetupWithoutServers(kv.ShardMapState{
		NumShards: 1,
		Nodes:     makeNodeInfos(10),
		ShardsToNodes: map[int][]string{
			1: {"n1", "n2", "n3", "n4", "n5", "n6", "n7", "n8", "n9", "n10"},
		},
	})
	for n := 1; n <= 10; n++ {
		nodeName := fmt.Sprintf("n%d", n)
		setup.clientPool.OverrideSetResponse(nodeName)
		setup.clientPool.OverrideDeleteResponse(nodeName)
		setup.clientPool.AddLatencyInjection(nodeName, 300*time.Millisecond)
	}

	start := time.Now()
	err := setup.Set("abc", "123", 1*time.Second)
	assert.Nil(t, err)

	// The request should only take ~300ms, because we send to all 10 nodes
	// concurrently and each node only waits ~300ms.
	latency := time.Since(start)
	assert.Greater(t, 800*time.Millisecond, latency)

	// Repeat the process for Delete() with the same semantics
	start = time.Now()
	err = setup.Delete("abc")
	assert.Nil(t, err)

	latency = time.Since(start)
	assert.Greater(t, 800*time.Millisecond, latency)
}

func TestClientMultiNodeMultiShard(t *testing.T) {
	// Tests actual sharding logic -- if we send
	// a Set(key1), then a Get(key1) should go to the same
	// shard.
	//
	// This setup has two nodes, each with 5 different shards, and they don't
	// have any shards in common. If a Set request went to node1, then
	// the Get request must also go to node1 because it is the host of that
	// shard exclusively.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeMultiShard())

	keys := RandomKeys(1000, 5)
	setup.clientPool.OverrideSetResponse("n1")
	setup.clientPool.OverrideSetResponse("n2")

	keysToNodes := make(map[string]string)
	for _, key := range keys {
		err := setup.Set(key, "v", 1*time.Second)
		assert.Nil(t, err)

		// Keep track of where Set requests went to
		if setup.clientPool.GetRequestsSent("n1") > 0 {
			keysToNodes[key] = "n1"
		} else {
			keysToNodes[key] = "n2"
		}

		// Reset the counters to make it easier to keep track
		setup.clientPool.ClearRequestsSent("n1")
		setup.clientPool.ClearRequestsSent("n2")
	}

	setup.clientPool.OverrideGetResponse("n1", "from n1", true)
	setup.clientPool.OverrideGetResponse("n2", "from n2", true)
	for _, key := range keys {
		val, wasFound, err := setup.Get(key)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		if keysToNodes[key] == "n1" {
			assert.Equal(t, "from n1", val)
		} else {
			assert.Equal(t, "from n2", val)
		}
	}
}

func TestClientMultiNodeMultiShardSwap(t *testing.T) {
	// Similar to TestClientMultiNodeMultiShard, but does a shard
	// swap in the middle. If your client caches any state from the
	// shard map it must be updated when shards are moved.
	//
	// In this case we move all the shards on n1 to n2, and all the shards
	// on n2 to n1. This means any Set() calls that *originally* went to n1
	// before the swap must be on n2 now (and vice-versa).
	setup := MakeTestSetupWithoutServers(MakeTwoNodeMultiShard())

	keys := RandomKeys(1000, 5)
	setup.clientPool.OverrideSetResponse("n1")
	setup.clientPool.OverrideSetResponse("n2")

	keysToOriginalNodes := make(map[string]string)
	for _, key := range keys {
		err := setup.Set(key, "v", 1*time.Second)
		assert.Nil(t, err)
		if setup.clientPool.GetRequestsSent("n1") > 0 {
			keysToOriginalNodes[key] = "n1"
		} else {
			keysToOriginalNodes[key] = "n2"
		}
		setup.clientPool.ClearRequestsSent("n1")
		setup.clientPool.ClearRequestsSent("n2")
	}

	setup.UpdateShardMapping(map[int][]string{
		1:  {"n2"},
		2:  {"n2"},
		3:  {"n2"},
		4:  {"n2"},
		5:  {"n2"},
		6:  {"n1"},
		7:  {"n1"},
		8:  {"n1"},
		9:  {"n1"},
		10: {"n1"},
	})

	setup.clientPool.OverrideGetResponse("n1", "from n1", true)
	setup.clientPool.OverrideGetResponse("n2", "from n2", true)
	for _, key := range keys {
		val, wasFound, err := setup.Get(key)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		// If it was originally written to n1, it should be routed to n2 now after shard swap
		if keysToOriginalNodes[key] == "n1" {
			assert.Equal(t, "from n2", val)
		} else {
			assert.Equal(t, "from n1", val)
		}
	}
}

func TestClientConcurrentGetSet(t *testing.T) {
	// Tests basic concurrency -- we send lots of Sets() and Gets()
	// from multiple goroutines. Run with the -race flag to check
	// for any concurrency bugs.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeMultiShard())
	setup.clientPool.OverrideSetResponse("n1")
	setup.clientPool.OverrideSetResponse("n2")
	setup.clientPool.OverrideGetResponse("n1", "val", true)
	setup.clientPool.OverrideGetResponse("n2", "val", true)

	keys := RandomKeys(1000, 5)

	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 2
	iters := 10
	for i := 0; i < goros; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < iters; j++ {
				for _, key := range keys {
					val, wasFound, err := setup.Get(key)
					assert.Nil(t, err)
					assert.True(t, wasFound)
					assert.Equal(t, "val", val)

					err = setup.Set(key, "val", 1*time.Second)
					assert.Nil(t, err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestClientConcurrentGetSetErrors(t *testing.T) {
	// Similar to above but stresses error handling with concurrency.
	// One node always succeeds and one node always fails.
	setup := MakeTestSetupWithoutServers(MakeTwoNodeMultiShard())
	setup.clientPool.OverrideSetResponse("n1")
	setup.clientPool.OverrideGetResponse("n1", "val", true)
	setup.clientPool.OverrideRpcError("n2", status.Error(codes.Aborted, "oh no!"))

	keys := RandomKeys(1000, 5)

	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 2
	iters := 10
	for i := 0; i < goros; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < iters; j++ {
				for _, key := range keys {
					// Don't care about the response, it might be an error or it may be successful
					// depending on the shard. No failover in this case because the shards
					// are assigned disjoint to nodes.
					//
					// If the Get failed though, the Set call should also fail.
					_, _, err := setup.Get(key)
					setErr := setup.Set(key, "val", 1*time.Second)

					if err != nil {
						assert.NotNil(t, setErr)
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
