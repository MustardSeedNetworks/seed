package database

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// SQL clause constants and other magic number constants.
const (
	sqlRecordedAtGTE = " AND recorded_at >= ?"
	sqlRecordedAtLTE = " AND recorded_at <= ?"

	// percentageMultiplier is used to convert decimal to percentage.
	percentageMultiplier = 100.0
	// p95Percentile is the 95th percentile value for latency calculation.
	p95Percentile = 0.95
)

// HealthCheckRepository provides operations for health check data.
type HealthCheckRepository struct {
	db *DB
}

// Record stores a new health check result.
func (r *HealthCheckRepository) Record(ctx context.Context, result *HealthCheckResult) error {
	if result.RecordedAt.IsZero() {
		result.RecordedAt = time.Now().UTC()
	}

	var statusCode sql.NullInt64
	if result.StatusCode != nil {
		statusCode.Int64 = int64(*result.StatusCode)
		statusCode.Valid = true
	}

	res, err := r.db.Exec(ctx, `
		INSERT INTO health_check_results
		(check_type, endpoint_name, endpoint_target, success, latency_ms, status_code, error_message, metadata_json, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, result.CheckType, result.EndpointName, result.EndpointTarget,
		boolToInt(result.Success), result.LatencyMs, statusCode,
		result.ErrorMessage, result.Metadata, result.RecordedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to record health check result: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		result.ID = id
	}

	return nil
}

// RecordBatch stores multiple health check results in a single transaction.
func (r *HealthCheckRepository) RecordBatch(ctx context.Context, results []*HealthCheckResult) error {
	if len(results) == 0 {
		return nil
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO health_check_results
			(check_type, endpoint_name, endpoint_target, success, latency_ms, status_code, error_message, metadata_json, recorded_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer func() { _ = stmt.Close() }()

		now := time.Now().UTC()
		for _, result := range results {
			if result.RecordedAt.IsZero() {
				result.RecordedAt = now
			}

			var statusCode sql.NullInt64
			if result.StatusCode != nil {
				statusCode.Int64 = int64(*result.StatusCode)
				statusCode.Valid = true
			}

			_, execErr := stmt.ExecContext(ctx, result.CheckType, result.EndpointName,
				result.EndpointTarget, boolToInt(result.Success), result.LatencyMs,
				statusCode, result.ErrorMessage, result.Metadata,
				result.RecordedAt.Format(time.RFC3339))
			if execErr != nil {
				return fmt.Errorf("failed to insert health check result: %w", execErr)
			}
		}
		return nil
	})
}

// Query retrieves health check results matching the given criteria.
func (r *HealthCheckRepository) Query(ctx context.Context, opts HealthCheckQueryOptions) ([]*HealthCheckResult, error) {
	query := `
		SELECT id, check_type, endpoint_name, endpoint_target, success, latency_ms,
		       status_code, error_message, metadata_json, recorded_at
		FROM health_check_results
		WHERE 1=1
	`
	var args []any

	if opts.CheckType != "" {
		query += " AND check_type = ?"
		args = append(args, opts.CheckType)
	}

	if opts.EndpointName != "" {
		query += " AND endpoint_name = ?"
		args = append(args, opts.EndpointName)
	}

	if !opts.TimeRange.Start.IsZero() {
		query += sqlRecordedAtGTE
		args = append(args, opts.TimeRange.Start.UTC().Format(time.RFC3339))
	}

	if !opts.TimeRange.End.IsZero() {
		query += sqlRecordedAtLTE
		args = append(args, opts.TimeRange.End.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY recorded_at DESC"

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
		return nil, fmt.Errorf("failed to query health check results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*HealthCheckResult
	for rows.Next() {
		result, scanErr := r.scanResult(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, result)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return results, nil
}

// GetLatest retrieves the most recent health check result for an endpoint.
func (r *HealthCheckRepository) GetLatest(
	ctx context.Context,
	checkType, endpointName string,
) (*HealthCheckResult, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, check_type, endpoint_name, endpoint_target, success, latency_ms,
		       status_code, error_message, metadata_json, recorded_at
		FROM health_check_results
		WHERE check_type = ? AND endpoint_name = ?
		ORDER BY recorded_at DESC
		LIMIT 1
	`, checkType, endpointName)

	var result HealthCheckResult
	var timestamp string
	var statusCode sql.NullInt64
	var errorMsg, metadata sql.NullString
	var success int

	err := row.Scan(&result.ID, &result.CheckType, &result.EndpointName,
		&result.EndpointTarget, &success, &result.LatencyMs,
		&statusCode, &errorMsg, &metadata, &timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil //nolint:nilnil // nil,nil is intentional for "not found"
		}
		return nil, fmt.Errorf("failed to get latest health check result: %w", err)
	}

	result.Success = success != 0
	if statusCode.Valid {
		sc := int(statusCode.Int64)
		result.StatusCode = &sc
	}
	result.ErrorMessage = errorMsg.String
	result.Metadata = metadata.String
	if t, parseErr := time.Parse(time.RFC3339, timestamp); parseErr == nil {
		result.RecordedAt = t
	}

	return &result, nil
}

// GetLatestForAllEndpoints retrieves the most recent result for each endpoint.
func (r *HealthCheckRepository) GetLatestForAllEndpoints(ctx context.Context) ([]*HealthCheckResult, error) {
	// Use a subquery to get the most recent result for each endpoint
	rows, err := r.db.Query(ctx, `
		SELECT h.id, h.check_type, h.endpoint_name, h.endpoint_target, h.success,
		       h.latency_ms, h.status_code, h.error_message, h.metadata_json, h.recorded_at
		FROM health_check_results h
		INNER JOIN (
			SELECT check_type, endpoint_name, MAX(recorded_at) as max_recorded
			FROM health_check_results
			GROUP BY check_type, endpoint_name
		) latest ON h.check_type = latest.check_type
		        AND h.endpoint_name = latest.endpoint_name
		        AND h.recorded_at = latest.max_recorded
		ORDER BY h.check_type, h.endpoint_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest health check results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*HealthCheckResult
	for rows.Next() {
		result, scanErr := r.scanResult(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, result)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return results, nil
}

// GetAvailability calculates availability percentage for an endpoint over a time range.
func (r *HealthCheckRepository) GetAvailability(
	ctx context.Context,
	checkType, endpointName string,
	timeRange TimeRange,
) (float64, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful
		FROM health_check_results
		WHERE check_type = ? AND endpoint_name = ?
	`
	args := []any{checkType, endpointName}

	if !timeRange.Start.IsZero() {
		query += sqlRecordedAtGTE
		args = append(args, timeRange.Start.UTC().Format(time.RFC3339))
	}

	if !timeRange.End.IsZero() {
		query += sqlRecordedAtLTE
		args = append(args, timeRange.End.UTC().Format(time.RFC3339))
	}

	var total, successful int64
	row := r.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&total, &successful); err != nil {
		return 0, fmt.Errorf("failed to get availability: %w", err)
	}

	if total == 0 {
		return percentageMultiplier, nil // No data = 100% available (or should we return 0?)
	}

	return float64(successful) / float64(total) * percentageMultiplier, nil
}

// appendTimeRangeFilters adds time range filters to a query.
func appendTimeRangeFilters(query string, args []any, tr TimeRange) (string, []any) {
	if !tr.Start.IsZero() {
		query += sqlRecordedAtGTE
		args = append(args, tr.Start.UTC().Format(time.RFC3339))
	}
	if !tr.End.IsZero() {
		query += sqlRecordedAtLTE
		args = append(args, tr.End.UTC().Format(time.RFC3339))
	}
	return query, args
}

// calculateP95 calculates the P95 latency from a sorted slice of latencies.
func calculateP95(latencies []float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	sort.Float64s(latencies)
	p95Index := int(float64(len(latencies)) * p95Percentile)
	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	return latencies[p95Index]
}

// fetchLatenciesForP95 fetches all latencies for P95 calculation.
func (r *HealthCheckRepository) fetchLatenciesForP95(
	ctx context.Context,
	checkType, endpointName string,
	timeRange TimeRange,
) ([]float64, error) {
	query := `
		SELECT latency_ms
		FROM health_check_results
		WHERE check_type = ? AND endpoint_name = ? AND success = 1
	`
	args := []any{checkType, endpointName}
	query, args = appendTimeRangeFilters(query, args, timeRange)
	query += " ORDER BY latency_ms ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query latencies for P95: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var latencies []float64
	for rows.Next() {
		var lat float64
		if scanErr := rows.Scan(&lat); scanErr != nil {
			return nil, fmt.Errorf("failed to scan latency: %w", scanErr)
		}
		latencies = append(latencies, lat)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return nil, fmt.Errorf("failed to iterate latencies: %w", rowErr)
	}

	return latencies, nil
}

// GetLatencyStats calculates latency statistics for an endpoint over a time range.
func (r *HealthCheckRepository) GetLatencyStats(
	ctx context.Context,
	checkType, endpointName string,
	timeRange TimeRange,
) (*LatencyStats, error) {
	// First get avg, min, max
	query := `
		SELECT
			AVG(latency_ms) as avg,
			MIN(latency_ms) as min,
			MAX(latency_ms) as max,
			COUNT(*) as count
		FROM health_check_results
		WHERE check_type = ? AND endpoint_name = ? AND success = 1
	`
	args := []any{checkType, endpointName}
	query, args = appendTimeRangeFilters(query, args, timeRange)

	var stats LatencyStats
	var avgVal, minVal, maxVal sql.NullFloat64
	row := r.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&avgVal, &minVal, &maxVal, &stats.Count); err != nil {
		return nil, fmt.Errorf("failed to get latency stats: %w", err)
	}

	stats.AvgMs = avgVal.Float64
	stats.MinMs = minVal.Float64
	stats.MaxMs = maxVal.Float64

	// Calculate P95 if we have data
	if stats.Count > 0 {
		latencies, err := r.fetchLatenciesForP95(ctx, checkType, endpointName, timeRange)
		if err != nil {
			return nil, err
		}
		stats.P95Ms = calculateP95(latencies)
	}

	return &stats, nil
}

// LatencyStats holds latency statistics.
type LatencyStats struct {
	Count int64
	AvgMs float64
	MinMs float64
	MaxMs float64
	P95Ms float64
}

// DeleteOlderThan removes health check results older than the given time.
func (r *HealthCheckRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM health_check_results WHERE recorded_at < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old health check results: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}
	return affected, nil
}

// CreateHourlyRollup creates or updates an hourly rollup for the given hour.
func (r *HealthCheckRepository) CreateHourlyRollup(
	ctx context.Context,
	checkType, endpointName string,
	hourBucket time.Time,
) error {
	// Calculate the hour boundaries
	hourStart := hourBucket.Truncate(time.Hour)
	hourEnd := hourStart.Add(time.Hour)

	// Get statistics for this hour
	stats, err := r.GetLatencyStats(ctx, checkType, endpointName, TimeRange{Start: hourStart, End: hourEnd})
	if err != nil {
		return fmt.Errorf("failed to get stats for rollup: %w", err)
	}

	// Get availability for this hour
	avail, err := r.GetAvailability(ctx, checkType, endpointName, TimeRange{Start: hourStart, End: hourEnd})
	if err != nil {
		return fmt.Errorf("failed to get availability for rollup: %w", err)
	}

	// Count total checks
	var totalChecks, successfulChecks int64
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*), SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END)
		FROM health_check_results
		WHERE check_type = ? AND endpoint_name = ?sqlRecordedAtGTE AND recorded_at < ?
	`, checkType, endpointName, hourStart.Format(time.RFC3339), hourEnd.Format(time.RFC3339))
	if scanErr := row.Scan(&totalChecks, &successfulChecks); scanErr != nil {
		return fmt.Errorf("failed to count checks: %w", scanErr)
	}

	if totalChecks == 0 {
		return nil // No data for this hour
	}

	// Upsert the rollup
	_, err = r.db.Exec(ctx, `
		INSERT INTO health_check_rollups_hourly
		(check_type, endpoint_name, hour_bucket, total_checks, successful_checks,
		 avg_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(check_type, endpoint_name, hour_bucket) DO UPDATE SET
			total_checks = excluded.total_checks,
			successful_checks = excluded.successful_checks,
			avg_latency_ms = excluded.avg_latency_ms,
			min_latency_ms = excluded.min_latency_ms,
			max_latency_ms = excluded.max_latency_ms,
			p95_latency_ms = excluded.p95_latency_ms
	`, checkType, endpointName, hourStart.Format(time.RFC3339),
		totalChecks, successfulChecks,
		stats.AvgMs, stats.MinMs, stats.MaxMs, stats.P95Ms)
	if err != nil {
		return fmt.Errorf("failed to upsert hourly rollup: %w", err)
	}

	_ = avail // Used for daily rollup, not hourly
	return nil
}

// CreateDailyRollup creates or updates a daily rollup for the given day.
func (r *HealthCheckRepository) CreateDailyRollup(
	ctx context.Context,
	checkType, endpointName string,
	dayBucket time.Time,
) error {
	// Calculate the day boundaries (UTC)
	dayStart := time.Date(dayBucket.Year(), dayBucket.Month(), dayBucket.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	// Aggregate from hourly rollups if available, otherwise from raw data
	row := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(total_checks), 0),
			COALESCE(SUM(successful_checks), 0),
			AVG(avg_latency_ms),
			MIN(min_latency_ms),
			MAX(max_latency_ms),
			AVG(p95_latency_ms)
		FROM health_check_rollups_hourly
		WHERE check_type = ? AND endpoint_name = ? AND hour_bucket >= ? AND hour_bucket < ?
	`, checkType, endpointName, dayStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))

	var totalChecks, successfulChecks int64
	var avgLatency, minLatency, maxLatency, p95Latency sql.NullFloat64
	if scanErr := row.Scan(
		&totalChecks,
		&successfulChecks,
		&avgLatency,
		&minLatency,
		&maxLatency,
		&p95Latency,
	); scanErr != nil {
		return fmt.Errorf("failed to aggregate hourly rollups: %w", scanErr)
	}

	// If no hourly data, try raw data
	if totalChecks == 0 {
		stats, err := r.GetLatencyStats(ctx, checkType, endpointName, TimeRange{Start: dayStart, End: dayEnd})
		if err != nil {
			return fmt.Errorf("failed to get stats for daily rollup: %w", err)
		}

		row = r.db.QueryRow(ctx, `
			SELECT COUNT(*), SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END)
			FROM health_check_results
			WHERE check_type = ? AND endpoint_name = ?sqlRecordedAtGTE AND recorded_at < ?
		`, checkType, endpointName, dayStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))
		if scanErr := row.Scan(&totalChecks, &successfulChecks); scanErr != nil {
			return fmt.Errorf("failed to count checks: %w", scanErr)
		}

		if totalChecks == 0 {
			return nil // No data for this day
		}

		avgLatency.Float64 = stats.AvgMs
		minLatency.Float64 = stats.MinMs
		maxLatency.Float64 = stats.MaxMs
		p95Latency.Float64 = stats.P95Ms
	}

	// Calculate availability
	availabilityPct := float64(0)
	if totalChecks > 0 {
		availabilityPct = float64(successfulChecks) / float64(totalChecks) * percentageMultiplier
	}

	// Upsert the rollup
	_, err := r.db.Exec(ctx, `
		INSERT INTO health_check_rollups_daily
		(check_type, endpoint_name, day_bucket, total_checks, successful_checks,
		 avg_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms, availability_percent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(check_type, endpoint_name, day_bucket) DO UPDATE SET
			total_checks = excluded.total_checks,
			successful_checks = excluded.successful_checks,
			avg_latency_ms = excluded.avg_latency_ms,
			min_latency_ms = excluded.min_latency_ms,
			max_latency_ms = excluded.max_latency_ms,
			p95_latency_ms = excluded.p95_latency_ms,
			availability_percent = excluded.availability_percent
	`, checkType, endpointName, dayStart.Format("2006-01-02"),
		totalChecks, successfulChecks,
		avgLatency.Float64, minLatency.Float64, maxLatency.Float64, p95Latency.Float64,
		availabilityPct)
	if err != nil {
		return fmt.Errorf("failed to upsert daily rollup: %w", err)
	}

	return nil
}

// GetHourlyRollups retrieves hourly rollups for an endpoint.
func (r *HealthCheckRepository) GetHourlyRollups(
	ctx context.Context,
	checkType, endpointName string,
	timeRange TimeRange,
) ([]*HealthCheckHourlyRollup, error) {
	query := `
		SELECT id, check_type, endpoint_name, hour_bucket, total_checks, successful_checks,
		       avg_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms
		FROM health_check_rollups_hourly
		WHERE check_type = ? AND endpoint_name = ?
	`
	args := []any{checkType, endpointName}

	if !timeRange.Start.IsZero() {
		query += " AND hour_bucket >= ?"
		args = append(args, timeRange.Start.UTC().Format(time.RFC3339))
	}

	if !timeRange.End.IsZero() {
		query += " AND hour_bucket <= ?"
		args = append(args, timeRange.End.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY hour_bucket ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly rollups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rollups []*HealthCheckHourlyRollup
	for rows.Next() {
		var rollup HealthCheckHourlyRollup
		var hourBucket string
		var avgLat, minLat, maxLat, p95Lat sql.NullFloat64

		scanErr := rows.Scan(&rollup.ID, &rollup.CheckType, &rollup.EndpointName, &hourBucket,
			&rollup.TotalChecks, &rollup.SuccessfulChecks,
			&avgLat, &minLat, &maxLat, &p95Lat)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan hourly rollup: %w", scanErr)
		}

		if t, parseErr := time.Parse(time.RFC3339, hourBucket); parseErr == nil {
			rollup.HourBucket = t
		}
		rollup.AvgLatencyMs = avgLat.Float64
		rollup.MinLatencyMs = minLat.Float64
		rollup.MaxLatencyMs = maxLat.Float64
		rollup.P95LatencyMs = p95Lat.Float64

		rollups = append(rollups, &rollup)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return rollups, nil
}

// GetDailyRollups retrieves daily rollups for an endpoint.
func (r *HealthCheckRepository) GetDailyRollups(
	ctx context.Context,
	checkType, endpointName string,
	timeRange TimeRange,
) ([]*HealthCheckDailyRollup, error) {
	query := `
		SELECT id, check_type, endpoint_name, day_bucket, total_checks, successful_checks,
		       avg_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms, availability_percent
		FROM health_check_rollups_daily
		WHERE check_type = ? AND endpoint_name = ?
	`
	args := []any{checkType, endpointName}

	if !timeRange.Start.IsZero() {
		query += " AND day_bucket >= ?"
		args = append(args, timeRange.Start.UTC().Format("2006-01-02"))
	}

	if !timeRange.End.IsZero() {
		query += " AND day_bucket <= ?"
		args = append(args, timeRange.End.UTC().Format("2006-01-02"))
	}

	query += " ORDER BY day_bucket ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily rollups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rollups []*HealthCheckDailyRollup
	for rows.Next() {
		var rollup HealthCheckDailyRollup
		var dayBucket string
		var avgLat, minLat, maxLat, p95Lat, availPct sql.NullFloat64

		scanErr := rows.Scan(&rollup.ID, &rollup.CheckType, &rollup.EndpointName, &dayBucket,
			&rollup.TotalChecks, &rollup.SuccessfulChecks,
			&avgLat, &minLat, &maxLat, &p95Lat, &availPct)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan daily rollup: %w", scanErr)
		}

		if t, parseErr := time.Parse("2006-01-02", dayBucket); parseErr == nil {
			rollup.DayBucket = t
		}
		rollup.AvgLatencyMs = avgLat.Float64
		rollup.MinLatencyMs = minLat.Float64
		rollup.MaxLatencyMs = maxLat.Float64
		rollup.P95LatencyMs = p95Lat.Float64
		rollup.AvailabilityPercent = availPct.Float64

		rollups = append(rollups, &rollup)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return rollups, nil
}

// DeleteHourlyRollupsOlderThan removes hourly rollups older than the given time.
func (r *HealthCheckRepository) DeleteHourlyRollupsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM health_check_rollups_hourly WHERE hour_bucket < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old hourly rollups: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}
	return affected, nil
}

// DeleteDailyRollupsOlderThan removes daily rollups older than the given time.
func (r *HealthCheckRepository) DeleteDailyRollupsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM health_check_rollups_daily WHERE day_bucket < ?
	`, cutoff.UTC().Format("2006-01-02"))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old daily rollups: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}
	return affected, nil
}

// GetDistinctEndpoints returns all unique endpoint names for a check type.
func (r *HealthCheckRepository) GetDistinctEndpoints(ctx context.Context, checkType string) ([]string, error) {
	query := `SELECT DISTINCT endpoint_name FROM health_check_results`
	var args []any

	if checkType != "" {
		query += " WHERE check_type = ?"
		args = append(args, checkType)
	}

	query += " ORDER BY endpoint_name"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct endpoints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var endpoints []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scan endpoint name: %w", scanErr)
		}
		endpoints = append(endpoints, name)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return endpoints, nil
}

// GetDistinctCheckTypes returns all unique check types.
func (r *HealthCheckRepository) GetDistinctCheckTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT check_type FROM health_check_results ORDER BY check_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct check types: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var types []string
	for rows.Next() {
		var t string
		if scanErr := rows.Scan(&t); scanErr != nil {
			return nil, fmt.Errorf("scan check type: %w", scanErr)
		}
		types = append(types, t)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("rows iteration: %w", rowsErr)
	}
	return types, nil
}

// Count returns the total number of health check results.
func (r *HealthCheckRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM health_check_results`)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count health check results: %w", err)
	}
	return count, nil
}

// scanResult scans a health check result from rows.
func (r *HealthCheckRepository) scanResult(rows *sql.Rows) (*HealthCheckResult, error) {
	var result HealthCheckResult
	var timestamp string
	var statusCode sql.NullInt64
	var errorMsg, metadata sql.NullString
	var success int

	err := rows.Scan(&result.ID, &result.CheckType, &result.EndpointName,
		&result.EndpointTarget, &success, &result.LatencyMs,
		&statusCode, &errorMsg, &metadata, &timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to scan health check result: %w", err)
	}

	result.Success = success != 0
	if statusCode.Valid {
		sc := int(statusCode.Int64)
		result.StatusCode = &sc
	}
	result.ErrorMessage = errorMsg.String
	result.Metadata = metadata.String
	if t, parseErr := time.Parse(time.RFC3339, timestamp); parseErr == nil {
		result.RecordedAt = t
	}

	return &result, nil
}
