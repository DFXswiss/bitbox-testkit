package quirks

import (
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/fake"
	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/scenarios"
	"github.com/joshuakrueger-dfx/bitbox-testkit/core/guards"
)

func init() {
	Register(Quirk{
		ID:          "A1",
		Name:        "gomobile-export-must-recover-panic",
		Category:    CategoryApp,
		Severity:    SeverityCritical,
		Description: "Every gomobile-exported function called from Flutter must defer-recover, otherwise a Go panic crashes the host app. Patterns: defer recoverPanic() at function entry, or wrap body in a helper returning (result, error).",
		Source:      "Observed across BitBox/Ledger/Trezor plugin patches",
		Firmware:    FirmwareRange{},
		Scenario: func() *fake.Fake {
			return scenarios.PanicMidQuery(1, "simulated panic in firmware call path")
		},
	})

	Register(Quirk{
		ID:          "A2",
		Name:        "transport-timeout-must-be-context-driven",
		Category:    CategoryApp,
		Severity:    SeverityCritical,
		Description: "Hard-coded transport timeouts (e.g. time.Sleep(10*time.Second), time.After(10*time.Second)) block legitimate user-confirm flows where the user takes longer than the timeout to confirm on-device. Use context deadlines with confirmation-aware extensions.",
		Source:      "Observed in production (long-confirm flow aborts)",
		Firmware:    FirmwareRange{},
		Detect: func(t guards.TB, srcDir, include string) {
			guards.NoHardcoded10sTransportTimeout(t, srcDir, include)
		},
		Scenario: func() *fake.Fake {
			return scenarios.SlowResponse(15*time.Second, []byte{0x00})
		},
	})

	Register(Quirk{
		ID:          "A3",
		Name:        "antiklepto-host-nonce-commitment-required",
		Category:    CategoryApp,
		Severity:    SeverityWarning,
		Description: "Modern firmware requires a host-nonce commitment phase before signing as anti-klepto protection. Clients that skip this phase fall back to unprotected signing, exposing key-recovery vectors via biased nonces.",
		Source:      "messages/antiklepto.proto",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
	})
}
