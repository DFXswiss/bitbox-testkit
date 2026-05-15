package fake

import (
	"bytes"
	"errors"
)

// MatcherFunc reports whether a request belongs to this handler.
type MatcherFunc func(req []byte) bool

// ResponderFunc produces a response (or error) for a matched request.
type ResponderFunc func(req []byte) ([]byte, error)

// MatchAny matches every request.
func MatchAny() MatcherFunc { return func([]byte) bool { return true } }

// MatchPrefix matches requests starting with prefix.
func MatchPrefix(prefix []byte) MatcherFunc {
	cp := append([]byte(nil), prefix...)
	return func(req []byte) bool { return bytes.HasPrefix(req, cp) }
}

// MatchEqual matches requests with identical bytes.
func MatchEqual(want []byte) MatcherFunc {
	cp := append([]byte(nil), want...)
	return func(req []byte) bool { return bytes.Equal(req, cp) }
}

// scriptedHandler matches via MatcherFunc and responds via ResponderFunc.
// If Limit > 0, the handler stops matching after Limit responses.
type scriptedHandler struct {
	matcher  MatcherFunc
	responder ResponderFunc
	limit    int
	calls    int
}

func (h *scriptedHandler) Match(req []byte) bool       { return h.matcher(req) }
func (h *scriptedHandler) Respond(req []byte) ([]byte, error) {
	h.calls++
	return h.responder(req)
}
func (h *scriptedHandler) Exhausted() bool { return h.limit > 0 && h.calls >= h.limit }

// Expect adds a handler that matches every request and replies with resp once.
// Use this to script a fixed sequence: Expect, Expect, Expect.
func (f *Fake) Expect(resp []byte) *Fake {
	cp := append([]byte(nil), resp...)
	return f.Add(&scriptedHandler{
		matcher:   MatchAny(),
		responder: func([]byte) ([]byte, error) { return append([]byte(nil), cp...), nil },
		limit:     1,
	})
}

// ExpectError adds a handler that matches once and returns err.
func (f *Fake) ExpectError(err error) *Fake {
	return f.Add(&scriptedHandler{
		matcher:   MatchAny(),
		responder: func([]byte) ([]byte, error) { return nil, err },
		limit:     1,
	})
}

// ExpectMatch adds a single-use handler with explicit matcher + responder.
func (f *Fake) ExpectMatch(m MatcherFunc, r ResponderFunc) *Fake {
	return f.Add(&scriptedHandler{matcher: m, responder: r, limit: 1})
}

// Always adds a handler that responds to every request indefinitely. Useful
// for unconditional sinks ("any further query returns generic OK").
func (f *Fake) Always(r ResponderFunc) *Fake {
	return f.Add(&scriptedHandler{matcher: MatchAny(), responder: r})
}

// AlwaysError installs a fallthrough handler returning err for everything
// that no prior handler matched.
func (f *Fake) AlwaysError(err error) *Fake {
	return f.Always(func([]byte) ([]byte, error) { return nil, err })
}

// PanicHandler installs a handler that calls panic(v) when matched. Lets
// tests verify recoverPanic shields gomobile exports from crashes.
func (f *Fake) PanicHandler(m MatcherFunc, v any) *Fake {
	return f.Add(&scriptedHandler{
		matcher:   m,
		responder: func([]byte) ([]byte, error) { panic(v) },
		limit:     1,
	})
}

// ErrInjected is the canonical error used by InjectError-style scenarios.
var ErrInjected = errors.New("bitbox-testkit/bitbox/fake: injected error")
