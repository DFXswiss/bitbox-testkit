// Package ble provides a fake BLE peripheral for testing wallet-plugin BLE
// adapters at the io.ReadWriteCloser layer.
//
// A real plugin BLE adapter reads notify-characteristic bytes and writes
// to a write-characteristic. The Peripheral here exposes the same shape:
// Read returns bytes injected via Inject; Write captures bytes the plugin
// sends so tests can assert frame sequences.
//
// This is the layer where BLE packet-dedup regressions show up: feeding a
// duplicate init frame at the right moment must NOT cause subsequent
// continuation pages to be dropped.
package ble

import (
	"errors"
	"io"
	"sync"
	"time"
)

// ErrClosed is returned from Read/Write after Close.
var ErrClosed = errors.New("bitbox-testkit/core/transport/ble: peripheral closed")

// Peripheral is a chan-backed io.ReadWriteCloser that simulates a BLE
// peripheral connection.
//
// Inject(b) makes b available to the next Read calls in order.
// Sent() returns a snapshot of every byte the consumer has written so far.
type Peripheral struct {
	mu     sync.Mutex
	closed bool

	// readBuf accumulates bytes Inject'd by the test; Read drains it.
	readBuf []byte
	readCh  chan struct{} // signals readBuf may now have content

	written []byte
}

// compile-time check that Peripheral is an io.ReadWriteCloser
var _ io.ReadWriteCloser = (*Peripheral)(nil)

// New returns a fresh Peripheral with empty buffers.
func New() *Peripheral {
	return &Peripheral{
		readCh: make(chan struct{}, 1),
	}
}

// Inject queues b to be returned by upcoming Read calls. Multiple Injects
// concatenate. Returns ErrClosed if Close has been called.
func (p *Peripheral) Inject(b []byte) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.readBuf = append(p.readBuf, b...)
	p.mu.Unlock()
	// non-blocking signal
	select {
	case p.readCh <- struct{}{}:
	default:
	}
	return nil
}

// Read pulls bytes from the injected queue. Blocks until data is available
// or Close is called.
func (p *Peripheral) Read(into []byte) (int, error) {
	for {
		p.mu.Lock()
		if p.closed && len(p.readBuf) == 0 {
			p.mu.Unlock()
			return 0, io.EOF
		}
		if len(p.readBuf) > 0 {
			n := copy(into, p.readBuf)
			p.readBuf = p.readBuf[n:]
			p.mu.Unlock()
			return n, nil
		}
		p.mu.Unlock()
		// wait for an Inject signal
		<-p.readCh
	}
}

// Write captures bytes for later inspection via Sent. Returns ErrClosed
// after Close.
func (p *Peripheral) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, ErrClosed
	}
	p.written = append(p.written, b...)
	return len(b), nil
}

// Sent returns a defensive copy of every byte written to this peripheral.
func (p *Peripheral) Sent() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]byte, len(p.written))
	copy(out, p.written)
	return out
}

// Close ends reads and writes. Idempotent.
func (p *Peripheral) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	// wake any pending Read so it can observe the closed flag
	select {
	case p.readCh <- struct{}{}:
	default:
	}
	return nil
}

// WaitForWrite blocks until n bytes have been written or d elapses.
// Used by tests to synchronize with the consumer's I/O without polling.
func (p *Peripheral) WaitForWrite(n int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		got := len(p.written)
		p.mu.Unlock()
		if got >= n {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}
