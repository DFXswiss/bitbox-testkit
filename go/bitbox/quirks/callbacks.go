package quirks

import (
	"regexp"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/fake"
	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/scenarios"
	"github.com/joshuakrueger-dfx/bitbox-testkit/go/core/guards"
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

// attachCallbacks fills in Detect / Scenario / Match on a freshly loaded
// quirk based on its ID. Quirks not listed here get the generic
// errInvalidInputFake scenario and the generic match.
func attachCallbacks(q *Quirk) {
	// Default behaviour applied first; specific cases overwrite below.
	q.Scenario = errInvalidInputFake
	q.Match = matchInvalidInput.MatchString

	switch q.ID {
	case "E1":
		q.Detect = func(t guards.TB, srcDir, include string) {
			guards.NoNonAsciiInEIP712Literals(t, srcDir, include)
		}
		q.Scenario = func() *fake.Fake { return scenarios.RegressionUmlautEIP712() }
	case "E7":
		q.Match = nil // not a wire-level firmware error; client-side enum validation
	case "C3":
		q.Match = nil
	case "B5":
		q.Match = nil // signature-length mismatches surface as client-side parse errors
	case "P1":
		q.Scenario = func() *fake.Fake {
			f, _ := scenarios.ChannelHashEarly([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 2)
			return f
		}
		q.Match = nil
	case "P2":
		q.Detect = func(t guards.TB, srcDir, include string) {
			guards.BitBoxDedupOrder(t, srcDir, include)
		}
		q.Scenario = func() *fake.Fake { return scenarios.DeviceDisconnect(2) }
		q.Match = nil
	case "A1":
		q.Scenario = func() *fake.Fake {
			return scenarios.PanicMidQuery(1, "simulated panic in firmware call path")
		}
		q.Match = nil
	case "A2":
		q.Detect = func(t guards.TB, srcDir, include string) {
			guards.NoHardcoded10sTransportTimeout(t, srcDir, include)
		}
		q.Scenario = func() *fake.Fake {
			return scenarios.SlowResponse(15*time.Second, []byte{0x00})
		}
		q.Match = nil
	}
}
