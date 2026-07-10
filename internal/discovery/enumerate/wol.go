package enumerate

// This file infers Wake-on-LAN (WoL) capability for discovered network
// devices based on device type, SNMP sysDescr, and OS fingerprinting.

import (
	"strings"
)

// WoLStatusUntested indicates WoL capability has not been tested.
const WoLStatusUntested = "untested"

// getWoLDeviceTypeSupport returns a map of device types to WoL support (true=yes, false=no).
func getWoLDeviceTypeSupport() map[string]bool {
	return map[string]bool{
		"switch": false, "router": false, "firewall": false,
		"access-point": false, "network-device": false,
		"printer": false, "print-server": false,
		"ip-camera": false, "camera": false,
		"computer": true, "desktop": true, "workstation": true, "server": true,
	}
}

// InferWoLCapability guesses whether a device likely supports Wake-on-LAN
// based on its device type and characteristics.
func InferWoLCapability(device *DiscoveredDevice) *bool {
	if device == nil {
		return nil
	}
	if result := inferWoLFromProfile(device); result != nil {
		return result
	}
	if result := inferWoLFromSNMP(device); result != nil {
		return result
	}
	return inferWoLFromOS(device)
}

func inferWoLFromProfile(device *DiscoveredDevice) *bool {
	if device.Profile == nil {
		return nil
	}
	deviceType := strings.ToLower(device.Profile.DeviceType)
	if deviceType == "laptop" || deviceType == "notebook" {
		return nil // Unknown - often disabled on laptops
	}
	if supported, ok := getWoLDeviceTypeSupport()[deviceType]; ok {
		return &supported
	}
	return nil
}

func inferWoLFromSNMP(device *DiscoveredDevice) *bool {
	if device.Profile == nil || device.Profile.SNMPInfo == nil {
		return nil
	}
	sysDescr := strings.ToLower(device.Profile.SNMPInfo.SysDescr)
	if containsAny(sysDescr, "switch", "router", "cisco", "juniper", "ubiquiti", "mikrotik") {
		f := false
		return &f
	}
	if containsAny(sysDescr, "windows", "linux") {
		t := true
		return &t
	}
	return nil
}

func inferWoLFromOS(device *DiscoveredDevice) *bool {
	if device.OSGuess == "" {
		return nil
	}
	osGuess := strings.ToLower(device.OSGuess)
	if containsAny(osGuess, "windows", "linux", "macos") {
		t := true
		return &t
	}
	if containsAny(osGuess, "ios", "switch") {
		f := false
		return &f
	}
	return nil
}
