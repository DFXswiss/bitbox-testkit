package quirks_test

import (
	"testing"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/quirks"
)

func TestFirmwareRangeApplies(t *testing.T) {
	cases := []struct {
		name string
		r    quirks.FirmwareRange
		v    string
		want bool
	}{
		{"open-open accepts all", quirks.FirmwareRange{}, "9.23.0", true},
		{"open-max within", quirks.FirmwareRange{Max: "9.24.0"}, "9.23.0", true},
		{"open-max boundary excluded", quirks.FirmwareRange{Max: "9.24.0"}, "9.24.0", false},
		{"min-open within", quirks.FirmwareRange{Min: "9.20.0"}, "9.21.0", true},
		{"min-open boundary included", quirks.FirmwareRange{Min: "9.20.0"}, "9.20.0", true},
		{"min-open below", quirks.FirmwareRange{Min: "9.20.0"}, "9.19.0", false},
		{"min-max within", quirks.FirmwareRange{Min: "9.20.0", Max: "9.25.0"}, "9.22.0", true},
		{"v-prefix tolerated", quirks.FirmwareRange{Min: "9.20.0"}, "v9.21.0", true},
		{"empty input passes", quirks.FirmwareRange{Min: "9.20.0"}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.Applies(tc.v); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRegistryNonEmpty(t *testing.T) {
	if len(quirks.Registry) == 0 {
		t.Fatal("registry is empty — category init() did not run?")
	}
}

func TestRegistryNoDuplicateIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, q := range quirks.Registry {
		if seen[q.ID] {
			t.Errorf("duplicate quirk ID: %s", q.ID)
		}
		seen[q.ID] = true
	}
}

func TestRegistryFieldsPresent(t *testing.T) {
	for _, q := range quirks.Registry {
		if q.ID == "" {
			t.Errorf("quirk missing ID: %+v", q)
		}
		if q.Name == "" {
			t.Errorf("quirk %s missing Name", q.ID)
		}
		if q.Category == "" {
			t.Errorf("quirk %s missing Category", q.ID)
		}
		if q.Description == "" {
			t.Errorf("quirk %s missing Description", q.ID)
		}
		if q.Source == "" {
			t.Errorf("quirk %s missing Source", q.ID)
		}
		if q.Detect == nil && q.Scenario == nil {
			t.Errorf("quirk %s has neither Detect nor Scenario — useless entry", q.ID)
		}
	}
}

func TestSubsetByCategory(t *testing.T) {
	eth := quirks.Subset(quirks.Filter{Category: quirks.CategoryETH})
	if len(eth) == 0 {
		t.Fatal("expected at least one ETH quirk")
	}
	for _, q := range eth {
		if q.Category != quirks.CategoryETH {
			t.Errorf("filter leaked non-ETH quirk %s", q.ID)
		}
	}
}

func TestSubsetByMinSeverity(t *testing.T) {
	crit := quirks.Subset(quirks.Filter{MinSeverity: quirks.SeverityCritical})
	for _, q := range crit {
		if q.Severity < quirks.SeverityCritical {
			t.Errorf("filter leaked non-critical quirk %s", q.ID)
		}
	}
}

func TestSubsetByFirmware(t *testing.T) {
	// Pick an old firmware; quirks with Min > 9.10.0 should be excluded.
	old := quirks.Subset(quirks.Filter{Firmware: "9.10.0"})
	for _, q := range old {
		if !q.Firmware.Applies("9.10.0") {
			t.Errorf("filter leaked out-of-range quirk %s (%s) for v9.10.0", q.ID, q.Firmware)
		}
	}
}

func TestSeverityString(t *testing.T) {
	cases := map[quirks.Severity]string{
		quirks.SeverityHint:     "hint",
		quirks.SeverityWarning:  "warning",
		quirks.SeverityCritical: "critical",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("Severity(%d).String() = %s, want %s", s, got, want)
		}
	}
}
