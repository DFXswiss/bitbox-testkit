package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DFXswiss/bitbox-testkit/go/bitbox/quirks"
)

func TestExtractQuirkIDsFromFullNames(t *testing.T) {
	known := map[string]bool{"E1": true, "A1": true, "M1": true, "P2": true}
	cases := []struct {
		in   string
		want []string
	}{
		{
			"BitboxProvider — quirk E1 (non-ASCII EIP-712) signs an ASCII message",
			[]string{"E1"},
		},
		{
			"quirk A1: bridge throws synchronously",
			[]string{"A1"},
		},
		{
			"unrelated test about something",
			nil,
		},
		{
			"Quirk M1 — 18-word recovery option",
			[]string{"M1"},
		},
		{
			"quirk E1 and quirk P2 both apply",
			[]string{"E1", "P2"},
		},
		{
			"quirk Z99 not in registry",
			nil,
		},
	}
	for _, tc := range cases {
		got := extractQuirkIDs(tc.in, known)
		if len(got) != len(tc.want) {
			t.Errorf("input %q: got %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("input %q: got %v, want %v", tc.in, got, tc.want)
				break
			}
		}
	}
}

func TestLoadTestCoverageJest(t *testing.T) {
	jestJSON := `{
		"numFailedTests": 0,
		"testResults": [{
			"assertionResults": [
				{"fullName": "Provider — quirk E1 — passes ASCII", "status": "passed"},
				{"fullName": "Provider — quirk A1 — bridge throws", "status": "passed"},
				{"fullName": "Provider — quirk B7 — fee attack guard", "status": "failed"},
				{"fullName": "unrelated test", "status": "passed"}
			]
		}]
	}`
	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")
	if err := os.WriteFile(path, []byte(jestJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := loadTestCoverage(path, quirks.Registry)
	if c == nil {
		t.Fatal("expected coverage, got nil")
	}
	if len(c.PassingIDs) != 2 {
		t.Errorf("expected 2 passing, got %v", c.PassingIDs)
	}
	if len(c.FailingIDs) != 1 || c.FailingIDs[0] != "B7" {
		t.Errorf("expected B7 in failing, got %v", c.FailingIDs)
	}
}

func TestLoadTestCoverageGoTestJSON(t *testing.T) {
	// go test -json emits newline-delimited events; only `pass` and `fail`
	// actions on named tests should count.
	goJSON := `{"Action":"run","Test":"TestQuirkA1Recover"}
{"Action":"pass","Test":"TestQuirkA1Recover"}
{"Action":"run","Test":"TestQuirkE1Umlaut"}
{"Action":"fail","Test":"TestQuirkE1Umlaut"}
{"Action":"pass","Test":"TestSomethingElse"}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "go-test.json")
	if err := os.WriteFile(path, []byte(goJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := loadTestCoverage(path, quirks.Registry)
	if c == nil {
		t.Fatal("expected coverage, got nil")
	}
	// A1 passed; E1 failed (failure dominates).
	if len(c.PassingIDs) != 1 || c.PassingIDs[0] != "A1" {
		t.Errorf("expected A1 passing, got %v", c.PassingIDs)
	}
	if len(c.FailingIDs) != 1 || c.FailingIDs[0] != "E1" {
		t.Errorf("expected E1 failing, got %v", c.FailingIDs)
	}
}

func TestLoadTestCoverageReturnsNilForMissingFile(t *testing.T) {
	c := loadTestCoverage("/nonexistent/path.json", quirks.Registry)
	if c != nil {
		t.Fatalf("expected nil for missing file, got %v", c)
	}
}

func TestLoadTestCoverageReturnsNilForEmptyPath(t *testing.T) {
	c := loadTestCoverage("", quirks.Registry)
	if c != nil {
		t.Fatal("expected nil for empty path")
	}
}

func TestFailureDominatesAcrossTests(t *testing.T) {
	// One passing + one failing reference to the same quirk → failed wins.
	jestJSON := `{
		"testResults": [{
			"assertionResults": [
				{"fullName": "quirk E1 happy path", "status": "passed"},
				{"fullName": "quirk E1 sad path", "status": "failed"}
			]
		}]
	}`
	dir := t.TempDir()
	path := filepath.Join(dir, "r.json")
	_ = os.WriteFile(path, []byte(jestJSON), 0o644)
	c := loadTestCoverage(path, quirks.Registry)
	if c == nil {
		t.Fatal("nil coverage")
	}
	if len(c.PassingIDs) != 0 {
		t.Errorf("expected E1 NOT in passing (it has a failure), got %v", c.PassingIDs)
	}
	if len(c.FailingIDs) != 1 || c.FailingIDs[0] != "E1" {
		t.Errorf("expected E1 in failing, got %v", c.FailingIDs)
	}
}
