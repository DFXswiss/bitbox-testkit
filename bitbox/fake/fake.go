// Package fake provides a scriptable in-memory implementation of
// firmware.Communication for testing BitBox02 client code without a device
// or simulator.
package fake

import (
	"errors"
	"sync"

	"github.com/BitBoxSwiss/bitbox02-api-go/api/firmware"
)

// ErrClosed is returned from Query after Close has been called.
var ErrClosed = errors.New("bitbox-testkit/bitbox/fake: communication closed")

// ErrUnexpectedQuery is returned when no handler matches an incoming query.
// Tests should treat this as a failure unless an UnhandledHandler is installed.
var ErrUnexpectedQuery = errors.New("bitbox-testkit/bitbox/fake: unexpected query, no handler matched")

// Handler reacts to a single Query call. Match returns true to claim the
// query; Respond produces the answer (or error). A Handler may consume itself
// after one match by returning true from Exhausted.
type Handler interface {
	Match(req []byte) bool
	Respond(req []byte) ([]byte, error)
	Exhausted() bool
}

// Fake implements firmware.Communication using a chain of Handlers.
//
// Handlers are tried in registration order; the first matching, non-exhausted
// handler responds. Calls are recorded for later assertions.
//
// queryMu serializes the entire Query call so handler state mutations in
// Respond do not need their own locking. stateMu protects only the metadata
// fields (handlers, calls, closed, onClose), which lets a Respond
// implementation safely call back into Close or Add without deadlocking.
type Fake struct {
	queryMu sync.Mutex

	stateMu  sync.Mutex
	handlers []Handler
	calls    [][]byte
	closed   bool
	onClose  func()
}

// New returns an empty Fake. Add handlers with Expect, ExpectMatch, etc.
func New() *Fake {
	return &Fake{}
}

// compile-time guarantee that Fake satisfies firmware.Communication
var _ firmware.Communication = (*Fake)(nil)

// Query dispatches req to the first matching live handler.
func (f *Fake) Query(req []byte) ([]byte, error) {
	f.queryMu.Lock()
	defer f.queryMu.Unlock()

	f.stateMu.Lock()
	if f.closed {
		f.stateMu.Unlock()
		return nil, ErrClosed
	}
	cp := make([]byte, len(req))
	copy(cp, req)
	f.calls = append(f.calls, cp)
	handlers := f.handlers
	f.stateMu.Unlock()

	for _, h := range handlers {
		if h.Exhausted() {
			continue
		}
		if h.Match(req) {
			return h.Respond(req)
		}
	}
	return nil, ErrUnexpectedQuery
}

// Close marks the Fake closed. Subsequent Query calls return ErrClosed.
// Safe to call from inside a Handler's Respond.
func (f *Fake) Close() {
	f.stateMu.Lock()
	wasClosed := f.closed
	f.closed = true
	cb := f.onClose
	f.stateMu.Unlock()
	if !wasClosed && cb != nil {
		cb()
	}
}

// OnClose registers a callback invoked exactly once when Close is first
// called. Useful for assertions or coordinating goroutines.
func (f *Fake) OnClose(fn func()) *Fake {
	f.stateMu.Lock()
	f.onClose = fn
	f.stateMu.Unlock()
	return f
}

// Add appends a Handler to the dispatch chain.
func (f *Fake) Add(h Handler) *Fake {
	f.stateMu.Lock()
	f.handlers = append(f.handlers, h)
	f.stateMu.Unlock()
	return f
}

// Calls returns a copy of every Query payload received so far.
func (f *Fake) Calls() [][]byte {
	f.stateMu.Lock()
	defer f.stateMu.Unlock()
	out := make([][]byte, len(f.calls))
	for i, c := range f.calls {
		cp := make([]byte, len(c))
		copy(cp, c)
		out[i] = cp
	}
	return out
}

// Closed reports whether Close has been called.
func (f *Fake) Closed() bool {
	f.stateMu.Lock()
	defer f.stateMu.Unlock()
	return f.closed
}
