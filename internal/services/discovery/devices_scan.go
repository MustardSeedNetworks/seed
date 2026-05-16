package discovery

// devices_scan.go contains DeviceDiscovery.Scan plus the per-protocol merge
// helpers (ARP/LLDP/CDP/EDP/NDP), the duplicate-IP detector, the vendor /
// mDNS / display-name post-processing pass, and the small helpers shared
// between them.

import (
	"context"
	"slices"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

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
