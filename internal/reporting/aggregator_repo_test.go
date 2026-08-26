package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// fakeMetricsRepo is an in-memory reporting.MetricsRepo. It lets the aggregator's
// severity-bucket and category mapping be tested with no database — the payoff
// of the MetricsRepo port.
type fakeMetricsRepo struct {
	devices   int
	vulns     map[string]int
	perf      reporting.PerformanceMetrics
	topIssues []reporting.IssueSummary
	trends    []reporting.DataPoint
}

func (f *fakeMetricsRepo) CountDevices(context.Context) (int, error) { return f.devices, nil }

func (f *fakeMetricsRepo) VulnerabilitySeverityCounts(
	context.Context, time.Time,
) (map[string]int, error) {
	return f.vulns, nil
}

func (f *fakeMetricsRepo) PerformanceMetrics(
	context.Context, time.Time,
) (reporting.PerformanceMetrics, error) {
	return f.perf, nil
}

func (f *fakeMetricsRepo) TopIssues(context.Context) ([]reporting.IssueSummary, error) {
	return f.topIssues, nil
}

func (f *fakeMetricsRepo) Trends(context.Context, string, string) ([]reporting.DataPoint, error) {
	return f.trends, nil
}

// TestAggregatorService_Aggregate_NoDB verifies the domain maps raw severity
// counts onto VulnCounts — known severities bucket; every severity (including
// unknown ones) still contributes to Total — without touching a database.
func TestAggregatorService_Aggregate_NoDB(t *testing.T) {
	metrics := &fakeMetricsRepo{
		devices: 12,
		vulns:   map[string]int{"critical": 2, "high": 3, "medium": 5, "low": 1, "unknown": 4},
		perf:    reporting.PerformanceMetrics{AvgLatencyMs: 10, UptimePercent: 99.5},
		topIssues: []reporting.IssueSummary{
			{Category: "vulnerability", Description: "CVE-X", Severity: "critical", Count: 2},
		},
	}
	as := reporting.NewAggregatorService(testConfig(), metrics)

	data, err := as.Aggregate(context.Background(), reporting.PeriodWeekly, "", "")
	require.NoError(t, err)

	assert.Equal(t, 12, data.DeviceCount)
	assert.Equal(t, 2, data.VulnCount.Critical)
	assert.Equal(t, 3, data.VulnCount.High)
	assert.Equal(t, 5, data.VulnCount.Medium)
	assert.Equal(t, 1, data.VulnCount.Low)
	assert.Equal(t, 15, data.VulnCount.Total) // 2+3+5+1+4 — unknown counts toward total
	assert.InEpsilon(t, 99.5, data.Performance.UptimePercent, 0.0001)
	assert.Len(t, data.TopIssues, 1)
}

// TestAggregatorService_GetTrends_NoDB confirms GetTrends passes the series
// through from the metrics port.
func TestAggregatorService_GetTrends_NoDB(t *testing.T) {
	metrics := &fakeMetricsRepo{trends: []reporting.DataPoint{{Value: 1.5}}}
	as := reporting.NewAggregatorService(testConfig(), metrics)

	summary, err := as.GetTrends(context.Background(), "latency", reporting.PeriodDaily)
	require.NoError(t, err)
	require.Len(t, summary.Points, 1)
	assert.InEpsilon(t, 1.5, summary.Points[0].Value, 0.0001)

	// One point has percentiles but nothing to smooth and no rate: a series
	// too short to analyse is not an error, it is a series you read directly.
	assert.InEpsilon(t, 1.5, summary.P50, 0.0001)
	assert.InEpsilon(t, 1.5, summary.P95, 0.0001)
	assert.Empty(t, summary.Smoothed)
	assert.Empty(t, summary.Change)
}

// TestAggregatorService_GetTrends_Summarises covers a series long enough for
// every statistic to apply.
func TestAggregatorService_GetTrends_Summarises(t *testing.T) {
	var points []reporting.DataPoint
	for _, v := range []float64{10, 12, 14, 16, 18, 20, 22} {
		points = append(points, reporting.DataPoint{Value: v})
	}
	as := reporting.NewAggregatorService(testConfig(), &fakeMetricsRepo{trends: points})

	summary, err := as.GetTrends(context.Background(), "latency", reporting.PeriodDaily)
	require.NoError(t, err)

	assert.InEpsilon(t, 16.0, summary.P50, 0.0001)

	// A five-point window over seven points leaves three averages.
	require.Len(t, summary.Smoothed, 3)
	assert.InEpsilon(t, 14.0, summary.Smoothed[0], 0.0001)

	// Six gaps between seven points, each a rise.
	require.Len(t, summary.Change, 6)
	for i, change := range summary.Change {
		assert.Positive(t, change, "gap %d should be a rise", i)
	}
}

// TestAggregatorService_GetTrends_Empty pins that no data is not an error. A
// metric with no samples yet is an ordinary state, not a failure.
func TestAggregatorService_GetTrends_Empty(t *testing.T) {
	as := reporting.NewAggregatorService(testConfig(), &fakeMetricsRepo{})

	summary, err := as.GetTrends(context.Background(), "latency", reporting.PeriodDaily)
	require.NoError(t, err)
	assert.Empty(t, summary.Points)
	assert.Zero(t, summary.P50)
	assert.Empty(t, summary.Smoothed)
}
