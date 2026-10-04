package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
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
		defer func() { _ = stmt.Close() }()

		now := time.Now().UTC()
		for _, m := range metrics {
			if m.Timestamp.IsZero() {
				m.Timestamp = now
			}
			_, execErr := stmt.ExecContext(ctx, m.InterfaceName, m.MetricType, m.Value, m.Unit,
				m.Timestamp.Format(time.RFC3339), m.Metadata)
			if execErr != nil {
				return fmt.Errorf("failed to insert metric: %w", execErr)
			}
		}
		return nil
	})
}

// RecordInterfaceRates stores each rate's six points in one transaction,
// keyed by client and by target_id "<polling target ID>/<ifIndex>".
// interface_name repeats target_id: the hourly and daily rollups are unique
// on interface_name, and an ifName is only unique within one device.
func (r *MetricsRepository) RecordInterfaceRates(ctx context.Context, rates []ifrate.Rate) error {
	if len(rates) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO metrics
			  (interface_name, metric_type, value, unit, timestamp,
			   client_id, target_kind, target_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare interface rate insert: %w", err)
		}
		defer func() { _ = stmt.Close() }()

		for _, rate := range rates {
			targetID := rate.TargetID + "/" + strconv.FormatUint(uint64(rate.IfIndex), 10)
			at := rate.At.UTC().Format(time.RFC3339)
			for _, pt := range rate.Points() {
				if _, execErr := stmt.ExecContext(ctx, targetID, pt.Type, pt.Value, pt.Unit, at,
					rate.ClientID, ifrate.TargetKind, targetID); execErr != nil {
					return fmt.Errorf("insert interface rate: %w", execErr)
				}
			}
		}
		return nil
	})
}

// InterfaceErrorPeaks reads the error and discard rates RecordInterfaceRates
// stored for one interface in the window (from, to].
func (r *MetricsRepository) InterfaceErrorPeaks(
	ctx context.Context, clientID, targetID string, ifIndex uint32, from, to time.Time,
) (ifrate.ErrorPeaks, error) {
	rows, err := r.db.Query(ctx, `
		SELECT metric_type, MAX(value), COUNT(*)
		FROM metrics
		WHERE interface_name = ? AND metric_type IN (?, ?, ?, ?)
		  AND timestamp > ? AND timestamp <= ?
		  AND client_id = ? AND target_kind = ?
		GROUP BY metric_type
	`, targetID+"/"+strconv.FormatUint(uint64(ifIndex), 10),
		ifrate.MetricInErrors, ifrate.MetricOutErrors, ifrate.MetricInDiscards, ifrate.MetricOutDiscards,
		from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339),
		clientID, ifrate.TargetKind)
	if err != nil {
		return ifrate.ErrorPeaks{}, fmt.Errorf("query interface error peaks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var peaks ifrate.ErrorPeaks
	for rows.Next() {
		var metric string
		var peak float64
		var polls int
		if scanErr := rows.Scan(&metric, &peak, &polls); scanErr != nil {
			return ifrate.ErrorPeaks{}, fmt.Errorf("scan interface error peak: %w", scanErr)
		}
		// Every rated poll stores all four, so each count is the poll count.
		peaks.Polls = polls
		switch metric {
		case ifrate.MetricInErrors:
			peaks.InErrors = peak
		case ifrate.MetricOutErrors:
			peaks.OutErrors = peak
		case ifrate.MetricInDiscards:
			peaks.InDiscards = peak
		case ifrate.MetricOutDiscards:
			peaks.OutDiscards = peak
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return ifrate.ErrorPeaks{}, fmt.Errorf("interface error peaks rows: %w", rowsErr)
	}
	return peaks, nil
}

// RecordTelemetry stores one telemetry sample's points in one transaction,
// keyed by target_id = the interface name.
func (r *MetricsRepository) RecordTelemetry(ctx context.Context, s telemetry.Sample) error {
	if len(s.Points) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO metrics
			  (interface_name, metric_type, value, unit, timestamp, target_kind, target_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare telemetry insert: %w", err)
		}
		defer func() { _ = stmt.Close() }()

		at := s.At.UTC().Format(time.RFC3339)
		for _, pt := range s.Points {
			if _, execErr := stmt.ExecContext(ctx, s.Interface, pt.Type, pt.Value, pt.Unit, at,
				telemetry.TargetKind, s.Interface); execErr != nil {
				return fmt.Errorf("insert telemetry point: %w", execErr)
			}
		}
		return nil
	})
}

// Query retrieves metrics matching opts (interface, metric type, time
// range, limit/offset), newest first.
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
		query += sqlAndTimestampLte
		args = append(args, opts.TimeRange.End.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += sqlOffset
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var metrics []*Metric
	for rows.Next() {
		m, scanErr := r.scanMetric(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		metrics = append(metrics, m)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return metrics, nil
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
func (r *MetricsRepository) GetLatest(
	ctx context.Context,
	interfaceName, metricType string,
) (*Metric, error) {
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

// GetAggregates computes count/avg/min/max/sum over metrics matching opts'
// interface, metric type, and time range in a single SQL aggregate query.
func (r *MetricsRepository) GetAggregates(
	ctx context.Context,
	opts MetricAggregateOptions,
) (*MetricAggregate, error) {
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
		query += sqlAndTimestampLte
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

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}
	return affected, nil
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
	rows, err := r.db.Query(
		ctx,
		`SELECT DISTINCT interface_name FROM metrics ORDER BY interface_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct interfaces: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var interfaces []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scan interface name: %w", scanErr)
		}
		interfaces = append(interfaces, name)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return interfaces, nil
}

// GetDistinctTypes returns all unique metric types.
func (r *MetricsRepository) GetDistinctTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT metric_type FROM metrics ORDER BY metric_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct types: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var types []string
	for rows.Next() {
		var t string
		if scanErr := rows.Scan(&t); scanErr != nil {
			return nil, fmt.Errorf("scan metric type: %w", scanErr)
		}
		types = append(types, t)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return types, nil
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
