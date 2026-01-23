//go:build windows

// Package detection provides intelligent network interface auto-detection.
// Windows-specific speed detection module uses PowerShell and WMI for
// interface speed detection and driver information.
package detection

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Command timeout for Windows detection operations.
const detectionTimeoutSeconds = 15

// getInterfaceSpeed returns the interface speed in bits per second.
func getInterfaceSpeed(name string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), detectionTimeoutSeconds*time.Second)
	defer cancel()

	// Try PowerShell Get-NetAdapter first (most reliable on modern Windows)
	psCmd := fmt.Sprintf(`(Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue).LinkSpeed`, name)
	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err == nil {
		speedStr := strings.TrimSpace(string(output))
		if speed := parseSpeedStringWindows(speedStr); speed > 0 {
			return speed
		}
	}

	// Fallback to WMI
	psCmd = fmt.Sprintf(`(Get-WmiObject Win32_NetworkAdapter | Where-Object { $_.NetConnectionID -eq '%s' }).Speed`, name)
	output, err = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err == nil {
		speedStr := strings.TrimSpace(string(output))
		if speed, err := strconv.ParseInt(speedStr, 10, 64); err == nil && speed > 0 {
			return speed
		}
	}

	return 0
}

// parseSpeedStringWindows parses Windows speed strings like "1 Gbps", "100 Mbps".
func parseSpeedStringWindows(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}

	var value float64
	var unit string

	// Try "10 Gbps" format
	if _, err := fmt.Sscanf(s, "%f %s", &value, &unit); err == nil {
		return convertSpeedToBps(value, unit)
	}

	// Try "10Gbps" format (no space)
	for _, suffix := range []string{"gbps", "mbps", "kbps", "bps"} {
		if strings.HasSuffix(s, suffix) {
			numPart := strings.TrimSuffix(s, suffix)
			if v, err := strconv.ParseFloat(numPart, 64); err == nil {
				return convertSpeedToBps(v, suffix)
			}
		}
	}

	// Try pure number (assume bps)
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}

	return 0
}

// convertSpeedToBps converts speed value and unit to bits per second.
func convertSpeedToBps(value float64, unit string) int64 {
	unit = strings.ToLower(strings.TrimSpace(unit))

	switch {
	case strings.HasPrefix(unit, "gbps") || strings.HasPrefix(unit, "gb"):
		return int64(value * 1_000_000_000)
	case strings.HasPrefix(unit, "mbps") || strings.HasPrefix(unit, "mb"):
		return int64(value * 1_000_000)
	case strings.HasPrefix(unit, "kbps") || strings.HasPrefix(unit, "kb"):
		return int64(value * 1_000)
	case strings.HasPrefix(unit, "bps") || strings.HasPrefix(unit, "b"):
		return int64(value)
	}

	return 0
}

// identifyByPlatform attempts platform-specific chipset identification on Windows.
func (db *ChipsetDatabase) identifyByPlatform(name string) *ChipsetInfo {
	ctx, cancel := context.WithTimeout(context.Background(), detectionTimeoutSeconds*time.Second)
	defer cancel()

	// Get adapter driver info via PowerShell
	psCmd := fmt.Sprintf(`Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue | `+
		`Select-Object DriverName, DriverDescription, InterfaceDescription | ConvertTo-Csv -NoTypeInformation`, name)

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil
	}

	// Parse CSV output
	dataLine := strings.TrimSpace(lines[1])
	parts := strings.Split(dataLine, ",")

	// Try to identify by driver name or description
	for _, part := range parts {
		part = strings.Trim(part, "\"")
		if part == "" {
			continue
		}

		// Try matching against known chipsets
		if chipset := db.IdentifyByKeyword(part); chipset != nil {
			return chipset
		}
	}

	// Fallback: try WMI for more detailed info
	psCmd = fmt.Sprintf(`Get-WmiObject Win32_NetworkAdapter | `+
		`Where-Object { $_.NetConnectionID -eq '%s' } | `+
		`Select-Object Name, Manufacturer, ProductName | ConvertTo-Csv -NoTypeInformation`, name)

	output, err = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return nil
	}

	lines = strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil
	}

	dataLine = strings.TrimSpace(lines[1])
	parts = strings.Split(dataLine, ",")

	for _, part := range parts {
		part = strings.Trim(part, "\"")
		if part == "" {
			continue
		}

		if chipset := db.IdentifyByKeyword(part); chipset != nil {
			return chipset
		}
	}

	return nil
}

// hasTDRCapability checks if the interface supports Time Domain Reflectometry.
// On Windows, TDR requires vendor-specific tools and is not exposed via standard APIs.
func hasTDRCapability(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), detectionTimeoutSeconds*time.Second)
	defer cancel()

	// Check driver name for known TDR-capable NICs
	psCmd := fmt.Sprintf(`(Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue).DriverDescription`, name)
	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return false
	}

	driverDesc := strings.ToLower(strings.TrimSpace(string(output)))

	// Intel NICs with Intel PROSet may support TDR
	// Broadcom NICs with BACS may support TDR
	// We return true if we detect these, but actual TDR requires vendor tools
	tdrCapableDrivers := []string{
		"intel",
		"broadcom",
		"marvell",
		"i210", "i211", "i225", "i350",
		"bcm57", "tg3",
	}

	for _, drv := range tdrCapableDrivers {
		if strings.Contains(driverDesc, drv) {
			// Note: This indicates *potential* TDR support
			// Actual TDR requires vendor management software
			return true
		}
	}

	return false
}

// hasDOMCapability checks if the interface supports Digital Optical Monitoring.
// On Windows, DOM requires vendor-specific tools and is not exposed via standard APIs.
func hasDOMCapability(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), detectionTimeoutSeconds*time.Second)
	defer cancel()

	// Check for SFP-capable NICs (10GbE and above)
	psCmd := fmt.Sprintf(`Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue | `+
		`Select-Object DriverDescription, MediaType | ConvertTo-Csv -NoTypeInformation`, name)
	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return false
	}

	outputStr := strings.ToLower(string(output))

	// SFP-capable NICs that may support DOM with vendor tools
	domCapableIndicators := []string{
		"sfp",
		"10gbe", "10 gbe", "10gbit",
		"25gbe", "25 gbe",
		"40gbe", "40 gbe",
		"100gbe", "100 gbe",
		"mellanox", "connectx",
		"intel x520", "intel x540", "intel x550", "intel x710",
		"ixgbe", "i40e",
	}

	for _, indicator := range domCapableIndicators {
		if strings.Contains(outputStr, indicator) {
			// Note: This indicates *potential* DOM support
			// Actual DOM requires vendor management software
			return true
		}
	}

	return false
}
