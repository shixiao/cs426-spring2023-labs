package kvtest

import (
	"testing"

	"cs426.yale.edu/lab4/kv"
	"github.com/stretchr/testify/assert"
)

// Internal tests for our testing library (test_setup.go)

// modifies smState in place
func checkAddReplicas(
	t *testing.T,
	smState *kv.ShardMapState,
	shard int,
	numToAdd int,
	dstNodes []string,
	shouldError bool,
) {
	ogNodes := smState.ShardsToNodes[shard]
	err := addReplicasForShard(smState, shard, numToAdd, dstNodes)
	assert.Equal(t, shouldError, err != nil)
	if shouldError {
		return
	}

	result := smState.ShardsToNodes[shard]
	assert.Equal(t, len(ogNodes)+numToAdd, len(result))

	resultSet := sliceToMap(result)
	// no duplicates
	assert.Equal(t, len(ogNodes)+numToAdd, len(resultSet))
	// all valid
	for node := range resultSet {
		_, present := smState.Nodes[node]
		assert.True(t, present)
	}
	// dstNodes are added
	for _, node := range dstNodes {
		_, present := resultSet[node]
		assert.True(t, present)
	}
}

func checkDropReplicas(
	t *testing.T,
	smState *kv.ShardMapState,
	shard int,
	numToDrop int,
	srcNodes []string,
	shouldError bool,
) {
	ogNodes := smState.ShardsToNodes[shard]
	err := dropReplicasForShard(smState, shard, numToDrop, srcNodes)
	assert.Equal(t, shouldError, err != nil)
	if shouldError {
		return
	}

	result := smState.ShardsToNodes[shard]
	assert.Equal(t, len(ogNodes)-numToDrop, len(result))

	resultSet := sliceToMap(result)
	// no duplicates
	assert.Equal(t, len(ogNodes)-numToDrop, len(resultSet))

	// all results were in ogNodes
	ogNodesSet := sliceToMap(ogNodes)
	for node := range resultSet {
		_, present := ogNodesSet[node]
		assert.True(t, present)
	}
	// all srcNodes were removed
	for _, node := range srcNodes {
		_, present := resultSet[node]
		assert.False(t, present)
	}
}

func TestSetupShardMapComputation(t *testing.T) {
	smState := MakeFourNodesWithFiveShards()
	checkAddReplicas(t, &smState, 2, 1, nil, false)
	// should error as shard 2 is already replicated on all four nodes
	checkAddReplicas(t, &smState, 2, 1, nil, true)
	// n4 already hosts shard 4
	checkAddReplicas(t, &smState, 4, 3, []string{"n4"}, true)
	checkAddReplicas(t, &smState, 5, 2, []string{"n4"}, false)

	checkDropReplicas(t, &smState, 1, 3, nil, false)
	// should error since shard 1 no longer has >= 3 replicas to drop
	checkDropReplicas(t, &smState, 1, 3, nil, true)
	checkDropReplicas(t, &smState, 2, 4, []string{"n1"}, false)
	checkDropReplicas(t, &smState, 5, 2, []string{"n2"}, false)
	// should error since shard 5 no longer has >= 3 replicas to drop
	checkDropReplicas(t, &smState, 5, 3, nil, true)
	// n4 does not host shard 3, hence should error
	checkDropReplicas(t, &smState, 3, 1, []string{"n4"}, true)
}

func TestShardMapStateGeneration(t *testing.T) {
	assert.True(t, MakeBasicOneShard().IsValid())
	assert.True(t, MakeMultiShardSingleNode().IsValid())
	assert.True(t, MakeNoShardAssigned().IsValid())
	assert.True(t, MakeSingleNodeHalfShardsAssigned().IsValid())
	assert.True(t, MakeTwoNodeBothAssignedSingleShard().IsValid())
	assert.True(t, MakeTwoNodeMultiShard().IsValid())
	assert.True(t, MakeFourNodesWithFiveShards().IsValid())

	assert.True(t, MakeManyNodesWithManyShards(0, 0).IsValid())
	assert.True(t, MakeManyNodesWithManyShards(0, 4).IsValid())
	assert.True(t, MakeManyNodesWithManyShards(3, 0).IsValid())
	assert.True(t, MakeManyNodesWithManyShards(4, 5).IsValid())
	assert.True(t, MakeManyNodesWithManyShards(4, 1).IsValid())
	assert.True(t, MakeManyNodesWithManyShards(20, 7).IsValid())
	assert.True(t, MakeManyNodesWithManyShards(20, 20).IsValid())
	for i := 0; i < 100; i++ {
		assert.True(t, MakeManyNodesWithManyShards(100, 4).IsValid())
		assert.True(t, MakeManyNodesWithManyShards(1000, 700).IsValid())
	}
}
