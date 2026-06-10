# ADR-0012: Wi-Fi monitor-mode capture port + 802.11 decode pipeline

**Status:** Accepted — 2026-06-05 · capture port + CGO-free 802.11 decode (`internal/wifi/{capture,dot11,visibility}`) + SSID→AP→BSSID airspace hierarchy implemented; opt-in and unreleased (monitor capture activates only when `SEED_WIFI_MONITOR_IFACE` is set, empty airspace otherwise). Full client-tracking + W1–W6 feature surface still pending.

## Context

Today's Wi-Fi is OS-level scanning only (`internal/wifi` shells to `airport`/`nmcli`/`iw`): a
flat neighbor-AP list of SSID/BSSID/signal/channel/security. There is no frame capture, no
802.11 decode, and no client tracking (`ClientCount` is hardcoded `0`). The owner wants Wi-Fi to
be **as good as or better than wired discovery + topology**: channel-hop every channel at ~110ms
dwell, capture **802.11 management frames**, fully decode them, and present a **cross-referenced
SSID → AP → BSSID → client hierarchy** with capability/IE decode (HT/VHT/HE/EHT, 802.11k RRM
neighbor reports, 802.11r FT, country/regulatory), client associations down to the BSSID, plus
co-channel / adjacent-channel interference analysis. This is also the data source for the
anomaly engine (ADR-0011).

Constraints: raw 802.11 capture needs a **monitor-mode-capable adapter + root/CAP_NET_RAW +
libpcap (CGO)**. It is reliable on **Linux**; on **Windows/macOS it is best-effort via a
third-party USB adapter** (the built-in macOS card cannot — Apple blocks raw frames). The
existing capture port (`internal/capture`, libpcap behind a CGO build tag, null stub under
CGO=0) already proves the seam used by wired discovery (LLDP/CDP/EDP), DHCP, and VLAN sampling.

## Decision

Extend the existing `internal/capture` port with a **Wi-Fi monitor-mode source**, splitting two
concerns so the OS-specific surface stays tiny:

- **(a) Capture + decode = OS-agnostic.** Capture runs over any monitor-capable **radiotap**
  interface via libpcap (CGO). Decode is **CGO-free** — gopacket's `layers.RadioTap` and
  `layers.Dot11*` are pure Go — so the decoder, the SSID→AP→BSSID→client model, the
  co-/adjacent-channel interference analysis, and the anomaly rules are all pure-Go and **fully
  unit-tested off-target** (macOS dev) against captured-frame fixtures. Same code path on
  Linux/Windows/macOS.
- **(b) Monitor-mode ENABLEMENT = per-OS best-effort helper, isolated.** Linux nl80211/`iw` is
  first-class (auto-enable + the active-diagnostic "+1s" — station dump, association probe,
  RTS/CTS test — registered as capabilities per ADR-0002/0011). Windows (Npcap) and macOS are
  best-effort. A **"bring-your-own monitor interface"** escape hatch lets a user pre-set the
  adapter to monitor mode and we capture it by name — so the full feature works wherever a
  radiotap interface exists, even where auto-enable doesn't.
- **Channel hopper** — async loop, ~110ms dwell across the regulatory-permitted 2.4/5/6 GHz
  channels, aggregating decoded frames per channel; feeds the model continuously.
- **Domain model** — beacons/probe-responses build the SSID→AP→BSSID tree (capabilities/IEs,
  RRM/FT, country); association/data frames populate clients keyed by MAC, cross-referenced down
  to the BSSID. Channel occupancy + retry/airtime feed co-channel/adjacent-channel detection.
- **Runs on the jobs spine (ADR-0005)** as a scan job kind (aligns with the existing
  `wifi-discovery-scan` kind; the monitor-capture scan extends it or lands as `wifi-capture-scan`),
  with SSE progress and cancellation.
- **Graceful degrade.** Under CGO=0, no monitor-capable interface, or an unsupporting platform,
  the feature falls back to today's OS-level neighbor-AP scan (the existing path becomes the
  degrade tier) and reports reduced fidelity rather than failing.
- **Pro-gated** (`wifi_management_capture` / `wifi_association_forensics`, already reserved in the
  license validator); Free/Starter keep the OS-scan + channel-utilization indicator.
- **Build-tagged** real vs null capture mirrors the established `wire_capture_real.go` /
  `wire_capture_null.go` split; new DTOs are camelCase (ADR-0010) via code-first schema (ADR-0008).

## Consequences

- Wi-Fi reaches wired-discovery-grade richness (a real, frame-fed, cross-referenced model) and
  becomes the first rule source for the anomaly engine (ADR-0011).
- Keeping decode/model/anomaly CGO-free means the bulk of the subsystem is fast, deterministic,
  and unit-tested on any dev machine; only the live capture handle needs libpcap.
- The only OS-specific code is monitor-mode *enablement*, isolated behind a small helper with a
  BYO-interface fallback — so Windows/macOS third-party adapters use the identical engine, with
  success gated on the user's driver, not on our code.
- **Validation needs real monitor-mode hardware** (Linux box / CI), exactly like the wired pcap
  path — a known-good split, not a new risk class. macOS/Windows third-party monitor mode stays
  best-effort and is not the validated reference.
- The legacy OS-scan path is retained (now the graceful-degrade tier), not deleted.
- A new Pro capability + scan job kind + schema DTOs are added; the channel hopper introduces
  timing/coverage considerations (beacons ~100ms vs 110ms dwell) to validate on hardware.
