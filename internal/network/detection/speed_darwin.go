//go:build darwin

package detection

// macOS-specific speed detection module uses system_profiler and networksetup
// for interface speed detection and hardware identification.

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Command timeout constants for macOS system utilities.
const (
	// networkSetupTimeout is the timeout for networksetup and ifconfig commands.
	// These are fast local system calls that should complete quickly.
	networkSetupTimeout = 5 * time.Second

	// systemProfilerTimeout is the timeout for system_profiler commands.
	// system_profiler can take longer as it queries hardware information.
	systemProfilerTimeout = 10 * time.Second
)

// Regex match index constants for parsing ifconfig media output.
const (
	// regexMinMatches is the minimum number of matches required for valid speed extraction.
	// Match 0 is the full match, match 1 is the speed value, match 2 is optional 'g' suffix.
	regexMinMatches = 2

	// regexGbpsSuffixIndex is the index of the optional 'g' suffix in regex matches.
	regexGbpsSuffixIndex = 3
)

// Speed conversion multipliers for network interface speeds.
const (
	// bitsPerGbps converts Gbps values to bits per second (1 Gbps = 1 billion bps).
	bitsPerGbps = 1_000_000_000

	// bitsPerMbps converts Mbps values to bits per second (1 Mbps = 1 million bps).
	bitsPerMbps = 1_000_000
)

// getInterfaceSpeed returns the interface speed in bits per second.
func getInterfaceSpeed(name string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), networkSetupTimeout)
	defer cancel()

	// Try networksetup first
	out, err := exec.CommandContext(ctx, "networksetup", "-getmedia", name).Output()
	if err == nil {
		return parseMediaSpeed(string(out))
	}

	// Fallback to ifconfig
	out, err = exec.CommandContext(ctx, "ifconfig", name).Output()
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

	if len(matches) >= regexMinMatches {
		speedVal, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0
		}

		// Check if it's in Gbps (has 'g' suffix)
		if len(matches) >= regexGbpsSuffixIndex && matches[2] == "g" {
			return int64(speedVal * bitsPerGbps)
		}

		// Otherwise assume Mbps
		return int64(speedVal * bitsPerMbps)
	}

	return 0
}

// identifyByPlatform attempts platform-specific chipset identification on macOS.
func (db *ChipsetDatabase) identifyByPlatform(_ string) *ChipsetInfo {
	ctx, cancel := context.WithTimeout(context.Background(), systemProfilerTimeout)
	defer cancel()

	// Use system_profiler to get hardware info
	out, err := exec.CommandContext(ctx, "system_profiler", "SPNetworkDataType", "-json").Output()
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
