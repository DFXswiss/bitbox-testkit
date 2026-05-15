package main

import (
	"os"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/quirks"
)

// Finding is the audit-runner-local alias for quirks.Finding so report.go
// keeps a stable shape independent of internal type moves.
type Finding = quirks.Finding

// scan walks every file once, asks quirks.ScanFile to apply each
// applicable rule, and aggregates findings.
func scan(root string, files []string, applicable []quirks.Quirk) []Finding {
	var out []Finding
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		rel := relative(root, path)
		out = append(out, quirks.ScanFile(rel, content, applicable)...)
	}
	return out
}

func relative(root, path string) string {
	if len(path) > len(root)+1 && path[:len(root)+1] == root+"/" {
		return path[len(root)+1:]
	}
	return path
}

// Coverage classifies each quirk's static-detectability and how the audit
// runner can report on it.
type Coverage struct {
	Static      []quirks.Quirk // has at least one Pattern → audit-runner checks it statically
	RuntimeOnly []quirks.Quirk // no Patterns → only catchable via runtime tests
}

func classify(applicable []quirks.Quirk) Coverage {
	c := Coverage{}
	for _, q := range applicable {
		if len(q.Patterns) > 0 {
			c.Static = append(c.Static, q)
		} else {
			c.RuntimeOnly = append(c.RuntimeOnly, q)
		}
	}
	return c
}
