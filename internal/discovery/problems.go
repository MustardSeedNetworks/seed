package discovery

// problems.go extends the discovery system with network problem detection.
// This integrates with the existing DiscoveredDevice system by linking problems
// to specific devices, interfaces, or network segments.
//
// Problem categories (based on CyberScope reference):
// - IP Conflicts: Duplicate IP addresses on the network
// - Duplex Mismatches: Half-duplex/full-duplex negotiation issues
// - STP Issues: Spanning Tree topology changes
// - Resource Thresholds: CPU, memory, disk usage alerts
// - Interface Errors: CRC errors, collisions, input/output errors
// - WiFi Issues: Rogue APs, channel interference, weak signals

import (
	"time"
)

// ProblemSeverity indicates the impact level of a detected problem.
type ProblemSeverity string

const (
	ProblemSeverityCritical ProblemSeverity = "critical"
	ProblemSeverityWarning  ProblemSeverity = "warning"
	ProblemSeverityInfo     ProblemSeverity = "info"
)

// ProblemStatus indicates the current state of a detected problem.
type ProblemStatus string

const (
	ProblemStatusActive   ProblemStatus = "active"
	ProblemStatusResolved ProblemStatus = "resolved"
	ProblemStatusIgnored  ProblemStatus = "ignored"
)

// ProblemCategory groups problems by type.
type ProblemCategory string

const (
	ProblemCategoryIPConflict      ProblemCategory = "ip_conflict"
	ProblemCategoryDuplexMismatch  ProblemCategory = "duplex_mismatch"
	ProblemCategorySTP             ProblemCategory = "stp"
	ProblemCategoryResourceUsage   ProblemCategory = "resource_usage"
	ProblemCategoryInterfaceErrors ProblemCategory = "interface_errors"
	ProblemCategoryWiFi            ProblemCategory = "wifi"
	ProblemCategoryConnectivity    ProblemCategory = "connectivity"
	ProblemCategorySecurity        ProblemCategory = "security"
)

// NetworkProblem represents a detected issue in the network.
// Problems are linked to devices/interfaces for correlation.
type NetworkProblem struct {
	ID          string          `json:"id"`
	Category    ProblemCategory `json:"category"`
	Type        string          `json:"type"` // Specific problem type within category
	Severity    ProblemSeverity `json:"severity"`
	Status      ProblemStatus   `json:"status"`
	Title       string          `json:"title"`
	Description string          `json:"description"`

	// Device correlation
	DeviceID      string `json:"device_id,omitempty"`      // Links to DiscoveredDevice
	DeviceMAC     string `json:"device_mac,omitempty"`     // MAC address involved
	InterfaceName string `json:"interface_name,omitempty"` // Specific interface if applicable

	// Additional context
	IPAddress    string `json:"ip_address,omitempty"`
	AffectedMACs string `json:"affected_macs,omitempty"` // Comma-separated for IP conflicts
	SSID         string `json:"ssid,omitempty"`          // WiFi network if applicable
	BSSID        string `json:"bssid,omitempty"`         // AP BSSID if applicable
	Channel      int    `json:"channel,omitempty"`       // WiFi channel if applicable

	// Metrics
	CurrentValue   float64 `json:"current_value,omitempty"`   // Current measured value
	ThresholdValue float64 `json:"threshold_value,omitempty"` // Threshold that was exceeded
	Unit           string  `json:"unit,omitempty"`            // Unit of measurement

	// Timestamps
	FirstSeen  time.Time  `json:"first_seen"`
	LastSeen   time.Time  `json:"last_seen"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	// Occurrence tracking
	OccurrenceCount int `json:"occurrence_count"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

// IPConflict represents a duplicate IP address situation.
type IPConflict struct {
	IPAddress  string    `json:"ip_address"`
	MACs       []string  `json:"macs"`       // All MACs claiming this IP
	DeviceIDs  []string  `json:"device_ids"` // Corresponding device IDs
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	IsResolved bool      `json:"is_resolved"`
}

// DuplexMismatch represents a speed/duplex negotiation issue.
type DuplexMismatch struct {
	DeviceID       string    `json:"device_id"`
	InterfaceName  string    `json:"interface_name"`
	LocalDuplex    string    `json:"local_duplex"`    // half/full
	LocalSpeed     int       `json:"local_speed"`     // Mbps
	RemoteDuplex   string    `json:"remote_duplex"`   // half/full (if detectable)
	RemoteSpeed    int       `json:"remote_speed"`    // Mbps
	CollisionCount int64     `json:"collision_count"` // High collisions indicate mismatch
	LateCollisions int64     `json:"late_collisions"` // Late collisions are a clear indicator
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
}

// STPEvent represents a Spanning Tree Protocol event.
type STPEvent struct {
	DeviceID      string    `json:"device_id"`
	InterfaceName string    `json:"interface_name"`
	EventType     string    `json:"event_type"` // topology_change, root_change, port_state_change
	OldState      string    `json:"old_state,omitempty"`
	NewState      string    `json:"new_state,omitempty"`
	RootBridgeID  string    `json:"root_bridge_id,omitempty"`
	BridgeCost    int       `json:"bridge_cost,omitempty"`
	RecordedAt    time.Time `json:"recorded_at"`
}

// ResourceThreshold represents a device resource usage alert.
type ResourceThreshold struct {
	DeviceID     string    `json:"device_id"`
	ResourceType string    `json:"resource_type"` // cpu, memory, disk, temperature
	CurrentValue float64   `json:"current_value"`
	Threshold    float64   `json:"threshold"`
	Unit         string    `json:"unit"` // percent, bytes, celsius
	IsExceeded   bool      `json:"is_exceeded"`
	RecordedAt   time.Time `json:"recorded_at"`
}

// InterfaceErrorStats represents error counters for an interface.
type InterfaceErrorStats struct {
	DeviceID      string `json:"device_id"`
	InterfaceName string `json:"interface_name"`

	// Input errors
	InputErrors  int64 `json:"input_errors"`
	CRCErrors    int64 `json:"crc_errors"`
	FrameErrors  int64 `json:"frame_errors"`
	Overruns     int64 `json:"overruns"`
	DroppedInput int64 `json:"dropped_input"`

	// Output errors
	OutputErrors  int64 `json:"output_errors"`
	Collisions    int64 `json:"collisions"`
	LateCollision int64 `json:"late_collision"`
	CarrierErrors int64 `json:"carrier_errors"`
	DroppedOutput int64 `json:"dropped_output"`

	// Delta calculations (change since last poll)
	InputErrorsDelta  int64 `json:"input_errors_delta,omitempty"`
	OutputErrorsDelta int64 `json:"output_errors_delta,omitempty"`

	RecordedAt time.Time `json:"recorded_at"`
}

// WiFiProblem represents a WiFi-specific issue.
type WiFiProblem struct {
	ProblemType string   `json:"problem_type"` // rogue_ap, weak_signal, channel_interference, unauthorized_client
	SSID        string   `json:"ssid,omitempty"`
	BSSID       string   `json:"bssid,omitempty"`
	Channel     int      `json:"channel,omitempty"`
	Band        WiFiBand `json:"band,omitempty"`

	// Signal issues
	SignalDBm    int     `json:"signal_dbm,omitempty"`
	NoiseDBm     int     `json:"noise_dbm,omitempty"`
	SNR          int     `json:"snr,omitempty"`
	RetryPercent float64 `json:"retry_percent,omitempty"`

	// Channel issues
	CoChannelAPs       int     `json:"co_channel_aps,omitempty"`       // APs on same channel
	AdjacentChannelAPs int     `json:"adjacent_channel_aps,omitempty"` // APs on adjacent channels
	UtilizationPercent float64 `json:"utilization_percent,omitempty"`

	// Rogue detection
	IsRogue        bool   `json:"is_rogue,omitempty"`
	IsUnauthorized bool   `json:"is_unauthorized,omitempty"`
	VendorMismatch bool   `json:"vendor_mismatch,omitempty"`
	ExpectedVendor string `json:"expected_vendor,omitempty"`
	ActualVendor   string `json:"actual_vendor,omitempty"`

	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// ProblemThresholds defines when to trigger problem detection.
type ProblemThresholds struct {
	// Resource thresholds
	CPUPercent    float64 `json:"cpu_percent"    yaml:"cpu_percent"`    // Default: 90
	MemoryPercent float64 `json:"memory_percent" yaml:"memory_percent"` // Default: 90
	DiskPercent   float64 `json:"disk_percent"   yaml:"disk_percent"`   // Default: 90
	TempCelsius   float64 `json:"temp_celsius"   yaml:"temp_celsius"`   // Default: 85

	// Interface error thresholds (errors per minute)
	InputErrorsPerMin  int64 `json:"input_errors_per_min"  yaml:"input_errors_per_min"`  // Default: 10
	OutputErrorsPerMin int64 `json:"output_errors_per_min" yaml:"output_errors_per_min"` // Default: 10
	CollisionsPerMin   int64 `json:"collisions_per_min"    yaml:"collisions_per_min"`    // Default: 100

	// WiFi thresholds
	MinSignalDBm    int     `json:"min_signal_dbm"     yaml:"min_signal_dbm"`     // Default: -75
	MaxRetryPercent float64 `json:"max_retry_percent"  yaml:"max_retry_percent"`  // Default: 15
	MaxChannelUtil  float64 `json:"max_channel_util"   yaml:"max_channel_util"`   // Default: 80
	MaxCoChannelAPs int     `json:"max_co_channel_aps" yaml:"max_co_channel_aps"` // Default: 3
}

// DefaultProblemThresholds returns sensible default thresholds.
func DefaultProblemThresholds() ProblemThresholds {
	return ProblemThresholds{
		CPUPercent:         90,
		MemoryPercent:      90,
		DiskPercent:        90,
		TempCelsius:        85,
		InputErrorsPerMin:  10,
		OutputErrorsPerMin: 10,
		CollisionsPerMin:   100,
		MinSignalDBm:       -75,
		MaxRetryPercent:    15,
		MaxChannelUtil:     80,
		MaxCoChannelAPs:    3,
	}
}

// ProblemSummary provides an overview of detected problems.
type ProblemSummary struct {
	TotalActive   int            `json:"total_active"`
	BySeverity    map[string]int `json:"by_severity"`
	ByCategory    map[string]int `json:"by_category"`
	RecentCount   int            `json:"recent_count"`   // Problems in last hour
	ResolvedToday int            `json:"resolved_today"` // Problems resolved today
	LastScanTime  time.Time      `json:"last_scan_time"`
}

// ProblemDetectionResult contains results from a problem detection scan.
type ProblemDetectionResult struct {
	Problems         []NetworkProblem      `json:"problems"`
	IPConflicts      []IPConflict          `json:"ip_conflicts"`
	DuplexMismatches []DuplexMismatch      `json:"duplex_mismatches"`
	STPEvents        []STPEvent            `json:"stp_events"`
	ResourceAlerts   []ResourceThreshold   `json:"resource_alerts"`
	InterfaceErrors  []InterfaceErrorStats `json:"interface_errors"`
	WiFiProblems     []WiFiProblem         `json:"wifi_problems"`
	ScanTime         time.Time             `json:"scan_time"`
	ScanDurationMS   int64                 `json:"scan_duration_ms"`
}

// SeverityForResourceUsage determines severity based on usage percentage.
func SeverityForResourceUsage(current, threshold float64) ProblemSeverity {
	ratio := current / threshold
	switch {
	case ratio >= 1.0:
		return ProblemSeverityCritical
	case ratio >= 0.9:
		return ProblemSeverityWarning
	default:
		return ProblemSeverityInfo
	}
}

// SeverityForSignalStrength determines severity based on WiFi signal.
func SeverityForSignalStrength(signalDBm, thresholdDBm int) ProblemSeverity {
	switch {
	case signalDBm <= thresholdDBm-20: // Very weak
		return ProblemSeverityCritical
	case signalDBm <= thresholdDBm-10: // Weak
		return ProblemSeverityWarning
	case signalDBm <= thresholdDBm: // Below threshold
		return ProblemSeverityInfo
	default:
		return ProblemSeverityInfo
	}
}

// SeverityForErrorRate determines severity based on interface error rates.
func SeverityForErrorRate(errorsPerMin, thresholdPerMin int64) ProblemSeverity {
	if errorsPerMin == 0 {
		return ProblemSeverityInfo
	}
	ratio := float64(errorsPerMin) / float64(thresholdPerMin)
	switch {
	case ratio >= 10.0:
		return ProblemSeverityCritical
	case ratio >= 2.0:
		return ProblemSeverityWarning
	default:
		return ProblemSeverityInfo
	}
}
