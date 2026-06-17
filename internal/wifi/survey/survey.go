package survey

import (
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// Type indicates the type of survey being conducted.
type Type string

// WiFi survey type constants.
const (
	TypePassive    Type = "passive"    // Passive scan (all visible networks)
	TypeActive     Type = "active"     // Active monitoring (current connection)
	TypeThroughput Type = "throughput" // Throughput testing with iperf3
)

// Status indicates the current status of a survey.
type Status string

// WiFi survey status constants.
const (
	StatusCreated    Status = "created"
	StatusInProgress Status = "in_progress"
	StatusPaused     Status = "paused"
	StatusCompleted  Status = "completed"
)

// Test duration constants for throughput surveys.
const (
	// defaultTestDurationSec is the default duration for throughput tests in seconds.
	defaultTestDurationSec = 3

	// maxTestDurationSec is the maximum allowed duration for throughput tests in seconds.
	maxTestDurationSec = 60
)

// WiFi frequency band constants.
const (
	// wifi6eMinFrequencyMHz is the minimum frequency in MHz for WiFi 6E band (6 GHz).
	wifi6eMinFrequencyMHz = 5900
)

// FloorPlan contains floor plan image and metadata.
type FloorPlan struct {
	ImageData string  `json:"imageData"` // Base64-encoded image
	Width     int     `json:"width"`     // Image width in pixels
	Height    int     `json:"height"`    // Image height in pixels
	ScaleM    float64 `json:"scaleM"`    // Meters per pixel
}

// Floor represents a single floor in a multi-floor survey.
type Floor struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`  // "Floor 1", "Basement", etc.
	Level     int            `json:"level"` // Numeric level (-1, 0, 1, 2...)
	FloorPlan *FloorPlan     `json:"floorPlan,omitempty"`
	Samples   []*SamplePoint `json:"samples"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// PassiveSample contains data from a passive WiFi scan.
type PassiveSample struct {
	Networks []*wifi.ScannedNetwork `json:"networks"` // All visible APs

	// Aggregated statistics for heatmap visualization
	UniqueSSIDs   int `json:"uniqueSSIDs"`   // Number of unique network names
	UniqueBSSIDs  int `json:"uniqueBSSIDs"`  // Number of unique access points
	APCount2_4    int `json:"apCount2_4"`    // Number of APs on 2.4 GHz band (2400-2500 MHz)
	APCount5      int `json:"apCount5"`      // Number of APs on 5 GHz band (5000-5900 MHz)
	APCount6      int `json:"apCount6"`      // Number of APs on 6 GHz band (5900+ MHz)
	CoChannelAPs  int `json:"coChannelAPs"`  // Number of APs on same channel as strongest AP
	AdjChannelAPs int `json:"adjChannelAPs"` // Number of APs on adjacent channels (±1-2 from strongest)
}

// CalculateAggregations computes aggregated statistics from the networks array.
// This should be called after populating the Networks field to generate heatmap data.
//
// The function calculates:
//   - Unique SSIDs and BSSIDs for network density metrics
//   - AP counts per frequency band (2.4 GHz, 5 GHz, 6 GHz) for band utilization
//   - Co-channel interference: APs on the same channel as the strongest AP
//   - Adjacent channel interference: APs on channels ±1 or ±2 from the strongest AP
//
// Handles nil or empty networks array gracefully.
func (p *PassiveSample) CalculateAggregations() {
	// Reset all aggregated fields
	p.UniqueSSIDs = 0
	p.UniqueBSSIDs = 0
	p.APCount2_4 = 0
	p.APCount5 = 0
	p.APCount6 = 0
	p.CoChannelAPs = 0
	p.AdjChannelAPs = 0

	// Handle nil or empty networks
	if len(p.Networks) == 0 {
		return
	}

	// Track unique SSIDs and BSSIDs using maps
	uniqueSSIDs := make(map[string]bool)
	uniqueBSSIDs := make(map[string]bool)

	// Find the strongest AP (first in the array, as they're sorted by signal strength)
	strongestChannel := p.Networks[0].Channel

	// Process each network
	for _, network := range p.Networks {
		// Count unique SSIDs (skip empty/hidden SSIDs)
		if network.SSID != "" {
			uniqueSSIDs[network.SSID] = true
		}

		// Count unique BSSIDs
		if network.BSSID != "" {
			uniqueBSSIDs[network.BSSID] = true
		}

		// Count APs by frequency band
		// 2.4 GHz: 2400-2500 MHz (channels 1-14)
		// 5 GHz: 5000-5900 MHz (channels 36-165)
		// 6 GHz: 5900+ MHz (channels 1-233 in 6GHz band)
		switch {
		case network.Frequency >= 2400 && network.Frequency < 2500:
			p.APCount2_4++
		case network.Frequency >= 5000 && network.Frequency < wifi6eMinFrequencyMHz:
			p.APCount5++
		case network.Frequency >= wifi6eMinFrequencyMHz:
			p.APCount6++
		}

		// Count co-channel interference (same channel as strongest)
		if network.Channel == strongestChannel {
			p.CoChannelAPs++
		}

		// Count adjacent channel interference (±1 or ±2 channels from strongest)
		// For 2.4 GHz, channels 1-11 are commonly used with ±2 overlap
		// For 5 GHz, channels are typically spaced further apart
		channelDiff := network.Channel - strongestChannel
		if channelDiff < 0 {
			channelDiff = -channelDiff
		}
		if channelDiff >= 1 && channelDiff <= 2 {
			p.AdjChannelAPs++
		}
	}

	// Set the counts
	p.UniqueSSIDs = len(uniqueSSIDs)
	p.UniqueBSSIDs = len(uniqueBSSIDs)
}

// ActiveSample contains data from active connection monitoring.
type ActiveSample struct {
	SSID          string  `json:"ssid"`
	BSSID         string  `json:"bssid"`
	RSSI          int     `json:"rssi"`                    // Signal strength in dBm
	DataRate      float64 `json:"dataRate"`                // Mbps
	RoamingEvent  bool    `json:"roamingEvent"`            // true if BSSID changed since last sample
	PreviousBSSID string  `json:"previousBssid,omitempty"` // BSSID before roaming event
	RoamCount     int     `json:"roamCount,omitempty"`     // Total number of roaming events during survey
}

// ThroughputSample contains data from iperf3 throughput testing.
type ThroughputSample struct {
	SSID         string  `json:"ssid"`
	BSSID        string  `json:"bssid"`
	RSSI         int     `json:"rssi"`
	DownloadMbps float64 `json:"downloadMbps"`
	UploadMbps   float64 `json:"uploadMbps"`
	Latency      float64 `json:"latency"`    // milliseconds
	Jitter       float64 `json:"jitter"`     // milliseconds
	PacketLoss   float64 `json:"packetLoss"` // percentage
}

// SamplePoint represents a measurement at a specific location.
type SamplePoint struct {
	X          int       `json:"x"` // Pixel X coordinate on floor plan
	Y          int       `json:"y"` // Pixel Y coordinate on floor plan
	Timestamp  time.Time `json:"timestamp"`
	SampleData any       `json:"sampleData"` // PassiveSample | ActiveSample | ThroughputSample
}

// Survey represents a WiFi site survey.
type Survey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SurveyType  Type      `json:"surveyType"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Multi-floor support
	Floors        []*Floor `json:"floors"`                  // Multiple floors in the building
	ActiveFloorID string   `json:"activeFloorId,omitempty"` // Currently active floor for data collection

	// Legacy single-floor fields (deprecated, kept for backwards compatibility)
	// When loading surveys, these are automatically migrated to the Floors array.
	FloorPlan *FloorPlan     `json:"floorPlan,omitempty"`
	Samples   []*SamplePoint `json:"samples,omitempty"`

	// Configuration
	Interface    string `json:"interface"`              // WiFi interface to use
	IperfServer  string `json:"iperfServer,omitempty"`  // For throughput surveys
	TestDuration int    `json:"testDuration,omitempty"` // seconds, for throughput tests

	// Imported / placed data (#727). These are sourced from AirMapper imports
	// or user placement in the planner and are persisted with the survey so
	// they survive reload.
	APLocations      []APLocation        `json:"apLocations,omitempty"`
	ClientLocations  []ClientLocation    `json:"clientLocations,omitempty"`
	PassFailCriteria []PassFailCriterion `json:"passFailCriteria,omitempty"`
}

// APLocation marks the floorplan-relative position of an access point. Sourced
// from imported survey data or user placement.
type APLocation struct {
	ID       string `json:"id"` // Stable identifier (uuid)
	X        int    `json:"x"`  // Pixel offset on the floor plan
	Y        int    `json:"y"`  // Pixel offset on the floor plan
	Label    string `json:"label,omitempty"`
	BSSID    string `json:"bssid,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	Notes    string `json:"notes,omitempty"`
	Imported bool   `json:"imported,omitempty"` // True when added by an importer
}

// ClientLocation marks the floorplan-relative position of a client/station.
// Sourced from imported AirMapper data; the placement UI may add more later.
type ClientLocation struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Label    string `json:"label,omitempty"`
	MAC      string `json:"mac,omitempty"`
	Imported bool   `json:"imported,omitempty"`
}

// PassFailCriterion is a single threshold expression used to flag survey
// samples as pass or fail. Mirrors the frontend PassFailCriterion shape.
type PassFailCriterion struct {
	Option   string  `json:"option"`           // Metric name (e.g. "rssi", "throughput")
	Name     string  `json:"name,omitempty"`   // Display label
	Limit    float64 `json:"limit"`            // Threshold value
	Suffix   string  `json:"suffix,omitempty"` // Unit suffix (e.g. "dBm", "Mbps")
	Enabled  bool    `json:"enabled"`
	Mode     string  `json:"mode,omitempty"` // "gte" or "lte"
	APCount  int     `json:"ap,omitempty"`   // For AP-density criteria
	Imported bool    `json:"imported,omitempty"`
}

// GetActiveFloor returns the currently active floor for data collection.
func (s *Survey) GetActiveFloor() *Floor {
	if s.ActiveFloorID == "" && len(s.Floors) > 0 {
		return s.Floors[0]
	}
	for _, floor := range s.Floors {
		if floor.ID == s.ActiveFloorID {
			return floor
		}
	}
	return nil
}

// GetFloorByID returns a floor by its ID.
func (s *Survey) GetFloorByID(floorID string) *Floor {
	for _, floor := range s.Floors {
		if floor.ID == floorID {
			return floor
		}
	}
	return nil
}

// GetAllSamples returns all samples across all floors (for backwards compatibility).
func (s *Survey) GetAllSamples() []*SamplePoint {
	var samples []*SamplePoint
	for _, floor := range s.Floors {
		samples = append(samples, floor.Samples...)
	}
	// Include legacy samples if present
	samples = append(samples, s.Samples...)
	return samples
}
