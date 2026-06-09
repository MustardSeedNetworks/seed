package enumerate

import "github.com/MustardSeedNetworks/seed/internal/discovery"

// Kernel type aliases for the wired/active discovery collector (ADR-0018 Phase 6
// enumerate split). DeviceDiscovery, Service, the protocol collectors, and the
// Manager moved here from the kernel; the device data types, the DeviceProfiler
// + SNMP result types, the metrics/retry infrastructure, and the registry stay
// kernel-resident. Aliasing them lets the moved collector reference them
// unqualified without inverting the kernel→stage dependency direction.
// The grouping below mirrors the kernel files the symbols live in: device data
// types, per-protocol sub-types, the profiler/SNMP result cluster, and the
// metrics infrastructure. Each is an alias, not a redefinition. The blank line
// below keeps this a free-standing comment rather than DiscoveredDevice's doc.

type (
	DiscoveredDevice  = discovery.DiscoveredDevice
	Method            = discovery.Method
	ConnectionType    = discovery.ConnectionType
	WiFiPresence      = discovery.WiFiPresence
	BluetoothPresence = discovery.BluetoothPresence
	Status            = discovery.Status
	DBDeviceWriter    = discovery.DBDeviceWriter
	MDNSService       = discovery.MDNSService
	OpenPort          = discovery.OpenPort

	LLDPDeviceInfo = discovery.LLDPDeviceInfo
	CDPDeviceInfo  = discovery.CDPDeviceInfo
	EDPDeviceInfo  = discovery.EDPDeviceInfo
	NDPDeviceInfo  = discovery.NDPDeviceInfo

	DeviceProfiler        = discovery.DeviceProfiler
	DeviceProfile         = discovery.DeviceProfile
	ProfilingStatus       = discovery.ProfilingStatus
	DeviceVulnerabilities = discovery.DeviceVulnerabilities
	Vulnerability         = discovery.Vulnerability
	SNMPFullData          = discovery.SNMPFullData
	SNMPEntity            = discovery.SNMPEntity
	SNMPInterface         = discovery.SNMPInterface
	SNMPIPAddress         = discovery.SNMPIPAddress
	SNMPLLDPNeighbor      = discovery.SNMPLLDPNeighbor
	SNMPMACEntry          = discovery.SNMPMACEntry
	SNMPVLAN              = discovery.SNMPVLAN

	Metrics           = discovery.Metrics
	ScanDelta         = discovery.ScanDelta
	DegradationStatus = discovery.DegradationStatus
)

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
