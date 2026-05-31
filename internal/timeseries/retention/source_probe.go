package retention

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

// hourFormat / dayFormat match the SQLite strftime forms used in
// the rollup SQL. Probe rollup buckets are stored as ISO-8601
// strings truncated to the hour or day, in UTC.
const (
	hourFormat = "2006-01-02T15:00:00Z"
	dayFormat  = "2006-01-02"
)

// ProbeResultsSource rolls up internal/database.probe_results into
// probe_rollups_hourly and probe_rollups_daily.
type ProbeResultsSource struct {
	db *database.DB
}

// NewProbeResultsSource returns a source bound to the given DB.
func NewProbeResultsSource(db *database.DB) *ProbeResultsSource {
	return &ProbeResultsSource{db: db}
}

// Name implements RollupSource.
func (*ProbeResultsSource) Name() string { return "probe_results" }

// RollupHour aggregates probe_results in [hourStart, hourStart+1h).
//
// Aggregation key: (client_id, kind, probe_id, hour_bucket).
// sample_count = COUNT(*), success_count = SUM(success).
// avg / min / max latency are computed in SQL; p95 is left NULL in
// V1.0 — operators querying p95 use the raw 7-day window.
func (s *ProbeResultsSource) RollupHour(ctx context.Context, hourStart time.Time) (int, error) {
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
func (s *ProbeResultsSource) RollupDay(ctx context.Context, dayStart time.Time) (int, error) {
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
func (s *ProbeResultsSource) PurgeRaw(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM probe_results WHERE timestamp < ?`,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	return purgeRows(res, err, "probe_results raw purge")
}

// PurgeHourly deletes probe_rollups_hourly rows whose bucket <
// cutoff.
func (s *ProbeResultsSource) PurgeHourly(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM probe_rollups_hourly WHERE hour_bucket < ?`,
		cutoff.UTC().Format(hourFormat),
	)
	return purgeRows(res, err, "probe_rollups_hourly purge")
}

// PurgeDaily deletes probe_rollups_daily rows whose bucket < cutoff.
func (s *ProbeResultsSource) PurgeDaily(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM probe_rollups_daily WHERE day_bucket < ?`,
		cutoff.UTC().Format(dayFormat),
	)
	return purgeRows(res, err, "probe_rollups_daily purge")
}

// purgeRows is the shared error-wrapping pattern for the three
// Purge* methods.
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
