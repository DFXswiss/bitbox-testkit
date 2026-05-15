package fake_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/BitBoxSwiss/bitbox02-api-go/api/firmware"
	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/fake"
)

// ensures Fake satisfies the upstream interface at use site too (not just package init)
func TestImplementsCommunication(t *testing.T) {
	var _ firmware.Communication = fake.New()
}

func TestExpectSequence(t *testing.T) {
	f := fake.New().
		Expect([]byte("one")).
		Expect([]byte("two"))

	got1, err := f.Query([]byte("a"))
	if err != nil || string(got1) != "one" {
		t.Fatalf("first query: got %q, %v", got1, err)
	}
	got2, err := f.Query([]byte("b"))
	if err != nil || string(got2) != "two" {
		t.Fatalf("second query: got %q, %v", got2, err)
	}
	if _, err := f.Query([]byte("c")); !errors.Is(err, fake.ErrUnexpectedQuery) {
		t.Fatalf("third query: want ErrUnexpectedQuery, got %v", err)
	}
}

func TestExpectError(t *testing.T) {
	want := errors.New("boom")
	f := fake.New().ExpectError(want)
	if _, err := f.Query(nil); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestCloseSticky(t *testing.T) {
	f := fake.New().Always(func([]byte) ([]byte, error) { return []byte{1}, nil })
	f.Close()
	if !f.Closed() {
		t.Fatal("not marked closed")
	}
	if _, err := f.Query(nil); !errors.Is(err, fake.ErrClosed) {
		t.Fatalf("after close: want ErrClosed, got %v", err)
	}
}

func TestOnCloseFiresOnce(t *testing.T) {
	var count int
	f := fake.New().OnClose(func() { count++ })
	f.Close()
	f.Close()
	if count != 1 {
		t.Fatalf("onClose fired %d times, want 1", count)
	}
}

func TestRecordsCalls(t *testing.T) {
	f := fake.New().Always(func([]byte) ([]byte, error) { return []byte{0xAA}, nil })
	_, _ = f.Query([]byte{1, 2})
	_, _ = f.Query([]byte{3, 4})
	calls := f.Calls()
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}
	// Mutating recorded copy must not affect future Calls() output.
	calls[0][0] = 0xFF
	if f.Calls()[0][0] == 0xFF {
		t.Fatal("Calls() returned shared backing array; should be defensive copy")
	}
}

func TestMatchPrefix(t *testing.T) {
	f := fake.New().
		ExpectMatch(fake.MatchPrefix([]byte{0x01}), func([]byte) ([]byte, error) { return []byte("first"), nil }).
		ExpectMatch(fake.MatchPrefix([]byte{0x02}), func([]byte) ([]byte, error) { return []byte("second"), nil })

	got, err := f.Query([]byte{0x02, 0x99})
	if err != nil || string(got) != "second" {
		t.Fatalf("prefix match failed: %q, %v", got, err)
	}
}

func TestConcurrentQuerySerialized(t *testing.T) {
	// Verify the internal lock prevents racy handler state.
	f := fake.New().Always(func([]byte) ([]byte, error) { return []byte{0}, nil })
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = f.Query([]byte{1}) }()
	}
	wg.Wait()
	if got := len(f.Calls()); got != 50 {
		t.Fatalf("got %d recorded calls, want 50", got)
	}
}

func TestPanicHandlerPanics(t *testing.T) {
	f := fake.New().PanicHandler(fake.MatchAny(), "kaboom")
	defer func() {
		r := recover()
		if r != "kaboom" {
			t.Fatalf("recovered %v, want kaboom", r)
		}
	}()
	_, _ = f.Query(nil)
}
