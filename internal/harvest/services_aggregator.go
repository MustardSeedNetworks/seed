package harvest

// services_aggregator.go contains AggregatorService: rolls up devices,
// vulnerabilities, performance, and top issues over a period; also exposes
// GetTrends for time-series chart data.

import (
	"context"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// AggregatorService aggregates data for reports.
type AggregatorService struct {
	cfg *config.Config
	db  *database.DB
}

// NewAggregatorService creates a new aggregator service.
func NewAggregatorService(cfg *config.Config, db *database.DB) *AggregatorService {
	return &AggregatorService{
		cfg: cfg,
		db:  db,
	}
}

// Aggregate collects and aggregates data for a time period.
func (s *AggregatorService) Aggregate(
	ctx context.Context,
	period, _, _ string,
) (*AggregatedData, error) {
	// Calculate date range based on period
	now := time.Now()
	var startDate time.Time

	switch period {
	case PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
	case PeriodWeekly:
		startDate = now.AddDate(0, 0, -7)
	case PeriodMonthly:
		startDate = now.AddDate(0, -1, 0)
	default:
		startDate = now.AddDate(0, 0, -7) // Default to weekly
	}

	data := &AggregatedData{
		Period:    period,
		StartDate: startDate,
		EndDate:   now,
	}

	// Aggregate device count
	row := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM devices")
	_ = row.Scan(&data.DeviceCount)

	// Aggregate vulnerability counts
	s.aggregateVulnerabilities(ctx, data, startDate)

	// Aggregate performance metrics
	s.aggregatePerformance(ctx, data, startDate)

	// Get top issues
	s.aggregateTopIssues(ctx, data)

	return data, nil
}

func (s *AggregatorService) aggregateVulnerabilities(
	ctx context.Context,
	data *AggregatedData,
	since time.Time,
) {
	rows, err := s.db.Query(ctx, `
		SELECT severity, COUNT(*) as count
		FROM device_vulnerabilities
		WHERE discovered_at >= ?
		GROUP BY severity
	`, since.Format(time.RFC3339))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int
		if scanErr := rows.Scan(&severity, &count); scanErr != nil {
			continue
		}

		switch severity {
		case statusCritical:
			data.VulnCount.Critical = count
		case "high":
			data.VulnCount.High = count
		case "medium":
			data.VulnCount.Medium = count
		case "low":
			data.VulnCount.Low = count
		}
		data.VulnCount.Total += count
	}
}

func (s *AggregatorService) aggregatePerformance(
	ctx context.Context,
	data *AggregatedData,
	since time.Time,
) {
	// Get average latency from gateway results
	row := s.db.QueryRow(ctx, `
		SELECT AVG(latency_ms), AVG(packet_loss)
		FROM gateway_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var avgLatency, avgPacketLoss *float64
	_ = row.Scan(&avgLatency, &avgPacketLoss)

	if avgLatency != nil {
		data.Performance.AvgLatencyMs = *avgLatency
	}
	if avgPacketLoss != nil {
		data.Performance.AvgPacketLoss = *avgPacketLoss
	}

	// Get average bandwidth from speedtest results
	row = s.db.QueryRow(ctx, `
		SELECT AVG((download_mbps + upload_mbps) / 2)
		FROM speedtest_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var avgBandwidth *float64
	_ = row.Scan(&avgBandwidth)
	if avgBandwidth != nil {
		data.Performance.AvgBandwidthMbps = *avgBandwidth
	}

	// Calculate uptime (simplified: based on successful gateway checks)
	row = s.db.QueryRow(ctx, `
		SELECT
			COUNT(CASE WHEN success = 1 THEN 1 END) * 100.0 / COUNT(*)
		FROM gateway_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var uptime *float64
	_ = row.Scan(&uptime)
	if uptime != nil {
		data.Performance.UptimePercent = *uptime
	} else {
		data.Performance.UptimePercent = 100.0 // Default to 100% if no data
	}
}

func (s *AggregatorService) aggregateTopIssues(ctx context.Context, data *AggregatedData) {
	rows, err := s.db.Query(ctx, `
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
		return
	}
	defer rows.Close()

	for rows.Next() {
		var issue IssueSummary
		if scanErr := rows.Scan(&issue.Severity, &issue.Description, &issue.Count); scanErr != nil {
			continue
		}
		issue.Category = "vulnerability"
		data.TopIssues = append(data.TopIssues, issue)
	}
}

// GetTrends retrieves trend data for a metric.
func (s *AggregatorService) GetTrends(
	ctx context.Context,
	metric, period string,
) ([]DataPoint, error) {
	// Determine time range and grouping
	now := time.Now()
	var startDate time.Time
	var groupFormat string

	switch period {
	case PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
		groupFormat = "%Y-%m-%d %H:00"
	case PeriodWeekly:
		startDate = now.AddDate(0, 0, -7)
		groupFormat = sqliteDateFormat
	case PeriodMonthly:
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
	case entityDevices:
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

	rows, err := s.db.Query(ctx, query, startDate.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("querying trends: %w", err)
	}
	defer rows.Close()

	var points []DataPoint
	for rows.Next() {
		var periodStr string
		var value float64
		if scanErr := rows.Scan(&periodStr, &value); scanErr != nil {
			continue
		}

		t, _ := time.Parse("2006-01-02", periodStr)
		points = append(points, DataPoint{
			Timestamp: t,
			Value:     value,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating trend data: %w", rowsErr)
	}

	return points, nil
}
