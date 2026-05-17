package main

import (
	"strings"
	"testing"
)

// TestClassify locks the Conventional Commits → bump mapping. Every
// row here is a contract the release-version tool ships to consumers;
// changing one without updating CONTRIBUTING.md "Releases" is a bug.
func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		body    string
		want    Bump
		reason  string
	}{
		// MAJOR (breaking)
		{"feat with ! suffix", "feat!: drop legacy API", "", BumpMajor, reasonBreakingSuffix},
		{"fix with ! suffix", "fix!: invert error code semantics", "", BumpMajor, reasonBreakingSuffix},
		{"feat with scope and ! suffix", "feat(api)!: rename endpoint", "", BumpMajor, reasonBreakingSuffix},
		{"chore with ! suffix", "chore!: bump go.mod requires go1.25", "", BumpMajor, reasonBreakingSuffix},

		// MAJOR (BREAKING CHANGE: footer)
		{"BREAKING CHANGE colon-space", "fix: small thing", "More text.\n\nBREAKING CHANGE: removed Foo()", BumpMajor, reasonBreakingBody},
		{"BREAKING-CHANGE hyphen", "fix: small thing", "BREAKING-CHANGE: dropped Bar", BumpMajor, reasonBreakingBody},
		{"BREAKING CHANGE mid-body", "feat: a", "preamble\nBREAKING CHANGE: foo\ntrailer", BumpMajor, reasonBreakingBody},
		// "BREAKING" alone, NOT a footer, doesn't trip the rule.
		{"plain BREAKING word in body", "fix: nothing", "the work is BREAKING ground", BumpPatch, reasonFix},

		// MINOR (feat)
		{"plain feat", "feat: new scenario", "", BumpMinor, reasonFeat},
		{"feat with scope", "feat(simulator): add BTC scenarios", "", BumpMinor, reasonFeat},
		{"feat with multi-word scope", "feat(go cmd): blah", "", BumpMinor, reasonFeat},
		{"feat case-insensitive type", "FEAT: capital type", "", BumpMinor, reasonFeat},

		// PATCH (fix/perf/refactor/revert)
		{"plain fix", "fix: address bug", "", BumpPatch, reasonFix},
		{"fix with scope", "fix(audit): suppress doc-comment false positive", "", BumpPatch, reasonFix},
		{"perf", "perf: avoid allocation in hot path", "", BumpPatch, reasonFix},
		{"refactor", "refactor: extract helper", "", BumpPatch, reasonFix},
		{"revert", "revert: undo bad change", "", BumpPatch, reasonFix},

		// PATCH (no-op-but-still-shipped categories)
		{"chore", "chore: dep bump", "", BumpPatch, reasonNoOp},
		{"ci", "ci: cache go modules", "", BumpPatch, reasonNoOp},
		{"docs", "docs: clarify CONTRIBUTING", "", BumpPatch, reasonNoOp},
		{"test", "test: cover edge case", "", BumpPatch, reasonNoOp},
		{"style", "style: gofmt", "", BumpPatch, reasonNoOp},
		{"build", "build: update Makefile", "", BumpPatch, reasonNoOp},

		// Unrecognised → patch + warning (the warning side is checked
		// in TestDecideBumpWarnsOnUnrecognised below).
		{"unrecognised no colon", "wat", "", BumpPatch, reasonUnrecognised},
		{"missing colon", "feat new thing", "", BumpPatch, reasonUnrecognised},
		{"colon but no message", "feat:", "", BumpPatch, reasonUnrecognised},
		{"colon but only whitespace after", "feat:   ", "", BumpPatch, reasonUnrecognised},
		{"odd prefix", "FEATURE: too verbose", "", BumpPatch, reasonNoOp}, // "FEATURE" parses as type → falls to default arm

		// Body without breaking footer doesn't promote.
		{"feat with normal body", "feat: thing", "we did the thing.", BumpMinor, reasonFeat},
		{"fix with body referencing breaks", "fix: nothing", "before this change tests were breaking, now fixed", BumpPatch, reasonFix},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, why := classify(Commit{Subject: tt.subject, Body: tt.body})
			if got != tt.want {
				t.Errorf("classify(%q, body=%q) = %s, want %s (reason: %s)",
					tt.subject, tt.body, got, tt.want, why)
			}
			if why != tt.reason {
				t.Errorf("classify(%q) reason = %q, want %q", tt.subject, why, tt.reason)
			}
		})
	}
}

// TestDecideBumpPicksHighest verifies the aggregator picks the largest
// bump across a range, not "the last one wins" or anything similar.
func TestDecideBumpPicksHighest(t *testing.T) {
	commits := []Commit{
		{Subject: "chore: dep bump"},
		{Subject: "fix: small thing"},
		{Subject: "feat: new scenario"}, // this one promotes the whole range to MINOR
		{Subject: "docs: update README"},
	}
	d := decideBump(commits, nil)
	if d.Bump != BumpMinor {
		t.Fatalf("decideBump = %s, want minor", d.Bump)
	}
	if d.TotalCommits != 4 {
		t.Errorf("TotalCommits = %d, want 4", d.TotalCommits)
	}
	if d.MinorCount != 1 {
		t.Errorf("MinorCount = %d, want 1", d.MinorCount)
	}
	if d.PatchCount != 3 {
		t.Errorf("PatchCount = %d, want 3", d.PatchCount)
	}
}

// TestDecideBumpBreakingWins verifies that a single breaking change in
// a sea of patches still bumps major.
func TestDecideBumpBreakingWins(t *testing.T) {
	commits := []Commit{
		{Subject: "fix: a"},
		{Subject: "fix: b"},
		{Subject: "feat!: remove deprecated API"},
		{Subject: "chore: c"},
	}
	d := decideBump(commits, nil)
	if d.Bump != BumpMajor {
		t.Fatalf("decideBump = %s, want major", d.Bump)
	}
	if d.MajorCount != 1 {
		t.Errorf("MajorCount = %d, want 1", d.MajorCount)
	}
}

// TestApplyBumpMatrix locks the SemVer math.
func TestApplyBumpMatrix(t *testing.T) {
	cur := Semver{Major: 0, Minor: 4, Patch: 6}
	tests := []struct {
		bump Bump
		want string
	}{
		{BumpPatch, "v0.4.7"},
		{BumpMinor, "v0.5.0"},
		{BumpMajor, "v1.0.0"},
		{BumpNone, "v0.4.7"}, // defensive — falls to patch path
	}
	for _, tt := range tests {
		got := applyBump(cur, tt.bump)
		if got != tt.want {
			t.Errorf("applyBump(%v, %s) = %q, want %q", cur, tt.bump, got, tt.want)
		}
	}
}

// TestApplyBumpResetsLowerComponents — a minor bump zeroes patch, a
// major bump zeroes minor AND patch.
func TestApplyBumpResetsLowerComponents(t *testing.T) {
	cur := Semver{Major: 1, Minor: 7, Patch: 3}
	if got := applyBump(cur, BumpMinor); got != "v1.8.0" {
		t.Errorf("minor bump from 1.7.3 = %q, want v1.8.0", got)
	}
	if got := applyBump(cur, BumpMajor); got != "v2.0.0" {
		t.Errorf("major bump from 1.7.3 = %q, want v2.0.0", got)
	}
}

// TestParseSemver accepts the v-prefix or bare form; rejects everything
// else loudly so the caller can't silently feed a tag that the auto-
// tag script can't increment.
func TestParseSemver(t *testing.T) {
	good := map[string]Semver{
		"v0.4.6": {0, 4, 6},
		"0.4.6":  {0, 4, 6},
		"v1.0.0": {1, 0, 0},
		"v10.20.30": {10, 20, 30},
	}
	for in, want := range good {
		got, err := parseSemver(in)
		if err != nil {
			t.Errorf("parseSemver(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseSemver(%q) = %+v, want %+v", in, got, want)
		}
	}

	bad := []string{
		"",
		"main",
		"v0.4",         // missing patch
		"v0.4.6-rc1",   // pre-release not supported
		"v0.4.6+build", // build metadata not supported
		"go/v0.4.6",    // submodule prefix is for the OTHER tag, not this one
		"latest",
	}
	for _, in := range bad {
		if _, err := parseSemver(in); err == nil {
			t.Errorf("parseSemver(%q) accepted, want error", in)
		}
	}
}

// TestParseLogIgnoresEmpty + handles trailing record-separator + keeps
// multi-paragraph bodies intact.
func TestParseLog(t *testing.T) {
	// Format: subject \x00 body \x1e
	in := "feat: thing\x00body line 1\n\nbody line 2\x1e" +
		"fix: other\x00\x1e" +
		"\x1e" + // empty record between (gracefully ignored)
		"chore: third\x00single-line body\x1e"
	got := parseLog(in)
	if len(got) != 3 {
		t.Fatalf("parseLog returned %d records, want 3 (got: %+v)", len(got), got)
	}
	if got[0].Subject != "feat: thing" {
		t.Errorf("record 0 subject = %q", got[0].Subject)
	}
	if !strings.Contains(got[0].Body, "body line 1") || !strings.Contains(got[0].Body, "body line 2") {
		t.Errorf("record 0 body lost multi-paragraph content: %q", got[0].Body)
	}
	if got[1].Subject != "fix: other" || got[1].Body != "" {
		t.Errorf("record 1 = %+v, want subject 'fix: other' empty body", got[1])
	}
	if got[2].Subject != "chore: third" {
		t.Errorf("record 2 subject = %q", got[2].Subject)
	}
}

// TestReportShape locks the report text format because consumers
// (CI logs, release-notes generators) parse it.
func TestReportShape(t *testing.T) {
	d := decideBump([]Commit{
		{Subject: "feat: a"},
		{Subject: "fix: b"},
	}, nil)
	r := d.Report()
	mustContain := []string{
		"commits analysed: 2",
		"minor:1",
		"patch:1",
		"winning bump: minor",
		"feat: a",
		"fix: b",
	}
	for _, s := range mustContain {
		if !strings.Contains(r, s) {
			t.Errorf("report missing %q in:\n%s", s, r)
		}
	}
}
