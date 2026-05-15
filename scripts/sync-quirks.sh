#!/usr/bin/env bash
# Copy the canonical quirks JSON from /go/bitbox/quirks/quirks.json
# (embedded into the Go binary) into /ts/src/quirks/quirks.json (read by
# the TypeScript loader). CI runs `--check` mode to fail on drift.
#
# Usage:
#   scripts/sync-quirks.sh           # copy go → ts
#   scripts/sync-quirks.sh --check   # diff and exit non-zero on drift

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/go/bitbox/quirks/quirks.json"
DST="$ROOT/ts/src/quirks/quirks.json"

if [[ ! -f "$SRC" ]]; then
  echo "sync-quirks: source not found: $SRC" >&2
  exit 1
fi

if [[ "${1:-}" == "--check" ]]; then
  if ! diff -q "$SRC" "$DST" >/dev/null 2>&1; then
    echo "sync-quirks: drift detected between $SRC and $DST" >&2
    diff -u "$SRC" "$DST" || true
    exit 1
  fi
  echo "sync-quirks: ok (files identical)"
  exit 0
fi

cp "$SRC" "$DST"
echo "sync-quirks: copied $SRC -> $DST"
