package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"cs426.yale.edu/lab4/checker"
	"cs426.yale.edu/lab4/kv"
	"cs426.yale.edu/lab4/logging"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// Stress tester for KV servers. Runs two loops to send a target
// number of queries per second (QPS) of both Gets and Sets.
// Deletes are not tested at this time.
//
// This allows for some basic performance testing -- we'll keep track
// of the error rate of your service, and you can increase the QPS
// to see how high it can go before erroring.
//
// Typical usage for Part D will just be to run:
//    go run cmd/stress/tester.go --shardmap shardmaps/$file.json
//
// This will run with a default of 100 get QPS and 30 set QPS. Your servers
// should be able to handle much more load, but we are not grading on performance
// past 250QPS.
//
// You can use more of these flags to stress your server in different ways,
// or potentially use this in your final project to test improvements.
// For example, you can set `--num-keys=1` to run the test all on one key,
// and stress hot-spot handling. Or run with `--get-qps=1`, `--set-qps=1000`
// to test high write rate.
//
// Included in this stress tester is a primitive correctness checker.
// We record the values that we have Set to the cluster, and make
// sure that Get calls return the right value (with some handling
// for concurrency and partial failures).
// See checker/checker.go for details.
var (
	shardMapFile       = flag.String("shardmap", "", "Path to a JSON file which describes the shard map")
	getQps             = flag.Int("get-qps", 100, "number of Get() calls per second across the cluster")
	setQps             = flag.Int("set-qps", 30, "number of Set() calls per second across the cluster")
	qpsBurst           = flag.Int("qps-burst", 20, "Maximum burst of QPS")
	duration           = flag.Duration("duration", 60*time.Second, "Duration of the stress test")
	timeout            = flag.Duration("timeout", 1*time.Second, "Timeout for RPCs sent to the cluster")
	maxPendingRequests = flag.Int("max-pending", 100, "Maximum number of in-flight requests before the stress tester slows down")
	numKeys            = flag.Int("num-keys", 1000, "Number of unique keys to stress")
	ttl                = flag.Duration("ttl", 2*time.Second, "TTL of values to set on keys")
)

/*
 * Shared state for a stress test run. Everything must be thread-safe as we run from
 * multiple goroutines.
 */
type stressTester struct {
	kv                *kv.Kv
	sem               *semaphore.Weighted
	ctx               context.Context
	successLogLimiter *rate.Limiter
	errorLogLimiter   *rate.Limiter
	wg                sync.WaitGroup

	keys            []string
	gets            uint64
	getErrs         uint64
	sets            uint64
	setErrs         uint64
	inconsistencies uint64

	cc *checker.ConsistencyChecker
}

func randomString(rng *rand.Rand, length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz"

	out := strings.Builder{}
	for i := 0; i < length; i++ {
		out.WriteByte(chars[rng.Int()%len(chars)])
	}
	return out.String()
}

func randomKeys(n, length int) []string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	out := make([]string, 0)
	for i := 0; i < n; i++ {
		out = append(out, randomString(rng, length))
	}
	return out
}

func makeStressTester(kv *kv.Kv) *stressTester {
	return &stressTester{
		kv:                kv,
		sem:               semaphore.NewWeighted(int64(*maxPendingRequests)),
		ctx:               context.Background(),
		successLogLimiter: rate.NewLimiter(1, 1),
		errorLogLimiter:   rate.NewLimiter(2, 10),
		keys:              randomKeys(*numKeys, 10),
		cc:                checker.MakeConsistencyChecker(),
	}
}

/*
 * Run a stress-test loop at a given QPS, kicking off requests using `sendReqFn`.
 *
 * Shared code between stressGets and stressSets to avoid boilerplate.
 */
func (st *stressTester) stressLoop(
	name string,
	qps int,
	sendReqFn func(ctx context.Context, key, value string),
) {
	limiter := rate.NewLimiter(rate.Limit(qps), *qpsBurst)
	logrus.Infof("Running %s stress test at %d QPS", name, qps)

	start := time.Now()
	rng := rand.New(rand.NewSource(time.Now().UnixMicro()))
	for time.Since(start) < *duration {
		limiter.Wait(st.ctx)
		st.sem.Acquire(st.ctx, 1)

		st.wg.Add(1)
		key := st.keys[rng.Int()%len(st.keys)]
		value := randomString(rand.New(rand.NewSource(time.Now().UnixMicro())), 32)
		go func() {
			ctx, cancel := context.WithTimeout(st.ctx, *timeout)
			sendReqFn(ctx, key, value)
			cancel()
			st.sem.Release(1)
			st.wg.Done()
		}()
	}
	st.wg.Done()
}

func (st *stressTester) stressGets() {
	st.stressLoop("Get", *getQps, func(ctx context.Context, key, _ string) {
		startTime := time.Now()

		initialVersion, writesPending := st.cc.BeginRead(key)
		val, wasFound, err := st.kv.Get(ctx, key)
		latency := time.Since(startTime)

		atomic.AddUint64(&st.gets, 1)
		if err != nil {
			if st.errorLogLimiter.Allow() {
				logrus.WithField("key", key).Errorf("get failed: %q", err)
			}
			atomic.AddUint64(&st.getErrs, 1)

		} else {
			err := st.cc.CheckReadCorrect(key, val, wasFound, startTime, initialVersion, writesPending)
			if err != nil && st.errorLogLimiter.Allow() {
				atomic.AddUint64(&st.inconsistencies, 1)
				logrus.WithField("key", key).Errorf("get returned wrong answer: %s", err)
			} else if st.successLogLimiter.Allow() {
				logrus.WithField("key", key).WithField("latency_us", latency.Microseconds()).Info("[sampled] get OK")
			}
		}
	})
}

func (st *stressTester) stressSets() {
	st.stressLoop("Set", *setQps, func(ctx context.Context, key, value string) {
		startTime := time.Now()
		initialVersion := st.cc.BeginWrite(key)

		err := st.kv.Set(ctx, key, value, *ttl)

		latency := time.Since(startTime)
		endTime := time.Now()
		atomic.AddUint64(&st.sets, 1)
		if err != nil {
			if st.errorLogLimiter.Allow() {
				logrus.WithField("key", key).Errorf("set failed: %q", err)
			}
			atomic.AddUint64(&st.setErrs, 1)
		} else {
			if st.successLogLimiter.Allow() {
				logrus.WithField("key", key).WithField("latency_us", latency.Microseconds()).Info("[sampled] set OK")
			}
		}
		st.cc.CompleteWrite(key, value, err, initialVersion, startTime.Add(*ttl), endTime.Add(*ttl))
	})
}

func (st *stressTester) wait() {
	st.wg.Wait()
}

func main() {
	flag.Parse()
	logging.InitLogging()

	fileSm, err := kv.WatchShardMapFile(*shardMapFile)
	if err != nil {
		logrus.Fatal(err)
	}

	clientPool := kv.MakeClientPool(&fileSm.ShardMap)
	client := kv.MakeKv(&fileSm.ShardMap, &clientPool)

	tester := makeStressTester(client)
	start := time.Now()
	tester.wg.Add(2)
	go tester.stressGets()
	go tester.stressSets()
	tester.wait()
	testDuration := time.Since(start)

	gets := atomic.LoadUint64(&tester.gets)
	sets := atomic.LoadUint64(&tester.sets)
	getErrs := atomic.LoadUint64(&tester.getErrs)
	setErrs := atomic.LoadUint64(&tester.setErrs)
	checks := atomic.LoadUint64(&tester.cc.ChecksRun)
	inconsistencies := atomic.LoadUint64(&tester.inconsistencies)
	fmt.Println("Stress test completed!")
	fmt.Printf("Get requests: %d/%d succeeded = %f%% success rate\n", gets-getErrs, gets, 100*float64(gets-getErrs)/float64(gets))
	fmt.Printf("Set requests: %d/%d succeeded = %f%% success rate\n", sets-setErrs, sets, 100*float64(sets-setErrs)/float64(sets))
	totalRequests := gets + sets
	fmt.Printf("Correct responses: %d/%d = %f%%\n", checks-inconsistencies, checks, 100*float64(checks-inconsistencies)/float64(checks))
	totalQps := float64(totalRequests) / testDuration.Seconds()
	fmt.Printf("Total requests: %d = %f QPS\n", totalRequests, totalQps)
}
