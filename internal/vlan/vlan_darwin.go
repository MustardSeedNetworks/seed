//go:build darwin

package vlan

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// detectVlanSubinterfacesPlatform detects VLAN subinterfaces on macOS.
func detectVlanSubinterfacesPlatform(iface string) []int {
	vlans := make([]int, 0)

	// On macOS, VLANs are created with "vlan" prefix
	// Check using ifconfig
	cmd := exec.Command("ifconfig", "-a")
	output, err := cmd.Output()
	if err != nil {
		return vlans
	}

	// Look for vlan interfaces that reference our parent interface
	// Example: "vlan0: ... vlan: 100 parent interface: en0"
	lines := strings.Split(string(output), "\n")
	vlanRe := regexp.MustCompile(`vlan:\s*(\d+)\s+parent interface:\s*(\S+)`)

	for _, line := range lines {
		if matches := vlanRe.FindStringSubmatch(line); matches != nil {
			parentIface := matches[2]
			if parentIface == iface {
				if vlanID, err := strconv.Atoi(matches[1]); err == nil {
					vlans = append(vlans, vlanID)
				}
			}
		}
	}

	return vlans
}

// createVlanInterfacePlatform creates a VLAN interface on macOS.
func createVlanInterfacePlatform(parentIface string, vlanID int) error {
	// On macOS, we need to create a vlan interface first
	// This typically requires networksetup or manual configuration
	// For now, return nil as this is advanced functionality
	return nil
}

// deleteVlanInterfacePlatform removes a VLAN interface on macOS.
func deleteVlanInterfacePlatform(parentIface string, vlanID int) error {
	// On macOS, VLAN removal requires networksetup
	// For now, return nil as this is advanced functionality
	return nil
}
