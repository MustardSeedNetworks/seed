// Package wifi provides platform-specific WiFi scanning and interface management.
//
// This file contains platform-specific implementations for WiFi network scanning and
// adapter management. Platform detection is handled at compile time via build tags.
//
// Platform support:
//   - Darwin (macOS): CoreWLAN framework via system_profiler and airport utility
//   - Linux: nl80211 netlink interface via iw/iwconfig commands
//
// The scanner provides:
//   - Available network detection with SSID, BSSID, signal strength
//   - Channel information and band detection (2.4GHz / 5GHz)
//   - Security protocol identification (WPA2, WPA3, etc.)
//   - WiFi adapter capabilities and current connection status

//go:build darwin

package wifi

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// scanPlatform performs a WiFi scan on macOS using the airport utility.
func scanPlatform(_ string) ([]*ScannedNetwork, error) {
	// Use airport utility for scanning
	// /System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport
	airportPath := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"

	cmd := exec.Command(airportPath, "-s")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run airport scan: %w", err)
	}

	// Parse output
	// Format:
	//                         SSID BSSID             RSSI CHANNEL HT CC SECURITY
	//           MyNetwork      aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA2(PSK/AES/AES)
	networks := make([]*ScannedNetwork, 0)
	lines := strings.Split(out.String(), "\n")

	// Skip header line
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		network := parseAirportLine(line)
		if network != nil {
			networks = append(networks, network)
		}
	}

	return networks, nil
}

// parseAirportLine parses a single line from airport -s output.
func parseAirportLine(line string) *ScannedNetwork {
	// Use regex to extract fields including HT flag
	// Example: "           MyNetwork      aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA2(PSK/AES/AES)"
	// Fields: SSID, BSSID, RSSI, CHANNEL, HT, CC, SECURITY
	re := regexp.MustCompile(`^\s*(\S.*?)\s+([0-9a-f:]{17})\s+(-?\d+)\s+(\d+)\s+([YN-])\s+.*?(Open|WEP|WPA|WPA2|WPA3)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 7 {
		return nil
	}

	ssid := strings.TrimSpace(matches[1])
	bssid := matches[2]
	signal, _ := strconv.Atoi(matches[3])  //nolint:errcheck // Parse failure defaults to 0
	channel, _ := strconv.Atoi(matches[4]) //nolint:errcheck // Parse failure defaults to 0
	htFlag := matches[5]
	security := matches[6]

	// Extract just the main security type
	//nolint:gocritic // ifElseChain: order matters for security type detection (WPA3 before WPA2 before WPA)
	if strings.Contains(security, "WPA3") {
		security = "WPA3"
	} else if strings.Contains(security, "WPA2") {
		security = "WPA2"
	} else if strings.Contains(security, "WPA") {
		security = "WPA"
	}

	// Determine channel width and HT mode from HT flag
	// Y = 802.11n capable (40MHz), N = legacy (20MHz)
	channelWidth := 20
	htMode := "HT20"
	if htFlag == "Y" {
		channelWidth = 40
		htMode = "HT40"
	}

	// Estimate noise floor (typical range: -90 to -100 dBm)
	// In practice, this should be obtained from 'airport -I' but we'll use a conservative estimate
	noiseFloor := -95

	// Calculate SNR (Signal-to-Noise Ratio)
	snr := signal - noiseFloor

	// Determine if this is a DFS channel
	isDFS := isDFSChannel(channel)

	network := &ScannedNetwork{
		SSID:         ssid,
		BSSID:        bssid,
		Signal:       signal,
		Channel:      channel,
		Frequency:    channelToFrequency(channel),
		Security:     mapSecurityType(security),
		ChannelWidth: channelWidth,
		NoiseFloor:   noiseFloor,
		SNR:          snr,
		HTMode:       htMode,
		IsDFS:        isDFS,
	}

	return network
}

// isDFSChannel checks if a given channel is a DFS (Dynamic Frequency Selection) channel.
// DFS channels are in the 5GHz band and require radar detection:
// - 52-64 (UNII-2).
// - 100-144 (UNII-2 Extended).
func isDFSChannel(channel int) bool {
	return (channel >= 52 && channel <= 64) || (channel >= 100 && channel <= 144)
}
