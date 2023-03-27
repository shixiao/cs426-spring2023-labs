# CS426 Lab 4: Sharded and replicated key-value cache

## Overview
In this lab, you will implement a key-value cache like Memcache or a simplified Redis.
Your key-value cache will only support 3 external APIs:
 - `Get(key)` -> value
 - `Set(key, value, ttl)`
 - `Delete(key)`

Your key-value cache will be partitioned into many shards, and each shard may be replicated
to one or many servers. This shard mapping will be *dynamic* -- shards may move between servers
at any time.

You will implement both the server (including in-memory, thread-safe storage) *and* the client logic
(including key hashing, retries, error handling). The client and servers will share a common shard map
which defines the nodes running instances of the server, and which shards are assigned to each server.

Similar to lab 1, you will be using gRPC for the client and server implementations.

## Logistics
**Policies**
- Lab 4 is our first group assignment. You may choose to complete it individually or in a group of up to 3. See "Working as a group" below.
- We will help you strategize how to debug but WE WILL NOT DEBUG YOUR CODE FOR YOU.
- Please keep and submit a time log **per person** of time spent and major challenges you've encountered. This may be familiar to you if you've taken CS323. See [Time logging](#time-logging) for details.

- Questions? post to Canvas or email the teaching staff at cs426ta@cs.yale.edu.
  - Xiao Shi (xiao.shi@aya.yale.edu)
  - Mahdi Soleimani (mahdi.soleimani@yale.edu)
  - Ross Johnson (ross.johnson@yale.edu)
  - Richard Yang (yry@cs.yale.edu)

**Submission deadline: 23:59 ET Wednesday April 12, 2023**

**Submission logistics** Submit a `.tar.gz` archive named after your NetID via
Canvas. The Canvas assignment will be up a day or two before the deadline.

Only one student needs to submit for a group. Include every
group member's NetID in a text file `net-ids.txt` with one per line.

Be sure to run `go vet` and fix any potential lints or warnings before
submitting your code.

Your submission for this lab should include the following files:
```
kv/server.go
kv/client.go
kv/util.go
kv/test/extra_test.go
discussions.md
go.mod
<net-id1>.time.log
<net-id2>.time.log // if group
<net-id3>.time.log // if group
net-ids.txt // if group
```

We will accept the same late policy as all labs starting with lab 2. You
have 72 discretionary hours to use across all labs this semester and
still receive full credit. Past the 72 discretionary hours we will accept
Dean's excuse.

We will also keep submissions open up to one week past the full-credit
deadline (including the 72 hours, if still available). Submissions
past the deadline but within the one week limit will be graded
with a penalty of 30% (e.g. 90% * 70% = 63%).

## Working as a group

This lab is intentionally designed so that work can be done in parallel by a few
people (up to three). Lab 5 (final project) may also be completed in a group
(details [here](https://docs.google.com/document/d/1694jDeCSZGjjKd7LjFNTfHUWFBSlz1BaTDkpEsHvfOU/edit?usp=sharing)). You always have the option to complete either lab
individually, however, you **cannot** be in two _different_ groups for Lab 4 and
Lab 5. In other words, unless you choose to complete lab 4/5 individually, you
must be in the same group. Additionally, everyone in the group will get the
same grade for this lab, so pick your group wisely.

You are free to divide work however you see fit (including
just pair programming together!). If you decide to parallelize, some suggested
chunks which can be divided up are:
 - Basic server storage implementation, which can be tested in isolation with
	a few of the tests from server_test.go without any other parts implemented
 - TTL implementation for the server, which can be tested with the later parts
	of server_test.go without any client implementation
 - Sharding logic
 - Basic client implementation, which can be tested by client_test.go without
   a working server implementation
 - Shard migrations between servers, which can be tested by server_test.go and
   integration_test.go but several tests do not require other parts (e.g. TTL)

Note that we list parts A (server), B (client), and C (shard migrations) in a specific
order, but you are free to work on them in any order you see fit.

**Important**: If you work as a group, include a section at the end of your `discussions.md` under the heading
**Group work** which summarizes how each team member contributed to the completion of this lab.

We recommend that you use a git repo to collaborate (e.g., on github), but please make your repo **private** especially if it contains code to prior labs.

## Setup
 1. Check out the repository like you did for previous labs lab 0, or pull to update: https://github.com/shixiao/cs426-spring2023-labs
 2. [Optional] The repo has already run the protobuf compiler to generate the binding code, same as lab 1. If you want to regenerate it, run `make` to build from the `Makefile`
	or check out its contents to learn more.
    * Note: you will need to install `protoc`, the protobuf compiler, to get started. You can find instructions in the gPRC documentation: https://grpc.io/docs/protoc-installation/
    * Additionally, you will need to install the Go plugins, following these steps: https://grpc.io/docs/languages/go/quickstart/
 3. This lab will be using Go, as with others. You will primarily be running tests in the `kv/test/` directory with `go test` (or `go test -v -run=TestClient` like in lab 0 and lab 1). You will not need extra tools like `kubectl` or `docker`.

## Use of Go libraries

You will likely not need any additional external libraries other than the ones
already in use (gRPC, logrus, testify, etc). If you wish to use one, please
post on Ed. We will not accept libraries which make substantial parts of
the assignment trivial (for example, no pre-made thread-safe caches).

If using an approved library, add it to your `go.mod` as usual, which you will
submit as part of the assignment.

# Background
## KV: The Key-Value API

By the end of the lab, you will produce an implementation of "KV" as an API for developers to use. This KV API
is a *client* API, and hides all the details of sharding, replication, etc. from the user. To do this you'll implement
not only the server to store the data, but also a "thick" client library which handles sharding, routing, and failover.

The user-facing API is all in `client.go` -- you will implement 3 functions here, and corresponding server code
for those 3 APIs as well.

```
func (kv *Kv) Get(ctx context.Context, key string) (string, bool, error) { }
func (kv *Kv) Set(ctx context.Context, key string, value string, ttl time.Duration) error { }
func (kv *Kv) Delete(ctx context.Context, key string) error { }
```

`Get()` takes a string key and returns the value stored in `KV` if one exists. The function returns three things:
 - `value`, a `string`. This is the value stored in `KV` if one exists and can be fetched at all. In case no key is currently stored or an error happens, it should be the empty string `""`.
 - `wasFound`, a `bool`. This is set to true if a value was found successfully in `KV`, and false in other cases (including errors). This bool is here to distinguish between cases where KV stored a valid value but that value was `""`. For example, `("", true, nil)` may be returned in the case that `KV` actually had a value stored at that key but the value was `""`. This differentiates from a return value of `("", false, nil)` which indicates that *no* value was stored for the key (or the value was deleted or expired).
 - `err`, an `error`. Like most cases you've seen so far, if `Get()` fails it can return an error. In those cases, it should return `("", false, err)`. In all other cases, `err` should be `nil`.

`Set()` takes three parameters: a key, a value, and a TTL. It sets the value on all replicas of the shard for the key.
It returns `nil` in successful cases, and returns a `error` if it fails.
All keys in `KV` have an associated TTL, and values are never returned past their TTL. For example:
`Set("abc", "123", 1 * time.Second)` sets a key that will expire in one second. Any calls to `Get("abc")`
more than one second in the future should return no value (and a false value for `wasFound`) unless
a future `Set` is called to set a new value or overwrite the value with a new TTL.
Repeated calls to `Set` the same key function as a strict overwrite on both the value and TTL.

`Delete()` takes a single parameter: a key. It removes the value from all replicas of the shard for the key.
This function succeeds even if no value is currently stored at the key.
`Delete()` essentially invalidates a given key eagerly -- even if it is before the TTL. Future calls
to `Get(key)` should return no value (and `wasFound` of `false`) unless a subsequent `Set`
inserts a new value.

These three client-facing APIs have corresponding gRPC APIs, defined in `kv/proto/kv.proto`:
```
service Kv {
	rpc Get(GetRequest) returns (GetResponse);
	rpc Set(SetRequest) returns (SetResponse);
	rpc Delete(DeleteRequest) returns (DeleteResponse);

	// Used in part C, but not exposed to the client
	rpc GetShardContents(GetShardContentsRequest) returns (GetShardContentsResponse);
}
```

You will use these gRPC APIs to implement the client-facing API in `client.go` -- using a `KvClient`
to talk to instances of the `KvServer` that you implement (which will actually store the data).

## ShardMap: Handling dynamic sharding

The `ShardMap` is the primary configuration for operating KV. It defines the set of nodes by name (the "nodeName") and
their address information (IP and port). It also defines the mapping of which shards are on which node, and how many total
shards there are. For example:


```
{
	"numShards": 5,
	"nodes": {
		"n1": {
			"address": "127.0.0.1",
			"port": 9001
		},
		"n2": {
			"address": "127.0.0.1",
			"port": 9002
		},
		"n3": {
			"address": "127.0.0.1",
			"port": 9003
		}
	},
	"shards": {
		"1": ["n1", "n3"],
		"2": ["n1"],
		"3": ["n2", "n3"]
		"4": ["n1", "n2", "n3"],
		"5": ["n2"],
	}
}
```

This configures a cluster with 3 nodes and 5 shards. Shard 1 maps to *both* node 1 and node 3 (i.e. it has a replication factor of 2) --
so all writes to shard 1 must be sent to both
N1 and N3. Reads to shard 1 may be sent to either node 1 or node 3 -- you will implement
primitive load-balancing in **Part B**. Note that in this lab
we will not deal with partial failures or divergent replicas:
`KV` operates like a cache so we assume it is ok if some data is
missing due to partial failures.

For the purpose of this lab, you may assume that `NumShards` is a constant that never changes once the cluster is set up, and that the shards are named `1`, `2`, ..., `NumShards`. However, this sharding scheme is *dynamic* -- shards may be moved between nodes.
The provided `ShardMap` implemention will wrap this configuration structure for you and
provide notifications (via a `chan`) when the shard map changes. Your `KvServerImpl`s will
implement a shard migration protocol in **Part C** to handle this without losing data.

Your client implementation (`Kv` in `client.go`) will use this shard map to find which server to route to.
For example, given a `Get("abc")` request, you will
 1. Compute the shard (by hashing `"abc"`)
 2. Look up which nodes host that shard (by name), e.g. if "abc" maps to shard 3, this would be "n2" and "n3"
 3. Use the existing `ClientPool` to map the nodeName to a `KvClient` (which uses the address / port given)
 4. Use the `KvClient` to send a request to the server

Your server will also use the `ShardMap` to ensure that requests sent to it are
valid. For example, if `n1` were to receive a `Get()` request for some key (e.g. "123")
which maps to shard `5`, it would reject the request because the key is not hosted on
any of the shards hosted by `n1`. This is a safety feature against misrouted requests
(from broken clients, or clients with a stale shard map).

## ClientPool: managing `KvClients`
`ClientPool` will be your primary interface for using `KvClient`s for this lab.
The implementation in `GrpcClientPool` will pool and reuse clients for you automatically,
so you don't need to worry about connection management for this lab.

You can view the documentation in `clientpool.go`, but practically `ClientPool`
provides one interface method for you: `GetClient(nodeName) -> (KvClient, error)`
This functions like a cached version of `grpc.Dial` combined with reading the
`ShardMap` to map the `nodeName` to the address and port. This means you will
likely not need to deal with `grpc.Dial()`, IPs/ports, or other connection
functionality like you did in lab 1 (and lab 3) as it should be handled
for you by the starter code in `GrpcClientPool`.

Importantly, you *must* use `ClientPool` for all `KvClient`s because it is how we
hook in the test harness for testing clients and servers separately. When testing
we use `TestClientPool` which removes networking overhead and allows us to inject
responses and failures to aid in unit testing. You may use `TestClientPool` when
implementing your own unit tests as well.

## Logging with levels

We've imported a leveled logging implementation ([Logrus](https://github.com/sirupsen/logrus))
and used it in this project. It provides levels instead of just a single Printf/Print
function, which can help filter severity of events. For example, you can log
very bad things with `logrus.Errorf()` to be output at the ERROR level, or
very unimportant things with `logrus.Debugf()` to be output at the DEBUG level.

You can use the `-log-level=debug` flag to set the minimum log level for the messages
to be output. By default this is `INFO` -- so only `logrus.Info()` or higher (meaning info, warn, error, or fatal/panic)
are output.

When writing your code you can sprinkle in `Debug()` or even `Trace()` (lowest level)
statements which are not output by default, but can be turned on when you want
to debug a test. For example, if a test fails you may run something like

```
go test -run=TestServerXyz -v -log-level=trace
```

You are free to use `logrus` with the existing flags or use the
standard library `log` package (or just write to stderr directly if you really want to).
We recommend adding log statements for edge cases as you go along to aid in debugging,
but they are not required. Read more about Logrus (including structured logging
with fields) in the [official docs](https://github.com/sirupsen/logrus).

If using `logrus`, we also provide a flag `-log-file=path/to/file` which outputs
to a file instead of `stderr`, for easier retrieval later.

# Part A. Server implementation

**Note**: This is listed first, but you can work out of order from the spec where it makes sense. You'll need
most of Part A done to start Part C, but Part B can be done out of order or concurrently with Part A,
and Part A4 (TTLs) can be done later as well. Many tests are designed to touch independent parts of your
code (e.g. only the server or only the client), so you should get some early signal as tests start passing.

For `KV` to function it must have some servers which store the key-value state. For this lab
you'll use simple in-memory structures like the cache from lab 1. Each node of `KV` runs
an instance of `KvServer` which stores some shards of the overall state. You'll start your
implementation of the server functionality in `server.go` by designing the in-memory state
(as members of the `KvServerImpl` struct) and implementing the gRPC API.

The server implementation will be implementing the gRPC API in `kv/proto/kv.proto`, similar to
how you implemented `VideoRecService` in Lab 1. Read through `kv.proto` thoroughly to understand
the messages and how they map to the user-facing API. You can ignore `GetShardContents` for now,
as you will deal with it in Part C.

## A1. Storing state

Your first task will be to design an in-memory data structure to actually store keys and values.
Functionally, this is like a `map[string]string` but with some extra requirements:


Requirements:
 1. All accesses to your state must be thread-safe. You can assume that concurrent calls are coming from
	multiple goroutines.
 2. You must allow concurrency at *some* level. For example, reads to unrelated keys
    can be handled concurrently using `sync.RWMutex` (assuming no writes). A single
	global mutex is insufficient (even if tests pass). Striping (as in lab 0) or partitioning
	your locks by shard is preferred to improve concurrency.
 3. Eventually you will need to support expiring values after their TTLs (A4), so you may consider that in your initial design.
	At the very least, you will need to store the expiry time alongside the key/value.

Get started by reading the existing `KvServerImpl` struct. This is the state for one instance of `KvServer`,
so you will be adding any fields here.
You can add any `struct` definitions you want to `server.go`, if you want to encapsulate parts of your data.

Some tips for your design:
 1. Your server must never return expired data (past the TTL). This does not necessarily mean it has to *delete* it immediately.
    You can combine a filtering strategy (ignoring data in cache that is past TTL on Get calls) with a more asynchronous process
	that deletes the data. If you do so, your asynchronous process must not be too slow -- typically it should delete
	data within a few seconds of the TTL. See `TestServerTtlDataActuallyCleanedUp`.
 2. You can store state however you like, but partitioning your data in memory by shard up front may help
    implement shard migrations in Part C.
 3. Start simple when handling concurrency and save your progress (e.g. as `git` commits). You can
    start with a non-thread-safe implementation and pass the non-concurrent tests, then add in a
	single mutex to pass the concurrent tests, then implement anything fancier
	(read/write locks, atomics, striping, etc) afterwards. You can continue on to later parts (A2, A3)
	with just a simple implementation here and take note of the test progress, then go back to improve it.
 4. You may assume that the "ideal" workload for `KV` is read heavy if you're making
	any design trade-offs (even if the small tests are not necessarily read dominant).

In the likely case that you need to *initialize* your data, modify `MakeKvServer` in `server.go`
which is called to construct an instance of `KvServerImpl`.

Once you have a rough idea of the state stored, you can start by implementing the `Set()` method
in `server.go`:

```
func (server *KvServerImpl) Set(ctx context.Context, request *proto.SetRequest) (*proto.SetResponse, error) {
	// ...
}
```

Semantics for set:
 1. All writes are a blind overwrite. If a value is already set, it is overwritten.
 2. For part A4, TTLs are also overwritten, not extended. If an item currently expires in 10s but the new TTL is 1s, it expires in 1s.
 3. Empty values are allowed and should be stored, differentiating from an empty stored value and no value stored.
 4. Empty keys are not allowed (error with INVALID_ARGUMENT).

## A2. Reading state and handling deletes

Assuming you've designed an appropriate data structure, you can now implement the other
two APIs: `Get()` and `Delete()`. You should find stubs already in place in `server.go`.

Semantics for `Get`:
 1. If a value existed and is not expired, return the value as `Value` and `WasFound: true`.
 2. It is not an error if the value does not exist, return `Value` as `""` with `WasFound: false`.
 3. Empty values that are stored are valid -- return `WasFound: true` and `Value: ""`.
 4. Empty keys are invalid, as with `Set`. Return `INVALID_ARGUMENT`.

Semantics for `Delete`:
 1. Deletes should clean up storage immediately.
 2. Deletes may happen even if the item has not yet expired.
 3. Deletes are successful in the case that no value currently exists, or if the item has expired. They are idempotent -- calling delete twice to the same key is fine.
 4. Empty keys are invalid, as with `Set`. Return `INVALID_ARGUMENT`.

### Testing

If you've implemented this much, even without thread-safety or concurrency, you should be able to pass some of the server tests.
`cd` into `kv/test/` and run `go test -run=TestServer`. Depending on what you've implemented so far, look at the results
of the following tests:
 - TestServerBasicUnsharded
 - TestServerOverwriteUnsharded
 - TestServerMultiKeyUnsharded

These tests do not depend on handling concurrent requests correctly, and only test Get/Set/Delete without
any sharding logic. Once you add thread-safety, the following tests should pass:
 - TestServerConcurrentGetsSingleKeyUnsharded
 - TestServerConcurrentGetsMultiKeyUnsharded
 - TestServerConcurrentGetsAndSetsUnsharded

Run these tests again with `go test -race -run='TestServerConcurrent.*Unsharded'` to look for any flagged
race conditions.

You may also pass many of the `TestServerMultiShardSingleNode` tests without any extra effort.
Particularly if you don't look at shards at all yet, this test assigns all the shards to a
single node (which in effect is an unsharded system). This test may break once you add sharding
logic in A3, so come back to it to find bugs.

We have tried to add comments to test cases to aid in debugging, so if you fail a test please
read through the code and comments to gain a beter understanding of what is going on.

### Writing your own tests

Like previous labs, we expect you to be testing as you complete the assignment. You can
add any extra test cases you want while you're working through to `extra_test.go` in `kv/test`.

By the time you finish the lab, you must have written at least **5 additional test cases**. They
can test the server, the client, or both together -- you can take a look at example
tests in `server_test.go`, `client_test.go`, or `integration_test.go` respectively.

Consider using these tests to help narrow down bugs you find in other tests -- if you fail
a large or complicated test, try to reproduce it with a smaller unit test so you can fix
it more quickly. Similarly, if you find a part of the written spec which is not covered by a test,
consider writing one to make sure you handle edge cases appropriately.

## A3. Considering the shard map

Up to this point you could have made progress without considering shards or the shard map at all!
The tests from A2 send requests to a single server which has shards assigned. The first step you'll
take towards sharding your system is adding a small safety feature to your server: rejecting requests
to unhosted shards.

Requests may be sent to the wrong server for a number of reasons -- the client may have a stale view
of the shardmap, the client may be misconfigured, or the request was simply racing with a shard map update.

To start with this safety feature, you'll first need a sharding function. We recommend defining a function
like `GetShardForKey(key string, numShards int) int` in `utils.go` that you can re-use across client
and server implementation. We have provided a basic implementation that you can use if you want, but you can use
any hash function you want as long as it distributes well across
the number of shards. (Recall from the [ShardMap background](#shardmap-handling-dynamic-sharding) section above that `NumShards` is a constant.)

Next, on all requests to your server, compute the shard for the key in the request. If the server
receiving the request does not host that shard, return an error with code [NotFound](https://pkg.go.dev/google.golang.org/grpc@v1.45.0/codes#NotFound).

You have a few strategies for determining if the server hosts the shard:
 1. You can use the ShardMap to trivially compute if the node currently hosts the shard, without
	storing any extra state in your server for now.
 2. You can fetch the set of hosted shards for the node in `MakeKvServer` and store it in your
	`KvServerImpl`. You can keep this up-to-date using a `ShardMapListener` to process `ShardMap`
	updates.

Strategy #1 is a fine starting point for now, but you will likely need a strategy more similar to #2
to begin handling shard migrations in Part C.

### Testing

```
go test -v -run=TestServerReject
```

If you've implemented this correctly, you should now pass the following tests:
 - TestServerRejectOpsWithoutShardAssignment
 - TestServerRejectWithoutShardAssignmentMultiShard

You should also pass the non-TTL sub-tests of `TestServerMultiShardSingleNode` if you have not already.

If you implemented a strategy that uses `ShardMap` directly (#1 from above) or processes updates,
you may also pass TestServerSingleShardDrop.

The remaining tests for `TestServer` require TTL support (A4) or shard migration support (C).

## A4. TTLs

To properly support invalidation, you will implement TTL support for the `KV` server.
All `Set` requests come with an attached `ttlMs`, which is a time duration in milliseconds.

`Get` requests processed by the server which occur more than `ttlMs` after the set finishes
*must* return no data (`wasFound: false`). Note the tip from part A1 though -- just because
`Get` requests must return no data promptly after the TTL does not mean your server must
immediately clean up the data if it is not read. You can employ an asynchronous or garbage
collector style strategy. Also note the semantics on overwriting TTLs from A1.

For this you may need to track additional state in your server such as expiry times next to values, or
lists of items which need to be evicted at certain times. If you spawn any additional goroutines,
you must cancel them or signal them to terminate in `KvServer.Shutdown()`.

### Testing

```
go test -v -run='TestServerTtl'
go test -v -run='TestServerMultiShardSingleNode/ttl'
```

You should now pass:
 - TestServerTtlUnsharded
 - TestServerTtlMultiKeyUnsharded
 - TestServerMultiShardSingleNode/ttl
 - TestServerMultiShardSingleNode/ttl-multikey
 - TestServerTtlDataActuallyCleanedUp

Note that these tests (particularly `TestServerTtlDataActuallyCleanedUp`) are slow because
they naively wait for the TTL of items to pass, so this section may take 20-30s to run.

### Discussion

What tradeoffs did you make with your TTL strategy? How fast does it clean up data and how
expensive is it in terms of number of keys?

If the server was write intensive, would you design it differently? How so?

Note your responses under a heading **A4** in your `discussions.md`.

# Part B. Client implementation

`client.go` has a similar structure to `server.go`. Here you will
be implementing three methods: `Get`, `Set`, and `Delete`. You can
store any additional needed client state in `Kv`, and modify
`MakeKv` to do any initialization as needed.

Your `Kv` implementation must also be safe for concurrent
calls to any methods, so if you add any state to the `Kv`
`struct`, ensure that they are handled in a thread-safe
manner (e.g. with mutexes, atomics, etc).

## B1. Get

You can start implementing the KV API with `Kv.Get` which reads
a single key from the cluster of nodes.

To begin, find the nodes which host the key in the request. You'll
need to calculate the appropriate shard (re-using the
same sharding function from Part A3 if you've done that).

Use the provided `ShardMap` instance in `Kv.shardMap` to find the set of
nodes which host the shard. For now (though see B2 and B3), pick any node
name which hosts the shard and use the provided `ClientPool.GetClient` to get
a `KvClient` to use to send the request. You'll use `gRPC` similarly to lab 1:
create a `GetRequest` and send it with `KvClient.Get`.

`Get` semantics from the client:
 1. If a request to a node is successful, return `(Value, WasFound, nil)`
 2. Requests where `WasFound == false` are still considered successful.
 3. For now (though see B3), any errors returned from `GetClient` or the RPC
	can be propagated back to the caller as `("", false, err)`.
 4. If no nodes are available (none host the shard), return an error.

### Testing
For Part B, you'll be running unit tests in `client.go`. These mock out
all server responses (or inject errors), so you do not need a working server
implementation to test them.

If you've implemented all parts of B1 correctly, you should pass a few tests already:
```
go test -run='TestClient(GetSingle|GetClientError|MultiNodeGet$)' -v
```

You may pass some further tests too, depending on your implementation decisions.

## B2. Load balancing Get calls

If there are multiple replicas of a shard (shard assigned to multiple nodes),
then the client has a choice -- which node does it send the RPC to? If you always
pick the same node to send Get calls to, then the load may be imbalanced across
your nodes.

To rememdy this, you'll implement load balancing. You may implement any strategy
which effectively spreads load among nodes which host a shard. Two potential options:
 1. Random load balancing: pick a random node to send the request to. Given enough requests and
    a sufficiently good randomization, this spreads the load among the nodes fairly.
 2. Round robin load balancing: given nodes N_1..N_m, send the first request to N_1,
    next to N_2, next to N_3, ... and wrap around back to N_1 after N_m.

### Testing
```
go test -run='TestClientMultiNodeGetLoadBalance' -v
```

You can test your load balancing strategy with the provided unit tests. It does
not particularly care how you balance, but does expect it to be relatively fair
(no more skewed from 50/50 on 2 nodes than 35/65).

### Discussion

What flaws could you see in the load balancing strategy you chose?
Consider cases where nodes may naturally have load imbalance already -- a node may have more CPUs available, or a node may have more shards assigned.

What other strategies could help in this scenario? You may assume you can make any
changes to the protocol or cluster operation.

Note your answers down under a heading **B2** in your `discussions.md`.

## B3. Failovers and error handling

Aside from load balancing, another benefit to having multiple nodes host the same
shard is **error handling**: if one node is down, we can failover `Get()` calls that
fail to another node. This provides us with *redundancy* in the presence of node failures,
we can still serve cached reads.

Modify your node selection logic in `Get` to handle errors:
 1. If `ClientPool.GetClient()` fails *or* `KvClient.Get()` fails for a single node, try another node.
 2. Order of nodes selected does not matter.
 3. Do not try the same node twice for the same call to `Get`.
 4. Return the first successful response from any node in the set.
 5. If no node returns a successful response, return the last error heard from any node.
 6. You must try all nodes before returning an error, but do not try additional nodes if you hear a single success.

### Testing
```
go test -run='TestClientMultiNodeGet(Failover|FullFailure)' -v
```

If you've completed this far you should pass the remaining Get-only client tests.

### Discussion

For this lab you will try every node until one succeeds. What flaws do you see in this strategy?
What could you do to help in these scenarios?

Note your reply down under heading **B3** in `discussions.md`.

## B4. Set

With `Get` calls fetching from effectively any node in the cluster which hosts the
shard for the key, your `Set` calls will need to fan out to *all* nodes which host
the shard. Otherwise the process is similar -- calculate the shard,
use the ShardMap to find all nodes which host the shard, and use `ClientPool` to get
clients to those nodes.

Semantics:
 1. `Set` calls must be sent to all nodes which host the shard, if there are many.
 2. `Set` calls must be sent *concurrently*, not in serial.
 3. If *any* node fails (in `GetClient()` or in the `Set()` RPC), return the error eventually, but still send the requests to all nodes.
 4. If multiple nodes fail, you may return *any* error heard (or an error which wraps them).

### Discussion

What are the ramifications of partial failures on `Set` calls? What
anomalies could a client observe?

Note your answer down under heading **B4** in `discussions.md`.

## B5. Delete
To complete the KV API, you'll implement the final part -- `Delete`.

Semantics for delete are the same as `Set` calls:
 1. `Delete` calls must be sent to all nodes which host the shard, and sent concurrently.
 2. If *any* node fails return an error, following the saame rules as `Set`.

### Testing
If you've finished all of Part B, you should be able to pass all of the tests in `client.go` now:

```
go test -run=TestClient -v
```

# Part C. Shard migrations

With Part A and Part B implemented, you should have a fully functioning sharded and replicated
key-value cache as long as the shard map does not change. In this section you will implement
dynamic shard migrations for `KV` servers to support online changes in the shard map.

In a production setting, you may have events where you need to change the shard placement:
servers need to be drained for maintenance, or some shards may become hot and need to be
moved to larger servers or away from other hot shards. The goal of a shard migration
protocol is to minimize the impact of these changes to clients of your service.

You'll start by implementing another RPC on `KvServer`: `GetShardContents` which can be used
to fetch *all* data hosted for a single shard. You'll then use this in combination with
a `ShardMapListener` to react to changes -- removing data when shards are removed from a server,
and copying data via `GetShardContents` RPCs when shards are added.

## C1. Using ShardMapListener and handling shard changes

To properly handle shard migrations, you'll need use a `ShardMapListener` to handle
changes. You'll move back to `server.go` (no changes should be
needed to `client.go` for Part C) to do this.

A `ShardMapListener` provides notifications to you via a Go [channel](https://gobyexample.com/channels).
Note that it does not provided the updated value, but instead expects you to read the new
state directly from the `ShardMap`. We've already wired the shardmap listener for you
in `server.go` to a background goroutine.

Start working on `handleShardMapUpdate` to process notifications:
 1. Updates must be processed immediately after receiving a notification via the update channel.
 2. Shards may be added and removed as part of the same update.
 3. Only one update can be processed at a time -- do not read from the update channel again until shards are totally added or dropped from the previous update. This restriction is required for the unit tests.

You may need to make changes to your `KvServerImpl` to actually *track* the set of shards
which you have added already, and process updates to that set in `handleShardMapUpdate`.

We recommend computing the set of shards to remove and the set of shards to add first, then processing
each of those updates independently and concurrently. Note that this may require some changes to your
data structures (in `KvServerImpl` and potentially initializing in `MakeKvServer`) to track the set
of shards you have already added.

With the set of shards to add:
 - Note the shard as tracked, so that future requests can succeed.
 - Leave a placeholder for your shard copy implementation (C3).

With the set of shards to remove:
 - Data must be cleaned up synchronously, or at least appear to be synchronous to clients -- future `Get` calls must not return data.

If you used a simple strategy of reading the `ShardMap` directly in Part A, you may need to
update your server initialization to explicitly *add* shards that are tracked.

### Testing
If you've implemented shard changes (without copying, but including dropping data)
correctly, you should pass a few more of the remaining `server_test.go` tests:

```
go test -v -run='TestServerSingleShard(Drop|MoveNoCopy)'
```

## C2. Implementing GetShardContents

To support real shard migrations, your servers need a way to copy data for a live
shard to another node. You'll implement this as the `GetShardContents` RPC which
effectively gives a snapshot of all the content for a given shard, so a node
newly adding a shard can request this snapshot from an existing peer.

Note that this call is only used between nodes in KV -- the user-facing API never needs to call this directly.

`GetShardContents` should return the subset of keys along with their values and
remaining time on their expiry for a given shard. Semantics:
 1. `GetShardContents` should fail if the server does not host the shard, same as all other RPCs.
 2. `TtlMsRemaining` should be the time remaining between now and the expiry time, *not* the original value of the TTL. This ensures that the key does not live significantly longer when it is copied to another node.
 3. All values for only the requested shard should be sent via `GetShardContents`.

There are no tests directly for `GetShardContents` in the provided tests, only for the public-facing APIs.
You may write your own unit tests as part of extra_tests.go to help guide your implementation.

## C3. Copying shard data
Finally, you will use the `GetShardContents` RPC to copy data in any case that
a shard is added to a node: both initialization and shard map changes after
initialization.

Using `GetShardContents` should follow similar semantics to `Get`
on the client from Part B:
 1. Use the provided `ClientPool` and `GetClient` to get a `KvClient` to a peer node.
 2. Be sure not to try to call `GetShardContents` on the current node.
 3. If possible, spread your `GetShardContents` requests out using some load balancing strategy (random or round robin acceptable).
 4. If a `GetShardContents` request fails, or `GetClient` fails for a given node, try another.
 5. If all peers fail, or if there are no peers available for a given shard, log an error and initialize the shard as empty.
 6. Do not consider a shard "added" (allow requests to succeed) until the shard copy is completed. Do not serve requests while the shard copy is in progress -- you may reject them (with the same error as if the shard was not added) or block them and wait until the shard copy finishes.

Your shard copies must be processed before a shard update is considered done. This means
the RPC must be finished and all keys must be inserted to state before you read another
update from the update channel, and before you allow requests for that shard to succeed.

Similarly, your server must complete all shard copies for shards which existing on initialization
before returning from `MakeKvServer`.

### Testing
At this point you should pass the remaining tests in `server_test.go`:
```
go test -run=TestServer
```

If you've completed all previous sections (A and B), you can move on to Part D to do the remaining
stress tests and benchmarks.


# Part D. Stress testing your implementation

## D1. Integration tests
If you have completed all parts up until this point, you should pass all the provided
tests in `kv/test/`, including all of the tests in `integration_test.go`. If you have not
yet tried running the integration tests, run them now with `go test -run=TestIntegration -v`
and debug any failures. Consider writing some additional unit tests if you have any failures
to help debug.

## D2. Stress test

All tests up to this point (including the integration tests) don't use real networking, they
wire together instances of the server to the client directly. In this final section, you will
run a cluster of many servers on the network and send it continuous traffic with our stress-test client.

### Starting servers
You can start an instance of the server by running the server CLI in `cmd/`:

```
go run cmd/server/server.go --shardmap $shardmap-file --node $nodename
```

We provide a few shardmaps for you, so you can try something like
```
go run cmd/server/server.go --shardmap shardmaps/single-node.json --node n1
```
which is a cluster of only one node.

If you are on a Unix-like machine, we provide a small utility script to start a full
cluster (assuming you have `jq`), essentially just running `go run cmd/server/server.go`
all the nodes listed in a shardmap file.

```
scripts/run-cluster.sh shardmaps/test-3-node.json
```

### Simple testing
If you've got a server (or cluster of servers) started, you can use the provided
client CLI to sanity check before the stress test.

To set a value:
```
go run cmd/client/client.go --shardmap $shardmap-file.json set my-key-name this-is-the-value 60000
```
To retrieve it back:
```
go run cmd/client/client.go --shardmap $shardmap-file.json get my-key-name
```

You must use the same shardmap file in all invocations of the client and server
here to avoid misrouted requests.

### Running the stress tester
Once you've verified the server or servers have started, you can start the stress tester.
The stress tester runs continuous Gets and Sets at a target amount of queries-per-second
for a set duration. By default, this runs at 100 Get QPS and 30 Set QPS for one minute.

Try running it now:
```
go run cmd/stress/tester.go --shardmap $shardmap-file.json
```
Similar to above, use the same shardmap file as the servers.

At the end it should output a summary of how you did: number of requests, QPS, success rate,
and correctness checks.

The stress tester has documentation on how it works and how the correctness checker
works inline in the comments, read those at the top of `cmd/stress/tester.go` if you
are trying to figure out how it works or have issues.

The stress tester also takes several flags to modify its behavior, you can try:
 - `--get-qps` and `--set-qps` to change the workload patterns
 - `--ttl` to change the TTL on keys set by the tester
 - `--num-keys` to change the number of unique keys in the workload
 - `--duration` to run for longer or shorter overall


For example, you can reproduce a "hot key" workload by setting `--num-keys=1` and
raising `--get-qps` (or `--set-qps` to make it a hot key on writes).


### Discussion

Run a few different experiments with different QPS levels, key levels, TTLs,
or shard maps.

Include at least 2 experiments in your `discussions.md` under heading **D2**.
Provide the full command (with flags), the shard map (if it is not a provided one),
what you were testing for, and any interesting results. If you find any bugs using
the stress tester that were not caught in prior testing, you can note that
as an interesting result.

Consider running an experiment where you change the content of the shardmap
file while the experiment is running: the provided shardmap watcher
should update the servers (testing your shard migration logic) and
the client automatically.

### Stress tester expectations
We will run your provided implementation with no more than 250 QPS (total) over
the provided shard map sets. Your server implementation should be able to handle
far more traffic though (likely thousands of QPS) on most hardware.

We will not run the correctness checker and move shards in the same stress test;
we do not define a complete correctness criteria for concurrent writes and shard
movements (though this could be a part of your final project proposal, should
you choose a stronger quorum protocol).

# End of Lab 4

# Time logging

Source: from Prof. Stan Eisenstat's CS223/323 courses. Obtained via Prof. James Glenn [here](https://zoo.cs.yale.edu/classes/cs223/f2020/Projects/log.html).

Each lab submission must contain a complete log `time.log`. Your log file should be a plain text file of the general form (that below is mostly fictitious):

```
ESTIMATE of time to complete assignment: 10 hours

      Time     Time
Date  Started  Spent Work completed
----  -------  ----  --------------
8/01  10:15pm  0:45  read assignment and played several games to help me
                     understand the rules.
8/02   9:00am  2:20  wrote functions for determining whether a roll is
                     three of a kind, four of a kind, and all the other
                     lower categories
8/04   4:45pm  1:15  wrote code to create the graph for the components
8/05   7:05pm  2:00  discovered and corrected two logical errors; code now
                     passes all tests except where choice is Yahtzee
8/07  11:00am  1:35  finished debugging; program passes all public tests
               ----
               7:55  TOTAL time spent

I discussed my solution with: Petey Salovey, Biddy Martin, and Biff Linnane
(and watched four episodes of Futurama).

Debugging the graph construction was difficult because the size of the
graph made it impossible to check by hand.  Using asserts helped
tremendously, as did counting the incoming and outgoing edges for
each vertex.  The other major problem was my use of two different variables
in the same function called _score and score.  The last bug ended up being
using one in place of the other; I now realize the danger of having two
variables with names varying only in punctuation -- since they both sound
the same when reading the code back in my head it was not obvious when
I was using the wrong one.
```

Your log MUST contain:
 - your estimate of the time required (made prior to writing any code),
 - the total time you actually spent on the assignment,
 - the names of all others (but not members of the teaching staff) with whom you discussed the assignment for more than 10 minutes, and
 - a brief discussion (100 words MINIMUM) of the major conceptual and coding difficulties that you encountered in developing and debugging the program (and there will always be some).

The estimated and total times should reflect time outside of class.  Submissions
with missing or incomplete logs will be subject to a penalty of 5-10% of the
total grade, and omitting the names of collaborators is a violation of the
academic honesty policy.

To facilitate analysis, the log file MUST the only file submitted whose name contains the string "log" and the estimate / total MUST be on the only line in that file that contains the string "ESTIMATE" / "TOTAL".
