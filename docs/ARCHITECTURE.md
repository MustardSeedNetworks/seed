# The Seed - System Architecture

## High-Level Architecture

```
                                    ┌─────────────────────────────────────────────────────────────────┐
                                    │                         FRONTEND (React)                        │
                                    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌──────────┐  │
                                    │  │   Pages     │ │ Components  │ │   Hooks     │ │  Stores  │  │
                                    │  │ (Dashboard, │ │ (Cards,     │ │ (useDevice, │ │ (Zustand)│  │
                                    │  │  Settings)  │ │  Modals)    │ │  useNetwork)│ │          │  │
                                    │  └─────────────┘ └─────────────┘ └─────────────┘ └──────────┘  │
                                    └───────────────────────────────┬─────────────────────────────────┘
                                                                    │
                                                    REST API + SSE + WebSocket
                                                                    │
                                                                    ▼
┌───────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        HTTP API LAYER (internal/httpapi)                                   │
│  ┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                          Server (server.go)                                          │ │
│  │   - Route registration      - HTTPS/TLS      - Middleware (auth, CORS, rate limiting)               │ │
│  │   - ServiceContainer        - SSE Hub        - WebSocket Hub                                         │ │
│  └──────────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                                            │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐     │
│  │ handlers_       │ │ handlers_       │ │ handlers_       │ │ handlers_       │ │ handlers_       │     │
│  │ discovery.go    │ │ devices.go      │ │ health*.go      │ │ network.go      │ │ settings.go     │     │
│  │ handlers_       │ │ handlers_       │ │ handlers_       │ │ handlers_       │ │ handlers_       │     │
│  │ engine.go       │ │ bluetooth.go    │ │ sse.go          │ │ snmp.go         │ │ auth.go         │     │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘     │
└───────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                    │
                              ┌─────────────────────┴─────────────────────┐
                              ▼                                           ▼
┌─────────────────────────────────────────────────┐   ┌─────────────────────────────────────────────────┐
│              MODULE SERVICES                     │   │              CORE SERVICES                       │
├─────────────────────────────────────────────────┤   ├─────────────────────────────────────────────────┤
│                                                 │   │                                                 │
│  ┌─────────────────────────────────────────┐   │   │   ┌─────────────────────────────────────────┐   │
│  │         SHELL (Security Posture)        │   │   │   │            NETWORK                      │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │   │   │   │  ┌──────────────────────────────────┐  │   │
│  │  │Discovery │ │Vulnerab- │ │ Port     │ │   │   │   │  │ Manager, Interfaces, Link       │  │   │
│  │  │Engine    │ │ility     │ │ Scanner  │ │   │   │   │  │ Monitor, Speed Duplex           │  │   │
│  │  └──────────┘ └──────────┘ └──────────┘ │   │   │   │  └──────────────────────────────────┘  │   │
│  └─────────────────────────────────────────┘   │   │   └─────────────────────────────────────────┘   │
│                                                 │   │                                                 │
│  ┌─────────────────────────────────────────┐   │   │   ┌─────────────────────────────────────────┐   │
│  │         SAP (Live Telemetry)            │   │   │   │              SNMP                       │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐       │   │   │   │  ┌──────────────────────────────────┐  │   │
│  │  │ DNS    │ │Gateway │ │Speed-  │       │   │   │   │  │ Query, Walk, SystemInfo          │  │   │
│  │  │ Tester │ │Tester  │ │test    │       │   │   │   │  │ IF-MIB, LLDP-MIB, ENTITY-MIB    │  │   │
│  │  ├────────┤ ├────────┤ ├────────┤       │   │   │   │  └──────────────────────────────────┘  │   │
│  │  │ DHCP   │ │ Iperf  │ │ VLAN   │       │   │   │   └─────────────────────────────────────────┘   │
│  │  │Monitor │ │Manager │ │Manager │       │   │   │                                                 │
│  │  ├────────┤ ├────────┤ ├────────┤       │   │   │   ┌─────────────────────────────────────────┐   │
│  │  │ Cable  │ │Public  │ │ Link   │       │   │   │   │            MIB DB                       │   │
│  │  │ Tester │ │ IP     │ │Monitor │       │   │   │   │  ┌──────────────────────────────────┐  │   │
│  │  └────────┘ └────────┘ └────────┘       │   │   │   │  │ OID Name Resolution (918+ OIDs) │  │   │
│  └─────────────────────────────────────────┘   │   │   │  │ RFC MIB Support                  │  │   │
│                                                 │   │   │  └──────────────────────────────────┘  │   │
│  ┌─────────────────────────────────────────┐   │   │   └─────────────────────────────────────────┘   │
│  │         CANOPY (WiFi Planning)          │   │   │                                                 │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │   │   │   ┌─────────────────────────────────────────┐   │
│  │  │ WiFi    │ │ Scanner  │ │ Survey   │ │   │   │   │            HEALTH                       │   │
│  │  │ Manager │ │          │ │ Manager  │ │   │   │   │  ┌──────────────────────────────────┐  │   │
│  │  └──────────┘ └──────────┘ └──────────┘ │   │   │   │  │ SLA Tracker, Anomaly Detector   │  │   │
│  └─────────────────────────────────────────┘   │   │   │  │ Dependency Manager, Scoring     │  │   │
│                                                 │   │   │  └──────────────────────────────────┘  │   │
│  ┌─────────────────────────────────────────┐   │   │   └─────────────────────────────────────────┘   │
│  │         ROOTS (Path Analysis)           │   │   │                                                 │
│  │  ┌──────────────────────────────────┐   │   │   │   ┌─────────────────────────────────────────┐   │
│  │  │ Traceroute, Public IP Checker    │   │   │   │   │            ALERTS                       │   │
│  │  └──────────────────────────────────┘   │   │   │   │  ┌──────────────────────────────────┐  │   │
│  └─────────────────────────────────────────┘   │   │   │  │ Alert Manager, Rules, Channels  │  │   │
│                                                 │   │   │  └──────────────────────────────────┘  │   │
│  ┌─────────────────────────────────────────┐   │   │   └─────────────────────────────────────────┘   │
│  │         HARVEST (Reporting)             │   │   │                                                 │
│  │  ┌──────────────────────────────────┐   │   │   │   ┌─────────────────────────────────────────┐   │
│  │  │ Report Scheduler, PDF Export     │   │   │   │   │            AUTH                         │   │
│  │  └──────────────────────────────────┘   │   │   │   │  ┌──────────────────────────────────┐  │   │
│  └─────────────────────────────────────────┘   │   │   │  │ JWT Manager, CSRF, OAuth         │  │   │
│                                                 │   │   │  └──────────────────────────────────┘  │   │
└─────────────────────────────────────────────────┘   │   └─────────────────────────────────────────┘   │
                                                      └─────────────────────────────────────────────────┘
                                                                              │
                                                                              ▼
┌───────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       DATA LAYER (internal/database)                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │                                         SQLite Database                                              │  │
│  │   ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐         │  │
│  │   │   Profiles    │ │    Metrics    │ │    Devices    │ │    Alerts     │ │ Health Checks │         │  │
│  │   │  Repository   │ │  Repository   │ │  Repository   │ │  Repository   │ │  Repository   │         │  │
│  │   └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘         │  │
│  │   ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐                           │  │
│  │   │   Settings    │ │     Logs      │ │   Discovery   │ │    MIB OIDs   │                           │  │
│  │   │  Repository   │ │  Repository   │ │  Repository   │ │    (mibdb)    │                           │  │
│  │   └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘                           │  │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Module Color Reference

| Module | Purpose | Color |
|--------|---------|-------|
| **Roots** | Path analysis (traceroute, public IP) | Amber #b45309 |
| **Canopy** | Wi-Fi planning (surveys, channel analysis) | Green #2d7a3e |
| **Shell** | Security posture (discovery, vulnerabilities) | Orange #ea580c |
| **Sap** | Live telemetry (DNS, DHCP, speed tests) | Cyan #0891b2 |
| **Harvest** | Reporting (scheduled reports, export) | Gold #d4a017 |

## Data Flow

```
┌──────────┐     HTTP/SSE/WS      ┌───────────┐     Service Calls     ┌──────────────┐
│ Frontend │ ◄──────────────────► │  HTTP API │ ◄──────────────────► │   Modules    │
│  (React) │                      │  (Go)     │                      │ (Shell/Sap)  │
└──────────┘                      └───────────┘                      └──────────────┘
                                       │                                    │
                                       │                                    │
                                       ▼                                    ▼
                                ┌───────────┐                        ┌──────────────┐
                                │  SQLite   │◄───────────────────────│ SNMP/Network │
                                │ Database  │                        │   Queries    │
                                └───────────┘                        └──────────────┘
```

## Key Packages

### HTTP API Layer (`internal/httpapi/`)
- `server.go` - Server initialization, routes, middleware
- `services.go` - ServiceContainer dependency injection
- `handlers_*.go` - Request handlers by domain
- `sse.go` - Server-Sent Events for real-time updates
- `websocket.go` - WebSocket support (deprecated, prefer SSE)

### Discovery System (`internal/discovery/`)
- `engine.go` - Unified DiscoveryEngine (primary)
- `registry.go` - Device registry with deduplication
- `pipeline.go` - Multi-phase discovery pipeline
- `profiler.go` - Device profiling (SNMP, ports)
- `bluetooth.go` - Bluetooth device scanning
- `wifi.go` - WiFi network discovery
- `problem_detector.go` - Network problem detection

### Data Layer (`internal/database/`)
- `database.go` - DB connection and pooling
- `migrations.go` - Schema migrations
- `repository_*.go` - Data access by domain

### Network (`internal/network/`)
- `manager.go` - Interface management
- `interfaces.go` - Interface detection
- `link.go` - Link state monitoring

### SNMP (`internal/snmp/`)
- `snmp.go` - Core query functions
- `interface.go` - IF-MIB operations
- `lldp.go` - LLDP-MIB operations
- `entitymib.go` - ENTITY-MIB operations

---

## Discovery Subsystem Architecture

The Discovery Engine is the central orchestrator for all device discovery, enrichment, and assessment.

```
                                    ┌─────────────────────────────────────────────────────────────┐
                                    │                    DISCOVERY ENGINE                          │
                                    │                      (engine.go)                             │
                                    │                                                              │
                                    │  ┌──────────────────────────────────────────────────────┐   │
                                    │  │ Configuration (EngineConfig)                         │   │
                                    │  │ - EnableWired/WiFi/Bluetooth                        │   │
                                    │  │ - EnableSNMP/PortScan/Profiling/VulnScan           │   │
                                    │  │ - AutoScanInterval, ScanTimeout, DeviceTTL         │   │
                                    │  └──────────────────────────────────────────────────────┘   │
                                    │                           │                                  │
                                    │              ┌────────────┴────────────┐                    │
                                    │              ▼                         ▼                    │
                                    │  ┌─────────────────────┐  ┌─────────────────────────────┐  │
                                    │  │   DeviceRegistry    │  │        EventBus             │  │
                                    │  │   (registry.go)     │  │       (events.go)           │  │
                                    │  │                     │  │                             │  │
                                    │  │ - Primary: MAC→Dev  │  │ - Async event distribution  │  │
                                    │  │ - Index: IP→Dev     │  │ - Subscribers (SSE, UI)     │  │
                                    │  │ - Index: Host→Dev   │  │ - Event types:              │  │
                                    │  │ - Index: Vendor→Dev │  │   - device.discovered       │  │
                                    │  │ - Auto deduplication│  │   - device.updated          │  │
                                    │  │ - TTL expiration    │  │   - device.lost             │  │
                                    │  └─────────────────────┘  │   - scan.* lifecycle        │  │
                                    │                           └─────────────────────────────┘  │
                                    └──────────────────────────────────────┬──────────────────────┘
                                                                           │
                    ┌──────────────────────────────────────────────────────┼──────────────────────────────────────────────────────┐
                    │                                                      │                                                      │
                    ▼                                                      ▼                                                      ▼
┌──────────────────────────────────────────┐  ┌──────────────────────────────────────────┐  ┌──────────────────────────────────────────┐
│            COLLECTORS                     │  │            ENRICHMENT                     │  │            ASSESSMENT                    │
│         (Discovery Sources)               │  │       (Device Profiling)                  │  │      (Security Analysis)                 │
├──────────────────────────────────────────┤  ├──────────────────────────────────────────┤  ├──────────────────────────────────────────┤
│                                          │  │                                          │  │                                          │
│  ┌────────────────────────────────────┐  │  │  ┌────────────────────────────────────┐  │  │  ┌────────────────────────────────────┐  │
│  │     Wired Collector (devices.go)   │  │  │  │   SNMP Collector (snmp_collector)  │  │  │  │  Vulnerability Scanner (vuln.go)  │  │
│  │  ┌──────────┐ ┌──────────┐        │  │  │  │  - System info (sysName, sysDescr) │  │  │  │  - NVD database lookup             │  │
│  │  │   ARP    │ │   NDP    │        │  │  │  │  - Interface metrics (IF-MIB)      │  │  │  │  - KEV (Known Exploited Vulns)    │  │
│  │  │ Scanner  │ │ Scanner  │        │  │  │  │  - Entity info (ENTITY-MIB)        │  │  │  │  - Local CVE cache                │  │
│  │  └──────────┘ └──────────┘        │  │  │  │  - LLDP neighbors (LLDP-MIB)       │  │  │  │  - Severity filtering             │  │
│  │  ┌──────────┐ ┌──────────┐        │  │  │  └────────────────────────────────────┘  │  │  └────────────────────────────────────┘  │
│  │  │   LLDP   │ │   CDP    │        │  │  │                                          │  │                                          │
│  │  │ Listener │ │ Listener │        │  │  │  ┌────────────────────────────────────┐  │  └──────────────────────────────────────────┘
│  │  └──────────┘ └──────────┘        │  │  │  │   Port Scanner (portscan.go)       │  │
│  │  ┌──────────┐ ┌──────────┐        │  │  │  │  - TCP SYN/Connect probes         │  │
│  │  │   mDNS   │ │ NetBIOS  │        │  │  │  │  - Service fingerprinting         │  │
│  │  │ Browser  │ │ Resolver │        │  │  │  │  - Common ports (22,80,443...)    │  │
│  │  └──────────┘ └──────────┘        │  │  │  └────────────────────────────────────┘  │
│  └────────────────────────────────────┘  │  │                                          │
│                                          │  │  ┌────────────────────────────────────┐  │
│  ┌────────────────────────────────────┐  │  │  │   Device Profiler (profiler.go)    │  │
│  │   WiFi Collector (wifi_bridge.go)  │  │  │  │  - OS fingerprinting              │  │
│  │  - AP discovery (SSID, BSSID)      │  │  │  │  - Device type classification     │  │
│  │  - Signal strength (RSSI)          │  │  │  │  - Manufacturer identification    │  │
│  │  - Channel/frequency               │  │  │  │  - Service correlation            │  │
│  │  - Security mode (WPA2/3)          │  │  │  │  - OUI lookup (vendor by MAC)     │  │
│  │  - Connected clients               │  │  │  └────────────────────────────────────┘  │
│  └────────────────────────────────────┘  │  │                                          │
│                                          │  └──────────────────────────────────────────┘
│  ┌────────────────────────────────────┐  │
│  │ Bluetooth Collector (bluetooth.go) │  │
│  │  - BLE device discovery            │  │
│  │  - Classic BT discovery            │  │
│  │  - Device class/type               │  │
│  │  - Signal strength                 │  │
│  │  - Platform: Darwin/Linux          │  │
│  └────────────────────────────────────┘  │
│                                          │
└──────────────────────────────────────────┘
```

### Discovery Pipeline (Multi-Phase Execution)

The pipeline executes discovery in sequential phases:

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  ENUMERATION │────►│  RESOLUTION  │────►│   SCANNING   │────►│  ASSESSMENT  │
│              │     │              │     │              │     │              │
│ - ARP sweep  │     │ - DNS lookup │     │ - Port scan  │     │ - Vuln scan  │
│ - NDP query  │     │ - mDNS query │     │ - SNMP walk  │     │ - CVE lookup │
│ - LLDP/CDP   │     │ - NetBIOS    │     │ - Profiling  │     │ - Scoring    │
│ - Ping sweep │     │ - OUI lookup │     │ - Services   │     │ - Alerts     │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
       │                    │                    │                    │
       └────────────────────┴────────────────────┴────────────────────┘
                                     │
                                     ▼
                          ┌───────────────────────┐
                          │     Event Stream      │
                          │  (SSE to Frontend)    │
                          │                       │
                          │ - phase_started       │
                          │ - phase_progress      │
                          │ - device_discovered   │
                          │ - pipeline_completed  │
                          └───────────────────────┘
```

### Scan Profiles

| Profile | Probe Delay | Host Delay | Concurrent | Phase Timeout |
|---------|-------------|------------|------------|---------------|
| **Polite** | 200ms | 100ms | 5 | 30 min |
| **Normal** | 50ms | 20ms | 20 | 10 min |
| **Aggressive** | 10ms | 5ms | 100 | 5 min |

### Key Files

| File | Purpose |
|------|---------|
| `engine.go` | Central orchestrator, scan coordination |
| `registry.go` | Single source of truth for devices |
| `events.go` | Event bus for real-time updates |
| `pipeline.go` | Multi-phase pipeline execution |
| `devices.go` | Wired discovery (ARP, NDP, LLDP, CDP) |
| `wifi_bridge.go` | WiFi discovery bridge |
| `bluetooth.go` | Bluetooth scanning |
| `snmp_collector.go` | SNMP data collection |
| `profiler.go` | Device fingerprinting |
| `portscan.go` | TCP/UDP port scanning |
| `vulnerability.go` | CVE/vulnerability assessment |
| `oui.go` | MAC vendor lookup (OUI database) |

### Event Types

**Device Lifecycle:**
- `device.discovered` - New device found
- `device.updated` - Device data changed
- `device.lost` - Device offline
- `device.merged` - Duplicate devices consolidated

**Discovery Sources:**
- `wired.arp`, `wired.ndp`, `wired.lldp`, `wired.cdp`, `wired.mdns`
- `wifi.ap.discovered`, `wifi.ap.updated`, `wifi.client.discovered`
- `bt.device.discovered`, `bt.device.updated`

**Enrichment:**
- `enrichment.port`, `enrichment.snmp`, `enrichment.profile`, `enrichment.name`

**Assessment:**
- `assessment.vuln`, `assessment.resolved`

**Scan Lifecycle:**
- `scan.started`, `scan.progress`, `scan.completed`, `scan.failed`, `scan.canceled`
