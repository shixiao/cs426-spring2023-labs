package kv

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"cs426.yale.edu/lab4/kv/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func makeConnection(addr string) (proto.KvClient, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	channel, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return proto.NewKvClient(channel), nil
}

/*
 * ClientPool is the main interface you will be using to call into the KvServers,
 * both from the client (Kv in client.go) and from the servers (when you implement
 * shard copying).
 *
 * Clients are cached by nodeName. We assume that the connection information (Address/Port)
 * will never change for a given nodeName.
 *
 * It is important to use ClientPool::GetClient() instead of your own logic
 * because unit-tests will use a mocked version of ClientPool to change behaviors, test with
 * failure injection, etc.
 */
type ClientPool interface {
	/*
	 * Returns a KvClient for a given node if one can be created. Returns (nil, err)
	 * otherwise. Errors are not cached, so subsequent calls may return a valid KvClient.
	 */
	GetClient(nodeName string) (proto.KvClient, error)
}

type GrpcClientPool struct {
	shardMap *ShardMap

	mutex   sync.RWMutex
	clients map[string]proto.KvClient
}

func MakeClientPool(shardMap *ShardMap) GrpcClientPool {
	return GrpcClientPool{shardMap: shardMap, clients: make(map[string]proto.KvClient)}
}

func (pool *GrpcClientPool) GetClient(nodeName string) (proto.KvClient, error) {
	// Optimistic read -- most cases we will have already cached the client, so
	// only take a read lock to maximize concurrency here
	pool.mutex.RLock()
	client, ok := pool.clients[nodeName]
	pool.mutex.RUnlock()
	if ok {
		return client, nil
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	// We may have lost a race and someone already created a client, try again
	// while holding the exclusive lock
	client, ok = pool.clients[nodeName]
	if ok {
		return client, nil
	}

	nodeInfo, ok := pool.shardMap.Nodes()[nodeName]
	if !ok {
		logrus.WithField("node", nodeName).Errorf("unknown nodename passed to GetClient")
		return nil, fmt.Errorf("no node named: %s", nodeName)
	}

	// Otherwise create the client: gRPC expects an address of the form "ip:port"
	address := fmt.Sprintf("%s:%d", nodeInfo.Address, nodeInfo.Port)
	client, err := makeConnection(address)
	if err != nil {
		logrus.WithField("node", nodeName).Debugf("failed to connect to node %s (%s): %q", nodeName, address, err)
		return nil, err
	}
	pool.clients[nodeName] = client
	return client, nil
}
