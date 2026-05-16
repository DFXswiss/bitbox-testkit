// Command bitbox-audit scans a repository for known BitBox02 firmware-quirk
// regressions and emits a structured report.
//
// The audit runs source-level pattern checks against every supported file
// type (.go, .ts, .tsx, .js, .jsx, .dart). For each finding, the report
// names the quirk from the shared knowledge base — severity, firmware range,
// source citation and description are all derived from quirks.json.
//
// Exit codes:
//
//	0 — no findings
//	1 — usage / IO error
//	2 — findings present (any severity)
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DFXswiss/bitbox-testkit/go/bitbox/quirks"
)

// version is overwritten at build time via -ldflags "-X main.version=…".
// Defaults to "dev" for `go install`-from-source consumers.
var version = "dev"

const usage = `bitbox-audit — scan a BitBox-integrating repo for known firmware-quirk regressions.

Usage:
  bitbox-audit [flags]

Flags:
  --repo <path>           Repository to scan (default: ".")
  --firmware <version>    Restrict quirks to those applying to this firmware
                          version (e.g. "9.23.0"). Empty = all quirks.
  --format <kind>         "json" (default) or "markdown".
  --output <file>         Write report to file instead of stdout.
  --test-results <file>   Jest or "go test -json" output. Quirks named in
                          passing/failing tests are folded into Coverage.
  --version               Print version and exit.
  --help                  Show this help.

Examples:
  # Static-only scan of the current repo, pretty Markdown to stdout
  bitbox-audit --format markdown

  # Full pipeline: run Jest, then audit with dynamic coverage
  npx jest --json --outputFile=jest.json
  bitbox-audit --test-results jest.json --format markdown --output report.md

  # Get a plain-language narrative
  bitbox-audit --format json | bitbox-audit-explain

Quirk reference: %d quirks documented (filter with --firmware to narrow).
Knowledge base: github.com/DFXswiss/bitbox-testkit (quirks/SCHEMA.md)
`

func main() {
	// Custom flag set so we can emit our own usage block.
	fs := flag.NewFlagSet("bitbox-audit", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress Go's default error spam; we print our own

	var (
		repo        = fs.String("repo", ".", "")
		firmware    = fs.String("firmware", "", "")
		format      = fs.String("format", "json", "")
		output      = fs.String("output", "", "")
		testResults = fs.String("test-results", "", "")
		showVersion = fs.Bool("version", false, "")
		showHelp    = fs.Bool("help", false, "")
	)
	if err := fs.Parse(os.Args[1:]); err != nil {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	if *showHelp {
		printUsage(os.Stdout)
		return
	}
	if *showVersion {
		fmt.Printf("bitbox-audit %s\n", version)
		return
	}

	if err := run(*repo, *firmware, *format, *output, *testResults); err != nil {
		fmt.Fprintf(os.Stderr, "bitbox-audit: %v\n\n", err)
		printUsage(os.Stderr)
		os.Exit(1)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, usage, len(quirks.Registry))
}

func quirkIDs(qs []quirks.Quirk) []string {
	out := make([]string, len(qs))
	for i, q := range qs {
		out[i] = q.ID
	}
	return out
}

func run(repo, firmware, format, output, testResults string) error {
	abs, err := absPath(repo)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", repo, err)
	}
	files, err := enumerateSources(abs)
	if err != nil {
		return fmt.Errorf("enumerate sources: %w", err)
	}

	relevant := quirks.Subset(quirks.Filter{Firmware: firmware})
	findings := scan(abs, files, relevant)
	coverage := classify(relevant)
	testCov := loadTestCoverage(testResults, relevant)

	report := Report{
		Repo:       abs,
		Firmware:   firmware,
		FileCount:  len(files),
		QuirkCount: len(relevant),
		Findings:   findings,
		Summary:    summarize(findings),
		Coverage: CoverageReport{
			StaticIDs:      quirkIDs(coverage.Static),
			RuntimeOnlyIDs: quirkIDs(coverage.RuntimeOnly),
			TestCoverage:   testCov,
		},
	}

	w := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("open output: %w", err)
		}
		defer f.Close()
		w = f
	}

	switch format {
	case "json":
		if err := report.WriteJSON(w); err != nil {
			return err
		}
	case "markdown":
		if err := report.WriteMarkdown(w); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown format %q (want json or markdown)", format)
	}

	if len(findings) > 0 {
		os.Exit(2)
	}
	return nil
}
