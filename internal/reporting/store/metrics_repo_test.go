package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/reporting/store"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
)

// newPerformanceRepo returns a repo over five hourly gateway samples, written
// as the telemetry engine writes them, and three speed tests. The oldest
// gateway sample got no reply, so it has no latency.
func newPerformanceRepo(t *testing.T) *store.MetricsRepo {
	t.Helper()
	db, err := database.Open(dbtest.Path(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	for i := range 5 {
		points := []telemetry.Point{
			{Type: telemetry.GatewayReachable, Value: 1, Unit: telemetry.UnitBool},
			{Type: telemetry.GatewayLossPct, Value: 10 * float64(i), Unit: telemetry.UnitPercent},
			{Type: telemetry.GatewayLatencyMs, Value: 10 + float64(i), Unit: telemetry.UnitMs},
		}
		if i == 4 {
			points = []telemetry.Point{
				{Type: telemetry.GatewayReachable, Value: 0, Unit: telemetry.UnitBool},
				{Type: telemetry.GatewayLossPct, Value: 100, Unit: telemetry.UnitPercent},
			}
		}
		require.NoError(t, db.Metrics().RecordTelemetry(t.Context(), telemetry.Sample{
			Interface: "eth0",
			At:        now.Add(-time.Duration(i) * time.Hour),
			Points:    points,
		}))
	}
	for i := range 3 {
		require.NoError(t, db.Metrics().RecordSpeedTest(t.Context(), &database.SpeedTestResult{
			InterfaceName: "eth0",
			DownloadMbps:  100 + 10*float64(i),
			UploadMbps:    50 + 5*float64(i),
			Timestamp:     now.Add(-time.Duration(i) * time.Hour).UTC(),
		}))
	}
	return store.NewMetricsRepo(db)
}

// TestPerformanceMetricsReadsGatewayTelemetry is #2623's gateway half: the
// report's latency, loss and uptime come from the samples the daemon keeps,
// so a gateway that stopped answering is no longer reported as 100% up.
func TestPerformanceMetricsReadsGatewayTelemetry(t *testing.T) {
	t.Parallel()
	repo := newPerformanceRepo(t)

	perf, err := repo.PerformanceMetrics(t.Context(), time.Now().Add(-24*time.Hour))
	require.NoError(t, err)

	assert.InDelta(t, 11.5, perf.AvgLatencyMs, 0.001, "mean of the four replies, 10..13 ms")
	assert.InDelta(t, 32.0, perf.AvgPacketLoss, 0.001, "mean of 0, 10, 20, 30 and 100%")
	assert.InDelta(t, 80.0, perf.UptimePercent, 0.001, "four of five samples got a reply")
	assert.InDelta(t, 82.5, perf.AvgBandwidthMbps, 0.001, "mean of (down+up)/2 over three tests")
}

func TestPerformanceMetricsWithoutSamples(t *testing.T) {
	t.Parallel()
	repo := newPerformanceRepo(t)

	perf, err := repo.PerformanceMetrics(t.Context(), time.Now().Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, reporting.PerformanceMetrics{UptimePercent: 100}, perf)
}

// TestTrendsLatencyHourlyBuckets pins the daily trend: one point per hour of
// gateway latency, each stamped with its hour rather than the zero time.
func TestTrendsLatencyHourlyBuckets(t *testing.T) {
	t.Parallel()
	repo := newPerformanceRepo(t)

	points, err := repo.Trends(t.Context(), "latency", reporting.PeriodDaily)
	require.NoError(t, err)

	require.Len(t, points, 4, "four samples with a reply, an hour apart")
	for _, p := range points {
		assert.False(t, p.Timestamp.IsZero(), "bucket %+v has no time", p)
		assert.Zero(t, p.Timestamp.Minute())
	}
}
