package discovery

// It manages the aggregation of device information discovered through various protocols
// including ARP, NDP, LLDP, CDP, EDP, mDNS, and ICMP ping. The package maintains a synchronized
// view of all discovered devices and their protocol-specific metadata.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
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
	Type         BluetoothType        `json:"type"`                   // classic, ble, dual
	DeviceClass  BluetoothDeviceClass `json:"deviceClass,omitempty"`  // computer, phone, etc.
	RSSI         int                  `json:"rssi,omitempty"`         // Signal strength
	TxPower      int                  `json:"txPower,omitempty"`      // Transmit power for distance calc
	IsPaired     bool                 `json:"isPaired"`               // Currently paired
	IsConnected  bool                 `json:"isConnected"`            // Currently connected
	IsAuthorized bool                 `json:"isAuthorized"`           // In authorized list
	Services     []string             `json:"services,omitempty"`     // Discovered services/UUIDs
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

// DeviceDiscovery aggregates device discovery from all sources.
type DeviceDiscovery struct {
	interfaceName   string
	oui             *OUIDatabase
	arpScanner      *ARPScanner
	ndpScanner      *NDPScanner
	protoManager    *Manager
	netbiosResolver *NetBIOSResolver
	mdnsResolver    *MDNSResolver // Active mDNS resolution
	mdnsListener    *MDNSListener // Passive mDNS capture
	mu              sync.RWMutex
	devices         map[string]*DiscoveredDevice // Key by MAC
	lastScan        time.Time
	scanning        bool
	nameResolution  bool           // Enable NetBIOS/mDNS name resolution
	deviceTTL       time.Duration  // How long to keep stale devices (fixes #829)
	nameResWg       sync.WaitGroup // Track name resolution goroutines (fixes #836)
	dbWriter        DBDeviceWriter // Database writer for persistence
}

// loadOUIDatabase loads the OUI database based on configuration.
// Uses early returns to minimize nesting complexity.
func loadOUIDatabase(oui *OUIDatabase, ouiPath string, ouiMaxAge time.Duration) {
	// No path provided: try standard locations silently
	if ouiPath == "" {
		_ = oui.TryLoadIEEEFile()
		return
	}

	// Path provided but no auto-update: just load from file
	if ouiMaxAge == 0 {
		loadOUIFromFile(oui, ouiPath)
		return
	}

	// Auto-update enabled: download if needed, then load
	loadOUIWithAutoUpdate(oui, ouiPath, ouiMaxAge)
}

// loadOUIFromFile loads OUI from a specific file path with fallback.
func loadOUIFromFile(oui *OUIDatabase, ouiPath string) {
	if err := oui.LoadFromIEEEFormat(ouiPath); err != nil {
		logging.GetLogger().Warn("Failed to load OUI from file", "path", ouiPath, "error", err)
		if loadErr := oui.TryLoadIEEEFile(); loadErr != nil {
			logging.GetLogger().Warn("Failed to load IEEE OUI file", "error", loadErr)
		}
		return
	}
	logging.GetLogger().Info("OUI database loaded from file", "path", ouiPath, "entries", oui.Count())
}

// loadOUIWithAutoUpdate updates OUI database if needed, then loads it.
func loadOUIWithAutoUpdate(oui *OUIDatabase, ouiPath string, ouiMaxAge time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), ouiUpdateTimeoutMinutes*time.Minute)
	defer cancel()

	if err := oui.UpdateIfNeeded(ctx, ouiPath, ouiMaxAge); err != nil {
		logging.GetLogger().Warn("Failed to update OUI database", "error", err)
		if loadErr := oui.TryLoadIEEEFile(); loadErr != nil {
			logging.GetLogger().Warn("Failed to load IEEE OUI file", "error", loadErr)
		}
		return
	}
	logging.GetLogger().Info("OUI database loaded", "entries", oui.Count())
}

// NewDeviceDiscovery creates a new device discovery aggregator.
func NewDeviceDiscovery(interfaceName string) *DeviceDiscovery {
	return NewDeviceDiscoveryWithOUI(interfaceName, "", 0)
}

// NewDeviceDiscoveryWithOUI creates a new device discovery aggregator with OUI configuration.
// ouiPath specifies the path to store/load the OUI database file.
// ouiMaxAge specifies how old the file can be before auto-downloading (0 = never auto-update).
func NewDeviceDiscoveryWithOUI(
	interfaceName, ouiPath string,
	ouiMaxAge time.Duration,
) *DeviceDiscovery {
	oui := NewOUIDatabase()
	loadOUIDatabase(oui, ouiPath, ouiMaxAge)

	return &DeviceDiscovery{
		interfaceName:   interfaceName,
		oui:             oui,
		arpScanner:      NewARPScanner(interfaceName, oui),
		ndpScanner:      NewNDPScanner(interfaceName),
		protoManager:    NewManager(interfaceName),
		netbiosResolver: NewNetBIOSResolver(),
		mdnsResolver:    NewMDNSResolver(interfaceName),
		mdnsListener:    NewMDNSListener(interfaceName),
		devices:         make(map[string]*DiscoveredDevice),
		nameResolution:  true,                       // Enabled by default
		deviceTTL:       deviceTTLHours * time.Hour, // Default: expire stale devices after 24h (fixes #829)
	}
}

// Start begins background protocol captures.
func (d *DeviceDiscovery) Start() error {
	// Start NDP scanner for IPv6 discovery
	if err := d.ndpScanner.Start(); err != nil {
		logging.GetLogger().Warn("Failed to start NDP scanner", "error", err)
		// Continue even if NDP fails (may be on macOS or no IPv6)
	}

	// Start mDNS listener for passive name discovery
	if d.nameResolution {
		if err := d.mdnsListener.Start(); err != nil {
			logging.GetLogger().Warn("Failed to start mDNS listener", "error", err)
		}
	}

	// Start protocol manager (LLDP/CDP/EDP captures)
	// This may fail without root/CAP_NET_RAW, but we continue with other features
	if err := d.protoManager.Start(); err != nil {
		logging.GetLogger().
			Warn("Failed to start protocol manager (passive discovery disabled)", "error", err)

		// Return nil to allow ARP scanning, port scanning, and profiling to continue
	}

	return nil
}

// Stop stops all discovery.
func (d *DeviceDiscovery) Stop() {
	// Wait for name resolution goroutines to complete (fixes #836)
	d.nameResWg.Wait()

	_ = d.ndpScanner.Stop()
	_ = d.arpScanner.Close() // Close ICMP pinger (fixes #818)
	d.mdnsListener.Stop()
	d.protoManager.Stop()
}

// SetDBWriter sets the database writer for device persistence.
// Once set, discovered devices will be persisted to the database after each scan.
func (d *DeviceDiscovery) SetDBWriter(w DBDeviceWriter) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dbWriter = w
}

// GetInterfaceName returns the current interface name.
func (d *DeviceDiscovery) GetInterfaceName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.interfaceName
}

// SetInterface updates the interface for all discovery methods.
// Validates the interface exists before updating. (fixes #840).
func (d *DeviceDiscovery) SetInterface(name string) error {
	// Validate interface exists before updating any components (fixes #840)
	if _, err := net.InterfaceByName(name); err != nil {
		return fmt.Errorf("invalid interface %s: %w", name, err)
	}

	d.mu.Lock()
	d.interfaceName = name
	d.mu.Unlock()

	// Update all sub-components that can accept interface changes (fixes #840)
	d.arpScanner.SetInterface(name)

	// Note: NDPScanner, MDNSResolver, MDNSListener, and NetBIOSResolver
	// don't have SetInterface methods - they're bound to the original interface.
	// For a complete interface change, Stop() and recreate the DeviceDiscovery.

	return d.protoManager.SetInterface(name)
}

// SetAdditionalSubnets configures extra subnets to scan.
func (d *DeviceDiscovery) SetAdditionalSubnets(cidrs []string) error {
	return d.arpScanner.SetAdditionalSubnets(cidrs)
}

// GetAdditionalSubnets returns the configured additional subnets.
func (d *DeviceDiscovery) GetAdditionalSubnets() []string {
	return d.arpScanner.GetAdditionalSubnets()
}

// ErrScanInProgress indicates a scan was requested while one is already running.
// Callers should check for this specific error to distinguish between "scan completed
// successfully" and "scan was skipped because one is already in progress".
var ErrScanInProgress = errors.New("scan already in progress")

// Scan performs an active network scan and aggregates results.
// Returns ErrScanInProgress if a scan is already running.
func (d *DeviceDiscovery) Scan(ctx context.Context) error {
	d.mu.Lock()
	if d.scanning {
		d.mu.Unlock()
		return ErrScanInProgress
	}
	d.scanning = true
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		d.scanning = false
		d.lastScan = time.Now()
		d.mu.Unlock()
	}()

	// Perform ARP scan
	if err := d.arpScanner.Scan(ctx); err != nil {
		return err
	}

	// Aggregate results
	d.aggregateResults()

	// Trigger name resolution in background (NetBIOS for Windows, mDNS for Apple/Linux)
	// Track with WaitGroup so Stop() can wait for completion (fixes #836)
	if d.nameResolution {
		d.nameResWg.Add(nameResGoroutineCount)
		go func() {
			defer d.nameResWg.Done()
			d.ResolveNetBIOSNames(ctx)
		}()
		go func() {
			defer d.nameResWg.Done()
			d.ResolveMDNSNames(ctx)
		}()
	}

	return nil
}

// aggregateResults combines ARP scan results with protocol discovery.
func (d *DeviceDiscovery) aggregateResults() {
	d.mu.Lock()
	d.mergeARPResults()
	d.mergeLLDPResults()
	d.mergeCDPResults()
	d.mergeEDPResults()
	d.mergeNDPResults()
	d.mergeMDNSNames()
	d.expireStaleDevices() // Remove old devices to prevent unbounded growth (fixes #829)
	d.detectDuplicateIPs()
	d.ensureVendorInfo()
	d.computeDisplayNames()

	// Copy devices for persistence (must be done under lock)
	var deviceList []*DiscoveredDevice
	dbWriter := d.dbWriter
	if dbWriter != nil {
		deviceList = make([]*DiscoveredDevice, 0, len(d.devices))
		for _, device := range d.devices {
			deviceList = append(deviceList, device)
		}
	}
	d.mu.Unlock()

	// Persist devices outside of lock to avoid holding it during I/O
	if dbWriter != nil && len(deviceList) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), dbPersistTimeoutSeconds*time.Second)
		defer cancel()
		if err := dbWriter.PersistDevices(ctx, deviceList); err != nil {
			logging.GetLogger().
				Warn("Failed to persist devices to database", "error", err, "count", len(deviceList))
		}
	}
}

// expireStaleDevices removes devices that haven't been seen within deviceTTL.
// This prevents unbounded memory growth when devices leave the network.
// Must be called with d.mu held. (fixes #829).
func (d *DeviceDiscovery) expireStaleDevices() {
	if d.deviceTTL <= 0 {
		return // TTL disabled
	}

	cutoff := time.Now().Add(-d.deviceTTL)
	expired := 0
	for key, device := range d.devices {
		if device.LastSeen.Before(cutoff) {
			delete(d.devices, key)
			expired++
		}
	}
	if expired > 0 {
		logging.GetLogger().Info("Expired stale devices", "count", expired, "ttl", d.deviceTTL)
	}
}

// mergeARPResults merges ARP scan entries into devices. Must be called with mu held.
func (d *DeviceDiscovery) mergeARPResults() {
	for _, arp := range d.arpScanner.GetEntries() {
		mac := arp.MAC
		// Use IP as key for PING_ONLY entries (no MAC available)
		key := mac
		if key == "" {
			key = "ip:" + arp.IP
		}

		device, exists := d.devices[key]
		if !exists {
			device = &DiscoveredDevice{
				MAC:             mac,
				DiscoveryMethod: []Method{},
			}
			d.devices[key] = device
		}

		device.IP = arp.IP
		device.Vendor = arp.Vendor
		device.TTL = arp.TTL
		device.OSGuess = arp.OSGuess
		device.LastSeen = arp.LastSeen
		device.IsLocal = arp.IsLocal
		if arp.Hostname != "" {
			device.Hostname = arp.Hostname
		}

		if mac != "" {
			if !containsMethod(device.DiscoveryMethod, MethodARP) {
				device.DiscoveryMethod = append(device.DiscoveryMethod, MethodARP)
			}
		} else {
			if !containsMethod(device.DiscoveryMethod, MethodPING) {
				device.DiscoveryMethod = append(device.DiscoveryMethod, MethodPING)
			}
		}
	}
}

// mergeLLDPResults merges LLDP neighbor data into devices. Must be called with mu held.
//

func (d *DeviceDiscovery) mergeLLDPResults() {
	for _, lldp := range d.protoManager.GetLLDPNeighbors() {
		mac := normalizeMac(lldp.SourceMAC)
		device := d.getOrCreateDevice(mac)

		device.LLDPInfo = &LLDPDeviceInfo{
			ChassisID:         lldp.ChassisID,
			PortID:            lldp.PortID,
			PortDescription:   lldp.PortDescription,
			SystemName:        lldp.SystemName,
			SystemDescription: lldp.SystemDescription,
			Capabilities:      lldp.SystemCapabilities,
			ManagementAddress: lldp.ManagementAddress,
		}

		if device.Hostname == "" && lldp.SystemName != "" {
			device.Hostname = lldp.SystemName
		}
		if device.IP == "" && lldp.ManagementAddress != "" {
			device.IP = lldp.ManagementAddress
		}

		device.LastSeen = lldp.LastSeen
		if !containsMethod(device.DiscoveryMethod, MethodLLDP) {
			device.DiscoveryMethod = append(device.DiscoveryMethod, MethodLLDP)
		}
	}
}

// mergeCDPResults merges CDP neighbor data into devices. Must be called with mu held.
//

func (d *DeviceDiscovery) mergeCDPResults() {
	for _, cdp := range d.protoManager.GetCDPNeighbors() {
		mac := normalizeMac(cdp.SourceMAC)
		device := d.getOrCreateDevice(mac)

		device.CDPInfo = &CDPDeviceInfo{
			DeviceID:          cdp.DeviceID,
			PortID:            cdp.PortID,
			Platform:          cdp.Platform,
			SoftwareVersion:   cdp.SoftwareVersion,
			Capabilities:      cdp.Capabilities,
			ManagementAddress: cdp.ManagementAddress,
			NativeVLAN:        cdp.NativeVLAN,
		}

		if device.Hostname == "" && cdp.DeviceID != "" {
			device.Hostname = cdp.DeviceID
		}
		if device.IP == "" && cdp.ManagementAddress != "" {
			device.IP = cdp.ManagementAddress
		}

		device.LastSeen = cdp.LastSeen
		if !containsMethod(device.DiscoveryMethod, MethodCDP) {
			device.DiscoveryMethod = append(device.DiscoveryMethod, MethodCDP)
		}
	}
}

// mergeEDPResults merges EDP neighbor data into devices. Must be called with mu held.
func (d *DeviceDiscovery) mergeEDPResults() {
	for _, edp := range d.protoManager.GetEDPNeighbors() {
		mac := normalizeMac(edp.SourceMAC)
		device := d.getOrCreateDevice(mac)

		device.EDPInfo = &EDPDeviceInfo{
			DeviceID:        edp.DeviceID,
			DisplayName:     edp.DisplayName,
			PortID:          edp.PortID,
			Platform:        edp.Platform,
			SoftwareVersion: edp.SoftwareVersion,
			VLAN:            edp.VLAN,
		}

		hostname := edp.DisplayName
		if hostname == "" {
			hostname = edp.DeviceID
		}
		if device.Hostname == "" && hostname != "" {
			device.Hostname = hostname
		}
		if device.IP == "" && edp.ManagementAddress != "" {
			device.IP = edp.ManagementAddress
		}

		device.LastSeen = edp.LastSeen
		if !containsMethod(device.DiscoveryMethod, MethodEDP) {
			device.DiscoveryMethod = append(device.DiscoveryMethod, MethodEDP)
		}
	}
}

// mergeNDPResults merges NDP neighbor data (IPv6) into devices. Must be called with mu held.
func (d *DeviceDiscovery) mergeNDPResults() {
	for _, ndp := range d.ndpScanner.GetNeighbors() {
		mac := normalizeMac(ndp.MAC)
		device := d.getOrCreateDevice(mac)

		d.mergeNDPNeighborIntoDevice(device, *ndp)
	}
}

// mergeNDPNeighborIntoDevice updates a device with NDP neighbor information.
func (d *DeviceDiscovery) mergeNDPNeighborIntoDevice(device *DiscoveredDevice, ndp NDPNeighbor) {
	updateDeviceIPv6Addresses(device, ndp.IPv6)

	if ndp.IsRouter {
		device.IsRouter = true
	}

	updateDeviceNDPInfo(device, ndp)

	if ndp.LastSeen.After(device.LastSeen) {
		device.LastSeen = ndp.LastSeen
	}

	if !containsMethod(device.DiscoveryMethod, MethodNDP) {
		device.DiscoveryMethod = append(device.DiscoveryMethod, MethodNDP)
	}
}

// updateDeviceIPv6Addresses adds an IPv6 address to the device if valid and not already present.
func updateDeviceIPv6Addresses(device *DiscoveredDevice, ipv6 string) {
	if ipv6 == "" {
		return
	}

	if device.IPv6Address == "" {
		device.IPv6Address = ipv6
	}

	// Fixes #884: Limit IPv6 addresses to prevent unbounded growth
	if len(device.IPv6Addresses) < maxIPv6AddressesPerDevice {
		if !containsIPv6(device.IPv6Addresses, ipv6) {
			device.IPv6Addresses = append(device.IPv6Addresses, ipv6)
		}
	}
}

// updateDeviceNDPInfo creates or updates the NDPInfo on a device.
func updateDeviceNDPInfo(device *DiscoveredDevice, ndp NDPNeighbor) {
	if device.NDPInfo == nil {
		device.NDPInfo = &NDPDeviceInfo{
			LinkLayerAddress: ndp.MAC,
			IsRouter:         ndp.IsRouter,
		}
	} else {
		device.NDPInfo.IsRouter = ndp.IsRouter
	}
}

// detectDuplicateIPs flags devices that share the same IP address. Must be called with mu held.
func (d *DeviceDiscovery) detectDuplicateIPs() {
	ipToMACs := d.buildIPToMACsMap()
	d.flagDuplicateIPDevices(ipToMACs)
}

// buildIPToMACsMap creates a mapping of IP addresses to all MACs that have that IP.
func (d *DeviceDiscovery) buildIPToMACsMap() map[string][]string {
	ipToMACs := make(map[string][]string)
	for _, device := range d.devices {
		if device.IP != "" && device.MAC != "" {
			ipToMACs[device.IP] = append(ipToMACs[device.IP], device.MAC)
		}
	}
	return ipToMACs
}

// flagDuplicateIPDevices marks devices with duplicate IPs and populates DuplicateMACs.
func (d *DeviceDiscovery) flagDuplicateIPDevices(ipToMACs map[string][]string) {
	for ip, macs := range ipToMACs {
		if len(macs) <= 1 {
			continue
		}
		d.markDevicesWithDuplicateIP(ip, macs)
	}
}

// markDevicesWithDuplicateIP flags all devices with the given IP and sets their DuplicateMACs.
func (d *DeviceDiscovery) markDevicesWithDuplicateIP(ip string, macs []string) {
	for _, device := range d.devices {
		if device.IP != ip || device.MAC == "" {
			continue
		}
		device.HasDuplicateIP = true
		device.DuplicateMACs = collectOtherMACs(macs, device.MAC)
	}
}

// collectOtherMACs returns all MACs from the slice except the excluded one.
func collectOtherMACs(macs []string, excludeMAC string) []string {
	others := make([]string, 0, len(macs)-1)
	for _, mac := range macs {
		if mac != excludeMAC {
			others = append(others, mac)
		}
	}
	return others
}

// isLocallyAdministeredMAC checks if a MAC address is locally administered.
// LAAs have the second-least-significant bit of the first octet set to 1.
// These are typically used by VMs, containers, or MAC randomization features.
func isLocallyAdministeredMAC(mac string) bool {
	if len(mac) < macOctetMinLen {
		return false
	}
	// Parse first octet (handles both AA:BB:CC and AA-BB-CC formats)
	firstOctet := mac[:macOctetMinLen]
	var b byte
	for i := range macOctetMinLen {
		c := firstOctet[i]
		b <<= 4
		switch {
		case c >= '0' && c <= '9':
			b |= c - '0'
		case c >= 'A' && c <= 'F':
			b |= c - 'A' + hexLetterOffset
		case c >= 'a' && c <= 'f':
			b |= c - 'a' + hexLetterOffset
		default:
			return false
		}
	}
	// Check the locally administered bit (second bit from right of first octet)
	return (b & localAdminBit) != 0
}

// ensureVendorInfo populates vendor information for devices missing it. Must be called with mu held.
func (d *DeviceDiscovery) ensureVendorInfo() {
	if d.oui == nil {
		logging.GetLogger().Warn("OUI database not available for vendor lookup")
		return
	}

	vendorCount := 0
	laaCount := 0
	for _, device := range d.devices {
		if device.Vendor == "" && device.MAC != "" {
			// Check if this is a locally administered address
			if isLocallyAdministeredMAC(device.MAC) {
				device.Vendor = "LAA"
				laaCount++
				continue
			}
			vendor := d.oui.LookupWithDefault(device.MAC, "Unknown")
			device.Vendor = vendor
			if vendor != "Unknown" {
				vendorCount++
			}
		}
	}
	logging.GetLogger().Info("Vendor info populated",
		"total_devices", len(d.devices),
		"vendors_found", vendorCount,
		"private_macs", laaCount,
		"oui_entries", d.oui.Count())
}

// mergeMDNSNames merges passively captured mDNS names into devices. Must be called with mu held.
func (d *DeviceDiscovery) mergeMDNSNames() {
	if d.mdnsListener == nil {
		return
	}

	mdnsNames := d.mdnsListener.GetNames()
	for _, device := range d.devices {
		if device.IP != "" && device.MDNSName == "" {
			if name, ok := mdnsNames[device.IP]; ok {
				device.MDNSName = name
				if !containsMethod(device.DiscoveryMethod, MethodMDNS) {
					device.DiscoveryMethod = append(device.DiscoveryMethod, MethodMDNS)
				}
			}
		}
	}
}

// computeDisplayNames computes the best display name for all devices. Must be called with mu held.
func (d *DeviceDiscovery) computeDisplayNames() {
	for _, device := range d.devices {
		device.DisplayName = device.ComputeDisplayName()
	}
}

// getOrCreateDevice returns an existing device or creates a new one.
func (d *DeviceDiscovery) getOrCreateDevice(mac string) *DiscoveredDevice {
	device, exists := d.devices[mac]
	if !exists {
		device = &DiscoveredDevice{
			MAC:             mac,
			DiscoveryMethod: []Method{},
		}
		d.devices[mac] = device
	}
	return device
}

// containsMethod checks if a method is in the slice.
func containsMethod(methods []Method, method Method) bool {
	return slices.Contains(methods, method)
}

// containsIPv6 checks if an IPv6 address is in the slice.
func containsIPv6(addresses []string, addr string) bool {
	return slices.Contains(addresses, addr)
}

// maxIPv6AddressesPerDevice limits IPv6 address accumulation to prevent
// unbounded memory growth from devices with many addresses (fixes #884).
const maxIPv6AddressesPerDevice = 16

// GetDevices returns all discovered devices.
// Returns copies to prevent external mutation of internal state.
func (d *DeviceDiscovery) GetDevices() []*DiscoveredDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	devices := make([]*DiscoveredDevice, 0, len(d.devices))
	for _, device := range d.devices {
		devices = append(devices, copyDevice(device))
	}
	return devices
}

// GetDevice returns a specific device by MAC.
// Returns a copy to prevent external mutation of internal state.
func (d *DeviceDiscovery) GetDevice(mac string) *DiscoveredDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()
	device := d.devices[normalizeMac(mac)]
	if device == nil {
		return nil
	}
	return copyDevice(device)
}

// GetDeviceByIP returns a device by IP address.
// Returns a copy to prevent external mutation of internal state.
func (d *DeviceDiscovery) GetDeviceByIP(ip string) *DiscoveredDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, device := range d.devices {
		if device.IP == ip {
			return copyDevice(device)
		}
	}
	return nil
}

// copyDevice creates a shallow copy with deep-copied slices to prevent mutation.
func copyDevice(device *DiscoveredDevice) *DiscoveredDevice {
	deviceCopy := *device

	// Deep copy top-level slices
	copyDeviceSlices(&deviceCopy, device)

	// Deep copy protocol-specific info pointers
	copyDeviceProtocolInfo(&deviceCopy, device)

	// Deep copy complex nested structures
	deviceCopy.Profile = copyDeviceProfile(device.Profile)
	deviceCopy.SNMPData = copySNMPData(device.SNMPData)
	deviceCopy.Vulnerabilities = copyVulnerabilities(device.Vulnerabilities)

	// Deep copy unified discovery presence info
	deviceCopy.WiFiPresence = copyWiFiPresence(device.WiFiPresence)
	deviceCopy.BluetoothPresence = copyBluetoothPresence(device.BluetoothPresence)

	return &deviceCopy
}

// copyWiFiPresence creates a deep copy of WiFiPresence.
func copyWiFiPresence(presence *WiFiPresence) *WiFiPresence {
	if presence == nil {
		return nil
	}
	presenceCopy := *presence
	return &presenceCopy
}

// copyBluetoothPresence creates a deep copy of BluetoothPresence.
func copyBluetoothPresence(presence *BluetoothPresence) *BluetoothPresence {
	if presence == nil {
		return nil
	}
	presenceCopy := *presence
	if presence.Services != nil {
		presenceCopy.Services = make([]string, len(presence.Services))
		copy(presenceCopy.Services, presence.Services)
	}
	return &presenceCopy
}

// copyDeviceSlices deep copies the top-level slices of a device.
func copyDeviceSlices(dst *DiscoveredDevice, src *DiscoveredDevice) {
	if src.DiscoveryMethod != nil {
		dst.DiscoveryMethod = make([]Method, len(src.DiscoveryMethod))
		copy(dst.DiscoveryMethod, src.DiscoveryMethod)
	}
	if src.IPv6Addresses != nil {
		dst.IPv6Addresses = make([]string, len(src.IPv6Addresses))
		copy(dst.IPv6Addresses, src.IPv6Addresses)
	}
	if src.DuplicateMACs != nil {
		dst.DuplicateMACs = make([]string, len(src.DuplicateMACs))
		copy(dst.DuplicateMACs, src.DuplicateMACs)
	}
	if src.ConnectionTypes != nil {
		dst.ConnectionTypes = make([]ConnectionType, len(src.ConnectionTypes))
		copy(dst.ConnectionTypes, src.ConnectionTypes)
	}
}

// copyDeviceProtocolInfo deep copies the protocol-specific info pointers.
func copyDeviceProtocolInfo(dst *DiscoveredDevice, src *DiscoveredDevice) {
	dst.LLDPInfo = copyLLDPInfo(src.LLDPInfo)
	dst.CDPInfo = copyCDPInfo(src.CDPInfo)
	dst.EDPInfo = copyEDPInfo(src.EDPInfo)
	dst.NDPInfo = copyNDPInfo(src.NDPInfo)
}

// copyLLDPInfo creates a deep copy of LLDPDeviceInfo.
func copyLLDPInfo(info *LLDPDeviceInfo) *LLDPDeviceInfo {
	if info == nil {
		return nil
	}
	infoCopy := *info
	if info.Capabilities != nil {
		infoCopy.Capabilities = make([]string, len(info.Capabilities))
		copy(infoCopy.Capabilities, info.Capabilities)
	}
	return &infoCopy
}

// copyCDPInfo creates a deep copy of CDPDeviceInfo.
func copyCDPInfo(info *CDPDeviceInfo) *CDPDeviceInfo {
	if info == nil {
		return nil
	}
	infoCopy := *info
	if info.Capabilities != nil {
		infoCopy.Capabilities = make([]string, len(info.Capabilities))
		copy(infoCopy.Capabilities, info.Capabilities)
	}
	return &infoCopy
}

// copyEDPInfo creates a deep copy of EDPDeviceInfo.
func copyEDPInfo(info *EDPDeviceInfo) *EDPDeviceInfo {
	if info == nil {
		return nil
	}
	infoCopy := *info
	return &infoCopy
}

// copyNDPInfo creates a deep copy of NDPDeviceInfo.
func copyNDPInfo(info *NDPDeviceInfo) *NDPDeviceInfo {
	if info == nil {
		return nil
	}
	infoCopy := *info
	return &infoCopy
}

// copyDeviceProfile creates a deep copy of DeviceProfile.
func copyDeviceProfile(profile *DeviceProfile) *DeviceProfile {
	if profile == nil {
		return nil
	}
	profileCopy := *profile
	if profile.OpenPorts != nil {
		profileCopy.OpenPorts = make([]OpenPort, len(profile.OpenPorts))
		copy(profileCopy.OpenPorts, profile.OpenPorts)
	}
	if profile.MDNSServices != nil {
		profileCopy.MDNSServices = make([]MDNSService, len(profile.MDNSServices))
		copy(profileCopy.MDNSServices, profile.MDNSServices)
	}
	if profile.DeviceIcons != nil {
		profileCopy.DeviceIcons = make([]string, len(profile.DeviceIcons))
		copy(profileCopy.DeviceIcons, profile.DeviceIcons)
	}
	return &profileCopy
}

// copySNMPData creates a deep copy of SNMPFullData.
func copySNMPData(data *SNMPFullData) *SNMPFullData {
	if data == nil {
		return nil
	}
	snmpCopy := *data
	copySNMPDataSlices(&snmpCopy, data)
	return &snmpCopy
}

// copySNMPDataSlices deep copies all slices within SNMPFullData.
func copySNMPDataSlices(dst *SNMPFullData, src *SNMPFullData) {
	if src.Interfaces != nil {
		dst.Interfaces = make([]SNMPInterface, len(src.Interfaces))
		copy(dst.Interfaces, src.Interfaces)
	}
	if src.IPAddresses != nil {
		dst.IPAddresses = make([]SNMPIPAddress, len(src.IPAddresses))
		copy(dst.IPAddresses, src.IPAddresses)
	}
	if src.VLANs != nil {
		dst.VLANs = make([]SNMPVLAN, len(src.VLANs))
		copy(dst.VLANs, src.VLANs)
	}
	if src.MACTable != nil {
		dst.MACTable = make([]SNMPMACEntry, len(src.MACTable))
		copy(dst.MACTable, src.MACTable)
	}
	if src.Inventory != nil {
		dst.Inventory = make([]SNMPEntity, len(src.Inventory))
		copy(dst.Inventory, src.Inventory)
	}
	if src.LLDPNeighbors != nil {
		dst.LLDPNeighbors = make([]SNMPLLDPNeighbor, len(src.LLDPNeighbors))
		copy(dst.LLDPNeighbors, src.LLDPNeighbors)
	}
}

// copyVulnerabilities creates a deep copy of DeviceVulnerabilities.
func copyVulnerabilities(vuln *DeviceVulnerabilities) *DeviceVulnerabilities {
	if vuln == nil {
		return nil
	}
	vulnCopy := *vuln
	if vuln.Vulnerabilities != nil {
		vulnCopy.Vulnerabilities = make(
			[]Vulnerability,
			len(vuln.Vulnerabilities),
		)
		copy(vulnCopy.Vulnerabilities, vuln.Vulnerabilities)
	}
	return &vulnCopy
}

// Count returns the number of discovered devices.
func (d *DeviceDiscovery) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.devices)
}

// IsScanning returns true if a scan is in progress.
func (d *DeviceDiscovery) IsScanning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.scanning
}

// LastScan returns the time of the last completed scan.
func (d *DeviceDiscovery) LastScan() time.Time {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastScan
}

// GetSubnetInfo returns network information.
func (d *DeviceDiscovery) GetSubnetInfo() (string, string) {
	return d.arpScanner.GetSubnetInfo()
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

// GetStatus returns the current discovery status.
func (d *DeviceDiscovery) GetStatus() *Status {
	d.mu.RLock()
	defer d.mu.RUnlock()

	subnet, localIP := d.arpScanner.GetSubnetInfo()

	return &Status{
		Scanning:    d.scanning,
		DeviceCount: len(d.devices),
		LastScan:    d.lastScan,
		Subnet:      subnet,
		LocalIP:     localIP,
		Interface:   d.interfaceName,
	}
}

// ClearDevices clears all discovered devices.
func (d *DeviceDiscovery) ClearDevices() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devices = make(map[string]*DiscoveredDevice)
}

// SetNameResolution enables or disables NetBIOS/mDNS name resolution.
func (d *DeviceDiscovery) SetNameResolution(enabled bool) {
	d.mu.Lock()
	d.nameResolution = enabled
	d.mu.Unlock()
}

// GetOUIDatabase returns the OUI database for vendor lookups.
// This allows other discovery components (Bluetooth, WiFi) to share the same OUI data.
func (d *DeviceDiscovery) GetOUIDatabase() *OUIDatabase {
	return d.oui
}

// ResolveNetBIOSNames triggers active NetBIOS name resolution for discovered devices.
// This is done asynchronously to avoid blocking the scan process.
func (d *DeviceDiscovery) ResolveNetBIOSNames(ctx context.Context) {
	if d.netbiosResolver == nil || !d.nameResolution {
		return
	}

	d.mu.RLock()
	var ips []string
	deviceMap := make(map[string]*DiscoveredDevice)
	for _, device := range d.devices {
		// Resolve for all devices without a NetBIOS name (not just local)
		if device.IP != "" && device.NetBIOSName == "" {
			ips = append(ips, device.IP)
			deviceMap[device.IP] = device
		}
	}
	d.mu.RUnlock()

	if len(ips) == 0 {
		return
	}

	logging.GetLogger().DebugContext(ctx, "NetBIOS: resolving names", "count", len(ips))

	// Resolve in batch
	results := d.netbiosResolver.ResolveBatch(ctx, ips)

	// Update devices with resolved names
	// Fixes #987: Re-check device existence under write lock to handle concurrent removal
	d.mu.Lock()
	for _, result := range results {
		if result.Err == nil && result.Name != "" {
			// Re-lookup device in d.devices instead of using stale deviceMap pointer
			if device, ok := d.devices[result.IP]; ok {
				device.NetBIOSName = result.Name
				device.DisplayName = device.ComputeDisplayName()
				logging.GetLogger().
					DebugContext(ctx, "NetBIOS: resolved name", "ip", result.IP, "name", result.Name)
			}
		}
	}
	d.mu.Unlock()
}

// ResolveMDNSNames triggers active mDNS name resolution for discovered devices.
// This queries devices directly for their .local hostname (Bonjour/Avahi).
func (d *DeviceDiscovery) ResolveMDNSNames(ctx context.Context) {
	if d.mdnsResolver == nil || !d.nameResolution {
		return
	}

	d.mu.RLock()
	var ips []string
	deviceMap := make(map[string]*DiscoveredDevice)
	for _, device := range d.devices {
		// Resolve for all devices without an mDNS name
		if device.IP != "" && device.MDNSName == "" {
			ips = append(ips, device.IP)
			deviceMap[device.IP] = device
		}
	}
	d.mu.RUnlock()

	if len(ips) == 0 {
		return
	}

	logging.GetLogger().DebugContext(ctx, "mDNS: resolving names", "count", len(ips))

	// Resolve in batch
	results := d.mdnsResolver.ResolveBatch(ctx, ips)

	// Update devices with resolved names
	// Fixes #987: Re-check device existence under write lock to handle concurrent removal
	d.mu.Lock()
	for _, result := range results {
		if result.Err == nil && result.Name != "" {
			// Re-lookup device in d.devices instead of using stale deviceMap pointer
			if device, ok := d.devices[result.IP]; ok {
				device.MDNSName = result.Name
				device.DisplayName = device.ComputeDisplayName()
				if !containsMethod(device.DiscoveryMethod, MethodMDNS) {
					device.DiscoveryMethod = append(device.DiscoveryMethod, MethodMDNS)
				}
				logging.GetLogger().
					DebugContext(ctx, "mDNS: resolved name", "ip", result.IP, "name", result.Name)
			}
		}
	}
	d.mu.Unlock()
}

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
