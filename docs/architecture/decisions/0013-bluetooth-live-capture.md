# ADR-0013: Bluetooth live-scan capture port

**Status:** Accepted for Linux — 2026-10-16, and for macOS — 2026-10-17, through seed plan row P-E3
(the owner's parity decision of 2026-09-17 put the BLE survey in v1). The Windows live scan stays
Proposed. History: demoted to Proposed 2026-09-14 (owner) as outside the charter with no plan row;
originally Accepted 2026-06-05. The 2026-10-16 amendment below replaces the Linux source this ADR
first named and its degrade rule; the 2026-10-17 amendment names the macOS source.

## Context

Seed already ships a Bluetooth visibility UI — decode tables (manufacturer ID →
company, service UUID → GATT name, appearance → label), a `useBluetoothScan`
hook on the unified jobs spine (`bluetooth-scan` kind), and a card + full-screen
device modal. What it lacks is a real _live_ scan: the scanner backend
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

## Amendment 2026-10-16: the Linux source is the kernel management interface

The Linux source is the kernel's Bluetooth management interface (the HCI
control channel, `internal/discovery/enumerate/bluetooth_mgmt_linux.go`), not
BlueZ D-Bus. `hcitool` and `bluetoothctl` discovery are gone.

- **No daemon.** The management interface is what bluetoothd itself drives.
  Going through D-Bus would make the scan depend on bluetoothd running and on
  its bus policy, which by default admits root and the `bluetooth` group, not
  the `seed` service user. A headless probe often runs no bluetoothd at all.
- **One capability.** The packaged unit already holds `CAP_NET_ADMIN`, which
  is all the management interface needs. No new dependency: the socket comes
  from `golang.org/x/sys/unix`.
- **Raw records.** Each Device Found event carries the address type, the RSSI
  and the advertising or EIR bytes, which a shared parser
  (`bluetooth_ad.go`) decodes. A random LE address gets no OUI vendor; its
  manufacturer ID names the vendor instead.
- **Adapter state.** A controller bluetoothd never touched is powered off with
  LE disabled. A scan turns on what it needs and restores both afterwards. If
  another client is already discovering, the scan listens to that discovery
  instead of starting its own.
- **Degrade, revised.** bluetoothd's cache of devices seen in the past is not a
  scan, so it is no longer the fallback. When bluetoothd is running it adds
  pairing state to the devices heard and adds connected devices, which stop
  advertising. When the management interface is unavailable, the scan fails
  with the cause (no adapter, missing `CAP_NET_ADMIN`) instead of returning a
  partial list as if it were complete.
- **Kept across scans.** Every scan upserts the devices it heard into
  `bluetooth_devices`, keyed by address, so a device keeps its id and the time
  it was first heard across scans and restarts. A scan whose devices cannot be
  stored fails rather than reporting them as new. Devices not heard for the
  inactive-device retention window age out with wired ones.

## Amendment 2026-10-17: the macOS source runs in the Seed Wi-Fi Helper

macOS answers CoreBluetooth only in a signed application bundle the user
granted Bluetooth to, in their login session. The root daemon is refused with
`CBManagerStateUnauthorized`, so it cannot scan in-process (owner decision
2026-10-17, option A).

- **Binding.** The CoreBluetooth binding is `foundation/pkg/corebluetooth`,
  beside `pkg/corewlan`.
- **Helper.** The Seed Wi-Fi Helper, which already holds the Location grant
  for CoreWLAN, also runs the LE scan. The daemon sends a `bleScan` request
  with the duration (at most one minute) over the existing verified socket,
  and the helper replies with what it heard. Its `Info.plist` carries
  `NSBluetoothAlwaysUsageDescription`. Without that key, macOS terminates the
  helper when the scan starts. At login, a helper without a Bluetooth
  decision raises the prompt.
- **Identity.** CoreBluetooth never reveals an LE device's address. A device
  is keyed by its CoreBluetooth identifier, a UUID that is stable on that Mac,
  and its address type is left empty. The manufacturer ID names the vendor.
  A paired LE device that is advertising is listed twice: once by address
  from `system_profiler` and once by identifier. Nothing links the two.
- **Degrade.** As on Linux, a scan that cannot listen fails with the cause.
  That covers no helper connected, a missing grant and a powered-off radio. It
  does not return the paired list as if it were complete. A scan configured
  without LE still returns the paired and connected devices.

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
