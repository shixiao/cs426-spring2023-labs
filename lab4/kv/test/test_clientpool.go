package kvtest

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"cs426.yale.edu/lab4/kv"
	"cs426.yale.edu/lab4/kv/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type TestClientPool struct {
	mutex           sync.RWMutex
	getClientErrors map[string]error
	nodes           map[string]*TestClient
}

func (cp *TestClientPool) Setup(nodes map[string]*kv.KvServerImpl) {
	cp.nodes = make(map[string]*TestClient)
	for nodeName, server := range nodes {
		cp.nodes[nodeName] = &TestClient{
			server: server,
			err:    nil,
		}
	}
}

func (cp *TestClientPool) GetClient(nodeName string) (proto.KvClient, error) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	err, ok := cp.getClientErrors[nodeName]
	if ok {
		return nil, err
	}

	if cp.nodes == nil {
		return nil, errors.Errorf("node %s node setup yet: test cluster may be starting", nodeName)
	}
	return cp.nodes[nodeName], nil
}

func (cp *TestClientPool) OverrideGetClientError(nodeName string, err error) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	if cp.getClientErrors == nil {
		cp.getClientErrors = make(map[string]error)
	}
	cp.getClientErrors[nodeName] = err
}
func (cp *TestClientPool) ClearGetClientErrors() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	cp.getClientErrors = nil
}
func (cp *TestClientPool) OverrideRpcError(nodeName string, err error) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].OverrideError(err)
}
func (cp *TestClientPool) OverrideGetResponse(nodeName string, val string, wasFound bool) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].OverrideGetResponse(val, wasFound)
}
func (cp *TestClientPool) OverrideSetResponse(nodeName string) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].OverrideSetResponse()
}
func (cp *TestClientPool) OverrideDeleteResponse(nodeName string) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].OverrideDeleteResponse()
}
func (cp *TestClientPool) OverrideGetShardContentsResponse(nodeName string, response *proto.GetShardContentsResponse) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].OverrideGetShardContentsResponse(response)
}
func (cp *TestClientPool) AddLatencyInjection(nodeName string, duration time.Duration) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].SetLatencyInjection(duration)
}
func (cp *TestClientPool) ClearRpcOverrides(nodeName string) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	cp.nodes[nodeName].ClearOverrides()
}
func (cp *TestClientPool) GetRequestsSent(nodeName string) int {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	client := cp.nodes[nodeName]
	return int(atomic.LoadUint64(&client.requestsSent))
}
func (cp *TestClientPool) ClearRequestsSent(nodeName string) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	client := cp.nodes[nodeName]
	atomic.StoreUint64(&client.requestsSent, 0)
}

func (cp *TestClientPool) ClearServerImpls() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	for nodeName := range cp.nodes {
		cp.nodes[nodeName].server = nil
	}
}

type TestClient struct {
	server *kv.KvServerImpl
	// requestsSent is managed atomically so we don't need write locks per request
	requestsSent uint64

	// mutex protects below variables which act as mock responses
	// for testing
	mutex                    sync.RWMutex
	err                      error
	getResponse              *proto.GetResponse
	setResponse              *proto.SetResponse
	deleteResponse           *proto.DeleteResponse
	getShardContentsResponse *proto.GetShardContentsResponse
	latencyInjection         *time.Duration
}

func (c *TestClient) Get(ctx context.Context, req *proto.GetRequest, opts ...grpc.CallOption) (*proto.GetResponse, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	atomic.AddUint64(&c.requestsSent, 1)
	if c.err != nil {
		return nil, c.err
	}
	if c.latencyInjection != nil {
		time.Sleep(*c.latencyInjection)
	}
	if c.getResponse != nil {
		return c.getResponse, nil
	}
	return c.server.Get(ctx, req)
}
func (c *TestClient) Set(ctx context.Context, req *proto.SetRequest, opts ...grpc.CallOption) (*proto.SetResponse, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	atomic.AddUint64(&c.requestsSent, 1)
	if c.err != nil {
		return nil, c.err
	}
	if c.latencyInjection != nil {
		time.Sleep(*c.latencyInjection)
	}
	if c.setResponse != nil {
		return c.setResponse, nil
	}
	return c.server.Set(ctx, req)
}
func (c *TestClient) Delete(ctx context.Context, req *proto.DeleteRequest, opts ...grpc.CallOption) (*proto.DeleteResponse, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	atomic.AddUint64(&c.requestsSent, 1)
	if c.err != nil {
		return nil, c.err
	}
	if c.latencyInjection != nil {
		time.Sleep(*c.latencyInjection)
	}
	if c.deleteResponse != nil {
		return c.deleteResponse, nil
	}
	return c.server.Delete(ctx, req)
}
func (c *TestClient) GetShardContents(ctx context.Context, req *proto.GetShardContentsRequest, opts ...grpc.CallOption) (*proto.GetShardContentsResponse, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	atomic.AddUint64(&c.requestsSent, 1)
	if c.err != nil {
		return nil, c.err
	}
	if c.latencyInjection != nil {
		time.Sleep(*c.latencyInjection)
	}
	if c.getShardContentsResponse != nil {
		return c.getShardContentsResponse, nil
	}
	return c.server.GetShardContents(ctx, req)
}

func (c *TestClient) ClearOverrides() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.err = nil
	c.getResponse = nil
	c.getShardContentsResponse = nil
	c.latencyInjection = nil
}

func (c *TestClient) OverrideError(err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.err = err
}

func (c *TestClient) OverrideGetResponse(val string, wasFound bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.getResponse = &proto.GetResponse{
		Value:    val,
		WasFound: wasFound,
	}
}
func (c *TestClient) OverrideSetResponse() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.setResponse = &proto.SetResponse{}
}
func (c *TestClient) OverrideDeleteResponse() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.deleteResponse = &proto.DeleteResponse{}
}

// Not used by tests, but you may use it in your own tests
func (c *TestClient) OverrideGetShardContentsResponse(response *proto.GetShardContentsResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.getShardContentsResponse = response
}

func (c *TestClient) SetLatencyInjection(duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.latencyInjection = &duration
}
