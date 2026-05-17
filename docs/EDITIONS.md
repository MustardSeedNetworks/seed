# Editions: Lite vs Pro Packaging

**Product:** Seed
**Status:** Plan of record (#251)
**Owner:** Mustard Seed Networks
**Last updated:** 2026-05-17

Companion to [DISTRIBUTION.md](DISTRIBUTION.md) (commercial channels and license tiers).
This document defines the **two hardware/software profiles** Seed ships under, the **licensing hooks** to build now while keeping behaviour permissive, and the **distribution mechanics** for each.

---

## 1. Hardware Profiles

| Profile | Target Hardware | Form Factor | Power | Audience |
|---------|-----------------|-------------|-------|----------|
| **Lite (portable)** | Raspberry Pi 4/5 or comparable SBC; 1 GbE native; optional USB 2.5G NIC (not guaranteed); on-board Wi-Fi for AP-mode onboarding | Small enclosure, battery-pack friendly, status LED + reset button | 5W typical | Field techs, walk-around audits, MSP truck rolls |
| **Pro (stationary)** | Fanless x86 mini PC (Intel N100 / N5105 class), dual 2.5 GbE, 8–16 GB RAM | Desk/rack 1U-half | 12W typical | Permanent rack install, continuous monitoring, fleet pilots |

### Why two profiles
- A single SKU forces a compromise: Pi-class hardware can't drive 2.5G iperf, and an x86 mini PC isn't battery-friendly or pocketable.
- Differentiating up front lets us build/ship/license each profile cleanly without per-feature run-time checks scattered across the codebase.

---

## 2. Software Differentiation

| Capability | Lite | Pro |
|------------|:----:|:---:|
| Core diagnostics (link, DHCP, DNS, gateway, basic perf) | ✓ | ✓ |
| Wi-Fi survey | ✓ | ✓ |
| iperf3 client + bundled server | ✓ | ✓ (heavier presets) |
| Headless onboarding (AP mode, captive portal) | ✓ | — |
| Continuous monitoring + alerting | — | ✓ |
| SNMP polling at scale (>50 devices) | — | ✓ |
| Auto-update channel | — | ✓ |
| Signed installer artifacts (.deb/.rpm/.pkg/Docker) | community-grade | signed |
| Pi OS image (write-to-SD) | ✓ | — |

A capability marked `—` does **not** mean compiled out — it means **license-gated**. Run-time checks live in one place: `license.IsPro()`.

### Implementation rule
Anywhere we would write `if cfg.Foo.Enabled` for a Pro-only knob, write `if license.IsPro() && cfg.Foo.Enabled`. The flag stays user-controllable on every edition; the gate is the license check.

---

## 3. Licensing Hooks (build now, leave permissive)

Tracked under epic #245. This section is the **wiring plan** so that turning licensing on later is a config flip, not a refactor.

### 3.1 Config surface

```yaml
license:
  key: ""                 # paste-in key, empty = Community
  status: ""              # "valid"|"expired"|"invalid"|"" (filled by validator)
  edition: "community"    # "community"|"lite"|"pro"
  expires: ""             # RFC3339; "" = never
  lastCheck: ""           # RFC3339 of last validate() attempt
```

Add to `internal/config/config_types.go` (or appropriate split file) and to `internal/config/schema.json` `$defs`. Default values keep the app fully usable.

### 3.2 Code surface

Create `internal/license` package with the following minimum surface:

```go
package license

type Edition string

const (
    EditionCommunity Edition = "community"
    EditionLite      Edition = "lite"
    EditionPro       Edition = "pro"
)

// Validate checks a license key against the local fingerprint. Network
// validation is best-effort; offline runs accept a cached "valid" up to a
// 7-day grace window.
func Validate(key, fingerprint string) (Status, error) { ... }

// IsPro returns true when the current edition is Pro. Returns false in
// Community mode AND when no license has been entered — Community is the
// safe default.
func IsPro() bool { ... }

// Fingerprint returns hash(machine-id + primary MAC). Only the hash is
// persisted; raw machine identifiers stay on the device.
func Fingerprint() string { ... }
```

### 3.3 API surface

`GET /api/license` — returns redacted status (`edition`, `expires`, `status`, `lastCheck`). Never echoes the key back.
`PUT /api/license` — accepts `{ key: "..." }`. Validates, stores, returns the redacted status.

Wire into existing settings drawer under a new "License" section.

### 3.4 Behaviour rules

- **No hard-fail when unlicensed.** Community runs every shipped feature except the Pro-gated ones.
- **Startup log line.** `INFO license edition=community status=` — operators must be able to tell at a glance.
- **Grace window.** Cached `valid` status from network validation is honoured for 7 days when the validator endpoint is unreachable. After that, the edition drops back to Community.
- **Fingerprint is one-way.** Only the hash is sent to the validator endpoint.

---

## 4. Distribution Mechanics

### 4.1 Lite

- **Primary delivery:** custom Pi OS image (`.img`/`.img.xz`). Hardware-locked at flash time.
- **Secondary delivery:** `.deb` for Pi running stock Raspberry Pi OS.
- **Iperf3:** bundled in the image; `PATH` includes `/usr/local/bin` and `./bin/iperf3`.
- **Onboarding:** boots into AP mode (`seed-setup` SSID) when no Ethernet link is detected; captive portal serves the Setup Wizard.

### 4.2 Pro

- **Primary delivery:** signed `.deb` (Debian/Ubuntu), `.rpm` (RHEL/Fedora), `.pkg` (macOS for dev), Docker image.
- **Signing:** binaries and packages signed with Mustard Seed Networks code-signing cert (deferred until the cert is in place).
- **Auto-update:** opt-in channel pulls from the customer portal; verifies signature before install.
- **Service PATH:** systemd unit prepends `/usr/local/bin` so bundled iperf3 wins over a stale system copy.

### 4.3 Shared

- **Versioning:** single source of truth = git tags. `make build` embeds via `ldflags`.
- **`__version` endpoint:** returns `version`, `commit`, `buildTime`, `uiBuildHash`, and `edition` (#251). The edition string is **observable but not enforceable** — clients must not trust it for gating.
- **Artifacts:** every package filename includes the version.

---

## 5. UX Considerations

- **Minimum supported width:** 480px (Lite portable case — phone landscape). Comfortable target: 768px+.
- **Headless first:** the Setup Wizard, Survey, and Settings drawer must work without a mouse — Lite users typically reach the box from a phone/tablet over the AP-mode SSID.
- **Settings drawer focus bug:** tracked separately under #230; do not block packaging on it.

---

## 6. Open Questions (do not block the plan)

| Question | Owner | Needed by |
|----------|-------|-----------|
| Exact Community vs Pro feature split | Product | First Pro paid pilot |
| Pricing bands for Lite vs Pro hardware bundles | Product | Same |
| Self-serve license rebind flow (when a board dies) | Eng + Support | First RMA |
| Whether the Pi OS image ships before or after first Pro release | Eng | Lite pilot |

---

## 7. Acceptance Criteria Mapping

From issue #251:

- [x] Two distinct hardware profiles defined — §1
- [x] Software features aligned with profiles — §2 + `license.IsPro()` gating rule
- [x] Licensing hooks planned (permissive, no hard-fail) — §3
- [x] Packaging and distribution methods planned for both editions — §4
- [x] UX considerations documented for different form factors — §5
