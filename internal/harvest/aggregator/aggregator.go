package aggregator

import (
	"errors"
	"math"
	"sort"
	"time"
)

// Period represents a time-based aggregation period.
type Period string

// Predefined aggregation periods.
const (
	PeriodHourly  Period = "hourly"
	PeriodDaily   Period = "daily"
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
)

// Common errors returned by aggregator functions.
var (
	ErrEmptyDataset     = errors.New("dataset is empty")
	ErrInvalidPeriod    = errors.New("invalid aggregation period")
	ErrInvalidTimeRange = errors.New("invalid time range: start must be before end")
	ErrNoDataInRange    = errors.New("no data points in specified time range")
)

// DataPoint represents a single time-series data point.
type DataPoint struct {
	Timestamp time.Time
	Value     float64
	Label     string
}

// AggregatedResult contains the result of an aggregation operation.
type AggregatedResult struct {
	Period     Period
	StartTime  time.Time
	EndTime    time.Time
	Count      int
	Sum        float64
	Min        float64
	Max        float64
	Avg        float64
	Median     float64
	StdDev     float64
	Percentile map[int]float64
}

// TimeRange defines a time window for filtering data.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Validate checks if the time range is valid.
func (tr TimeRange) Validate() error {
	if tr.Start.After(tr.End) || tr.Start.Equal(tr.End) {
		return ErrInvalidTimeRange
	}
	return nil
}

// Contains checks if a timestamp falls within the time range.
func (tr TimeRange) Contains(t time.Time) bool {
	return (t.Equal(tr.Start) || t.After(tr.Start)) && t.Before(tr.End)
}

// Duration returns the duration of the time range.
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Statistics calculates basic statistics for a slice of float64 values.
type Statistics struct {
	Count  int
	Sum    float64
	Min    float64
	Max    float64
	Avg    float64
	Median float64
	StdDev float64
}

// Calculate computes statistics for the given values.
func Calculate(values []float64) (Statistics, error) {
	if len(values) == 0 {
		return Statistics{}, ErrEmptyDataset
	}

	stats := Statistics{
		Count: len(values),
		Min:   values[0],
		Max:   values[0],
	}

	// Calculate sum, min, max
	for _, v := range values {
		stats.Sum += v
		if v < stats.Min {
			stats.Min = v
		}
		if v > stats.Max {
			stats.Max = v
		}
	}

	// Calculate average
	stats.Avg = stats.Sum / float64(stats.Count)

	// Calculate median
	stats.Median = calculateMedian(values)

	// Calculate standard deviation
	stats.StdDev = calculateStdDev(values, stats.Avg)

	return stats, nil
}

// calculateMedian returns the median value of a slice.
func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Create a copy to avoid modifying the original
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// calculateStdDev calculates the standard deviation given values and their mean.
func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}

	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values))
	return math.Sqrt(variance)
}

// Percentile calculates the p-th percentile of the values.
// p should be between 0 and 100.
func Percentile(values []float64, p int) (float64, error) {
	if len(values) == 0 {
		return 0, ErrEmptyDataset
	}
	if p < 0 || p > 100 {
		return 0, errors.New("percentile must be between 0 and 100")
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	if p == 0 {
		return sorted[0], nil
	}
	if p == 100 {
		return sorted[len(sorted)-1], nil
	}

	// Calculate the index
	index := float64(p) / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower], nil
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight, nil
}

// GroupByPeriod groups data points by the specified time period.
func GroupByPeriod(points []DataPoint, period Period) (map[string][]DataPoint, error) {
	if len(points) == 0 {
		return nil, ErrEmptyDataset
	}

	result := make(map[string][]DataPoint)
	for _, p := range points {
		key, err := periodKey(p.Timestamp, period)
		if err != nil {
			return nil, err
		}
		result[key] = append(result[key], p)
	}

	return result, nil
}

// periodKey returns a string key for grouping timestamps by period.
func periodKey(t time.Time, period Period) (string, error) {
	switch period {
	case PeriodHourly:
		return t.Format("2006-01-02-15"), nil
	case PeriodDaily:
		return t.Format("2006-01-02"), nil
	case PeriodWeekly:
		year, week := t.ISOWeek()
		return t.Format("2006") + "-W" + padInt(week, 2) + "-" + padInt(year, 4), nil
	case PeriodMonthly:
		return t.Format("2006-01"), nil
	default:
		return "", ErrInvalidPeriod
	}
}

// padInt pads an integer with leading zeros to the specified width.
func padInt(n, width int) string {
	s := ""
	for range width {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// FilterByRange filters data points to only include those within the time range.
func FilterByRange(points []DataPoint, tr TimeRange) ([]DataPoint, error) {
	if err := tr.Validate(); err != nil {
		return nil, err
	}

	var filtered []DataPoint
	for _, p := range points {
		if tr.Contains(p.Timestamp) {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return nil, ErrNoDataInRange
	}

	return filtered, nil
}

// AggregatePoints calculates an aggregated result from a slice of data points.
func AggregatePoints(points []DataPoint, period Period) (*AggregatedResult, error) {
	if len(points) == 0 {
		return nil, ErrEmptyDataset
	}

	values := make([]float64, len(points))
	var minTime, maxTime time.Time

	for i, p := range points {
		values[i] = p.Value
		if i == 0 || p.Timestamp.Before(minTime) {
			minTime = p.Timestamp
		}
		if i == 0 || p.Timestamp.After(maxTime) {
			maxTime = p.Timestamp
		}
	}

	stats, err := Calculate(values)
	if err != nil {
		return nil, err
	}

	// Calculate common percentiles
	percentiles := make(map[int]float64)
	for _, p := range []int{25, 50, 75, 90, 95, 99} {
		val, pErr := Percentile(values, p)
		if pErr == nil {
			percentiles[p] = val
		}
	}

	return &AggregatedResult{
		Period:     period,
		StartTime:  minTime,
		EndTime:    maxTime,
		Count:      stats.Count,
		Sum:        stats.Sum,
		Min:        stats.Min,
		Max:        stats.Max,
		Avg:        stats.Avg,
		Median:     stats.Median,
		StdDev:     stats.StdDev,
		Percentile: percentiles,
	}, nil
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
	if len(values) < 2 {
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

// SumByLabel groups data points by label and sums their values.
func SumByLabel(points []DataPoint) map[string]float64 {
	result := make(map[string]float64)
	for _, p := range points {
		result[p.Label] += p.Value
	}
	return result
}

// CountByLabel groups data points by label and counts occurrences.
func CountByLabel(points []DataPoint) map[string]int {
	result := make(map[string]int)
	for _, p := range points {
		result[p.Label]++
	}
	return result
}

// TopN returns the top N data points by value (highest first).
func TopN(points []DataPoint, n int) []DataPoint {
	if n <= 0 || len(points) == 0 {
		return nil
	}
	if n >= len(points) {
		sorted := make([]DataPoint, len(points))
		copy(sorted, points)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Value > sorted[j].Value
		})
		return sorted
	}

	sorted := make([]DataPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	return sorted[:n]
}

// BottomN returns the bottom N data points by value (lowest first).
func BottomN(points []DataPoint, n int) []DataPoint {
	if n <= 0 || len(points) == 0 {
		return nil
	}
	if n >= len(points) {
		sorted := make([]DataPoint, len(points))
		copy(sorted, points)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Value < sorted[j].Value
		})
		return sorted
	}

	sorted := make([]DataPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value < sorted[j].Value
	})

	return sorted[:n]
}

// Normalize scales values to be between 0 and 1.
func Normalize(values []float64) ([]float64, error) {
	if len(values) == 0 {
		return nil, ErrEmptyDataset
	}

	minValue := values[0]
	maxValue := values[0]
	for _, v := range values {
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
	}

	rangeVal := maxValue - minValue
	if rangeVal == 0 {
		// All values are the same, return all 0.5
		result := make([]float64, len(values))
		for i := range result {
			result[i] = 0.5
		}
		return result, nil
	}

	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = (v - minValue) / rangeVal
	}

	return result, nil
}

// Outliers identifies outliers using the IQR method.
// Returns indices of values that are outliers (below Q1-1.5*IQR or above Q3+1.5*IQR).
func Outliers(values []float64) ([]int, error) {
	if len(values) < 4 {
		return nil, errors.New("at least 4 values required for outlier detection")
	}

	q1, err := Percentile(values, 25)
	if err != nil {
		return nil, err
	}
	q3, err := Percentile(values, 75)
	if err != nil {
		return nil, err
	}

	iqr := q3 - q1
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	var outlierIndices []int
	for i, v := range values {
		if v < lowerBound || v > upperBound {
			outlierIndices = append(outlierIndices, i)
		}
	}

	return outlierIndices, nil
}
