//go:build linux

package wifi

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// scanPlatform performs a WiFi scan on Linux using nmcli.
// nmcli doesn't require root privileges unlike iw scan.
func scanPlatform(iface string) ([]*ScannedNetwork, error) {
	logger := logging.GetLogger()

	// Use nmcli to list WiFi networks - doesn't require root
	// First rescan to get fresh results
	rescanCmd := exec.Command("nmcli", "device", "wifi", "rescan", "ifname", iface)
	_ = rescanCmd.Run() // Ignore error, rescan might fail if already scanning

	// Get the list of networks
	//nolint:gosec // iface is validated by caller
	cmd := exec.Command("nmcli", "-t", "-f", "BSSID,SSID,MODE,CHAN,FREQ,RATE,SIGNAL,SECURITY", "device", "wifi", "list", "ifname", iface)
	output, err := cmd.Output()
	if err != nil {
		// Fallback to iw if nmcli fails
		logger.Debug("nmcli failed, falling back to iw", "error", err)
		return scanWithIW(iface)
	}

	// Parse nmcli output
	networks := parseNmcliOutput(string(output))

	logger.Debug("WiFi scan complete", "interface", iface, "networks", len(networks))

	return networks, nil
}

// parseNmcliOutput parses nmcli -t output format.
// Format: BSSID:SSID:MODE:CHAN:FREQ:RATE:SIGNAL:SECURITY
func parseNmcliOutput(output string) []*ScannedNetwork {
	var networks []*ScannedNetwork

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// nmcli -t uses : as delimiter, but BSSID also contains colons
		// BSSID is first 17 chars (XX:XX:XX:XX:XX:XX), then fields are separated by :
		if len(line) < 17 {
			continue
		}

		bssid := line[:17]
		rest := line[18:] // Skip BSSID and its trailing :

		// Split remaining fields
		parts := strings.Split(rest, ":")
		if len(parts) < 7 {
			continue
		}

		// Parse fields: SSID:MODE:CHAN:FREQ:RATE:SIGNAL:SECURITY
		ssid := parts[0]
		// mode := parts[1] // Infra/Ad-Hoc
		channel, _ := strconv.Atoi(parts[2])
		freqStr := parts[3]
		// rate := parts[4] // e.g., "540 Mbit/s"
		signal, _ := strconv.Atoi(parts[5])
		security := parts[6]

		// Parse frequency (e.g., "2437 MHz" -> 2437)
		freq := 0
		if idx := strings.Index(freqStr, " "); idx > 0 {
			freq, _ = strconv.Atoi(freqStr[:idx])
		} else {
			freq, _ = strconv.Atoi(freqStr)
		}

		// Convert signal percentage to dBm (approximate)
		// nmcli reports signal as 0-100 percentage
		signalDbm := percentToDbm(signal)

		network := &ScannedNetwork{
			SSID:         ssid,
			BSSID:        strings.ToUpper(bssid),
			Channel:      channel,
			Frequency:    freq,
			Signal:       signalDbm,
			Security:     mapNmcliSecurity(security),
			LastSeen:     time.Now(),
			ChannelWidth: guessChannelWidth(freq),
			NoiseFloor:   -95,
			IsDFS:        (freq >= 5250 && freq <= 5350) || (freq >= 5470 && freq <= 5725),
		}
		network.SNR = network.Signal - network.NoiseFloor

		networks = append(networks, network)
	}

	return networks
}

// percentToDbm converts signal percentage (0-100) to dBm.
// Approximate conversion: 100% ≈ -30 dBm, 0% ≈ -100 dBm.
func percentToDbm(percent int) int {
	// Linear mapping: 0% = -100 dBm, 100% = -30 dBm
	return -100 + (percent * 70 / 100)
}

// mapNmcliSecurity maps nmcli security string to standard security type.
func mapNmcliSecurity(security string) string {
	switch {
	case strings.Contains(security, "WPA3"):
		return "WPA3"
	case strings.Contains(security, "WPA2") && strings.Contains(security, "802.1X"):
		return "WPA2-Enterprise"
	case strings.Contains(security, "WPA2"):
		return "WPA2"
	case strings.Contains(security, "WPA"):
		return "WPA"
	case strings.Contains(security, "WEP"):
		return "WEP"
	case security == "" || security == "--":
		return "Open"
	default:
		return security
	}
}

// guessChannelWidth estimates channel width from frequency.
func guessChannelWidth(freq int) int {
	if freq >= 5180 { // 5 GHz typically uses wider channels
		return 80
	}
	return 20 // 2.4 GHz typically 20 MHz
}

// scanWithIW is a fallback scanner using iw command.
func scanWithIW(iface string) ([]*ScannedNetwork, error) {
	logger := logging.GetLogger()

	// First, check if interface is up
	if err := ensureInterfaceUp(iface); err != nil {
		logger.Warn("Failed to ensure interface is up", "interface", iface, "error", err)
	}

	// Try iw scan dump (doesn't trigger new scan, uses cached results)
	//nolint:gosec // iface is validated by caller
	cmd := exec.Command("iw", "dev", iface, "scan", "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan WiFi networks: %w", err)
	}

	return parseIWScanOutput(string(output)), nil
}

// ensureInterfaceUp brings the interface up if it's down.
func ensureInterfaceUp(iface string) error {
	//nolint:gosec // iface is validated by caller
	cmd := exec.Command("ip", "link", "set", iface, "up")
	return cmd.Run()
}

// parseIWScanOutput parses the output of 'iw dev <iface> scan'.
func parseIWScanOutput(output string) []*ScannedNetwork {
	var networks []*ScannedNetwork
	var current *ScannedNetwork

	// Regex patterns for parsing
	bssRegex := regexp.MustCompile(`^BSS ([0-9a-fA-F:]{17})`)
	freqRegex := regexp.MustCompile(`freq:\s*(\d+)`)
	signalRegex := regexp.MustCompile(`signal:\s*(-?\d+(?:\.\d+)?)\s*dBm`)
	ssidRegex := regexp.MustCompile(`SSID:\s*(.*)`)
	htRegex := regexp.MustCompile(`\* secondary channel offset: (above|below|no secondary)`)
	vhtRegex := regexp.MustCompile(`\* channel width: (\d+)\s*\((\d+)\)?\s*MHz`)
	heRegex := regexp.MustCompile(`HE capabilities`)
	rsnRegex := regexp.MustCompile(`RSN:`)
	wpaRegex := regexp.MustCompile(`WPA:`)
	wepRegex := regexp.MustCompile(`WEP:`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	inRSN := false
	inWPA := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// New BSS entry
		if matches := bssRegex.FindStringSubmatch(line); matches != nil {
			// Save previous network
			if current != nil && current.BSSID != "" {
				networks = append(networks, current)
			}
			current = &ScannedNetwork{
				BSSID:        strings.ToUpper(matches[1]),
				LastSeen:     time.Now(),
				ChannelWidth: 20, // Default
				NoiseFloor:   -95, // Default noise floor
			}
			inRSN = false
			inWPA = false
			continue
		}

		if current == nil {
			continue
		}

		// Parse frequency
		if matches := freqRegex.FindStringSubmatch(trimmed); matches != nil {
			freq, _ := strconv.Atoi(matches[1])
			current.Frequency = freq
			current.Channel = frequencyToChannel(freq)
			// Detect DFS channels (5250-5350, 5470-5725 MHz)
			current.IsDFS = (freq >= 5250 && freq <= 5350) || (freq >= 5470 && freq <= 5725)
		}

		// Parse signal strength
		if matches := signalRegex.FindStringSubmatch(trimmed); matches != nil {
			signal, _ := strconv.ParseFloat(matches[1], 64)
			current.Signal = int(signal)
			current.SNR = current.Signal - current.NoiseFloor
		}

		// Parse SSID
		if matches := ssidRegex.FindStringSubmatch(trimmed); matches != nil {
			current.SSID = matches[1]
		}

		// Parse HT (802.11n) - 40 MHz
		if htRegex.MatchString(trimmed) {
			if strings.Contains(trimmed, "above") || strings.Contains(trimmed, "below") {
				current.ChannelWidth = 40
				current.HTMode = "HT40"
			} else {
				current.HTMode = "HT20"
			}
		}

		// Parse VHT (802.11ac) - 80/160 MHz
		if matches := vhtRegex.FindStringSubmatch(trimmed); matches != nil {
			width, _ := strconv.Atoi(matches[1])
			if width > current.ChannelWidth {
				current.ChannelWidth = width
			}
			switch width {
			case 80:
				current.HTMode = "VHT80"
			case 160:
				current.HTMode = "VHT160"
			}
		}

		// Detect HE (802.11ax/WiFi 6)
		if heRegex.MatchString(trimmed) {
			if current.HTMode == "" || strings.HasPrefix(current.HTMode, "HT") {
				current.HTMode = "HE" + strconv.Itoa(current.ChannelWidth)
			}
		}

		// Track RSN/WPA sections for security detection
		if rsnRegex.MatchString(trimmed) {
			inRSN = true
			inWPA = false
		}
		if wpaRegex.MatchString(trimmed) {
			inWPA = true
			inRSN = false
		}

		// Parse security from RSN/WPA sections
		if inRSN || inWPA {
			if strings.Contains(trimmed, "SAE") {
				current.Security = "WPA3"
			} else if strings.Contains(trimmed, "PSK") {
				if current.Security != "WPA3" {
					if inRSN {
						current.Security = "WPA2"
					} else {
						current.Security = "WPA"
					}
				}
			} else if strings.Contains(trimmed, "802.1X") || strings.Contains(trimmed, "EAP") {
				current.Security = "WPA2-Enterprise"
			}
		}

		// Check for WEP
		if wepRegex.MatchString(trimmed) {
			if current.Security == "" {
				current.Security = "WEP"
			}
		}

		// Check for open network (Privacy capability)
		if strings.Contains(trimmed, "capability:") && !strings.Contains(trimmed, "Privacy") {
			if current.Security == "" {
				current.Security = "Open"
			}
		}
	}

	// Don't forget the last network
	if current != nil && current.BSSID != "" {
		networks = append(networks, current)
	}

	// Set default security for networks without detected security
	for _, n := range networks {
		if n.Security == "" {
			n.Security = "Unknown"
		}
		// Set HTMode if not detected
		if n.HTMode == "" {
			if n.ChannelWidth >= 40 {
				n.HTMode = fmt.Sprintf("HT%d", n.ChannelWidth)
			}
		}
	}

	return networks
}
