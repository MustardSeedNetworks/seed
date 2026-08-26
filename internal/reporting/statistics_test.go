package reporting_test

// Tests and benchmarks for the statistical primitives, carried over with them
// from internal/reporting/aggregator. Deleting a package is no reason to delete
// the coverage of the code kept out of it — and the two benchmarks are in the
// suite the allocation gate watches.

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

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
			got, err := reporting.Percentile(tt.values, tt.percentile)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

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
			got, err := reporting.MovingAverage(tt.values, tt.window)
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
			got, err := reporting.ExponentialMovingAverage(tt.values, tt.alpha)
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
			got, err := reporting.RateOfChange(tt.values)
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

func BenchmarkPercentile(b *testing.B) {
	values := make([]float64, 10000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = reporting.Percentile(values, 95)
	}
}

func BenchmarkMovingAverage(b *testing.B) {
	values := make([]float64, 10000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = reporting.MovingAverage(values, 100)
	}
}
