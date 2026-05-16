package discovery

// Package discovery aggregates device information discovered through various
// protocols (ARP, NDP, LLDP, CDP, EDP, mDNS, and ICMP ping) and maintains a
// synchronized view of all discovered devices and their protocol-specific
// metadata.
//
// devices.go holds the DeviceDiscovery struct, NewDeviceDiscovery /
// NewDeviceDiscoveryWithOUI, the Start/Stop lifecycle, OUI-database loaders,
// interface / subnet configuration, accessors that copy devices out
// (GetDevices/GetDevice/GetDeviceByIP/Count/IsScanning/LastScan/GetStatus),
// the NetBIOS / mDNS active resolution methods, and ClearDevices /
// SetNameResolution / GetOUIDatabase. The DiscoveredDevice type + its sub-
// types live in devices_types.go; the scan + merge + dedupe logic lives in
// devices_scan.go; the deep-copy helpers live in devices_copy.go.

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

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
