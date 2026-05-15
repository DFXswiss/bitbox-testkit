package ble_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/core/transport/ble"
)

func TestInjectThenReadReturnsBytes(t *testing.T) {
	p := ble.New()
	if err := p.Inject([]byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	n, err := p.Read(buf)
	if err != nil || n != 4 {
		t.Fatalf("read: n=%d err=%v", n, err)
	}
	if !bytes.Equal(buf, []byte{1, 2, 3, 4}) {
		t.Fatalf("got %v", buf)
	}
}

func TestReadBlocksUntilInject(t *testing.T) {
	p := ble.New()
	buf := make([]byte, 8)
	var (
		got    []byte
		readN  int
		readEr error
	)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		readN, readEr = p.Read(buf)
		got = append(got, buf[:readN]...)
	}()
	time.Sleep(5 * time.Millisecond)
	_ = p.Inject([]byte("late"))
	wg.Wait()
	if readEr != nil {
		t.Fatal(readEr)
	}
	if string(got) != "late" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteCapturedInOrder(t *testing.T) {
	p := ble.New()
	if _, err := p.Write([]byte{0xAA}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write([]byte{0xBB, 0xCC}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p.Sent(), []byte{0xAA, 0xBB, 0xCC}) {
		t.Fatalf("got %x", p.Sent())
	}
}

func TestCloseUnblocksRead(t *testing.T) {
	p := ble.New()
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4)
		_, _ = p.Read(buf)
	}()
	time.Sleep(5 * time.Millisecond)
	_ = p.Close()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Read did not unblock after Close")
	}
}

func TestReadAfterCloseEOF(t *testing.T) {
	p := ble.New()
	_ = p.Close()
	_, err := p.Read(make([]byte, 4))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("got %v, want EOF", err)
	}
}

func TestInjectAfterCloseRejected(t *testing.T) {
	p := ble.New()
	_ = p.Close()
	if err := p.Inject([]byte{1}); !errors.Is(err, ble.ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
}

func TestWriteAfterCloseRejected(t *testing.T) {
	p := ble.New()
	_ = p.Close()
	if _, err := p.Write([]byte{1}); !errors.Is(err, ble.ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
}

func TestSentReturnsDefensiveCopy(t *testing.T) {
	p := ble.New()
	_, _ = p.Write([]byte{0x42})
	sent := p.Sent()
	sent[0] = 0xFF
	if p.Sent()[0] != 0x42 {
		t.Fatal("Sent returned shared backing array")
	}
}

func TestWaitForWriteDeadline(t *testing.T) {
	p := ble.New()
	go func() {
		time.Sleep(5 * time.Millisecond)
		_, _ = p.Write([]byte{1, 2})
	}()
	if !p.WaitForWrite(2, 200*time.Millisecond) {
		t.Fatal("did not see 2 written bytes in time")
	}
	if p.WaitForWrite(100, 20*time.Millisecond) {
		t.Fatal("incorrectly reported 100 bytes available")
	}
}

// TestStressInjectReadInterleaving runs Inject and Read in tight
// goroutines and verifies no bytes are dropped or duplicated. Locks in
// the correctness of the channel-cap-1 signaling: even with thousands
// of interleaved operations, every injected byte makes it to Read.
func TestStressInjectReadInterleaving(t *testing.T) {
	const N = 5000
	p := ble.New()
	defer p.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			_ = p.Inject([]byte{byte(i & 0xff)})
		}
	}()

	got := make([]byte, 0, N)
	buf := make([]byte, 64)
	for len(got) < N {
		n, err := p.Read(buf)
		if err != nil {
			t.Fatalf("read after %d bytes: %v", len(got), err)
		}
		got = append(got, buf[:n]...)
	}
	wg.Wait()

	if len(got) != N {
		t.Fatalf("got %d bytes, want %d", len(got), N)
	}
	for i, b := range got {
		if b != byte(i&0xff) {
			t.Fatalf("byte %d: got %02x, want %02x", i, b, byte(i&0xff))
		}
	}
}

// TestStressCloseRaceUnblocksRead verifies Close consistently unblocks
// pending Reads even under tight interleaving.
func TestStressCloseRaceUnblocksRead(t *testing.T) {
	for trial := 0; trial < 50; trial++ {
		p := ble.New()
		done := make(chan struct{})
		go func() {
			buf := make([]byte, 4)
			_, _ = p.Read(buf)
			close(done)
		}()
		time.Sleep(time.Microsecond * time.Duration(trial))
		_ = p.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("trial %d: Read did not unblock after Close", trial)
		}
	}
}
