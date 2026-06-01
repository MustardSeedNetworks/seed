//go:build windows

// Windows-specific Wi-Fi scanner implementation using netsh wlan.
// Scans for available wireless networks and parses their properties.
package wifi

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
)

// scanPlatform performs a WiFi scan on Windows using netsh wlan.
func scanPlatform(iface string) ([]*ScannedNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	// Trigger a scan first
	_ = exec.CommandContext(ctx, "netsh", "wlan", "show", "networks", "mode=bssid").Run()

	// Small delay to allow scan to complete
	time.Sleep(500 * time.Millisecond)

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

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// New network starts with SSID
		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID") {
			if current != nil && current.SSID != "" {
				networks = append(networks, current)
			}
			current = &ScannedNetwork{
				NoiseFloor: defaultNoiseFloorDbmWindows,
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.SSID = strings.TrimSpace(parts[1])
			}
		}

		if current == nil {
			continue
		}

		// Parse fields
		if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.BSSID = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				var pct int
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d%%", &pct)
				// Convert percentage to dBm (approximation)
				current.Signal = -100 + (pct * 70 / 100)
				current.SNR = current.Signal - current.NoiseFloor
			}
		} else if strings.HasPrefix(line, "Channel") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.Channel, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
				current.Frequency = channelToFrequencyWindows(current.Channel)
				current.IsDFS = isDFSChannelWindows(current.Channel)
			}
		} else if strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "認証") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.Security = mapSecurityType(strings.TrimSpace(parts[1]))
			}
		} else if strings.HasPrefix(line, "Radio type") || strings.HasPrefix(line, "無線の種類") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				radioType := strings.TrimSpace(parts[1])
				current.HTMode, current.ChannelWidth = parseRadioType(radioType)
			}
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
	time.Sleep(500 * time.Millisecond)

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

// parseNetworkList parses the output of "netsh wlan show networks mode=bssid".
func parseNetworkList(output string) []*Network {
	var networks []*Network
	var current *Network

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// New network starts with SSID
		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID") {
			if current != nil && current.SSID != "" {
				networks = append(networks, current)
			}
			current = &Network{}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.SSID = strings.TrimSpace(parts[1])
			}
		}

		if current == nil {
			continue
		}

		// Parse fields
		if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.BSSID = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				var pct int
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d%%", &pct)
				// Convert percentage to dBm
				current.Signal = -100 + (pct * 70 / 100)
			}
		} else if strings.HasPrefix(line, "Channel") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.Channel, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
				// Estimate frequency from channel
				current.Frequency = channelToFrequencyWindows(current.Channel)
			}
		} else if strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "認証") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.Security = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Radio type") || strings.HasPrefix(line, "無線の種類") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.RadioType = strings.TrimSpace(parts[1])
			}
		}
	}

	// Don't forget the last network
	if current != nil && current.SSID != "" {
		networks = append(networks, current)
	}

	return networks
}

// channelToFrequencyWindows converts Wi-Fi channel to frequency in MHz.
// Named differently to avoid conflict with shared function if present.
func channelToFrequencyWindows(channel int) int {
	// 2.4 GHz band
	if channel >= 1 && channel <= 13 {
		return 2407 + channel*5
	}
	if channel == 14 {
		return 2484
	}

	// 5 GHz band
	if channel >= 36 && channel <= 64 {
		return 5180 + (channel-36)*5
	}
	if channel >= 100 && channel <= 144 {
		return 5500 + (channel-100)*5
	}
	if channel >= 149 && channel <= 165 {
		return 5745 + (channel-149)*5
	}

	return 0
}
