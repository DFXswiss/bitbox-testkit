package testutil_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/core/testutil"
)

func TestMustWithinPasses(t *testing.T) {
	testutil.MustWithin(t, 50*time.Millisecond, func() {
		time.Sleep(time.Millisecond)
	})
}

func TestMustWithinDetectsHang(t *testing.T) {
	fake := &fakeTB{}
	testutil.MustWithin(fake, 5*time.Millisecond, func() {
		time.Sleep(50 * time.Millisecond)
	})
	if !fake.failed {
		t.Fatal("expected fakeTB to have been failed by hang")
	}
}

func TestMustWithinCtxPropagatesError(t *testing.T) {
	fake := &fakeTB{}
	want := errors.New("inner")
	testutil.MustWithinCtx(fake, 50*time.Millisecond, func(context.Context) error {
		return want
	})
	if !fake.failed {
		t.Fatal("expected failure on inner error")
	}
}

func TestCounterAtomic(t *testing.T) {
	var c testutil.Counter
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Inc() }()
	}
	wg.Wait()
	if c.Load() != 100 {
		t.Fatalf("got %d, want 100", c.Load())
	}
}

func TestAssertEventually(t *testing.T) {
	var flag struct {
		sync.Mutex
		set bool
	}
	go func() {
		time.Sleep(5 * time.Millisecond)
		flag.Lock()
		flag.set = true
		flag.Unlock()
	}()
	testutil.AssertEventually(t, 100*time.Millisecond, func() bool {
		flag.Lock()
		defer flag.Unlock()
		return flag.set
	}, "flag never flipped")
}

// fakeTB captures Fatalf calls without aborting the surrounding goroutine,
// so we can assert helpers correctly mark failures.
type fakeTB struct {
	failed bool
}

func (f *fakeTB) Helper()                  {}
func (f *fakeTB) Fatalf(string, ...any)    { f.failed = true }
