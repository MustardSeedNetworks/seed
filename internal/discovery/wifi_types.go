package discovery

// wifi_types.go holds the Wi-Fi result/data types that stay in the discovery
// kernel (ADR-0018, Phase 6 enumerate split). The Wi-Fi collector logic
// (WiFiBridge) lives in internal/discovery/enumerate, but these types remain
// here because they are referenced by kernel-resident code: the enumerate-stage
// converter (wifiAPToDevice in stages.go), the WiFiCollectorPort interface the
// Engine drives, and the kernel registry. Moving them would invert the
// kernel→stage direction and create an import cycle. The enumerate package
// aliases them.

import "time"

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
	IsHidden            bool                    `json:"isHidden"`
	SecurityType        WiFiSecurityType        `json:"securityType"`
	AuthorizationStatus WiFiAuthorizationStatus `json:"authorizationStatus"`
	FirstSeen           time.Time               `json:"firstSeen"`
	LastSeen            time.Time               `json:"lastSeen"`

	// Computed fields (not stored, populated on query)
	APCount    int `json:"apCount,omitempty"`
	BestSignal int `json:"bestSignal,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

// WiFiAccessPoint represents a WiFi access point (BSSID).
// This links to a DiscoveredDevice when the AP is also discovered via ARP/LLDP.
type WiFiAccessPoint struct {
	ID       string `json:"id"`
	DeviceID string `json:"deviceId,omitempty"` // Links to DiscoveredDevice if correlated
	BSSID    string `json:"bssid"`
	SSIDID   string `json:"ssidId,omitempty"`
	SSIDName string `json:"ssidName,omitempty"` // Denormalized for convenience
	APName   string `json:"apName,omitempty"`
	Vendor   string `json:"vendor,omitempty"`

	// Radio characteristics
	Channel      int      `json:"channel"`
	ChannelWidth int      `json:"channelWidth"` // 20, 40, 80, 160 MHz
	FrequencyMHz int      `json:"frequencyMhz"`
	Band         WiFiBand `json:"band"`
	WiFiStandard []string `json:"wifiStandard,omitempty"` // ax, ac, n, g, a, b

	// Signal quality
	SignalDBm int `json:"signalDbm"`
	NoiseDBm  int `json:"noiseDbm,omitempty"`

	// Client info
	ClientCount  int  `json:"clientCount"`
	MaxClients   int  `json:"maxClients,omitempty"`
	IsAuthorized bool `json:"isAuthorized"`

	FirstSeen time.Time      `json:"firstSeen"`
	LastSeen  time.Time      `json:"lastSeen"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// WiFiClient extends client discovery with WiFi-specific attributes.
// This supplements the DiscoveredDevice with WiFi connection details.
type WiFiClient struct {
	MAC          string    `json:"mac"`
	DeviceID     string    `json:"deviceId,omitempty"` // Links to DiscoveredDevice
	Vendor       string    `json:"vendor,omitempty"`
	SSID         string    `json:"ssid,omitempty"`
	BSSID        string    `json:"bssid,omitempty"`
	SignalDBm    int       `json:"signalDbm,omitempty"`
	NoiseDBm     int       `json:"noiseDbm,omitempty"`
	Channel      int       `json:"channel,omitempty"`
	WiFiStandard []string  `json:"wifiStandard,omitempty"`
	LastSeen     time.Time `json:"lastSeen"`
}

// ChannelUtilization represents WiFi channel usage metrics.
// Used for channel planning and interference analysis.
type ChannelUtilization struct {
	ID           string   `json:"id"`
	Channel      int      `json:"channel"`
	Band         WiFiBand `json:"band"`
	FrequencyMHz int      `json:"frequencyMhz"`

	// Utilization metrics
	UtilizationPercent float64 `json:"utilizationPercent"` // Total airtime usage
	NonWiFiPercent     float64 `json:"nonWifiPercent"`     // Non-802.11 interference
	RetryPercent       float64 `json:"retryPercent"`       // Retry rate
	APCount            int     `json:"apCount"`            // APs on this channel
	ClientCount        int     `json:"clientCount"`        // Clients on this channel

	RecordedAt time.Time `json:"recordedAt"`
}

// WiFiScanResult contains results from a WiFi scan.
type WiFiScanResult struct {
	Networks    []WiFiNetwork        `json:"networks"`
	APs         []WiFiAccessPoint    `json:"aps"`
	Clients     []WiFiClient         `json:"clients"`
	Utilization []ChannelUtilization `json:"utilization,omitempty"`
	ScanTime    time.Time            `json:"scanTime"`
	Interface   string               `json:"interface"`
}

// WiFiDiscoveryStats provides WiFi discovery statistics.
type WiFiDiscoveryStats struct {
	TotalNetworks     int            `json:"totalNetworks"`
	HiddenNetworks    int            `json:"hiddenNetworks"`
	TotalAPs          int            `json:"totalAps"`
	AuthorizedAPs     int            `json:"authorizedAps"`
	UnauthorizedAPs   int            `json:"unauthorizedAps"`
	TotalClients      int            `json:"totalClients"`
	ChannelsByBand    map[string]int `json:"channelsByBand"`
	SecurityBreakdown map[string]int `json:"securityBreakdown"`
	VendorBreakdown   map[string]int `json:"vendorBreakdown"`
	LastScanTime      time.Time      `json:"lastScanTime"`
}
