package canopy

import (
	"time"
)

// WiFiNetwork represents a detected WiFi network.
type WiFiNetwork struct {
	SSID           string         `json:"ssid"`
	BSSID          string         `json:"bssid"`
	Channel        int            `json:"channel"`
	Frequency      int            `json:"frequencyMHz"`
	Band           WiFiBand       `json:"band"`
	SignalStrength int            `json:"signalDbm"`
	SignalQuality  int            `json:"signalQuality"` // 0-100
	NoiseFloor     int            `json:"noiseFloorDbm"`
	SNR            int            `json:"snrDb"`
	Security       []SecurityType `json:"security"`
	Standard       WiFiStandard   `json:"standard"`
	ChannelWidth   int            `json:"channelWidthMHz"`
	IsHidden       bool           `json:"isHidden"`
	Vendor         string         `json:"vendor,omitempty"`
	LastSeen       time.Time      `json:"lastSeen"`
}

// WiFiBand represents the WiFi frequency band.
type WiFiBand string

// WiFiBand values.
const (
	Band2_4GHz WiFiBand = "2.4GHz"
	Band5GHz   WiFiBand = "5GHz"
	Band6GHz   WiFiBand = "6GHz"
)

// SecurityType represents WiFi security protocols.
type SecurityType string

// SecurityType values.
const (
	SecurityOpen    SecurityType = "Open"
	SecurityWEP     SecurityType = "WEP"
	SecurityWPA     SecurityType = "WPA"
	SecurityWPA2    SecurityType = "WPA2"
	SecurityWPA3    SecurityType = "WPA3"
	SecurityWPA2Ent SecurityType = "WPA2-Enterprise"
	SecurityWPA3Ent SecurityType = "WPA3-Enterprise"
)

// WiFiStandard represents the 802.11 standard.
type WiFiStandard string

// WiFiStandard values.
const (
	Standard80211a  WiFiStandard = "802.11a"
	Standard80211b  WiFiStandard = "802.11b"
	Standard80211g  WiFiStandard = "802.11g"
	Standard80211n  WiFiStandard = "802.11n"
	Standard80211ac WiFiStandard = "802.11ac"
	Standard80211ax WiFiStandard = "802.11ax"
	Standard80211be WiFiStandard = "802.11be"
)

// ScanResult contains results from a WiFi scan.
type ScanResult struct {
	Interface   string        `json:"interface"`
	Networks    []WiFiNetwork `json:"networks"`
	ScanTime    time.Duration `json:"scanTime"`
	ScanTimeMs  float64       `json:"scanTimeMs"`
	ScannedAt   time.Time     `json:"scannedAt"`
	ChannelHops int           `json:"channelHops,omitempty"`
}

// ConnectionStatus represents current WiFi connection state.
type ConnectionStatus struct {
	Connected    bool         `json:"connected"`
	SSID         string       `json:"ssid,omitempty"`
	BSSID        string       `json:"bssid,omitempty"`
	Channel      int          `json:"channel,omitempty"`
	Frequency    int          `json:"frequencyMHz,omitempty"`
	Band         WiFiBand     `json:"band,omitempty"`
	Signal       int          `json:"signalDbm,omitempty"`
	TxRate       float64      `json:"txRateMbps,omitempty"`
	RxRate       float64      `json:"rxRateMbps,omitempty"`
	Security     SecurityType `json:"security,omitempty"`
	IPAddress    string       `json:"ipAddress,omitempty"`
	Gateway      string       `json:"gateway,omitempty"`
	ConnectedAt  time.Time    `json:"connectedAt,omitzero"`
	ConnectedFor string       `json:"connectedFor,omitempty"`
}

// Survey represents a WiFi site survey.
type Survey struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	FloorPlan   *FloorPlan        `json:"floorPlan,omitempty"`
	Points      []SurveyPoint     `json:"points"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Status      SurveyStatus      `json:"status"`
	Coverage    *CoverageAnalysis `json:"coverage,omitempty"`
}

// SurveyStatus represents survey completion status.
type SurveyStatus string

// SurveyStatus values.
const (
	SurveyStatusDraft      SurveyStatus = "draft"
	SurveyStatusInProgress SurveyStatus = "in_progress"
	SurveyStatusComplete   SurveyStatus = "complete"
)

// FloorPlan represents a floor plan image for surveys.
type FloorPlan struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ImageURL string  `json:"imageUrl"`
	Width    float64 `json:"widthMeters"`
	Height   float64 `json:"heightMeters"`
	Scale    float64 `json:"pixelsPerMeter"`
}

// SurveyPoint represents a measurement point in a survey.
type SurveyPoint struct {
	ID         string        `json:"id"`
	X          float64       `json:"x"`
	Y          float64       `json:"y"`
	Networks   []WiFiNetwork `json:"networks"`
	MeasuredAt time.Time     `json:"measuredAt"`
	Notes      string        `json:"notes,omitempty"`
}

// CoverageAnalysis contains WiFi coverage analysis results.
type CoverageAnalysis struct {
	TotalArea       float64          `json:"totalAreaSqM"`
	CoveredArea     float64          `json:"coveredAreaSqM"`
	CoveragePercent float64          `json:"coveragePercent"`
	DeadZones       []DeadZone       `json:"deadZones,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
	HeatmapData     []HeatmapPoint   `json:"heatmapData,omitempty"`
}

// DeadZone represents an area with poor WiFi coverage.
type DeadZone struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radiusMeters"`
	Signal int     `json:"estimatedSignalDbm"`
}

// Recommendation is an AI-generated suggestion.
type Recommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Location    *struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"location,omitempty"`
}

// HeatmapPoint represents signal strength at a location.
type HeatmapPoint struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Signal int     `json:"signalDbm"`
	SSID   string  `json:"ssid"`
}

// ChannelAnalysis contains channel utilization analysis.
type ChannelAnalysis struct {
	Band               WiFiBand      `json:"band"`
	Channels           []ChannelInfo `json:"channels"`
	RecommendedChannel int           `json:"recommendedChannel"`
	AnalyzedAt         time.Time     `json:"analyzedAt"`
}

// ChannelInfo contains information about a specific channel.
type ChannelInfo struct {
	Number        int     `json:"number"`
	CenterFreqMHz int     `json:"centerFreqMHz"`
	NetworkCount  int     `json:"networkCount"`
	Utilization   float64 `json:"utilizationPercent"`
	Interference  float64 `json:"interferenceScore"` // 0-100
	IsRecommended bool    `json:"isRecommended"`
	IsDFS         bool    `json:"isDfs"`
	MaxPowerDbm   int     `json:"maxPowerDbm,omitempty"`
}
