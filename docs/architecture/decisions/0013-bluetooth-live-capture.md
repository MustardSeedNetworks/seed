# ADR-0013: Bluetooth live-scan capture port

**Status:** Accepted — 2026-06-05 · `Scanner` port + data types defined (`internal/discovery/bluetooth`); per-OS live-scan drivers (CoreBluetooth / BlueZ D-Bus / WinRT) not yet implemented, so live BLE advertisement sweep remains pending.

## Context

Seed already ships a Bluetooth visibility UI — decode tables (manufacturer ID →
company, service UUID → GATT name, appearance → label), a `useBluetoothScan`
hook on the unified jobs spine (`bluetooth-scan` kind), and a card + full-screen
device modal. What it lacks is a real *live* scan: the scanner backend
(`internal/discovery/bluetooth_{darwin,linux,windows}.go`) is uneven —

- **macOS**: `system_profiler` (+ optional `blueutil`) → only paired/connected
  devices; no live BLE advertisement sweep (that needs CoreBluetooth).
- **Linux**: `bluetoothctl` + `hcitool` → paired/connected + some classic/BLE.
- **Windows**: a stub returning a "requires additional setup" error.

The owner wants Bluetooth to match the Wi-Fi ambition: scan **everything in the
air and decode all of it**, on Linux, macOS, and Windows (via a third-party or
built-in adapter, as each platform allows). This is the Bluetooth analogue of
the Wi-Fi capture decision (ADR-0012); it gets its own ADR because Bluetooth is
a distinct radio with its own per-OS capture mechanisms — it is a peer capture
subsystem, not an instance of the Wi-Fi one.

## Decision

Add a **live Bluetooth scan source** behind a small interface, mirroring the
capture-port split that already works for Wi-Fi/pcap (ADR-0012):

- **(a) Live advertisement scan = per-OS source, one interface.** A
  `BluetoothScanner`-style port with platform implementations:
  - **macOS** — CoreBluetooth `CBCentralManager` advertisement scan (live BLE).
  - **Linux** — BlueZ D-Bus `org.bluez` LE scan (`StartDiscovery` +
    `InterfacesAdded`/advertisement properties) for live BLE + classic inquiry.
  - **Windows** — WinRT `BluetoothLEAdvertisementWatcher` (live BLE).
  Each yields raw advertisement records (address, RSSI, manufacturer data,
  service UUIDs, appearance, flags).
- **(b) Decode is shared + already shipped.** The per-OS source feeds the
  existing decode (`internal/api/bluetooth_decode.go` → companyName /
  serviceNames / appearanceLabel) and the existing `bluetooth-scan` job +
  card + modal. No new decode or UI is needed — only a richer data source.
- **Graceful degrade.** Where a live scan is unavailable (no adapter, missing
  driver, insufficient privilege), fall back to the current paired/connected
  enumeration and report reduced fidelity rather than failing — the same
  degrade contract as the Wi-Fi capture port.
- **Capability-gated.** Live-scan availability registers as a capability
  (ADR-0002), consistent with the Wi-Fi engine, so the UI can reflect what the
  current host actually supports.
- **Tiering** follows `LICENSE_STRATEGY.md`: the basic device view stays as-is;
  whether deep live-scan depth is Pro-gated is settled at implementation against
  the locked tier matrix (not invented here).

## Consequences

- Bluetooth reaches the same "live, fully decoded" bar as the Wi-Fi feature,
  reusing the decode + jobs + card/modal that already shipped — the upgrade is
  a data-source swap, not a rewrite.
- The per-OS capture code is isolated behind one interface (CGO/native-binding
  where required), keeping the rest of the stack platform-agnostic — the same
  shape as ADR-0012's enablement-vs-capture split.
- CoreBluetooth/WinRT bindings introduce platform-native dependencies that need
  real hardware per OS to validate (like the Wi-Fi monitor path); the existing
  paired/connected path remains as the degrade tier and the test substitute.
- Scheduled relative to the Wi-Fi engine (same capture pattern); not a blocker
  for the Wi-Fi slices.
