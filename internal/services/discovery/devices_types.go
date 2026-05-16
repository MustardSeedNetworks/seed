package discovery

// devices_types.go contains the DiscoveredDevice value type, the discovery
// method / connection / presence enums and their constants, the per-protocol
// info structs (LLDP/CDP/EDP/NDP/WiFi/Bluetooth), the persistence interface,
// and the Status snapshot returned by GetStatus.

import (
	"context"
	"errors"
	"time"
)

// Method indicates how a device was discovered.
type Method string

// Device discovery method constants.
const (
	MethodARP  Method = "arp"
	MethodNDP  Method = "ndp" // IPv6 Neighbor Discovery
	MethodLLDP Method = "lldp"
	MethodCDP  Method = "cdp"
	MethodEDP  Method = "edp"
	MethodMDNS Method = "mdns"
	MethodPING Method = "ping"
)

// ConnectionType indicates how a device connects to the network.
type ConnectionType string

// Connection type constants for unified discovery.
const (
	ConnectionWired     ConnectionType = "wired"     // Discovered via ARP/LLDP/CDP/EDP
	ConnectionWiFi      ConnectionType = "wifi"      // Discovered as WiFi AP or client
	ConnectionBluetooth ConnectionType = "bluetooth" // Discovered via Bluetooth/BLE
)

// WiFiPresence contains WiFi-specific discovery information for a device.
// This is populated when a device's MAC matches a WiFi AP BSSID.
type WiFiPresence struct {
	SSID          string    `json:"ssid,omitempty"`
	Channel       int       `json:"channel,omitempty"`
	ChannelWidth  int       `json:"channelWidth,omitempty"`
	FrequencyMHz  int       `json:"frequencyMHz,omitempty"`
	SignalDBm     int       `json:"signalDbm,omitempty"`
	IsAccessPoint bool      `json:"isAccessPoint"`
	IsAuthorized  bool      `json:"isAuthorized"`
	SecurityType  string    `json:"securityType,omitempty"`
	Band          string    `json:"band,omitempty"` // "2.4GHz", "5GHz", "6GHz"
	LastSeen      time.Time `json:"lastSeen"`
}

// BluetoothPresence contains Bluetooth-specific discovery information for a device.
// This is populated when a device's MAC matches a Bluetooth device address.
type BluetoothPresence struct {
	Name         string               `json:"name,omitempty"`
	Type         BluetoothType        `json:"type"`                  // classic, ble, dual
	DeviceClass  BluetoothDeviceClass `json:"deviceClass,omitempty"` // computer, phone, etc.
	RSSI         int                  `json:"rssi,omitempty"`        // Signal strength
	TxPower      int                  `json:"txPower,omitempty"`     // Transmit power for distance calc
	IsPaired     bool                 `json:"isPaired"`              // Currently paired
	IsConnected  bool                 `json:"isConnected"`           // Currently connected
	IsAuthorized bool                 `json:"isAuthorized"`          // In authorized list
	Services     []string             `json:"services,omitempty"`    // Discovered services/UUIDs
	LastSeen     time.Time            `json:"lastSeen"`
}

// Time constants for device discovery operations.
const (
	ouiUpdateTimeoutMinutes = 2  // Timeout for OUI database updates
	nameResGoroutineCount   = 2  // Number of name resolution goroutines
	dbPersistTimeoutSeconds = 30 // Timeout for database persistence operations
)

// MAC address parsing constants.
const (
	macOctetMinLen  = 2    // Minimum length to parse a MAC octet
	hexLetterOffset = 10   // Offset to add when parsing A-F hex digits (after subtracting 'A' or 'a')
	localAdminBit   = 0x02 // Bit mask for locally administered MAC address check
	deviceTTLHours  = 24   // Default device TTL in hours before expiration
)

// maxIPv6AddressesPerDevice limits IPv6 address accumulation to prevent
// unbounded memory growth from devices with many addresses (fixes #884).
const maxIPv6AddressesPerDevice = 16

// DiscoveredDevice represents a network device with aggregated discovery info.
type DiscoveredDevice struct {
	IP              string    `json:"ip"`                      // Primary IPv4 address
	IPv6Address     string    `json:"ipv6,omitempty"`          // Primary IPv6 address
	IPv6Addresses   []string  `json:"ipv6Addresses,omitempty"` // All IPv6 addresses
	MAC             string    `json:"mac"`
	Hostname        string    `json:"hostname,omitempty"`    // DNS PTR resolved name
	NetBIOSName     string    `json:"netbiosName,omitempty"` // Windows NetBIOS name (UDP 137)
	MDNSName        string    `json:"mdnsName,omitempty"`    // mDNS/Bonjour .local name
	DisplayName     string    `json:"displayName,omitempty"` // Best available name for UI display
	Vendor          string    `json:"vendor,omitempty"`
	OSGuess         string    `json:"osGuess,omitempty"`
	TTL             int       `json:"ttl,omitempty"`
	DiscoveryMethod []Method  `json:"discoveryMethod"`
	LastSeen        time.Time `json:"lastSeen"`
	IsLocal         bool      `json:"isLocal"`            // true if device is on local subnet
	IsRouter        bool      `json:"isRouter,omitempty"` // true if detected as IPv6 router via NDP

	// Duplicate IP detection
	HasDuplicateIP bool     `json:"hasDuplicateIP,omitempty"` // true if same IP seen with multiple MACs
	DuplicateMACs  []string `json:"duplicateMACs,omitempty"`  // Other MACs seen with this IP

	// Protocol-specific details (populated if discovered via that protocol)
	LLDPInfo *LLDPDeviceInfo `json:"lldpInfo,omitempty"`
	CDPInfo  *CDPDeviceInfo  `json:"cdpInfo,omitempty"`
	EDPInfo  *EDPDeviceInfo  `json:"edpInfo,omitempty"`
	NDPInfo  *NDPDeviceInfo  `json:"ndpInfo,omitempty"` // IPv6 Neighbor Discovery info

	// Auto-profiling results
	Profile *DeviceProfile `json:"profile,omitempty"`

	// Extended SNMP data from Phase 3 scanning
	SNMPData *SNMPFullData `json:"snmpData,omitempty"`

	// Vulnerability assessment results from Phase 4
	Vulnerabilities *DeviceVulnerabilities `json:"vulnerabilities,omitempty"`

	// Wake-on-LAN capability
	WoLCapable *bool  `json:"wolCapable,omitempty"` // nil=unknown, true=likely supports WoL, false=likely not
	WoLStatus  string `json:"wolStatus,omitempty"`  // "untested", "success", "failed"

	// Unified discovery: connection types and cross-system presence
	// These fields are populated by UnifiedDiscoveryService when correlating
	// devices across wired, WiFi, and Bluetooth discovery.
	ConnectionTypes   []ConnectionType   `json:"connectionTypes,omitempty"`   // wired, wifi, bluetooth
	WiFiPresence      *WiFiPresence      `json:"wifiPresence,omitempty"`      // WiFi AP/client info if MAC matches
	BluetoothPresence *BluetoothPresence `json:"bluetoothPresence,omitempty"` // Bluetooth info if MAC matches
}

// LLDPDeviceInfo contains LLDP-specific device information.
type LLDPDeviceInfo struct {
	ChassisID         string   `json:"chassisId"`
	PortID            string   `json:"portId"`
	PortDescription   string   `json:"portDescription,omitempty"`
	SystemName        string   `json:"systemName,omitempty"`
	SystemDescription string   `json:"systemDescription,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ManagementAddress string   `json:"managementAddress,omitempty"`
}

// CDPDeviceInfo contains CDP-specific device information.
type CDPDeviceInfo struct {
	DeviceID          string   `json:"deviceId"`
	PortID            string   `json:"portId"`
	Platform          string   `json:"platform,omitempty"`
	SoftwareVersion   string   `json:"softwareVersion,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ManagementAddress string   `json:"managementAddress,omitempty"`
	NativeVLAN        int      `json:"nativeVlan,omitempty"`
	VoiceVLAN         int      `json:"voiceVlan,omitempty"`
}

// EDPDeviceInfo contains EDP-specific device information.
type EDPDeviceInfo struct {
	DeviceID        string `json:"deviceId"`
	DisplayName     string `json:"displayName,omitempty"`
	PortID          string `json:"portId"`
	Platform        string `json:"platform,omitempty"`
	SoftwareVersion string `json:"softwareVersion,omitempty"`
	VLAN            int    `json:"vlan,omitempty"`
}

// NDPDeviceInfo contains IPv6 NDP-specific device information.
type NDPDeviceInfo struct {
	LinkLayerAddress  string    `json:"linkLayerAddress"`           // MAC from NDP
	IsRouter          bool      `json:"isRouter"`                   // From Router Advertisement
	ReachableTime     uint32    `json:"reachableTime,omitempty"`    // milliseconds
	RetransTimer      uint32    `json:"retransTimer,omitempty"`     // milliseconds
	Flags             uint8     `json:"flags,omitempty"`            // NDP flags
	LastAdvertisement time.Time `json:"lastAdvertisement,omitzero"` // Last RA received
}

// DBDeviceWriter defines the interface for persisting devices to a database.
// This interface allows the discovery package to persist devices without depending on
// the database package (avoiding circular imports).
type DBDeviceWriter interface {
	// PersistDevices persists a batch of discovered devices to the database.
	PersistDevices(ctx context.Context, devices []*DiscoveredDevice) error
}

// Status represents the current discovery status.
type Status struct {
	Scanning    bool      `json:"scanning"`
	DeviceCount int       `json:"deviceCount"`
	LastScan    time.Time `json:"lastScan"`
	Subnet      string    `json:"subnet"`
	LocalIP     string    `json:"localIP"`
	Interface   string    `json:"interface"`
}

// ErrScanInProgress indicates a scan was requested while one is already running.
// Callers should check for this specific error to distinguish between "scan completed
// successfully" and "scan was skipped because one is already in progress".
var ErrScanInProgress = errors.New("scan already in progress")

// ComputeDisplayName returns the best available name for a device.
// Priority order:
//  1. LLDP/CDP SystemName (network devices identify themselves)
//  2. mDNS name (Apple/Linux devices with Bonjour)
//  3. NetBIOS name (Windows devices)
//  4. DNS hostname (PTR record)
//  5. IP address (fallback)
func (device *DiscoveredDevice) ComputeDisplayName() string {
	// Network device names from discovery protocols
	if device.LLDPInfo != nil && device.LLDPInfo.SystemName != "" {
		return device.LLDPInfo.SystemName
	}
	if device.CDPInfo != nil && device.CDPInfo.DeviceID != "" {
		return device.CDPInfo.DeviceID
	}
	if device.EDPInfo != nil && device.EDPInfo.DisplayName != "" {
		return device.EDPInfo.DisplayName
	}

	// mDNS name (usually friendly like "Johns-MacBook.local")
	if device.MDNSName != "" {
		return device.MDNSName
	}

	// NetBIOS name (Windows: DESKTOP-ABC123)
	if device.NetBIOSName != "" {
		return device.NetBIOSName
	}

	// DNS hostname (PTR record)
	if device.Hostname != "" {
		return device.Hostname
	}

	// Fallback to IP
	if device.IP != "" {
		return device.IP
	}

	// Last resort: MAC address
	return device.MAC
}
