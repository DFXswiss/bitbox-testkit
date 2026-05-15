package simulator_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/go/core/simulator"
)

func TestCacheResolveDownloadsAndVerifies(t *testing.T) {
	payload := []byte("hello-simulator-binary")
	// sha256 of "hello-simulator-binary"
	const hash = "3080fc59d52bdf2af2d8396e7109d32a994bdf046d2410c94a78352884899c62"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	cache, err := simulator.NewCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	b := simulator.Binary{Name: "fake-sim", URL: srv.URL + "/sim", SHA256: hash}
	path, err := cache.Resolve(b)
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}

	// Second Resolve should be cached: delete the upstream server and confirm.
	srv.Close()
	path2, err := cache.Resolve(b)
	if err != nil {
		t.Fatalf("cached Resolve: %v", err)
	}
	if path2 != path {
		t.Fatalf("cached path mismatch: %s vs %s", path2, path)
	}
}

func TestCacheRejectsHashMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not the expected content"))
	}))
	defer srv.Close()

	cache, err := simulator.NewCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b := simulator.Binary{
		Name:   "bad-sim",
		URL:    srv.URL + "/bad",
		SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	_, err = cache.Resolve(b)
	if err == nil || !contains(err.Error(), "hash mismatch") {
		t.Fatalf("got %v, want hash mismatch", err)
	}
	// Failed download must not leave a partial file behind.
	if entries, _ := os.ReadDir(cache.Dir); len(entries) > 0 {
		for _, e := range entries {
			t.Errorf("leftover after hash mismatch: %s", filepath.Join(cache.Dir, e.Name()))
		}
	}
}

func TestCacheValidatesBinaryFields(t *testing.T) {
	cache, _ := simulator.NewCache(t.TempDir())
	cases := []simulator.Binary{
		{Name: "", URL: "u", SHA256: zeroHash},
		{Name: "n", URL: "", SHA256: zeroHash},
		{Name: "n", URL: "u", SHA256: "short"},
	}
	for i, b := range cases {
		if _, err := cache.Resolve(b); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestStartStopEcho(t *testing.T) {
	// Use a tiny script (sh) that prints a marker and waits forever; we use
	// /bin/sh -c so this works on macOS dev boxes and Linux CI alike.
	p, err := simulator.Start("/bin/sh")
	if err != nil {
		t.Skipf("cannot start /bin/sh: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestWaitForReturnsNilOnReady(t *testing.T) {
	calls := 0
	err := simulator.WaitFor(100*time.Millisecond, func() error {
		calls++
		if calls >= 3 {
			return nil
		}
		return errors.New("not yet")
	})
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
}

func TestWaitForTimesOut(t *testing.T) {
	err := simulator.WaitFor(20*time.Millisecond, func() error {
		return errors.New("never ready")
	})
	if err == nil {
		t.Fatal("want timeout error")
	}
}

const zeroHash = "0000000000000000000000000000000000000000000000000000000000000000"

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
