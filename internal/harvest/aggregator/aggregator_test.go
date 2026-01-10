package aggregator_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/harvest/aggregator"
)

// ----------------------------------------------------------------------------
// TimeRange Tests
// ----------------------------------------------------------------------------

func TestTimeRange_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		tr      aggregator.TimeRange
		wantErr bool
	}{
		{
			name: "valid range",
			tr: aggregator.TimeRange{
				Start: now.Add(-24 * time.Hour),
				End:   now,
			},
			wantErr: false,
		},
		{
			name: "start after end",
			tr: aggregator.TimeRange{
				Start: now,
				End:   now.Add(-24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "start equals end",
			tr: aggregator.TimeRange{
				Start: now,
				End:   now,
			},
			wantErr: true,
		},
		{
			name: "one second difference",
			tr: aggregator.TimeRange{
				Start: now,
				End:   now.Add(time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tr.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, aggregator.ErrInvalidTimeRange)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTimeRange_Contains(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	tr := aggregator.TimeRange{
		Start: baseTime,
		End:   baseTime.Add(24 * time.Hour),
	}

	tests := []struct {
		name     string
		testTime time.Time
		want     bool
	}{
		{
			name:     "time at start (inclusive)",
			testTime: baseTime,
			want:     true,
		},
		{
			name:     "time at end (exclusive)",
			testTime: baseTime.Add(24 * time.Hour),
			want:     false,
		},
		{
			name:     "time in middle",
			testTime: baseTime.Add(12 * time.Hour),
			want:     true,
		},
		{
			name:     "time before range",
			testTime: baseTime.Add(-1 * time.Hour),
			want:     false,
		},
		{
			name:     "time after range",
			testTime: baseTime.Add(25 * time.Hour),
			want:     false,
		},
		{
			name:     "time one nanosecond before end",
			testTime: baseTime.Add(24*time.Hour - time.Nanosecond),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.Contains(tt.testTime)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimeRange_Duration(t *testing.T) {
	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantDur time.Duration
	}{
		{
			name:    "one hour",
			start:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:     time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
			wantDur: time.Hour,
		},
		{
			name:    "one day",
			start:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			wantDur: 24 * time.Hour,
		},
		{
			name:    "one week",
			start:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:     time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			wantDur: 7 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := aggregator.TimeRange{Start: tt.start, End: tt.end}
			assert.Equal(t, tt.wantDur, tr.Duration())
		})
	}
}

// ----------------------------------------------------------------------------
// Calculate (Statistics) Tests
// ----------------------------------------------------------------------------

func TestCalculate(t *testing.T) {
	tests := []struct {
		name      string
		values    []float64
		wantStats aggregator.Statistics
		wantErr   bool
	}{
		{
			name:   "single value",
			values: []float64{5.0},
			wantStats: aggregator.Statistics{
				Count:  1,
				Sum:    5.0,
				Min:    5.0,
				Max:    5.0,
				Avg:    5.0,
				Median: 5.0,
				StdDev: 0,
			},
			wantErr: false,
		},
		{
			name:   "two values",
			values: []float64{4.0, 6.0},
			wantStats: aggregator.Statistics{
				Count:  2,
				Sum:    10.0,
				Min:    4.0,
				Max:    6.0,
				Avg:    5.0,
				Median: 5.0,
				StdDev: 1.0,
			},
			wantErr: false,
		},
		{
			name:   "multiple values odd count",
			values: []float64{1, 2, 3, 4, 5},
			wantStats: aggregator.Statistics{
				Count:  5,
				Sum:    15.0,
				Min:    1.0,
				Max:    5.0,
				Avg:    3.0,
				Median: 3.0,
				StdDev: 1.414, // sqrt(2)
			},
			wantErr: false,
		},
		{
			name:   "multiple values even count",
			values: []float64{1, 2, 3, 4},
			wantStats: aggregator.Statistics{
				Count:  4,
				Sum:    10.0,
				Min:    1.0,
				Max:    4.0,
				Avg:    2.5,
				Median: 2.5,
				StdDev: 1.118, // sqrt(1.25)
			},
			wantErr: false,
		},
		{
			name:   "negative values",
			values: []float64{-5, -3, -1, 0, 1, 3, 5},
			wantStats: aggregator.Statistics{
				Count:  7,
				Sum:    0.0,
				Min:    -5.0,
				Max:    5.0,
				Avg:    0.0,
				Median: 0.0,
				StdDev: 3.162, // approx
			},
			wantErr: false,
		},
		{
			name:   "all same values",
			values: []float64{7, 7, 7, 7},
			wantStats: aggregator.Statistics{
				Count:  4,
				Sum:    28.0,
				Min:    7.0,
				Max:    7.0,
				Avg:    7.0,
				Median: 7.0,
				StdDev: 0,
			},
			wantErr: false,
		},
		{
			name:    "empty slice",
			values:  []float64{},
			wantErr: true,
		},
		{
			name:    "nil slice",
			values:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := aggregator.Calculate(tt.values)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, aggregator.ErrEmptyDataset)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantStats.Count, stats.Count)
			assert.InDelta(t, tt.wantStats.Sum, stats.Sum, 0.001)
			assert.InDelta(t, tt.wantStats.Min, stats.Min, 0.001)
			assert.InDelta(t, tt.wantStats.Max, stats.Max, 0.001)
			assert.InDelta(t, tt.wantStats.Avg, stats.Avg, 0.001)
			assert.InDelta(t, tt.wantStats.Median, stats.Median, 0.001)
			// StdDev only checked when explicitly set in expected
			if tt.wantStats.StdDev >= 0 {
				assert.InDelta(t, tt.wantStats.StdDev, stats.StdDev, 0.001)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Percentile Tests
// ----------------------------------------------------------------------------

func TestPercentile(t *testing.T) {
	tests := []struct {
		name       string
		values     []float64
		percentile int
		want       float64
		wantErr    bool
	}{
		{
			name:       "0th percentile (min)",
			values:     []float64{1, 2, 3, 4, 5},
			percentile: 0,
			want:       1.0,
			wantErr:    false,
		},
		{
			name:       "100th percentile (max)",
			values:     []float64{1, 2, 3, 4, 5},
			percentile: 100,
			want:       5.0,
			wantErr:    false,
		},
		{
			name:       "50th percentile (median)",
			values:     []float64{1, 2, 3, 4, 5},
			percentile: 50,
			want:       3.0,
			wantErr:    false,
		},
		{
			name:       "25th percentile",
			values:     []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 25,
			want:       3.25,
			wantErr:    false,
		},
		{
			name:       "75th percentile",
			values:     []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 75,
			want:       7.75,
			wantErr:    false,
		},
		{
			name:       "90th percentile",
			values:     []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 90,
			want:       9.1,
			wantErr:    false,
		},
		{
			name:       "empty values",
			values:     []float64{},
			percentile: 50,
			wantErr:    true,
		},
		{
			name:       "negative percentile",
			values:     []float64{1, 2, 3},
			percentile: -1,
			wantErr:    true,
		},
		{
			name:       "percentile over 100",
			values:     []float64{1, 2, 3},
			percentile: 101,
			wantErr:    true,
		},
		{
			name:       "single value any percentile",
			values:     []float64{42},
			percentile: 50,
			want:       42,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.Percentile(tt.values, tt.percentile)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// ----------------------------------------------------------------------------
// GroupByPeriod Tests
// ----------------------------------------------------------------------------

func TestGroupByPeriod(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		points    []aggregator.DataPoint
		period    aggregator.Period
		wantKeys  []string
		wantErr   bool
		errTarget error
	}{
		{
			name: "hourly grouping",
			points: []aggregator.DataPoint{
				{Timestamp: baseTime, Value: 1},                       // 10:30 -> hour 10
				{Timestamp: baseTime.Add(30 * time.Minute), Value: 2}, // 11:00 -> hour 11
				{Timestamp: baseTime.Add(90 * time.Minute), Value: 3}, // 12:00 -> hour 12
			},
			period:   aggregator.PeriodHourly,
			wantKeys: []string{"2024-01-15-10", "2024-01-15-11", "2024-01-15-12"},
			wantErr:  false,
		},
		{
			name: "daily grouping",
			points: []aggregator.DataPoint{
				{Timestamp: baseTime, Value: 1},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3},
			},
			period:   aggregator.PeriodDaily,
			wantKeys: []string{"2024-01-15", "2024-01-16", "2024-01-17"},
			wantErr:  false,
		},
		{
			name: "monthly grouping",
			points: []aggregator.DataPoint{
				{Timestamp: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Value: 1},
				{Timestamp: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), Value: 2},
				{Timestamp: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), Value: 3},
			},
			period:   aggregator.PeriodMonthly,
			wantKeys: []string{"2024-01", "2024-02", "2024-03"},
			wantErr:  false,
		},
		{
			name:      "empty points",
			points:    []aggregator.DataPoint{},
			period:    aggregator.PeriodDaily,
			wantErr:   true,
			errTarget: aggregator.ErrEmptyDataset,
		},
		{
			name: "invalid period",
			points: []aggregator.DataPoint{
				{Timestamp: baseTime, Value: 1},
			},
			period:    aggregator.Period("invalid"),
			wantErr:   true,
			errTarget: aggregator.ErrInvalidPeriod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := aggregator.GroupByPeriod(tt.points, tt.period)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					require.ErrorIs(t, err, tt.errTarget)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, len(tt.wantKeys))
			for _, key := range tt.wantKeys {
				assert.Contains(t, result, key)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// FilterByRange Tests
// ----------------------------------------------------------------------------

func TestFilterByRange(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	points := []aggregator.DataPoint{
		{Timestamp: baseTime.Add(-48 * time.Hour), Value: 1, Label: "before"},
		{Timestamp: baseTime.Add(-24 * time.Hour), Value: 2, Label: "start"},
		{Timestamp: baseTime, Value: 3, Label: "middle"},
		{Timestamp: baseTime.Add(12 * time.Hour), Value: 4, Label: "middle2"},
		{Timestamp: baseTime.Add(24 * time.Hour), Value: 5, Label: "end"},
		{Timestamp: baseTime.Add(48 * time.Hour), Value: 6, Label: "after"},
	}

	tests := []struct {
		name      string
		points    []aggregator.DataPoint
		tr        aggregator.TimeRange
		wantCount int
		wantErr   bool
		errTarget error
	}{
		{
			name:   "filter middle range",
			points: points,
			tr: aggregator.TimeRange{
				Start: baseTime.Add(-24 * time.Hour),
				End:   baseTime.Add(24 * time.Hour),
			},
			wantCount: 3, // start, middle, middle2
			wantErr:   false,
		},
		{
			name:   "filter all out of range",
			points: points,
			tr: aggregator.TimeRange{
				Start: baseTime.Add(100 * time.Hour),
				End:   baseTime.Add(200 * time.Hour),
			},
			wantErr:   true,
			errTarget: aggregator.ErrNoDataInRange,
		},
		{
			name:   "invalid time range",
			points: points,
			tr: aggregator.TimeRange{
				Start: baseTime.Add(24 * time.Hour),
				End:   baseTime,
			},
			wantErr:   true,
			errTarget: aggregator.ErrInvalidTimeRange,
		},
		{
			name:   "single point in range",
			points: points,
			tr: aggregator.TimeRange{
				Start: baseTime.Add(-1 * time.Hour),
				End:   baseTime.Add(1 * time.Hour),
			},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := aggregator.FilterByRange(tt.points, tt.tr)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					require.ErrorIs(t, err, tt.errTarget)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

// ----------------------------------------------------------------------------
// AggregatePoints Tests
// ----------------------------------------------------------------------------

func TestAggregatePoints(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		points    []aggregator.DataPoint
		period    aggregator.Period
		wantCount int
		wantSum   float64
		wantMin   float64
		wantMax   float64
		wantErr   bool
	}{
		{
			name: "basic aggregation",
			points: []aggregator.DataPoint{
				{Timestamp: baseTime, Value: 10},
				{Timestamp: baseTime.Add(time.Hour), Value: 20},
				{Timestamp: baseTime.Add(2 * time.Hour), Value: 30},
			},
			period:    aggregator.PeriodDaily,
			wantCount: 3,
			wantSum:   60,
			wantMin:   10,
			wantMax:   30,
			wantErr:   false,
		},
		{
			name: "single point",
			points: []aggregator.DataPoint{
				{Timestamp: baseTime, Value: 42},
			},
			period:    aggregator.PeriodDaily,
			wantCount: 1,
			wantSum:   42,
			wantMin:   42,
			wantMax:   42,
			wantErr:   false,
		},
		{
			name:    "empty points",
			points:  []aggregator.DataPoint{},
			period:  aggregator.PeriodDaily,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := aggregator.AggregatePoints(tt.points, tt.period)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantCount, result.Count)
			assert.InDelta(t, tt.wantSum, result.Sum, 0.001)
			assert.InDelta(t, tt.wantMin, result.Min, 0.001)
			assert.InDelta(t, tt.wantMax, result.Max, 0.001)
			assert.Equal(t, tt.period, result.Period)
			assert.NotEmpty(t, result.Percentile)
		})
	}
}

// ----------------------------------------------------------------------------
// MovingAverage Tests
// ----------------------------------------------------------------------------

func TestMovingAverage(t *testing.T) {
	tests := []struct {
		name    string
		values  []float64
		window  int
		want    []float64
		wantErr bool
	}{
		{
			name:   "window of 3",
			values: []float64{1, 2, 3, 4, 5},
			window: 3,
			want:   []float64{2, 3, 4},
		},
		{
			name:   "window of 2",
			values: []float64{1, 2, 3, 4},
			window: 2,
			want:   []float64{1.5, 2.5, 3.5},
		},
		{
			name:   "window equals length",
			values: []float64{1, 2, 3},
			window: 3,
			want:   []float64{2},
		},
		{
			name:    "window larger than data",
			values:  []float64{1, 2},
			window:  5,
			wantErr: true,
		},
		{
			name:    "zero window",
			values:  []float64{1, 2, 3},
			window:  0,
			wantErr: true,
		},
		{
			name:    "negative window",
			values:  []float64{1, 2, 3},
			window:  -1,
			wantErr: true,
		},
		{
			name:    "empty values",
			values:  []float64{},
			window:  3,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.MovingAverage(tt.values, tt.window)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.InDelta(t, tt.want[i], got[i], 0.001)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// ExponentialMovingAverage Tests
// ----------------------------------------------------------------------------

func TestExponentialMovingAverage(t *testing.T) {
	tests := []struct {
		name    string
		values  []float64
		alpha   float64
		wantErr bool
	}{
		{
			name:    "valid alpha 0.5",
			values:  []float64{1, 2, 3, 4, 5},
			alpha:   0.5,
			wantErr: false,
		},
		{
			name:    "alpha 1.0 (all weight to current)",
			values:  []float64{1, 2, 3},
			alpha:   1.0,
			wantErr: false,
		},
		{
			name:    "alpha close to 0",
			values:  []float64{1, 2, 3},
			alpha:   0.01,
			wantErr: false,
		},
		{
			name:    "alpha 0 (invalid)",
			values:  []float64{1, 2, 3},
			alpha:   0,
			wantErr: true,
		},
		{
			name:    "alpha greater than 1 (invalid)",
			values:  []float64{1, 2, 3},
			alpha:   1.5,
			wantErr: true,
		},
		{
			name:    "negative alpha",
			values:  []float64{1, 2, 3},
			alpha:   -0.5,
			wantErr: true,
		},
		{
			name:    "empty values",
			values:  []float64{},
			alpha:   0.5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.ExponentialMovingAverage(tt.values, tt.alpha)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, len(tt.values))

			// First value should equal input
			assert.InDelta(t, tt.values[0], got[0], 0.000001)

			// Alpha 1.0 should return original values
			if tt.alpha == 1.0 {
				for i := range tt.values {
					assert.InDelta(t, tt.values[i], got[i], 0.000001)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// RateOfChange Tests
// ----------------------------------------------------------------------------

func TestRateOfChange(t *testing.T) {
	tests := []struct {
		name    string
		values  []float64
		want    []float64
		wantErr bool
	}{
		{
			name:   "simple increase",
			values: []float64{100, 110, 121},
			want:   []float64{0.1, 0.1},
		},
		{
			name:   "decrease",
			values: []float64{100, 90, 81},
			want:   []float64{-0.1, -0.1},
		},
		{
			name:   "mixed",
			values: []float64{100, 150, 100},
			want:   []float64{0.5, -1.0 / 3.0},
		},
		{
			name:   "division by zero previous",
			values: []float64{0, 10},
			want:   []float64{math.Inf(1)},
		},
		{
			name:   "division by zero both",
			values: []float64{0, 0},
			want:   []float64{0},
		},
		{
			name:    "single value",
			values:  []float64{100},
			wantErr: true,
		},
		{
			name:    "empty values",
			values:  []float64{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.RateOfChange(tt.values)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				if math.IsInf(tt.want[i], 0) {
					assert.True(t, math.IsInf(got[i], 0))
				} else {
					assert.InDelta(t, tt.want[i], got[i], 0.001)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// SumByLabel Tests
// ----------------------------------------------------------------------------

func TestSumByLabel(t *testing.T) {
	tests := []struct {
		name   string
		points []aggregator.DataPoint
		want   map[string]float64
	}{
		{
			name: "multiple labels",
			points: []aggregator.DataPoint{
				{Label: "A", Value: 10},
				{Label: "B", Value: 20},
				{Label: "A", Value: 5},
				{Label: "B", Value: 15},
			},
			want: map[string]float64{"A": 15, "B": 35},
		},
		{
			name: "single label",
			points: []aggregator.DataPoint{
				{Label: "X", Value: 1},
				{Label: "X", Value: 2},
				{Label: "X", Value: 3},
			},
			want: map[string]float64{"X": 6},
		},
		{
			name:   "empty points",
			points: []aggregator.DataPoint{},
			want:   map[string]float64{},
		},
		{
			name: "empty label",
			points: []aggregator.DataPoint{
				{Label: "", Value: 10},
				{Label: "", Value: 5},
			},
			want: map[string]float64{"": 15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.SumByLabel(tt.points)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ----------------------------------------------------------------------------
// CountByLabel Tests
// ----------------------------------------------------------------------------

func TestCountByLabel(t *testing.T) {
	tests := []struct {
		name   string
		points []aggregator.DataPoint
		want   map[string]int
	}{
		{
			name: "multiple labels",
			points: []aggregator.DataPoint{
				{Label: "A", Value: 10},
				{Label: "B", Value: 20},
				{Label: "A", Value: 5},
				{Label: "A", Value: 1},
			},
			want: map[string]int{"A": 3, "B": 1},
		},
		{
			name:   "empty points",
			points: []aggregator.DataPoint{},
			want:   map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.CountByLabel(tt.points)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ----------------------------------------------------------------------------
// TopN and BottomN Tests
// ----------------------------------------------------------------------------

func TestTopN(t *testing.T) {
	points := []aggregator.DataPoint{
		{Label: "A", Value: 30},
		{Label: "B", Value: 10},
		{Label: "C", Value: 50},
		{Label: "D", Value: 20},
		{Label: "E", Value: 40},
	}

	tests := []struct {
		name       string
		points     []aggregator.DataPoint
		n          int
		wantLabels []string
	}{
		{
			name:       "top 3",
			points:     points,
			n:          3,
			wantLabels: []string{"C", "E", "A"},
		},
		{
			name:       "top 1",
			points:     points,
			n:          1,
			wantLabels: []string{"C"},
		},
		{
			name:       "n equals count",
			points:     points,
			n:          5,
			wantLabels: []string{"C", "E", "A", "D", "B"},
		},
		{
			name:       "n greater than count",
			points:     points,
			n:          10,
			wantLabels: []string{"C", "E", "A", "D", "B"},
		},
		{
			name:       "n is zero",
			points:     points,
			n:          0,
			wantLabels: nil,
		},
		{
			name:       "n is negative",
			points:     points,
			n:          -1,
			wantLabels: nil,
		},
		{
			name:       "empty points",
			points:     []aggregator.DataPoint{},
			n:          3,
			wantLabels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.TopN(tt.points, tt.n)
			if tt.wantLabels == nil {
				assert.Nil(t, got)
				return
			}

			require.Len(t, got, len(tt.wantLabels))
			for i, label := range tt.wantLabels {
				assert.Equal(t, label, got[i].Label)
			}
		})
	}
}

func TestBottomN(t *testing.T) {
	points := []aggregator.DataPoint{
		{Label: "A", Value: 30},
		{Label: "B", Value: 10},
		{Label: "C", Value: 50},
		{Label: "D", Value: 20},
		{Label: "E", Value: 40},
	}

	tests := []struct {
		name       string
		points     []aggregator.DataPoint
		n          int
		wantLabels []string
	}{
		{
			name:       "bottom 3",
			points:     points,
			n:          3,
			wantLabels: []string{"B", "D", "A"},
		},
		{
			name:       "bottom 1",
			points:     points,
			n:          1,
			wantLabels: []string{"B"},
		},
		{
			name:       "n is zero",
			points:     points,
			n:          0,
			wantLabels: nil,
		},
		{
			name:       "empty points",
			points:     []aggregator.DataPoint{},
			n:          3,
			wantLabels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregator.BottomN(tt.points, tt.n)
			if tt.wantLabels == nil {
				assert.Nil(t, got)
				return
			}

			require.Len(t, got, len(tt.wantLabels))
			for i, label := range tt.wantLabels {
				assert.Equal(t, label, got[i].Label)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Normalize Tests
// ----------------------------------------------------------------------------

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		values  []float64
		want    []float64
		wantErr bool
	}{
		{
			name:   "normal range",
			values: []float64{0, 25, 50, 75, 100},
			want:   []float64{0, 0.25, 0.5, 0.75, 1},
		},
		{
			name:   "negative values",
			values: []float64{-50, 0, 50},
			want:   []float64{0, 0.5, 1},
		},
		{
			name:   "all same values",
			values: []float64{5, 5, 5, 5},
			want:   []float64{0.5, 0.5, 0.5, 0.5},
		},
		{
			name:   "two values",
			values: []float64{10, 20},
			want:   []float64{0, 1},
		},
		{
			name:    "empty values",
			values:  []float64{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.Normalize(tt.values)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.InDelta(t, tt.want[i], got[i], 0.001)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Outliers Tests
// ----------------------------------------------------------------------------

func TestOutliers(t *testing.T) {
	tests := []struct {
		name         string
		values       []float64
		wantOutliers []int
		wantErr      bool
	}{
		{
			name:         "no outliers",
			values:       []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			wantOutliers: nil,
		},
		{
			name:         "one high outlier",
			values:       []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100},
			wantOutliers: []int{9},
		},
		{
			name:         "one low outlier",
			values:       []float64{-100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			wantOutliers: []int{0},
		},
		{
			name:         "multiple outliers",
			values:       []float64{-100, 1, 2, 3, 4, 5, 6, 7, 8, 100},
			wantOutliers: []int{0, 9},
		},
		{
			name:    "too few values",
			values:  []float64{1, 2, 3},
			wantErr: true,
		},
		{
			name:    "empty values",
			values:  []float64{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aggregator.Outliers(tt.values)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantOutliers == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.wantOutliers, got)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Period Constant Tests
// ----------------------------------------------------------------------------

func TestPeriodConstants(t *testing.T) {
	// Verify period constants are properly defined
	assert.Equal(t, aggregator.PeriodHourly, aggregator.Period("hourly"))
	assert.Equal(t, aggregator.PeriodDaily, aggregator.Period("daily"))
	assert.Equal(t, aggregator.PeriodWeekly, aggregator.Period("weekly"))
	assert.Equal(t, aggregator.PeriodMonthly, aggregator.Period("monthly"))
}

// ----------------------------------------------------------------------------
// Error Constant Tests
// ----------------------------------------------------------------------------

func TestErrorConstants(t *testing.T) {
	// Verify error messages are meaningful
	assert.Contains(t, aggregator.ErrEmptyDataset.Error(), "empty")
	assert.Contains(t, aggregator.ErrInvalidPeriod.Error(), "period")
	assert.Contains(t, aggregator.ErrInvalidTimeRange.Error(), "time range")
	assert.Contains(t, aggregator.ErrNoDataInRange.Error(), "no data")
}

// ----------------------------------------------------------------------------
// Edge Case Tests
// ----------------------------------------------------------------------------

func TestEdgeCases(t *testing.T) {
	t.Run("very large values", func(t *testing.T) {
		values := []float64{1e15, 2e15, 3e15}
		stats, err := aggregator.Calculate(values)
		require.NoError(t, err)
		assert.InDelta(t, 2e15, stats.Avg, 1e10)
	})

	t.Run("very small values", func(t *testing.T) {
		values := []float64{1e-15, 2e-15, 3e-15}
		stats, err := aggregator.Calculate(values)
		require.NoError(t, err)
		assert.InDelta(t, 2e-15, stats.Avg, 1e-20)
	})

	t.Run("mixed positive and negative", func(t *testing.T) {
		values := []float64{-100, -50, 0, 50, 100}
		stats, err := aggregator.Calculate(values)
		require.NoError(t, err)
		assert.InDelta(t, 0.0, stats.Avg, 0.000001)
		assert.InDelta(t, -100.0, stats.Min, 0.000001)
		assert.InDelta(t, 100.0, stats.Max, 0.000001)
	})

	t.Run("single element percentiles", func(t *testing.T) {
		for p := 0; p <= 100; p += 25 {
			val, err := aggregator.Percentile([]float64{42}, p)
			require.NoError(t, err)
			assert.InDelta(t, 42.0, val, 0.000001)
		}
	})
}

// ----------------------------------------------------------------------------
// Concurrency Safety Tests
// ----------------------------------------------------------------------------

func TestConcurrencySafety(t *testing.T) {
	t.Parallel()

	values := make([]float64, 1000)
	for i := range values {
		values[i] = float64(i)
	}

	done := make(chan bool, 10)
	for range 10 {
		go func() {
			defer func() { done <- true }()
			// These should all be safe for concurrent reads
			_, _ = aggregator.Calculate(values)
			_, _ = aggregator.Percentile(values, 50)
			_, _ = aggregator.MovingAverage(values, 10)
			_, _ = aggregator.Normalize(values)
		}()
	}

	for range 10 {
		<-done
	}
}

// ----------------------------------------------------------------------------
// Benchmark Tests
// ----------------------------------------------------------------------------

func BenchmarkCalculate(b *testing.B) {
	values := make([]float64, 10000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = aggregator.Calculate(values)
	}
}

func BenchmarkPercentile(b *testing.B) {
	values := make([]float64, 10000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = aggregator.Percentile(values, 95)
	}
}

func BenchmarkMovingAverage(b *testing.B) {
	values := make([]float64, 10000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = aggregator.MovingAverage(values, 100)
	}
}
