package discovery

// registry.go implements the unified device registry.
//
// The DeviceRegistry is the single source of truth for all discovered devices.
// It provides:
// - Thread-safe device storage indexed by MAC address
// - Secondary indexes for fast lookup by IP and hostname
// - Event emission when devices are added, updated, or removed
// - Automatic correlation across discovery sources

import (
	"slices"
	"strings"
	"sync"
	"time"
)

// Registry configuration defaults.
const (
	// registryDeviceTTLHours is the default device TTL in hours.
	registryDeviceTTLHours = 24
	// registryExpireIntervalMinutes is the default expire check interval in minutes.
	registryExpireIntervalMinutes = 5
	// multiConnectionThreshold is the minimum number of connection types for multi-connected.
	multiConnectionThreshold = 2
)

// DeviceRegistry stores and manages all discovered devices.
// It serves as the single source of truth for the discovery system.
type DeviceRegistry struct {
	// Primary index: normalized MAC -> device
	devices map[string]*DiscoveredDevice

	// Secondary indexes for fast lookup
	byIP       map[string]*DiscoveredDevice   // IP -> device
	byHostname map[string]*DiscoveredDevice   // hostname -> device
	byVendor   map[string][]*DiscoveredDevice // vendor -> devices

	// Event bus for change notifications
	eventBus *EventBus

	// Configuration
	config *RegistryConfig

	// Statistics
	stats RegistryStats

	mu sync.RWMutex
}

// RegistryConfig configures the device registry behavior.
type RegistryConfig struct {
	// DeviceTTL is how long before a device is considered stale
	DeviceTTL time.Duration

	// EmitEvents controls whether the registry emits events
	EmitEvents bool

	// MaxDevices limits the number of devices (0 = unlimited)
	MaxDevices int

	// AutoExpire enables automatic device expiration
	AutoExpire bool

	// ExpireInterval is how often to check for stale devices
	ExpireInterval time.Duration
}

// DefaultRegistryConfig returns sensible defaults.
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		DeviceTTL:      registryDeviceTTLHours * time.Hour,
		EmitEvents:     true,
		MaxDevices:     0, // unlimited
		AutoExpire:     false,
		ExpireInterval: registryExpireIntervalMinutes * time.Minute,
	}
}

// RegistryStats contains registry metrics.
type RegistryStats struct {
	TotalDevices   int       `json:"totalDevices"`
	WiredDevices   int       `json:"wiredDevices"`
	WiFiDevices    int       `json:"wifiDevices"`
	BTDevices      int       `json:"btDevices"`
	MultiConnected int       `json:"multiConnected"` // devices seen on 2+ connections
	AddCount       int64     `json:"addCount"`
	UpdateCount    int64     `json:"updateCount"`
	RemoveCount    int64     `json:"removeCount"`
	LastUpdate     time.Time `json:"lastUpdate"`
}

// NewDeviceRegistry creates a new device registry.
func NewDeviceRegistry(eventBus *EventBus, config *RegistryConfig) *DeviceRegistry {
	if config == nil {
		config = DefaultRegistryConfig()
	}

	return &DeviceRegistry{
		devices:    make(map[string]*DiscoveredDevice),
		byIP:       make(map[string]*DiscoveredDevice),
		byHostname: make(map[string]*DiscoveredDevice),
		byVendor:   make(map[string][]*DiscoveredDevice),
		eventBus:   eventBus,
		config:     config,
	}
}

// AddOrUpdate adds a new device or updates an existing one.
// Returns the device and whether it was newly created.
func (r *DeviceRegistry) AddOrUpdate(device *DiscoveredDevice) (*DiscoveredDevice, bool) {
	if device == nil || device.MAC == "" {
		return nil, false
	}

	mac := normalizeMAC(device.MAC)

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.devices[mac]
	if exists {
		// Update existing device
		changes := r.mergeDevice(existing, device)
		existing.LastSeen = time.Now()
		r.stats.UpdateCount++
		r.stats.LastUpdate = time.Now()

		// Update indexes
		r.updateIndexes(existing)

		// Emit update event
		if r.config.EmitEvents && r.eventBus != nil && len(changes) > 0 {
			r.eventBus.Publish(NewDeviceUpdatedEvent(SourceEngine, existing, changes))
		}

		return existing, false
	}

	// Add new device
	device.MAC = mac // ensure normalized
	if device.LastSeen.IsZero() {
		device.LastSeen = time.Now()
	}

	r.devices[mac] = device
	r.stats.TotalDevices++
	r.stats.AddCount++
	r.stats.LastUpdate = time.Now()

	// Update statistics by connection type
	r.updateConnectionStats(device, true)

	// Build indexes
	r.updateIndexes(device)

	// Emit discovery event
	if r.config.EmitEvents && r.eventBus != nil {
		r.eventBus.Publish(NewDeviceDiscoveredEvent(SourceEngine, device))
	}

	return device, true
}

// mergeWiFiPresence merges incoming WiFi presence data into existing device.
func mergeWiFiPresence(existing, incoming *DiscoveredDevice, changes map[string]any) {
	if incoming.WiFiPresence == nil {
		return
	}
	if existing.WiFiPresence == nil {
		existing.WiFiPresence = incoming.WiFiPresence
		changes["wifiPresence"] = existing.WiFiPresence
		return
	}
	if incoming.WiFiPresence.SSID != "" {
		existing.WiFiPresence.SSID = incoming.WiFiPresence.SSID
	}
	if incoming.WiFiPresence.SignalDBm != 0 {
		existing.WiFiPresence.SignalDBm = incoming.WiFiPresence.SignalDBm
	}
	existing.WiFiPresence.LastSeen = incoming.WiFiPresence.LastSeen
	changes["wifiPresence"] = existing.WiFiPresence
}

// mergeBluetoothPresence merges incoming Bluetooth presence data into existing device.
func mergeBluetoothPresence(existing, incoming *DiscoveredDevice, changes map[string]any) {
	if incoming.BluetoothPresence == nil {
		return
	}
	if existing.BluetoothPresence == nil {
		existing.BluetoothPresence = incoming.BluetoothPresence
		changes["bluetoothPresence"] = existing.BluetoothPresence
		return
	}
	if incoming.BluetoothPresence.Name != "" {
		existing.BluetoothPresence.Name = incoming.BluetoothPresence.Name
	}
	if incoming.BluetoothPresence.RSSI != 0 {
		existing.BluetoothPresence.RSSI = incoming.BluetoothPresence.RSSI
	}
	existing.BluetoothPresence.LastSeen = incoming.BluetoothPresence.LastSeen
	changes["bluetoothPresence"] = existing.BluetoothPresence
}

// mergeStringField updates a string field if incoming has a non-empty different value.
func mergeStringField(existing, incoming *string, key string, changes map[string]any) {
	if *incoming != "" && *incoming != *existing {
		*existing = *incoming
		changes[key] = *incoming
	}
}

// mergeNetworkIdentifiers merges IP, hostname, and name fields.
func (r *DeviceRegistry) mergeNetworkIdentifiers(
	existing, incoming *DiscoveredDevice,
	changes map[string]any,
) {
	// Update IP if new one provided
	if incoming.IP != "" && incoming.IP != existing.IP {
		if existing.IP != "" {
			delete(r.byIP, existing.IP)
		}
		changes["ip"] = incoming.IP
		existing.IP = incoming.IP
	}

	// Add IPv6 addresses (merge, don't replace)
	for _, addr := range incoming.IPv6Addresses {
		if !containsIPv6(existing.IPv6Addresses, addr) {
			existing.IPv6Addresses = append(existing.IPv6Addresses, addr)
			changes["ipv6Addresses"] = existing.IPv6Addresses
		}
	}

	mergeStringField(&existing.Hostname, &incoming.Hostname, "hostname", changes)
	mergeStringField(&existing.NetBIOSName, &incoming.NetBIOSName, "netbiosName", changes)
	mergeStringField(&existing.MDNSName, &incoming.MDNSName, "mdnsName", changes)
}

// mergeDeviceMetadata merges vendor, OS, and discovery information.
func (r *DeviceRegistry) mergeDeviceMetadata(
	existing, incoming *DiscoveredDevice,
	changes map[string]any,
) {
	// Update vendor if provided
	if incoming.Vendor != "" && incoming.Vendor != existing.Vendor {
		r.removeFromVendorIndex(existing)
		changes["vendor"] = incoming.Vendor
		existing.Vendor = incoming.Vendor
	}

	mergeStringField(&existing.OSGuess, &incoming.OSGuess, "osGuess", changes)

	// Merge discovery methods
	for _, method := range incoming.DiscoveryMethod {
		if !containsMethod(existing.DiscoveryMethod, method) {
			existing.DiscoveryMethod = append(existing.DiscoveryMethod, method)
			changes["discoveryMethod"] = existing.DiscoveryMethod
		}
	}

	// Merge connection types
	oldConnCount := len(existing.ConnectionTypes)
	for _, connType := range incoming.ConnectionTypes {
		if !containsConnectionType(existing.ConnectionTypes, connType) {
			existing.ConnectionTypes = append(existing.ConnectionTypes, connType)
		}
	}
	if len(existing.ConnectionTypes) != oldConnCount {
		changes["connectionTypes"] = existing.ConnectionTypes
		if len(existing.ConnectionTypes) >= multiConnectionThreshold && oldConnCount < multiConnectionThreshold {
			r.stats.MultiConnected++
		}
	}
}

// mergeProtocolInfo merges LLDP, CDP, EDP, NDP, profile, and SNMP data.
func mergeProtocolInfo(existing, incoming *DiscoveredDevice, changes map[string]any) {
	if incoming.LLDPInfo != nil {
		existing.LLDPInfo = incoming.LLDPInfo
		changes["lldpInfo"] = existing.LLDPInfo
	}
	if incoming.CDPInfo != nil {
		existing.CDPInfo = incoming.CDPInfo
		changes["cdpInfo"] = existing.CDPInfo
	}
	if incoming.EDPInfo != nil {
		existing.EDPInfo = incoming.EDPInfo
		changes["edpInfo"] = existing.EDPInfo
	}
	if incoming.NDPInfo != nil {
		existing.NDPInfo = incoming.NDPInfo
		changes["ndpInfo"] = existing.NDPInfo
	}
	if incoming.Profile != nil {
		existing.Profile = incoming.Profile
		changes["profile"] = existing.Profile
	}
	if incoming.SNMPData != nil {
		existing.SNMPData = incoming.SNMPData
		changes["snmpData"] = existing.SNMPData
	}
}

// mergeVulnerabilities merges vulnerability data.
func mergeVulnerabilities(existing, incoming *DiscoveredDevice, changes map[string]any) {
	if incoming.Vulnerabilities == nil {
		return
	}
	if existing.Vulnerabilities == nil {
		existing.Vulnerabilities = incoming.Vulnerabilities
		changes["vulnerabilities"] = existing.Vulnerabilities
		return
	}
	for _, vuln := range incoming.Vulnerabilities.Vulnerabilities {
		if !containsVuln(existing.Vulnerabilities.Vulnerabilities, vuln) {
			existing.Vulnerabilities.Vulnerabilities = append(existing.Vulnerabilities.Vulnerabilities, vuln)
		}
	}
	changes["vulnerabilities"] = existing.Vulnerabilities
}

// mergeDeviceFlags merges boolean flags.
func mergeDeviceFlags(existing, incoming *DiscoveredDevice, changes map[string]any) {
	if incoming.IsLocal {
		existing.IsLocal = true
	}
	if incoming.IsRouter {
		existing.IsRouter = true
		changes["isRouter"] = true
	}
	if incoming.WoLCapable != nil && *incoming.WoLCapable {
		trueVal := true
		existing.WoLCapable = &trueVal
	}
}

// mergeDevice merges new data into an existing device.
// Returns a map of field names to their new values (for event changes).
func (r *DeviceRegistry) mergeDevice(existing, incoming *DiscoveredDevice) map[string]any {
	changes := make(map[string]any)

	r.mergeNetworkIdentifiers(existing, incoming, changes)
	r.mergeDeviceMetadata(existing, incoming, changes)
	mergeWiFiPresence(existing, incoming, changes)
	mergeBluetoothPresence(existing, incoming, changes)
	mergeProtocolInfo(existing, incoming, changes)
	mergeVulnerabilities(existing, incoming, changes)
	mergeDeviceFlags(existing, incoming, changes)

	return changes
}

// GetDevice returns a device by MAC address.
func (r *DeviceRegistry) GetDevice(mac string) *DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.devices[normalizeMAC(mac)]
}

// GetDeviceByIP returns a device by IP address.
func (r *DeviceRegistry) GetDeviceByIP(ip string) *DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byIP[ip]
}

// GetDeviceByHostname returns a device by hostname.
func (r *DeviceRegistry) GetDeviceByHostname(hostname string) *DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byHostname[strings.ToLower(hostname)]
}

// GetDevices returns all devices.
func (r *DeviceRegistry) GetDevices() []*DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]*DiscoveredDevice, 0, len(r.devices))
	for _, device := range r.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetDevicesByVendor returns all devices from a specific vendor.
func (r *DeviceRegistry) GetDevicesByVendor(vendor string) []*DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byVendor[strings.ToLower(vendor)]
}

// GetDevicesByConnectionType returns devices seen on a specific connection type.
func (r *DeviceRegistry) GetDevicesByConnectionType(connType ConnectionType) []*DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*DiscoveredDevice
	for _, device := range r.devices {
		if slices.Contains(device.ConnectionTypes, connType) {
			result = append(result, device)
		}
	}
	return result
}

// GetMultiConnectedDevices returns devices seen on multiple connection types.
func (r *DeviceRegistry) GetMultiConnectedDevices() []*DiscoveredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*DiscoveredDevice
	for _, device := range r.devices {
		if len(device.ConnectionTypes) >= multiConnectionThreshold {
			result = append(result, device)
		}
	}
	return result
}

// Remove removes a device by MAC address.
func (r *DeviceRegistry) Remove(mac string) bool {
	mac = normalizeMAC(mac)

	r.mu.Lock()
	defer r.mu.Unlock()

	device, exists := r.devices[mac]
	if !exists {
		return false
	}

	// Update stats
	r.updateConnectionStats(device, false)
	r.stats.TotalDevices--
	r.stats.RemoveCount++
	r.stats.LastUpdate = time.Now()

	// Remove from indexes
	r.removeFromIndexes(device)

	// Remove from primary store
	delete(r.devices, mac)

	// Emit lost event
	if r.config.EmitEvents && r.eventBus != nil {
		r.eventBus.Publish(NewDeviceLostEvent(SourceEngine, mac))
	}

	return true
}

// Clear removes all devices from the registry.
func (r *DeviceRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devices = make(map[string]*DiscoveredDevice)
	r.byIP = make(map[string]*DiscoveredDevice)
	r.byHostname = make(map[string]*DiscoveredDevice)
	r.byVendor = make(map[string][]*DiscoveredDevice)
	r.stats = RegistryStats{}
}

// Count returns the number of devices.
func (r *DeviceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.devices)
}

// Stats returns registry statistics.
func (r *DeviceRegistry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.stats
}

// ExpireStale removes devices not seen within the TTL.
func (r *DeviceRegistry) ExpireStale() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-r.config.DeviceTTL)
	var expired []string

	for mac, device := range r.devices {
		if device.LastSeen.Before(cutoff) {
			expired = append(expired, mac)
		}
	}

	for _, mac := range expired {
		device := r.devices[mac]
		r.updateConnectionStats(device, false)
		r.stats.TotalDevices--
		r.stats.RemoveCount++
		r.removeFromIndexes(device)
		delete(r.devices, mac)

		if r.config.EmitEvents && r.eventBus != nil {
			r.eventBus.Publish(NewDeviceLostEvent(SourceEngine, mac))
		}
	}

	if len(expired) > 0 {
		r.stats.LastUpdate = time.Now()
	}

	return len(expired)
}

// updateIndexes updates secondary indexes for a device.
func (r *DeviceRegistry) updateIndexes(device *DiscoveredDevice) {
	// IP index
	if device.IP != "" {
		r.byIP[device.IP] = device
	}

	// Hostname index
	if device.Hostname != "" {
		r.byHostname[strings.ToLower(device.Hostname)] = device
	}

	// Vendor index
	if device.Vendor != "" {
		vendorKey := strings.ToLower(device.Vendor)
		// Check if already in list
		for _, d := range r.byVendor[vendorKey] {
			if d.MAC == device.MAC {
				return // already indexed
			}
		}
		r.byVendor[vendorKey] = append(r.byVendor[vendorKey], device)
	}
}

// removeFromIndexes removes a device from all secondary indexes.
func (r *DeviceRegistry) removeFromIndexes(device *DiscoveredDevice) {
	// IP index
	if device.IP != "" {
		delete(r.byIP, device.IP)
	}

	// Hostname index
	if device.Hostname != "" {
		delete(r.byHostname, strings.ToLower(device.Hostname))
	}

	// Vendor index
	r.removeFromVendorIndex(device)
}

// removeFromVendorIndex removes a device from the vendor index.
func (r *DeviceRegistry) removeFromVendorIndex(device *DiscoveredDevice) {
	if device.Vendor == "" {
		return
	}

	vendorKey := strings.ToLower(device.Vendor)
	devices := r.byVendor[vendorKey]
	for i, d := range devices {
		if d.MAC == device.MAC {
			r.byVendor[vendorKey] = append(devices[:i], devices[i+1:]...)
			break
		}
	}
}

// updateConnectionStats updates statistics based on connection types.
func (r *DeviceRegistry) updateConnectionStats(device *DiscoveredDevice, adding bool) {
	delta := 1
	if !adding {
		delta = -1
	}

	for _, ct := range device.ConnectionTypes {
		switch ct {
		case ConnectionWired:
			r.stats.WiredDevices += delta
		case ConnectionWiFi:
			r.stats.WiFiDevices += delta
		case ConnectionBluetooth:
			r.stats.BTDevices += delta
		}
	}

	if len(device.ConnectionTypes) >= multiConnectionThreshold {
		r.stats.MultiConnected += delta
	}
}

// Helper functions

// containsConnectionType checks if a connection type is in the slice.
func containsConnectionType(types []ConnectionType, t ConnectionType) bool {
	return slices.Contains(types, t)
}

// containsVuln checks if a vulnerability is already in the slice.
func containsVuln(vulns []Vulnerability, v Vulnerability) bool {
	for _, existing := range vulns {
		if existing.CVEID == v.CVEID {
			return true
		}
	}
	return false
}
