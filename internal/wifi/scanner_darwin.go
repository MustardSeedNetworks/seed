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
func scanPlatform(iface string) ([]*ScannedNetwork, error) {
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
	// Use regex to extract fields
	// Example: "           MyNetwork      aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA2(PSK/AES/AES)"
	re := regexp.MustCompile(`^\s*(\S.*?)\s+([0-9a-f:]{17})\s+(-?\d+)\s+(\d+)\s+.*?(Open|WEP|WPA|WPA2|WPA3)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 6 {
		return nil
	}

	ssid := strings.TrimSpace(matches[1])
	bssid := matches[2]
	signal, _ := strconv.Atoi(matches[3])
	channel, _ := strconv.Atoi(matches[4])
	security := matches[5]

	// Extract just the main security type
	if strings.Contains(security, "WPA3") {
		security = "WPA3"
	} else if strings.Contains(security, "WPA2") {
		security = "WPA2"
	} else if strings.Contains(security, "WPA") {
		security = "WPA"
	}

	network := &ScannedNetwork{
		SSID:      ssid,
		BSSID:     bssid,
		Signal:    signal,
		Channel:   channel,
		Frequency: channelToFrequency(channel),
		Security:  mapSecurityType(security),
	}

	return network
}
