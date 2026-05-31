package retention

// RollupSource describes a time-series surface the retention engine
// rolls up and purges. Implementations declare their raw table,
// hourly/daily aggregate tables, key columns (for GROUP BY), and the
// value column for AVG/MIN/MAX/P95 computation.
//
// The retention engine treats every registered source uniformly: one
// hourly rollup pass per hour, one daily rollup pass per day, one
// tier-aware purge per tick across all sources.
type RollupSource interface {
	// Name is a stable identifier (e.g. "probe_results", "metrics").
	// Used in logs and metrics.
	Name() string

	// RawTable returns the raw time-series table name.
	RawTable() string

	// HourlyTable returns the hourly-aggregate table name.
	HourlyTable() string

	// DailyTable returns the daily-aggregate table name.
	DailyTable() string

	// KeyColumns lists the columns that uniquely identify a series
	// within the table (e.g. ["client_id", "kind", "probe_id"] for
	// probe_results). The retention engine GROUPs BY these columns
	// when computing rollups.
	KeyColumns() []string

	// ValueColumn names the numeric column to aggregate for
	// avg/min/max/p95. For success/failure surfaces, also surface a
	// success-rate via SuccessColumn (returning "" disables it).
	ValueColumn() string

	// SuccessColumn names the boolean (integer 0/1) column to count
	// successes from. Empty string disables success-rate computation.
	SuccessColumn() string
}
