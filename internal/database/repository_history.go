package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// repository_history.go is the bounded read side of the tiered time-series
// store (#175). The write side and the rollup cadence live in
// rollup_sources.go and internal/timeseries/retention; nothing here writes.
//
// Two shapes only — a per-probe trend and a per-day anomaly count — and both
// take a fixed window. Deliberately not a query surface: seed stays
// handheld-class and exposes a fixed window over what it recorded (#175).
//
// Each read has a raw form and a rollup form. The caller (internal/api's
// resolveHistoryWindow) picks between them from the licence tier's horizons:
// the raw form is identical on every tier and includes the bucket in progress,
// the rollup form reaches back past the raw horizon.

// HistoryRepository reads the time-series tables. Obtained via DB.History().
type HistoryRepository struct {
	db *DB
}

// ProbeTrendPoint is one bucket of a probe's history. Latencies are over the
// successful runs in the bucket; SampleCount counts every run, so
// SampleCount-SuccessCount is the failure count the UI draws as availability.
type ProbeTrendPoint struct {
	Bucket       time.Time `json:"bucket"`
	SampleCount  int       `json:"sampleCount"`
	SuccessCount int       `json:"successCount"`
	AvgLatencyMs float64   `json:"avgLatencyMs"`
	MinLatencyMs float64   `json:"minLatencyMs"`
	MaxLatencyMs float64   `json:"maxLatencyMs"`
}

// AnomalyDayCount is one day's anomaly census: how many distinct (def,
// subject) anomalies were open on that day, and the highest severity among
// them.
type AnomalyDayCount struct {
	Day         string `json:"day"`
	Count       int    `json:"count"`
	MaxSeverity string `json:"maxSeverity"`
}

// probeTrendRawSQL aggregates probe_results on read. The bucket expression is
// substituted, not parameterised: SQLite will not take a bind parameter inside
// strftime's format, and the two forms are constants below, never caller input.
const probeTrendRawSQL = `
	SELECT strftime(%s, timestamp) AS bucket,
	       COUNT(*),
	       SUM(success),
	       AVG(CASE WHEN success = 1 THEN latency_ms END),
	       MIN(CASE WHEN success = 1 THEN latency_ms END),
	       MAX(CASE WHEN success = 1 THEN latency_ms END)
	FROM probe_results
	WHERE probe_id = ? AND client_id = ? AND timestamp >= ? AND timestamp < ?
	GROUP BY bucket
	ORDER BY bucket`

// SQLite strftime formats matching the rollup tables' bucket strings, so a raw
// series and a rollup series are the same shape.
const (
	sqliteHourBucket = `'%Y-%m-%dT%H:00:00Z'`
	sqliteDayBucket  = `'%Y-%m-%d'`
)

// ProbeTrendRaw aggregates probe_results in [from, to) into hourly or daily
// buckets. Unlike the rollup tables it includes the bucket in progress, which
// is why it is preferred for any window the raw horizon covers.
func (r *HistoryRepository) ProbeTrendRaw(
	ctx context.Context, clientID, probeID string, from, to time.Time, daily bool,
) ([]ProbeTrendPoint, error) {
	bucketExpr, layout := sqliteHourBucket, hourFormat
	if daily {
		bucketExpr, layout = sqliteDayBucket, dayFormat
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(probeTrendRawSQL, bucketExpr),
		probeID, clientID,
		from.UTC().Format(time.RFC3339Nano),
		to.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("probe trend (raw): %w", err)
	}
	return scanProbeTrend(rows, layout, "probe trend (raw)")
}

// probeTrendRollupSQL reads one of the two probe rollup tables. The table and
// bucket-column names are substituted from the constants below, never from
// caller input.
const probeTrendRollupSQL = `
	SELECT %[2]s, sample_count, success_count,
	       avg_latency_ms, min_latency_ms, max_latency_ms
	FROM %[1]s
	WHERE probe_id = ? AND client_id = ? AND %[2]s >= ? AND %[2]s < ?
	ORDER BY %[2]s`

// ProbeTrendRollup reads the hourly or daily probe rollup over [from, to).
// The bucket in progress is absent by construction: a rollup row is written
// once its bucket closes.
func (r *HistoryRepository) ProbeTrendRollup(
	ctx context.Context, clientID, probeID string, from, to time.Time, daily bool,
) ([]ProbeTrendPoint, error) {
	table, column, layout := "probe_rollups_hourly", "hour_bucket", hourFormat
	if daily {
		table, column, layout = "probe_rollups_daily", "day_bucket", dayFormat
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(probeTrendRollupSQL, table, column),
		probeID, clientID,
		from.UTC().Format(layout),
		to.UTC().Format(layout),
	)
	if err != nil {
		return nil, fmt.Errorf("probe trend (rollup): %w", err)
	}
	return scanProbeTrend(rows, layout, "probe trend (rollup)")
}

// scanProbeTrend reads the six columns both trend queries select, in order.
// Every latency is NULL for a bucket in which no run succeeded; those surface
// as zero beside a SuccessCount of zero, which reads as "no successful run"
// rather than "zero milliseconds".
func scanProbeTrend(rows *sql.Rows, layout, op string) ([]ProbeTrendPoint, error) {
	defer func() { _ = rows.Close() }()

	var points []ProbeTrendPoint
	for rows.Next() {
		var (
			bucket              string
			avg, minV, maxV     sql.NullFloat64
			samples, successful int
		)
		if err := rows.Scan(&bucket, &samples, &successful, &avg, &minV, &maxV); err != nil {
			return nil, fmt.Errorf("%s scan: %w", op, err)
		}
		ts, err := time.Parse(layout, bucket)
		if err != nil {
			return nil, fmt.Errorf("%s bucket %q: %w", op, bucket, err)
		}
		points = append(points, ProbeTrendPoint{
			Bucket:       ts,
			SampleCount:  samples,
			SuccessCount: successful,
			AvgLatencyMs: avg.Float64,
			MinLatencyMs: minV.Float64,
			MaxLatencyMs: maxV.Float64,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s rows: %w", op, err)
	}
	return points, nil
}

// anomalyCountsRollupSQL counts the census rows for each day. One row per
// (def, subject) per day, so COUNT(*) is the number of distinct anomalies open
// that day; count_cumulative is a per-anomaly running total and summing it
// across days would double-count (ADR-0028 §4).
const anomalyCountsRollupSQL = `
	SELECT day_bucket, COUNT(*), MAX(max_severity)
	FROM anomaly_rollups_daily
	WHERE day_bucket >= ? AND day_bucket <= ?
	GROUP BY day_bucket
	ORDER BY day_bucket`

// AnomalyCountsByDayRollup reads the daily anomaly census over the inclusive
// day range. Present on Pro only: the census is written and retained under the
// DailyDays horizon, which is zero below Pro.
func (r *HistoryRepository) AnomalyCountsByDayRollup(
	ctx context.Context, from, to time.Time,
) ([]AnomalyDayCount, error) {
	rows, err := r.db.Query(ctx, anomalyCountsRollupSQL,
		from.UTC().Format(dayFormat), to.UTC().Format(dayFormat))
	if err != nil {
		return nil, fmt.Errorf("anomaly counts (rollup): %w", err)
	}
	return scanAnomalyCounts(rows, "anomaly counts (rollup)")
}

// anomalyCountsLiveSQL derives the same per-day counts from the live anomalies
// table by joining each day in the window against the rows open on it. The
// open predicate is the census's own (repository_anomalies.go censusDaySQL),
// so a tier without the census reads the same numbers for the days its raw
// horizon still covers.
const anomalyCountsLiveSQL = `
	WITH RECURSIVE days(day) AS (
		SELECT date(?)
		UNION ALL
		SELECT date(day, '+1 day') FROM days WHERE day < date(?)
	)
	SELECT days.day, COUNT(anomalies.id), COALESCE(MAX(anomalies.severity), '')
	FROM days
	LEFT JOIN anomalies
	  ON date(anomalies.first_seen) <= days.day
	 AND days.day <= date(COALESCE(anomalies.resolved_at, anomalies.last_seen))
	GROUP BY days.day
	ORDER BY days.day`

// AnomalyCountsByDayLive derives per-day counts from the live anomalies table
// for the inclusive day range. It is what a tier with no daily census answers
// with, over the days its raw horizon covers. Days with no anomaly are
// returned with a zero count so the series has no gaps.
func (r *HistoryRepository) AnomalyCountsByDayLive(
	ctx context.Context, from, to time.Time,
) ([]AnomalyDayCount, error) {
	rows, err := r.db.Query(ctx, anomalyCountsLiveSQL,
		from.UTC().Format(dayFormat), to.UTC().Format(dayFormat))
	if err != nil {
		return nil, fmt.Errorf("anomaly counts (live): %w", err)
	}
	return scanAnomalyCounts(rows, "anomaly counts (live)")
}

func scanAnomalyCounts(rows *sql.Rows, op string) ([]AnomalyDayCount, error) {
	defer func() { _ = rows.Close() }()

	var counts []AnomalyDayCount
	for rows.Next() {
		var c AnomalyDayCount
		if err := rows.Scan(&c.Day, &c.Count, &c.MaxSeverity); err != nil {
			return nil, fmt.Errorf("%s scan: %w", op, err)
		}
		counts = append(counts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s rows: %w", op, err)
	}
	return counts, nil
}
