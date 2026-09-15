package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
)

// historyNow is a fixed instant inside an hour and a day, so the tests assert
// that the window ends at the request instant rather than at the last closed
// bucket.
func historyNow() time.Time {
	return time.Date(2026, time.September, 15, 13, 42, 17, 0, time.UTC)
}

func TestResolveHistoryWindow(t *testing.T) {
	t.Parallel()

	free := retention.HorizonsFor(license.TierFree)
	starter := retention.HorizonsFor(license.TierStarter)
	pro := retention.HorizonsFor(license.TierPro)

	tests := []struct {
		name       string
		horizons   retention.TierHorizons
		requested  time.Duration
		resolution string
		source     string
		days       int
		clamped    bool
	}{
		{
			name:     "free 24h reads raw at hourly resolution",
			horizons: free, requested: 24 * time.Hour,
			resolution: "hourly", source: "raw", days: 1,
		},
		{
			name:     "free 30d clamps to its 7-day raw horizon rather than erroring",
			horizons: free, requested: 30 * 24 * time.Hour,
			resolution: "hourly", source: "raw", days: 7, clamped: true,
		},
		{
			name:     "starter 30d reads the hourly rollup",
			horizons: starter, requested: 30 * 24 * time.Hour,
			resolution: "hourly", source: "rollup", days: 30,
		},
		{
			name:     "starter 7d still reads raw, so the open bucket is included",
			horizons: starter, requested: 7 * 24 * time.Hour,
			resolution: "hourly", source: "raw", days: 7,
		},
		{
			name:     "starter 1y clamps to its 30-day hourly horizon",
			horizons: starter, requested: 365 * 24 * time.Hour,
			resolution: "hourly", source: "rollup", days: 30, clamped: true,
		},
		{
			name:     "pro 90d crosses into daily",
			horizons: pro, requested: 90 * 24 * time.Hour,
			resolution: "daily", source: "rollup", days: 90,
		},
		{
			name:     "pro 2y is served whole",
			horizons: pro, requested: 730 * 24 * time.Hour,
			resolution: "daily", source: "rollup", days: 730,
		},
		{
			name:     "pro beyond 2y clamps to the daily horizon",
			horizons: pro, requested: 1000 * 24 * time.Hour,
			resolution: "daily", source: "rollup", days: 730, clamped: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveHistoryWindow(historyNow(), tc.requested, tc.horizons)

			require.Equal(t, tc.resolution, got.Resolution)
			require.Equal(t, tc.source, got.Source)
			require.Equal(t, tc.days, got.Days)
			require.Equal(t, tc.clamped, got.Clamped)
			require.Equal(t, historyNow(), got.To)
			require.Equal(t, historyNow().AddDate(0, 0, -tc.days), got.From)
		})
	}
}

// A request shorter than a day still spans whole days of storage, because the
// horizons are expressed in days; the window must not round down to zero.
func TestResolveHistoryWindowSubDayRequestKeepsOneDay(t *testing.T) {
	t.Parallel()

	got := resolveHistoryWindow(historyNow(), 90*time.Minute, retention.HorizonsFor(license.TierFree))

	require.Equal(t, 1, got.Days)
	require.False(t, got.Clamped)
	require.Equal(t, "raw", got.Source)
}

// The anomaly series has no hourly rollup, so the hourly horizon must not
// count towards what it can serve. Before this was separated, a Starter
// deployment — hourly 30 days, daily zero — resolved a 14-day request to
// "rollup, not clamped" and read the census table that tier never writes,
// answering an empty series as though it were complete.
func TestResolveAnomalyWindowIgnoresTheHourlyHorizon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tier      license.Tier
		requested time.Duration
		days      int
		source    string
		clamped   bool
	}{
		{
			name: "starter falls back to its raw horizon and says it clamped",
			tier: license.TierStarter, requested: 14 * 24 * time.Hour,
			days: 7, source: historySourceRaw, clamped: true,
		},
		{
			name: "free likewise",
			tier: license.TierFree, requested: 30 * 24 * time.Hour,
			days: 7, source: historySourceRaw, clamped: true,
		},
		{
			name: "pro reaches the census",
			tier: license.TierPro, requested: 14 * 24 * time.Hour,
			days: 14, source: historySourceRollup,
		},
		{
			name: "pro inside the raw horizon still reads the live table",
			tier: license.TierPro, requested: 3 * 24 * time.Hour,
			days: 3, source: historySourceRaw,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveAnomalyWindow(historyNow(), tc.requested,
				retention.HorizonsFor(tc.tier))

			require.Equal(t, tc.days, got.Days)
			require.Equal(t, tc.source, got.Source)
			require.Equal(t, tc.clamped, got.Clamped)
			require.Equal(t, historyResolutionDaily, got.Resolution,
				"anomaly buckets are days whatever the span")
		})
	}
}
