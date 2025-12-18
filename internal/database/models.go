// Package database provides model types for database entities.
package database

import (
	"time"
)

// Profile represents a configuration profile.
type Profile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ConfigJSON  string    `json:"config"` // JSON string of config
	IsDefault   bool      `json:"isDefault"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Metric represents a single metric data point.
type Metric struct {
	ID            int64     `json:"id"`
	InterfaceName string    `json:"interfaceName"`
	MetricType    string    `json:"metricType"` // e.g., "latency", "throughput", "packet_loss"
	Value         float64   `json:"value"`
	Unit          string    `json:"unit,omitempty"` // e.g., "ms", "Mbps", "%"
	Timestamp     time.Time `json:"timestamp"`
	Metadata      string    `json:"metadata,omitempty"` // JSON string for extra data
}

// MetricType constants for common metric types.
const (
	MetricTypeLatency     = "latency"
	MetricTypeThroughput  = "throughput"
	MetricTypePacketLoss  = "packet_loss"
	MetricTypeJitter      = "jitter"
	MetricTypeSignal      = "signal"
	MetricTypeNoise       = "noise"
	MetricTypeSNR         = "snr"
	MetricTypeDNSResponse = "dns_response"
)

// Device represents a discovered network device.
type Device struct {
	ID         string    `json:"id"`
	IPAddress  string    `json:"ipAddress"`
	MACAddress string    `json:"macAddress,omitempty"`
	Hostname   string    `json:"hostname,omitempty"`
	Vendor     string    `json:"vendor,omitempty"`
	DeviceType string    `json:"deviceType,omitempty"`
	OSFamily   string    `json:"osFamily,omitempty"`
	FirstSeen  time.Time `json:"firstSeen"`
	LastSeen   time.Time `json:"lastSeen"`
	IsActive   bool      `json:"isActive"`
	PortsJSON  string    `json:"ports,omitempty"`    // JSON array of open ports
	Metadata   string    `json:"metadata,omitempty"` // JSON string for extra data
}

// Alert represents a system alert.
type Alert struct {
	ID             int64      `json:"id"`
	Type           string     `json:"type"`     // e.g., "security", "performance", "connectivity"
	Severity       string     `json:"severity"` // "info", "warning", "error", "critical"
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	Source         string     `json:"source,omitempty"` // What generated the alert
	DeviceID       *string    `json:"deviceId,omitempty"`
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedBy *string    `json:"acknowledgedBy,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	Resolved       bool       `json:"resolved"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Metadata       string     `json:"metadata,omitempty"` // JSON string for extra data
}

// AlertType constants for common alert types.
const (
	AlertTypeSecurity     = "security"
	AlertTypePerformance  = "performance"
	AlertTypeConnectivity = "connectivity"
	AlertTypeSystem       = "system"
	AlertTypeDiscovery    = "discovery"
)

// AlertSeverity constants.
const (
	AlertSeverityInfo     = "info"
	AlertSeverityWarning  = "warning"
	AlertSeverityError    = "error"
	AlertSeverityCritical = "critical"
)

// SpeedTestResult represents a speed test result.
type SpeedTestResult struct {
	ID             int64     `json:"id"`
	InterfaceName  string    `json:"interfaceName"`
	ServerName     string    `json:"serverName,omitempty"`
	ServerLocation string    `json:"serverLocation,omitempty"`
	DownloadMbps   float64   `json:"downloadMbps"`
	UploadMbps     float64   `json:"uploadMbps"`
	LatencyMs      float64   `json:"latencyMs"`
	JitterMs       float64   `json:"jitterMs,omitempty"`
	PacketLoss     float64   `json:"packetLoss,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Metadata       string    `json:"metadata,omitempty"`
}

// SurveySample represents a WiFi survey data point.
type SurveySample struct {
	ID           int64     `json:"id"`
	SurveyID     string    `json:"surveyId"`
	X            float64   `json:"x"`
	Y            float64   `json:"y"`
	SignalDBm    *int      `json:"signalDbm,omitempty"`
	NoiseDBm     *int      `json:"noiseDbm,omitempty"`
	SNRDB        *int      `json:"snrDb,omitempty"`
	Channel      *int      `json:"channel,omitempty"`
	FrequencyMHz *int      `json:"frequencyMhz,omitempty"`
	BSSID        string    `json:"bssid,omitempty"`
	SSID         string    `json:"ssid,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	NetworksJSON string    `json:"networks,omitempty"` // JSON array of visible networks
	Metadata     string    `json:"metadata,omitempty"`
}

// DNSResult represents a DNS test result.
type DNSResult struct {
	ID             int64     `json:"id"`
	InterfaceName  string    `json:"interfaceName"`
	Server         string    `json:"server"`
	Hostname       string    `json:"hostname"`
	ResponseTimeMs float64   `json:"responseTimeMs"`
	ResolvedIP     string    `json:"resolvedIp,omitempty"`
	Status         string    `json:"status"` // "success", "timeout", "error"
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// GatewayResult represents a gateway ping result.
type GatewayResult struct {
	ID            int64     `json:"id"`
	InterfaceName string    `json:"interfaceName"`
	Gateway       string    `json:"gateway"`
	LatencyMs     float64   `json:"latencyMs"`
	PacketLoss    float64   `json:"packetLoss"`
	Reachable     bool      `json:"reachable"`
	Timestamp     time.Time `json:"timestamp"`
}

// AuditLogEntry represents an audit log entry.
type AuditLogEntry struct {
	ID           int64     `json:"id"`
	Action       string    `json:"action"`
	User         string    `json:"user,omitempty"`
	ResourceType string    `json:"resourceType,omitempty"`
	ResourceID   string    `json:"resourceId,omitempty"`
	OldValueJSON string    `json:"oldValue,omitempty"`
	NewValueJSON string    `json:"newValue,omitempty"`
	IPAddress    string    `json:"ipAddress,omitempty"`
	UserAgent    string    `json:"userAgent,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// Setting represents a key-value setting.
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TimeRange represents a time range for queries.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Pagination represents pagination parameters.
type Pagination struct {
	Offset int
	Limit  int
}

// DefaultPagination returns default pagination (first 100 items).
func DefaultPagination() Pagination {
	return Pagination{Offset: 0, Limit: 100}
}
