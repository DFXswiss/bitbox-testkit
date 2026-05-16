# Security model — bitbox-testkit

This document captures the threat model the testkit defends against, the boundaries it cannot defend, and the responsibilities that fall on the consumer wallet.

## Scope

The kit operates between the wallet's UI and the BitBox firmware:

```
  ┌─────────────┐    ┌──────────────┐    ┌────────┐
  │  Wallet UI  │ →  │  bitbox-api  │ →  │ Device │
  │  (consumer) │    │  (Rust/WASM) │    │        │
  └─────────────┘    └──────────────┘    └────────┘
         ▲                  ▲
         │                  │
         │           testkit's API fake
         │           (firmware.Communication / PairedBitBox)
         │
   testkit's source guards + audit-runner
   (compile-time + lint-time enforcement)
```

It does NOT defend the device itself, the firmware, the user's seed, or anything below the wire protocol. Those are upstream responsibilities. It DOES help the wallet detect known classes of bug-pattern that would otherwise reach the user as silent corruption or unsafe behaviour.

## Threats the testkit is designed to address

### T1 — Address-display bypass (CRIT-3 / quirk A4)

A malicious or compromised intermediate layer (the WebView, a substituted bitbox-api WASM, a hostile dependency) returns an attacker-controlled address. With `displayOnDevice: false`, the user has no second channel to verify they are receiving funds to their own keypair.

**Mitigation:**
- Static quirk A4 fires on any source file that contains `displayOnDevice: false` in a BitBox context.
- The recommended consumer API forces every address-derivation call to take an opts object; the implementation defaults `displayOnDevice` to true.

### T2 — Wire-error misclassification

A wallet that lumps "user rejected on device" and "firmware reject" and "transport failure" into a single "Operation failed" UX cannot guide the user to the right next step. A user rejection looks like a bug; a firmware reject looks like a user mistake.

**Mitigation:**
- Typed error classes shipped (`HwUserAbortError`, `HwFirmwareRejectError`, `HwBridgeTimeoutError`, `HwTransportFailureError`, …) with stable `kind` discriminants.
- Helpers `isUserAbort()` and `parseFirmwareError()` classify raw exceptions consistently.
- UI is expected to branch on these classes; the testkit's static checks ensure the imports are present.

### T3 — Address-derivation chain confusion (CRIT-4)

A wallet that hardcodes one chain ID would display "Ethereum mainnet" on the device while the user thought they were authorising a Polygon transaction.

**Mitigation:**
- The recommended consumer API takes `chainId: bigint` as a required field of every address / sign-request opts object.
- Existing tests in the dfx-wallet integration assert chainId reaches the bridge unchanged.

### T4 — Stale-session attack on the WebView bridge

A WebView that survives the React-Native side reloading, or an injected message from a hostile content load, could interleave with the live bridge session.

**Mitigation:**
- 128-bit cryptographic session nonce, regenerated on every `setWebView()`.
- Every cross-bridge message carries the nonce; mismatches are dropped.
- The bridge HTML's CSP forbids remote origins, inline imports of foreign WASM, and base navigation.
- The hidden WebView runs with `originWhitelist: ['about:*']`, `cacheEnabled: false`, and `incognito: true`.

### T5 — Supply-chain substitution of bitbox-api WASM

A compromised npm registry or a malicious update to `bitbox-api` could ship key-extracting WASM.

**Mitigation:**
- `scripts/setup-bitbox-wasm.sh` downloads the WASM from the npm tarball and verifies SHA-256 against pinned hashes in the script before staging.
- `--check` mode runs in CI; a mismatched hash fails the build.
- The bridge HTML loads only relative-path assets from the bundled directory; remote URLs are forbidden by CSP.

### T6 — BLE packet-dedup regression (quirk P2)

Plugin-side BLE deduplication that drops legitimate retransmits aborts multi-page signing flows (quirk P2). This was observed in production in adjacent codebases.

**Mitigation:**
- The testkit ships an `ordered_pair` static detector flagging `seenPackets.removeAll(...)` before `seenPackets.contains(...)`.
- Test scenarios drive the bug class via `scenarioDeviceDisconnect` and similar.

### T7 — User-confirm timeout bottleneck (quirk A2)

A hard-coded 10-second timeout in transport code blocks legitimate long user-confirm flows. The user reads a many-page transaction on the device; the wallet aborts before they finish.

**Mitigation:**
- Quirk A2's regex detection now requires BitBox context to fire (won't falsely flag UI animations).
- `WasmBridge.call` accepts a per-call `timeoutMs` so signing flows can extend to 120s.

### T8 — Secret leakage in logs

A debug log that includes a signature byte array, a private key, an extended key, or even a recipient address creates an exfiltration channel for whoever has access to the log aggregator (often more people than the wallet trusts).

**Mitigation:**
- `services/log.ts` ships a redacting logger. Field names like `seed`, `password`, `passphrase`, `signature`, `keypath`, `recipient` are stripped before emission.
- Long hex strings, EVM addresses, bech32 addresses, and extended keys are pattern-matched and redacted in free-form text.
- Byte arrays >= 16 bytes are reported as `[N-byte buffer]` not contents.
- Tests assert each of the above invariants.

## Threats the testkit cannot address

### T9 — Compromised host OS

If the host phone is rooted or the JavaScript runtime is patched, all bets are off. This is the same envelope every wallet operates in.

### T10 — Compromised BitBox firmware

If the device firmware is malicious (counterfeit or supply-chain-compromised on the device side), the testkit's mocks against bitbox-api cannot tell. The wallet's defence is to verify the device's attestation certificate at pairing — that is in scope for bitbox-api itself, not the testkit.

### T11 — Phishing / social engineering

If the user is convinced to confirm a malicious transaction on-device, the wallet cannot prevent it. The testkit can help by ensuring the wallet's UX surfaces complete, accurate information to the user before they confirm — but the on-device confirmation remains the user's responsibility.

### T12 — Side channels in the device

Power-analysis attacks, timing leaks, fault injection — all out of scope. The BitBox firmware is designed against these; the testkit has nothing to add.

## Consumer responsibilities

A wallet using this testkit MUST:

1. Set `displayOnDevice: true` for every receive-address generation that will be shared with a counterparty. The opt-out form is for transient internal derivation only.
2. Pass the actual `chainId` for every EVM operation. Never hardcode.
3. Map `HwUserAbortError` to dedicated user-facing copy ("You rejected on device" — not "Connection failed").
4. Run `scripts/setup-bitbox-wasm.sh --check` in CI so a tampered WASM blob fails the build.
5. Run `bitbox-audit --repo .` in CI and treat any critical finding as a merge blocker.
6. Use `HwLogger` (or an equivalent redaction layer) for all hardware-wallet log lines.
7. Validate firmware version before signing operations; refuse known-incompatible combinations using the testkit's `Quirk.firmware` range information.

## Reporting issues

Security issues against the testkit itself: open a private security advisory on the testkit repository, do not file a public issue. Issues against bitbox-api or BitBox firmware: report to Shift Crypto via the channels documented at https://bitbox.swiss/.
