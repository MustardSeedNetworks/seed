package harvest

import (
	"time"
)

// Report represents a generated report.
type Report struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        ReportType   `json:"type"`
	Format      ExportFormat `json:"format"`
	Template    string       `json:"template,omitempty"`
	Status      ReportStatus `json:"status"`
	FilePath    string       `json:"filePath,omitempty"`
	FileSize    int64        `json:"fileSize,omitempty"`
	Parameters  ReportParams `json:"parameters,omitzero"`
	CreatedAt   time.Time    `json:"createdAt"`
	CompletedAt *time.Time   `json:"completedAt,omitempty"`
	ExpiresAt   *time.Time   `json:"expiresAt,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// ReportType categorizes reports.
type ReportType string

// ReportType values.
const (
	ReportTypeExecutive     ReportType = "executive"     // High-level summary
	ReportTypeDetailed      ReportType = "detailed"      // Full technical details
	ReportTypeVulnerability ReportType = "vulnerability" // Security-focused
	ReportTypeCompliance    ReportType = "compliance"    // Compliance audit
	ReportTypeInventory     ReportType = "inventory"     // Asset inventory
	ReportTypePerformance   ReportType = "performance"   // Performance metrics
	ReportTypeIncident      ReportType = "incident"      // Incident response
	ReportTypeCustom        ReportType = "custom"        // Custom template
)

// ExportFormat specifies output format.
type ExportFormat string

// ExportFormat values.
const (
	FormatPDF      ExportFormat = "pdf"
	FormatHTML     ExportFormat = "html"
	FormatCSV      ExportFormat = "csv"
	FormatJSON     ExportFormat = "json"
	FormatExcel    ExportFormat = "xlsx"
	FormatMarkdown ExportFormat = "md"
)

// ReportStatus tracks report generation status.
type ReportStatus string

// ReportStatus values.
const (
	StatusPending    ReportStatus = "pending"
	StatusGenerating ReportStatus = "generating"
	StatusComplete   ReportStatus = "complete"
	StatusFailed     ReportStatus = "failed"
	StatusExpired    ReportStatus = "expired"
)

// ReportParams contains report generation parameters.
type ReportParams struct {
	DateRange       *DateRange        `json:"dateRange,omitempty"`
	Filters         map[string]string `json:"filters,omitempty"`
	IncludeSections []string          `json:"includeSections,omitempty"`
	ExcludeSections []string          `json:"excludeSections,omitempty"`
	CustomData      map[string]any    `json:"customData,omitempty"`
}

// DateRange specifies a time range for reports.
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Template represents a report template.
type Template struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        ReportType        `json:"type"`
	Formats     []ExportFormat    `json:"formats"`
	Sections    []TemplateSection `json:"sections"`
	IsBuiltIn   bool              `json:"isBuiltIn"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// TemplateSection defines a section in a template.
type TemplateSection struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Optional bool   `json:"optional"`
	Order    int    `json:"order"`
}

// ScheduledReport represents a recurring report schedule.
type ScheduledReport struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Template   string       `json:"template"`
	Format     ExportFormat `json:"format"`
	Schedule   Schedule     `json:"schedule"`
	Parameters ReportParams `json:"parameters,omitzero"`
	Recipients []Recipient  `json:"recipients,omitempty"`
	Enabled    bool         `json:"enabled"`
	LastRun    *time.Time   `json:"lastRun,omitempty"`
	NextRun    *time.Time   `json:"nextRun,omitempty"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}

// Schedule defines when a report runs.
type Schedule struct {
	Frequency  ScheduleFrequency `json:"frequency"`
	DayOfWeek  *int              `json:"dayOfWeek,omitempty"`  // 0-6
	DayOfMonth *int              `json:"dayOfMonth,omitempty"` // 1-31
	Hour       int               `json:"hour"`                 // 0-23
	Minute     int               `json:"minute"`               // 0-59
	Timezone   string            `json:"timezone"`
}

// ScheduleFrequency specifies how often a report runs.
type ScheduleFrequency string

// ScheduleFrequency values.
const (
	FrequencyDaily   ScheduleFrequency = "daily"
	FrequencyWeekly  ScheduleFrequency = "weekly"
	FrequencyMonthly ScheduleFrequency = "monthly"
)

// Recipient represents a report recipient.
type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// AggregatedData contains pre-aggregated data for reports.
type AggregatedData struct {
	Period      string                 `json:"period"` // daily, weekly, monthly
	StartDate   time.Time              `json:"startDate"`
	EndDate     time.Time              `json:"endDate"`
	DeviceCount int                    `json:"deviceCount"`
	VulnCount   VulnCounts             `json:"vulnCounts"`
	Performance PerformanceMetrics     `json:"performance"`
	Incidents   []IncidentSummary      `json:"incidents,omitempty"`
	TopIssues   []IssueSummary         `json:"topIssues,omitempty"`
	Trends      map[string][]DataPoint `json:"trends,omitempty"`
}

// VulnCounts contains vulnerability counts by severity.
type VulnCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

// PerformanceMetrics contains aggregated performance data.
type PerformanceMetrics struct {
	AvgLatencyMs     float64 `json:"avgLatencyMs"`
	AvgPacketLoss    float64 `json:"avgPacketLossPercent"`
	AvgBandwidthMbps float64 `json:"avgBandwidthMbps"`
	UptimePercent    float64 `json:"uptimePercent"`
}

// IncidentSummary summarizes a security incident.
type IncidentSummary struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	DetectedAt time.Time  `json:"detectedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
}

// IssueSummary summarizes a top issue.
type IssueSummary struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	Severity    string `json:"severity"`
}

// DataPoint represents a time-series data point.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// ExportRequest represents a data export request.
type ExportRequest struct {
	Type      string            `json:"type"` // devices, vulnerabilities, etc.
	Format    ExportFormat      `json:"format"`
	Filters   map[string]string `json:"filters,omitempty"`
	Fields    []string          `json:"fields,omitempty"`
	DateRange *DateRange        `json:"dateRange,omitempty"`
}

// ExportResult contains export operation result.
type ExportResult struct {
	ID          string       `json:"id"`
	FilePath    string       `json:"filePath"`
	FileSize    int64        `json:"fileSize"`
	RecordCount int          `json:"recordCount"`
	Format      ExportFormat `json:"format"`
	CreatedAt   time.Time    `json:"createdAt"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	DownloadURL string       `json:"downloadUrl,omitempty"`
}
