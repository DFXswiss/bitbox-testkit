// Package testutil provides assertion helpers, deadlock detection and
// timeout wrappers shared by every bitbox-testkit consumer.
package testutil

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
)

// TB is the minimal subset of testing.TB the helpers need. *testing.T and
// *testing.B both satisfy it; tests can substitute their own implementation
// to assert that a helper fails.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// MustWithin runs fn and fails t if it does not return within d. The check
// uses a goroutine; deadlocked code is detected at d and the test fails with
// a stack dump of the runtime.
func MustWithin(t TB, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("operation did not finish within %s\n%s", d, allGoroutineStacks())
	}
}

// MustWithinCtx runs fn(ctx) with a deadline of d. If fn returns an error,
// t.Fatalf is called.
func MustWithinCtx(t TB, d time.Duration, fn func(ctx context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- fn(ctx) }()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("operation failed: %v", err)
		}
	case <-time.After(d + 50*time.Millisecond):
		t.Fatalf("operation hung past context deadline (%s)\n%s", d, allGoroutineStacks())
	}
}

// Counter is an atomic int64 with test-friendly accessors.
type Counter struct{ v int64 }

func (c *Counter) Inc()         { atomic.AddInt64(&c.v, 1) }
func (c *Counter) Add(n int64)  { atomic.AddInt64(&c.v, n) }
func (c *Counter) Load() int64  { return atomic.LoadInt64(&c.v) }
func (c *Counter) Reset()       { atomic.StoreInt64(&c.v, 0) }

// AssertEventually polls cond at 1ms intervals until it returns true or d
// elapses. Cheaper than time.Sleep for "did the background work finish".
func AssertEventually(t TB, d time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	if !cond() {
		t.Fatalf("condition not met within %s: %s", d, msg)
	}
}

func allGoroutineStacks() []byte {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	return buf[:n]
}
