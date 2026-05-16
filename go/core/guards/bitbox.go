// audit-skip-file: this file documents the BitBox guards consumed by
// test suites. Without this marker it would self-flag its own pattern
// references.

package guards

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/DFXswiss/bitbox-testkit/go/bitbox/quirks"
)

// RunQuirk applies every detect rule of the named quirk to source files
// under `root` matching `include` (a single glob like "*.go"). Each
// finding becomes a t.Errorf so the failure is attributable to the call
// site.
//
// Returns the number of findings emitted; usually you discard the value
// and rely on the test framework's failure tracking.
func RunQuirk(t TB, root, include, quirkID string) int {
	t.Helper()
	q := quirks.FindByID(quirkID)
	if q.ID == "" {
		t.Errorf("guards: no quirk with ID %q in registry", quirkID)
		return 0
	}
	return runPatterns(t, root, include, q)
}

// BitBoxDedupOrder fails if any source file under root reverses the
// `seenPackets.contains/has/includes` → `seenPackets.clear/removeAll/delete`
// ordering. Thin wrapper around quirk P2.
func BitBoxDedupOrder(t TB, root, include string) {
	t.Helper()
	RunQuirk(t, root, include, "P2")
}

// NoHardcoded10sTransportTimeout fails if any source file contains a
// hard-coded 10-second timeout pattern in a BitBox transport context.
// Thin wrapper around quirk A2.
func NoHardcoded10sTransportTimeout(t TB, root, include string) {
	t.Helper()
	RunQuirk(t, root, include, "A2")
}

// NoNonAsciiInEIP712Literals fails if any file in an EIP-712 context
// contains string literals with non-ASCII bytes. Thin wrapper around quirk E1.
func NoNonAsciiInEIP712Literals(t TB, root, include string) {
	t.Helper()
	RunQuirk(t, root, include, "E1")
}

// runPatterns walks `root`, opens every matching file, applies the quirk's
// detect rules, and reports findings.
func runPatterns(t TB, root, include string, q quirks.Quirk) int {
	t.Helper()
	if len(q.Patterns) == 0 {
		return 0
	}

	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || name == "dist" || name == "build" {
				return fs.SkipDir
			}
			if strings.HasPrefix(name, ".") && name != "." {
				return fs.SkipDir
			}
			return nil
		}
		if ok, _ := filepath.Match(include, filepath.Base(path)); !ok {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel := path
		if strings.HasPrefix(path, root+string(filepath.Separator)) {
			rel = path[len(root)+1:]
		}
		findings := quirks.ScanFile(rel, content, []quirks.Quirk{q})
		for _, f := range findings {
			count++
			t.Errorf("guards/%s: %s:%d  %s\n  reason: %s\n  fix:    %s",
				f.QuirkID, f.File, f.Line, f.Snippet, f.Reason, f.FixHint)
		}
		return nil
	})
	if err != nil {
		t.Errorf("guards: walk %s: %v", root, err)
	}
	return count
}
