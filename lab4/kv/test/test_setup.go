package kvtest

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cs426.yale.edu/lab4/kv"
	"cs426.yale.edu/lab4/kv/proto"
	"github.com/sirupsen/logrus"
)

type TestSetup struct {
	shardMap   *kv.ShardMap
	nodes      map[string]*kv.KvServerImpl
	clientPool TestClientPool
	kv         *kv.Kv
	ctx        context.Context
}

func MakeTestSetup(shardMap kv.ShardMapState) *TestSetup {
	setup := TestSetup{
		shardMap: &kv.ShardMap{},
		ctx:      context.Background(),
		nodes:    make(map[string]*kv.KvServerImpl),
	}
	setup.shardMap.Update(&shardMap)
	for name := range setup.shardMap.Nodes() {
		setup.nodes[name] = kv.MakeKvServer(
			name,
			setup.shardMap,
			&setup.clientPool,
		)
	}
	setup.clientPool.Setup(setup.nodes)
	setup.kv = kv.MakeKv(setup.shardMap, &setup.clientPool)

	logrus.WithFields(
		logrus.Fields{"nodes": len(shardMap.Nodes), "shards": len(shardMap.ShardsToNodes)},
	).Debug("created test setup with servers")
	return &setup
}
func MakeTestSetupWithoutServers(shardMap kv.ShardMapState) *TestSetup {
	// Remove nodes so we never have a chance of sending data
	// to the KvServerImpl attached as a safety measure for client_test.go
	//
	// Server tests happen in server_test.go, so we can test
	// client implementations separately.
	//
	// Combined tests happen in integration_tests.go (server and client together),
	// along with stress tests.
	setup := TestSetup{
		shardMap: &kv.ShardMap{},
		ctx:      context.Background(),
		nodes:    make(map[string]*kv.KvServerImpl),
	}
	setup.shardMap.Update(&shardMap)
	for name := range setup.shardMap.Nodes() {
		setup.nodes[name] = nil
	}
	setup.clientPool.Setup(setup.nodes)
	setup.kv = kv.MakeKv(setup.shardMap, &setup.clientPool)
	logrus.WithFields(
		logrus.Fields{"nodes": len(shardMap.Nodes), "shards": len(shardMap.ShardsToNodes)},
	).Debug("created test setup without servers")
	return &setup
}

func (ts *TestSetup) NodeGet(nodeName string, key string) (string, bool, error) {
	response, err := ts.nodes[nodeName].Get(context.Background(), &proto.GetRequest{Key: key})
	if err != nil {
		return "", false, err
	}
	return response.Value, response.WasFound, nil
}
func (ts *TestSetup) NodeSet(nodeName string, key string, value string, ttl time.Duration) error {
	_, err := ts.nodes[nodeName].Set(
		context.Background(),
		&proto.SetRequest{Key: key, Value: value, TtlMs: ttl.Milliseconds()},
	)
	return err
}
func (ts *TestSetup) NodeDelete(nodeName string, key string) error {
	_, err := ts.nodes[nodeName].Delete(context.Background(), &proto.DeleteRequest{Key: key})
	return err
}

func (ts *TestSetup) Get(key string) (string, bool, error) {
	return ts.kv.Get(ts.ctx, key)
}
func (ts *TestSetup) Set(key string, value string, ttl time.Duration) error {
	return ts.kv.Set(ts.ctx, key, value, ttl)
}
func (ts *TestSetup) Delete(key string) error {
	return ts.kv.Delete(ts.ctx, key)
}

func (ts *TestSetup) UpdateShardMapping(shardsToNodes map[int][]string) {
	state := kv.ShardMapState{
		Nodes:         ts.shardMap.Nodes(),
		NumShards:     ts.shardMap.NumShards(),
		ShardsToNodes: shardsToNodes,
	}
	ts.shardMap.Update(&state)

	// We update again, which effectively waits for the first update to have been processed,
	// because we wait for updates to be processed stricly in order.
	//
	// TODO: decide if we want this or a different sync API
	ts.shardMap.Update(&state)
}

func (ts *TestSetup) NumShards() int {
	return ts.shardMap.NumShards()
}

func nodesExcept(nodes []string, nodeToRemove string) []string {
	newNodes := make([]string, 0)
	for _, node := range nodes {
		if node != nodeToRemove {
			newNodes = append(newNodes, node)
		}
	}
	return newNodes
}

/*
 * ======== REPLICA ADJUSTMENT AND SHARD MOVEMENT WORKFLOWS ========
 * These functions facilitate different types of shard and replica adjustment
 * workflows a production system might require.
 *
 * Assumes one workflow at a time; workflows functions are synchronous.
 * TODO: add a mutex/semaphore to ensure serial execution
 */
func sliceToMap(nodes []string) map[string]struct{} {
	nodeSet := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		nodeSet[node] = struct{}{}
	}
	return nodeSet
}

/*
 * Helper function to deep copy ShardMapState:
 *
 * The *ShardMapState we obtain from the ShardMap should be treated as
 * read-only. We perform a deep copy so that the workflows can do in-place
 * edits.
 *
 * This also implies that after EACH updateShardMapping call (which updates the
 * ShardsToNodes map in the ShardMap and propagates to its listeners), a new
 * copy must be created before further edits.
 */
func copyShardMapState(src *kv.ShardMapState) kv.ShardMapState {
	copiedNodes := make(map[string]kv.NodeInfo)
	for nodeName, node := range src.Nodes {
		copiedNodes[nodeName] = node
	}
	copiedShardsToNodes := make(map[int][]string)
	for shard, nodes := range src.ShardsToNodes {
		copiedNodeNames := make([]string, len(nodes))
		copy(copiedNodeNames, nodes)
		copiedShardsToNodes[shard] = copiedNodeNames
	}
	return kv.ShardMapState{
		NumShards:     src.NumShards,
		Nodes:         copiedNodes,
		ShardsToNodes: copiedShardsToNodes,
	}
}
func (ts *TestSetup) getShardMapStateCopy() kv.ShardMapState {
	return copyShardMapState(ts.shardMap.GetState())
}

/*
 * Add numReplicas nodes to host a particular shard. If dstNodes are specified,
 * replicate to those nodes; pick randomly otherwise.
 *
 * 0 <= len(dstNodes) <= numReplicasToAdd
 * dstNodes must be valid nodes in smState; dstNodes should NOT contain
 * duplicates or nodes that already host the shard; dstNodes may be nil.
 *
 * This function modifies smState in-place and returns any error encountered
 */
func addReplicasForShard(
	smState *kv.ShardMapState,
	shard int,
	numReplicasToAdd int,
	dstNodes []string,
) error {
	nodes := make([]string, len(smState.ShardsToNodes[shard]))
	copy(nodes, smState.ShardsToNodes[shard])
	allNodesSet := smState.Nodes
	if len(nodes)+numReplicasToAdd > len(allNodesSet) {
		return fmt.Errorf(
			"not enough nodes available to satisfy replication requirement for shard %d",
			shard,
		)
	}
	nodesSet := sliceToMap(nodes)
	// add dstNodes first
	for _, node := range dstNodes {
		_, present := nodesSet[node]
		if present {
			return fmt.Errorf("dstNode %v already added for shard %d", node, shard)
		}
		nodesSet[node] = struct{}{}
		nodes = append(nodes, node)
	}
	dstOptions := make([]string, 0, len(allNodesSet)-len(nodes))
	for node := range allNodesSet {
		_, present := nodesSet[node]
		if !present {
			dstOptions = append(dstOptions, node)
		}
	}
	perm := rand.Perm(len(dstOptions))
	for i := 0; i < numReplicasToAdd-len(dstNodes); i++ {
		nodes = append(nodes, dstOptions[perm[i]])
	}

	smState.ShardsToNodes[shard] = nodes
	return nil
}

/*
 * Remove numReplicas nodes from a particular shard. If srcNodes are specified,
 * drop those nodes; pick randomly otherwise.
 *
 * 0 <= len(srcNodes) <= numReplicasToDrop
 * srcNodes must be valid, i.e., a subset of smState.NodesForShard(shard).
 *
 * This function modifies smState in-place and returns any error encountered
 */
func dropReplicasForShard(
	smState *kv.ShardMapState,
	shard int,
	numReplicasToDrop int,
	srcNodes []string,
) error {
	nodes := smState.ShardsToNodes[shard]
	numCurrentNodes := len(nodes)
	if numCurrentNodes < numReplicasToDrop {
		return fmt.Errorf(
			"attempting to drop %d replicas when there are only %d nodes for shard %d",
			numReplicasToDrop,
			numCurrentNodes,
			shard,
		)
	}
	if len(srcNodes) > numReplicasToDrop {
		return fmt.Errorf(
			"provided too many (%d) srcNodes (> numReplicasToDrop %d) to drop for shard %d",
			len(srcNodes),
			numReplicasToDrop,
			shard,
		)
	}

	// drop srcNodes first
	srcNodesSet := sliceToMap(srcNodes)
	candidates := make([]string, 0)
	for i := 0; i < numCurrentNodes; i++ {
		_, present := srcNodesSet[nodes[i]]
		if !present {
			candidates = append(candidates, nodes[i])
		} else {
			delete(srcNodesSet, nodes[i])
		}
	}
	if len(srcNodesSet) > 0 {
		return fmt.Errorf("srcNodes %v not in the node set for shard %d", srcNodesSet, shard)
	}
	// drop random nodes until we reach the desired number of replicas
	perm := rand.Perm(numCurrentNodes - len(srcNodes))
	nodes = make([]string, 0)
	for i := 0; i < numCurrentNodes-numReplicasToDrop; i++ {
		nodes = append(nodes, candidates[perm[i]])
	}

	smState.ShardsToNodes[shard] = nodes
	return nil
}

/*
 * Take out a node from the cluster directly, ungracefully, potentially results in dataloss
 */
func (ts *TestSetup) RemoveNode(nodeToRemove string) error {
	smState := ts.getShardMapStateCopy()
	shards := ts.shardMap.ShardsForNode(nodeToRemove)
	for _, shard := range shards {
		err := dropReplicasForShard(&smState, shard, 1, []string{nodeToRemove})
		if err != nil {
			return err
		}
	}
	ts.UpdateShardMapping(smState.ShardsToNodes)
	return nil
}

/*
 * Gracefully move all shards hosted on a node away (e.g., so that the node can
 * be taken away for maintenance.)
 */
func (ts *TestSetup) DrainNode(nodeToDrain string) error {
	smState := ts.getShardMapStateCopy()
	numNodes := len(ts.shardMap.Nodes())
	shards := ts.shardMap.ShardsForNode(nodeToDrain)
	for _, shard := range shards {
		nodes := smState.ShardsToNodes[shard]
		if len(nodes) < numNodes {
			err := addReplicasForShard(&smState, shard, 1, nil)
			if err != nil {
				return err
			}
		}
	}
	ts.UpdateShardMapping(smState.ShardsToNodes)

	smState = ts.getShardMapStateCopy()
	for _, shard := range shards {
		err := dropReplicasForShard(&smState, shard, 1, []string{nodeToDrain})
		if err != nil {
			return err
		}
	}
	ts.UpdateShardMapping(smState.ShardsToNodes)
	return nil
}

func (ts *TestSetup) MoveShard(shardToMove int, srcNode string, dstNode string) error {
	if srcNode == dstNode {
		// no-op
		return nil
	}
	smState := ts.getShardMapStateCopy()
	// copy shard to dstNode
	err := addReplicasForShard(&smState, shardToMove, 1, []string{dstNode})
	if err != nil {
		return err
	}
	ts.UpdateShardMapping(smState.ShardsToNodes)

	smState = ts.getShardMapStateCopy()
	// remove from srcNode
	err = dropReplicasForShard(&smState, shardToMove, 1, []string{srcNode})
	if err != nil {
		return err
	}
	ts.UpdateShardMapping(smState.ShardsToNodes)
	return nil
}

// pick a random shard to move from one of its hosting nodes to a different node
func (ts *TestSetup) MoveRandomShard() error {
	shardToMove := 0
	var nodes []string
	// find a shard which has some nodes hosting but not replicated on all nodes
	// (so that we can move it from one to another)
	for {
		shardToMove = rand.Int()%ts.NumShards() + 1
		nodes = ts.shardMap.NodesForShard(shardToMove)
		if len(nodes) > 0 && len(nodes) < len(ts.shardMap.Nodes()) {
			break
		}
	}
	nodesMap := sliceToMap(nodes)
	srcNode := nodes[rand.Int()%len(nodes)]
	dstCandidate := make([]string, 0)
	for node := range ts.shardMap.Nodes() {
		_, present := nodesMap[node]
		if !present {
			dstCandidate = append(dstCandidate, node)
		}
	}
	dstNode := dstCandidate[rand.Int()%len(dstCandidate)]
	return ts.MoveShard(shardToMove, srcNode, dstNode)
}

/*
 * Drop a shard from every node that hosts it, deliberately drop / lose data.
 */
func (ts *TestSetup) DropShard(shardToDrop int) error {
	smState := ts.getShardMapStateCopy()
	err := dropReplicasForShard(
		&smState,
		shardToDrop,
		len(smState.ShardsToNodes[shardToDrop]),
		[]string{},
	)
	if err != nil {
		return err
	}
	ts.UpdateShardMapping(smState.ShardsToNodes)
	return nil
}

func (ts *TestSetup) DropRandomShard() error {
	shardToDrop := rand.Int()%ts.NumShards() + 1
	return ts.DropShard(shardToDrop)
}

func (ts *TestSetup) Shutdown() {
	logrus.WithFields(logrus.Fields{
		"nodes":  len(ts.nodes),
		"shards": len(ts.getShardMapStateCopy().ShardsToNodes),
	}).Debug("shutting down test")
	for _, node := range ts.nodes {
		node.Shutdown()
	}
}

/*
 * Run test with a setup and shuts down the setup after
 */
func RunTestWith(t *testing.T, testFunc func(*testing.T, *TestSetup), setup *TestSetup) {
	testFunc(t, setup)
	setup.Shutdown()
}

/*
 * Makes a set of N nodes labeled n1, n2, n3, ... with fake Address/Port.
 */
func makeNodeInfos(n int) map[string]kv.NodeInfo {
	nodes := make(map[string]kv.NodeInfo)
	for i := 1; i <= n; i++ {
		nodes[fmt.Sprintf("n%d", i)] = kv.NodeInfo{Address: "", Port: int32(i)}
	}
	return nodes
}

func MakeBasicOneShard() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards: 1,
		Nodes:     makeNodeInfos(1),
		ShardsToNodes: map[int][]string{
			1: {"n1"},
		},
	}
}

func MakeMultiShardSingleNode() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards: 5,
		Nodes:     makeNodeInfos(1),
		ShardsToNodes: map[int][]string{
			1: {"n1"},
			2: {"n1"},
			3: {"n1"},
			4: {"n1"},
			5: {"n1"},
		},
	}
}

func MakeNoShardAssigned() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards:     1,
		Nodes:         makeNodeInfos(1),
		ShardsToNodes: map[int][]string{},
	}
}

func MakeSingleNodeHalfShardsAssigned() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards: 8,
		Nodes:     makeNodeInfos(1),
		ShardsToNodes: map[int][]string{
			1: {"n1"},
			2: {"n1"},
			3: {"n1"},
			4: {"n1"},
			5: {},
			6: {},
			7: {},
			8: {},
		},
	}
}

func MakeTwoNodeBothAssignedSingleShard() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards: 1,
		Nodes:     makeNodeInfos(2),
		ShardsToNodes: map[int][]string{
			1: {"n1", "n2"},
		},
	}

}

func MakeTwoNodeMultiShard() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards: 10,
		Nodes:     makeNodeInfos(2),
		ShardsToNodes: map[int][]string{
			1:  {"n1"},
			2:  {"n1"},
			3:  {"n1"},
			4:  {"n1"},
			5:  {"n1"},
			6:  {"n2"},
			7:  {"n2"},
			8:  {"n2"},
			9:  {"n2"},
			10: {"n2"},
		},
	}
}

func MakeFourNodesWithFiveShards() kv.ShardMapState {
	return kv.ShardMapState{
		NumShards: 5,
		Nodes:     makeNodeInfos(4),
		ShardsToNodes: map[int][]string{
			1: {"n1", "n2", "n3"},
			2: {"n1", "n2", "n4"},
			3: {"n2", "n3"},
			4: {"n3", "n4"},
			5: {"n2"},
		},
	}
}

func MakeManyNodesWithManyShards(numShards int, numNodes int) kv.ShardMapState {
	shardsToNodes := make(map[int][]string, numShards)
	nodeNames := make([]string, 0, numNodes)
	for i := 1; i <= numNodes; i++ {
		nodeNames = append(nodeNames, fmt.Sprintf("n%d", i))
	}
	for shard := 1; shard <= numShards; shard++ {
		maxR := 5
		if maxR > numNodes {
			maxR = numNodes
		}
		minR := 2
		if minR > maxR {
			minR = maxR
		}
		// random number in [minR, maxR] (rand.Intn uses half-open interval)
		r := rand.Intn(maxR-minR+1) + minR

		// pick r unique nodes
		replicas := make(map[string]struct{}, 0)
		nodes := make([]string, 0)
		for len(replicas) < r {
			idx := rand.Intn(numNodes)
			node := nodeNames[idx]
			if node == "" {
				panic(fmt.Sprintf("%v %v %v", nodeNames, idx, node))
			}
			// if already picked, skip
			_, present := replicas[node]
			if present {
				continue
			}
			replicas[node] = struct{}{}
			nodes = append(nodes, node)
		}
		shardsToNodes[shard] = nodes
	}
	return kv.ShardMapState{
		NumShards:     numShards,
		Nodes:         makeNodeInfos(numNodes),
		ShardsToNodes: shardsToNodes,
	}
}

func randomString(rng *rand.Rand, length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz"

	out := strings.Builder{}
	for i := 0; i < length; i++ {
		out.WriteByte(chars[rng.Int()%len(chars)])
	}
	return out.String()
}

func RandomKeys(n, length int) []string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	out := make([]string, 0)
	for i := 0; i < n; i++ {
		out = append(out, randomString(rng, length))
	}
	return out
}
