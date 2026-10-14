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

// Channel width constants in MHz for WiFi standards.
const (
	ChannelWidth20MHz  = 20  // 802.11a/b/g/n (HT20)
	ChannelWidth40MHz  = 40  // 802.11n (HT40)
	ChannelWidth80MHz  = 80  // 802.11ac (VHT80)
	ChannelWidth160MHz = 160 // 802.11ac/ax (VHT160/HE160)
	ChannelWidth320MHz = 320 // 802.11be (EHT320)
)

// ChannelNetwork represents a network for channel visualization.
// Contains all information needed to render the network on a channel overlap graph.
type ChannelNetwork struct {
	SSID    string `json:"ssid"`
	BSSID   string `json:"bssid"`
	Channel int    `json:"channel"` // primary channel
	// The units are in the wire names; ChannelWidth is 20, 40, 80, 160 or 320.
	CenterFreq   int    `json:"centerFreqMhz"`
	ChannelWidth int    `json:"channelWidthMhz"`
	Signal       int    `json:"signalDbm"`
	Band         string `json:"band"            jsonschema:"enum=2.4GHz,enum=5GHz,enum=6GHz"`
	IsConnected  bool   `json:"isConnected"`
}

// ChannelGraphData contains data for channel overlap visualization.
// Organizes networks by frequency band for rendering separate graphs.
type ChannelGraphData struct {
	Networks2_4GHz []ChannelNetwork `json:"networks24Ghz"`
	Networks5GHz   []ChannelNetwork `json:"networks5Ghz"`
	Networks6GHz   []ChannelNetwork `json:"networks6Ghz"`
	ConnectedBSSID string           `json:"connectedBssid,omitempty"`
	ScanTime       time.Time        `json:"scanTime"`
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
		return ChannelWidth20MHz
	case "HT40", "HT40+", "HT40-", "VHT40", "HE40", "EHT40":
		return ChannelWidth40MHz
	case "VHT80", "HE80", "EHT80":
		return ChannelWidth80MHz
	case "VHT160", "HE160", "EHT160":
		return ChannelWidth160MHz
	case "EHT320":
		return ChannelWidth320MHz
	}

	// Fallback: Guess based on frequency band
	band := getBand(freq)
	switch band {
	case Band24GHz:
		return ChannelWidth20MHz // 2.4 GHz typically uses 20 MHz (rarely 40 MHz)
	case Band5GHz:
		return ChannelWidth80MHz // 5 GHz often uses 80 MHz (can be 20, 40, 80, 160)
	case Band6GHz:
		return ChannelWidth160MHz // 6 GHz often uses 160 MHz or 320 MHz
	default:
		return ChannelWidth20MHz // Default to 20 MHz if unknown
	}
}
