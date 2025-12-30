// Package wifi provides wireless network information functionality.
package wifi

import (
	"time"
)

// Band constants for frequency bands.
const (
	Band24GHz = "2.4GHz"
	Band5GHz  = "5GHz"
	Band6GHz  = "6GHz"
)

// ChannelNetwork represents a network for channel visualization.
// Contains all information needed to render the network on a channel overlap graph.
type ChannelNetwork struct {
	SSID         string `json:"ssid"`              // Network name (SSID)
	BSSID        string `json:"bssid"`             // Access point MAC address (BSSID)
	Channel      int    `json:"channel"`           // Primary channel number
	CenterFreq   int    `json:"center_freq_mhz"`   // Center frequency in MHz
	ChannelWidth int    `json:"channel_width_mhz"` // Channel width in MHz (20, 40, 80, 160, 320)
	Signal       int    `json:"signal_dbm"`        // Signal strength in dBm
	Band         string `json:"band"`              // Frequency band ("2.4GHz", "5GHz", "6GHz")
	IsConnected  bool   `json:"is_connected"`      // Whether this is the connected network
}

// ChannelGraphData contains data for channel overlap visualization.
// Organizes networks by frequency band for rendering separate graphs.
type ChannelGraphData struct {
	Networks2_4GHz []ChannelNetwork `json:"networks_2_4ghz"`           // 2.4 GHz band networks (channels 1-14)
	Networks5GHz   []ChannelNetwork `json:"networks_5ghz"`             // 5 GHz band networks (channels 36-165)
	Networks6GHz   []ChannelNetwork `json:"networks_6ghz"`             // 6 GHz band networks (channels 1-233)
	ConnectedBSSID string           `json:"connected_bssid,omitempty"` // BSSID of connected network
	ScanTime       time.Time        `json:"scan_time"`                 // Timestamp of the scan
}

// GetChannelGraphData converts scanned networks into channel graph visualization data.
// It organizes networks by frequency band and determines channel widths.
//
// Parameters:
//   - networks: Slice of scanned networks from Scanner.Scan()
//   - connectedBSSID: BSSID of the currently connected network (empty string if not connected)
//
// Returns a ChannelGraphData structure with networks organized by band.
func GetChannelGraphData(networks []*ScannedNetwork, connectedBSSID string) *ChannelGraphData {
	if networks == nil {
		networks = []*ScannedNetwork{}
	}

	data := &ChannelGraphData{
		Networks2_4GHz: []ChannelNetwork{},
		Networks5GHz:   []ChannelNetwork{},
		Networks6GHz:   []ChannelNetwork{},
		ConnectedBSSID: connectedBSSID,
		ScanTime:       time.Now(),
	}

	for _, network := range networks {
		// Determine band from frequency
		band := getBand(network.Frequency)
		if band == "" {
			continue // Skip networks with unknown band
		}

		// Determine channel width (prefer from scan data, fallback to detection)
		channelWidth := network.ChannelWidth
		if channelWidth == 0 {
			channelWidth = detectChannelWidth(network.Frequency, network.HTMode)
		}

		// Build ChannelNetwork
		cn := ChannelNetwork{
			SSID:         network.SSID,
			BSSID:        network.BSSID,
			Channel:      network.Channel,
			CenterFreq:   network.Frequency,
			ChannelWidth: channelWidth,
			Signal:       network.Signal,
			Band:         band,
			IsConnected:  network.BSSID == connectedBSSID,
		}

		// Add to appropriate band slice
		switch band {
		case Band24GHz:
			data.Networks2_4GHz = append(data.Networks2_4GHz, cn)
		case Band5GHz:
			data.Networks5GHz = append(data.Networks5GHz, cn)
		case Band6GHz:
			data.Networks6GHz = append(data.Networks6GHz, cn)
		}
	}

	return data
}

// getBand determines the frequency band from the frequency in MHz.
// Returns Band24GHz, Band5GHz, Band6GHz, or empty string for unknown.
func getBand(freq int) string {
	switch {
	case freq >= 2400 && freq <= 2500:
		return Band24GHz
	case freq >= 5150 && freq <= 5895:
		return Band5GHz
	case freq >= 5925 && freq <= 7125:
		return Band6GHz
	default:
		return ""
	}
}

// detectChannelWidth attempts to detect the channel width from HTMode or frequency.
// Returns the width in MHz (20, 40, 80, 160, 320), defaulting to 20 if unknown.
func detectChannelWidth(freq int, htMode string) int {
	// Parse HTMode string (e.g., "HT20", "HT40", "VHT80", "HE160", "EHT320")
	switch htMode {
	case "HT20", "VHT20", "HE20", "EHT20":
		return 20
	case "HT40", "HT40+", "HT40-", "VHT40", "HE40", "EHT40":
		return 40
	case "VHT80", "HE80", "EHT80":
		return 80
	case "VHT160", "HE160", "EHT160":
		return 160
	case "EHT320":
		return 320
	}

	// Fallback: Guess based on frequency band
	band := getBand(freq)
	switch band {
	case Band24GHz:
		return 20 // 2.4 GHz typically uses 20 MHz (rarely 40 MHz)
	case Band5GHz:
		return 80 // 5 GHz often uses 80 MHz (can be 20, 40, 80, 160)
	case Band6GHz:
		return 160 // 6 GHz often uses 160 MHz or 320 MHz
	default:
		return 20 // Default to 20 MHz if unknown
	}
}
