package enumerate

import "github.com/MustardSeedNetworks/seed/internal/discovery"

// Kernel type aliases for the wired/active discovery collector (ADR-0018 Phase 6
// enumerate split). DeviceDiscovery, Service, the protocol collectors, and the
// Manager moved here from the kernel; the device data types, the DeviceProfiler
// + SNMP result types, the metrics/retry infrastructure, and the registry stay
// kernel-resident. Aliasing them lets the moved collector reference them
// unqualified without inverting the kernel→stage dependency direction.
// The order below mirrors the kernel files the symbols live in: device data
// types, per-protocol sub-types, the profiler/SNMP result cluster, and the
// metrics infrastructure. Each is an alias, not a redefinition.

// DiscoveredDevice is an alias for discovery.DiscoveredDevice; see there for docs.
type DiscoveredDevice = discovery.DiscoveredDevice

// Method is an alias for discovery.Method; see there for docs.
type Method = discovery.Method

// ConnectionType is an alias for discovery.ConnectionType; see there for docs.
type ConnectionType = discovery.ConnectionType

// WiFiPresence is an alias for discovery.WiFiPresence; see there for docs.
type WiFiPresence = discovery.WiFiPresence

// BluetoothPresence is an alias for discovery.BluetoothPresence; see there for docs.
type BluetoothPresence = discovery.BluetoothPresence

// Status is an alias for discovery.Status; see there for docs.
type Status = discovery.Status

// DBDeviceWriter is an alias for discovery.DBDeviceWriter; see there for docs.
type DBDeviceWriter = discovery.DBDeviceWriter

// MDNSService is an alias for discovery.MDNSService; see there for docs.
type MDNSService = discovery.MDNSService

// OpenPort is an alias for discovery.OpenPort; see there for docs.
type OpenPort = discovery.OpenPort

// LLDPDeviceInfo is an alias for discovery.LLDPDeviceInfo; see there for docs.
type LLDPDeviceInfo = discovery.LLDPDeviceInfo

// CDPDeviceInfo is an alias for discovery.CDPDeviceInfo; see there for docs.
type CDPDeviceInfo = discovery.CDPDeviceInfo

// EDPDeviceInfo is an alias for discovery.EDPDeviceInfo; see there for docs.
type EDPDeviceInfo = discovery.EDPDeviceInfo

// NDPDeviceInfo is an alias for discovery.NDPDeviceInfo; see there for docs.
type NDPDeviceInfo = discovery.NDPDeviceInfo

// DeviceProfiler is an alias for discovery.DeviceProfiler; see there for docs.
type DeviceProfiler = discovery.DeviceProfiler

// DeviceProfile is an alias for discovery.DeviceProfile; see there for docs.
type DeviceProfile = discovery.DeviceProfile

// ProfilingStatus is an alias for discovery.ProfilingStatus; see there for docs.
type ProfilingStatus = discovery.ProfilingStatus

// DeviceVulnerabilities is an alias for discovery.DeviceVulnerabilities; see there for docs.
type DeviceVulnerabilities = discovery.DeviceVulnerabilities

// Vulnerability is an alias for discovery.Vulnerability; see there for docs.
type Vulnerability = discovery.Vulnerability

// SNMPFullData is an alias for discovery.SNMPFullData; see there for docs.
type SNMPFullData = discovery.SNMPFullData

// SNMPEntity is an alias for discovery.SNMPEntity; see there for docs.
type SNMPEntity = discovery.SNMPEntity

// SNMPInterface is an alias for discovery.SNMPInterface; see there for docs.
type SNMPInterface = discovery.SNMPInterface

// SNMPIPAddress is an alias for discovery.SNMPIPAddress; see there for docs.
type SNMPIPAddress = discovery.SNMPIPAddress

// SNMPLLDPNeighbor is an alias for discovery.SNMPLLDPNeighbor; see there for docs.
type SNMPLLDPNeighbor = discovery.SNMPLLDPNeighbor

// SNMPMACEntry is an alias for discovery.SNMPMACEntry; see there for docs.
type SNMPMACEntry = discovery.SNMPMACEntry

// SNMPVLAN is an alias for discovery.SNMPVLAN; see there for docs.
type SNMPVLAN = discovery.SNMPVLAN

// Metrics is an alias for discovery.Metrics; see there for docs.
type Metrics = discovery.Metrics

// ScanDelta is an alias for discovery.ScanDelta; see there for docs.
type ScanDelta = discovery.ScanDelta

// DegradationStatus is an alias for discovery.DegradationStatus; see there for docs.
type DegradationStatus = discovery.DegradationStatus

// Kernel constant aliases (device discovery methods).
const (
	MethodARP  = discovery.MethodARP
	MethodPING = discovery.MethodPING
	MethodLLDP = discovery.MethodLLDP
	MethodCDP  = discovery.MethodCDP
	MethodEDP  = discovery.MethodEDP
	MethodNDP  = discovery.MethodNDP
	MethodMDNS = discovery.MethodMDNS
)

// ErrScanInProgress is the kernel sentinel returned when a scan is requested
// while one is already running; re-exported so the moved collector returns the
// same value callers compare against. (Kernel functions like NewDeviceProfiler
// are called qualified — discovery.NewDeviceProfiler — to avoid package-level
// function aliases.)
var ErrScanInProgress = discovery.ErrScanInProgress

// Impl-tuning constants that moved with the collector (used only by the moved
// devices.go / devices_scan.go, never by kernel-staying code).
const (
	ouiUpdateTimeoutMinutes = 2  // Timeout for OUI database updates
	nameResGoroutineCount   = 2  // Number of name resolution goroutines
	dbPersistTimeoutSeconds = 30 // Timeout for database persistence operations
	deviceTTLHours          = 24 // Default device TTL in hours before expiration

	macOctetMinLen  = 2    // Minimum length to parse a MAC octet
	hexLetterOffset = 10   // Offset for A-F hex digits (after subtracting 'A'/'a')
	localAdminBit   = 0x02 // Bit mask for locally administered MAC address check

	// maxIPv6AddressesPerDevice caps IPv6 accumulation per device (fixes #884).
	maxIPv6AddressesPerDevice = 16
)
