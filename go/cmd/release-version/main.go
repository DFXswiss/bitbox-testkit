// Command release-version reads commit subjects since a base ref and
// decides the next semantic-version bump according to Conventional
// Commits 1.0 — emitting the next "vMAJOR.MINOR.PATCH" tag (or, with
// --report, a human-readable explanation) on stdout.
//
// Designed to be called from the auto-tag CI workflow:
//
//	NEXT=$(go run ./cmd/release-version --base "$LATEST_TAG")
//	git tag -a "$NEXT" -m "Release $NEXT"
//
// Conventional Commits → semver mapping (see CONTRIBUTING.md "Releases"):
//
//	feat!:, fix!:, refactor!:, ...        → MAJOR
//	BREAKING CHANGE: in body              → MAJOR
//	feat:, feat(scope):                   → MINOR
//	fix:, perf:, refactor:                → PATCH
//	chore:, ci:, docs:, test:, style:,    → PATCH (defensive default)
//	build:, revert:                       → PATCH
//	(unrecognised subject)                → PATCH + warning on stderr
//
// Aggregate over every commit in the range: pick the highest bump
// encountered. A merge commit subject ("Merge pull request #N from …")
// is ignored — only the squash-style commits the PR actually
// contributed are read. Empty ranges report exit 4 ("no commits, no
// release") so the caller can short-circuit.
//
// Exit codes:
//
//	0  success — wrote next tag to stdout
//	2  invalid CLI flags or git invocation failed
//	3  base ref does not exist / unparseable input
//	4  no commits in the range (caller should skip the release step)
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	base := flag.String("base", "", "Base ref (latest tag). Range = base..HEAD. Empty = treat as initial release.")
	head := flag.String("head", "HEAD", "Head ref. Default HEAD.")
	report := flag.Bool("report", false, "Print a human-readable summary to stdout instead of just the version.")
	initial := flag.String("initial", "v0.1.0", "Tag to emit when base is empty (no prior tags).")
	flag.Parse()

	if err := run(*base, *head, *report, *initial, os.Stdout, os.Stderr); err != nil {
		exitCode := 2
		var ce codedError
		if errors.As(err, &ce) {
			exitCode = ce.code
		}
		fmt.Fprintln(os.Stderr, "release-version:", err)
		os.Exit(exitCode)
	}
}

// codedError is a sentinel error that carries an exit code.
type codedError struct {
	code int
	err  error
}

func (e codedError) Error() string { return e.err.Error() }
func (e codedError) Unwrap() error { return e.err }

func coded(code int, err error) error { return codedError{code: code, err: err} }

func run(base, head string, report bool, initial string, stdout, stderr *os.File) error {
	if base == "" {
		fmt.Fprintln(stdout, initial)
		if report {
			fmt.Fprintln(stdout, "(no prior tag — emitting initial)")
		}
		return nil
	}

	current, err := parseSemver(base)
	if err != nil {
		return coded(3, fmt.Errorf("parse base %q: %w", base, err))
	}

	commits, err := gitLog(base, head)
	if err != nil {
		return coded(2, err)
	}
	if len(commits) == 0 {
		return coded(4, fmt.Errorf("no commits between %s..%s — no release", base, head))
	}

	decision := decideBump(commits, stderr)
	next := applyBump(current, decision.Bump)

	fmt.Fprintln(stdout, next)
	if report {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, decision.Report())
	}
	return nil
}

// gitLog returns the commit subjects + bodies (separated by NUL byte) in
// base..head. Subjects/bodies are joined by \x00 inside one record;
// records are separated by \x1e (record-separator) so multi-paragraph
// bodies stay intact.
func gitLog(base, head string) ([]Commit, error) {
	rng := base + ".." + head
	cmd := exec.Command("git", "log", "--no-merges",
		"--pretty=format:%s%x00%b%x1e", rng)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log %s: %w", rng, err)
	}
	return parseLog(string(out)), nil
}

// parseLog splits the git log output into Commit records. Exposed for
// testing — callers feed pre-recorded fixture strings.
func parseLog(s string) []Commit {
	var out []Commit
	for _, rec := range strings.Split(s, "\x1e") {
		rec = strings.Trim(rec, "\n")
		if rec == "" {
			continue
		}
		// One record = subject \x00 body
		parts := strings.SplitN(rec, "\x00", 2)
		c := Commit{Subject: strings.TrimSpace(parts[0])}
		if len(parts) == 2 {
			c.Body = strings.TrimSpace(parts[1])
		}
		out = append(out, c)
	}
	return out
}

// Commit is one log record.
type Commit struct {
	Subject string
	Body    string
}

// Bump enumerates the semver bump levels.
type Bump int

const (
	BumpNone Bump = iota
	BumpPatch
	BumpMinor
	BumpMajor
)

func (b Bump) String() string {
	switch b {
	case BumpMajor:
		return "major"
	case BumpMinor:
		return "minor"
	case BumpPatch:
		return "patch"
	}
	return "none"
}

// subjectPattern matches Conventional Commits subject lines:
//
//	type(scope)!: message
//	   ^^^^^ ^^^ ^
//	   group1  group2 (the breaking "!")
var subjectPattern = regexp.MustCompile(`^(\w+)(?:\([^)]+\))?(!)?:\s+\S`)

// breakingPattern matches "BREAKING CHANGE:" (or "BREAKING-CHANGE:")
// anywhere in the commit body — the spec allows both spellings.
var breakingPattern = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE:`)

// decideBump aggregates per-commit bumps and returns the highest.
func decideBump(commits []Commit, warnOut *os.File) Decision {
	d := Decision{TotalCommits: len(commits)}
	for _, c := range commits {
		b, why := classify(c)
		d.PerCommit = append(d.PerCommit, CommitBump{Commit: c, Bump: b, Reason: why})
		if b > d.Bump {
			d.Bump = b
		}
		switch b {
		case BumpMajor:
			d.MajorCount++
		case BumpMinor:
			d.MinorCount++
		case BumpPatch:
			d.PatchCount++
		}
		if why == reasonUnrecognised && warnOut != nil {
			fmt.Fprintf(warnOut,
				"release-version: warning — non-conventional subject %q, treating as patch\n",
				c.Subject)
		}
	}
	if d.Bump == BumpNone && d.TotalCommits > 0 {
		// All commits were "no bump" classified (currently unreachable
		// because unrecognised falls back to patch, but defensive).
		d.Bump = BumpPatch
	}
	return d
}

const (
	reasonBreakingSuffix = "subject contains '!:' breaking suffix"
	reasonBreakingBody   = "body contains BREAKING CHANGE: footer"
	reasonFeat           = "feat: subject"
	reasonFix            = "fix/perf/refactor/etc subject"
	reasonNoOp           = "chore/ci/docs/test/style/build subject (patch-only categories)"
	reasonUnrecognised   = "non-conventional subject — defaulting to patch"
)

// classify decides the bump level for a single commit, returning the
// human-readable reason for the report.
func classify(c Commit) (Bump, string) {
	if breakingPattern.MatchString(c.Body) {
		return BumpMajor, reasonBreakingBody
	}
	m := subjectPattern.FindStringSubmatch(c.Subject)
	if m == nil {
		return BumpPatch, reasonUnrecognised
	}
	typ := strings.ToLower(m[1])
	breaking := m[2] == "!"
	if breaking {
		return BumpMajor, reasonBreakingSuffix
	}
	switch typ {
	case "feat":
		return BumpMinor, reasonFeat
	case "fix", "perf", "refactor", "revert":
		return BumpPatch, reasonFix
	default:
		// chore, ci, docs, test, style, build — patch-only categories.
		// Still bump patch so the release isn't completely missed.
		return BumpPatch, reasonNoOp
	}
}

// Decision is the aggregated outcome over all commits in the range.
type Decision struct {
	TotalCommits int
	MajorCount   int
	MinorCount   int
	PatchCount   int
	Bump         Bump
	PerCommit    []CommitBump
}

// CommitBump is one commit's decision.
type CommitBump struct {
	Commit Commit
	Bump   Bump
	Reason string
}

// Report renders a multi-line summary suitable for CI logs.
func (d Decision) Report() string {
	var b strings.Builder
	fmt.Fprintf(&b, "commits analysed: %d (major:%d minor:%d patch:%d)\n",
		d.TotalCommits, d.MajorCount, d.MinorCount, d.PatchCount)
	fmt.Fprintf(&b, "winning bump: %s\n\n", d.Bump)
	fmt.Fprintln(&b, "per-commit breakdown:")
	for _, cb := range d.PerCommit {
		// Truncate subject to keep CI logs readable; full subject is in
		// the git history if a maintainer needs it.
		subj := cb.Commit.Subject
		if len(subj) > 72 {
			subj = subj[:69] + "..."
		}
		fmt.Fprintf(&b, "  [%s] %s — %s\n", cb.Bump, subj, cb.Reason)
	}
	return b.String()
}

// applyBump computes the next semver from current + bump.
func applyBump(cur Semver, b Bump) string {
	switch b {
	case BumpMajor:
		return fmt.Sprintf("v%d.0.0", cur.Major+1)
	case BumpMinor:
		return fmt.Sprintf("v%d.%d.0", cur.Major, cur.Minor+1)
	default:
		return fmt.Sprintf("v%d.%d.%d", cur.Major, cur.Minor, cur.Patch+1)
	}
}

// Semver is a parsed major.minor.patch.
type Semver struct {
	Major, Minor, Patch int
}

var semverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

// parseSemver accepts "v0.4.6" or "0.4.6". Pre-release / build metadata
// is intentionally NOT supported here — the auto-tag flow only deals in
// release tags, not pre-releases.
func parseSemver(s string) (Semver, error) {
	m := semverPattern.FindStringSubmatch(s)
	if m == nil {
		return Semver{}, fmt.Errorf("not a vMAJOR.MINOR.PATCH tag: %q", s)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Semver{Major: major, Minor: minor, Patch: patch}, nil
}

