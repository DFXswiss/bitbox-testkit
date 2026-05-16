package scenarios_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DFXswiss/bitbox-testkit/go/bitbox/fake"
	"github.com/DFXswiss/bitbox-testkit/go/bitbox/scenarios"
)

func TestDeviceDisconnectAfterN(t *testing.T) {
	f := scenarios.DeviceDisconnect(2)
	if _, err := f.Query(nil); err != nil {
		t.Fatalf("query 1: %v", err)
	}
	if _, err := f.Query(nil); err != nil {
		t.Fatalf("query 2: %v", err)
	}
	if _, err := f.Query(nil); !errors.Is(err, fake.ErrClosed) {
		t.Fatalf("query 3: want ErrClosed, got %v", err)
	}
}

func TestSlowResponseHonoredDelay(t *testing.T) {
	f := scenarios.SlowResponse(10*time.Millisecond, []byte("ok"))
	start := time.Now()
	got, err := f.Query(nil)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("returned faster than configured delay")
	}
	if string(got) != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestPanicMidQueryPanicsOnTarget(t *testing.T) {
	f := scenarios.PanicMidQuery(2, "boom")
	// first query: ok
	if _, err := f.Query(nil); err != nil {
		t.Fatalf("query 1: %v", err)
	}
	defer func() {
		if r := recover(); r != "boom" {
			t.Fatalf("recovered %v, want boom", r)
		}
	}()
	_, _ = f.Query(nil)
}

func TestRegressionUmlautEIP712(t *testing.T) {
	f := scenarios.RegressionUmlautEIP712()
	// ASCII passes
	if _, err := f.Query([]byte("hello world")); err != nil {
		t.Fatalf("ascii query failed: %v", err)
	}
	// Non-ASCII fails with the documented firmware error
	_, err := f.Query([]byte("hëllo"))
	if !errors.Is(err, scenarios.ErrInvalidInput101) {
		t.Fatalf("got %v, want ErrInvalidInput101", err)
	}
}

func TestChannelHashEarlyFlow(t *testing.T) {
	hash := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	f, confirm := scenarios.ChannelHashEarly(hash, 2)

	// First two queries return the hash even before user confirm
	for i := 0; i < 2; i++ {
		got, err := f.Query(nil)
		if err != nil {
			t.Fatalf("hash query %d: %v", i, err)
		}
		if string(got) != string(hash) {
			t.Fatalf("hash query %d: got %x", i, got)
		}
	}
	// Subsequent query before confirm signals the wait
	if _, err := f.Query(nil); !errors.Is(err, scenarios.ErrAwaitingUserConfirm) {
		t.Fatalf("want ErrAwaitingUserConfirm, got %v", err)
	}
	// Confirm unblocks
	confirm()
	if _, err := f.Query(nil); err != nil {
		t.Fatalf("after confirm: %v", err)
	}
}
