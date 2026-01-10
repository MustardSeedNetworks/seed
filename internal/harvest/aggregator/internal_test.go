package aggregator_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/krisarmstrong/seed/internal/harvest/aggregator"
)

// ----------------------------------------------------------------------------
// Internal Function Tests (via export_test.go)
// ----------------------------------------------------------------------------

func TestCalculateMedian_Internal(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{
			name:   "empty slice",
			values: []float64{},
			want:   0,
		},
		{
			name:   "single element",
			values: []float64{5},
			want:   5,
		},
		{
			name:   "two elements",
			values: []float64{3, 7},
			want:   5,
		},
		{
			name:   "odd count",
			values: []float64{1, 3, 5},
			want:   3,
		},
		{
			name:   "even count",
			values: []float64{1, 2, 3, 4},
			want:   2.5,
		},
		{
			name:   "unsorted input",
			values: []float64{5, 1, 3, 2, 4},
			want:   3,
		},
		{
			name:   "negative values",
			values: []float64{-5, -3, -1, 0, 2},
			want:   -1,
		},
		{
			name:   "duplicate values",
			values: []float64{1, 1, 1, 1},
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.ExportCalculateMedian(tt.values)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestCalculateStdDev_Internal(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		mean   float64
		want   float64
	}{
		{
			name:   "single value",
			values: []float64{5},
			mean:   5,
			want:   0,
		},
		{
			name:   "uniform values",
			values: []float64{5, 5, 5, 5},
			mean:   5,
			want:   0,
		},
		{
			name:   "simple distribution",
			values: []float64{2, 4, 4, 4, 5, 5, 7, 9},
			mean:   5,
			want:   2,
		},
		{
			name:   "two values symmetric around mean",
			values: []float64{0, 10},
			mean:   5,
			want:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.ExportCalculateStdDev(tt.values, tt.mean)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestPeriodKey_Internal(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name    string
		time    time.Time
		period  aggregator.Period
		want    string
		wantErr bool
	}{
		{
			name:   "hourly",
			time:   testTime,
			period: aggregator.PeriodHourly,
			want:   "2024-03-15-14",
		},
		{
			name:   "daily",
			time:   testTime,
			period: aggregator.PeriodDaily,
			want:   "2024-03-15",
		},
		{
			name:   "monthly",
			time:   testTime,
			period: aggregator.PeriodMonthly,
			want:   "2024-03",
		},
		{
			name:   "weekly",
			time:   testTime,
			period: aggregator.PeriodWeekly,
			want:   "2024-W11-2024", // Week 11 of 2024
		},
		{
			name:    "invalid period",
			time:    testTime,
			period:  aggregator.Period("invalid"),
			wantErr: true,
		},
		{
			name:   "hourly at midnight",
			time:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			period: aggregator.PeriodHourly,
			want:   "2024-01-01-00",
		},
		{
			name:   "daily at year boundary",
			time:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			period: aggregator.PeriodDaily,
			want:   "2024-12-31",
		},
		{
			name:   "monthly January",
			time:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			period: aggregator.PeriodMonthly,
			want:   "2024-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.ExportPeriodKey(tt.time, tt.period)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPadInt_Internal(t *testing.T) {
	tests := []struct {
		name  string
		n     int
		width int
		want  string
	}{
		{
			name:  "single digit width 2",
			n:     5,
			width: 2,
			want:  "05",
		},
		{
			name:  "two digit width 2",
			n:     42,
			width: 2,
			want:  "42",
		},
		{
			name:  "single digit width 4",
			n:     7,
			width: 4,
			want:  "0007",
		},
		{
			name:  "zero width 2",
			n:     0,
			width: 2,
			want:  "00",
		},
		{
			name:  "year format",
			n:     2024,
			width: 4,
			want:  "2024",
		},
		{
			name:  "week number",
			n:     11,
			width: 2,
			want:  "11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.ExportPadInt(tt.n, tt.width)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ----------------------------------------------------------------------------
// Integration Tests
// ----------------------------------------------------------------------------

func TestAggregationWorkflow(t *testing.T) {
	// Test a typical aggregation workflow
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create sample data points
	points := make([]aggregator.DataPoint, 100)
	for i := range points {
		points[i] = aggregator.DataPoint{
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Value:     float64(i + 1),
			Label:     "metric",
		}
	}

	// Filter by time range
	tr := aggregator.TimeRange{
		Start: baseTime,
		End:   baseTime.Add(48 * time.Hour),
	}
	filtered, err := aggregator.FilterByRange(points, tr)
	assert.NoError(t, err)
	assert.Len(t, filtered, 48)

	// Group by period
	grouped, err := aggregator.GroupByPeriod(filtered, aggregator.PeriodDaily)
	assert.NoError(t, err)
	assert.Len(t, grouped, 2) // 2 days

	// Aggregate the entire filtered set
	result, err := aggregator.AggregatePoints(filtered, aggregator.PeriodDaily)
	assert.NoError(t, err)
	assert.Equal(t, 48, result.Count)
	assert.Equal(t, float64(48*(48+1)/2), result.Sum) // Sum of 1 to 48

	// Get top values
	top5 := aggregator.TopN(filtered, 5)
	assert.Len(t, top5, 5)
	assert.Equal(t, float64(48), top5[0].Value)
}

func TestStatisticalAnalysisWorkflow(t *testing.T) {
	// Test a statistical analysis workflow
	values := []float64{12, 15, 18, 22, 25, 28, 32, 35, 38, 100} // 100 is an outlier

	// Calculate basic statistics
	stats, err := aggregator.Calculate(values)
	assert.NoError(t, err)
	assert.Equal(t, 10, stats.Count)

	// Find outliers
	outlierIndices, err := aggregator.Outliers(values)
	assert.NoError(t, err)
	assert.Contains(t, outlierIndices, 9) // 100 should be identified as outlier

	// Calculate percentiles
	p50, err := aggregator.Percentile(values, 50)
	assert.NoError(t, err)
	assert.InDelta(t, 26.5, p50, 0.5)

	p90, err := aggregator.Percentile(values, 90)
	assert.NoError(t, err)
	assert.InDelta(t, 44.2, p90, 1) // 38 + 0.1*(100-38) = 44.2

	// Normalize values
	normalized, err := aggregator.Normalize(values)
	assert.NoError(t, err)
	assert.Len(t, normalized, len(values))
	assert.InDelta(t, 0, normalized[0], 0.01) // Min should normalize to 0
	assert.InDelta(t, 1, normalized[9], 0.01) // Max should normalize to 1
}

func TestTimeSeriesAnalysisWorkflow(t *testing.T) {
	// Simulate time series data with trend
	values := make([]float64, 50)
	for i := range values {
		values[i] = float64(i*2 + 10) // Linear trend
	}

	// Calculate moving average
	ma, err := aggregator.MovingAverage(values, 5)
	assert.NoError(t, err)
	assert.Len(t, ma, 46)

	// Calculate EMA
	ema, err := aggregator.ExponentialMovingAverage(values, 0.3)
	assert.NoError(t, err)
	assert.Len(t, ema, 50)

	// Calculate rate of change
	roc, err := aggregator.RateOfChange(values)
	assert.NoError(t, err)
	assert.Len(t, roc, 49)

	// Rate of change should be positive for upward trend
	for _, r := range roc {
		assert.Positive(t, r)
	}
}

// ----------------------------------------------------------------------------
// DataPoint Struct Tests
// ----------------------------------------------------------------------------

func TestDataPointStruct(t *testing.T) {
	dp := aggregator.DataPoint{
		Timestamp: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Value:     42.5,
		Label:     "test",
	}

	assert.Equal(t, 2024, dp.Timestamp.Year())
	assert.Equal(t, 42.5, dp.Value)
	assert.Equal(t, "test", dp.Label)
}

func TestAggregatedResultStruct(t *testing.T) {
	result := &aggregator.AggregatedResult{
		Period:    aggregator.PeriodDaily,
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Count:     100,
		Sum:       5050,
		Min:       1,
		Max:       100,
		Avg:       50.5,
		Median:    50.5,
		StdDev:    28.87,
		Percentile: map[int]float64{
			25: 25,
			50: 50,
			75: 75,
			90: 90,
			95: 95,
			99: 99,
		},
	}

	assert.Equal(t, aggregator.PeriodDaily, result.Period)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), result.StartTime)
	assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), result.EndTime)
	assert.Equal(t, 100, result.Count)
	assert.Equal(t, 5050.0, result.Sum)
	assert.Equal(t, 1.0, result.Min)
	assert.Equal(t, 100.0, result.Max)
	assert.Equal(t, 50.5, result.Avg)
	assert.Equal(t, 50.5, result.Median)
	assert.Equal(t, 28.87, result.StdDev)
	assert.Equal(t, 6, len(result.Percentile))
}

func TestStatisticsStruct(t *testing.T) {
	stats := aggregator.Statistics{
		Count:  10,
		Sum:    55,
		Min:    1,
		Max:    10,
		Avg:    5.5,
		Median: 5.5,
		StdDev: 2.87,
	}

	assert.Equal(t, 10, stats.Count)
	assert.Equal(t, 55.0, stats.Sum)
	assert.Equal(t, 1.0, stats.Min)
	assert.Equal(t, 10.0, stats.Max)
	assert.Equal(t, 5.5, stats.Avg)
	assert.Equal(t, 5.5, stats.Median)
	assert.Equal(t, 2.87, stats.StdDev)
}
