//go:build darwin

package link

import (
	"context"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// checkLinkStatePlatform checks link state on macOS using net.Interface flags.
func checkLinkStatePlatform(interfaceName string) State {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return StateUnknown
	}

	// Check if interface is UP and RUNNING (has carrier)
	if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0 {
		return StateUp
	}

	// If interface is up but not running, link is down
	if iface.Flags&net.FlagUp != 0 {
		return StateDown
	}

	return StateUnknown
}

// getSpeedDuplex retrieves speed and duplex from macOS using networksetup.
func getSpeedDuplex(interfaceName string) (Speed, Duplex) {
	// Try to get media info using networksetup or ifconfig
	// This requires the interface name mapping (e.g., en0 -> "Wi-Fi")
	output, err := exec.CommandContext(context.Background(), "ifconfig", interfaceName).Output()
	if err != nil {
		return 0, DuplexUnknown
	}

	outputStr := string(output)

	// Try to parse media line like "media: autoselect (1000baseT <full-duplex>)"
	mediaRegex := regexp.MustCompile(`media:.*\((\d+)base[A-Z].*<(full|half)-duplex>`)
	matches := mediaRegex.FindStringSubmatch(outputStr)
	if len(matches) >= 3 {
		speedVal, _ := strconv.Atoi(matches[1])
		duplex := ParseDuplex(matches[2])
		return Speed(speedVal), duplex
	}

	// Try parsing simpler format
	speedRegex := regexp.MustCompile(`(\d+)base[A-Z]`)
	speedMatches := speedRegex.FindStringSubmatch(outputStr)
	if len(speedMatches) >= 2 {
		speedVal, _ := strconv.Atoi(speedMatches[1])
		return Speed(speedVal), DuplexUnknown
	}

	return 0, DuplexUnknown
}

// isPhysicalInterfacePlatform checks if an interface is physical on macOS.
func isPhysicalInterfacePlatform(name string) bool {
	// On macOS, physical interfaces are typically en*, eth*
	// Virtual interfaces include lo*, bridge*, utun*, etc.
	switch {
	case strings.HasPrefix(name, "en"):
		return true
	case strings.HasPrefix(name, "eth"):
		return true
	case strings.HasPrefix(name, "lo"):
		return false
	case strings.HasPrefix(name, "bridge"):
		return false
	case strings.HasPrefix(name, "utun"):
		return false
	case strings.HasPrefix(name, "awdl"):
		return false
	case strings.HasPrefix(name, "llw"):
		return false
	default:
		return false
	}
}

// parseSpeedPlatform parses speed strings on macOS.
func parseSpeedPlatform(s string) Speed {
	// Handle common macOS speed formats
	s = strings.ToLower(strings.TrimSpace(s))

	// Try direct numeric
	if speedVal, err := strconv.Atoi(s); err == nil {
		return Speed(speedVal)
	}

	// Handle "1000baseT" style
	baseRegex := regexp.MustCompile(`(\d+)base`)
	if matches := baseRegex.FindStringSubmatch(s); len(matches) >= 2 {
		if speedVal, err := strconv.Atoi(matches[1]); err == nil {
			return Speed(speedVal)
		}
	}

	// Handle "1 Gbps" or "100 Mbps" style
	switch {
	case strings.Contains(s, "100g"):
		return Speed100000
	case strings.Contains(s, "40g"):
		return Speed40000
	case strings.Contains(s, "25g"):
		return Speed25000
	case strings.Contains(s, "10g"):
		return Speed10000
	case strings.Contains(s, "5g"):
		return Speed5000
	case strings.Contains(s, "2.5g"):
		return Speed2500
	case strings.Contains(s, "1g"), strings.Contains(s, "1000"):
		return Speed1000
	case strings.Contains(s, "100"):
		return Speed100
	case strings.Contains(s, "10"):
		return Speed10
	}

	return 0
}
