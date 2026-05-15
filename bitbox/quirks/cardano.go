package quirks

func init() {
	Register(Quirk{
		ID:          "C1",
		Name:        "cardano-max-tx-16kb",
		Category:    CategoryCardano,
		Severity:    SeverityCritical,
		Description: "Cardano transactions are capped at 16384 bytes (CIP-0009). Bundles exceeding this — typically large native-token outputs or many UTXO inputs — are rejected.",
		Source:      "messages/cardano.proto: CIP-0009 16KB ceiling",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "C2",
		Name:        "cardano-network-mainnet-testnet-only",
		Category:    CategoryCardano,
		Severity:    SeverityCritical,
		Description: "network enum accepts CardanoMainnet (0) or CardanoTestnet (1). Any other value is rejected.",
		Source:      "messages/cardano.proto: CardanoNetwork enum",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
	})

	Register(Quirk{
		ID:          "C3",
		Name:        "cardano-drep-type-enum-bounded",
		Category:    CategoryCardano,
		Severity:    SeverityHint,
		Description: "Vote-delegation DRep type accepts KEY_HASH, SCRIPT_HASH, ALWAYS_ABSTAIN, ALWAYS_NO_CONFIDENCE only. Other values are rejected.",
		Source:      "messages/cardano.proto: CardanoCertificate.VoteDelegation DRep types",
		Firmware:    FirmwareRange{},
		Scenario:    errInvalidInputFake,
	})

	Register(Quirk{
		ID:          "C4",
		Name:        "cardano-duplicate-token-keys-rejected",
		Category:    CategoryCardano,
		Severity:    SeverityCritical,
		Description: "From v9.9.1, the firmware rejects Cardano outputs containing duplicate token keys (same policy_id + asset_name within one output). Pre-v9.9.1 quietly merged them — undefined value behaviour.",
		Source:      "CHANGELOG v9.9.1: 'Disallow duplicate token keys in Cardano outputs'",
		Firmware:    FirmwareRange{Min: "9.9.1"},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})
}
