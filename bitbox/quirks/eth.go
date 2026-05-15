package quirks

import (
	"regexp"

	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/fake"
	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/scenarios"
	"github.com/joshuakrueger-dfx/bitbox-testkit/core/guards"
)

// errInvalidInputFake is the common firmware response for malformed input.
// Most validation-rejecting quirks share this wire-level behavior.
func errInvalidInputFake() *fake.Fake {
	return fake.New().AlwaysError(scenarios.ErrInvalidInput101)
}

var (
	matchInvalidInput = regexp.MustCompile(`(?i)\binvalid[- ]?input\b|\b101\b`)

	nonAsciiInEIP712Line = regexp.MustCompile(`(?i)(eip712|signtyped).{0,200}["'][^"']*[\x80-\xff]`)
	gasLimitU64Line      = regexp.MustCompile(`(?m)gas[A-Za-z_]*\s*[:=]\s*(uint64|int64|big\.NewInt)\b`)
	missingChainID       = regexp.MustCompile(`(?m)ETHSignRequest{[^}]*}|NewSignRequest\([^,]+\)`)
)

func init() {
	Register(Quirk{
		ID:          "E1",
		Name:        "non-ascii-eip712-string",
		Category:    CategoryETH,
		Severity:    SeverityCritical,
		Description: "Firmware rejects EIP-712 string values containing non-ASCII bytes (umlauts, emojis) with ErrInvalidInput 101. The client must transliterate (e.g. NFKD + ASCII fallback) before signTypedData.",
		Source:      "Observed in production; ErrInvalidInput=101 in api/firmware/error.go",
		Firmware:    FirmwareRange{},
		Detect: func(t guards.TB, srcDir, include string) {
			guards.NoNonAsciiInEIP712Literals(t, srcDir, include)
		},
		Scenario: func() *fake.Fake { return scenarios.RegressionUmlautEIP712() },
		Match:    matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E2",
		Name:        "eth-nonce-max-16-bytes",
		Category:    CategoryETH,
		Severity:    SeverityWarning,
		Description: "nonce field of ETHSignRequest/EIP1559Request is capped at 16 bytes (smallest big-endian encoding). Leading zero-bytes or values > 2^128 are rejected.",
		Source:      "messages/eth.proto: nonce documented as max 16 bytes",
		Firmware:    FirmwareRange{},
		Detect: func(t guards.TB, srcDir, include string) {
			guards.MustNotMatch(t, srcDir, include, gasLimitU64Line,
				"prefer big.Int with MinimalBytes encoding for nonce/gas; uint64 paths can produce non-minimal bytes")
		},
		Scenario: errInvalidInputFake,
		Match:    matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E3",
		Name:        "eth-recipient-must-be-20-bytes",
		Category:    CategoryETH,
		Severity:    SeverityCritical,
		Description: "recipient field of ETHSignRequest must be exactly 20 bytes. Anything else (incl. EIP-55 strings, 0x-prefixed hex, ENS resolved late) is rejected with ErrInvalidInput.",
		Source:      "messages/eth.proto: recipient = 20 bytes",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E4",
		Name:        "eth-value-max-32-bytes",
		Category:    CategoryETH,
		Severity:    SeverityWarning,
		Description: "value field is capped at 32 bytes (smallest big-endian). Values > 2^256 are firmware-rejected.",
		Source:      "messages/eth.proto: value max 32 bytes",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E5",
		Name:        "eth-eip1559-fee-fields-max-16-bytes",
		Category:    CategoryETH,
		Severity:    SeverityWarning,
		Description: "max_priority_fee_per_gas and max_fee_per_gas are each capped at 16 bytes (smallest big-endian).",
		Source:      "messages/eth.proto: ETHSignEIP1559Request fee fields",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E6",
		Name:        "eth-numerics-must-be-minimal-bigendian",
		Category:    CategoryETH,
		Severity:    SeverityWarning,
		Description: "All ETH numeric fields require 'smallest big-endian serialization'. Leading zero bytes (incl. left-pad to 32) cause ErrInvalidInput. Strip leading zeros before sending.",
		Source:      "messages/eth.proto: documented for all numeric fields",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E7",
		Name:        "eth-address-case-enum-bounded",
		Category:    CategoryETH,
		Severity:    SeverityHint,
		Description: "address_case enum accepts 0 (MIXED), 1 (UPPER), 2 (LOWER) only. Other values rejected.",
		Source:      "messages/eth.proto: ETHAddressCase enum",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
	})

	Register(Quirk{
		ID:          "E8",
		Name:        "eth-data-length-must-match-streamed-data",
		Category:    CategoryETH,
		Severity:    SeverityCritical,
		Description: "When using streamed data signing, data_length must exactly match the byte count actually streamed. Mismatch results in firmware abort and signing failure mid-stream.",
		Source:      "messages/eth.proto: data_length / ETHSignRequest streaming",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E9",
		Name:        "eth-empty-data-zero-value-requires-v9.15.0+",
		Category:    CategoryETH,
		Severity:    SeverityWarning,
		Description: "Firmware < v9.15.0 rejects ETH transactions with empty data AND zero value (no-op transfers). v9.15.0 allows them.",
		Source:      "CHANGELOG v9.15.0: 'Allow ETH transactions with empty data + zero value'",
		Firmware:    FirmwareRange{Max: "9.15.0"},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "E10",
		Name:        "eth-known-network-list-versioned",
		Category:    CategoryETH,
		Severity:    SeverityHint,
		Description: "The firmware ships an allowlist of well-known chain IDs (Mainnet, BSC, etc.). New chains (HYPE, SONIC added v9.23.1) are unknown on older firmware and may require explicit user-confirmation flows on the client side, or fall back to 'unknown network' UX.",
		Source:      "CHANGELOG v9.23.1: 'Added HyperEVM (HYPE) and SONIC (S)'",
		Firmware:    FirmwareRange{Max: "9.23.1"},
		Scenario:    errInvalidInputFake,
	})

	_ = nonAsciiInEIP712Line // reserved for follow-up E1 inline pattern variants
	_ = missingChainID       // reserved for follow-up chain_id check
}
