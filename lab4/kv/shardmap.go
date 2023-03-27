package kv

import (
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type NodeInfo struct {
	Address string `json:"address"`
	Port    int32  `json:"port"`
}

/*
 * Internal state of the ShardMap, which may be updated as
 * shards are moved around.
 */
type ShardMapState struct {
	// Set of nodes mapped by their nodeName
	//
	// Nodes may change -- nodes can be added or removed,
	// but we assume that a given nodeName will only ever
	// refer to a single node (no changing the IP/port for a node)
	Nodes map[string]NodeInfo `json:"nodes"`
	// Mapping of shard to the set of nodes (potentially many or none)
	// which host that shard, listed by nodeName
	ShardsToNodes map[int][]string `json:"shards"`
	// NumShards is constant -- no resharding
	NumShards int `json:"numShards"`
}

/*
 * Whether a ShardMapState is valid.
 */
func (smState ShardMapState) IsValid() bool {
	if len(smState.ShardsToNodes) > smState.NumShards {
		return false
	}
	for shard, nodes := range smState.ShardsToNodes {
		// shard must be 1..NumShards
		if shard < 1 || shard > smState.NumShards {
			return false
		}
		nodeSet := make(map[string]struct{})
		for _, node := range nodes {
			// node must be exist
			_, present := smState.Nodes[node]
			if !present {
				return false
			}
			// cannot have duplicate nodes for a shard
			_, present = nodeSet[node]
			if present {
				return false
			}
			nodeSet[node] = struct{}{}
		}
	}
	return true
}

/*
 * Dynamically configurable ShardMap -- contains Nodes in the cluster
 * and a map of which shards are assigned to which node.
 *
 * The internal state of the ShardMap can be updated at any time
 * using Update(). All external methods are thread-safe.
 *
 * You will primarily interact with ShardMap in one of two ways:
 *  1. Reading state using internal state which is always up-to-date,
 *     like by calling shardMap.ShardsForNode(nodeName).
 *  2. Reacting to changes by getting updates from a ShardMapListener.
 *
 * ShardMap is designed so that many components can "listen" for updates
 * and process changes. For instance, your server may want to listen for
 * ShardMap changes and then delete data for shards that are no longer assigned.
 *
 * To do so, create a ShardMapListener via shardMap.MakeListener(),
 * and read updates on shardMapListener.UpdateChannel(). This channel will
 * receive a notification any time Update() is called, at which point you
 * may read the latest state by calling Nodes(), ShardsForNode(), etc.
 *
 * Note that all shards are 1-indexed in this lab. For instance, given
 * NumShards of 5, the shards [1, 2, 3, 4, 5] are valid.
 */
type ShardMap struct {
	// Internally we store the state as an atomic pointer, which allows us to
	// swap in a new value with a single atomic operation and avoid locks on
	// most codepaths
	state atomic.Value

	// mutex protects the set of `updateChannels` which we send update notifications
	// to whenever Update() is called with a new state
	mutex sync.RWMutex
	// `updateChannels` is effectively a set of notification channels -- these channels
	// carry no data, just a signal that is passed whenever the state updates.
	//
	// We store them by an integer key so that they can be removed later
	// if the ShardMapListener is Closed
	updateChannels map[int]chan struct{}
	// Trivial counter which is used to generate unique keys into `updateChannels`
	// whenever a new Listener is created
	updateChannelCounter int
}

/*
 * Gets a pointer to the latest internal state via a thread-safe atomic load.
 */
func (sm *ShardMap) GetState() *ShardMapState {
	state := sm.state.Load().(*ShardMapState)
	return state
}

/*
 * Gets the set of nodes for the entire cluster keyed by nodeName.
 * To access a single node, use Nodes()[nodeName].
 */
func (sm *ShardMap) Nodes() map[string]NodeInfo {
	return sm.GetState().Nodes
}

/*
 * Gets the total number of shards for all keys in the cluster. Note that
 * shards may not necessarily be assigned to any nodes in failure cases.
 *
 * You may assume that NumShards() does not ever change.
 */
func (sm *ShardMap) NumShards() int {
	return sm.GetState().NumShards
}

/*
 * Gets the set of integer shards assigned to a given node (by node name).
 */
func (sm *ShardMap) ShardsForNode(nodeName string) []int {
	state := sm.GetState()
	shards := make([]int, 0)
	for shard, nodes := range state.ShardsToNodes {
		for _, node := range nodes {
			if node == nodeName {
				shards = append(shards, shard)
			}
		}
	}
	return shards
}

/*
 * Gets the set of nodes that currently host a given shard, returned
 * by node name. If you need full information, use the node name
 * as a key to query the Nodes() map.
 */
func (sm *ShardMap) NodesForShard(shard int) []string {
	state := sm.GetState()
	nodeNames, ok := state.ShardsToNodes[shard]

	if !ok {
		return []string{}
	}
	return nodeNames
}

/*
 * Create a Listener for this ShardMap, which gets notifications
 * on a channel whenever Update() is called to change the ShardMap
 * state. Listeners can be used to react to changes (invalidate cache,
 * dynamically add/remove shards, etc).
 *
 * Listeners must be shutdown safely with their Close() method.
 *
 * This method is safe to call from many threads.
 */
func (sm *ShardMap) MakeListener() ShardMapListener {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	ch := make(chan struct{})
	id := sm.updateChannelCounter
	if sm.updateChannels == nil {
		sm.updateChannels = make(map[int]chan struct{})
	}

	sm.updateChannelCounter += 1
	sm.updateChannels[id] = ch
	return ShardMapListener{
		shardMap: sm,
		ch:       ch,
		id:       id,
	}
}

/*
 * Update the ShardMap internal state and notify all active Listeners.
 * This method is safe to call from many threads, but may block until
 * listeners receive the notification.
 */
func (sm *ShardMap) Update(state *ShardMapState) {
	logrus.Trace("updating shardmap state")

	sm.state.Swap(state)
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	for _, ch := range sm.updateChannels {
		ch <- struct{}{}
	}
}

/*
 * A single listener for updates to a given ShardMap. Updates
 * are given as notifications on a dummy channel.
 *
 * To use the updated value, simply read it from the underlying
 * ShardMap (e.g. via Nodes()/NodesForShard(), etc).
 *
 * ShardMapListener.Close() must be called to safely stop
 * listening for updates.
 */
type ShardMapListener struct {
	shardMap *ShardMap
	ch       chan struct{}
	id       int
}

func (listener *ShardMapListener) UpdateChannel() chan struct{} {
	return listener.ch
}

func (listener *ShardMapListener) Close() {
	listener.shardMap.mutex.Lock()
	defer listener.shardMap.mutex.Unlock()

	delete(listener.shardMap.updateChannels, listener.id)
	close(listener.ch)
}
