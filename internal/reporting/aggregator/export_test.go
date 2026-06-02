package aggregator

import "time"

// Export internal functions for testing

// ExportCalculateMedian exports the internal calculateMedian function for testing.
func ExportCalculateMedian(values []float64) float64 {
	return calculateMedian(values)
}

// ExportCalculateStdDev exports the internal calculateStdDev function for testing.
func ExportCalculateStdDev(values []float64, mean float64) float64 {
	return calculateStdDev(values, mean)
}

// ExportPeriodKey exports the internal periodKey function for testing.
func ExportPeriodKey(t time.Time, period Period) (string, error) {
	return periodKey(t, period)
}

// ExportPadInt exports the internal padInt function for testing.
func ExportPadInt(n, width int) string {
	return padInt(n, width)
}
