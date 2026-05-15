package quirks

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Finding is one detected occurrence of a quirk in source. Returned by Scan;
// shared between the audit-runner CLI and the core/guards test helpers so
// both surfaces stay drift-free.
type Finding struct {
	QuirkID   string `json:"quirk_id"`
	QuirkName string `json:"quirk_name"`
	Category  string `json:"category"`
	Severity  string `json:"severity"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Snippet   string `json:"snippet"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	FixHint   string `json:"fix_hint,omitempty"`
}

// ScanFile applies the detect rules of every quirk in `applicable` to a
// single file's contents and returns findings. relPath is what shows up in
// the Finding.File field (typically a repo-relative path).
//
// This is the engine used by both `cmd/bitbox-audit` (file enumeration +
// reporting around it) and `core/guards` (test-time wrappers that lift
// findings to testing.TB.Errorf calls). Keeping the engine here means
// adding or tuning a detection rule in quirks.json takes effect in both
// places without further code changes.
func ScanFile(relPath string, content []byte, applicable []Quirk) []Finding {
	cache := &regexCache{}
	prepared := prepareRules(cache, applicable)
	return scanPrepared(relPath, content, prepared)
}

// ScanContent is identical to ScanFile but does not require a path.
// Useful when scanning in-memory fixtures (e.g. test code).
func ScanContent(content []byte, applicable []Quirk) []Finding {
	return ScanFile("<content>", content, applicable)
}

// regexCache prevents recompiling the same pattern across rules.
type regexCache struct {
	mu sync.Mutex
	m  map[string]*regexp.Regexp
}

func (c *regexCache) get(p string) (*regexp.Regexp, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.m[p]; ok {
		return r, nil
	}
	r, err := regexp.Compile(p)
	if err != nil {
		return nil, err
	}
	if c.m == nil {
		c.m = map[string]*regexp.Regexp{}
	}
	c.m[p] = r
	return r, nil
}

type preparedRule struct {
	quirk       Quirk
	rule        DetectRule
	regex       *regexp.Regexp
	context     *regexp.Regexp
	before      *regexp.Regexp
	after       *regexp.Regexp
	pair        *regexp.Regexp
	withinLines int
}

func prepareRules(cache *regexCache, applicable []Quirk) []preparedRule {
	var prepared []preparedRule
	for _, q := range applicable {
		for _, rule := range q.Patterns {
			pr := preparedRule{quirk: q, rule: rule, withinLines: rule.WithinLines}
			if pr.withinLines <= 0 {
				pr.withinLines = 5
			}
			ok := true
			pair := func(target **regexp.Regexp, pattern string) {
				if !ok || pattern == "" {
					return
				}
				re, err := cache.get(pattern)
				if err != nil {
					ok = false
					return
				}
				*target = re
			}
			pair(&pr.regex, rule.Regex)
			pair(&pr.context, rule.ContextRegex)
			pair(&pr.before, rule.BeforeRegex)
			pair(&pr.after, rule.AfterRegex)
			pair(&pr.pair, rule.PairRegex)
			if ok {
				prepared = append(prepared, pr)
			}
		}
	}
	return prepared
}

type fileBuffer struct {
	rel       string
	content   []byte
	lines     []string
	skipLines map[int]bool
}

func newFileBuffer(rel string, content []byte) *fileBuffer {
	lines := strings.Split(string(content), "\n")
	skip := map[int]bool{}
	for i, line := range lines {
		if strings.Contains(line, "audit-skip-line") {
			skip[i+1] = true
		}
	}
	return &fileBuffer{rel: rel, content: content, lines: lines, skipLines: skip}
}

func scanPrepared(relPath string, content []byte, rules []preparedRule) []Finding {
	fb := newFileBuffer(relPath, content)
	var out []Finding
	for _, pr := range rules {
		if !matchesGlobs(relPath, pr.rule.FileGlobs) {
			continue
		}
		out = append(out, pr.apply(fb)...)
	}
	return out
}

func (pr preparedRule) apply(fb *fileBuffer) []Finding {
	switch pr.rule.Kind {
	case "regex":
		return pr.applyRegex(fb)
	case "regex_in_context":
		return pr.applyRegexInContext(fb)
	case "ordered_pair":
		return pr.applyOrderedPair(fb)
	case "missing_pair_within":
		return pr.applyMissingPairWithin(fb)
	}
	return nil
}

func (pr preparedRule) applyRegex(fb *fileBuffer) []Finding {
	if pr.regex == nil {
		return nil
	}
	var out []Finding
	for i, line := range fb.lines {
		if pr.regex.MatchString(line) && !fb.skipLines[i+1] {
			out = append(out, pr.finding(fb, i+1, line))
		}
	}
	return out
}

func (pr preparedRule) applyRegexInContext(fb *fileBuffer) []Finding {
	if pr.regex == nil || pr.context == nil || !pr.context.Match(fb.content) {
		return nil
	}
	var out []Finding
	for i, line := range fb.lines {
		if pr.regex.MatchString(line) && !fb.skipLines[i+1] {
			out = append(out, pr.finding(fb, i+1, line))
		}
	}
	return out
}

func (pr preparedRule) applyOrderedPair(fb *fileBuffer) []Finding {
	if pr.before == nil || pr.after == nil {
		return nil
	}
	beforeLoc := pr.before.FindIndex(fb.content)
	afterLoc := pr.after.FindIndex(fb.content)
	if beforeLoc == nil || afterLoc == nil || afterLoc[0] >= beforeLoc[0] {
		return nil
	}
	line := 1 + bytes.Count(fb.content[:afterLoc[0]], []byte{'\n'})
	if fb.skipLines[line] {
		return nil
	}
	return []Finding{pr.finding(fb, line, extractLine(fb.content, afterLoc[0]))}
}

func (pr preparedRule) applyMissingPairWithin(fb *fileBuffer) []Finding {
	if pr.regex == nil || pr.pair == nil {
		return nil
	}
	var out []Finding
	for i, line := range fb.lines {
		if !pr.regex.MatchString(line) || fb.skipLines[i+1] {
			continue
		}
		end := i + 1 + pr.withinLines
		if end > len(fb.lines) {
			end = len(fb.lines)
		}
		paired := false
		for _, follower := range fb.lines[i+1 : end] {
			if pr.pair.MatchString(follower) {
				paired = true
				break
			}
		}
		if !paired {
			out = append(out, pr.finding(fb, i+1, line))
		}
	}
	return out
}

func (pr preparedRule) finding(fb *fileBuffer, line int, snippet string) Finding {
	return Finding{
		QuirkID:   pr.quirk.ID,
		QuirkName: pr.quirk.Name,
		Category:  string(pr.quirk.Category),
		Severity:  pr.quirk.Severity.String(),
		File:      fb.rel,
		Line:      line,
		Snippet:   strings.TrimSpace(snippet),
		Reason:    pr.rule.Reason,
		Source:    pr.quirk.Source,
		FixHint:   pr.rule.FixHint,
	}
}

func matchesGlobs(path string, globs []string) bool {
	if len(globs) == 0 {
		return true
	}
	base := filepath.Base(path)
	for _, g := range globs {
		if ok, _ := filepath.Match(g, base); ok {
			return true
		}
	}
	return false
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

// FindByID returns the quirk with the matching ID, or zero-value Quirk if
// not found. Convenience for guards / consumers that operate on a single
// quirk at a time.
func FindByID(id string) Quirk {
	for _, q := range Registry {
		if q.ID == id {
			return q
		}
	}
	return Quirk{}
}
