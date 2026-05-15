package ble_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/core/transport/ble"
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
