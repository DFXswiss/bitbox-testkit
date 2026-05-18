# Contributing to bitbox-testkit

The kit's value scales with the size of the quirks knowledge base. If you discover a new BitBox02 firmware constraint — a quirk that bit your code or a customer's — please file it back into the JSON so every consumer picks up coverage on their next install.

## Adding a new quirk

1. **Pick an ID.** Choose the next free slot in the category (e.g. `E11` for an 11th ETH quirk). Categories: `eth`, `btc`, `cardano`, `mnemonic`, `protocol`, `app`.

2. **Edit `/go/bitbox/quirks/quirks.json`.** Add an entry with:

   ```jsonc
   {
     "id": "E11",
     "name": "kebab-case-slug",
     "category": "eth",
     "severity": "critical",                       // hint | warning | critical
     "description": "One-paragraph human explanation.",
     "source": "messages/eth.proto: …  or  CHANGELOG v9.X.Y: …  or  observed in production",
     "firmware": { "min": "9.0.0", "max": "" },     // empty = unbounded
     "match_regex": "(?i)…",                         // optional: test-output classifier
     "detect": [                                     // optional: static-detection rules
       { "kind": "regex", "regex": "…", "reason": "…", "fix_hint": "…" }
     ]
   }
   ```

   See [`quirks/SCHEMA.md`](quirks/SCHEMA.md) for the full schema.

3. **Sync to the TypeScript side:**

   ```bash
   ./scripts/sync-quirks.sh
   ```

   The TS loader reads the synced copy at `/ts/src/quirks/quirks.json`. CI runs `--check` and fails on drift.

4. **(Optional) Add a Scenario factory** if the firmware response shape differs from the generic `ErrInvalidInput101` rejection.

   - `/go/bitbox/scenarios/scenarios.go` — Go side.
   - `/ts/src/scenarios/index.ts` — TS side.
   - Wire into `/go/bitbox/quirks/callbacks.go` and `/ts/src/quirks/callbacks.ts` `attachCallbacks`/`switch q.id` arms so the quirk's `Scenario` field gets your factory instead of the default.

5. **Add tests for the new detection rule** in `/go/cmd/bitbox-audit/audit_test.go`. Include both a positive case (pattern fires correctly) and a negative case (similar code that should NOT fire). Without the negative test, false-positive regressions creep in.

6. **Re-validate against real repos.** Run `bitbox-audit --repo <somewhere-with-bitbox-code>` against at least one consumer that exercises the new quirk's surface. False positives here cost more than missing detection — a noisy audit gets muted by consumers.

7. **Update `CHANGELOG.md`** under the next-release section with a one-line entry.

## Detection rule kinds (cheat sheet)

| Kind                 | When to use                                                                 |
| -------------------- | --------------------------------------------------------------------------- |
| `regex`              | Simple line-level match; no per-file gating needed.                         |
| `regex_in_context`   | Match `regex` only in files whose content satisfies `context_regex`. Suppresses noise where the same surface text means different things in different contexts. |
| `ordered_pair`       | "X must appear before Y." File must contain both `before_regex` and `after_regex`; a finding is emitted only when `after_regex` precedes `before_regex` in byte offset. |
| `missing_pair_within`| "Every X must be followed by Y within N lines." Use for export-with-guard, lock-with-defer-unlock, and other proximity-paired patterns. |

Regex compatibility: Go uses RE2 (no lookahead / lookbehind / backreferences). JS regex is RE2-compatible for the patterns we need. Write regexes that work in both engines; the audit-runner uses Go RE2.

## Adding a Scenario factory

A Scenario returns a configured fake suitable for use in a single test. Two questions to ask:

1. **Does the firmware respond differently from a generic `ErrInvalidInput101`?** If no, reuse the default. Adding identical wrappers just adds noise.

2. **Is there a multi-step flow?** Use a closure that captures state (counter, confirm flag) and returns different responses on subsequent calls. See `scenarioChannelHashEarly` for the canonical example.

## Validating your changes locally

```bash
# Go: full test sweep with race detector
(cd go && go test -race -timeout 60s ./...)

# TS: typecheck + jest
(cd ts && npx tsc --noEmit && npm test)

# JSON sync gate (CI runs this as well)
./scripts/sync-quirks.sh --check

# Audit against your own working tree
(cd go && go run ./cmd/bitbox-audit --repo .. --format markdown)
```

## Releases

Tags follow semver `vMAJOR.MINOR.PATCH`. The TypeScript package and the Go module both pick up the same tag — there's no separate cadence. CHANGELOG.md is updated as part of the change PR (not as a separate "release commit").

### Automatic flow (the normal path)

Releases happen automatically off `main`. Once a PR merges into `develop`, the `Auto Release PR` workflow opens a `Release: develop -> main` PR. When that PR is merged, the `Auto Tag on Merge` workflow runs, looks at every commit between the previous tag and the new `main` HEAD, parses each subject as Conventional Commits, picks the highest bump, and creates **both** tags (`vX.Y.Z` and `go/vX.Y.Z`) plus the matching GitHub Release.

The Go module lives at `/go/`, so Go's submodule-tagging convention requires the dual tag at the same commit. Without the `go/` prefixed tag, consumers hit:

> `module github.com/DFXswiss/bitbox-testkit@vX.Y.Z found, but does not contain package …/go/cmd/bitbox-audit`

#### Commit message → bump table

The auto-tagger reads Conventional Commits 1.0. Use these subjects in your PR commits:

| Subject prefix                           | Bump      | Example                                            |
| ---------------------------------------- | --------- | -------------------------------------------------- |
| `feat!:`, `fix!:`, `<type>!:`            | **MAJOR** | `feat!: drop legacy bitbox-api v0.11 support`      |
| `BREAKING CHANGE:` in commit body        | **MAJOR** | (paired with any subject)                          |
| `feat:`, `feat(scope):`                  | **MINOR** | `feat(simulator): add BTC scenarios`               |
| `fix:`, `perf:`, `refactor:`, `revert:`  | **PATCH** | `fix(audit): suppress doc-comment false positive`  |
| `chore:`, `ci:`, `docs:`, `test:`, `style:`, `build:` | **PATCH** | `ci: cache go modules`              |
| (anything else)                          | **PATCH** + warning | the auto-tagger logs a warning to the CI step |

The aggregator picks the **highest** bump across every commit in the range — one `feat!:` is enough to promote the whole release to a major bump, one `feat:` is enough for a minor.

#### Local preview

Before merging a release PR you can preview the version the auto-tagger will pick:

```bash
go -C go run ./cmd/release-version --base "$(git describe --tags --abbrev=0 --match='v*.*.*')" --report
```

The first stdout line is the next tag; the rest is a per-commit explanation of why each commit voted the way it did.

### Manual release (escape hatch)

If you ever need to ship out-of-band — e.g. an emergency security fix from a hotfix branch — push the tags by hand:

```bash
# Update CHANGELOG.md, /ts/package.json version
git commit -am "Release vX.Y.Z"

# Two tags, one commit. The 'go/' prefix is required by Go's
# submodule resolver — see https://go.dev/ref/mod#vcs-version.
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git tag -a go/vX.Y.Z -m "go/vX.Y.Z: submodule tag matching vX.Y.Z" vX.Y.Z^{}

git push origin main --tags
```

When the auto-tagger next runs, it will see the manual tags in `git tag -l` and pick the bump relative to them — no special-case handling needed.
