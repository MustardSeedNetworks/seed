package enumerate

// Package discovery aggregates device information discovered through various
// protocols (ARP, NDP, LLDP, CDP, EDP, mDNS, and ICMP ping) and maintains a
// synchronized view of all discovered devices and their protocol-specific
// metadata.
//
// devices.go holds the DeviceDiscovery struct, NewDeviceDiscovery /
// NewDeviceDiscoveryWithOUI, the Start/Stop lifecycle, the OUI file load and refresh,
// interface / subnet configuration, accessors that copy devices out
// (GetDevices/GetDevice/GetDeviceByIP/Count/IsScanning/LastScan/GetStatus),
// the NetBIOS / mDNS active resolution methods, and ClearDevices /
// SetNameResolution / GetOUIDatabase. The DiscoveredDevice type + its sub-
// types live in devices_types.go; the scan + merge + dedupe logic lives in
// devices_scan.go; the deep-copy helpers live in devices_copy.go.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// DeviceDiscovery aggregates device discovery from all sources.
type DeviceDiscovery struct {
	interfaceName   string
	oui             *resolve.OUIDatabase
	arpScanner      *ARPScanner
	ndpScanner      *NDPScanner
	protoManager    *Manager
	netbiosResolver *resolve.NetBIOSResolver
	mdnsResolver    *resolve.MDNSResolver // Active mDNS resolution
	mdnsListener    *resolve.MDNSListener // Passive mDNS capture
	mu              sync.RWMutex
	devices         map[string]*DiscoveredDevice // Key by MAC
	lastScan        time.Time
	scanning        bool
	nameResolution  bool           // Enable NetBIOS/mDNS name resolution
	deviceTTL       time.Duration  // How long to keep stale devices (fixes #829)
	nameResWg       sync.WaitGroup // Track name resolution goroutines (fixes #836)
	dbWriter        DBDeviceWriter // Database writer for persistence
}

// loadOUIFile layers an on-disk IEEE registry over the embedded one. It reads
// local files only: startup must never wait on the network (#2970). A missing
// file is the normal case, since the embedded registry is complete.
func loadOUIFile(oui *resolve.OUIDatabase, ouiPath string) {
	if ouiPath == "" {
		_ = oui.TryLoadIEEEFile()
		return
	}
	if err := oui.LoadFromIEEEFormat(ouiPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logging.GetLogger().Warn("Failed to load OUI from file", "path", ouiPath, "error", err)
		}
		_ = oui.TryLoadIEEEFile()
		return
	}
	logging.GetLogger().Info("OUI database loaded from file", "path", ouiPath, "entries", oui.Count())
}

// NewDeviceDiscovery creates a new device discovery aggregator.
func NewDeviceDiscovery(interfaceName string, opts ...Option) *DeviceDiscovery {
	return NewDeviceDiscoveryWithOUI(interfaceName, "", opts...)
}

// NewDeviceDiscoveryWithOUI creates a new device discovery aggregator whose
// vendor lookups layer the IEEE file at ouiPath, when present, over the
// embedded registry. It does no network I/O; RefreshOUI is the download.
// opts inject optional dependencies such as the live-capture Opener (WithCapture).
func NewDeviceDiscoveryWithOUI(interfaceName, ouiPath string, opts ...Option) *DeviceDiscovery {
	oui := resolve.NewOUIDatabase()
	loadOUIFile(oui, ouiPath)

	return &DeviceDiscovery{
		interfaceName:   interfaceName,
		oui:             oui,
		arpScanner:      NewARPScanner(interfaceName, oui),
		ndpScanner:      NewNDPScanner(interfaceName),
		protoManager:    NewManager(interfaceName, opts...),
		netbiosResolver: resolve.NewNetBIOSResolver(),
		mdnsResolver:    resolve.NewMDNSResolver(interfaceName),
		mdnsListener:    resolve.NewMDNSListener(interfaceName),
		devices:         make(map[string]*DiscoveredDevice),
		nameResolution:  true,                       // Enabled by default
		deviceTTL:       deviceTTLHours * time.Hour, // Default: expire stale devices after 24h (fixes #829)
	}
}

// RefreshOUI downloads the IEEE registry to ouiPath when the copy there is
// older than maxAge, then loads it. It is the operator's opt-in callout
// (oui_max_age > 0) and runs off the startup path, bounded by
// ouiUpdateTimeoutMinutes; lookups keep using the registry already loaded
// until it finishes.
func (d *DeviceDiscovery) RefreshOUI(ouiPath string, maxAge time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), ouiUpdateTimeoutMinutes*time.Minute)
	defer cancel()

	if err := d.oui.UpdateIfNeeded(ctx, ouiPath, maxAge); err != nil {
		logging.GetLogger().Warn("Failed to refresh OUI database", "path", ouiPath, "error", err)
		return
	}
	logging.GetLogger().Info("OUI database refreshed", "path", ouiPath, "entries", d.oui.Count())
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

	// Note: NDPScanner, resolve.MDNSResolver, resolve.MDNSListener, and resolve.NetBIOSResolver
	// don't have SetInterface methods - they're bound to the original interface.
	// For a complete interface change, Stop() and recreate the DeviceDiscovery.

	return d.protoManager.SetInterface(name)
}

// SetTargetNetworks configures extra subnets to scan.
func (d *DeviceDiscovery) SetTargetNetworks(cidrs []string) error {
	return d.arpScanner.SetTargetNetworks(cidrs)
}

// SetSNMPCredentials lets each sweep ask the target-network addresses that
// answered no echo for SNMP (seed#2449). A nil source asks nothing.
func (d *DeviceDiscovery) SetSNMPCredentials(creds discovery.SNMPCredentialProvider) {
	d.arpScanner.SetSNMPCredentials(creds)
}

// LastSNMPProbe reports what the last sweep's SNMP probe of silent addresses
// did.
func (d *DeviceDiscovery) LastSNMPProbe() SNMPProbeReport {
	return d.arpScanner.LastSNMPProbe()
}

// SetSweepEvidence records the addresses discovery has seen in use, which
// pick the /24s a sweep of a wide target network probes (seed#2832).
func (d *DeviceDiscovery) SetSweepEvidence(addrs []netip.Addr) {
	d.arpScanner.SetSweepEvidence(addrs)
}

// ReadNeighbourCache returns this device's own neighbour cache as the kernel
// holds it right now (#328).
//
// Unfiltered on purpose: an operator debugging why an IP is not resolving to a
// MAC needs to see what is actually in the cache, including entries outside the
// configured discovery scope.
func (d *DeviceDiscovery) ReadNeighbourCache() ([]*ARPEntry, error) {
	return d.arpScanner.ReadNeighbourCache()
}

// PingSweepUnavailable reports why the last sweep could not open an ICMP socket,
// or "" when it could (seed#2629).
func (d *DeviceDiscovery) PingSweepUnavailable() string {
	return d.arpScanner.PingSweepUnavailable()
}

// GetTargetNetworks returns the configured target networks.
func (d *DeviceDiscovery) GetTargetNetworks() []string {
	return d.arpScanner.GetTargetNetworks()
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
func (d *DeviceDiscovery) GetOUIDatabase() *resolve.OUIDatabase {
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
