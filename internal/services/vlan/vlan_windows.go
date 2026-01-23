//go:build windows

// Windows-specific VLAN implementation.
// Windows VLAN support varies by NIC driver - most consumer NICs don't support VLANs
// through the OS. Enterprise NICs from Intel, Broadcom, etc. use their own management
// tools (Intel PROSet, Broadcom BACS) rather than netsh.
//
// Platform limitations:
//   - VLAN support requires compatible NIC driver
//   - Consumer NICs typically don't expose VLAN configuration to Windows
//   - Enterprise NICs use vendor-specific tools, not standard Windows APIs
//   - Hyper-V virtual switches support VLANs via PowerShell, not covered here
package vlan

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for VLAN operations.
const vlanTimeoutSeconds = 15

// detectVlanSubinterfacesPlatform detects VLAN subinterfaces on Windows.
// Windows doesn't have a standard VLAN subinterface naming convention like Linux (eth0.100).
// This function attempts to detect VLANs configured through Intel/Broadcom tools.
func detectVlanSubinterfacesPlatform(parentIface string) []int {
	ctx, cancel := context.WithTimeout(context.Background(), vlanTimeoutSeconds*time.Second)
	defer cancel()

	// Try to detect using PowerShell Get-NetAdapterVlan
	// This queries for VLAN-related info on the interface
	psCmd := `Get-NetAdapterVlan -ErrorAction SilentlyContinue | Select-Object Name, InterfaceDescription, VlanID | ConvertTo-Csv -NoTypeInformation`

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		// PowerShell cmdlet not available or no VLANs - this is normal for most Windows systems
		return []int{}
	}

	return parseVlanCsv(string(output), parentIface)
}

// parseVlanCsv parses PowerShell CSV output for VLAN information.
func parseVlanCsv(output, parentIface string) []int {
	var vlanIDs []int

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Skip header
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// CSV format: "Name","InterfaceDescription","VlanID"
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			name := strings.Trim(parts[0], "\"")
			description := strings.Trim(parts[1], "\"")
			vlanIDStr := strings.Trim(parts[2], "\"")

			var vlanID int
			fmt.Sscanf(vlanIDStr, "%d", &vlanID)

			if vlanID > 0 {
				// Filter by parent interface if specified
				if parentIface != "" && !strings.Contains(name, parentIface) && !strings.Contains(description, parentIface) {
					continue
				}

				vlanIDs = append(vlanIDs, vlanID)
			}
		}
	}

	return vlanIDs
}

// createVlanInterfacePlatform creates a VLAN interface on Windows.
// This is not supported through standard Windows APIs for most NICs.
func createVlanInterfacePlatform(parentIface string, vlanID int) error {
	return fmt.Errorf("VLAN creation on Windows requires vendor-specific tools (Intel PROSet, Broadcom BACS) "+
		"or Hyper-V virtual switch configuration via PowerShell. "+
		"Standard Windows APIs do not support VLAN interface creation for interface %s with VLAN ID %d. "+
		"See HARDWARE.md for platform-specific VLAN requirements", parentIface, vlanID)
}

// deleteVlanInterfacePlatform deletes a VLAN interface on Windows.
// This is not supported through standard Windows APIs for most NICs.
func deleteVlanInterfacePlatform(parentIface string, vlanID int) error {
	return fmt.Errorf("VLAN deletion on Windows requires vendor-specific tools (Intel PROSet, Broadcom BACS) "+
		"or Hyper-V virtual switch configuration via PowerShell. "+
		"Standard Windows APIs do not support VLAN interface deletion for interface %s with VLAN ID %d. "+
		"See HARDWARE.md for platform-specific VLAN requirements", parentIface, vlanID)
}

// IsVLANSupported checks if the system supports VLAN configuration.
func IsVLANSupported() bool {
	ctx, cancel := context.WithTimeout(context.Background(), vlanTimeoutSeconds*time.Second)
	defer cancel()

	// Check if Get-NetAdapterVlan cmdlet is available
	err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
		"Get-Command Get-NetAdapterVlan -ErrorAction SilentlyContinue").Run()

	return err == nil
}
