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

// Problem threshold defaults.
const (
	defaultCPUPercent         = 90
	defaultMemoryPercent      = 90
	defaultDiskPercent        = 90
	defaultTempCelsius        = 85
	defaultInputErrorsPerMin  = 10
	defaultOutputErrorsPerMin = 10
	defaultCollisionsPerMin   = 100
	defaultMinSignalDBm       = -75
	defaultMaxRetryPercent    = 15
	defaultMaxChannelUtil     = 80
	defaultMaxCoChannelAPs    = 3
)

// Severity threshold ratios.
const (
	severityWarningRatio  = 0.9
	severityCriticalRatio = 1.0
	errorRateCritical     = 10.0
	errorRateWarning      = 2.0
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
	DeviceID      string `json:"deviceId,omitempty"`      // Links to DiscoveredDevice
	DeviceMAC     string `json:"deviceMac,omitempty"`     // MAC address involved
	InterfaceName string `json:"interfaceName,omitempty"` // Specific interface if applicable

	// Additional context
	IPAddress    string `json:"ipAddress,omitempty"`
	AffectedMACs string `json:"affectedMacs,omitempty"` // Comma-separated for IP conflicts
	SSID         string `json:"ssid,omitempty"`         // WiFi network if applicable
	BSSID        string `json:"bssid,omitempty"`        // AP BSSID if applicable
	Channel      int    `json:"channel,omitempty"`      // WiFi channel if applicable

	// Metrics
	CurrentValue   float64 `json:"currentValue,omitempty"`   // Current measured value
	ThresholdValue float64 `json:"thresholdValue,omitempty"` // Threshold that was exceeded
	Unit           string  `json:"unit,omitempty"`           // Unit of measurement

	// Timestamps
	FirstSeen  time.Time  `json:"firstSeen"`
	LastSeen   time.Time  `json:"lastSeen"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`

	// Occurrence tracking
	OccurrenceCount int `json:"occurrenceCount"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

// IPConflict represents a duplicate IP address situation.
type IPConflict struct {
	IPAddress  string    `json:"ipAddress"`
	MACs       []string  `json:"macs"`      // All MACs claiming this IP
	DeviceIDs  []string  `json:"deviceIds"` // Corresponding device IDs
	FirstSeen  time.Time `json:"firstSeen"`
	LastSeen   time.Time `json:"lastSeen"`
	IsResolved bool      `json:"isResolved"`
}

// DuplexMismatch represents a speed/duplex negotiation issue.
type DuplexMismatch struct {
	DeviceID       string    `json:"deviceId"`
	InterfaceName  string    `json:"interfaceName"`
	LocalDuplex    string    `json:"localDuplex"`    // half/full
	LocalSpeed     int       `json:"localSpeed"`     // Mbps
	RemoteDuplex   string    `json:"remoteDuplex"`   // half/full (if detectable)
	RemoteSpeed    int       `json:"remoteSpeed"`    // Mbps
	CollisionCount int64     `json:"collisionCount"` // High collisions indicate mismatch
	LateCollisions int64     `json:"lateCollisions"` // Late collisions are a clear indicator
	FirstSeen      time.Time `json:"firstSeen"`
	LastSeen       time.Time `json:"lastSeen"`
}

// STPEvent represents a Spanning Tree Protocol event.
type STPEvent struct {
	DeviceID      string    `json:"deviceId"`
	InterfaceName string    `json:"interfaceName"`
	EventType     string    `json:"eventType"` // topology_change, root_change, port_state_change
	OldState      string    `json:"oldState,omitempty"`
	NewState      string    `json:"newState,omitempty"`
	RootBridgeID  string    `json:"rootBridgeId,omitempty"`
	BridgeCost    int       `json:"bridgeCost,omitempty"`
	RecordedAt    time.Time `json:"recordedAt"`
}

// ResourceThreshold represents a device resource usage alert.
type ResourceThreshold struct {
	DeviceID     string    `json:"deviceId"`
	ResourceType string    `json:"resourceType"` // cpu, memory, disk, temperature
	CurrentValue float64   `json:"currentValue"`
	Threshold    float64   `json:"threshold"`
	Unit         string    `json:"unit"` // percent, bytes, celsius
	IsExceeded   bool      `json:"isExceeded"`
	RecordedAt   time.Time `json:"recordedAt"`
}

// InterfaceErrorStats represents error counters for an interface.
type InterfaceErrorStats struct {
	DeviceID      string `json:"deviceId"`
	InterfaceName string `json:"interfaceName"`

	// Input errors
	InputErrors  int64 `json:"inputErrors"`
	CRCErrors    int64 `json:"crcErrors"`
	FrameErrors  int64 `json:"frameErrors"`
	Overruns     int64 `json:"overruns"`
	DroppedInput int64 `json:"droppedInput"`

	// Output errors
	OutputErrors  int64 `json:"outputErrors"`
	Collisions    int64 `json:"collisions"`
	LateCollision int64 `json:"lateCollision"`
	CarrierErrors int64 `json:"carrierErrors"`
	DroppedOutput int64 `json:"droppedOutput"`

	// Delta calculations (change since last poll)
	InputErrorsDelta  int64 `json:"inputErrorsDelta,omitempty"`
	OutputErrorsDelta int64 `json:"outputErrorsDelta,omitempty"`

	RecordedAt time.Time `json:"recordedAt"`
}

// WiFiProblem represents a WiFi-specific issue.
type WiFiProblem struct {
	ProblemType string   `json:"problemType"` // rogue_ap, weak_signal, channel_interference, unauthorized_client
	SSID        string   `json:"ssid,omitempty"`
	BSSID       string   `json:"bssid,omitempty"`
	Channel     int      `json:"channel,omitempty"`
	Band        WiFiBand `json:"band,omitempty"`

	// Signal issues
	SignalDBm    int     `json:"signalDbm,omitempty"`
	NoiseDBm     int     `json:"noiseDbm,omitempty"`
	SNR          int     `json:"snr,omitempty"`
	RetryPercent float64 `json:"retryPercent,omitempty"`

	// Channel issues
	CoChannelAPs       int     `json:"coChannelAps,omitempty"`       // APs on same channel
	AdjacentChannelAPs int     `json:"adjacentChannelAps,omitempty"` // APs on adjacent channels
	UtilizationPercent float64 `json:"utilizationPercent,omitempty"`

	// Rogue detection
	IsRogue        bool   `json:"isRogue,omitempty"`
	IsUnauthorized bool   `json:"isUnauthorized,omitempty"`
	VendorMismatch bool   `json:"vendorMismatch,omitempty"`
	ExpectedVendor string `json:"expectedVendor,omitempty"`
	ActualVendor   string `json:"actualVendor,omitempty"`

	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// ProblemThresholds defines when to trigger problem detection.
type ProblemThresholds struct {
	// Resource thresholds
	CPUPercent    float64 `json:"cpuPercent"    yaml:"cpu_percent"`    // Default: 90
	MemoryPercent float64 `json:"memoryPercent" yaml:"memory_percent"` // Default: 90
	DiskPercent   float64 `json:"diskPercent"   yaml:"disk_percent"`   // Default: 90
	TempCelsius   float64 `json:"tempCelsius"   yaml:"temp_celsius"`   // Default: 85

	// Interface error thresholds (errors per minute)
	InputErrorsPerMin  int64 `json:"inputErrorsPerMin"  yaml:"input_errors_per_min"`  // Default: 10
	OutputErrorsPerMin int64 `json:"outputErrorsPerMin" yaml:"output_errors_per_min"` // Default: 10
	CollisionsPerMin   int64 `json:"collisionsPerMin"   yaml:"collisions_per_min"`    // Default: 100

	// WiFi thresholds
	MinSignalDBm    int     `json:"minSignalDbm"    yaml:"min_signal_dbm"`     // Default: -75
	MaxRetryPercent float64 `json:"maxRetryPercent" yaml:"max_retry_percent"`  // Default: 15
	MaxChannelUtil  float64 `json:"maxChannelUtil"  yaml:"max_channel_util"`   // Default: 80
	MaxCoChannelAPs int     `json:"maxCoChannelAps" yaml:"max_co_channel_aps"` // Default: 3
}

// DefaultProblemThresholds returns sensible default thresholds.
func DefaultProblemThresholds() ProblemThresholds {
	return ProblemThresholds{
		CPUPercent:         defaultCPUPercent,
		MemoryPercent:      defaultMemoryPercent,
		DiskPercent:        defaultDiskPercent,
		TempCelsius:        defaultTempCelsius,
		InputErrorsPerMin:  defaultInputErrorsPerMin,
		OutputErrorsPerMin: defaultOutputErrorsPerMin,
		CollisionsPerMin:   defaultCollisionsPerMin,
		MinSignalDBm:       defaultMinSignalDBm,
		MaxRetryPercent:    defaultMaxRetryPercent,
		MaxChannelUtil:     defaultMaxChannelUtil,
		MaxCoChannelAPs:    defaultMaxCoChannelAPs,
	}
}

// ProblemSummary provides an overview of detected problems.
type ProblemSummary struct {
	TotalActive   int            `json:"totalActive"`
	BySeverity    map[string]int `json:"bySeverity"`
	ByCategory    map[string]int `json:"byCategory"`
	RecentCount   int            `json:"recentCount"`   // Problems in last hour
	ResolvedToday int            `json:"resolvedToday"` // Problems resolved today
	LastScanTime  time.Time      `json:"lastScanTime"`
}

// ProblemDetectionResult contains results from a problem detection scan.
type ProblemDetectionResult struct {
	Problems         []NetworkProblem      `json:"problems"`
	IPConflicts      []IPConflict          `json:"ipConflicts"`
	DuplexMismatches []DuplexMismatch      `json:"duplexMismatches"`
	STPEvents        []STPEvent            `json:"stpEvents"`
	ResourceAlerts   []ResourceThreshold   `json:"resourceAlerts"`
	InterfaceErrors  []InterfaceErrorStats `json:"interfaceErrors"`
	WiFiProblems     []WiFiProblem         `json:"wifiProblems"`
	ScanTime         time.Time             `json:"scanTime"`
	ScanDurationMS   int64                 `json:"scanDurationMs"`
}

// SeverityForResourceUsage determines severity based on usage percentage.
func SeverityForResourceUsage(current, threshold float64) ProblemSeverity {
	ratio := current / threshold
	switch {
	case ratio >= severityCriticalRatio:
		return ProblemSeverityCritical
	case ratio >= severityWarningRatio:
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
	case ratio >= errorRateCritical:
		return ProblemSeverityCritical
	case ratio >= errorRateWarning:
		return ProblemSeverityWarning
	default:
		return ProblemSeverityInfo
	}
}
