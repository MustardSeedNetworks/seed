# The Seed

> Portable network diagnostic appliance with real-time web UI.

[![CI](https://github.com/krisarmstrong/seed/actions/workflows/ci.yml/badge.svg)](https://github.com/krisarmstrong/seed/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/krisarmstrong/seed?logo=github)](https://github.com/krisarmstrong/seed/releases/latest)
[![CodeQL](https://github.com/krisarmstrong/seed/actions/workflows/codeql.yml/badge.svg)](https://github.com/krisarmstrong/seed/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/krisarmstrong/seed/badge)](https://scorecard.dev/viewer/?uri=github.com/krisarmstrong/seed)
[![Go Reference](https://pkg.go.dev/badge/github.com/krisarmstrong/seed.svg)](https://pkg.go.dev/github.com/krisarmstrong/seed)
[![Go Report Card](https://goreportcard.com/badge/github.com/krisarmstrong/seed)](https://goreportcard.com/report/github.com/krisarmstrong/seed)
[![License: BSL 1.1](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)

The Seed is a network diagnostic appliance from **Mustard Seed Networks**.
Plug it into any network jack and the web UI shows link status, switch
information, DHCP/DNS health, gateway reachability, Wi-Fi survey data, and
security posture in real time. Built to run on a Raspberry Pi or any modern
Linux box.

## Modules

| Module | Purpose | Color |
|--------|---------|-------|
| **Roots** | Path analysis, traceroute, deep connectivity | Amber |
| **Canopy** | Wi-Fi planning, surveys, coverage heat maps | Green |
| **Shell** | Security posture, hardening, vulnerability checks | Orange |
| **Sap** | Live telemetry, monitoring, data flow | Cyan |
| **Harvest** | Reporting, compliance, exports | Gold |

## Features

- **Real-time diagnostics** — live updates over WebSocket
- **Link status** — speed, duplex, advertised capabilities, flap counts
- **Switch discovery** — LLDP / CDP / EDP / FDP / Foundry
- **DHCP analysis** — phase timing breakdown (Discover/Offer/Request/Ack)
- **DNS testing** — forward + reverse lookups with timing
- **Gateway health** — ping, traceroute, latency tracking
- **VLAN detection** — tagged and native VLAN identification
- **Wi-Fi** — signal strength, channel survey, security info (nl80211)
- **Path discovery** — multi-hop topology mapping
- **Health checks** — TCP / UDP / HTTP probes with thresholds
- **Vulnerability scanning** — CISA KEV + CVE feeds
- **Threshold alerts** — configurable green / yellow / red indicators
- **Modern UI** — Tailwind v4 design system, dark/light themes, mobile-responsive
- **i18n-ready** — translated UI namespaces
- **Secure** — HTTPS by default with self-signed cert; password-only after first-run setup

## Quick Start

### Prerequisites

- Linux (Raspberry Pi 4 or any modern x86/arm64 box)
- Go 1.26+
- Node.js 26+
- libpcap-dev

### Hardware notes

| Capability | Recommended adapter |
|------------|---------------------|
| Basic diagnostics | any |
| Wi-Fi survey | nl80211-compatible (Intel AX200/210) |
| Cable diagnostics (TDR) | Intel I350/I210 or Broadcom BCM5719/5720 |

See [HARDWARE.md](HARDWARE.md) for the full compatibility matrix.

The Seed needs raw-socket access for diagnostics. On Linux either:

```bash
# run as root
sudo ./seed

# or grant capabilities once
sudo setcap cap_net_raw,cap_net_admin=+ep ./seed
./seed
```

### Install + run

```bash
git clone https://github.com/krisarmstrong/seed.git
cd seed
make build            # builds frontend + backend in one step
sudo ./seed           # listens on https://localhost:8443
```

Or grab a package from the [releases page](https://github.com/krisarmstrong/seed/releases)
(`.deb`, `.rpm`, macOS `.pkg`, Windows `.zip`) or install via Homebrew:

```bash
brew install krisarmstrong/tap/seed
```

### First run

1. Open `https://<device-ip>:8443` (accept the self-signed cert).
2. Walk the first-run setup wizard to create the admin password — there is
   no shipped default password.

### First-time TLS trust setup (optional)

Seed serves its UI over HTTPS with a self-signed certificate. To eliminate
the browser warning, install that certificate into your OS trust store:

```bash
sudo seed install-ca
```

This adds seed's root certificate to the macOS Keychain, the Linux system
CA bundle (Debian/Ubuntu via `update-ca-certificates`, RHEL/Fedora via
`update-ca-trust`), or the Windows Certificate Store. After the install
command finishes it prints the certificate's SHA-256 fingerprint. Compare
it against what your browser shows ("View certificate → Details →
Fingerprints") and against the value served at `/__version`:

```bash
seed install-ca --print-fingerprint
curl -k https://localhost:8443/__version | jq -r .tlsFingerprint
```

The two values must match.

To remove the certificate from the trust store:

```bash
sudo seed install-ca --uninstall
```

## Configuration

`seed.yaml` (and `SEED_*` env vars) configure the appliance:

```yaml
server:
  port: 8443      # HTTPS
  https: true

interface:
  default: eth0

thresholds:
  dhcp:  { warning: 500ms, critical: 2s }
  dns:   { warning: 100ms, critical: 500ms }
  ping:  { warning:  50ms, critical: 200ms }
```

Common environment overrides:

```bash
SEED_HTTP_PORT=8443
SEED_LOG_LEVEL=info       # debug | info | warn | error
SEED_DB_PATH=/var/lib/seed/data.db
```

## Architecture

```
ui/src/              → React/TypeScript frontend (Vite)
                            ↓ npm run build
internal/api/ui/     → Built assets (embedded via go:embed)
                            ↓
cmd/seed/            → Entry point
internal/
├── api/             → HTTP/WebSocket handlers
├── database/        → SQLite store + migrations
├── network/         → Link, DHCP, DNS, Wi-Fi, cable, path probes
├── config/          → YAML + env loading
├── auth/            → JWT + first-run setup
├── telemetry/       → Metrics, structured logging
├── i18n/locales/    → Translation namespaces
└── version/         → Build metadata (injected via ldflags)
```

The frontend builds **directly into `internal/api/ui/`** and is embedded
via `//go:embed` — no copy step, no runtime dependency on the source tree.

## Build

| Command | Purpose |
|---------|---------|
| `make build` | Full build (frontend + backend) |
| `make test` | Go + frontend unit/integration tests |
| `make test-e2e` | Playwright UI tests |
| `make lint` | golangci-lint + Biome |
| `make security` | govulncheck + gosec + npm audit + gitleaks |
| `make fmt-check` | Format check (Go + TS) |
| `make fmt-all` | Auto-format everything |
| `make packages` | `.deb` + `.rpm` via GoReleaser |
| `make pkg` | macOS `.pkg` |
| `make verify` | Full local CI gate (lint + test + security + build) |

Frontend-only iteration:
```bash
cd ui
npm run dev          # http://localhost:3000 with proxy to backend
npm run lint
npm run test
npm run e2e
```

Verified versions: **Go 1.26.3**, Node.js 26, golangci-lint v2.12.1.
Cross-platform releases (linux/macOS/windows × amd64/arm64) are built by
`release.yml` on tag push and signed with cosign keyless OIDC.

## Container

```bash
docker run --rm --net host --cap-add NET_RAW --cap-add NET_ADMIN \
  ghcr.io/krisarmstrong/seed:latest
```

Multi-arch images (linux/amd64, linux/arm64) built on native runners with
SLSA-3 provenance and Syft-generated SBOM.

## Frontend design system

The UI uses a Tailwind v4 CSS-first theme with semantic tokens:

- [`ui/src/styles/DESIGN_SYSTEM.md`](ui/src/styles/DESIGN_SYSTEM.md) —
  full token reference (colors, typography, spacing, components)
- [`STYLE_GUIDE.md`](STYLE_GUIDE.md) — coding conventions

## Versioning & Releases

Conventional commits drive [release-please](https://github.com/googleapis/release-please).
Tags trigger `release.yml` which builds binaries, packages, container
images, and (when configured) updates the Homebrew tap.

## License

[Business Source License 1.1](LICENSE) — free for non-commercial use;
commercial use requires a license. Converts to Apache-2.0 on the change
date stated in the LICENSE file.

For commercial licensing inquiries: `kris.armstrong@icloud.com`.

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability-disclosure policy.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Related projects

The Seed is the diagnostic appliance. Two sibling tools complete the
Mustard Seed Networks testing toolkit:

- **[stem](https://github.com/krisarmstrong/stem)** — RFC-compliant network performance testing
- **[niac-go](https://github.com/krisarmstrong/niac-go)** — network device simulator
