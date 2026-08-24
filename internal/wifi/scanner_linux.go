//go:build linux

package wifi

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// wifiCmdTimeout bounds external Wi-Fi tooling (nmcli/iw/ip) to prevent
// hangs when the wireless stack is in a bad state.
const wifiCmdTimeout = 30 * time.Second

const (
	bssidTextLength      = 17
	nmcliFieldCount      = 7
	signalPercentMaximum = 100
	signalDbmRange       = 70
	signalDbmMinimum     = -100
	defaultChannelWidth  = 20
	wideChannelWidth     = 80
	fiveGHzMinimumMHz    = 5180
)

// scanPlatform performs a WiFi scan on Linux using nmcli.
// nmcli doesn't require root privileges unlike iw scan.
func scanPlatform(iface string, _ Helper) ([]*ScannedNetwork, error) {
	logger := logging.GetLogger()

	// nmcli rescan can hang on stuck Wi-Fi stacks; bound to wifiCmdTimeout.
	ctx, cancel := context.WithTimeout(context.Background(), wifiCmdTimeout)
	defer cancel()

	// Use nmcli to list WiFi networks - doesn't require root
	// First rescan to get fresh results
	rescanCmd := exec.CommandContext(ctx, "nmcli", "device", "wifi", "rescan", "ifname", iface)
	_ = rescanCmd.Run() // Ignore error, rescan might fail if already scanning

	// Get the list of networks
	cmd := exec.CommandContext(
		ctx,
		"nmcli",
		"-t",
		"-f",
		"BSSID,SSID,MODE,CHAN,FREQ,RATE,SIGNAL,SECURITY",
		"device",
		"wifi",
		"list",
		"ifname",
		iface,
	)
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
// Format: BSSID:SSID:MODE:CHAN:FREQ:RATE:SIGNAL:SECURITY.
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
		if len(line) < bssidTextLength {
			continue
		}

		bssid := line[:bssidTextLength]
		rest := line[18:] // Skip BSSID and its trailing :

		// Split remaining fields
		parts := strings.Split(rest, ":")
		if len(parts) < nmcliFieldCount {
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
		var freq int
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
	return signalDbmMinimum + (percent * signalDbmRange / signalPercentMaximum)
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
	if freq >= fiveGHzMinimumMHz { // 5 GHz typically uses wider channels
		return wideChannelWidth
	}
	return defaultChannelWidth
}

// scanWithIW is a fallback scanner using iw command.
func scanWithIW(iface string) ([]*ScannedNetwork, error) {
	logger := logging.GetLogger()

	// First, check if interface is up
	if err := ensureInterfaceUp(iface); err != nil {
		logger.Warn("Failed to ensure interface is up", "interface", iface, "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), wifiCmdTimeout)
	defer cancel()

	// Try iw scan dump (doesn't trigger new scan, uses cached results)
	cmd := exec.CommandContext(ctx, "iw", "dev", iface, "scan", "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan WiFi networks: %w", err)
	}

	return parseIWScanOutput(string(output)), nil
}

// ensureInterfaceUp brings the interface up if it's down.
func ensureInterfaceUp(iface string) error {
	ctx, cancel := context.WithTimeout(context.Background(), wifiCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ip", "link", "set", iface, "up")
	return cmd.Run()
}

// parseIWScanOutput parses the output of 'iw dev <iface> scan'.
func parseIWScanOutput(output string) []*ScannedNetwork {
	state := newIWScanState()
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		state.consume(scanner.Text())
	}
	return state.result()
}
