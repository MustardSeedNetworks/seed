package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// rollup_sources.go holds the time-series retention SQL adapters. They
// implement the retention.RollupSource port (internal/timeseries/retention)
// — the engine there orchestrates cadence; these adapters own the table
// layout and SQL. Wired into the engine at the composition root
// (internal/api initRetentionEngine).

// hourFormat / dayFormat match the SQLite strftime forms used in the rollup
// SQL. Rollup buckets are stored as ISO-8601 strings truncated to the hour or
// day, in UTC.
const (
	hourFormat = "2006-01-02T15:00:00Z"
	dayFormat  = "2006-01-02"
)

// MetricsRollupSource rolls up the metrics table into metrics_hourly and
// metrics_daily.
//
// Aggregation key: (client_id, target_kind, target_id, metric_type, bucket).
// Stage A2.1 added target_kind + target_id columns; new writes set them,
// legacy rows default to ('interface', interface_name) which keeps the dual
// representation working during the transition.
type MetricsRollupSource struct {
	db *DB
}

// NewMetricsRollupSource returns a source bound to the given DB.
func NewMetricsRollupSource(db *DB) *MetricsRollupSource {
	return &MetricsRollupSource{db: db}
}

// Name implements retention.RollupSource.
func (*MetricsRollupSource) Name() string { return "metrics" }

// RollupHour aggregates metrics in [hourStart, hourStart+1h) into
// metrics_hourly. Keyed by (client_id, target_kind, target_id, metric_type,
// hour_bucket). The legacy interface_name column is dual-populated for callers
// that haven't migrated to target_id.
func (s *MetricsRollupSource) RollupHour(ctx context.Context, hourStart time.Time) (int, error) {
	hourEnd := hourStart.Add(time.Hour)
	hourBucket := hourStart.UTC().Format(hourFormat)

	res, err := s.db.Exec(ctx, `
		INSERT OR REPLACE INTO metrics_hourly
		  (metric_type, interface_name, hour_bucket, sample_count,
		   avg_value, min_value, max_value, p95_value,
		   client_id, target_kind, target_id)
		SELECT
		  metric_type,
		  -- Preserve interface_name for backwards compat; new writes
		  -- set target_kind=interface, target_id=interface_name.
		  COALESCE(NULLIF(MAX(interface_name), ''), MAX(target_id)),
		  ?, COUNT(*),
		  AVG(value), MIN(value), MAX(value), NULL,
		  client_id, target_kind, target_id
		FROM metrics
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY client_id, target_kind, target_id, metric_type
	`,
		hourBucket,
		hourStart.UTC().Format(time.RFC3339Nano),
		hourEnd.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("metrics hourly rollup: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// RollupDay aggregates metrics_hourly rows in [dayStart, dayStart+24h) into
// metrics_daily. AVG-of-AVG; see ProbeRollupSource for the same simplification
// note.
func (s *MetricsRollupSource) RollupDay(ctx context.Context, dayStart time.Time) (int, error) {
	dayEnd := dayStart.Add(hoursPerDay * time.Hour)
	dayBucket := dayStart.UTC().Format(dayFormat)

	res, err := s.db.Exec(ctx, `
		INSERT OR REPLACE INTO metrics_daily
		  (metric_type, interface_name, day_bucket, sample_count,
		   avg_value, min_value, max_value, p95_value,
		   client_id, target_kind, target_id)
		SELECT
		  metric_type,
		  COALESCE(NULLIF(MAX(interface_name), ''), MAX(target_id)),
		  ?, SUM(sample_count),
		  AVG(avg_value), MIN(min_value), MAX(max_value), NULL,
		  client_id, target_kind, target_id
		FROM metrics_hourly
		WHERE hour_bucket >= ? AND hour_bucket < ?
		GROUP BY client_id, target_kind, target_id, metric_type
	`,
		dayBucket,
		dayStart.UTC().Format(hourFormat),
		dayEnd.UTC().Format(hourFormat),
	)
	if err != nil {
		return 0, fmt.Errorf("metrics daily rollup: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// PurgeRaw deletes metrics older than cutoff.
func (s *MetricsRollupSource) PurgeRaw(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM metrics WHERE timestamp < ?`,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	return purgeRows(res, err, "metrics raw purge")
}

// PurgeHourly deletes metrics_hourly rows whose bucket < cutoff.
func (s *MetricsRollupSource) PurgeHourly(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM metrics_hourly WHERE hour_bucket < ?`,
		cutoff.UTC().Format(hourFormat),
	)
	return purgeRows(res, err, "metrics_hourly purge")
}

// PurgeDaily deletes metrics_daily rows whose bucket < cutoff.
func (s *MetricsRollupSource) PurgeDaily(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM metrics_daily WHERE day_bucket < ?`,
		cutoff.UTC().Format(dayFormat),
	)
	return purgeRows(res, err, "metrics_daily purge")
}

// ProbeRollupSource rolls up the probe_results table into
// probe_rollups_hourly and probe_rollups_daily.
type ProbeRollupSource struct {
	db *DB
}

// NewProbeRollupSource returns a source bound to the given DB.
func NewProbeRollupSource(db *DB) *ProbeRollupSource {
	return &ProbeRollupSource{db: db}
}

// Name implements retention.RollupSource.
func (*ProbeRollupSource) Name() string { return "probe_results" }

// RollupHour aggregates probe_results in [hourStart, hourStart+1h).
//
// Aggregation key: (client_id, kind, probe_id, hour_bucket).
// sample_count = COUNT(*), success_count = SUM(success).
// avg / min / max latency are computed in SQL; p95 is left NULL in
// V1.0 — operators querying p95 use the raw 7-day window.
func (s *ProbeRollupSource) RollupHour(ctx context.Context, hourStart time.Time) (int, error) {
	hourEnd := hourStart.Add(time.Hour)
	hourBucket := hourStart.UTC().Format(hourFormat)

	res, err := s.db.Exec(ctx, `
		INSERT OR REPLACE INTO probe_rollups_hourly
		  (client_id, kind, probe_id, hour_bucket,
		   sample_count, success_count,
		   avg_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms)
		SELECT
		  client_id, kind, probe_id, ?,
		  COUNT(*), SUM(success),
		  AVG(latency_ms), MIN(latency_ms), MAX(latency_ms), NULL
		FROM probe_results
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY client_id, kind, probe_id
	`,
		hourBucket,
		hourStart.UTC().Format(time.RFC3339Nano),
		hourEnd.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("probe_results hourly rollup: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// RollupDay aggregates probe_rollups_hourly rows whose hour_bucket
// falls in [dayStart, dayStart+24h) into probe_rollups_daily.
func (s *ProbeRollupSource) RollupDay(ctx context.Context, dayStart time.Time) (int, error) {
	dayEnd := dayStart.Add(hoursPerDay * time.Hour)
	dayBucket := dayStart.UTC().Format(dayFormat)

	res, err := s.db.Exec(ctx, `
		INSERT OR REPLACE INTO probe_rollups_daily
		  (client_id, kind, probe_id, day_bucket,
		   sample_count, success_count,
		   avg_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms)
		SELECT
		  client_id, kind, probe_id, ?,
		  SUM(sample_count), SUM(success_count),
		  -- Re-aggregate from the hourly rows. AVG-of-AVG is OK when
		  -- buckets have equal sample_count; for unequal samples a
		  -- weighted average would be slightly more accurate.
		  -- V1.0 accepts the simple form.
		  AVG(avg_latency_ms), MIN(min_latency_ms), MAX(max_latency_ms), NULL
		FROM probe_rollups_hourly
		WHERE hour_bucket >= ? AND hour_bucket < ?
		GROUP BY client_id, kind, probe_id
	`,
		dayBucket,
		dayStart.UTC().Format(hourFormat),
		dayEnd.UTC().Format(hourFormat),
	)
	if err != nil {
		return 0, fmt.Errorf("probe_results daily rollup: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// PurgeRaw deletes probe_results older than cutoff.
func (s *ProbeRollupSource) PurgeRaw(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM probe_results WHERE timestamp < ?`,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	return purgeRows(res, err, "probe_results raw purge")
}

// PurgeHourly deletes probe_rollups_hourly rows whose bucket < cutoff.
func (s *ProbeRollupSource) PurgeHourly(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM probe_rollups_hourly WHERE hour_bucket < ?`,
		cutoff.UTC().Format(hourFormat),
	)
	return purgeRows(res, err, "probe_rollups_hourly purge")
}

// PurgeDaily deletes probe_rollups_daily rows whose bucket < cutoff.
func (s *ProbeRollupSource) PurgeDaily(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM probe_rollups_daily WHERE day_bucket < ?`,
		cutoff.UTC().Format(dayFormat),
	)
	return purgeRows(res, err, "probe_rollups_daily purge")
}

// purgeRows is the shared error-wrapping pattern for the Purge* methods.
func purgeRows(res sql.Result, err error, op string) (int64, error) {
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	if res == nil {
		return 0, errors.New(op + ": nil sql.Result")
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// FlowRollupSource rolls flow_records up into two pairs of tables:
// flow_conversations_hourly/_daily, one row per (src, dst, protocol) per
// bucket, and flow_applications_hourly/_daily, one row per application per
// bucket. A flow lands in the bucket it ended in, the same instant PurgeRaw
// ages it out by. Both pairs are written and purged in one transaction, so
// they never cover different spans.
//
// Counters are summed with TOTAL and cast back, not SUM: SUM raises an
// integer overflow, and one corrupt record pinned at MaxInt64 by
// saturatingInt64 would then fail every pass for its hour. The cast
// saturates at MaxInt64 instead, and is exact below 2^53 bytes.
type FlowRollupSource struct {
	db *DB
}

// NewFlowRollupSource returns a source bound to the given DB.
func NewFlowRollupSource(db *DB) *FlowRollupSource {
	return &FlowRollupSource{db: db}
}

// Name implements retention.RollupSource.
func (*FlowRollupSource) Name() string { return "flow_records" }

// RollupHour aggregates flows that ended in [hourStart, hourStart+1h).
func (s *FlowRollupSource) RollupHour(ctx context.Context, hourStart time.Time) (int, error) {
	n, err := s.execAll(ctx, []any{
		hourStart.UTC().Format(hourFormat),
		hourStart.UTC().Format(flowTimeFormat),
		hourStart.Add(time.Hour).UTC().Format(flowTimeFormat),
	}, `
		INSERT OR REPLACE INTO flow_conversations_hourly
		  (client_id, src_addr, dst_addr, protocol, hour_bucket, bytes, packets)
		SELECT client_id, src_addr, dst_addr, protocol, ?,
		       CAST(TOTAL(bytes) AS INTEGER), CAST(TOTAL(packets) AS INTEGER)
		FROM flow_records
		WHERE flow_end >= ? AND flow_end < ?
		GROUP BY client_id, src_addr, dst_addr, protocol
	`, `
		INSERT OR REPLACE INTO flow_applications_hourly
		  (client_id, application, hour_bucket, bytes, packets)
		SELECT client_id, application, ?,
		       CAST(TOTAL(bytes) AS INTEGER), CAST(TOTAL(packets) AS INTEGER)
		FROM flow_records
		WHERE flow_end >= ? AND flow_end < ?
		GROUP BY client_id, application
	`)
	if err != nil {
		return 0, fmt.Errorf("flow hourly rollup: %w", err)
	}
	return int(n), nil
}

// RollupDay aggregates the day's hourly rows into the daily tables.
func (s *FlowRollupSource) RollupDay(ctx context.Context, dayStart time.Time) (int, error) {
	n, err := s.execAll(ctx, []any{
		dayStart.UTC().Format(dayFormat),
		dayStart.UTC().Format(hourFormat),
		dayStart.Add(hoursPerDay * time.Hour).UTC().Format(hourFormat),
	}, `
		INSERT OR REPLACE INTO flow_conversations_daily
		  (client_id, src_addr, dst_addr, protocol, day_bucket, bytes, packets)
		SELECT client_id, src_addr, dst_addr, protocol, ?,
		       CAST(TOTAL(bytes) AS INTEGER), CAST(TOTAL(packets) AS INTEGER)
		FROM flow_conversations_hourly
		WHERE hour_bucket >= ? AND hour_bucket < ?
		GROUP BY client_id, src_addr, dst_addr, protocol
	`, `
		INSERT OR REPLACE INTO flow_applications_daily
		  (client_id, application, day_bucket, bytes, packets)
		SELECT client_id, application, ?,
		       CAST(TOTAL(bytes) AS INTEGER), CAST(TOTAL(packets) AS INTEGER)
		FROM flow_applications_hourly
		WHERE hour_bucket >= ? AND hour_bucket < ?
		GROUP BY client_id, application
	`)
	if err != nil {
		return 0, fmt.Errorf("flow daily rollup: %w", err)
	}
	return int(n), nil
}

// PurgeRaw deletes flows that ended before cutoff.
func (s *FlowRollupSource) PurgeRaw(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.db.FlowRecords().DeleteOlderThan(ctx, cutoff)
}

// PurgeHourly deletes hourly flow rollup rows whose bucket < cutoff.
func (s *FlowRollupSource) PurgeHourly(ctx context.Context, cutoff time.Time) (int64, error) {
	n, err := s.execAll(ctx, []any{cutoff.UTC().Format(hourFormat)},
		`DELETE FROM flow_conversations_hourly WHERE hour_bucket < ?`,
		`DELETE FROM flow_applications_hourly WHERE hour_bucket < ?`)
	if err != nil {
		return 0, fmt.Errorf("flow hourly purge: %w", err)
	}
	return n, nil
}

// PurgeDaily deletes daily flow rollup rows whose bucket < cutoff.
func (s *FlowRollupSource) PurgeDaily(ctx context.Context, cutoff time.Time) (int64, error) {
	n, err := s.execAll(ctx, []any{cutoff.UTC().Format(dayFormat)},
		`DELETE FROM flow_conversations_daily WHERE day_bucket < ?`,
		`DELETE FROM flow_applications_daily WHERE day_bucket < ?`)
	if err != nil {
		return 0, fmt.Errorf("flow daily purge: %w", err)
	}
	return n, nil
}

// execAll runs each statement with the same arguments in one transaction
// and returns the rows they affected between them.
func (s *FlowRollupSource) execAll(ctx context.Context, args []any, stmts ...string) (int64, error) {
	var total int64
	err := s.db.WithTx(ctx, func(tx *sql.Tx) error {
		for _, stmt := range stmts {
			res, err := tx.ExecContext(ctx, stmt, args...)
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			total += n
		}
		return nil
	})
	return total, err
}
