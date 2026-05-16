package discovery

// devices_copy.go contains the deep-copy helpers that GetDevices / GetDevice /
// GetDeviceByIP use to return immutable snapshots to callers, plus the per-
// nested-type copy helpers for protocol info, profiles, SNMP data, and
// vulnerabilities.

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
