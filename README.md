# LuminetIQ

> Portable Network Diagnostic Tool with Real-Time Web UI

[![CI](https://github.com/krisarmstrong/netscope/actions/workflows/ci.yml/badge.svg)](https://github.com/krisarmstrong/netscope/actions/workflows/ci.yml)
[![License: BSL 1.1](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)

LuminetIQ is a professional-grade network diagnostic appliance designed for network technicians and
engineers. Plug it into any network jack and instantly see link status, switch information, DHCP
details, DNS health, and gateway connectivity through a modern web interface.

## Features

- **Real-time diagnostics** - Live updates via WebSocket
- **Link status** - Speed, duplex, advertised capabilities
- **Switch discovery** - LLDP/CDP/EDP/FDP support
- **DHCP analysis** - Phase timing breakdown (Discover/Offer/Request/Ack)
- **DNS testing** - Forward/reverse lookups with timing
- **Gateway health** - Ping tests with latency tracking
- **VLAN detection** - Tagged and native VLAN identification
- **Wi-Fi support** - Signal strength, channel, security info
- **Threshold alerts** - Configurable green/yellow/red indicators
- **Modern UI** - Dark/light mode, mobile-responsive
- **Secure** - HTTPS by default, authentication required

## Screenshots

_Coming soon_

## Quick Start

### Prerequisites

- Raspberry Pi 4 (or any Linux system)
- Go 1.22+
- Node.js 24+ (LTS)
- libpcap-dev

**Hardware Considerations:**

- **Wi-Fi Diagnostics:** Requires nl80211-compatible adapter (Intel AX200/210 recommended)
- **Cable Diagnostics (TDR):** Requires Intel I350/I210 or Broadcom BCM5719/5720
- **Basic Diagnostics:** Works with any network adapter

See **[HARDWARE.md](HARDWARE.md)** for detailed compatibility guide and recommendations.

**Note:** LuminetIQ requires raw ICMP socket access for ping functionality. On Linux, this requires
either:

- Running as root (`sudo ./luminetiq`), or
- Setting capabilities: `sudo setcap cap_net_raw=+ep ./luminetiq`

### Installation

```bash
# Clone the repository
git clone https://github.com/krisarmstrong/netscope.git
cd netscope

# Build backend
make build

# Build frontend
cd web && npm ci && npm run build && cd ..

# Run
sudo ./luminetiq
```

### Access

Open `https://<device-ip>:8443` in your browser.

Default credentials:

- Username: `admin`
- Password: `luminetiq`

**Change these on first login!**

## Configuration

Configuration is stored in `luminetiq.yaml`. Default settings:

```yaml
server:
  port: 8443 # Default HTTPS port
  https: true # HTTPS enabled by default

interface:
  default: eth0

thresholds:
  dhcp:
    warning: 500ms
    critical: 2s
  dns:
    warning: 100ms
    critical: 500ms
  ping:
    warning: 50ms
    critical: 200ms
```

## Development

```bash
# Run backend in development mode (requires privileges for ICMP)
sudo go run cmd/netscope/main.go

# Or build and set capabilities (preferred for repeated runs)
go build -o luminetiq ./cmd/netscope
sudo setcap cap_net_raw=+ep ./luminetiq
./luminetiq

# Run frontend in development mode
cd web && npm run dev
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

### Frontend Development

The frontend uses a centralized design system with semantic tokens. See:

- **[web/THEMING.md](web/THEMING.md)** - Color tokens, typography, spacing, and component patterns
- **[STYLE_GUIDE.md](STYLE_GUIDE.md)** - Naming conventions and code standards

## Architecture

```
┌─────────────────────────────────────────┐
│              Web Browser                │
│         (React + WebSocket)             │
└─────────────────┬───────────────────────┘
                  │ HTTPS/WSS
┌─────────────────▼───────────────────────┐
│             Go Backend                  │
│  ┌─────────┬─────────┬─────────┐       │
│  │   API   │   WS    │  Auth   │       │
│  └────┬────┴────┬────┴────┬────┘       │
│       │         │         │             │
│  ┌────▼────┬────▼────┬────▼────┐       │
│  │ Network │ Capture │  DHCP   │       │
│  │  Mgmt   │ (pcap)  │ Client  │       │
│  └─────────┴─────────┴─────────┘       │
└─────────────────────────────────────────┘
```

## Roadmap

- [x] Project setup
- [ ] v0.1.0 - Foundation (CI/CD, scaffold)
- [ ] v0.2.0 - Core infrastructure (WebSocket, auth)
- [ ] v0.3.0 - Link & Switch cards
- [ ] v0.4.0 - DHCP & DNS cards
- [ ] v0.5.0 - Gateway & VLAN cards
- [ ] v0.6.0 - Wi-Fi & Cable cards
- [ ] v0.7.0 - Settings & Polish
- [ ] v1.0.0 - Production release

### Future (v2+)

- Historical sparklines
- iperf performance testing
- PDF export
- Alerting (webhooks)

## License

This project is licensed under the [Business Source License 1.1](LICENSE).

- **Free for non-commercial use**
- **Commercial use requires a license**
- **Converts to Apache 2.0 on 2029-12-01**

For commercial licensing inquiries, contact: [your-email]

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting PRs.

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.
