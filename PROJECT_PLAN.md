# NetScope - Project Plan

> Portable Network Diagnostic Tool with Real-Time Web UI

---

## Overview

NetScope is a professional-grade network diagnostic appliance designed for network technicians and engineers. It provides real-time visibility into network connectivity through a modern card-based web interface.

### Target Platform

- **Primary**: Raspberry Pi (ARM64)
- **Future**: x86_64 Linux, potentially other embedded platforms

### Design Philosophy

- Modern, clean UI (WiFi Vigilante color scheme)
- Real-time updates via WebSocket
- Threshold-based warnings (green/yellow/red indicators)
- Mobile-responsive for field use
- HTTPS by default with authentication

---

## Color Scheme (WiFi Vigilante Theme)

### Dark Mode (Default)

| Token            | Value     | Usage                  |
| ---------------- | --------- | ---------------------- |
| `brand-primary`  | `#60a5fa` | Primary actions, links |
| `brand-accent`   | `#3b82f6` | Accent elements        |
| `surface-base`   | `#0f172a` | Background             |
| `surface-raised` | `#1e293b` | Cards                  |
| `surface-border` | `#334155` | Borders                |
| `text-primary`   | `#f1f5f9` | Primary text           |
| `text-muted`     | `#a1b4c9` | Secondary text         |
| `status-success` | `#34d399` | Green indicators       |
| `status-warning` | `#fbbf24` | Yellow indicators      |
| `status-error`   | `#f87171` | Red indicators         |
| `status-info`    | `#60a5fa` | Info indicators        |

### Light Mode

| Token            | Value     | Usage                  |
| ---------------- | --------- | ---------------------- |
| `brand-primary`  | `#2563eb` | Primary actions, links |
| `surface-base`   | `#ffffff` | Background             |
| `surface-raised` | `#f8fafc` | Cards                  |
| `status-success` | `#10b981` | Green indicators       |
| `status-warning` | `#f59e0b` | Yellow indicators      |
| `status-error`   | `#ef4444` | Red indicators         |

---

## Architecture

### Tech Stack

| Layer              | Technology                    | Rationale                                                     |
| ------------------ | ----------------------------- | ------------------------------------------------------------- |
| **Backend**        | Go 1.22+                      | Low-level network access, single binary, cross-compile to ARM |
| **Frontend**       | React 18+ / Vite              | Modern, component-based, matches existing patterns            |
| **Styling**        | Tailwind CSS 4.x              | Utility-first, consistent with krisarmstrong-web              |
| **Real-time**      | WebSocket (gorilla/websocket) | Live card updates                                             |
| **Packet Capture** | gopacket (libpcap)            | Raw frames for LLDP/CDP/EDP/FDP                               |
| **Storage**        | SQLite (v2+)                  | Lightweight, portable, for historical data                    |
| **Auth**           | JWT + bcrypt                  | Simple, stateless authentication                              |
| **HTTPS**          | Built-in TLS                  | Self-signed default, custom cert upload                       |

### Project Structure

```
netscope/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              # Test, lint, build
│   │   ├── release.yml         # Semantic release
│   │   └── codeql.yml          # Security scanning
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── feature_request.md
│   │   └── task.md
│   ├── CODEOWNERS
│   └── dependabot.yml
├── cmd/
│   └── netscope/
│       └── main.go             # Entry point
├── internal/
│   ├── api/                    # HTTP/WebSocket handlers
│   ├── capture/                # Packet capture (LLDP/CDP/etc)
│   ├── network/                # Network interface management
│   ├── dhcp/                   # DHCP client with timing
│   ├── dns/                    # DNS testing
│   ├── config/                 # Configuration management
│   └── auth/                   # Authentication
├── web/                        # React frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── cards/          # Diagnostic cards
│   │   │   ├── settings/       # Settings drawer
│   │   │   └── ui/             # Shared UI components
│   │   ├── hooks/              # Custom React hooks
│   │   ├── lib/                # Utilities
│   │   ├── types/              # TypeScript types
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
├── configs/                    # Default configuration files
├── scripts/                    # Build/deploy scripts
├── .husky/                     # Git hooks
├── .gitignore
├── .goreleaser.yml            # Cross-platform releases
├── CHANGELOG.md
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── PROJECT_PLAN.md            # This document
├── README.md
├── SECURITY.md
├── commitlint.config.js
├── go.mod
├── go.sum
└── package.json               # Root for frontend tooling
```

---

## v1 Features (MVP)

### Cards

| #   | Card               | Description                                                      | Status Indicators                    |
| --- | ------------------ | ---------------------------------------------------------------- | ------------------------------------ |
| 1   | **Link**           | Physical link state, negotiated speed, duplex, advertised speeds | 🟢 Up / 🔴 Down                      |
| 2   | **Cable Test**     | TDR diagnostics (length, faults) if NIC supports                 | 🟢 OK / 🟡 Warning / 🔴 Fault        |
| 3   | **VLAN**           | Detected VLAN tags, native VLAN, voice VLAN                      | 🟢 Tagged / 🟡 Native only           |
| 4   | **Nearest Switch** | LLDP/CDP/EDP/FDP - switch name, port, mgmt IP, description       | 🟢 Found / 🟡 Partial / 🔴 No frames |
| 5   | **Wi-Fi**          | SSID, BSSID, signal strength, channel, security type             | 🟢/🟡/🔴 by signal                   |
| 6   | **DHCP**           | IP, subnet, server, lease, phase timing (DORA)                   | Threshold-based                      |
| 7   | **DNS**            | Forward/reverse lookup, response time                            | Threshold-based                      |
| 8   | **Gateway**        | Gateway IP, 3x ping results, latency                             | Threshold-based                      |

### DHCP Timing Breakdown

Track each phase of DHCP transaction:

- Discover → Offer (time)
- Offer → Request (time)
- Request → Ack (time)
- Total transaction time

### DNS Timing

- Forward lookup time
- Reverse lookup time
- Per-query response time

### Global Settings (Gear Icon → Drawer)

- Interface selection (eth0, wlan0, etc.)
- VLAN tagging (802.1Q tag ID)
- IP Mode: Static vs DHCP
- Static IP configuration (IP, subnet, gateway, DNS)
- Threshold configuration (per-card)
- Authentication credentials
- HTTPS certificate management (self-signed / upload custom)
- Dark/Light mode toggle

### Threshold Defaults

| Metric         | 🟢 Green  | 🟡 Yellow      | 🔴 Red         |
| -------------- | --------- | -------------- | -------------- |
| DHCP Total     | < 500ms   | 500ms - 2s     | > 2s / timeout |
| DHCP per-phase | < 200ms   | 200ms - 1s     | > 1s           |
| DNS query      | < 100ms   | 100ms - 500ms  | > 500ms / fail |
| Gateway ping   | < 50ms    | 50ms - 200ms   | > 200ms / loss |
| Wi-Fi signal   | > -50 dBm | -50 to -70 dBm | < -70 dBm      |

### Security

- HTTPS by default (self-signed certificate)
- Default credentials with first-login change prompt
- JWT-based session management
- Rate limiting on auth endpoints

### Export

- JSON export of current diagnostic state
- Include all card data + timestamps

---

## v2+ Roadmap (Future)

| Feature                   | Description                                       | Priority |
| ------------------------- | ------------------------------------------------- | -------- |
| **Historical sparklines** | Mini graphs on cards showing last N readings      | High     |
| **Performance testing**   | iperf3 integration for throughput tests           | High     |
| **Data logging**          | SQLite time-series storage for historical queries | High     |
| **PDF export**            | Formatted diagnostic reports                      | Medium   |
| **Alerting**              | Webhook/email on threshold breach                 | Medium   |
| **Multi-interface**       | Monitor multiple ports simultaneously             | Medium   |
| **Packet capture**        | Lightweight tcpdump integration                   | Low      |
| **PoE detection**         | Show PoE class/power if available                 | Low      |
| **802.1X status**         | EAP authentication state                          | Low      |
| **Custom tests**          | User-defined ping/DNS targets as cards            | Low      |
| **Remote access**         | Cloud tunnel for remote diagnostics               | Low      |

---

## Discovery Protocol Support

### LLDP (Link Layer Discovery Protocol)

- IEEE 802.1AB standard
- Most common, vendor-neutral
- Supported by most enterprise switches

### CDP (Cisco Discovery Protocol)

- Cisco proprietary
- Widely deployed in Cisco environments

### EDP (Extreme Discovery Protocol)

- Extreme Networks proprietary

### FDP (Foundry Discovery Protocol)

- Brocade/Foundry proprietary

### Implementation Notes

- Listen on raw socket (AF_PACKET / BPF)
- Multicast addresses:
  - LLDP: `01:80:c2:00:00:0e`
  - CDP: `01:00:0c:cc:cc:cc`
- Frame parsing with gopacket
- Timeout: configurable (default 30s)
- Start listening immediately on link up

---

## UI Wireframe

```
┌──────────────────────────────────────────────────────────────────┐
│  ◉ NetScope                              [eth0 ▾]  [🌙]  [⚙️]   │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│   │    Link      │  │  Cable Test  │  │     VLAN     │          │
│   │     🟢       │  │      🟢      │  │      🟢      │          │
│   │   1 Gbps     │  │   32m OK     │  │    ID: 10    │          │
│   │   Full       │  │              │  │   Voice: 20  │          │
│   └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                  │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│   │   Switch     │  │     DHCP     │  │     DNS      │          │
│   │     🟢       │  │      🟡      │  │      🟢      │          │
│   │  SW-CORE-01  │  │  192.168.1.50│  │   8.8.8.8    │          │
│   │  Port Gi1/0/1│  │    1.2s      │  │    45ms      │          │
│   └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                  │
│   ┌──────────────┐                                               │
│   │   Gateway    │                                               │
│   │     🟢       │                                               │
│   │ 192.168.1.1  │                                               │
│   │ 12/14/11 ms  │                                               │
│   └──────────────┘                                               │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

Settings Drawer (slides from right):
┌─────────────────────────┐
│ ⚙️ Settings        [✕]  │
├─────────────────────────┤
│ Interface               │
│ [eth0          ▾]       │
│                         │
│ IP Mode                 │
│ (●) DHCP  ( ) Static    │
│                         │
│ VLAN Tagging            │
│ [ ] Enable              │
│ VLAN ID: [____]         │
│                         │
│ ─── Thresholds ───      │
│ DHCP Total Warning      │
│ [500    ] ms            │
│ ...                     │
│                         │
│ ─── Security ───        │
│ [Change Password]       │
│ [Manage Certificates]   │
│                         │
│ [Export JSON]           │
└─────────────────────────┘
```

---

## Development Phases

### Phase 1: Foundation

- [ ] Repository setup with all tooling
- [ ] CI/CD pipeline
- [ ] Basic Go project structure
- [ ] React frontend scaffold
- [ ] WebSocket infrastructure
- [ ] Authentication system

### Phase 2: Core Cards

- [ ] Link card (ethtool integration)
- [ ] Network interface management
- [ ] Real-time updates working

### Phase 3: Discovery

- [ ] LLDP frame parsing
- [ ] CDP frame parsing
- [ ] EDP/FDP frame parsing
- [ ] Nearest Switch card

### Phase 4: IP & DNS

- [ ] DHCP client with timing hooks
- [ ] DHCP card with phase breakdown
- [ ] DNS testing module
- [ ] DNS card

### Phase 5: Connectivity

- [ ] Gateway detection
- [ ] ICMP ping implementation
- [ ] Gateway card with ping results

### Phase 6: Additional Cards

- [ ] VLAN detection card
- [ ] Wi-Fi card (when on wireless)
- [ ] Cable test card (TDR if supported)

### Phase 7: Polish

- [ ] Settings drawer
- [ ] Threshold configuration
- [ ] JSON export
- [ ] Mobile responsiveness
- [ ] Dark/light mode toggle

### Phase 8: Release

- [ ] Cross-compilation for ARM64
- [ ] Raspberry Pi deployment guide
- [ ] Documentation
- [ ] v1.0.0 release

---

## GitHub Issues Structure

Issues will be created for each feature with labels:

### Labels

- `type: feature` - New feature
- `type: bug` - Bug fix
- `type: chore` - Maintenance
- `type: docs` - Documentation
- `priority: critical` - Must have for release
- `priority: high` - Should have
- `priority: medium` - Nice to have
- `priority: low` - Future consideration
- `component: backend` - Go backend
- `component: frontend` - React frontend
- `component: infra` - CI/CD, tooling
- `card: link` - Link card specific
- `card: switch` - Switch discovery card
- `card: dhcp` - DHCP card
- `card: dns` - DNS card
- `card: gateway` - Gateway card
- `card: vlan` - VLAN card
- `card: wifi` - Wi-Fi card
- `card: cable` - Cable test card

### Milestones

- **v0.1.0** - Foundation (repo, CI/CD, scaffold)
- **v0.2.0** - Core infrastructure (WebSocket, auth)
- **v0.3.0** - Link & Switch cards
- **v0.4.0** - DHCP & DNS cards
- **v0.5.0** - Gateway & VLAN cards
- **v0.6.0** - Wi-Fi & Cable cards
- **v0.7.0** - Settings & Polish
- **v1.0.0** - Production release

---

## References

- [gopacket documentation](https://pkg.go.dev/github.com/google/gopacket)
- [LLDP IEEE 802.1AB](https://standards.ieee.org/standard/802_1AB-2016.html)
- [ethtool man page](https://man7.org/linux/man-pages/man8/ethtool.8.html)
- [Tailwind CSS](https://tailwindcss.com/)
- [shadcn/ui](https://ui.shadcn.com/)

---

_Last updated: 2025-12-02_
_Version: Draft 1.0_
