package failure_injection

import (
	"math/rand"
	"sync/atomic"
	"time"

	pb "cs426.yale.edu/lab1/failure_injection/proto"
)

// module init, runs when the package is imported
func init() {
	rand.Seed(time.Now().UnixNano())
}

// Thread-safe class to simulate failure injection
//
// Intended to be instantiated by the backend service (incl. mock servers).  The
// server should call in its request handling goroutine MaybeInject() at the
// beginning of each request.  Based on the current injection config, the
// goroutine will sleep for SleepNs, or perhaps hang forever to similate
// response omissions; it then returns whether the caller should error out this
// request outright.
type FailureInjector struct {
	Config atomic.Value
}

func MakeFailureInjector() *FailureInjector {
	var fi FailureInjector
	// default initial state, no failures
	fi.Config.Store(&pb.InjectionConfig{
		SleepNs:              0,
		FailureRate:          0,
		ResponseOmissionRate: 0,
	})
	return &fi
}

func (fi *FailureInjector) ClearInjectionConfig() {
	fi.SetInjectionConfig(0, 0, 0)
}

func (fi *FailureInjector) SetInjectionConfig(sleepNs, failureRate, responseOmissionRate int64) {
	fi.Config.Store(&pb.InjectionConfig{
		SleepNs:              sleepNs,
		FailureRate:          failureRate,
		ResponseOmissionRate: responseOmissionRate,
	})
}

func (fi *FailureInjector) SetInjectionConfigPb(config *pb.InjectionConfig) {
	fi.Config.Store(config)
}

func (fi *FailureInjector) GetInjectionConfig() *pb.InjectionConfig {
	config := fi.Config.Load()
	return config.(*pb.InjectionConfig)
}

// returns true one in n times;
// if n == 0, always returns false
func oneIn(n int64) bool {
	if n == 0 {
		return false
	}
	return rand.Int63n(n) == 0
}

func (fi *FailureInjector) MaybeInject() (shouldError bool) {
	config := fi.GetInjectionConfig()
	time.Sleep(time.Nanosecond * time.Duration(config.SleepNs))

	shouldError = oneIn(config.FailureRate)
	shouldOmitResponse := oneIn(config.ResponseOmissionRate)
	if shouldOmitResponse {
		// block forever
		select {}
	}
	return
}
