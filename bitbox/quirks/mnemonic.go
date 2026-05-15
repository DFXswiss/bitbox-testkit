package quirks

func init() {
	Register(Quirk{
		ID:          "M1",
		Name:        "mnemonic-18-words-removed",
		Category:    CategoryMnemonic,
		Severity:    SeverityWarning,
		Description: "From v9.24.0, the firmware no longer accepts 18-word recovery phrases — only 12 or 24. Clients that still expose an 18-word UI flow will fail at restore-time on newer firmware.",
		Source:      "CHANGELOG v9.24.0: 'Removed support for 18-word recovery phrases'",
		Firmware:    FirmwareRange{Min: "9.24.0"},
		Scenario:    errInvalidInputFake,
		Match:       matchInvalidInput.MatchString,
	})

	Register(Quirk{
		ID:          "M2",
		Name:        "mnemonic-final-word-checksum-restricted",
		Category:    CategoryMnemonic,
		Severity:    SeverityHint,
		Description: "When restoring 12 or 18 (pre-v9.24.0) words, the firmware restricts the final word selection to checksum-valid candidates only. Clients must surface only valid candidates in the final-word picker, not the full BIP-39 list.",
		Source:      "CHANGELOG v9.16.0: 'restrict input to valid candidate words'",
		Firmware:    FirmwareRange{Min: "9.16.0"},
		Scenario:    errInvalidInputFake,
	})

	Register(Quirk{
		ID:          "M3",
		Name:        "mnemonic-zero-seed-words-versioned",
		Category:    CategoryMnemonic,
		Severity:    SeverityHint,
		Description: "Pre-v9.9.0, recovery phrases that derive a zero seed were rejected. From v9.9.0 they are accepted. Test seeds used for development across firmware versions may behave inconsistently.",
		Source:      "CHANGELOG v9.9.0: 'allow recovery words that convert to a zero seed'",
		Firmware:    FirmwareRange{Max: "9.9.0"},
		Scenario:    errInvalidInputFake,
	})
}
