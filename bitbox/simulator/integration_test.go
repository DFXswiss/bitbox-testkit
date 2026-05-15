//go:build simulator

package simulator_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/simulator"
)

// TestSimulatorRoundtrip launches the newest known BitBox02 simulator and
// verifies basic connectivity. Gated by the `simulator` build tag so
// `go test ./...` on a developer machine never triggers a binary download.
func TestSimulatorRoundtrip(t *testing.T) {
	cacheDir := os.Getenv("WALLET_TESTKIT_SIMCACHE")
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "bitbox-testkit-simcache")
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	inst, err := simulator.Launch(cacheDir)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	t.Cleanup(inst.Stop)

	if inst.Conn == nil {
		t.Fatal("Launch returned nil Conn")
	}
	if inst.Comm == nil {
		t.Fatal("Launch returned nil Comm")
	}
	// Give the simulator a moment to be fully ready, then close cleanly.
	time.Sleep(100 * time.Millisecond)
}
