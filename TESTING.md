# Testing with bitbox-testkit

Read this before writing the first test in your plugin or app. The kit ships two implementations sharing one knowledge base. Pick the one that matches your stack.

## Which implementation?

| You're using                              | Pick |
| ----------------------------------------- | ---- |
| Flutter plugin built on `bitbox02-api-go` (Go + gomobile) | `/go/` |
| React Native + WASM via `bitbox-api` (Rust → WebView)     | `/ts/` |
| Web app talking to `bitbox-api` directly                  | `/ts/` |

Both sides cover the same firmware quirks, since the firmware doesn't care which SDK speaks to it. The implementations differ only in where they plug in: the Go fake replaces `firmware.Communication`; the TS fake replaces `PairedBitBox`.

## Test layers, cheapest first

| Layer | Catches | Where | Go | TS |
| ----- | ------- | ----- | -- | -- |
| **API fake** (`bitbox/fake`, `fake/`) | App logic, error paths, panics | Anywhere | ✓ | ✓ |
| **Transport fake** (`core/transport/ble`) | BLE framing, packet de-duplication | Anywhere | ✓ | — |
| **Source guards** (`core/guards`, `guards/`) | Known bad patterns | CI / pre-commit | ✓ | ✓ |
| **Vendor simulator** (`bitbox/simulator`) | Real firmware behaviour end-to-end | Linux CI | ✓ | — |

The TS side currently focuses on the API-fake layer. BLE-transport-faking is Go-only because it requires intercepting the framing layer below `bitbox-api`'s WASM. Most TS-side regressions are reachable through the API fake.

## Go cookbook

### Run a known scenario

```go
import (
    "github.com/BitBoxSwiss/bitbox02-api-go/api/firmware"
    "github.com/BitBoxSwiss/bitbox02-api-go/api/firmware/mocks"
    "github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/scenarios"
)

fake := scenarios.RegressionUmlautEIP712()
dev := firmware.NewDevice(nil, nil, &mocks.Config{}, fake, &mocks.Logger{})
// drive dev, assert your client transliterates EIP-712 payloads
```

### Drive every applicable quirk in a regression suite

```go
import "github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/quirks"

for _, q := range quirks.Subset(quirks.Filter{
    Category: quirks.CategoryETH,
    MinSeverity: quirks.SeverityWarning,
    Firmware: "9.23.0",
}) {
    t.Run(q.Name, func(t *testing.T) {
        if q.Scenario == nil { t.Skip(); return }
        fake := q.Scenario()
        // build a fresh device with this fake, run the relevant client code
    })
}
```

### Static guards on your source

```go
import "github.com/joshuakrueger-dfx/bitbox-testkit/go/core/guards"

func TestSourceGuards(t *testing.T) {
    guards.BitBoxDedupOrder(t, "go/u2fhid", "*.go")
    guards.NoHardcoded10sTransportTimeout(t, "go/u2fhid", "*.go")
    guards.NoNonAsciiInEIP712Literals(t, "go", "*.go")
}
```

### Vendor simulator

Build-tagged so dev machines don't download 50 MB on `go test ./...`:

```go
//go:build simulator

import "github.com/joshuakrueger-dfx/bitbox-testkit/go/bitbox/simulator"

func TestE2E(t *testing.T) {
    inst, err := simulator.Launch(os.Getenv("WALLET_TESTKIT_SIMCACHE"))
    if errors.Is(err, simulator.ErrUnsupportedPlatform) { t.Skip("Linux/amd64 only") }
    if err != nil { t.Fatal(err) }
    t.Cleanup(inst.Stop)
    // use inst.Comm with firmware.NewDevice
}
```

## TypeScript cookbook

### Install

```bash
npm install --save-dev @joshuakrueger-dfx/bitbox-testkit
```

### Mock bitbox-api in a Jest test

```ts
import { buildPairedBitBox } from '@joshuakrueger-dfx/bitbox-testkit/fake';
import { scenarioRegressionUmlautEIP712 } from '@joshuakrueger-dfx/bitbox-testkit/scenarios';

jest.mock('bitbox-api', () => {
  // Build the fake PairedBitBox once; reuse across the test file.
  const paired = buildPairedBitBox(scenarioRegressionUmlautEIP712());
  return {
    bitbox02ConnectAuto: async () => ({
      unlockAndPair: async () => ({ waitConfirm: async () => paired }),
    }),
  };
});

it('transliterates umlauts before signTypedMessage', async () => {
  // drive your client code; assert it doesn't hit ErrInvalidInput101
});
```

### Drive every applicable quirk

```ts
import { Registry, subset } from '@joshuakrueger-dfx/bitbox-testkit/quirks';
import { buildPairedBitBox } from '@joshuakrueger-dfx/bitbox-testkit/fake';

for (const q of subset({ category: 'eth', minSeverity: 'critical' })) {
  it(`handles quirk ${q.id} — ${q.name}`, async () => {
    if (!q.scenario) return;
    const paired = buildPairedBitBox(q.scenario() as ReturnType<typeof q.scenario>);
    // wire `paired` as the bitbox-api return value, run your code, assert
  });
}
```

### Static guards on your source

```ts
import { detectNonAsciiInEIP712Literals, expandGlobs } from '@joshuakrueger-dfx/bitbox-testkit/guards';

test('no non-ASCII in EIP-712 literals', () => {
  const files = expandGlobs(['src/features/hardware-wallet']);
  expect(detectNonAsciiInEIP712Literals(files)).toEqual([]);
});
```

## Adding a new quirk

1. Add an entry to `/go/bitbox/quirks/quirks.json` (the canonical source).
2. Run `./scripts/sync-quirks.sh` to refresh the TS copy.
3. Add a `case` in `/go/bitbox/quirks/callbacks.go` for the new ID with a Scenario factory (and Detect if a static pattern fits).
4. Add the same case in `/ts/src/quirks/callbacks.ts`.
5. Add tests in both languages.
6. Run `./scripts/sync-quirks.sh --check` locally — CI runs the same and fails on drift.

## What this kit deliberately does NOT do

- Full BitBox protobuf message construction. Use the official simulator for protocol-fidelity tests.
- Real noise-protocol handshake simulation.
- macOS / Windows simulator support (upstream only ships Linux/amd64).
- iOS / Android device emulation. The kit operates at the SDK boundary, not at the OS BLE/USB stack.
- Dart-side Flutter testing (`TestDefaultBinaryMessenger`) — that layer sits below us. Wire your widget tests around the API fake.
