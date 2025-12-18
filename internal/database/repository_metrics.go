// Package database provides metrics repository for time-series data.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MetricsRepository provides operations for metrics data.
type MetricsRepository struct {
	db *DB
}

// Record stores a new metric data point.
func (r *MetricsRepository) Record(ctx context.Context, metric *Metric) error {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now().UTC()
	}

	result, err := r.db.Exec(ctx, `
		INSERT INTO metrics (interface_name, metric_type, value, unit, timestamp, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?)
	`, metric.InterfaceName, metric.MetricType, metric.Value, metric.Unit,
		metric.Timestamp.Format(time.RFC3339), metric.Metadata)
	if err != nil {
		return fmt.Errorf("failed to record metric: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		metric.ID = id
	}

	return nil
}

// RecordBatch stores multiple metrics in a single transaction.
func (r *MetricsRepository) RecordBatch(ctx context.Context, metrics []*Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO metrics (interface_name, metric_type, value, unit, timestamp, metadata_json)
			VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		now := time.Now().UTC()
		for _, m := range metrics {
			if m.Timestamp.IsZero() {
				m.Timestamp = now
			}
			_, err := stmt.ExecContext(ctx, m.InterfaceName, m.MetricType, m.Value, m.Unit,
				m.Timestamp.Format(time.RFC3339), m.Metadata)
			if err != nil {
				return fmt.Errorf("failed to insert metric: %w", err)
			}
		}
		return nil
	})
}

// Query retrieves metrics matching the given criteria.
func (r *MetricsRepository) Query(ctx context.Context, opts MetricQueryOptions) ([]*Metric, error) {
	query := `
		SELECT id, interface_name, metric_type, value, unit, timestamp, metadata_json
		FROM metrics
		WHERE 1=1
	`
	var args []any

	if opts.InterfaceName != "" {
		query += " AND interface_name = ?"
		args = append(args, opts.InterfaceName)
	}

	if opts.MetricType != "" {
		query += " AND metric_type = ?"
		args = append(args, opts.MetricType)
	}

	if !opts.TimeRange.Start.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.TimeRange.Start.UTC().Format(time.RFC3339))
	}

	if !opts.TimeRange.End.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.TimeRange.End.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*Metric
	for rows.Next() {
		m, err := r.scanMetric(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// MetricQueryOptions specifies criteria for querying metrics.
type MetricQueryOptions struct {
	InterfaceName string
	MetricType    string
	TimeRange     TimeRange
	Limit         int
	Offset        int
}

// GetLatest retrieves the most recent metric of the given type.
func (r *MetricsRepository) GetLatest(ctx context.Context, interfaceName, metricType string) (*Metric, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, interface_name, metric_type, value, unit, timestamp, metadata_json
		FROM metrics
		WHERE interface_name = ? AND metric_type = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, interfaceName, metricType)

	var m Metric
	var timestamp string
	var unit, metadata sql.NullString

	err := row.Scan(&m.ID, &m.InterfaceName, &m.MetricType, &m.Value, &unit, &timestamp, &metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil //nolint:nilnil // nil,nil is intentional for "not found"
		}
		return nil, fmt.Errorf("failed to get latest metric: %w", err)
	}

	if t, parseErr := time.Parse(time.RFC3339, timestamp); parseErr == nil {
		m.Timestamp = t
	}
	m.Unit = unit.String
	m.Metadata = metadata.String

	return &m, nil
}

// GetAggregates returns aggregated metrics over a time range.
func (r *MetricsRepository) GetAggregates(ctx context.Context, opts MetricAggregateOptions) (*MetricAggregate, error) {
	query := `
		SELECT
			COUNT(*) as count,
			AVG(value) as avg,
			MIN(value) as min,
			MAX(value) as max,
			SUM(value) as sum
		FROM metrics
		WHERE interface_name = ? AND metric_type = ?
	`
	args := []any{opts.InterfaceName, opts.MetricType}

	if !opts.TimeRange.Start.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.TimeRange.Start.UTC().Format(time.RFC3339))
	}

	if !opts.TimeRange.End.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.TimeRange.End.UTC().Format(time.RFC3339))
	}

	var agg MetricAggregate
	var avgVal, minVal, maxVal, sumVal sql.NullFloat64

	row := r.db.QueryRow(ctx, query, args...)
	err := row.Scan(&agg.Count, &avgVal, &minVal, &maxVal, &sumVal)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregates: %w", err)
	}

	agg.Avg = avgVal.Float64
	agg.Min = minVal.Float64
	agg.Max = maxVal.Float64
	agg.Sum = sumVal.Float64

	return &agg, nil
}

// MetricAggregateOptions specifies criteria for aggregate queries.
type MetricAggregateOptions struct {
	InterfaceName string
	MetricType    string
	TimeRange     TimeRange
}

// MetricAggregate holds aggregated metric values.
type MetricAggregate struct {
	Count int64
	Avg   float64
	Min   float64
	Max   float64
	Sum   float64
}

// DeleteOlderThan removes metrics older than the given time.
func (r *MetricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	// Ensure UTC for consistent comparison with stored timestamps
	result, err := r.db.Exec(ctx, `
		DELETE FROM metrics WHERE timestamp < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old metrics: %w", err)
	}

	return result.RowsAffected()
}

// Count returns the total number of metrics.
func (r *MetricsRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM metrics`)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count metrics: %w", err)
	}
	return count, nil
}

// GetDistinctInterfaces returns all unique interface names with metrics.
func (r *MetricsRepository) GetDistinctInterfaces(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT interface_name FROM metrics ORDER BY interface_name`)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct interfaces: %w", err)
	}
	defer rows.Close()

	var interfaces []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		interfaces = append(interfaces, name)
	}

	return interfaces, rows.Err()
}

// GetDistinctTypes returns all unique metric types.
func (r *MetricsRepository) GetDistinctTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT metric_type FROM metrics ORDER BY metric_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct types: %w", err)
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}

	return types, rows.Err()
}

// scanMetric scans a metric from rows.
func (r *MetricsRepository) scanMetric(rows *sql.Rows) (*Metric, error) {
	var m Metric
	var timestamp string
	var unit, metadata sql.NullString

	err := rows.Scan(&m.ID, &m.InterfaceName, &m.MetricType, &m.Value, &unit, &timestamp, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to scan metric: %w", err)
	}

	if t, parseErr := time.Parse(time.RFC3339, timestamp); parseErr == nil {
		m.Timestamp = t
	}
	m.Unit = unit.String
	m.Metadata = metadata.String

	return &m, nil
}

// RecordSpeedTest stores a speed test result.
func (r *MetricsRepository) RecordSpeedTest(ctx context.Context, result *SpeedTestResult) error {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now().UTC()
	}

	res, err := r.db.Exec(ctx, `
		INSERT INTO speedtest_results
		(interface_name, server_name, server_location, download_mbps, upload_mbps,
		 latency_ms, jitter_ms, packet_loss, timestamp, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, result.InterfaceName, result.ServerName, result.ServerLocation,
		result.DownloadMbps, result.UploadMbps, result.LatencyMs, result.JitterMs,
		result.PacketLoss, result.Timestamp.Format(time.RFC3339), result.Metadata)
	if err != nil {
		return fmt.Errorf("failed to record speed test: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		result.ID = id
	}

	return nil
}

// GetSpeedTestHistory retrieves speed test history for an interface.
func (r *MetricsRepository) GetSpeedTestHistory(ctx context.Context, interfaceName string, limit int) ([]*SpeedTestResult, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, interface_name, server_name, server_location, download_mbps, upload_mbps,
		       latency_ms, jitter_ms, packet_loss, timestamp, metadata_json
		FROM speedtest_results
		WHERE interface_name = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, interfaceName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query speed tests: %w", err)
	}
	defer rows.Close()

	var results []*SpeedTestResult
	for rows.Next() {
		var r SpeedTestResult
		var timestamp string
		var serverName, serverLocation, metadata sql.NullString
		var jitter, packetLoss sql.NullFloat64

		err := rows.Scan(&r.ID, &r.InterfaceName, &serverName, &serverLocation,
			&r.DownloadMbps, &r.UploadMbps, &r.LatencyMs, &jitter, &packetLoss,
			&timestamp, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan speed test: %w", err)
		}

		if t, parseErr := time.Parse(time.RFC3339, timestamp); parseErr == nil {
			r.Timestamp = t
		}
		r.ServerName = serverName.String
		r.ServerLocation = serverLocation.String
		r.JitterMs = jitter.Float64
		r.PacketLoss = packetLoss.Float64
		r.Metadata = metadata.String

		results = append(results, &r)
	}

	return results, rows.Err()
}

// RecordDNSResult stores a DNS test result.
func (r *MetricsRepository) RecordDNSResult(ctx context.Context, result *DNSResult) error {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now().UTC()
	}

	res, err := r.db.Exec(ctx, `
		INSERT INTO dns_results
		(interface_name, server, hostname, response_time_ms, resolved_ip, status, error_message, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, result.InterfaceName, result.Server, result.Hostname, result.ResponseTimeMs,
		result.ResolvedIP, result.Status, result.ErrorMessage, result.Timestamp.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to record DNS result: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		result.ID = id
	}

	return nil
}

// RecordGatewayResult stores a gateway test result.
func (r *MetricsRepository) RecordGatewayResult(ctx context.Context, result *GatewayResult) error {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now().UTC()
	}

	res, err := r.db.Exec(ctx, `
		INSERT INTO gateway_results
		(interface_name, gateway, latency_ms, packet_loss, reachable, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, result.InterfaceName, result.Gateway, result.LatencyMs, result.PacketLoss,
		boolToInt(result.Reachable), result.Timestamp.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to record gateway result: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		result.ID = id
	}

	return nil
}
