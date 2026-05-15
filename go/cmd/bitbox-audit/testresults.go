package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/quirks"
)

// quirkIDPattern matches references to a quirk ID inside test names.
// Three patterns are supported:
//
//   - "quirk E1" / "Quirk E1"           — separator after the word
//   - "QuirkE1" / "TestQuirkE1Foo"      — camelCase concatenation
//   - "E1:" / "E1 — foo" / "E1_bar"     — standalone with separator
//
// The character class after the captured ID ensures we don't pick "E10"
// in a name like "FE100Hz". The regex captures the ID into either of two
// groups depending on which alternative fired.
var quirkIDPattern = regexp.MustCompile(`[Qq]uirk[_ -]?([A-Z]\d{1,3})(?:[A-Z]|[^A-Za-z0-9]|$)|(?:^|[^A-Za-z0-9])([A-Z]\d{1,3})(?:[A-Z]|[^A-Za-z0-9]|$)`)

// TestCoverage maps each quirk that appears in a test name to whether all
// referencing tests passed. A quirk with at least one failing test is
// marked as failed regardless of any siblings passing.
type TestCoverage struct {
	// Covered IDs (referenced by at least one test), all tests passed.
	PassingIDs []string `json:"passing_ids"`
	// Referenced IDs with at least one failing test.
	FailingIDs []string `json:"failing_ids"`
}

// jestReport is the subset of `jest --json` we need. Per-file results live
// under `assertionResults` (not the intuitive `testResults` — the outer
// `testResults` is the file-level array).
type jestReport struct {
	TestResults []struct {
		AssertionResults []struct {
			FullName string `json:"fullName"`
			Title    string `json:"title"`
			Status   string `json:"status"` // "passed" | "failed" | "skipped" | …
		} `json:"assertionResults"`
	} `json:"testResults"`
}

// goTestEvent is the subset of `go test -json` we need.
type goTestEvent struct {
	Action  string `json:"Action"`
	Test    string `json:"Test"`
	Package string `json:"Package"`
}

// loadTestCoverage reads the file at `path`, auto-detects whether it's
// Jest (`--json --outputFile=...`) or `go test -json` output, and returns
// per-quirk coverage. Returns nil on read/parse failure rather than
// erroring out — the audit should still produce a report.
func loadTestCoverage(path string, applicable []quirks.Quirk) *TestCoverage {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit: cannot read %s: %v\n", path, err)
		return nil
	}

	known := map[string]bool{}
	for _, q := range applicable {
		known[q.ID] = true
	}

	statusByID := map[string]string{} // quirk ID -> "passed" | "failed"

	if isJestJSON(data) {
		var rep jestReport
		if err := json.Unmarshal(data, &rep); err != nil {
			fmt.Fprintf(os.Stderr, "audit: not parseable as Jest JSON: %v\n", err)
			return nil
		}
		for _, file := range rep.TestResults {
			for _, t := range file.AssertionResults {
				name := t.FullName
				if name == "" {
					name = t.Title
				}
				for _, id := range extractQuirkIDs(name, known) {
					updateStatus(statusByID, id, t.Status == "passed")
				}
			}
		}
	} else {
		// go test -json is newline-delimited JSON events, not a single object.
		for _, line := range splitLines(data) {
			var ev goTestEvent
			if err := json.Unmarshal(line, &ev); err != nil {
				continue
			}
			if ev.Test == "" {
				continue
			}
			if ev.Action != "pass" && ev.Action != "fail" {
				continue
			}
			for _, id := range extractQuirkIDs(ev.Test, known) {
				updateStatus(statusByID, id, ev.Action == "pass")
			}
		}
	}

	c := &TestCoverage{}
	for id, status := range statusByID {
		if status == "failed" {
			c.FailingIDs = append(c.FailingIDs, id)
		} else {
			c.PassingIDs = append(c.PassingIDs, id)
		}
	}
	sort.Strings(c.PassingIDs)
	sort.Strings(c.FailingIDs)
	return c
}

func isJestJSON(b []byte) bool {
	// Jest output starts with `{`; go test -json starts with `{` too on
	// every line but as multiple JSON values. A single Unmarshal succeeds
	// for Jest; for go test it would fail on the second line concatenated.
	var probe map[string]any
	return json.Unmarshal(b, &probe) == nil && probe["testResults"] != nil
}

func splitLines(b []byte) [][]byte {
	out := [][]byte{}
	start := 0
	for i, c := range b {
		if c == '\n' {
			if i > start {
				out = append(out, b[start:i])
			}
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}

func extractQuirkIDs(s string, known map[string]bool) []string {
	matches := quirkIDPattern.FindAllStringSubmatch(s, -1)
	var out []string
	seen := map[string]bool{}
	for _, m := range matches {
		for _, candidate := range m[1:] {
			if candidate == "" || seen[candidate] {
				continue
			}
			if known[candidate] {
				out = append(out, candidate)
				seen[candidate] = true
			}
		}
	}
	return out
}

func updateStatus(m map[string]string, id string, passed bool) {
	cur, exists := m[id]
	if !exists {
		if passed {
			m[id] = "passed"
		} else {
			m[id] = "failed"
		}
		return
	}
	// Once failed, stays failed.
	if cur == "failed" {
		return
	}
	if !passed {
		m[id] = "failed"
	}
}
