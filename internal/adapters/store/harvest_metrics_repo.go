package store

import (
	"context"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/modules/harvest"
)

// sqliteDateFormat is the strftime grouping format for trend buckets. It is a
// SQLite-dialect concern, so it lives with the SQL in the adapter.
const sqliteDateFormat = "%Y-%m-%d"

// MetricsRepo implements harvest.MetricsRepo over the diagnostics tables. The
// SQL and scanning were lifted verbatim from the harvest module
// (services_aggregator.go) when the module was made I/O-free — Phase 3 slice
// 1b-v. The repo returns raw results (e.g. severity → count); the domain owns
// the severity-bucket and category semantics.
type MetricsRepo struct {
	db *database.DB
}

// NewMetricsRepo constructs a MetricsRepo backed by db.
func NewMetricsRepo(db *database.DB) *MetricsRepo {
	return &MetricsRepo{db: db}
}

// Compile-time assertion that the adapter satisfies the module's port.
var _ harvest.MetricsRepo = (*MetricsRepo)(nil)

// CountDevices returns the total device count.
func (r *MetricsRepo) CountDevices(ctx context.Context) (int, error) {
	var count int
	row := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM devices")
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting devices: %w", err)
	}
	return count, nil
}

// VulnerabilitySeverityCounts returns severity → count discovered since `since`.
func (r *MetricsRepo) VulnerabilitySeverityCounts(
	ctx context.Context,
	since time.Time,
) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT severity, COUNT(*) as count
		FROM device_vulnerabilities
		WHERE discovered_at >= ?
		GROUP BY severity
	`, since.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("querying vulnerability counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if scanErr := rows.Scan(&severity, &count); scanErr != nil {
			continue
		}
		counts[severity] = count
	}

	return counts, nil
}

// PerformanceMetrics returns averaged latency / packet-loss / bandwidth /
// uptime since `since`. Defaults (notably 100% uptime with no data) are
// preserved from the original aggregator.
func (r *MetricsRepo) PerformanceMetrics(
	ctx context.Context,
	since time.Time,
) (harvest.PerformanceMetrics, error) {
	var perf harvest.PerformanceMetrics

	// Get average latency from gateway results
	row := r.db.QueryRow(ctx, `
		SELECT AVG(latency_ms), AVG(packet_loss)
		FROM gateway_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var avgLatency, avgPacketLoss *float64
	_ = row.Scan(&avgLatency, &avgPacketLoss)

	if avgLatency != nil {
		perf.AvgLatencyMs = *avgLatency
	}
	if avgPacketLoss != nil {
		perf.AvgPacketLoss = *avgPacketLoss
	}

	// Get average bandwidth from speedtest results
	row = r.db.QueryRow(ctx, `
		SELECT AVG((download_mbps + upload_mbps) / 2)
		FROM speedtest_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var avgBandwidth *float64
	_ = row.Scan(&avgBandwidth)
	if avgBandwidth != nil {
		perf.AvgBandwidthMbps = *avgBandwidth
	}

	// Calculate uptime (simplified: based on successful gateway checks)
	row = r.db.QueryRow(ctx, `
		SELECT
			COUNT(CASE WHEN success = 1 THEN 1 END) * 100.0 / COUNT(*)
		FROM gateway_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var uptime *float64
	_ = row.Scan(&uptime)
	if uptime != nil {
		perf.UptimePercent = *uptime
	} else {
		perf.UptimePercent = 100.0 // Default to 100% if no data
	}

	return perf, nil
}

// TopIssues returns the highest-impact vulnerability issues.
func (r *MetricsRepo) TopIssues(ctx context.Context) ([]harvest.IssueSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT severity, description, COUNT(*) as count
		FROM device_vulnerabilities
		GROUP BY description
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				ELSE 4
			END,
			count DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, fmt.Errorf("querying top issues: %w", err)
	}
	defer rows.Close()

	var issues []harvest.IssueSummary
	for rows.Next() {
		var issue harvest.IssueSummary
		if scanErr := rows.Scan(&issue.Severity, &issue.Description, &issue.Count); scanErr != nil {
			continue
		}
		issue.Category = "vulnerability"
		issues = append(issues, issue)
	}

	return issues, nil
}

// Trends returns time-series points for a metric over a period.
func (r *MetricsRepo) Trends(
	ctx context.Context,
	metric, period string,
) ([]harvest.DataPoint, error) {
	// Determine time range and grouping
	now := time.Now()
	var startDate time.Time
	var groupFormat string

	switch period {
	case harvest.PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
		groupFormat = "%Y-%m-%d %H:00"
	case harvest.PeriodWeekly:
		startDate = now.AddDate(0, 0, -7)
		groupFormat = sqliteDateFormat
	case harvest.PeriodMonthly:
		startDate = now.AddDate(0, -1, 0)
		groupFormat = sqliteDateFormat
	default:
		startDate = now.AddDate(0, 0, -7)
		groupFormat = sqliteDateFormat
	}

	var query string
	switch metric {
	case "latency":
		query = fmt.Sprintf(`
			SELECT strftime('%s', timestamp) as period, AVG(latency_ms)
			FROM gateway_results
			WHERE timestamp >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
	case "bandwidth":
		query = fmt.Sprintf(`
			SELECT strftime('%s', timestamp) as period, AVG(download_mbps)
			FROM speedtest_results
			WHERE timestamp >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
	case "devices":
		query = fmt.Sprintf(`
			SELECT strftime('%s', last_seen) as period, COUNT(*)
			FROM devices
			WHERE last_seen >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
	default:
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}

	rows, err := r.db.Query(ctx, query, startDate.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("querying trends: %w", err)
	}
	defer rows.Close()

	var points []harvest.DataPoint
	for rows.Next() {
		var periodStr string
		var value float64
		if scanErr := rows.Scan(&periodStr, &value); scanErr != nil {
			continue
		}

		t, _ := time.Parse("2006-01-02", periodStr)
		points = append(points, harvest.DataPoint{
			Timestamp: t,
			Value:     value,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating trend data: %w", rowsErr)
	}

	return points, nil
}
