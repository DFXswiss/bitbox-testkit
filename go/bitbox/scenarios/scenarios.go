// Package scenarios provides pre-built BitBox02 communication scenarios for
// known bug classes and common test situations.
//
// Each scenario returns a configured *fake.Fake ready to plug into
// firmware.NewDevice. See TESTING.md for the underlying contract and how to
// add a new scenario when a new bug class shows up in production.
package scenarios

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/DFXswiss/bitbox-testkit/go/bitbox/fake"
)

// FirmwareError mirrors the wire-level error a real BitBox firmware would
// emit. The fake does not perform real noise framing, so consumers asserting
// on error codes must wrap their own decode in tests.
type FirmwareError struct {
	Code    int
	Message string
}

func (e *FirmwareError) Error() string { return e.Message }

// ErrInvalidInput101 is the firmware response observed when the BitBox
// rejects non-ASCII characters inside an EIP-712 string value. Use
// RegressionUmlautEIP712 to drive a Fake that emits this error.
var ErrInvalidInput101 = &FirmwareError{Code: 101, Message: "firmware: invalid input (101)"}

// DeviceDisconnect installs a fake that closes itself after n successful
// queries. Use to verify graceful-shutdown paths.
func DeviceDisconnect(after int) *fake.Fake {
	f := fake.New()
	var seen int32
	f.Always(func([]byte) ([]byte, error) {
		if int(atomic.AddInt32(&seen, 1)) > after {
			f.Close()
			return nil, fake.ErrClosed
		}
		return []byte{0x00}, nil
	})
	return f
}

// SlowResponse delays every reply by d. Useful for timeout-tuning regression
// tests — e.g. proving the legacy 10s hard-coded transport timeout cannot
// come back.
func SlowResponse(d time.Duration, payload []byte) *fake.Fake {
	cp := append([]byte(nil), payload...)
	return fake.New().Always(func([]byte) ([]byte, error) {
		time.Sleep(d)
		return append([]byte(nil), cp...), nil
	})
}

// PanicMidQuery panics on the n-th query (1-indexed). The plugin-side
// recoverPanic guard on every gomobile export should keep this from
// crashing the host app.
func PanicMidQuery(n int, value any) *fake.Fake {
	f := fake.New()
	var seen int32
	f.Always(func([]byte) ([]byte, error) {
		count := atomic.AddInt32(&seen, 1)
		if int(count) == n {
			panic(value)
		}
		return []byte{0x00}, nil
	})
	return f
}

// RegressionUmlautEIP712 returns ErrInvalidInput101 for any query containing
// a non-ASCII byte. Plug this into firmware.NewDevice to assert the client
// transliterates EIP-712 payloads to ASCII before sending.
func RegressionUmlautEIP712() *fake.Fake {
	return fake.New().Always(func(req []byte) ([]byte, error) {
		for _, b := range req {
			if b > 0x7F {
				return nil, ErrInvalidInput101
			}
		}
		return []byte{0x00}, nil
	})
}

// ChannelHashEarly simulates the BitBox02 race observed during pairing where
// the channel hash is available before the user has confirmed on-device.
// The first n queries return the channel-hash payload; subsequent queries
// require the consumer to call SignalConfirm before they succeed. The
// upstream pair() blocks on user-confirm, so clients need a parallel-poll
// workaround that this scenario exercises.
func ChannelHashEarly(channelHash []byte, hashRepeats int) (*fake.Fake, SignalConfirm) {
	hash := append([]byte(nil), channelHash...)
	var hashCount, confirmed int32
	f := fake.New().Always(func([]byte) ([]byte, error) {
		if int(atomic.LoadInt32(&hashCount)) < hashRepeats {
			atomic.AddInt32(&hashCount, 1)
			return append([]byte(nil), hash...), nil
		}
		if atomic.LoadInt32(&confirmed) == 0 {
			return nil, ErrAwaitingUserConfirm
		}
		return []byte{0x00}, nil
	})
	signal := func() { atomic.StoreInt32(&confirmed, 1) }
	return f, signal
}

// SignalConfirm flips a scenario from "awaiting user" to "user confirmed".
type SignalConfirm func()

// ErrAwaitingUserConfirm is returned while a scenario waits for SignalConfirm.
var ErrAwaitingUserConfirm = errors.New("bitbox-testkit/bitbox/scenarios: awaiting user confirmation")
