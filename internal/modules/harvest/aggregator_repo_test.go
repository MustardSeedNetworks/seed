package harvest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/modules/harvest"
)

// fakeMetricsRepo is an in-memory harvest.MetricsRepo. It lets the aggregator's
// severity-bucket and category mapping be tested with no database — the payoff
// of the MetricsRepo port (PHASE3_EXTRACTION_PLAN.md §4.5).
type fakeMetricsRepo struct {
	devices   int
	vulns     map[string]int
	perf      harvest.PerformanceMetrics
	topIssues []harvest.IssueSummary
	trends    []harvest.DataPoint
}

func (f *fakeMetricsRepo) CountDevices(context.Context) (int, error) { return f.devices, nil }

func (f *fakeMetricsRepo) VulnerabilitySeverityCounts(
	context.Context, time.Time,
) (map[string]int, error) {
	return f.vulns, nil
}

func (f *fakeMetricsRepo) PerformanceMetrics(
	context.Context, time.Time,
) (harvest.PerformanceMetrics, error) {
	return f.perf, nil
}

func (f *fakeMetricsRepo) TopIssues(context.Context) ([]harvest.IssueSummary, error) {
	return f.topIssues, nil
}

func (f *fakeMetricsRepo) Trends(context.Context, string, string) ([]harvest.DataPoint, error) {
	return f.trends, nil
}

// TestAggregatorService_Aggregate_NoDB verifies the domain maps raw severity
// counts onto VulnCounts — known severities bucket; every severity (including
// unknown ones) still contributes to Total — without touching a database.
func TestAggregatorService_Aggregate_NoDB(t *testing.T) {
	metrics := &fakeMetricsRepo{
		devices: 12,
		vulns:   map[string]int{"critical": 2, "high": 3, "medium": 5, "low": 1, "unknown": 4},
		perf:    harvest.PerformanceMetrics{AvgLatencyMs: 10, UptimePercent: 99.5},
		topIssues: []harvest.IssueSummary{
			{Category: "vulnerability", Description: "CVE-X", Severity: "critical", Count: 2},
		},
	}
	as := harvest.NewAggregatorService(testConfig(), metrics)

	data, err := as.Aggregate(context.Background(), harvest.PeriodWeekly, "", "")
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

// TestAggregatorService_GetTrends_NoDB confirms GetTrends delegates straight to
// the metrics port.
func TestAggregatorService_GetTrends_NoDB(t *testing.T) {
	metrics := &fakeMetricsRepo{trends: []harvest.DataPoint{{Value: 1.5}}}
	as := harvest.NewAggregatorService(testConfig(), metrics)

	points, err := as.GetTrends(context.Background(), "latency", harvest.PeriodDaily)
	require.NoError(t, err)
	require.Len(t, points, 1)
	assert.InEpsilon(t, 1.5, points[0].Value, 0.0001)
}
