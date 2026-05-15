package quirks

import (
	"regexp"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/fake"
	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/scenarios"
)

// matchInvalidInput is the canonical regex for "this test failed with a
// firmware ErrInvalidInput". Used as Match for any quirk whose runtime
// signature is the wire-level error 101.
var matchInvalidInput = regexp.MustCompile(`(?i)\binvalid[- ]?input\b|\b101\b`)

// errInvalidInputFake is the common firmware response simulating a
// validation rejection. Default Scenario for any quirk that doesn't need
// special-case behaviour.
func errInvalidInputFake() *fake.Fake {
	return fake.New().AlwaysError(scenarios.ErrInvalidInput101)
}

// attachCallbacks fills in Scenario and Match on a freshly loaded quirk
// based on its ID. Static detection is fully data-driven via the Patterns
// field — no function pointer required here.
func attachCallbacks(q *Quirk) {
	q.Scenario = errInvalidInputFake
	q.Match = matchInvalidInput.MatchString

	switch q.ID {
	case "E1":
		q.Scenario = func() *fake.Fake { return scenarios.RegressionUmlautEIP712() }
	case "E7", "C3", "B5":
		// Client-side validation surfaces these before they reach firmware,
		// so the wire-level invalid-input match would mis-attribute failures.
		q.Match = nil
	case "P1":
		q.Scenario = func() *fake.Fake {
			f, _ := scenarios.ChannelHashEarly([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 2)
			return f
		}
		q.Match = nil
	case "P2":
		q.Scenario = func() *fake.Fake { return scenarios.DeviceDisconnect(2) }
		q.Match = nil
	case "A1":
		q.Scenario = func() *fake.Fake {
			return scenarios.PanicMidQuery(1, "simulated panic in firmware call path")
		}
		q.Match = nil
	case "A2":
		q.Scenario = func() *fake.Fake {
			return scenarios.SlowResponse(15*time.Second, []byte{0x00})
		}
		q.Match = nil
	}
}
