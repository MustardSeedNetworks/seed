package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
)

// sqliteDateFormat is the strftime grouping format for trend buckets. It is a
// SQLite-dialect concern, so it lives with the SQL in the adapter.
const sqliteDateFormat = "%Y-%m-%d"

// MetricsRepo implements reporting.MetricsRepo over the diagnostics tables. The
// SQL and scanning were lifted verbatim from the reporting package
// (services_aggregator.go) when reporting was made I/O-free — Phase 3 slice
// 1b-v. The repo returns raw results (e.g. severity → count); the domain owns
// the severity-bucket and category semantics.
type MetricsRepo struct {
	db *database.DB
}

// NewMetricsRepo constructs a MetricsRepo backed by db.
func NewMetricsRepo(db *database.DB) *MetricsRepo {
	return &MetricsRepo{db: db}
}

// Compile-time assertion that the adapter satisfies reporting's port.
var _ reporting.MetricsRepo = (*MetricsRepo)(nil)

// CountDevices returns the total device count.
func (r *MetricsRepo) CountDevices(ctx context.Context) (int, error) {
	var count int
	row := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM devices")
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting devices: %w", err)
	}
	return count, nil
}

// VulnerabilitySeverityCounts returns severity → count of the open findings
// discovered since `since`; resolved and ignored findings are not counted.
func (r *MetricsRepo) VulnerabilitySeverityCounts(
	ctx context.Context,
	since time.Time,
) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT severity, COUNT(*) as count
		FROM device_vulnerabilities
		WHERE detected_at >= ? AND status NOT IN (?, ?)
		GROUP BY severity
	`, since.Format(time.RFC3339), database.VulnStatusResolved, database.VulnStatusIgnored)
	if err != nil {
		return nil, fmt.Errorf("querying vulnerability counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if scanErr := rows.Scan(&severity, &count); scanErr != nil {
			return nil, fmt.Errorf("scanning vulnerability count: %w", scanErr)
		}
		counts[severity] = count
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating vulnerability counts: %w", rowsErr)
	}

	return counts, nil
}

// PerformanceMetrics returns averaged latency / packet-loss / bandwidth /
// uptime since `since`. Gateway figures come from the telemetry series in
// metrics; uptime is the share of gateway samples that got a reply, 100% when
// there are none.
func (r *MetricsRepo) PerformanceMetrics(
	ctx context.Context,
	since time.Time,
) (reporting.PerformanceMetrics, error) {
	var perf reporting.PerformanceMetrics

	row := r.db.QueryRow(ctx, `
		SELECT
			AVG(CASE WHEN metric_type = ? THEN value END),
			AVG(CASE WHEN metric_type = ? THEN value END),
			AVG(CASE WHEN metric_type = ? THEN value END) * 100.0
		FROM metrics
		WHERE target_kind = ? AND metric_type IN (?, ?, ?) AND timestamp >= ?
	`, telemetry.GatewayLatencyMs, telemetry.GatewayLossPct, telemetry.GatewayReachable,
		telemetry.TargetKind, telemetry.GatewayLatencyMs, telemetry.GatewayLossPct, telemetry.GatewayReachable,
		since.UTC().Format(time.RFC3339))

	var avgLatency, avgPacketLoss, uptime sql.NullFloat64
	if err := row.Scan(&avgLatency, &avgPacketLoss, &uptime); err != nil {
		return perf, fmt.Errorf("querying gateway telemetry: %w", err)
	}
	perf.AvgLatencyMs = avgLatency.Float64
	perf.AvgPacketLoss = avgPacketLoss.Float64
	perf.UptimePercent = 100.0
	if uptime.Valid {
		perf.UptimePercent = uptime.Float64
	}

	row = r.db.QueryRow(ctx, `
		SELECT AVG((download_mbps + upload_mbps) / 2)
		FROM speedtest_results
		WHERE timestamp >= ?
	`, since.UTC().Format(time.RFC3339))

	var avgBandwidth sql.NullFloat64
	if err := row.Scan(&avgBandwidth); err != nil {
		return perf, fmt.Errorf("querying speed tests: %w", err)
	}
	perf.AvgBandwidthMbps = avgBandwidth.Float64

	return perf, nil
}

// TopIssues returns the highest-impact open vulnerability issues.
func (r *MetricsRepo) TopIssues(ctx context.Context) ([]reporting.IssueSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT severity, COALESCE(description, cve_id), COUNT(*) as count
		FROM device_vulnerabilities
		WHERE status NOT IN (?, ?)
		GROUP BY cve_id
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				ELSE 4
			END,
			count DESC
		LIMIT 10
	`, database.VulnStatusResolved, database.VulnStatusIgnored)
	if err != nil {
		return nil, fmt.Errorf("querying top issues: %w", err)
	}
	defer rows.Close()

	var issues []reporting.IssueSummary
	for rows.Next() {
		var issue reporting.IssueSummary
		if scanErr := rows.Scan(&issue.Severity, &issue.Description, &issue.Count); scanErr != nil {
			return nil, fmt.Errorf("scanning top issue: %w", scanErr)
		}
		issue.Category = "vulnerability"
		issues = append(issues, issue)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating top issues: %w", rowsErr)
	}

	return issues, nil
}

// Trends returns time-series points for a metric over a period.
func (r *MetricsRepo) Trends(
	ctx context.Context,
	metric, period string,
) ([]reporting.DataPoint, error) {
	// Determine time range and grouping
	now := time.Now()
	var startDate time.Time
	groupFormat, bucketLayout := sqliteDateFormat, time.DateOnly

	switch period {
	case reporting.PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
		groupFormat, bucketLayout = "%Y-%m-%d %H:00", "2006-01-02 15:04"
	case reporting.PeriodMonthly:
		startDate = now.AddDate(0, -1, 0)
	default:
		startDate = now.AddDate(0, 0, -7)
	}

	var query string
	var args []any
	switch metric {
	case "latency":
		query = fmt.Sprintf(`
			SELECT strftime('%s', timestamp) as period, AVG(value)
			FROM metrics
			WHERE target_kind = ? AND metric_type = ? AND timestamp >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
		args = append(args, telemetry.TargetKind, telemetry.GatewayLatencyMs)
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

	args = append(args, startDate.UTC().Format(time.RFC3339))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trends: %w", err)
	}
	defer rows.Close()

	var points []reporting.DataPoint
	for rows.Next() {
		var periodStr string
		var value float64
		if scanErr := rows.Scan(&periodStr, &value); scanErr != nil {
			continue
		}

		t, _ := time.Parse(bucketLayout, periodStr)
		points = append(points, reporting.DataPoint{
			Timestamp: t,
			Value:     value,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating trend data: %w", rowsErr)
	}

	return points, nil
}
