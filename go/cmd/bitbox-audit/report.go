package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Report is the top-level audit output.
type Report struct {
	Repo       string    `json:"repo"`
	Firmware   string    `json:"firmware,omitempty"`
	FileCount  int       `json:"files_scanned"`
	QuirkCount int       `json:"quirks_evaluated"`
	Findings   []Finding `json:"findings"`
	Summary    Summary   `json:"summary"`
}

type Summary struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Hint     int `json:"hint"`
	Total    int `json:"total"`
}

func summarize(findings []Finding) Summary {
	s := Summary{}
	for _, f := range findings {
		switch f.Severity {
		case "critical":
			s.Critical++
		case "warning":
			s.Warning++
		case "hint":
			s.Hint++
		}
	}
	s.Total = s.Critical + s.Warning + s.Hint
	return s
}

func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func (r Report) WriteMarkdown(w io.Writer) error {
	fmt.Fprintf(w, "# BitBox audit — %s\n\n", r.Repo)
	if r.Firmware != "" {
		fmt.Fprintf(w, "Firmware target: `%s`\n\n", r.Firmware)
	}
	fmt.Fprintf(w, "Files scanned: **%d** — Quirks evaluated: **%d**\n\n", r.FileCount, r.QuirkCount)
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "| Severity | Count |\n|---|---|\n")
	fmt.Fprintf(w, "| critical | %d |\n| warning | %d |\n| hint | %d |\n| **total** | **%d** |\n\n",
		r.Summary.Critical, r.Summary.Warning, r.Summary.Hint, r.Summary.Total)

	if len(r.Findings) == 0 {
		fmt.Fprintln(w, "_No findings — your codebase is clean for the static checks this kit knows about._")
		return nil
	}

	// Group by severity for readability.
	byOrder := []string{"critical", "warning", "hint"}
	groups := map[string][]Finding{}
	for _, f := range r.Findings {
		groups[f.Severity] = append(groups[f.Severity], f)
	}

	for _, sev := range byOrder {
		fs := groups[sev]
		if len(fs) == 0 {
			continue
		}
		sort.Slice(fs, func(i, j int) bool {
			if fs[i].QuirkID != fs[j].QuirkID {
				return fs[i].QuirkID < fs[j].QuirkID
			}
			if fs[i].File != fs[j].File {
				return fs[i].File < fs[j].File
			}
			return fs[i].Line < fs[j].Line
		})
		fmt.Fprintf(w, "## %s findings\n\n", titleize(sev))
		for _, f := range fs {
			fmt.Fprintf(w, "### `%s` — %s\n\n", f.QuirkID, f.QuirkName)
			fmt.Fprintf(w, "- **File:** `%s:%d`\n", f.File, f.Line)
			fmt.Fprintf(w, "- **Snippet:** `%s`\n", f.Snippet)
			fmt.Fprintf(w, "- **Reason:** %s\n", f.Reason)
			if f.FixHint != "" {
				fmt.Fprintf(w, "- **Fix:** %s\n", f.FixHint)
			}
			fmt.Fprintf(w, "- **Source:** %s\n\n", f.Source)
		}
	}
	return nil
}

func titleize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
