package reporting

import (
	"errors"
	"math"
	"sort"
)

// Statistical primitives for trend analysis.
//
// Lifted from internal/reporting/aggregator, a package nothing imported. The
// service in this package aggregates domains — vulnerabilities, performance,
// top issues — and had none of these, so GetTrends returned a raw series and
// left every caller to work out for itself whether the series was going
// anywhere.
//
// Only the four that operate on a plain []float64 came across. The rest of that
// package duplicated types this one already has (its own DataPoint carried a
// Label this one has no concept of), which is how it drifted out of use in the
// first place.

// percentileMin and percentileMax bound the percentile argument.
const (
	percentileMin = 0
	percentileMax = 100
	// percentileScale converts a whole-number percentile to a fraction.
	percentileScale = 100.0
	// minRateOfChangeSamples is the fewest points a rate of change needs: a
	// rate is defined between two values, so one value has no rate.
	minRateOfChangeSamples = 2
)

// ErrEmptyDataset is returned when a statistic is asked of no data.
var ErrEmptyDataset = errors.New("dataset is empty")

// Percentile calculates the p-th percentile of the values.
// p should be between 0 and 100.
func Percentile(values []float64, p int) (float64, error) {
	if len(values) == 0 {
		return 0, ErrEmptyDataset
	}
	if p < percentileMin || p > percentileMax {
		return 0, errors.New("percentile must be between 0 and 100")
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	if p == 0 {
		return sorted[0], nil
	}
	if p == percentileMax {
		return sorted[len(sorted)-1], nil
	}

	// Calculate the index
	index := float64(p) / percentileScale * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower], nil
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight, nil
}

// MovingAverage calculates a simple moving average over n periods.
func MovingAverage(values []float64, n int) ([]float64, error) {
	if len(values) == 0 {
		return nil, ErrEmptyDataset
	}
	if n <= 0 {
		return nil, errors.New("window size must be positive")
	}
	if n > len(values) {
		return nil, errors.New("window size larger than dataset")
	}

	result := make([]float64, len(values)-n+1)
	var windowSum float64

	// Initialize first window
	for i := range n {
		windowSum += values[i]
	}
	result[0] = windowSum / float64(n)

	// Slide the window
	for i := n; i < len(values); i++ {
		windowSum = windowSum - values[i-n] + values[i]
		result[i-n+1] = windowSum / float64(n)
	}

	return result, nil
}

// ExponentialMovingAverage calculates an EMA with the given alpha (smoothing factor).
// Alpha should be between 0 and 1, where higher values give more weight to recent data.
func ExponentialMovingAverage(values []float64, alpha float64) ([]float64, error) {
	if len(values) == 0 {
		return nil, ErrEmptyDataset
	}
	if alpha <= 0 || alpha > 1 {
		return nil, errors.New("alpha must be between 0 and 1 (exclusive of 0)")
	}

	result := make([]float64, len(values))
	result[0] = values[0]

	for i := 1; i < len(values); i++ {
		result[i] = alpha*values[i] + (1-alpha)*result[i-1]
	}

	return result, nil
}

// RateOfChange calculates the rate of change between consecutive values.
// Returns a slice of length n-1 where each value represents (current - previous) / previous.
func RateOfChange(values []float64) ([]float64, error) {
	if len(values) < minRateOfChangeSamples {
		return nil, errors.New("at least 2 values required for rate of change")
	}

	result := make([]float64, len(values)-1)
	for i := 1; i < len(values); i++ {
		prev := values[i-1]
		curr := values[i]
		switch {
		case prev == 0 && curr == 0:
			result[i-1] = 0
		case prev == 0 && curr < 0:
			result[i-1] = math.Inf(-1)
		case prev == 0:
			result[i-1] = math.Inf(1)
		default:
			result[i-1] = (curr - prev) / prev
		}
	}

	return result, nil
}
