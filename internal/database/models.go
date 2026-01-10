package database

import (
	"time"
)

// Pagination defaults.
const (
	// defaultPaginationLimit is the default number of items returned per page.
	defaultPaginationLimit = 100
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

// HealthCheckResult represents a single health check result.
type HealthCheckResult struct {
	ID             int64     `json:"id"`
	CheckType      string    `json:"checkType"`      // PING, TCP, UDP, HTTP, RTSP, DICOM, HL7, FHIR, etc.
	EndpointName   string    `json:"endpointName"`   // User-friendly name
	EndpointTarget string    `json:"endpointTarget"` // Actual target (IP, URL, etc.)
	Success        bool      `json:"success"`
	LatencyMs      float64   `json:"latencyMs,omitempty"`
	StatusCode     *int      `json:"statusCode,omitempty"` // HTTP status code if applicable
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	Metadata       string    `json:"metadata,omitempty"` // JSON string for protocol-specific data
	RecordedAt     time.Time `json:"recordedAt"`
}

// HealthCheckType constants for health check types.
const (
	HealthCheckTypePing   = "PING"
	HealthCheckTypeTCP    = "TCP"
	HealthCheckTypeUDP    = "UDP"
	HealthCheckTypeHTTP   = "HTTP"
	HealthCheckTypeHTTPS  = "HTTPS"
	HealthCheckTypeRTSP   = "RTSP"
	HealthCheckTypeDICOM  = "DICOM"
	HealthCheckTypeHL7    = "HL7"
	HealthCheckTypeFHIR   = "FHIR"
	HealthCheckTypeLTI    = "LTI"
	HealthCheckTypeLDAP   = "LDAP"
	HealthCheckTypeOPCUA  = "OPCUA"
	HealthCheckTypeModbus = "MODBUS"
)

// HealthCheckHourlyRollup represents hourly aggregated health check data.
type HealthCheckHourlyRollup struct {
	ID               int64     `json:"id"`
	CheckType        string    `json:"checkType"`
	EndpointName     string    `json:"endpointName"`
	HourBucket       time.Time `json:"hourBucket"` // Truncated to hour
	TotalChecks      int       `json:"totalChecks"`
	SuccessfulChecks int       `json:"successfulChecks"`
	AvgLatencyMs     float64   `json:"avgLatencyMs"`
	MinLatencyMs     float64   `json:"minLatencyMs"`
	MaxLatencyMs     float64   `json:"maxLatencyMs"`
	P95LatencyMs     float64   `json:"p95LatencyMs"`
}

// HealthCheckDailyRollup represents daily aggregated health check data.
type HealthCheckDailyRollup struct {
	ID                  int64     `json:"id"`
	CheckType           string    `json:"checkType"`
	EndpointName        string    `json:"endpointName"`
	DayBucket           time.Time `json:"dayBucket"` // Truncated to day
	TotalChecks         int       `json:"totalChecks"`
	SuccessfulChecks    int       `json:"successfulChecks"`
	AvgLatencyMs        float64   `json:"avgLatencyMs"`
	MinLatencyMs        float64   `json:"minLatencyMs"`
	MaxLatencyMs        float64   `json:"maxLatencyMs"`
	P95LatencyMs        float64   `json:"p95LatencyMs"`
	AvailabilityPercent float64   `json:"availabilityPercent"`
}

// HealthCheckQueryOptions specifies criteria for querying health check results.
type HealthCheckQueryOptions struct {
	CheckType    string
	EndpointName string
	TimeRange    TimeRange
	Limit        int
	Offset       int
}

// EndpointHealthScore represents the computed health score for an endpoint.
type EndpointHealthScore struct {
	EndpointName     string    `json:"endpointName"`
	CheckType        string    `json:"checkType"`
	AvailabilityPct  float64   `json:"availabilityPct"`  // Last 24h uptime percentage
	LatencyScore     float64   `json:"latencyScore"`     // 0-100 based on P95 vs threshold
	CriticalityScore float64   `json:"criticalityScore"` // User-defined 1-10 scale (normalized to 0-100)
	CompositeScore   float64   `json:"compositeScore"`   // 0.4*Avail + 0.3*Latency + 0.3*Criticality
	Status           string    `json:"status"`           // healthy, degraded, critical
	LastCheck        time.Time `json:"lastCheck"`
}

// Pagination represents pagination parameters.
type Pagination struct {
	Offset int
	Limit  int
}

// DefaultPagination returns default pagination (first 100 items).
func DefaultPagination() Pagination {
	return Pagination{Offset: 0, Limit: defaultPaginationLimit}
}
