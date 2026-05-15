package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/quirks"
)

func TestDetectNonAsciiEIP712Flags(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sign.ts", `import { signTypedData } from 'bitbox-api';
const msg = "hëllo from eip712 land";
signTypedData(msg);
`)
	files, err := enumerateSources(dir)
	if err != nil {
		t.Fatal(err)
	}
	q := findQuirk(t, "E1")
	got := detectNonAsciiEIP712(files, q, dir)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].QuirkID != "E1" {
		t.Fatalf("wrong quirk id: %s", got[0].QuirkID)
	}
	if !strings.Contains(got[0].Snippet, "hëllo") {
		t.Fatalf("snippet missing umlaut: %q", got[0].Snippet)
	}
}

func TestDetectNonAsciiEIP712IgnoresUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "i18n.ts", `export const greeting = "Grüße";`)
	files, _ := enumerateSources(dir)
	q := findQuirk(t, "E1")
	got := detectNonAsciiEIP712(files, q, dir)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (no EIP-712 context), got %d", len(got))
	}
}

func TestDetectBLEDedupOrderFlagsReversal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "buggy.go", `package u2fhid
func process(id string) {
    seenPackets.removeAll(stale)
    if seenPackets.contains(id) { return }
}
`)
	files, _ := enumerateSources(dir)
	q := findQuirk(t, "P2")
	got := detectBLEDedupOrder(files, q, dir)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestDetectBLEDedupOrderPassesCorrect(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "fixed.go", `package u2fhid
func process(id string) {
    if seenPackets.contains(id) { return }
    seenPackets.removeAll(stale)
}
`)
	files, _ := enumerateSources(dir)
	q := findQuirk(t, "P2")
	got := detectBLEDedupOrder(files, q, dir)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(got))
	}
}

func TestDetectHardcoded10sTimeoutFlags(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "transport.go", `package transport
import "time"
func wait() { time.Sleep(10 * time.Second) }
`)
	writeFile(t, dir, "ts-side.ts", `setTimeout(cb, 10000);`)
	files, _ := enumerateSources(dir)
	q := findQuirk(t, "A2")
	got := detectHardcoded10sTimeout(files, q, dir)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 findings (go + ts), got %d", len(got))
	}
}

func TestEnumerateSourcesSkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/a.ts", "")
	writeFile(t, dir, "node_modules/bad.ts", "")
	files, _ := enumerateSources(dir)
	if len(files) != 1 {
		t.Fatalf("expected 1 file (node_modules skipped), got %d: %v", len(files), files)
	}
}

func TestReportSummary(t *testing.T) {
	r := Report{
		Findings: []Finding{
			{Severity: "critical"},
			{Severity: "critical"},
			{Severity: "warning"},
		},
	}
	s := summarize(r.Findings)
	if s.Critical != 2 || s.Warning != 1 || s.Total != 3 {
		t.Fatalf("summary off: %+v", s)
	}
}

func findQuirk(t *testing.T, id string) quirks.Quirk {
	t.Helper()
	for _, q := range quirks.Registry {
		if q.ID == id {
			return q
		}
	}
	t.Fatalf("quirk %s not in registry", id)
	return quirks.Quirk{}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
