# Discovery Engine Architecture

## Vision: One Discovery Engine to Rule Them All

The Discovery Engine is the single source of truth for all device discovery,
correlation, and enrichment across wired, WiFi, and Bluetooth networks.

## Proposed Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           DISCOVERY ENGINE (NEW)                                 │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                         DiscoveryEngine                                  │    │
│  │                     (Central Orchestrator)                               │    │
│  │                                                                          │    │
│  │  • Unified device registry (single source of truth)                     │    │
│  │  • Event-driven architecture (real-time updates)                        │    │
│  │  • Automatic correlation on any discovery event                         │    │
│  │  • Configurable scan policies                                           │    │
│  │  • Metrics and telemetry                                                │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                    │                                             │
│                                    │ Events                                      │
│                    ┌───────────────┼───────────────┐                            │
│                    │               │               │                            │
│                    ▼               ▼               ▼                            │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                           EVENT BUS                                      │    │
│  │                                                                          │    │
│  │  DeviceDiscovered | DeviceUpdated | DeviceLost | ScanComplete           │    │
│  │  WiFiAPSeen | BluetoothDeviceSeen | VulnFound | PortOpen               │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                    │               │               │                            │
│        ┌───────────┴───┐   ┌───────┴───────┐   ┌──┴────────────┐               │
│        ▼               ▼   ▼               ▼   ▼               ▼               │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐      │
│  │  Wired   │   │   WiFi   │   │Bluetooth │   │  Enrich  │   │  Assess  │      │
│  │ Collector│   │ Collector│   │Collector │   │  Engine  │   │  Engine  │      │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘      │
│       │               │               │               │               │         │
│       ▼               ▼               ▼               ▼               ▼         │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐          │
│  │ARP/NDP  │   │Airport  │   │CoreBT   │   │SNMP     │   │CVE/NVD  │          │
│  │LLDP/CDP │   │Scanner  │   │blueutil │   │PortScan │   │KEV      │          │
│  │mDNS     │   │         │   │         │   │Profiler │   │         │          │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘   └─────────┘          │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. DiscoveryEngine (Central Orchestrator)

```go
type DiscoveryEngine struct {
    // Unified device registry - THE source of truth
    registry *DeviceRegistry

    // Collectors (discovery sources)
    wired     *WiredCollector
    wifi      *WiFiCollector
    bluetooth *BluetoothCollector

    // Enrichment engines
    enricher  *EnrichmentEngine  // SNMP, ports, profiling
    assessor  *AssessmentEngine  // Vulnerabilities

    // Event system
    eventBus  *EventBus

    // Configuration
    config    *EngineConfig

    // State
    mu        sync.RWMutex
    running   bool
}
```

### 2. DeviceRegistry (Unified Device Store)

```go
type DeviceRegistry struct {
    // Primary index by normalized MAC
    devices map[string]*Device

    // Secondary indexes for fast lookup
    byIP       map[string]*Device
    byHostname map[string]*Device
    byVendor   map[string][]*Device

    // Event callbacks
    onChange func(event DeviceEvent)

    mu sync.RWMutex
}

type Device struct {
    // Identity (immutable after creation)
    ID         string    // UUID
    PrimaryMAC string    // Normalized MAC (canonical identifier)

    // Discovered identities
    MACs       []string  // All known MACs for this device
    IPs        []string  // All known IPs
    Hostnames  []string  // All resolved names

    // Classification
    Vendor     string
    DeviceType DeviceType
    OSFamily   string

    // Connection presence
    Wired      *WiredPresence
    WiFi       *WiFiPresence
    Bluetooth  *BTPresence

    // Enrichment data
    SNMP       *SNMPData
    Ports      []Port
    Profile    *DeviceProfile
    Vulns      []Vulnerability

    // Metadata
    FirstSeen  time.Time
    LastSeen   time.Time
    LastUpdate time.Time

    // Authorization
    Status     AuthStatus  // authorized, unknown, rogue
}
```

### 3. EventBus (Real-time Updates)

```go
type EventBus struct {
    subscribers map[EventType][]EventHandler
    mu          sync.RWMutex
}

type EventType string
const (
    EventDeviceDiscovered EventType = "device.discovered"
    EventDeviceUpdated    EventType = "device.updated"
    EventDeviceLost       EventType = "device.lost"
    EventWiFiAPSeen       EventType = "wifi.ap.seen"
    EventBTDeviceSeen     EventType = "bt.device.seen"
    EventPortDiscovered   EventType = "port.discovered"
    EventVulnFound        EventType = "vuln.found"
    EventScanStarted      EventType = "scan.started"
    EventScanComplete     EventType = "scan.complete"
)

type DeviceEvent struct {
    Type      EventType
    Device    *Device
    Timestamp time.Time
    Source    string  // "wired", "wifi", "bluetooth", "enrichment"
    Changes   map[string]any  // What changed
}
```

### 4. Collectors (Discovery Sources)

Each collector is responsible for ONE discovery method:

```go
// WiredCollector - ARP, NDP, LLDP, CDP, EDP, mDNS
type WiredCollector struct {
    arp      *ARPScanner
    ndp      *NDPScanner
    l2       *L2Manager  // LLDP/CDP/EDP
    mdns     *MDNSResolver
    eventBus *EventBus
}

// WiFiCollector - WiFi AP and client discovery
type WiFiCollector struct {
    scanner  platform.WiFiScanner  // airport (macOS) or iw (Linux)
    eventBus *EventBus
}

// BluetoothCollector - Classic and BLE
type BluetoothCollector struct {
    scanner  platform.BTScanner
    eventBus *EventBus
}
```

### 5. EnrichmentEngine (SNMP, Ports, Profiling)

```go
type EnrichmentEngine struct {
    snmp     *SNMPCollector
    ports    *PortScanner
    profiler *DeviceProfiler
    eventBus *EventBus

    // Work queue for async enrichment
    queue    chan *Device
}
```

### 6. AssessmentEngine (Vulnerabilities)

```go
type AssessmentEngine struct {
    nvd      *NVDClient
    kev      *KEVClient
    local    *LocalCVEDB
    eventBus *EventBus
}
```

## Key Improvements Over Current

| Aspect | Current | Proposed |
|--------|---------|----------|
| **Device Store** | Each subsystem has own list | Single DeviceRegistry |
| **Correlation** | Only at scan time | Real-time via EventBus |
| **ARP Scanning** | Done twice (DeviceDiscovery + Pipeline) | Once via WiredCollector |
| **Orchestration** | Service + UnifiedDiscovery | Single DiscoveryEngine |
| **Updates** | Poll-based | Event-driven |
| **Port Scanning** | PortScanner + Profiler | Single PortScanner in Enrichment |

## API Design

```go
// Engine lifecycle
engine := discovery.NewEngine(config)
engine.Start(ctx)
engine.Stop()

// Scanning
engine.Scan(ctx, ScanOptions{...})        // Full scan
engine.QuickScan(ctx)                      // Correlation only
engine.ScanWired(ctx)                      // Just wired
engine.ScanWiFi(ctx)                       // Just WiFi
engine.ScanBluetooth(ctx)                  // Just Bluetooth

// Device access
devices := engine.GetDevices()             // All devices
device := engine.GetDevice(mac)            // By MAC
devices := engine.GetDevicesByType(Router) // By type
devices := engine.GetRogueDevices()        // Unauthorized

// Real-time events
engine.Subscribe(EventDeviceDiscovered, handler)
engine.Subscribe(EventVulnFound, handler)

// Stats
stats := engine.GetStats()
```

## Migration Path

### Phase 1: Create Engine Shell
- Create `engine.go` with DiscoveryEngine struct
- Create `registry.go` with DeviceRegistry
- Create `events.go` with EventBus

### Phase 2: Refactor Collectors
- Wrap existing ARPScanner, WiFiBridge, BluetoothScanner
- Have them emit events instead of storing state

### Phase 3: Consolidate Enrichment
- Merge port scanning into single component
- Connect SNMP collector to event system

### Phase 4: Wire Everything Together
- Engine orchestrates collectors
- Registry receives all events
- Correlation happens automatically

### Phase 5: Deprecate Old Code
- Mark Service, UnifiedDiscoveryService as deprecated
- Update API handlers to use Engine
- Remove redundant code

## File Structure (Proposed)

```
internal/discovery/
├── engine.go           # DiscoveryEngine (main orchestrator)
├── registry.go         # DeviceRegistry (unified store)
├── device.go           # Device struct and methods
├── events.go           # EventBus and event types
├── collectors/
│   ├── wired.go        # WiredCollector (ARP/NDP/L2/mDNS)
│   ├── wifi.go         # WiFiCollector
│   └── bluetooth.go    # BluetoothCollector
├── enrichment/
│   ├── engine.go       # EnrichmentEngine
│   ├── snmp.go         # SNMP collection
│   ├── ports.go        # Port scanning
│   └── profiler.go     # Device profiling
├── assessment/
│   ├── engine.go       # AssessmentEngine
│   └── cve.go          # CVE/NVD/KEV
├── scanners/           # Low-level scanners (existing)
│   ├── arp.go
│   ├── ndp.go
│   ├── icmp.go
│   ├── lldp.go
│   ├── cdp.go
│   └── edp.go
└── platform/           # Platform-specific implementations
    ├── wifi_darwin.go
    ├── wifi_linux.go
    ├── bt_darwin.go
    └── bt_linux.go
```

## Benefits

1. **Single Source of Truth** - One device registry, no more scattered state
2. **Real-time Correlation** - Devices correlated as they're discovered
3. **No Duplication** - Each capability implemented once
4. **Event-Driven** - UI can subscribe to real-time updates
5. **Testable** - Clear interfaces, easy to mock
6. **Extensible** - Add new collectors without changing core
7. **Maintainable** - Clear separation of concerns
