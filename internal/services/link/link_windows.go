//go:build windows

// Windows-specific link status implementation.
// Uses netsh, WMI, and PowerShell for link status detection.
package link

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Command timeout for link status operations.
const linkTimeoutSeconds = 15

// checkLinkStatePlatform checks if the interface has link on Windows.
func checkLinkStatePlatform(iface string) State {
	// Use net.InterfaceByName to check if running
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return StateUnknown
	}

	// Check if interface is up and running
	if netIface.Flags&net.FlagUp != 0 && netIface.Flags&net.FlagRunning != 0 {
		return StateUp
	} else if netIface.Flags&net.FlagUp != 0 {
		return StateDown
	}
	return StateUnknown
}

// getSpeedDuplex gets the link speed and duplex for an interface on Windows.
func getSpeedDuplex(iface string) (Speed, Duplex) {
	ctx, cancel := context.WithTimeout(context.Background(), linkTimeoutSeconds*time.Second)
	defer cancel()

	// Try PowerShell Get-NetAdapter first (most reliable)
	speed, duplex := getSpeedDuplexPowerShell(ctx, iface)
	if speed != 0 {
		return speed, duplex
	}

	// Fallback to WMI
	return getSpeedDuplexWMI(ctx, iface)
}

// getSpeedDuplexPowerShell uses PowerShell Get-NetAdapter for speed/duplex.
func getSpeedDuplexPowerShell(ctx context.Context, iface string) (Speed, Duplex) {
	// PowerShell command to get adapter info
	psCmd := fmt.Sprintf(`Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue | Select-Object LinkSpeed, FullDuplex | ConvertTo-Csv -NoTypeInformation`, iface)

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return 0, DuplexUnknown
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, DuplexUnknown
	}

	// Parse CSV (skip header)
	dataLine := strings.TrimSpace(lines[1])
	parts := strings.Split(dataLine, ",")

	var speed Speed
	duplex := DuplexUnknown

	if len(parts) >= 1 {
		speedStr := strings.Trim(parts[0], "\"")
		speed = parseSpeedString(speedStr)
	}
	if len(parts) >= 2 {
		fullDuplex := strings.Trim(parts[1], "\"")
		if strings.EqualFold(fullDuplex, "True") {
			duplex = DuplexFull
		} else if strings.EqualFold(fullDuplex, "False") {
			duplex = DuplexHalf
		}
	}

	return speed, duplex
}

// getSpeedDuplexWMI uses WMI for speed/duplex as fallback.
func getSpeedDuplexWMI(ctx context.Context, iface string) (Speed, Duplex) {
	// Query WMI via PowerShell
	psCmd := fmt.Sprintf(`Get-WmiObject Win32_NetworkAdapter | Where-Object { $_.NetConnectionID -eq '%s' } | Select-Object Speed | ConvertTo-Csv -NoTypeInformation`, iface)

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return 0, DuplexUnknown
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, DuplexUnknown
	}

	dataLine := strings.TrimSpace(lines[1])
	speedBits := strings.Trim(dataLine, "\"")

	// Convert bits/s to Speed (Mbps)
	speed := bitsToSpeed(speedBits)

	// WMI doesn't reliably provide duplex, assume full for modern NICs
	return speed, DuplexFull
}

// bitsToSpeed converts bits/s string to Speed (Mbps).
func bitsToSpeed(bitsStr string) Speed {
	bits, err := strconv.ParseInt(bitsStr, 10, 64)
	if err != nil || bits <= 0 {
		return 0
	}

	// Convert bits per second to Mbps
	mbps := bits / bpsPerMbps
	return Speed(mbps)
}

// parseSpeedString parses Windows speed strings like "1 Gbps", "100 Mbps".
func parseSpeedString(s string) Speed {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}

	var value float64
	var unit string

	// Try "10 Gbps" format
	if _, err := fmt.Sscanf(s, "%f %s", &value, &unit); err == nil {
		return convertToSpeed(value, unit)
	}

	// Try "10Gbps" format (no space)
	for _, suffix := range []string{"gbps", "mbps", "kbps", "bps"} {
		if strings.HasSuffix(s, suffix) {
			numPart := strings.TrimSuffix(s, suffix)
			if v, err := strconv.ParseFloat(numPart, 64); err == nil {
				return convertToSpeed(v, suffix)
			}
		}
	}

	// Try pure number (assume bps)
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return Speed(v / bpsPerMbps)
	}

	return 0
}

// convertToSpeed converts value and unit to Speed (Mbps).
func convertToSpeed(value float64, unit string) Speed {
	unit = strings.ToLower(strings.TrimSpace(unit))

	switch {
	case strings.HasPrefix(unit, "gbps") || strings.HasPrefix(unit, "gb"):
		return Speed(value * mbpsPerGbps)
	case strings.HasPrefix(unit, "mbps") || strings.HasPrefix(unit, "mb"):
		return Speed(value)
	case strings.HasPrefix(unit, "kbps") || strings.HasPrefix(unit, "kb"):
		return Speed(value / 1000)
	case strings.HasPrefix(unit, "bps") || strings.HasPrefix(unit, "b"):
		return Speed(value / bpsPerMbps)
	}

	return 0
}

// isPhysicalInterfacePlatform checks if interface is physical on Windows.
func isPhysicalInterfacePlatform(iface string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), linkTimeoutSeconds*time.Second)
	defer cancel()

	// Use PowerShell to check if adapter is physical
	psCmd := fmt.Sprintf(`Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Physical`, iface)

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		// Fallback: assume physical if we can't determine
		return true
	}

	result := strings.TrimSpace(string(output))
	return strings.EqualFold(result, "True")
}

// parseSpeedPlatform parses a speed string to Speed value.
func parseSpeedPlatform(speedStr string) Speed {
	return parseSpeedString(speedStr)
}
