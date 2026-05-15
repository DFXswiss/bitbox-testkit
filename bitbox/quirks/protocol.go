package quirks

import (
	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/fake"
	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/scenarios"
	"github.com/joshuakrueger-dfx/bitbox-testkit/core/guards"
)

func init() {
	Register(Quirk{
		ID:          "P1",
		Name:        "pairing-channel-hash-before-confirm",
		Category:    CategoryProtocol,
		Severity:    SeverityCritical,
		Description: "bitbox02-api-go's pair() call blocks waiting for on-device user confirmation. The channel hash needed for client-side display is, however, already available before confirm. Sequential clients will display nothing until the user has confirmed (chicken-and-egg). Workaround: parallel-poll for the channel hash while pair() is blocking.",
		Source:      "Observed in production",
		Firmware:    FirmwareRange{},
		Scenario: func() *fake.Fake {
			f, _ := scenarios.ChannelHashEarly([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 2)
			return f
		},
	})

	Register(Quirk{
		ID:          "P2",
		Name:        "ble-init-frame-retransmit-must-pass",
		Category:    CategoryProtocol,
		Severity:    SeverityCritical,
		Description: "BLE links can legitimately retransmit init frames. Plugin-side de-dup logic must NOT drop them — otherwise multi-page signing aborts mid-flow (typically 1/13 → 2/13). The historical bug: seenPackets.removeAll() called before seenPackets.contains() check.",
		Source:      "Observed in production (multi-page sign abort)",
		Firmware:    FirmwareRange{},
		Detect: func(t guards.TB, srcDir, include string) {
			guards.BitBoxDedupOrder(t, srcDir, include)
		},
		Scenario: func() *fake.Fake {
			// Closest-fit available scenario for "operation aborts mid-stream".
			return scenarios.DeviceDisconnect(2)
		},
	})

	Register(Quirk{
		ID:          "P3",
		Name:        "btc-sequence-warning-changed-9.16.0",
		Category:    CategoryProtocol,
		Severity:    SeverityHint,
		Description: "Pre-v9.16.0 the firmware emitted a warning for 'unusual' sequence numbers in BTC inputs. v9.16.0 removed that warning. Clients building cross-version UX must account for both shapes.",
		Source:      "CHANGELOG v9.16.0",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
	})
}
