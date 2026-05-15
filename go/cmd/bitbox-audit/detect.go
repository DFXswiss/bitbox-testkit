package main

import (
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/quirks"
)

// Finding is one detected occurrence of a quirk in source.
type Finding struct {
	QuirkID    string `json:"quirk_id"`
	QuirkName  string `json:"quirk_name"`
	Category   string `json:"category"`
	Severity   string `json:"severity"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Snippet    string `json:"snippet"`
	Reason     string `json:"reason"`
	Source     string `json:"source"`
	FixHint    string `json:"fix_hint,omitempty"`
}

// scan applies every applicable detector to the given source files and
// returns the aggregated findings.
func scan(root string, files []string, applicable []quirks.Quirk) []Finding {
	var out []Finding
	for _, q := range applicable {
		switch q.ID {
		case "E1":
			out = append(out, detectNonAsciiEIP712(files, q, root)...)
		case "P2":
			out = append(out, detectBLEDedupOrder(files, q, root)...)
		case "A2":
			out = append(out, detectHardcoded10sTimeout(files, q, root)...)
		}
	}
	return out
}

// Pre-compiled patterns used by static detectors. Kept in this file so
// adding a new quirk detector means one edit.
var (
	eip712Keyword       = regexp.MustCompile(`(?i)eip712|signtyped|signEthTyped`)
	nonAsciiStringLit   = regexp.MustCompile(`["'][^"']*[\x80-\xff][^"']*["']`)
	seenPacketsContains = regexp.MustCompile(`seenPackets\.(has|contains|includes)\s*\(`)
	seenPacketsRemove   = regexp.MustCompile(`seenPackets\.(clear|removeAll|delete)\s*\(`)
	hardcoded10sTimeout = regexp.MustCompile(`time\.(Sleep|After)\(\s*10\s*\*\s*time\.Second\s*\)|setTimeout\s*\([^,)]+,\s*10000\s*\)|10\s*\*\s*1000`)
)

func detectNonAsciiEIP712(files []string, q quirks.Quirk, root string) []Finding {
	var out []Finding
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if !eip712Keyword.Match(content) {
			continue
		}
		for i, line := range strings.Split(string(content), "\n") {
			if nonAsciiStringLit.MatchString(line) {
				out = append(out, Finding{
					QuirkID:   q.ID,
					QuirkName: q.Name,
					Category:  string(q.Category),
					Severity:  q.Severity.String(),
					File:      relative(root, path),
					Line:      i + 1,
					Snippet:   strings.TrimSpace(line),
					Reason:    "non-ASCII string literal in a file that touches EIP-712 / signTyped APIs",
					Source:    q.Source,
					FixHint:   "transliterate via NFKD + ASCII fallback before sending to BitBox firmware",
				})
			}
		}
	}
	return out
}

func detectBLEDedupOrder(files []string, q quirks.Quirk, root string) []Finding {
	var out []Finding
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		containsLoc := seenPacketsContains.FindIndex(content)
		removeLoc := seenPacketsRemove.FindIndex(content)
		if containsLoc == nil || removeLoc == nil {
			continue
		}
		if removeLoc[0] < containsLoc[0] {
			line := 1 + bytes.Count(content[:removeLoc[0]], []byte{'\n'})
			out = append(out, Finding{
				QuirkID:   q.ID,
				QuirkName: q.Name,
				Category:  string(q.Category),
				Severity:  q.Severity.String(),
				File:      relative(root, path),
				Line:      line,
				Snippet:   extractLine(content, removeLoc[0]),
				Reason:    "seenPackets.clear/removeAll/delete appears before contains/has/includes",
				Source:    q.Source,
				FixHint:   "reorder so the membership check runs before any removal",
			})
		}
	}
	return out
}

func detectHardcoded10sTimeout(files []string, q quirks.Quirk, root string) []Finding {
	var out []Finding
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(content), "\n") {
			if hardcoded10sTimeout.MatchString(line) {
				out = append(out, Finding{
					QuirkID:   q.ID,
					QuirkName: q.Name,
					Category:  string(q.Category),
					Severity:  q.Severity.String(),
					File:      relative(root, path),
					Line:      i + 1,
					Snippet:   strings.TrimSpace(line),
					Reason:    "hard-coded 10-second timeout in transport code",
					Source:    q.Source,
					FixHint:   "switch to context-driven deadlines that can be extended during long user-confirm flows",
				})
			}
		}
	}
	return out
}

func extractLine(content []byte, offset int) string {
	start := offset
	for start > 0 && content[start-1] != '\n' {
		start--
	}
	end := offset
	for end < len(content) && content[end] != '\n' {
		end++
	}
	return strings.TrimSpace(string(content[start:end]))
}

func relative(root, path string) string {
	if strings.HasPrefix(path, root+"/") {
		return path[len(root)+1:]
	}
	return path
}
