package kvtest

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cs426.yale.edu/lab4/logging"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

// These tests will cover the full project -- both client and
// server implementation working together. You must have completed
// at least primitive functionality (gets, sets, etc) on both
// for these tests to pass.

func TestIntegrationBasic(t *testing.T) {
	setup := MakeTestSetup(MakeBasicOneShard())

	err := setup.Set("abc", "123", 10*time.Second)
	assert.Nil(t, err)

	val, wasFound, err := setup.Get("abc")
	assert.Nil(t, err)
	assert.True(t, wasFound)
	assert.Equal(t, "123", val)

	setup.Shutdown()
}

func TestIntegrationConcurrentGetsAndSets(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())

	const numGoros = 30
	const numIters = 500 // should be >= 200
	keys := RandomKeys(200, 20)
	vals := RandomKeys(200, 40)

	found := make([]int32, 200)
	var wg sync.WaitGroup
	wg.Add(numGoros)
	for i := 0; i < numGoros; i++ {
		go func(i int) {
			for j := 0; j < numIters; j++ {
				err := setup.Set(keys[(i*100+j)%179], vals[(j*100+i)%179], 100*time.Second)
				assert.Nil(t, err)
				for k := 0; k < 200; k++ {
					_, wasFound, err := setup.Get(keys[k])
					assert.Nil(t, err)
					if wasFound {
						atomic.StoreInt32(&found[k], 1)
					}
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	// eventually the Gets will have observed the Sets
	for i := 0; i < 179; i++ {
		assert.True(t, found[i] == 1)
	}
	for i := 179; i < 200; i++ {
		assert.False(t, found[i] == 1)
	}

	setup.Shutdown()
}

func initSetup(t *testing.T, setup *TestSetup) (int, []string, []string) {
	numKeys := 500
	keys := RandomKeys(numKeys, 20)
	vals := RandomKeys(numKeys, 40)
	for i := 0; i < numKeys; i++ {
		err := setup.Set(keys[i], vals[i], 100*time.Second)
		assert.Nil(t, err)
	}
	return numKeys, keys, vals
}

type CountResult struct {
	numKeysFound int
	// request was successful but the key was not found
	numKeysMissing int
	// map from error -> numErrors
	errorMap map[error]int
}

// Check the status of each key in keys, record results
//
// If nodeToQuery is not empty, query the specific node; otherwise query the
// cluster and let kv find the proper node to connect to.
func countKeysFoundHelper(
	t *testing.T,
	setup *TestSetup,
	keys []string,
	vals []string,
	nodeToQuery string,
) CountResult {
	cntFound := 0
	cntMissing := 0
	errorMap := make(map[error]int, 0)
	for i, k := range keys {
		var val string
		var wasFound bool
		var err error
		if len(nodeToQuery) == 0 {
			val, wasFound, err = setup.Get(k)
		} else {
			val, wasFound, err = setup.NodeGet(nodeToQuery, k)
		}
		if err == nil {
			if wasFound {
				assert.Equal(
					t,
					vals[i],
					val,
					fmt.Sprintf("expected value for key %s is %s, found %s", k, vals[i], val),
				)
				cntFound++
			} else {
				cntMissing++
			}
		} else {
			errorMap[err]++
		}
	}
	return CountResult{numKeysFound: cntFound, numKeysMissing: cntMissing, errorMap: errorMap}
}

func countKeysFound(
	t *testing.T,
	setup *TestSetup,
	keys []string,
	vals []string,
) CountResult {
	return countKeysFoundHelper(t, setup, keys, vals, "")
}

func countKeysFoundOnNode(
	t *testing.T,
	setup *TestSetup,
	keys []string,
	vals []string,
	nodeToQuery string,
) CountResult {
	return countKeysFoundHelper(t, setup, keys, vals, nodeToQuery)
}

func TestIntegrationDropShard(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())
	numKeys, keys, vals := initSetup(t, setup)

	err := setup.DropRandomShard()
	assert.Nil(t, err)

	// after dropping one of five shards, expect at least half the keys remain
	cntRes := countKeysFound(t, setup, keys, vals)
	assert.Greater(t, cntRes.numKeysFound, int(numKeys/2))
	assert.Less(t, cntRes.numKeysMissing, numKeys)

	setup.Shutdown()
}

/*
 * This tests directly removes nodes from the cluster and the shardmap, without
 * copying the data on that node to other nodes. This ungraceful node draining
 * is potentially useful as emergency tooling but may result in data loss.
 */

func TestIntegrationRemoveNode(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())
	numKeys, keys, vals := initSetup(t, setup)

	err := setup.RemoveNode("n4")
	assert.Nil(t, err)
	// since there isn't a shard that is only on n4, we expect all the keys are still present.
	cntRes := countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)
	assert.Equal(t, 0, len(cntRes.errorMap))

	err = setup.RemoveNode("n1")
	assert.Nil(t, err)
	// n1 and n4 is not a copy set, so we don't expect data loss
	cntRes = countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)
	assert.Equal(t, 0, len(cntRes.errorMap))

	// if we drain too many nodes, we lose some data
	err = setup.RemoveNode("n3")
	assert.Nil(t, err)
	cntRes = countKeysFound(t, setup, keys, vals)
	assert.Greater(t, cntRes.numKeysFound, 0)
	assert.Less(t, cntRes.numKeysFound, numKeys)

	setup.Shutdown()
}

/*
 * Gracefully drain nodes from the cluster. This workflow copies shards hosted
 * on nodeToDrain elsewhere first, before dropping the node.
 */
func TestIntegrationDrainNode(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())
	numKeys, keys, vals := initSetup(t, setup)

	err := setup.DrainNode("n4")
	assert.Nil(t, err)
	cntRes := countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)
	assert.Equal(t, 0, len(cntRes.errorMap))

	err = setup.DrainNode("n1")
	assert.Nil(t, err)
	// n1 and n4 is not a copy set, so we don't expect data loss
	cntRes = countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)
	assert.Equal(t, 0, len(cntRes.errorMap))

	// n2 is still up and should now host all the shards, so we still don't
	// expect to lose data. (In contrast to RemoveNode in the test case above)
	err = setup.DrainNode("n3")
	assert.Nil(t, err)
	cntRes = countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)
	assert.Equal(t, 0, len(cntRes.errorMap))

	setup.Shutdown()
}

/*
 * This test performs graceful shard movements (see MoveShard in test_setup.go)
 * by performing each movement in two steps--first copy data over to the
 * destination node, then drop the shard on the source node. Through this
 * process, no keys should be lost.
 */
func TestIntegrationGracefulShardMovementBasic(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())
	numKeys, keys, vals := initSetup(t, setup)

	// initially the keys should be close to evenly distributed among the shards;
	// given the setup, n3 should have shards 1, 3, and 4.
	cntRes := countKeysFoundOnNode(t, setup, keys, vals, "n3")
	assert.Greater(t, cntRes.numKeysFound, int(numKeys/2))
	assert.Less(t, cntRes.numKeysFound, numKeys)

	err := setup.MoveShard(2, "n1", "n3")
	assert.Nil(t, err)
	err = setup.MoveShard(5, "n2", "n3")
	assert.Nil(t, err)

	// now n3 should have all 5 shards
	cntRes = countKeysFoundOnNode(t, setup, keys, vals, "n3")
	assert.Equal(t, numKeys, cntRes.numKeysFound)

	// accessing all keys through the cluster should also work
	cntRes = countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)

	setup.Shutdown()
}

/*
 * This test performs random single shard moves in a sequential manner.
 * MoveRandomShard picks a random shard to move from a random node that hosts
 * that shard to a host that does not. The invariant is there should not be any data lost.
 */
func TestIntegrationGracefulShardMovementConsecutiveSingleShardMoves(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())
	numKeys, keys, vals := initSetup(t, setup)

	numIters := 200
	for i := 0; i < numIters; i++ {
		err := setup.MoveRandomShard()
		assert.Nil(t, err)
	}

	// After many single shard movements, all keys should still be present
	cntRes := countKeysFound(t, setup, keys, vals)
	assert.Equal(t, numKeys, cntRes.numKeysFound)
}

// Moving multiple shards / moving multiple replicas for a shard at once
func TestIntegrationGracefulMultiShardMovement(t *testing.T) {
	setup := MakeTestSetup(MakeFourNodesWithFiveShards())
	numKeys, keys, vals := initSetup(t, setup)

	// initially the keys should be close to evenly distributed among the shards;
	// given the setup, n3 should have shards 1, 3, and 4; n4 should have shards 2 and 4
	cntRes := countKeysFoundOnNode(t, setup, keys, vals, "n3")
	assert.Greater(t, cntRes.numKeysFound, int(numKeys/2))
	assert.Less(t, cntRes.numKeysFound, numKeys)
	cntRes = countKeysFoundOnNode(t, setup, keys, vals, "n4")
	assert.Greater(t, cntRes.numKeysFound, 0)

	// in a batch (but still two steps), perform the following actions:
	// * shard 2: n2->n3, drop n4 (reduce replication factor by 1)
	// * shard 4: drop n4, but increase replication factor by 1
	// * shard 5: n2->n3
	setup.UpdateShardMapping(map[int][]string{
		1: {"n1", "n2", "n3"},
		2: {"n1", "n2", "n3", "n4"},
		3: {"n2", "n3"},
		4: {"n3", "n4", "n1", "n2"},
		5: {"n2", "n3"},
	})
	setup.UpdateShardMapping(map[int][]string{
		1: {"n1", "n2", "n3"},
		2: {"n1", "n3"},
		3: {"n2", "n3"},
		4: {"n1", "n2", "n3"},
		5: {"n3"},
	})

	// now n3 should have all 5 shards
	cntRes = countKeysFoundOnNode(t, setup, keys, vals, "n3")
	assert.Equal(t, numKeys, cntRes.numKeysFound)
	// n4 should have zero data
	cntRes = countKeysFoundOnNode(t, setup, keys, vals, "n4")
	assert.Equal(t, 0, cntRes.numKeysFound)

	setup.Shutdown()
}

/*
 * This test performs shard movements in one goro while other client goros
 * continues to read and write to the cluster.
 */
func TestIntegrationFull(t *testing.T) {
	logrus.Debugf("starting integration setup")
	setup := MakeTestSetup(MakeManyNodesWithManyShards(1000, 700))

	const numKeys = 1000
	keys := RandomKeys(numKeys, 20)
	vals := RandomKeys(numKeys, 40)
	for i := 0; i < numKeys; i++ {
		setup.Set(keys[i], vals[i], 100*time.Second)
	}

	logrus.Debugf("finished integration setup")

	const numClientGoros = 10
	endTime := time.Now().Add(20 * time.Second)
	var wg sync.WaitGroup
	wg.Add(numClientGoros)
	clientsDone := make(chan struct{})
	// shard movement goro
	go func() {
		for {
			select {
			case <-clientsDone:
				return
			default:
				logrus.Trace("moving random shard")
				err := setup.MoveRandomShard()
				assert.Nil(t, err)
				time.Sleep(30 * time.Millisecond)
			}
		}
	}()
	for i := 0; i < numClientGoros; i++ {
		go func(i int) {
			// loop till endTime or at least iterate once through
			once := false
			for time.Now().Before(endTime) || !once {
				for j := 0; j < numKeys*2; j++ {
					_ = setup.Set(keys[(i*100+j)%179], vals[(i*100+j)%179], 100*time.Second)
					for k := 0; k < 10; k++ {
						_, _, _ = setup.Get(keys[(i*100+j)%numKeys])
					}
					_ = setup.Delete(keys[(i*100+j)%179])
				}
				once = true
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	clientsDone <- struct{}{}

	// eventually the Gets will have observed the Sets and Deletes
	cntRes := countKeysFound(t, setup, keys, vals)
	assert.GreaterOrEqual(t, cntRes.numKeysFound, numKeys-179)
	assert.Less(t, cntRes.numKeysFound, numKeys)

	assert.Equal(t, 0, len(cntRes.errorMap))
	setup.Shutdown()
}

func TestMain(m *testing.M) {
	flag.Parse()
	logging.InitLogging()
	code := m.Run()
	os.Exit(code)
}
