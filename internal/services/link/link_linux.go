//go:build linux

package link

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// sysfsNetPath is the base path for network interface information in sysfs.
const sysfsNetPath = "/sys/class/net"

// checkLinkStatePlatform checks link state on Linux using sysfs.
func checkLinkStatePlatform(interfaceName string) State {
	// Read /sys/class/net/<iface>/carrier
	carrierPath := filepath.Join(sysfsNetPath, interfaceName, "carrier")

	data, err := os.ReadFile(carrierPath)
	if err != nil {
		// Try operstate as fallback
		return checkOperState(interfaceName)
	}

	carrier := strings.TrimSpace(string(data))
	if carrier == "1" {
		return StateUp
	}
	return StateDown
}

// checkOperState reads the operational state from sysfs.
func checkOperState(interfaceName string) State {
	operstatePath := filepath.Join(sysfsNetPath, interfaceName, "operstate")

	data, err := os.ReadFile(operstatePath)
	if err != nil {
		return StateUnknown
	}

	state := strings.TrimSpace(string(data))
	switch state {
	case "up":
		return StateUp
	case "down":
		return StateDown
	case "dormant":
		return StateDormant
	default:
		return StateUnknown
	}
}

// getSpeedDuplex retrieves speed and duplex from sysfs on Linux.
func getSpeedDuplex(interfaceName string) (Speed, Duplex) {
	speed := getSpeedFromSysfs(interfaceName)
	duplex := getDuplexFromSysfs(interfaceName)
	return speed, duplex
}

// getSpeedFromSysfs reads the speed from sysfs.
func getSpeedFromSysfs(interfaceName string) Speed {
	speedPath := filepath.Join(sysfsNetPath, interfaceName, "speed")

	data, err := os.ReadFile(speedPath)
	if err != nil {
		return 0
	}

	speedVal, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || speedVal < 0 {
		return 0
	}

	return Speed(speedVal)
}

// getDuplexFromSysfs reads the duplex mode from sysfs.
func getDuplexFromSysfs(interfaceName string) Duplex {
	duplexPath := filepath.Join(sysfsNetPath, interfaceName, "duplex")

	data, err := os.ReadFile(duplexPath)
	if err != nil {
		return DuplexUnknown
	}

	duplex := strings.TrimSpace(string(data))
	return ParseDuplex(duplex)
}

// isPhysicalInterfacePlatform checks if an interface is physical on Linux.
func isPhysicalInterfacePlatform(name string) bool {
	// Check if interface has a device symlink in sysfs
	devicePath := filepath.Join(sysfsNetPath, name, "device")
	if _, err := os.Stat(devicePath); err == nil {
		return true
	}

	// Fallback: check by name patterns
	switch {
	case strings.HasPrefix(name, "eth"):
		return true
	case strings.HasPrefix(name, "enp"), strings.HasPrefix(name, "ens"):
		return true
	case strings.HasPrefix(name, "eno"):
		return true
	case strings.HasPrefix(name, "wlan"), strings.HasPrefix(name, "wlp"):
		return true
	case strings.HasPrefix(name, "lo"):
		return false
	case strings.HasPrefix(name, "docker"):
		return false
	case strings.HasPrefix(name, "veth"):
		return false
	case strings.HasPrefix(name, "br-"):
		return false
	case strings.HasPrefix(name, "virbr"):
		return false
	default:
		return false
	}
}

// parseSpeedPlatform parses speed strings on Linux.
func parseSpeedPlatform(s string) Speed {
	s = strings.ToLower(strings.TrimSpace(s))

	// Try direct numeric (common from sysfs)
	if speedVal, err := strconv.Atoi(s); err == nil {
		return Speed(speedVal)
	}

	// Handle ethtool output formats
	speedRegex := regexp.MustCompile(`(\d+)\s*(mb|gb|mbps|gbps)?`)
	matches := speedRegex.FindStringSubmatch(s)
	if len(matches) >= 2 {
		speedVal, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0
		}
		// If Gbps, multiply by 1000
		if len(matches) >= 3 && (matches[2] == "gb" || matches[2] == "gbps") {
			return Speed(speedVal * 1000)
		}
		return Speed(speedVal)
	}

	return 0
}
