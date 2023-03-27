package kvtest

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"cs426.yale.edu/lab4/kv"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RunBasic(t *testing.T, setup *TestSetup) {
	// For a given setup (nodes and shard placement), runs
	// very basic tests -- just get, set, and delete on one key.
	_, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	err = setup.NodeSet("n1", "abc", "123", 5*time.Second)
	assert.Nil(t, err)

	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)
	assert.Nil(t, err)
	val, wasFound, err = setup.NodeGet("n1", "abc")
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)
	assert.Nil(t, err)

	err = setup.NodeDelete("n1", "abc")
	assert.Nil(t, err)

	_, wasFound, err = setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)
}

func RunOverwrite(t *testing.T, setup *TestSetup) {
	// Tests overwriting a single key for a given setup.
	_, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	err = setup.NodeSet("n1", "abc", "123", 5*time.Second)
	assert.Nil(t, err)

	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)
	assert.Nil(t, err)

	err = setup.NodeSet("n1", "abc", "1234", 2*time.Second)
	assert.Nil(t, err)

	val, wasFound, err = setup.NodeGet("n1", "abc")
	assert.True(t, wasFound)
	assert.Equal(t, "1234", val)
	assert.Nil(t, err)
}

func RunMultiKey(t *testing.T, setup *TestSetup) {
	// Runs gets and sets across a few keys for a given setup.
	keys := []string{"a", "b", "c", "d", "e"}
	for _, key := range keys {
		_, wasFound, err := setup.NodeGet("n1", key)
		assert.Nil(t, err)
		assert.False(t, wasFound)
	}

	for i, key := range keys {
		err := setup.NodeSet("n1", key, key+"-value!", 5*time.Second)
		assert.Nil(t, err)

		for _, key := range keys[i+1:] {
			_, wasFound, err := setup.NodeGet("n1", key)
			assert.Nil(t, err)
			assert.False(t, wasFound)
		}

		val, wasFound, err := setup.NodeGet("n1", key)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, key+"-value!", val)
	}

	for _, key := range keys {
		val, wasFound, err := setup.NodeGet("n1", key)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, key+"-value!", val)
	}
}

func RunTtl(t *testing.T, setup *TestSetup) {
	// Tests basic TTL operations on a single key. Your server must not return values if the TTL has
	// past. You do not *necessarily* need to delete the items immediately (and free the items),
	// but you must make sure they are not returned from Get calls if they are past expiry.
	//
	// This test takes some time to run as we wait out the TTL with a bit of buffer so expect it
	// to run a few seconds.
	err := setup.NodeSet("n1", "abc", "123", 500*time.Millisecond)
	assert.Nil(t, err)
	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	time.Sleep(800 * time.Millisecond)

	_, wasFound, err = setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	err = setup.NodeSet("n1", "abc", "124", 500*time.Millisecond)
	assert.Nil(t, err)

	val, wasFound, err = setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "124", val)

	err = setup.NodeSet("n1", "abc", "125", 100*time.Second)
	assert.Nil(t, err)

	time.Sleep(800 * time.Millisecond)
	val, wasFound, err = setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "125", val)

	err = setup.NodeSet("n1", "abc", "126", 100*time.Millisecond)
	assert.Nil(t, err)

	time.Sleep(500 * time.Millisecond)
	_, wasFound, err = setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)
}

func RunTtlMultiKey(t *testing.T, setup *TestSetup) {
	// Tests TTL across multiple keys for a given setup. Similar to RunTtl,
	// you must not return a value from Get() if the TTL has past, even if
	// you do not necessarily clean up the map immediately.
	//
	// Expect this test to run for a second or two.
	keys := RandomKeys(500, 8)
	for i := 0; i < 5; i++ {
		for _, k := range keys {
			err := setup.NodeSet("n1", k, fmt.Sprintf("%s-%d", k, i), 1*time.Second)
			assert.Nil(t, err)
		}
	}
	for _, k := range keys {
		val, wasFound, err := setup.NodeGet("n1", k)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, k+"-4", val)
	}
	time.Sleep(1200 * time.Millisecond)
	for _, k := range keys {
		_, wasFound, err := setup.NodeGet("n1", k)
		assert.Nil(t, err)
		assert.False(t, wasFound)
	}
}

func RunConcurrentGetsSingleKey(t *testing.T, setup *TestSetup) {
	// Simple test for gets only (after a single set) -- your server
	// must be able to handle requests sent concurrently on different goroutines
	err := setup.NodeSet("n1", "abc", "123", 100*time.Second)
	assert.Nil(t, err)
	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 2
	iters := 1000
	for i := 0; i < goros; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < iters; j++ {
				val, wasFound, err := setup.NodeGet("n1", "abc")
				assert.Nil(t, err)
				assert.True(t, wasFound)
				assert.Equal(t, "123", val)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func RunConcurrentGetsMultiKey(t *testing.T, setup *TestSetup) {
	// Similarly to above, your server must be able to handle concurrent
	// requests to different keys.
	keys := RandomKeys(200, 10)
	for _, k := range keys {
		err := setup.NodeSet("n1", k, "1234", 100*time.Second)
		assert.Nil(t, err)
	}

	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 2
	iters := 5
	for i := 0; i < goros; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < iters; j++ {
				for _, k := range keys {
					val, wasFound, err := setup.NodeGet("n1", k)
					assert.Nil(t, err)
					assert.True(t, wasFound)
					assert.Equal(t, "1234", val)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func RunConcurrentGetsAndSets(t *testing.T, setup *TestSetup) {
	// Your server must be able to handle concurrent reads and writes,
	// though you may need to use exclusive locks to handle writes
	// (per-stripe or per-shard preferably).
	keys := RandomKeys(200, 10)
	for _, k := range keys {
		err := setup.NodeSet("n1", k, "abcd", 100*time.Second)
		assert.Nil(t, err)
	}

	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 2
	readIters := 20
	writeIters := 10
	for i := 0; i < goros; i++ {
		wg.Add(2)
		// start a writing goro
		go func() {
			for j := 0; j < writeIters; j++ {
				for _, k := range keys {
					err := setup.NodeSet("n1", k, fmt.Sprintf("abc:%d", j), 100*time.Second)
					assert.Nil(t, err)
				}
			}
			wg.Done()
		}()
		// start a reading goro
		go func() {
			for j := 0; j < readIters; j++ {
				for _, k := range keys {
					val, wasFound, err := setup.NodeGet("n1", k)
					assert.Nil(t, err)
					assert.True(t, wasFound)
					assert.True(t, strings.HasPrefix(val, "abc"))
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestServerBasicNoDelete(t *testing.T) {
	// Most basic server test -- if you've implemented Get() and Set()
	// even without concurrency control, you should pass this test
	setup := MakeTestSetup(MakeBasicOneShard())
	_, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	err = setup.NodeSet("n1", "abc", "123", 5*time.Second)
	assert.Nil(t, err)

	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)
	assert.Nil(t, err)

	setup.Shutdown()
}

func TestServerEmptyKey(t *testing.T) {
	// Tests that your server properly handles the empty key semantics:
	// returning an error with INVALID_ARGUMENT if the key is empty
	setup := MakeTestSetup(MakeBasicOneShard())

	_, _, err := setup.NodeGet("n1", "")
	assertErrorWithCode(t, err, codes.InvalidArgument)

	err = setup.NodeSet("n1", "", "", 1*time.Second)
	assertErrorWithCode(t, err, codes.InvalidArgument)

	err = setup.NodeDelete("n1", "")
	assertErrorWithCode(t, err, codes.InvalidArgument)

	setup.Shutdown()
}

// Test*Unsharded run on a simple setup: one node with one shard
// assigned. You should be able to pass this without considering
// shards at all in your code (but you can implement your initial
// version with shards in mind if you wish).
//
// If you do have dynamic sharding implemented, you must have shards
// assigned and ready to go by the time MakeKvServer returns -- they
// can't be initialized asynchronously (otherwise the tests may fail,
// if your server starts and the shards are not considered assigned).
func TestServerBasicUnsharded(t *testing.T) {
	RunTestWith(t, RunBasic, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerOverwriteUnsharded(t *testing.T) {
	RunTestWith(t, RunOverwrite, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerMultiKeyUnsharded(t *testing.T) {
	RunTestWith(t, RunMultiKey, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerTtlUnsharded(t *testing.T) {
	RunTestWith(t, RunTtl, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerTtlMultiKeyUnsharded(t *testing.T) {
	RunTestWith(t, RunTtlMultiKey, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerTtlExtension(t *testing.T) {
	// Your server must appropriately update TTLs, even if the value already
	// exists in cache -- in this case we set a TTL of 100ms, then overwrite
	// with a TTL of 100s.
	setup := MakeTestSetup(MakeBasicOneShard())

	err := setup.NodeSet("n1", "abc", "123", 100*time.Millisecond)
	assert.Nil(t, err)

	err = setup.NodeSet("n1", "abc", "123", 100*time.Second)
	assert.Nil(t, err)

	time.Sleep(150 * time.Millisecond)

	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	setup.Shutdown()
}

func TestServerSetEmptyKey(t *testing.T) {
	// Your server must treat a set key with an empty value ""
	// differently from a key which has not been set -- you
	// return "wasFound" if the key was set, but still return
	// the empty string.
	setup := MakeTestSetup(MakeBasicOneShard())
	err := setup.NodeSet("n1", "abc", "", 100*time.Second)
	assert.Nil(t, err)

	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "", val)

	setup.Shutdown()
}

func TestServerConcurrentGetsSingleKeyUnsharded(t *testing.T) {
	RunTestWith(t, RunConcurrentGetsSingleKey, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerConcurrentGetsMultiKeyUnsharded(t *testing.T) {
	RunTestWith(t, RunConcurrentGetsMultiKey, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerConcurrentGetsAndSets(t *testing.T) {
	RunTestWith(t, RunConcurrentGetsAndSets, MakeTestSetup(MakeBasicOneShard()))
}

func TestServerMultiShardSingleNode(t *testing.T) {
	// Runs all the basic tests on a single node setup,
	// but with multiple shards assigned. This shouldn't
	// be functionally much different from a single node
	// with a single shard, but may stress cases if you store
	// data for different shards in separate storage.
	setup := MakeTestSetup(MakeMultiShardSingleNode())
	t.Run("basic", func(t *testing.T) {
		RunBasic(t, setup)
	})
	t.Run("overwrite", func(t *testing.T) {
		RunOverwrite(t, setup)
	})
	t.Run("multikey", func(t *testing.T) {
		RunMultiKey(t, setup)
	})
	t.Run("ttl", func(t *testing.T) {
		RunTtl(t, setup)
	})
	t.Run("ttl-multikey", func(t *testing.T) {
		RunTtlMultiKey(t, setup)
	})
	t.Run("concurrent", func(t *testing.T) {
		RunConcurrentGetsAndSets(t, setup)
	})
	setup.Shutdown()
}

func assertErrorWithCode(t *testing.T, err error, code codes.Code) {
	assert.NotNil(t, err)

	e, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, code, e.Code())
}

func assertShardNotAssigned(t *testing.T, err error) {
	assertErrorWithCode(t, err, codes.NotFound)
}

func TestServerRejectOpsWithoutShardAssignment(t *testing.T) {
	// The first step of dynamic sharding is rejecting requests on the
	// server which are to shards which are not hosted. This prevents
	// a misconfigured or stale client from sending requests to a server
	// and assuming it was handled correctly.
	setup := MakeTestSetup(MakeNoShardAssigned())
	err := setup.NodeSet("n1", "abc", "123", 1*time.Second)
	assertShardNotAssigned(t, err)

	_, _, err = setup.NodeGet("n1", "abc")
	assertShardNotAssigned(t, err)

	err = setup.NodeDelete("n1", "abc")
	assertShardNotAssigned(t, err)
	setup.Shutdown()
}

func TestServerRejectWithoutShardAssignmentMultiShard(t *testing.T) {
	// Similary, your service should function even if only some of
	// the shards are assigned to a server. In this case, half of the
	// shards are assigned and half are entirely unassigned.
	setup := MakeTestSetup(MakeSingleNodeHalfShardsAssigned())

	keys := RandomKeys(100, 5)
	wasAssigned := 0
	for _, k := range keys {
		err := setup.NodeSet("n1", k, "123", 1*time.Second)
		if err != nil {
			assertShardNotAssigned(t, err)
			_, _, err = setup.NodeGet("n1", k)
			assertShardNotAssigned(t, err)

			err = setup.NodeDelete("n1", k)
			assertShardNotAssigned(t, err)
		} else {
			wasAssigned = wasAssigned + 1
		}
	}
	// Exactly one half of the shards are assigned, so we would expect about half
	// of the keys to generate errors, and half to succeed.
	//
	// We allow pretty generous wiggle room here depending on the sharding function,
	// but if outside this range you should try a more evenly distributed sharding function
	assert.Less(t, 20, wasAssigned)
	assert.Greater(t, 80, wasAssigned)

	setup.Shutdown()
}

func TestServerTtlDataActuallyCleanedUp(t *testing.T) {
	// This test ensure that you actually clean up values from internal storage after some time.
	// Note that this doesn't necessarily mean immediately deleting from the map once TTL
	// has expired, but you must do so in a relatively timely fashion (a few seconds).
	//
	// We send 100 keys with values of size 1MB each (total 100MB), make sure they
	// exist, then wait for the TTL and check that the server memory usage declines.
	//
	// We send some dummy requests in case your server only deletes items on-demand
	// when requests come in. Note that we do not send requests to the keys that need
	// to be expired -- you must be able to functionally delete these keys even
	// if there are no requests to those keys directly.
	//
	// Given the large amount of data in this test, we are relatively lenient with times.
	// For the 100MB of data we give a 10s TTL, and it must be mostly cleaned up by 20s.
	//
	// Expect this test to run for 20s or more (but less than a minute).
	setup := MakeTestSetup(MakeMultiShardSingleNode())
	var baselineMemory runtime.MemStats
	runtime.ReadMemStats(&baselineMemory)
	logrus.Debugf("baseline memory usage: %dMB", baselineMemory.Alloc/1024/1024)

	keys := RandomKeys(100, 10)
	vals := RandomKeys(100, 1024*1024)
	logrus.Debugf("created random data")
	for i, k := range keys {
		val := vals[i]
		err := setup.NodeSet("n1", k, val, 10*time.Second)
		assert.Nil(t, err)
	}

	for _, k := range keys {
		_, wasFound, err := setup.NodeGet("n1", k)
		assert.Nil(t, err)
		assert.True(t, wasFound)
	}

	runtime.GC()
	var startingMemory runtime.MemStats
	runtime.ReadMemStats(&startingMemory)
	assert.Greater(t, startingMemory.Alloc, uint64(100*1024*1024))
	logrus.Debugf(
		"memory usage after setting 100MB worth of data: %dMB",
		startingMemory.Alloc/1024/1024,
	)

	for i := 0; i < 10; i++ {
		for _, k := range keys {
			_, _, err := setup.NodeGet("n1", k+"-new")
			assert.Nil(t, err)

			err = setup.NodeSet("n1", k+"-new", "1", 1*time.Second)
			assert.Nil(t, err)
		}
		time.Sleep(2 * time.Second)
		runtime.GC()
	}

	var endingMemory runtime.MemStats
	runtime.ReadMemStats(&endingMemory)
	memoryDiff := startingMemory.Alloc - endingMemory.Alloc
	logrus.Debugf("memory after waiting out TTL: %dMB", endingMemory.Alloc/1024/1024)
	assert.Greater(t, startingMemory.Alloc, endingMemory.Alloc)
	// Expect a difference of at least 80MB -- cleaning up at least 80% of the 100MB set
	assert.Less(t, uint64(80*1024*1024), memoryDiff)

	setup.Shutdown()
}

func TestServerSingleShardDrop(t *testing.T) {
	// First test of dynamic shard assignment. We start with the same setup
	// as the basic tests -- single node "n1" with single shard 1 assigned,
	// then we update the shard map so that "n1" has no shards assigned,
	// so requests should now fail.
	setup := MakeTestSetup(MakeBasicOneShard())
	err := setup.NodeSet("n1", "abc", "123", 10*time.Second)
	assert.Nil(t, err)
	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	// Remove shard 1 from n1
	setup.UpdateShardMapping(map[int][]string{
		1: {},
	})

	_, _, err = setup.NodeGet("n1", "abc")
	assertShardNotAssigned(t, err)

	err = setup.NodeSet("n1", "abc", "123", 10*time.Second)
	assertShardNotAssigned(t, err)

	setup.Shutdown()
}

func TestServerSingleShardDropDataDeleted(t *testing.T) {
	// Similar to TestSingleShardDrop, but we remove the shard
	// and then add it back and ensure that the data was actually
	// deleted in between. If a shard is dropped the server must
	// clean up all data, or at least not return it from future
	// Get() calls.
	setup := MakeTestSetup(MakeBasicOneShard())
	err := setup.NodeSet("n1", "abc", "123", 10*time.Second)
	assert.Nil(t, err)

	// Remove shard 1 from n1
	setup.UpdateShardMapping(map[int][]string{
		1: {},
	})

	_, _, err = setup.NodeGet("n1", "abc")
	assertShardNotAssigned(t, err)

	// Add shard 1 back to n1
	setup.UpdateShardMapping(map[int][]string{
		1: {"n1"},
	})

	_, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	setup.Shutdown()
}

func TestServerSingleShardMoveNoCopy(t *testing.T) {
	// This test sets up a two node cluster with a single shard
	// Shard 1 stards on node n1, has some data written, then shard 1
	// is dropped entirely.
	//
	// Shard 1 is then assigned to n2 (after the drop),
	// so n2 should not have any data to copy and should start cold
	setup := MakeTestSetup(
		kv.ShardMapState{
			NumShards: 1,
			Nodes:     makeNodeInfos(2),
			ShardsToNodes: map[int][]string{
				1: {"n1"},
			},
		},
	)

	// n1 hosts the shard, so we should be able to set data
	err := setup.NodeSet("n1", "abc", "123", 10*time.Second)
	assert.Nil(t, err)
	// n2 doesnt, so all requests should fail
	err = setup.NodeSet("n2", "abc", "123", 10*time.Second)
	assertShardNotAssigned(t, err)

	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	// Remove shard 1 from n1
	setup.UpdateShardMapping(map[int][]string{
		1: {},
	})
	// Add shard 1 to n2
	setup.UpdateShardMapping(map[int][]string{
		1: {"n2"},
	})

	_, _, err = setup.NodeGet("n1", "abc")
	// shard no longer mapped, so should error
	assertShardNotAssigned(t, err)

	// should be assigned to n2 now, but no data exists
	_, wasFound, err = setup.NodeGet("n2", "abc")
	assert.Nil(t, err)
	assert.False(t, wasFound)

	setup.Shutdown()
}

func TestServerSingleShardMoveWithCopy(t *testing.T) {
	// Similar to above, but actually tests shard copying from n1 to n2.
	// A single shard is assigned to n1, one key written, then we add the
	// shard to n2. At this point n2 should copy data from its peer n1.
	//
	// We then delete the shard from n1, so n2 is the sole owner, and ensure
	// that n2 has the data originally written to n1.
	setup := MakeTestSetup(
		kv.ShardMapState{
			NumShards: 1,
			Nodes:     makeNodeInfos(2),
			ShardsToNodes: map[int][]string{
				1: {"n1"},
			},
		},
	)

	// n1 hosts the shard, so we should be able to set data
	err := setup.NodeSet("n1", "abc", "123", 10*time.Second)
	assert.Nil(t, err)

	// Add shard 1 to n2
	setup.UpdateShardMapping(map[int][]string{
		1: {"n1", "n2"},
	})
	// Remove shard 1 from n1
	setup.UpdateShardMapping(map[int][]string{
		1: {"n2"},
	})

	_, _, err = setup.NodeGet("n1", "abc")
	// shard no longer mapped, so should error
	assertShardNotAssigned(t, err)

	// should be assigned to n2 now and data should've
	// been copied over from n1
	val, wasFound, err := setup.NodeGet("n2", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	setup.Shutdown()
}

func TestServerMultiShardMoveWithCopy(t *testing.T) {
	// Expands on the fundamentals of SingleShardMoveWithCopy, but
	// does a shard swap between two nodes. Here we generate 100 random
	// keys, and attempt to assign them all to both nodes. Half of the
	// requests should fail (given the shard is not mapped to that node),
	// but we record where the key was mapped originally. After the shard
	// swap, the keys should flip nodes.
	//
	// Starts with n1: {1,2,3,4,5} and n2: {6,7,8,9,10}
	setup := MakeTestSetup(MakeTwoNodeMultiShard())

	keys := RandomKeys(100, 10)
	keysToOriginalNode := make(map[string]string)
	for _, key := range keys {
		// Try setting key on `n1` -- it should either succeed
		// if they key maps to shards 1-5 (we don't know in this case
		// because we don't control your sharding function), or it
		// should fail if it maps to shards 6-10.
		err := setup.NodeSet("n1", key, "123", 10*time.Second)
		if err == nil {
			keysToOriginalNode[key] = "n1"
		} else {
			assertShardNotAssigned(t, err)

			// If the request failed, it should succeed on n2 which
			// hosts the remaining shards
			err = setup.NodeSet("n2", key, "123", 10*time.Second)
			assert.Nil(t, err)
			keysToOriginalNode[key] = "n2"
		}
	}

	// Start the swap by adding all shards to both nodes,
	// allowing them to copy from each other
	setup.UpdateShardMapping(map[int][]string{
		1:  {"n1", "n2"},
		2:  {"n1", "n2"},
		3:  {"n1", "n2"},
		4:  {"n1", "n2"},
		5:  {"n1", "n2"},
		6:  {"n1", "n2"},
		7:  {"n1", "n2"},
		8:  {"n1", "n2"},
		9:  {"n1", "n2"},
		10: {"n1", "n2"},
	})
	// Now remove the duplication, doing the full swap
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

	for _, key := range keys {
		var newNode string
		if keysToOriginalNode[key] == "n1" {
			newNode = "n2"
		} else {
			newNode = "n1"
		}

		// Key should no longer exist on the original node after the swap
		_, _, err := setup.NodeGet(keysToOriginalNode[key], key)
		assertShardNotAssigned(t, err)

		// But it should exist on the new node
		val, wasFound, err := setup.NodeGet(newNode, key)
		assert.Nil(t, err)
		assert.True(t, wasFound)
		assert.Equal(t, "123", val)
	}

	setup.Shutdown()
}

func TestServerRestartShardCopy(t *testing.T) {
	// Tests that your server copies data at startup as well, not just
	// shard movements after it is running.
	//
	// We have two nodes with a single shard, and we shutdown and restart n2
	// and it should copy data from n1.
	setup := MakeTestSetup(MakeTwoNodeBothAssignedSingleShard())

	err := setup.NodeSet("n1", "abc", "123", 100*time.Second)
	assert.Nil(t, err)
	err = setup.NodeSet("n2", "abc", "123", 100*time.Second)
	assert.Nil(t, err)

	// Value should exist on n1 and n2
	val, wasFound, err := setup.NodeGet("n1", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	val, wasFound, err = setup.NodeGet("n2", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	setup.nodes["n2"].Shutdown()
	setup.nodes["n2"] = kv.MakeKvServer("n2", setup.shardMap, &setup.clientPool)

	// n2 should copy the data from n1 on restart
	val, wasFound, err = setup.NodeGet("n2", "abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	setup.Shutdown()
}
