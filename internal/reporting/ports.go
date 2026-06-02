package reporting

import (
	"context"
	"time"
)

// ports.go declares the infrastructure interfaces the reporting package
// depends on. The package is persistence-free: it owns these ports, and the
// concrete I/O lives in internal/reporting/store (inward-only — the store
// adapter imports reporting, never the reverse).

// ReportRepo persists report records. The SQLite implementation lives in
// internal/reporting/store. The service depends on this interface so report
// orchestration can be unit-tested with a fake repo (no database).
type ReportRepo interface {
	// GetReport returns the report row with the given id.
	GetReport(ctx context.Context, id string) (*Report, error)
	// ListReports returns all report rows, newest first.
	ListReports(ctx context.Context) ([]Report, error)
	// SaveReport upserts a report row.
	SaveReport(ctx context.Context, r *Report) error
	// DeleteReport removes the report row only; file cleanup is the service's
	// orchestration concern.
	DeleteReport(ctx context.Context, id string) error
}

// ScheduleRepo persists scheduled-report records. The scheduler keeps its
// in-memory map and tick loop; only the row CRUD crosses this port.
type ScheduleRepo interface {
	// ListSchedules returns every persisted scheduled report.
	ListSchedules(ctx context.Context) ([]ScheduledReport, error)
	// SaveSchedule upserts a scheduled-report row.
	SaveSchedule(ctx context.Context, sr *ScheduledReport) error
	// DeleteSchedule removes a scheduled-report row.
	DeleteSchedule(ctx context.Context, id string) error
}

// MetricsRepo reads the aggregate metrics that feed reports. It returns raw
// query results; the domain (AggregatorService) owns the severity-bucket and
// category semantics so the meaning of "critical" stays in the module.
type MetricsRepo interface {
	// CountDevices returns the total device count.
	CountDevices(ctx context.Context) (int, error)
	// VulnerabilitySeverityCounts returns severity → count discovered since the
	// given time. The domain maps these onto VulnCounts.
	VulnerabilitySeverityCounts(ctx context.Context, since time.Time) (map[string]int, error)
	// PerformanceMetrics returns averaged latency / packet-loss / bandwidth /
	// uptime since the given time.
	PerformanceMetrics(ctx context.Context, since time.Time) (PerformanceMetrics, error)
	// TopIssues returns the highest-impact vulnerability issues.
	TopIssues(ctx context.Context) ([]IssueSummary, error)
	// Trends returns time-series points for a metric over a period.
	Trends(ctx context.Context, metric, period string) ([]DataPoint, error)
}

// ExportRepo reads raw rows for bulk data export. Rows are returned as generic
// maps because the export serializers (JSON/CSV) are schema-agnostic.
type ExportRepo interface {
	// ExportDevices returns all device rows, newest-seen first.
	ExportDevices(ctx context.Context) ([]map[string]any, error)
	// ExportVulnerabilities returns all vulnerability rows joined to device IPs.
	ExportVulnerabilities(ctx context.Context) ([]map[string]any, error)
}
