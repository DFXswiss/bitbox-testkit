# quirks.json schema

The single source of truth for every documented BitBox02 firmware constraint.

**Canonical file:** `/go/bitbox/quirks/quirks.json` (must live inside the Go module so it can be embedded via `//go:embed`).

The Go loader (`/go/bitbox/quirks/loader.go`) embeds and parses this file at init time, attaching language-specific Scenario/Detect callbacks by quirk ID. The TypeScript loader (`/ts/src/quirks/loader.ts`) reads a synchronised copy at `/ts/src/quirks/quirks.json` — a parity test in both languages ensures the copy stays byte-identical.

## Top-level fields

| Field | Type | Notes |
|-------|------|-------|
| `schema_version` | string | semantic version of the schema |
| `description` | string | short summary, no semantic meaning |
| `quirks` | array | the entries — see below |

## Quirk entry fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | string | yes | Stable identifier. Prefix matches category (E*=ETH, B*=BTC, C*=Cardano, M*=Mnemonic, P*=Protocol, A*=App). Must be unique. |
| `name` | string | yes | kebab-case slug, suitable for log lines. |
| `category` | string | yes | One of `eth`, `btc`, `cardano`, `mnemonic`, `protocol`, `app`. |
| `severity` | string | yes | One of `hint`, `warning`, `critical`. |
| `description` | string | yes | One-paragraph human explanation. |
| `source` | string | yes | Citation (proto file, CHANGELOG version, observed-in-production note). |
| `firmware.min` | string | yes | Inclusive lower bound (e.g. `"9.15.0"`). Empty means no minimum. |
| `firmware.max` | string | yes | Exclusive upper bound (e.g. `"9.15.0"`). Empty means no maximum. |
| `match_regex` | string | no | Regex matching test failure output that could indicate this quirk. Used by audit runners. |

## Adding a new quirk

1. Pick the next free ID for the category.
2. Add entry to `quirks` array.
3. Add language-specific Scenario callback in `/go/bitbox/quirks/<category>.go` and `/ts/src/quirks/<category>.ts`.
4. (Optional) Add Detect callback if a static source pattern can flag the bug class.
5. Bump `schema_version` if you change the field shape — not when you just add entries.

## Severity guide

- **hint** — useful to know, no immediate hazard
- **warning** — wrong but recoverable, may surface as user-visible error
- **critical** — silent data loss, crash, or security regression
