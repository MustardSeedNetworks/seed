// Package discovery implements multi-protocol network device discovery.
// It manages the aggregation of device information discovered through various protocols
// including ARP, NDP, LLDP, CDP, EDP, mDNS, and ICMP ping. The package maintains a synchronized
// view of all discovered devices and their protocol-specific metadata.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"sync"
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

// NewDeviceDiscovery creates a new device discovery aggregator.
func NewDeviceDiscovery(interfaceName string) *DeviceDiscovery {
	return NewDeviceDiscoveryWithOUI(interfaceName, "", 0)
}

// NewDeviceDiscoveryWithOUI creates a new device discovery aggregator with OUI configuration.
// ouiPath specifies the path to store/load the OUI database file.
// ouiMaxAge specifies how old the file can be before auto-downloading (0 = never auto-update).
func NewDeviceDiscoveryWithOUI(interfaceName, ouiPath string, ouiMaxAge time.Duration) *DeviceDiscovery {
	oui := NewOUIDatabase()

	// Try to load/update OUI database
	//nolint:gocritic // ifElseChain: conditions check different combinations, switch not applicable
	if ouiPath != "" && ouiMaxAge > 0 {
		// Auto-update enabled: download if needed, then load
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := oui.UpdateIfNeeded(ctx, ouiPath, ouiMaxAge); err != nil {
			slog.Warn("Failed to update OUI database", "error", err)
			// Try loading from standard locations as fallback
			if loadErr := oui.TryLoadIEEEFile(); loadErr != nil {
				slog.Warn("Failed to load IEEE OUI file", "error", loadErr)
			}
		} else {
			slog.Info("OUI database loaded", "entries", oui.Count())
		}
	} else if ouiPath != "" {
		// Path provided but no auto-update: just load from file
		if err := oui.LoadFromIEEEFormat(ouiPath); err != nil {
			slog.Warn("Failed to load OUI from file", "path", ouiPath, "error", err)
			if loadErr := oui.TryLoadIEEEFile(); loadErr != nil {
				slog.Warn("Failed to load IEEE OUI file", "error", loadErr)
			}
		} else {
			slog.Info("OUI database loaded from file", "path", ouiPath, "entries", oui.Count())
		}
	} else {
		// No path provided: try standard locations silently
		// Embedded OUI database has 200+ common vendors as fallback
		_ = oui.TryLoadIEEEFile()
	}

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
		nameResolution:  true,           // Enabled by default
		deviceTTL:       24 * time.Hour, // Default: expire stale devices after 24h (fixes #829)
	}
}

// Start begins background protocol captures.
func (d *DeviceDiscovery) Start() error {
	// Start NDP scanner for IPv6 discovery
	if err := d.ndpScanner.Start(); err != nil {
		slog.Warn("Failed to start NDP scanner", "error", err)
		// Continue even if NDP fails (may be on macOS or no IPv6)
	}

	// Start mDNS listener for passive name discovery
	if d.nameResolution {
		if err := d.mdnsListener.Start(); err != nil {
			slog.Warn("Failed to start mDNS listener", "error", err)
		}
	}

	// Start protocol manager (LLDP/CDP/EDP captures)
	// This may fail without root/CAP_NET_RAW, but we continue with other features
	if err := d.protoManager.Start(); err != nil {
		slog.Warn("Failed to start protocol manager (passive discovery disabled)", "error", err)
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
		d.nameResWg.Add(2)
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := dbWriter.PersistDevices(ctx, deviceList); err != nil {
			slog.Warn("Failed to persist devices to database", "error", err, "count", len(deviceList))
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
		slog.Info("Expired stale devices", "count", expired, "ttl", d.deviceTTL)
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

		if ndp.IPv6 != "" {
			if device.IPv6Address == "" {
				device.IPv6Address = ndp.IPv6
			}
			// Fixes #884: Limit IPv6 addresses to prevent unbounded growth
			if len(device.IPv6Addresses) < maxIPv6AddressesPerDevice {
				if !containsIPv6(device.IPv6Addresses, ndp.IPv6) {
					device.IPv6Addresses = append(device.IPv6Addresses, ndp.IPv6)
				}
			}
		}

		if ndp.IsRouter {
			device.IsRouter = true
		}

		if device.NDPInfo == nil {
			device.NDPInfo = &NDPDeviceInfo{
				LinkLayerAddress: ndp.MAC,
				IsRouter:         ndp.IsRouter,
			}
		} else {
			device.NDPInfo.IsRouter = ndp.IsRouter
		}

		if ndp.LastSeen.After(device.LastSeen) {
			device.LastSeen = ndp.LastSeen
		}

		if !containsMethod(device.DiscoveryMethod, MethodNDP) {
			device.DiscoveryMethod = append(device.DiscoveryMethod, MethodNDP)
		}
	}
}

// detectDuplicateIPs flags devices that share the same IP address. Must be called with mu held.
func (d *DeviceDiscovery) detectDuplicateIPs() {
	ipToMACs := make(map[string][]string)
	for _, device := range d.devices {
		if device.IP != "" && device.MAC != "" {
			ipToMACs[device.IP] = append(ipToMACs[device.IP], device.MAC)
		}
	}

	for ip, macs := range ipToMACs {
		if len(macs) > 1 {
			for _, device := range d.devices {
				if device.IP == ip && device.MAC != "" {
					device.HasDuplicateIP = true
					device.DuplicateMACs = make([]string, 0, len(macs)-1)
					for _, mac := range macs {
						if mac != device.MAC {
							device.DuplicateMACs = append(device.DuplicateMACs, mac)
						}
					}
				}
			}
		}
	}
}

// isLocallyAdministeredMAC checks if a MAC address is locally administered.
// LAAs have the second-least-significant bit of the first octet set to 1.
// These are typically used by VMs, containers, or MAC randomization features.
func isLocallyAdministeredMAC(mac string) bool {
	if len(mac) < 2 {
		return false
	}
	// Parse first octet (handles both AA:BB:CC and AA-BB-CC formats)
	firstOctet := mac[:2]
	var b byte
	for i := range 2 {
		c := firstOctet[i]
		b <<= 4
		switch {
		case c >= '0' && c <= '9':
			b |= c - '0'
		case c >= 'A' && c <= 'F':
			b |= c - 'A' + 10
		case c >= 'a' && c <= 'f':
			b |= c - 'a' + 10
		default:
			return false
		}
	}
	// Check the locally administered bit (second bit from right of first octet)
	return (b & 0x02) != 0
}

// ensureVendorInfo populates vendor information for devices missing it. Must be called with mu held.
func (d *DeviceDiscovery) ensureVendorInfo() {
	if d.oui == nil {
		slog.Warn("OUI database not available for vendor lookup")
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
	slog.Info("Vendor info populated",
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

	// Deep copy slices that could be mutated
	if device.DiscoveryMethod != nil {
		deviceCopy.DiscoveryMethod = make([]Method, len(device.DiscoveryMethod))
		copy(deviceCopy.DiscoveryMethod, device.DiscoveryMethod)
	}
	if device.IPv6Addresses != nil {
		deviceCopy.IPv6Addresses = make([]string, len(device.IPv6Addresses))
		copy(deviceCopy.IPv6Addresses, device.IPv6Addresses)
	}
	if device.DuplicateMACs != nil {
		deviceCopy.DuplicateMACs = make([]string, len(device.DuplicateMACs))
		copy(deviceCopy.DuplicateMACs, device.DuplicateMACs)
	}

	// Deep copy protocol-specific info pointers
	if device.LLDPInfo != nil {
		infoCopy := *device.LLDPInfo
		if device.LLDPInfo.Capabilities != nil {
			infoCopy.Capabilities = make([]string, len(device.LLDPInfo.Capabilities))
			copy(infoCopy.Capabilities, device.LLDPInfo.Capabilities)
		}
		deviceCopy.LLDPInfo = &infoCopy
	}
	if device.CDPInfo != nil {
		infoCopy := *device.CDPInfo
		if device.CDPInfo.Capabilities != nil {
			infoCopy.Capabilities = make([]string, len(device.CDPInfo.Capabilities))
			copy(infoCopy.Capabilities, device.CDPInfo.Capabilities)
		}
		deviceCopy.CDPInfo = &infoCopy
	}
	if device.EDPInfo != nil {
		infoCopy := *device.EDPInfo
		deviceCopy.EDPInfo = &infoCopy
	}
	if device.NDPInfo != nil {
		infoCopy := *device.NDPInfo
		deviceCopy.NDPInfo = &infoCopy
	}

	// Deep copy profile (contains slices)
	if device.Profile != nil {
		profileCopy := *device.Profile
		if device.Profile.OpenPorts != nil {
			profileCopy.OpenPorts = make([]OpenPort, len(device.Profile.OpenPorts))
			copy(profileCopy.OpenPorts, device.Profile.OpenPorts)
		}
		if device.Profile.MDNSServices != nil {
			profileCopy.MDNSServices = make([]MDNSService, len(device.Profile.MDNSServices))
			copy(profileCopy.MDNSServices, device.Profile.MDNSServices)
		}
		if device.Profile.DeviceIcons != nil {
			profileCopy.DeviceIcons = make([]string, len(device.Profile.DeviceIcons))
			copy(profileCopy.DeviceIcons, device.Profile.DeviceIcons)
		}
		deviceCopy.Profile = &profileCopy
	}

	// Deep copy SNMP data (contains slices)
	if device.SNMPData != nil {
		snmpCopy := *device.SNMPData
		if device.SNMPData.Interfaces != nil {
			snmpCopy.Interfaces = make([]SNMPInterface, len(device.SNMPData.Interfaces))
			copy(snmpCopy.Interfaces, device.SNMPData.Interfaces)
		}
		if device.SNMPData.IPAddresses != nil {
			snmpCopy.IPAddresses = make([]SNMPIPAddress, len(device.SNMPData.IPAddresses))
			copy(snmpCopy.IPAddresses, device.SNMPData.IPAddresses)
		}
		if device.SNMPData.VLANs != nil {
			snmpCopy.VLANs = make([]SNMPVLAN, len(device.SNMPData.VLANs))
			copy(snmpCopy.VLANs, device.SNMPData.VLANs)
		}
		if device.SNMPData.MACTable != nil {
			snmpCopy.MACTable = make([]SNMPMACEntry, len(device.SNMPData.MACTable))
			copy(snmpCopy.MACTable, device.SNMPData.MACTable)
		}
		if device.SNMPData.Inventory != nil {
			snmpCopy.Inventory = make([]SNMPEntity, len(device.SNMPData.Inventory))
			copy(snmpCopy.Inventory, device.SNMPData.Inventory)
		}
		if device.SNMPData.LLDPNeighbors != nil {
			snmpCopy.LLDPNeighbors = make([]SNMPLLDPNeighbor, len(device.SNMPData.LLDPNeighbors))
			copy(snmpCopy.LLDPNeighbors, device.SNMPData.LLDPNeighbors)
		}
		deviceCopy.SNMPData = &snmpCopy
	}

	// Deep copy vulnerabilities (contains slices)
	if device.Vulnerabilities != nil {
		vulnCopy := *device.Vulnerabilities
		if device.Vulnerabilities.Vulnerabilities != nil {
			vulnCopy.Vulnerabilities = make([]Vulnerability, len(device.Vulnerabilities.Vulnerabilities))
			copy(vulnCopy.Vulnerabilities, device.Vulnerabilities.Vulnerabilities)
		}
		deviceCopy.Vulnerabilities = &vulnCopy
	}

	return &deviceCopy
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

	slog.Debug("NetBIOS: resolving names", "count", len(ips))

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
				slog.Debug("NetBIOS: resolved name", "ip", result.IP, "name", result.Name)
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

	slog.Debug("mDNS: resolving names", "count", len(ips))

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
				slog.Debug("mDNS: resolved name", "ip", result.IP, "name", result.Name)
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
