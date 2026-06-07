package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// MetricsSource rolls up internal/database.metrics into
// metrics_hourly and metrics_daily.
//
// Aggregation key: (client_id, target_kind, target_id, metric_type,
// bucket). Stage A2.1 added target_kind + target_id columns; new
// writes set them, legacy rows default to ('interface',
// interface_name) which keeps the dual representation working
// during the transition.
type MetricsSource struct {
	db *database.DB
}

// NewMetricsSource returns a source bound to the given DB.
func NewMetricsSource(db *database.DB) *MetricsSource {
	return &MetricsSource{db: db}
}

// Name implements RollupSource.
func (*MetricsSource) Name() string { return "metrics" }

// RollupHour aggregates metrics in [hourStart, hourStart+1h) into
// metrics_hourly. Keyed by (client_id, target_kind, target_id,
// metric_type, hour_bucket). The legacy interface_name column is
// dual-populated for callers that haven't migrated to target_id.
func (s *MetricsSource) RollupHour(ctx context.Context, hourStart time.Time) (int, error) {
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

// RollupDay aggregates metrics_hourly rows in [dayStart,
// dayStart+24h) into metrics_daily. AVG-of-AVG; see ProbeResultsSource
// for the same simplification note.
func (s *MetricsSource) RollupDay(ctx context.Context, dayStart time.Time) (int, error) {
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
func (s *MetricsSource) PurgeRaw(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM metrics WHERE timestamp < ?`,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	return purgeRows(res, err, "metrics raw purge")
}

// PurgeHourly deletes metrics_hourly rows whose bucket < cutoff.
func (s *MetricsSource) PurgeHourly(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM metrics_hourly WHERE hour_bucket < ?`,
		cutoff.UTC().Format(hourFormat),
	)
	return purgeRows(res, err, "metrics_hourly purge")
}

// PurgeDaily deletes metrics_daily rows whose bucket < cutoff.
func (s *MetricsSource) PurgeDaily(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM metrics_daily WHERE day_bucket < ?`,
		cutoff.UTC().Format(dayFormat),
	)
	return purgeRows(res, err, "metrics_daily purge")
}
