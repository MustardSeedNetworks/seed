package discovery

import "slices"

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
		if !slices.Contains(existing.IPv6Addresses, addr) {
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
		if !slices.Contains(existing.DiscoveryMethod, method) {
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
