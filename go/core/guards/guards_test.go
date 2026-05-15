package guards_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/core/guards"
)

type fakeTB struct {
	errs []string
}

func (f *fakeTB) Helper()                            {}
func (f *fakeTB) Errorf(format string, args ...any)  { f.errs = append(f.errs, fmt.Sprintf(format, args...)) }
func (f *fakeTB) Logf(format string, args ...any)    {}

func TestMustNotMatchPassesOnClean(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "ok.go", "package x\n\nfunc Good() {}\n")

	f := &fakeTB{}
	guards.MustNotMatch(f, dir, "*.go", regexp.MustCompile(`forbidden`), "nope")
	if len(f.errs) != 0 {
		t.Fatalf("unexpected errors: %v", f.errs)
	}
}

func TestMustNotMatchFiresOnHit(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad.go", "package x\n// forbidden marker\nfunc X(){}\n")

	f := &fakeTB{}
	guards.MustNotMatch(f, dir, "*.go", regexp.MustCompile(`forbidden`), "no marker")
	if len(f.errs) == 0 {
		t.Fatal("expected guard to fire")
	}
}

func TestMustMatchAtLeast(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package x\nfunc A() { recoverPanic() }\n")
	write(t, dir, "b.go", "package x\nfunc B() { /* no guard */ }\n")

	f := &fakeTB{}
	guards.MustMatchAtLeast(f, dir, "*.go", regexp.MustCompile(`recoverPanic\(\)`), 2,
		"every export must guard")
	if len(f.errs) == 0 {
		t.Fatal("expected fewer-than-min failure")
	}
}

func TestMustOrderPairedFiresOnReversal(t *testing.T) {
	dir := t.TempDir()
	// removeAll appears before contains — bug pattern
	write(t, dir, "buggy.go", `package x
func f() {
	seenPackets.removeAll(stale)
	if seenPackets.contains(id) { return }
}
`)
	f := &fakeTB{}
	guards.BitBoxDedupOrder(f, dir, "*.go")
	if len(f.errs) == 0 {
		t.Fatal("BitBoxDedupOrder should have fired on reversed order")
	}

	// Same file but corrected order — guard should not fire
	dir2 := t.TempDir()
	write(t, dir2, "fixed.go", `package x
func f() {
	if seenPackets.contains(id) { return }
	seenPackets.removeAll(stale)
}
`)
	f2 := &fakeTB{}
	guards.BitBoxDedupOrder(f2, dir2, "*.go")
	if len(f2.errs) != 0 {
		t.Fatalf("correct order should pass, got: %v", f2.errs)
	}
}

func TestNoHardcoded10sTimeout(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad.go", `package x
import "time"
func f() { time.Sleep(10 * time.Second) }
`)
	f := &fakeTB{}
	guards.NoHardcoded10sTransportTimeout(f, dir, "*.go")
	if len(f.errs) == 0 {
		t.Fatal("expected 10s-timeout guard to fire")
	}
}

func TestNoNonAsciiEIP712(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad.go", `package x
var msg = "EIP712 hëllo"
`)
	f := &fakeTB{}
	guards.NoNonAsciiInEIP712Literals(f, dir, "*.go")
	if len(f.errs) == 0 {
		t.Fatal("expected umlaut guard to fire")
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
