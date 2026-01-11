package discovery

// wifi.go extends the discovery system with WiFi-specific network and access point tracking.
// This integrates with the existing DiscoveredDevice system by linking AP devices to their
// WiFi metadata (SSIDs, BSSIDs, channels, signal strength, etc.)
//
// The WiFi discovery data complements the existing ARP/NDP/LLDP discovery:
// - WiFiNetwork tracks SSIDs (network names) across multiple APs
// - WiFiAccessPoint tracks individual BSSIDs with radio characteristics
// - ChannelUtilization tracks spectrum usage for channel planning
// - WiFiClient extends client discovery with WiFi-specific attributes

import (
	"strings"
	"time"
)

// WiFiBand represents the WiFi frequency band.
type WiFiBand string

const (
	WiFiBand24GHz WiFiBand = "2.4GHz"
	WiFiBand5GHz  WiFiBand = "5GHz"
	WiFiBand6GHz  WiFiBand = "6GHz"
)

// WiFiSecurityType represents WiFi security protocol.
type WiFiSecurityType string

const (
	WiFiSecurityOpen WiFiSecurityType = "open"
	WiFiSecurityWEP  WiFiSecurityType = "wep"
	WiFiSecurityWPA  WiFiSecurityType = "wpa"
	WiFiSecurityWPA2 WiFiSecurityType = "wpa2"
	WiFiSecurityWPA3 WiFiSecurityType = "wpa3"
)

// WiFiAuthorizationStatus indicates if a network/device is authorized.
type WiFiAuthorizationStatus string

const (
	WiFiAuthAuthorized   WiFiAuthorizationStatus = "authorized"
	WiFiAuthUnauthorized WiFiAuthorizationStatus = "unauthorized"
	WiFiAuthUnknown      WiFiAuthorizationStatus = "unknown"
)

// WiFiNetwork represents a discovered WiFi network (SSID).
// Multiple access points can broadcast the same SSID.
type WiFiNetwork struct {
	ID                  string                  `json:"id"`
	SSID                string                  `json:"ssid"`
	IsHidden            bool                    `json:"is_hidden"`
	SecurityType        WiFiSecurityType        `json:"security_type"`
	AuthorizationStatus WiFiAuthorizationStatus `json:"authorization_status"`
	FirstSeen           time.Time               `json:"first_seen"`
	LastSeen            time.Time               `json:"last_seen"`

	// Computed fields (not stored, populated on query)
	APCount    int `json:"ap_count,omitempty"`
	BestSignal int `json:"best_signal,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

// WiFiAccessPoint represents a WiFi access point (BSSID).
// This links to a DiscoveredDevice when the AP is also discovered via ARP/LLDP.
type WiFiAccessPoint struct {
	ID       string `json:"id"`
	DeviceID string `json:"device_id,omitempty"` // Links to DiscoveredDevice if correlated
	BSSID    string `json:"bssid"`
	SSIDID   string `json:"ssid_id,omitempty"`
	SSIDName string `json:"ssid_name,omitempty"` // Denormalized for convenience
	APName   string `json:"ap_name,omitempty"`
	Vendor   string `json:"vendor,omitempty"`

	// Radio characteristics
	Channel      int      `json:"channel"`
	ChannelWidth int      `json:"channel_width"` // 20, 40, 80, 160 MHz
	FrequencyMHz int      `json:"frequency_mhz"`
	Band         WiFiBand `json:"band"`
	WiFiStandard []string `json:"wifi_standard,omitempty"` // ax, ac, n, g, a, b

	// Signal quality
	SignalDBm int `json:"signal_dbm"`
	NoiseDBm  int `json:"noise_dbm,omitempty"`

	// Client info
	ClientCount  int  `json:"client_count"`
	MaxClients   int  `json:"max_clients,omitempty"`
	IsAuthorized bool `json:"is_authorized"`

	FirstSeen time.Time      `json:"first_seen"`
	LastSeen  time.Time      `json:"last_seen"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// WiFiClient extends client discovery with WiFi-specific attributes.
// This supplements the DiscoveredDevice with WiFi connection details.
type WiFiClient struct {
	MAC          string    `json:"mac"`
	DeviceID     string    `json:"device_id,omitempty"` // Links to DiscoveredDevice
	Vendor       string    `json:"vendor,omitempty"`
	SSID         string    `json:"ssid,omitempty"`
	BSSID        string    `json:"bssid,omitempty"`
	SignalDBm    int       `json:"signal_dbm,omitempty"`
	NoiseDBm     int       `json:"noise_dbm,omitempty"`
	Channel      int       `json:"channel,omitempty"`
	WiFiStandard []string  `json:"wifi_standard,omitempty"`
	LastSeen     time.Time `json:"last_seen"`
}

// ChannelUtilization represents WiFi channel usage metrics.
// Used for channel planning and interference analysis.
type ChannelUtilization struct {
	ID           string   `json:"id"`
	Channel      int      `json:"channel"`
	Band         WiFiBand `json:"band"`
	FrequencyMHz int      `json:"frequency_mhz"`

	// Utilization metrics
	UtilizationPercent float64 `json:"utilization_percent"` // Total airtime usage
	NonWiFiPercent     float64 `json:"non_wifi_percent"`    // Non-802.11 interference
	RetryPercent       float64 `json:"retry_percent"`       // Retry rate
	APCount            int     `json:"ap_count"`            // APs on this channel
	ClientCount        int     `json:"client_count"`        // Clients on this channel

	RecordedAt time.Time `json:"recorded_at"`
}

// WiFiScanResult contains results from a WiFi scan.
type WiFiScanResult struct {
	Networks    []WiFiNetwork        `json:"networks"`
	APs         []WiFiAccessPoint    `json:"aps"`
	Clients     []WiFiClient         `json:"clients"`
	Utilization []ChannelUtilization `json:"utilization,omitempty"`
	ScanTime    time.Time            `json:"scan_time"`
	Interface   string               `json:"interface"`
}

// WiFiDiscoveryStats provides WiFi discovery statistics.
type WiFiDiscoveryStats struct {
	TotalNetworks     int            `json:"total_networks"`
	HiddenNetworks    int            `json:"hidden_networks"`
	TotalAPs          int            `json:"total_aps"`
	AuthorizedAPs     int            `json:"authorized_aps"`
	UnauthorizedAPs   int            `json:"unauthorized_aps"`
	TotalClients      int            `json:"total_clients"`
	ChannelsByBand    map[string]int `json:"channels_by_band"`
	SecurityBreakdown map[string]int `json:"security_breakdown"`
	VendorBreakdown   map[string]int `json:"vendor_breakdown"`
	LastScanTime      time.Time      `json:"last_scan_time"`
}

// ChannelToBand returns the WiFi band for a given channel number.
func ChannelToBand(channel int) WiFiBand {
	switch {
	case channel >= 1 && channel <= 14:
		return WiFiBand24GHz
	case channel >= 32 && channel <= 177:
		return WiFiBand5GHz
	case channel >= 1 && channel <= 233: // 6GHz uses different numbering
		// 6GHz channels start at 1 but go higher
		// This is simplified - actual 6GHz detection needs frequency
		return WiFiBand6GHz
	default:
		return WiFiBand24GHz
	}
}

// ChannelToFrequency returns the center frequency in MHz for a given channel.
func ChannelToFrequency(channel int, band WiFiBand) int {
	switch band {
	case WiFiBand24GHz:
		if channel >= 1 && channel <= 13 {
			return 2407 + (channel * 5)
		}
		if channel == 14 {
			return 2484
		}
	case WiFiBand5GHz:
		return 5000 + (channel * 5)
	case WiFiBand6GHz:
		return 5950 + (channel * 5)
	}
	return 0
}

// FrequencyToChannel returns the channel number for a given frequency.
func FrequencyToChannel(freqMHz int) (int, WiFiBand) {
	switch {
	case freqMHz >= 2412 && freqMHz <= 2484:
		if freqMHz == 2484 {
			return 14, WiFiBand24GHz
		}
		return (freqMHz - 2407) / 5, WiFiBand24GHz
	case freqMHz >= 5170 && freqMHz <= 5885:
		return (freqMHz - 5000) / 5, WiFiBand5GHz
	case freqMHz >= 5955 && freqMHz <= 7115:
		return (freqMHz - 5950) / 5, WiFiBand6GHz
	}
	return 0, WiFiBand24GHz
}

// normalizeMAC normalizes a MAC address to uppercase with colons.
func normalizeMAC(mac string) string {
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}
