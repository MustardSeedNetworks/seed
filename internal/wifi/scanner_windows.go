//go:build windows

package wifi

// Windows-specific Wi-Fi scanner implementation using netsh wlan.
// Scans for available wireless networks and parses their properties.

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Scanner constants for Windows.
const (
	defaultNoiseFloorDbmWindows = -95 // Typical noise floor estimate in dBm

	// scanSettleDelay gives netsh's triggered scan time to populate results
	// before the follow-up list command reads them.
	scanSettleDelay = 500 * time.Millisecond

	// colonSplitParts caps SplitN at 2 pieces when parsing "key: value"
	// netsh output lines, so a value containing ':' is not split further.
	colonSplitParts = 2

	// Percentage-to-dBm approximation: netsh reports signal as 0-100%;
	// signalDbmFloor is 0%, signalDbmFloor+signalDbmRange is 100%.
	signalDbmFloor     = -100
	signalDbmRange     = 70
	signalPercentScale = 100

	// channelToFrequencyWindows band constants, MHz.
	channel14         = 14
	freq24GHzBase     = 2407
	freq24GHzCh14     = 2484
	freq24GHzSpacing  = 5
	band5GHzLowStart  = 36
	band5GHzLowBase   = 5180
	band5GHzMidStart  = 100
	band5GHzMidBase   = 5500
	band5GHzHighStart = 149
	band5GHzHighBase  = 5745
	band5GHzSpacing   = 5
)

// scanPlatform performs a WiFi scan on Windows using netsh wlan.
func scanPlatform(iface string, _ Helper) ([]*ScannedNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	// Trigger a scan first
	_ = exec.CommandContext(ctx, "netsh", "wlan", "show", "networks", "mode=bssid").Run()

	// Small delay to allow scan to complete
	time.Sleep(scanSettleDelay)

	// Get network list with BSSID details
	args := []string{"wlan", "show", "networks", "mode=bssid"}
	if iface != "" {
		args = append(args, fmt.Sprintf("interface=%s", iface))
	}

	output, err := exec.CommandContext(ctx, "netsh", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan networks: %w", err)
	}

	return parseScannedNetworks(string(output)), nil
}

// parseScannedNetworks parses netsh output into ScannedNetwork structs.
func parseScannedNetworks(output string) []*ScannedNetwork {
	var networks []*ScannedNetwork
	var current *ScannedNetwork

	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)

		// New network starts with SSID
		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID") {
			if current != nil && current.SSID != "" {
				networks = append(networks, current)
			}
			current = &ScannedNetwork{
				NoiseFloor: defaultNoiseFloorDbmWindows,
			}
			parts := strings.SplitN(line, ":", colonSplitParts)
			if len(parts) == colonSplitParts {
				current.SSID = strings.TrimSpace(parts[1])
			}
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "BSSID"):
			parseScannedBSSID(line, current)
		case strings.HasPrefix(line, "Signal"):
			parseScannedSignal(line, current)
		case strings.HasPrefix(line, "Channel"):
			parseScannedChannel(line, current)
		case strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "認証"):
			parseScannedAuth(line, current)
		case strings.HasPrefix(line, "Radio type") || strings.HasPrefix(line, "無線の種類"):
			parseScannedRadioType(line, current)
		}
	}

	// Don't forget the last network
	if current != nil && current.SSID != "" {
		networks = append(networks, current)
	}

	return networks
}

// parseRadioType parses Windows radio type string to HT mode and channel width.
func parseRadioType(radioType string) (string, int) {
	radioType = strings.ToLower(radioType)

	switch {
	case strings.Contains(radioType, "802.11ax") || strings.Contains(radioType, "wi-fi 6"):
		return "HE80", ChannelWidth80MHz
	case strings.Contains(radioType, "802.11ac"):
		return "VHT80", ChannelWidth80MHz
	case strings.Contains(radioType, "802.11n"):
		return "HT40", ChannelWidth40MHz
	case strings.Contains(radioType, "802.11a") || strings.Contains(radioType, "802.11g"):
		return "HT20", ChannelWidth20MHz
	default:
		return "HT20", ChannelWidth20MHz
	}
}

// isDFSChannelWindows checks if a channel is a DFS channel.
func isDFSChannelWindows(channel int) bool {
	return (channel >= 52 && channel <= 64) || (channel >= 100 && channel <= 144)
}

// ScanNetworks scans for available Wi-Fi networks on Windows.
// This is an alternative function that returns a simpler Network struct.
func ScanNetworks(iface string) ([]*Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	// Trigger a scan first (optional, may require admin)
	_ = exec.CommandContext(ctx, "netsh", "wlan", "show", "networks", "mode=bssid").Run()

	// Small delay to allow scan to complete
	time.Sleep(scanSettleDelay)

	// Get network list
	args := []string{"wlan", "show", "networks", "mode=bssid"}
	if iface != "" {
		args = append(args, fmt.Sprintf("interface=%s", iface))
	}

	output, err := exec.CommandContext(ctx, "netsh", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan networks: %w", err)
	}

	return parseNetworkList(string(output)), nil
}

// Network represents a discovered Wi-Fi network.
type Network struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
	RadioType string `json:"radioType"`
}

// parseScannedBSSID reads the BSSID field of a "netsh wlan show networks
// mode=bssid" record into current.
func parseScannedBSSID(line string, current *ScannedNetwork) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.BSSID = strings.TrimSpace(parts[1])
	}
}

// parseScannedSignal reads the Signal field, converting netsh's percentage to
// an approximate dBm reading and deriving SNR from the scanner's noise floor.
func parseScannedSignal(line string, current *ScannedNetwork) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		var pct int
		_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d%%", &pct) // best-effort; zero value is fine
		current.Signal = signalDbmFloor + (pct * signalDbmRange / signalPercentScale)
		current.SNR = current.Signal - current.NoiseFloor
	}
}

// parseScannedChannel reads the Channel field and derives frequency and DFS
// status from it.
func parseScannedChannel(line string, current *ScannedNetwork) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.Channel, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
		current.Frequency = channelToFrequencyWindows(current.Channel)
		current.IsDFS = isDFSChannelWindows(current.Channel)
	}
}

// parseScannedAuth reads the Authentication field into current's Security.
func parseScannedAuth(line string, current *ScannedNetwork) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.Security = mapSecurityType(strings.TrimSpace(parts[1]))
	}
}

// parseScannedRadioType reads the Radio type field into current's HTMode and
// ChannelWidth.
func parseScannedRadioType(line string, current *ScannedNetwork) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		radioType := strings.TrimSpace(parts[1])
		current.HTMode, current.ChannelWidth = parseRadioType(radioType)
	}
}

// parseNetworkList parses the output of "netsh wlan show networks mode=bssid".
func parseNetworkList(output string) []*Network {
	var networks []*Network
	var current *Network

	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)

		// New network starts with SSID
		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID") {
			if current != nil && current.SSID != "" {
				networks = append(networks, current)
			}
			current = &Network{}
			parts := strings.SplitN(line, ":", colonSplitParts)
			if len(parts) == colonSplitParts {
				current.SSID = strings.TrimSpace(parts[1])
			}
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "BSSID"):
			parseNetworkBSSID(line, current)
		case strings.HasPrefix(line, "Signal"):
			parseNetworkSignal(line, current)
		case strings.HasPrefix(line, "Channel"):
			parseNetworkChannel(line, current)
		case strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "認証"):
			parseNetworkAuth(line, current)
		case strings.HasPrefix(line, "Radio type") || strings.HasPrefix(line, "無線の種類"):
			parseNetworkRadioType(line, current)
		}
	}

	// Don't forget the last network
	if current != nil && current.SSID != "" {
		networks = append(networks, current)
	}

	return networks
}

// parseNetworkBSSID reads the BSSID field of a "netsh wlan show networks
// mode=bssid" record into current.
func parseNetworkBSSID(line string, current *Network) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.BSSID = strings.TrimSpace(parts[1])
	}
}

// parseNetworkSignal reads the Signal field, converting netsh's percentage to
// an approximate dBm reading.
func parseNetworkSignal(line string, current *Network) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		var pct int
		_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d%%", &pct) // best-effort; zero value is fine
		current.Signal = signalDbmFloor + (pct * signalDbmRange / signalPercentScale)
	}
}

// parseNetworkChannel reads the Channel field and estimates frequency from it.
func parseNetworkChannel(line string, current *Network) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.Channel, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
		current.Frequency = channelToFrequencyWindows(current.Channel)
	}
}

// parseNetworkAuth reads the Authentication field into current's Security.
func parseNetworkAuth(line string, current *Network) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.Security = strings.TrimSpace(parts[1])
	}
}

// parseNetworkRadioType reads the Radio type field into current's RadioType.
func parseNetworkRadioType(line string, current *Network) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		current.RadioType = strings.TrimSpace(parts[1])
	}
}

// channelToFrequencyWindows converts Wi-Fi channel to frequency in MHz.
// Named differently to avoid conflict with shared function if present.
func channelToFrequencyWindows(channel int) int {
	// 2.4 GHz band
	if channel >= 1 && channel <= 13 {
		return freq24GHzBase + channel*freq24GHzSpacing
	}
	if channel == channel14 {
		return freq24GHzCh14
	}

	// 5 GHz band
	if channel >= band5GHzLowStart && channel <= 64 {
		return band5GHzLowBase + (channel-band5GHzLowStart)*band5GHzSpacing
	}
	if channel >= band5GHzMidStart && channel <= 144 {
		return band5GHzMidBase + (channel-band5GHzMidStart)*band5GHzSpacing
	}
	if channel >= band5GHzHighStart && channel <= 165 {
		return band5GHzHighBase + (channel-band5GHzHighStart)*band5GHzSpacing
	}

	return 0
}
