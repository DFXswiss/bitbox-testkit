package simulator_test

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/simulator"
)

func TestSimulatorsListPopulated(t *testing.T) {
	bins := simulator.Simulators()
	if len(bins) == 0 {
		t.Fatal("embedded simulator list is empty")
	}
	for _, b := range bins {
		if !strings.HasPrefix(b.Name, "bitbox02-multi-") {
			t.Errorf("unexpected name: %s", b.Name)
		}
		if len(b.SHA256) != 64 {
			t.Errorf("bad hash for %s: %s", b.Name, b.SHA256)
		}
		if !strings.HasPrefix(b.URL, "https://github.com/BitBoxSwiss/") {
			t.Errorf("unexpected URL host for %s: %s", b.Name, b.URL)
		}
	}
}

func TestSimulatorsSortedNewestFirst(t *testing.T) {
	bins := simulator.Simulators()
	for i := 1; i < len(bins); i++ {
		if bins[i-1].Name < bins[i].Name {
			t.Fatalf("not sorted at index %d: %s < %s", i, bins[i-1].Name, bins[i].Name)
		}
	}
}

func TestLaunchRejectsNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("only meaningful off linux-amd64")
	}
	_, err := simulator.Launch(t.TempDir())
	if !errors.Is(err, simulator.ErrUnsupportedPlatform) {
		t.Fatalf("got %v, want ErrUnsupportedPlatform", err)
	}
}
