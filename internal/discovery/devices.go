package discovery

import (
	"context"
	"log"
	"sync"
	"time"
)

// DiscoveryMethod indicates how a device was discovered.
type DiscoveryMethod string

const (
	MethodARP  DiscoveryMethod = "arp"
	MethodNDP  DiscoveryMethod = "ndp"  // IPv6 Neighbor Discovery
	MethodLLDP DiscoveryMethod = "lldp"
	MethodCDP  DiscoveryMethod = "cdp"
	MethodEDP  DiscoveryMethod = "edp"
	MethodMDNS DiscoveryMethod = "mdns"
	MethodPING DiscoveryMethod = "ping"
)

// DiscoveredDevice represents a network device with aggregated discovery info.
type DiscoveredDevice struct {
	IP              string            `json:"ip"`              // Primary IPv4 address
	IPv6Address     string            `json:"ipv6,omitempty"`  // Primary IPv6 address
	IPv6Addresses   []string          `json:"ipv6Addresses,omitempty"` // All IPv6 addresses
	MAC             string            `json:"mac"`
	Hostname        string            `json:"hostname,omitempty"`
	Vendor          string            `json:"vendor,omitempty"`
	OSGuess         string            `json:"osGuess,omitempty"`
	TTL             int               `json:"ttl,omitempty"`
	DiscoveryMethod []DiscoveryMethod `json:"discoveryMethod"`
	LastSeen        time.Time         `json:"lastSeen"`
	IsLocal         bool              `json:"isLocal"` // true if device is on local subnet
	IsRouter        bool              `json:"isRouter,omitempty"` // true if detected as IPv6 router via NDP

	// Protocol-specific details (populated if discovered via that protocol)
	LLDPInfo *LLDPDeviceInfo `json:"lldpInfo,omitempty"`
	CDPInfo  *CDPDeviceInfo  `json:"cdpInfo,omitempty"`
	EDPInfo  *EDPDeviceInfo  `json:"edpInfo,omitempty"`
	NDPInfo  *NDPDeviceInfo  `json:"ndpInfo,omitempty"` // IPv6 Neighbor Discovery info

	// Auto-profiling results
	Profile *DeviceProfile `json:"profile,omitempty"`
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
	LinkLayerAddress  string    `json:"linkLayerAddress"` // MAC from NDP
	IsRouter          bool      `json:"isRouter"`         // From Router Advertisement
	ReachableTime     uint32    `json:"reachableTime,omitempty"`     // milliseconds
	RetransTimer      uint32    `json:"retransTimer,omitempty"`      // milliseconds
	Flags             uint8     `json:"flags,omitempty"`             // NDP flags
	LastAdvertisement time.Time `json:"lastAdvertisement,omitempty"` // Last RA received
}

// DeviceDiscovery aggregates device discovery from all sources.
type DeviceDiscovery struct {
	interfaceName string
	oui           *OUIDatabase
	arpScanner    *ARPScanner
	ndpScanner    *NDPScanner
	protoManager  *Manager
	mu            sync.RWMutex
	devices       map[string]*DiscoveredDevice // Key by MAC
	lastScan      time.Time
	scanning      bool
}

// NewDeviceDiscovery creates a new device discovery aggregator.
func NewDeviceDiscovery(interfaceName string) *DeviceDiscovery {
	oui := NewOUIDatabase()
	// Try to load extended OUI file if available
	if err := oui.TryLoadIEEEFile(); err != nil {
		log.Printf("warning: failed to load IEEE OUI file: %v", err)
	}

	return &DeviceDiscovery{
		interfaceName: interfaceName,
		oui:           oui,
		arpScanner:    NewARPScanner(interfaceName, oui),
		ndpScanner:    NewNDPScanner(interfaceName),
		protoManager:  NewManager(interfaceName),
		devices:       make(map[string]*DiscoveredDevice),
	}
}

// Start begins background protocol captures.
func (d *DeviceDiscovery) Start() error {
	// Start NDP scanner for IPv6 discovery
	if err := d.ndpScanner.Start(); err != nil {
		log.Printf("warning: failed to start NDP scanner: %v", err)
		// Continue even if NDP fails (may be on macOS or no IPv6)
	}
	return d.protoManager.Start()
}

// Stop stops all discovery.
func (d *DeviceDiscovery) Stop() {
	d.ndpScanner.Stop()
	d.protoManager.Stop()
}

// SetInterface updates the interface for all discovery methods.
func (d *DeviceDiscovery) SetInterface(name string) error {
	d.mu.Lock()
	d.interfaceName = name
	d.mu.Unlock()

	d.arpScanner.SetInterface(name)
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

// Scan performs an active network scan and aggregates results.
func (d *DeviceDiscovery) Scan(ctx context.Context) error {
	d.mu.Lock()
	if d.scanning {
		d.mu.Unlock()
		return nil // Already scanning
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

	return nil
}

// aggregateResults combines ARP scan results with protocol discovery.
func (d *DeviceDiscovery) aggregateResults() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Start with ARP entries
	for _, arp := range d.arpScanner.GetEntries() {
		mac := arp.MAC
		// Use IP as key for PING_ONLY entries (no MAC available)
		// This prevents all PING_ONLY entries from colliding on empty string key
		key := mac
		if key == "" {
			key = "ip:" + arp.IP
		}

		device, exists := d.devices[key]
		if !exists {
			device = &DiscoveredDevice{
				MAC:             mac,
				DiscoveryMethod: []DiscoveryMethod{},
			}
			d.devices[key] = device
		}

		// Update from ARP data
		device.IP = arp.IP
		device.Vendor = arp.Vendor
		device.TTL = arp.TTL
		device.OSGuess = arp.OSGuess
		device.LastSeen = arp.LastSeen
		device.IsLocal = arp.IsLocal
		if arp.Hostname != "" {
			device.Hostname = arp.Hostname
		}

		// Use correct discovery method based on whether we have MAC (ARP) or not (PING)
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

	// Merge LLDP neighbors
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

		// Use LLDP system name as hostname if not already set
		if device.Hostname == "" && lldp.SystemName != "" {
			device.Hostname = lldp.SystemName
		}

		// Use management address as IP if not already set
		if device.IP == "" && lldp.ManagementAddress != "" {
			device.IP = lldp.ManagementAddress
		}

		device.LastSeen = lldp.LastSeen
		if !containsMethod(device.DiscoveryMethod, MethodLLDP) {
			device.DiscoveryMethod = append(device.DiscoveryMethod, MethodLLDP)
		}
	}

	// Merge CDP neighbors
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

	// Merge EDP neighbors
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

	// Merge NDP neighbors (IPv6)
	for _, ndp := range d.ndpScanner.GetNeighbors() {
		mac := normalizeMac(ndp.MAC)

		// Try to find existing device by MAC
		device := d.getOrCreateDevice(mac)

		// Add IPv6 address to the device
		if ndp.IPv6 != "" {
			// Set primary IPv6 if not set
			if device.IPv6Address == "" {
				device.IPv6Address = ndp.IPv6
			}

			// Add to IPv6 addresses list if not already present
			if !containsIPv6(device.IPv6Addresses, ndp.IPv6) {
				device.IPv6Addresses = append(device.IPv6Addresses, ndp.IPv6)
			}
		}

		// Update router status
		if ndp.IsRouter {
			device.IsRouter = true
		}

		// Create or update NDP info
		if device.NDPInfo == nil {
			device.NDPInfo = &NDPDeviceInfo{
				LinkLayerAddress: ndp.MAC,
				IsRouter:         ndp.IsRouter,
			}
		} else {
			device.NDPInfo.IsRouter = ndp.IsRouter
		}

		// Update last seen
		if ndp.LastSeen.After(device.LastSeen) {
			device.LastSeen = ndp.LastSeen
		}

		// Add NDP discovery method
		if !containsMethod(device.DiscoveryMethod, MethodNDP) {
			device.DiscoveryMethod = append(device.DiscoveryMethod, MethodNDP)
		}
	}

	// Ensure all devices have vendor info
	for _, device := range d.devices {
		if device.Vendor == "" && device.MAC != "" {
			device.Vendor = d.oui.LookupWithDefault(device.MAC, "Unknown")
		}
	}
}

// getOrCreateDevice returns an existing device or creates a new one.
func (d *DeviceDiscovery) getOrCreateDevice(mac string) *DiscoveredDevice {
	device, exists := d.devices[mac]
	if !exists {
		device = &DiscoveredDevice{
			MAC:             mac,
			DiscoveryMethod: []DiscoveryMethod{},
		}
		d.devices[mac] = device
	}
	return device
}

// containsMethod checks if a method is in the slice.
func containsMethod(methods []DiscoveryMethod, method DiscoveryMethod) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

// containsIPv6 checks if an IPv6 address is in the slice.
func containsIPv6(addresses []string, addr string) bool {
	for _, a := range addresses {
		if a == addr {
			return true
		}
	}
	return false
}

// GetDevices returns all discovered devices.
func (d *DeviceDiscovery) GetDevices() []*DiscoveredDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	devices := make([]*DiscoveredDevice, 0, len(d.devices))
	for _, device := range d.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetDevice returns a specific device by MAC.
func (d *DeviceDiscovery) GetDevice(mac string) *DiscoveredDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.devices[normalizeMac(mac)]
}

// GetDeviceByIP returns a device by IP address.
func (d *DeviceDiscovery) GetDeviceByIP(ip string) *DiscoveredDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, device := range d.devices {
		if device.IP == ip {
			return device
		}
	}
	return nil
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
func (d *DeviceDiscovery) GetSubnetInfo() (subnet, localIP string) {
	return d.arpScanner.GetSubnetInfo()
}

// DiscoveryStatus represents the current discovery status.
type DiscoveryStatus struct {
	Scanning    bool      `json:"scanning"`
	DeviceCount int       `json:"deviceCount"`
	LastScan    time.Time `json:"lastScan"`
	Subnet      string    `json:"subnet"`
	LocalIP     string    `json:"localIP"`
	Interface   string    `json:"interface"`
}

// GetStatus returns the current discovery status.
func (d *DeviceDiscovery) GetStatus() *DiscoveryStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	subnet, localIP := d.arpScanner.GetSubnetInfo()

	return &DiscoveryStatus{
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
