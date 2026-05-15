package quirks

import (
	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/fake"
)

func init() {
	Register(Quirk{
		ID:          "B1",
		Name:        "btc-locktime-below-500m",
		Category:    CategoryBTC,
		Severity:    SeverityWarning,
		Description: "BTCSignInitRequest.locktime must be < 500_000_000. Timestamp-style locktimes (>= 500M, interpreted as Unix seconds in Bitcoin Script) are firmware-rejected.",
		Source:      "messages/btc.proto: locktime constraint",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "B2",
		Name:        "btc-version-must-be-1-or-2",
		Category:    CategoryBTC,
		Severity:    SeverityCritical,
		Description: "BTCSignInitRequest.version must be 1 or 2. Other transaction versions are rejected.",
		Source:      "messages/btc.proto: version field",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "B3",
		Name:        "btc-sequence-restricted-values",
		Category:    CategoryBTC,
		Severity:    SeverityWarning,
		Description: "BTCSignInputRequest.sequence must be 0xffffffff, 0xffffffff-1 (RBF), or 0xffffffff-2. Other sequences were warning-flagged pre-v9.16.0 and may behave differently across versions.",
		Source:      "messages/btc.proto: sequence + CHANGELOG v9.16.0",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "B4",
		Name:        "btc-payload-size-matches-address-type",
		Category:    CategoryBTC,
		Severity:    SeverityCritical,
		Description: "BTCSignOutputRequest.payload must be 20 bytes for p2pkh/p2sh/p2wpkh and 32 bytes for p2wsh. Mismatch triggers firmware reject.",
		Source:      "messages/btc.proto: BTCSignOutputRequest payload",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "B5",
		Name:        "btc-signature-lengths-format-specific",
		Category:    CategoryBTC,
		Severity:    SeverityWarning,
		Description: "Transaction signatures: 64 bytes (32R+32S big-endian). Message signatures: 65 bytes (64 + 1 recovery ID). Mismatched length on the client side will fail signature verification.",
		Source:      "messages/btc.proto: BTCSignNextResponse signature, BTCSignMessageResponse signature",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
	})

	Register(Quirk{
		ID:          "B6",
		Name:        "btc-multisig-account-limit-25",
		Category:    CategoryBTC,
		Severity:    SeverityWarning,
		Description: "Firmware caps registered multisig accounts at 25 (was 10 pre-v9.6.0). Attempts beyond the limit are rejected.",
		Source:      "CHANGELOG v9.6.0: 'Maximum multisig accounts increased from 10 to 25'",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "B7",
		Name:        "btc-fee-attack-prev-out-index-check",
		Category:    CategoryBTC,
		Severity:    SeverityCritical,
		Description: "From v9.15.0 the firmware validates the index of each input's previous output against the supplied prevout (anti-fee-attack). Inconsistent prevout indexes are firmware-rejected. Older firmware accepted them — a silent fee-inflation attack vector.",
		Source:      "CHANGELOG v9.15.0: 'check index of input's previous output to prevent the fee attack'",
		Firmware:    FirmwareRange{Min: "9.15.0"},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	_ = fake.New // keep fake import live for future scenarios
}
