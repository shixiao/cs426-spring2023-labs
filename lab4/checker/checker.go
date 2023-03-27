package checker

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Correctness checker to run alongside the stress tester.
//
// Every Set(), we record the value stored. We record the value even if the
// set returned an error because it may have partially succeeded.
//
// The consistency checker maintains state for each key of the potential
// values -- if a write was successful and not-concurrent, it is stored
// as the only correct value. If it errored, we add it to a list of
// potential values that may be returned.
//
// All successful Get calls are checked against the set of correct
// responses from previous writes.
//
// Care is taken to avoid false positives -- notably concurrent reads
// and writes to the same key can introduce false positives: if the
// Set has hit the server first but not been recorded, the Get
// response may seem incorrect (and other similar anomalies).
//
// Concurrency detection is done through two states per key:
// the "version" and the reference count (rc). Before we do
// a read, we check the latest version (which starts at 0,
// and increments by 1 for every Set). If the version is the
// same before and after we read, we know that no Set
// has updated the state since the read started.
// The reference count tracks the number of concurrent writers to
// a key. If there are multiple writes happening concurrently,
// we do not know which write actually finished last, so we
// have to allow any of the possible results to be "correct".

// If you want to run the stress-tester without checking for correctness, you can disable
// it via these flags.
var checkCorrectness = flag.Bool("check", true, "If set to false, don't check correctness of values")
var checkTtl = flag.Bool("check-ttl", true, "If set to false, don't for expired TTLs")
var ttlCheckBuffer = flag.Duration("ttl-check-buffer", 10*time.Millisecond, "Buffer time to allow for implementation differences in TTL")

/*
 * Value recorded from a potential Set() call. We record the value
 * and a lower bound and upper bound of when it may expire so we can check if Get()
 * calls return the correct value.
 *
 * Note that there may be many values stored -- in the case of concurrent writes
 * or partially failed writes, we don't know which value succeeded, so we allow
 * any of them to be returned.
 */
type stateValue struct {
	value               string
	writtenAt           time.Time
	maybeExpiredBy      time.Time
	definitelyExpiredBy time.Time
	err                 error
	overwrittenAt       time.Time
}

func (val stateValue) String() string {
	return fmt.Sprintf(
		"{val=%s, ttlRemaining=%dms, writtenAtAgo=%dms, wasError=%t}, ",
		val.value,
		time.Until(val.maybeExpiredBy).Milliseconds(),
		time.Since(val.writtenAt).Milliseconds(),
		val.err != nil,
	)
}

// Helper to print previously written stateValues if Get() returns the wrong answer
func printValues(vals []stateValue) string {
	str := strings.Builder{}
	for _, val := range vals {
		str.WriteString(val.String())
	}
	return str.String()
}

type ConsistencyChecker struct {
	// Updated atomically
	ChecksRun uint64

	mutex sync.RWMutex
	// Current valid values for a given key
	values map[string]map[string]stateValue
	// Old values for a given key that have definitely been overwritten,
	// used for error messages
	overwrittenValues map[string]map[string]stateValue
	// Current version of the key, incremented by 1 for every write
	version map[string]int
	// Reference count of pending writes for concurrency control
	rc map[string]int
}

func MakeConsistencyChecker() *ConsistencyChecker {
	return &ConsistencyChecker{
		values:            make(map[string]map[string]stateValue),
		overwrittenValues: make(map[string]map[string]stateValue),
		version:           make(map[string]int),
		rc:                make(map[string]int),
	}
}

func (cc *ConsistencyChecker) BeginWrite(key string) int {
	if !*checkCorrectness {
		return 0
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.rc[key] += 1
	return cc.version[key]
}

func (cc *ConsistencyChecker) CompleteWrite(
	key, value string,
	err error,
	initialVersion int,
	maybeExpiredBy time.Time,
	definitelyExpiredBy time.Time,
) {

	if !*checkCorrectness {
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	newVal := stateValue{
		value:               value,
		maybeExpiredBy:      maybeExpiredBy,
		definitelyExpiredBy: definitelyExpiredBy.Add(*ttlCheckBuffer),
		writtenAt:           time.Now(),
		err:                 err,
	}

	currentVersion := cc.version[key]
	if _, ok := cc.values[key][value]; ok {
		panic("duplicate values are not supported!")
	}
	if _, ok := cc.overwrittenValues[key][value]; ok {
		panic("duplicate values are not supported!")
	}

	if err == nil && initialVersion == currentVersion && cc.rc[key] == 1 {
		for _, val := range cc.values[key] {
			if cc.overwrittenValues[key] == nil {
				cc.overwrittenValues[key] = make(map[string]stateValue)
			}
			cc.overwrittenValues[key][val.value] = val
		}
		cc.values[key] = map[string]stateValue{value: newVal}
	} else {
		if cc.values[key] == nil {
			cc.values[key] = make(map[string]stateValue)
		}
		cc.values[key][value] = newVal
	}

	cc.rc[key] -= 1
	cc.version[key] += 1
}

func (cc *ConsistencyChecker) BeginRead(key string) (int, bool) {
	if !*checkCorrectness {
		return 0, false
	}
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	return cc.version[key], cc.rc[key] != 0
}

func (cc *ConsistencyChecker) CheckReadCorrect(key, value string, wasFound bool, startTime time.Time, initialVersion int, writesPending bool) error {
	if !*checkCorrectness {
		return nil
	}

	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	currentVersion := cc.version[key]
	if currentVersion != initialVersion || writesPending || cc.rc[key] != 0 {
		return nil
	}

	atomic.AddUint64(&cc.ChecksRun, 1)

	now := time.Now()
	values := cc.values[key]
	if !wasFound {
		someValueExpired := len(values) == 0
		valsNotExpired := make([]stateValue, 0)
		for _, value := range values {
			if value.maybeExpiredBy.Before(now) {
				someValueExpired = true
			} else {
				valsNotExpired = append(valsNotExpired, value)
			}
		}
		if !someValueExpired {
			return fmt.Errorf("no value found, but there unexpired potential values: %s", printValues(valsNotExpired))
		}
	} else {
		val, found := cc.values[key][value]
		if !found {
			overwrittenVal, found := cc.overwrittenValues[key][value]
			if found {
				return fmt.Errorf("incorrect value %s; was overrwritten %dms ago", value, time.Since(overwrittenVal.overwrittenAt).Milliseconds())
			} else {
				return fmt.Errorf("incorrect value %s; never written to key", val)
			}
		}
		if *checkTtl && val.err == nil && val.definitelyExpiredBy.Before(startTime) {
			timeDiff := startTime.Sub(val.maybeExpiredBy)
			return fmt.Errorf("value %s was written, but expired at least %dms ago", value, timeDiff.Milliseconds())
		}
	}
	return nil
}
