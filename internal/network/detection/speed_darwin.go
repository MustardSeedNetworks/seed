//go:build darwin

// Package detection provides intelligent network interface auto-detection.
// macOS-specific speed detection module uses system_profiler and networksetup
// for interface speed detection and hardware identification.
package detection

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// getInterfaceSpeed returns the interface speed in bits per second.
func getInterfaceSpeed(name string) int64 {
	// Try networksetup first
	//nolint:gosec // G204: name is validated interface name
	out, err := exec.Command("networksetup", "-getmedia", name).Output()
	if err == nil {
		return parseMediaSpeed(string(out))
	}

	// Fallback to ifconfig
	//nolint:gosec // G204: name is validated interface name
	out, err = exec.Command("ifconfig", name).Output()
	if err == nil {
		return parseIfconfigSpeed(string(out))
	}

	return 0
}

// parseMediaSpeed extracts speed from networksetup output.
func parseMediaSpeed(output string) int64 {
	output = strings.ToLower(output)

	// Look for patterns like "1000baseT", "100baseTX", "10GbaseT"
	patterns := []struct {
		pattern string
		speed   int64
	}{
		{`100gbase`, 100_000_000_000},
		{`40gbase`, 40_000_000_000},
		{`25gbase`, 25_000_000_000},
		{`10gbase`, 10_000_000_000},
		{`5gbase`, 5_000_000_000},
		{`2\.5gbase`, 2_500_000_000},
		{`1000base`, 1_000_000_000},
		{`100base`, 100_000_000},
		{`10base`, 10_000_000},
	}

	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p.pattern, output); matched {
			return p.speed
		}
	}

	return 0
}

// parseIfconfigSpeed extracts speed from ifconfig output.
func parseIfconfigSpeed(output string) int64 {
	// Look for "media: autoselect (1000baseT <full-duplex>)"
	re := regexp.MustCompile(`media:.*\((\d+(?:\.\d+)?)(g)?base`)
	matches := re.FindStringSubmatch(strings.ToLower(output))

	if len(matches) >= 2 {
		speedVal, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0
		}

		// Check if it's in Gbps (has 'g' suffix)
		if len(matches) >= 3 && matches[2] == "g" {
			return int64(speedVal * 1_000_000_000)
		}

		// Otherwise assume Mbps
		return int64(speedVal * 1_000_000)
	}

	return 0
}

// identifyByPlatform attempts platform-specific chipset identification on macOS.
func (db *ChipsetDatabase) identifyByPlatform(name string) *ChipsetInfo {
	// Use system_profiler to get hardware info
	out, err := exec.Command("system_profiler", "SPNetworkDataType", "-json").Output()
	if err != nil {
		return nil
	}

	// Simple text search for chipset keywords
	text := strings.ToLower(string(out))
	return db.IdentifyByKeyword(text)
}

// hasTDRCapability checks if the interface supports Time Domain Reflectometry.
// macOS generally doesn't expose TDR capabilities directly.
func hasTDRCapability(_ string) bool {
	// TDR is not typically accessible on macOS
	// Would need specialized drivers or hardware tools
	return false
}

// hasDOMCapability checks if the interface supports Digital Optical Monitoring.
// macOS generally doesn't expose DOM capabilities directly.
func hasDOMCapability(_ string) bool {
	// DOM is not typically accessible on macOS consumer hardware
	return false
}
