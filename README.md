# bitbox-testkit

Test infrastructure for BitBox02 Flutter plugins: scriptable API fakes, a BLE-transport fake, official Linux simulator integration, and source-level regression guards.

## Layout

```
core/
  simulator/    vendor simulator binary lifecycle (download, launch, teardown)
  scenario/     generic scriptable scenario framework
  transport/    fake transports (BLE peripheral, U2FHID framing helpers)
  guards/       source-level regression checks
  testutil/     deadlock detection, timeouts, t.Helper wrappers

bitbox/
  fake/         in-memory firmware.Communication implementation
  simulator/    convenience wrapper around core/simulator for the BitBox02 sim
  scenarios/    library of pre-built scenarios (Umlaut-reject, BLE-dedup, ...)
```

## Test layers

| Layer                                | Catches                                     | Cost     |
| ------------------------------------ | ------------------------------------------- | -------- |
| `bitbox/fake` (API)                  | App-logic bugs, error paths, gomobile panics| fast     |
| `core/transport/ble` (transport)     | BLE framing, dedup, stale buffer            | fast     |
| `bitbox/simulator` (end-to-end)      | Firmware-level behavior                     | Linux CI |

See [TESTING.md](TESTING.md).
