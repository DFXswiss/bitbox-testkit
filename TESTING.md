# Testing with bitbox-testkit

Consumer-facing guide. Read this before writing the first test in your plugin or app.

## Three layers, in order of cost

Pick the cheapest layer that can catch the bug class you care about.

| Layer | Catches | Where | Cost |
|-------|---------|-------|------|
| **API fake** (`bitbox/fake`, `bitbox/scenarios`) | App logic, error paths, gomobile panics, firmware error handling | Anywhere | < 1 s per test |
| **Transport fake** (`core/transport/ble`) | BLE framing, packet de-duplication, stale buffer | Anywhere | < 1 s per test |
| **Vendor simulator** (`bitbox/simulator`) | Real firmware behavior end-to-end | Linux CI only | 5–30 s per test |

Source-level **guards** (`core/guards`) run alongside the test suite and prevent known bad patterns from coming back through a refactor. Zero runtime cost.

## API fake

Use when you want to test client logic that talks to a BitBox via `firmware.NewDevice`. The fake plugs in as the `Communication`.

```go
import (
    "github.com/BitBoxSwiss/bitbox02-api-go/api/firmware"
    "github.com/BitBoxSwiss/bitbox02-api-go/api/firmware/mocks"
    "github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/fake"
)

f := fake.New().
    Expect([]byte{0x00}).
    ExpectError(myCustomFirmwareError)

dev := firmware.NewDevice(nil, nil, &mocks.Config{}, f, &mocks.Logger{})
// ... drive dev, then assert with f.Calls()
```

### Pre-built scenarios

`bitbox/scenarios` ships ready-made fakes for known bug classes:

- `RegressionUmlautEIP712()` — returns `ErrInvalidInput101` for any query containing non-ASCII bytes. Use this to assert your client transliterates EIP-712 payloads via `toBitboxSafeAscii` before sending.
- `ChannelHashEarly(hash, n)` — emits the channel hash before user confirm; verifies the parallel-poll pairing workaround.
- `DeviceDisconnect(after)` — closes the connection mid-flow.
- `PanicMidQuery(n, v)` — panics on the n-th query; proves `recoverPanic` on gomobile exports protects the host app.
- `SlowResponse(d, payload)` — for timeout-tuning regression tests.

## Transport fake

Use when the bug lives below the firmware.Communication contract — typically in BLE framing or packet de-duplication. The `Peripheral` is an `io.ReadWriteCloser` that your plugin's BLE adapter wraps.

```go
import "github.com/joshuakrueger-dfx/bitbox-testkit/core/transport/ble"

p := ble.New()
defer p.Close()

// Inject bytes the plugin's adapter will Read:
_ = p.Inject(initFramePayload)
_ = p.Inject(initFramePayload) // legitimate duplicate retransmit

// ... drive the plugin code that wraps p, then assert:
got := p.Sent()       // bytes the plugin wrote (next-page request, etc.)
p.WaitForWrite(N, d)  // synchronize without polling
```

This is the layer that catches the BLE-dedup class of bug — feed a duplicate init frame, then assert the plugin still asks for the continuation page rather than silently dropping it.

## Vendor simulator

Run the official BitBox02 Linux simulator binary and talk to it via the real wire format. Use sparingly: it's slow, Linux-only, and only meaningful for firmware-level behavior tests.

```go
//go:build simulator

import "github.com/joshuakrueger-dfx/bitbox-testkit/bitbox/simulator"

func TestE2E(t *testing.T) {
    inst, err := simulator.Launch(os.Getenv("WALLET_TESTKIT_SIMCACHE"))
    if err != nil {
        if errors.Is(err, simulator.ErrUnsupportedPlatform) {
            t.Skip("simulator requires linux/amd64")
        }
        t.Fatal(err)
    }
    t.Cleanup(inst.Stop)

    dev := firmware.NewDevice(nil, nil, &mocks.Config{}, inst.Comm, &mocks.Logger{})
    // ... drive dev against the real firmware logic
}
```

Gate the file with the `simulator` build tag so dev machines don't trigger a 50 MB download on `go test ./...`. The CI workflow runs both jobs in parallel: the unit job runs everything except the simulator integration; the simulator job runs only `-tags simulator` tests on Linux.

Override the binary at runtime by setting `BITBOX_SIMULATOR=/path/to/binary` — useful when developing firmware locally.

## Source guards

Run from your plugin's `_test.go` to enforce patterns the type system can't:

```go
import "github.com/joshuakrueger-dfx/bitbox-testkit/core/guards"

func TestSourceGuards(t *testing.T) {
    guards.BitBoxDedupOrder(t, "go/u2fhid", "*.go")
    guards.NoHardcoded10sTransportTimeout(t, "go/u2fhid", "*.go")
    guards.NoNonAsciiInEIP712Literals(t, "go", "*.go")
}
```

Each guard targets a specific historical bug — see the doc comment on the function for the incident reference.

## Adding a new scenario when a new bug class shows up

1. Write the smallest fake that reproduces the wire-level behavior the buggy firmware exhibited.
2. Add a constructor to `bitbox/scenarios` returning a configured `*fake.Fake`.
3. Add a doc comment that links to the bug context (commit, issue, memory key).
4. Add a unit test in `bitbox/scenarios/scenarios_test.go` verifying the scenario behaves as documented.
5. If the bug had a source-level pattern that should never come back, add a guard in `core/guards/bitbox.go`.

## What this kit deliberately does NOT do

- Full BitBox protobuf message construction — too much surface for too little gain. Use the official simulator when you need real protocol fidelity.
- Real noise-protocol handshake simulation — same reason.
- macOS/Windows simulator support — upstream only ships Linux/amd64.
- iOS/Android device emulation — the kit operates below platform channels, at the Go layer.

## Local commands

```bash
# Unit tests (no simulator, runs everywhere)
go test -race ./...

# Simulator integration (Linux/amd64 only; ~30s first run)
go test -tags simulator -timeout 5m ./bitbox/simulator/...
```

## CI

`.github/workflows/test.yml` runs the unit job on every PR and the simulator job in parallel on Ubuntu. Simulator binaries are cached by their SHA256 — refreshes only happen when `bitbox/simulator/embedded.go` changes.
