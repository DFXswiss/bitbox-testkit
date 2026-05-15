# bitbox-testkit

Test infrastructure for BitBox02 integrations. Two implementations share one knowledge base:

| Stack | Lives at | Targets |
| ----- | -------- | ------- |
| **Go** | `/go/` | `bitbox02-api-go` (Flutter plugins, native Go consumers) |
| **TypeScript** | `/ts/` | `bitbox-api` Rust/WASM (React Native, web, Node) |

Both load the same firmware-constraint database (`/go/bitbox/quirks/quirks.json`) and attach language-specific Scenario / Detect callbacks by quirk ID. A CI check (`scripts/sync-quirks.sh --check`) prevents drift between the Go-side canonical and the TS-side copy.

## What it provides

- **Scriptable fakes** — drop-in replacement for `bitbox02-api-go`'s `firmware.Communication` (Go) or `bitbox-api`'s `PairedBitBox` (TS).
- **Pre-built scenarios** — Umlaut-rejection, BLE-dedup retransmit, gomobile/WebView panic, slow user-confirm, etc.
- **Quirks registry** — 30 documented BitBox firmware constraints with severity, source citation and firmware version range.
- **Source-level guards** — regex-based static checks for known bad patterns (BLE-dedup ordering, hard-coded 10s timeouts, non-ASCII in EIP-712 string literals).
- **Vendor simulator integration** (Go only, Linux/amd64) — downloads and runs the official BitBox02 simulator binary.

## Quick start

### Go

```bash
go get github.com/joshuakrueger-dfx/bitbox-testkit/go
```

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

### TypeScript

```bash
npm install --save-dev @joshuakrueger-dfx/bitbox-testkit
```

```ts
import { buildPairedBitBox } from '@joshuakrueger-dfx/bitbox-testkit/fake';
import { scenarioRegressionUmlautEIP712 } from '@joshuakrueger-dfx/bitbox-testkit/scenarios';

jest.mock('bitbox-api', () => {
  return {
    bitbox02ConnectAuto: async () => ({
      unlockAndPair: async () => ({
        waitConfirm: async () => buildPairedBitBox(scenarioRegressionUmlautEIP712()),
      }),
    }),
  };
});
```

See [`TESTING.md`](TESTING.md) for the full cookbook and per-layer guidance.

## Audit any BitBox-integrating repo

The `bitbox-audit` CLI scans a repository for known regressions and emits a structured report. Pipe the JSON into `bitbox-audit-explain` for a plain-language narrative (uses Claude if `ANTHROPIC_API_KEY` is set, otherwise prints the ready-to-paste prompt).

```bash
go install github.com/joshuakrueger-dfx/bitbox-testkit/go/cmd/bitbox-audit@latest
go install github.com/joshuakrueger-dfx/bitbox-testkit/go/cmd/bitbox-audit-explain@latest

bitbox-audit --repo /path/to/your/wallet > findings.json
bitbox-audit-explain --input findings.json   # narrative report
```

The audit detects:
- non-ASCII string literals in EIP-712 contexts (quirk E1)
- BLE packet-dedup ordering reversals (quirk P2)
- hard-coded 10-second transport timeouts (quirk A2)

Other quirks have no static signature and surface only through dedicated tests using the language-specific Scenario factories.

## Layout

```
/quirks/SCHEMA.md           # canonical schema for the knowledge base
/go/bitbox/quirks/          # Go module + embedded quirks.json
/ts/src/                    # TypeScript source
/scripts/sync-quirks.sh     # keep ts/src/quirks/quirks.json byte-identical to the Go side
.github/workflows/test.yml  # CI: Go vet/race + TS unit + sync check
```

## Adding a new quirk

1. Add an entry to `/go/bitbox/quirks/quirks.json`.
2. Run `./scripts/sync-quirks.sh` to refresh the TS copy.
3. Add the Go callback in `/go/bitbox/quirks/callbacks.go` (case branch on the new ID).
4. Add the TS callback in `/ts/src/quirks/callbacks.ts`.
5. Add tests in both languages.

The quirks JSON is the single source of truth; both languages just attach callbacks by ID.
