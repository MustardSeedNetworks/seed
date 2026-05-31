package retention

import (
	"context"
	"time"
)

// RollupSource is one time-series surface the retention engine
// rolls up and purges. Each implementation encapsulates its own
// SQL — the engine orchestrates cadence, the source owns its
// table layout.
//
// Stage A2 V1.0 sources: probe_results (probe.Engine writes) and
// metrics (servermon / microburst writes). V1.1+ may add flow
// aggregates from NetFlow listeners.
type RollupSource interface {
	// Name is a stable identifier used in logs (e.g. "probe_results",
	// "metrics").
	Name() string

	// RollupHour aggregates raw rows whose timestamp falls in
	// [hourStart, hourStart+1h) into the source's hourly-aggregate
	// table. Returns the number of distinct buckets written.
	// Idempotent — re-running for the same hour replaces the row.
	RollupHour(ctx context.Context, hourStart time.Time) (int, error)

	// RollupDay aggregates the prior hourly rows into the daily
	// table for the day-bucket starting at dayStart. Returns the
	// number of distinct buckets written. Idempotent.
	RollupDay(ctx context.Context, dayStart time.Time) (int, error)

	// PurgeRaw deletes raw rows with timestamp < cutoff. Returns
	// the affected row count.
	PurgeRaw(ctx context.Context, cutoff time.Time) (int64, error)

	// PurgeHourly deletes hourly rollups whose bucket < cutoff.
	// Returns the affected row count.
	PurgeHourly(ctx context.Context, cutoff time.Time) (int64, error)

	// PurgeDaily deletes daily rollups whose bucket < cutoff.
	// Returns the affected row count.
	PurgeDaily(ctx context.Context, cutoff time.Time) (int64, error)
}

// TierHorizons captures per-tier retention policy. Stored / read
// dynamically each pass so an in-place license upgrade takes effect
// on the next tick.
type TierHorizons struct {
	// RawDays is how many days of raw rows to keep. All tiers
	// retain raw for 7 days as a baseline observability window.
	RawDays int

	// HourlyDays is how many days of hourly rollups to keep. Free
	// keeps zero (immediate purge); Starter 30; Pro 90.
	HourlyDays int

	// DailyDays is how many days of daily rollups to keep. Free
	// and Starter keep zero (immediate purge); Pro 730 (2 years).
	DailyDays int
}
