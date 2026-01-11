package health

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// Anomaly severity levels.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Anomaly types.
const (
	AnomalyTypeLatencySpike    = "latency_spike"
	AnomalyTypeLatencyDrop     = "latency_improvement"
	AnomalyTypeAvailabilityDip = "availability_dip"
	AnomalyTypeErrorSpike      = "error_spike"
	AnomalyTypePatternChange   = "pattern_change"
)

// Statistical thresholds for anomaly detection.
const (
	// DefaultStdDevThreshold is the number of standard deviations for anomaly detection.
	DefaultStdDevThreshold = 2.0

	// MinSamplesForDetection is the minimum number of samples needed before anomaly detection.
	MinSamplesForDetection = 10

	// MaxSamplesForStats is the maximum number of samples to keep for rolling statistics.
	MaxSamplesForStats = 100

	// DefaultAnomalyWindow is the default time window for anomaly detection.
	DefaultAnomalyWindow = 1 * time.Hour

	// MinSamplesForStdDev is the minimum samples needed to calculate standard deviation.
	MinSamplesForStdDev = 2

	// SeverityWarningDeviation is the deviation threshold for warning severity.
	SeverityWarningDeviation = 3.0

	// SeverityCriticalDeviation is the deviation threshold for critical severity.
	SeverityCriticalDeviation = 4.0
)

// Anomaly represents a detected anomaly in health check metrics.
type Anomaly struct {
	ID           string    `json:"id"`
	EndpointName string    `json:"endpointName"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Value        float64   `json:"value"`
	Expected     float64   `json:"expected"`
	Deviation    float64   `json:"deviation"` // How many stddevs from mean
	DetectedAt   time.Time `json:"detectedAt"`
	ResolvedAt   time.Time `json:"resolvedAt,omitzero"`
	IsResolved   bool      `json:"isResolved"`
}

// EndpointStats holds rolling statistics for an endpoint.
type EndpointStats struct {
	EndpointName string    `json:"endpointName"`
	Mean         float64   `json:"mean"`
	StdDev       float64   `json:"stdDev"`
	Min          float64   `json:"min"`
	Max          float64   `json:"max"`
	SampleCount  int       `json:"sampleCount"`
	LastValue    float64   `json:"lastValue"`
	LastUpdate   time.Time `json:"lastUpdate"`
	samples      []float64 // Rolling window of samples
}

// AnomalyDetector provides statistical anomaly detection for health metrics.
type AnomalyDetector struct {
	mu              sync.RWMutex
	stats           map[string]*EndpointStats // endpoint name -> stats
	activeAnomalies map[string]*Anomaly       // anomaly ID -> anomaly
	stdDevThreshold float64
	maxSamples      int
	onAnomaly       func(*Anomaly)
}

// AnomalyDetectorConfig configures the anomaly detector.
type AnomalyDetectorConfig struct {
	// StdDevThreshold is the number of standard deviations to consider an anomaly.
	StdDevThreshold float64

	// MaxSamples is the maximum number of samples to keep for rolling statistics.
	MaxSamples int

	// OnAnomaly is called when an anomaly is detected.
	OnAnomaly func(*Anomaly)
}

// NewAnomalyDetector creates a new anomaly detector.
func NewAnomalyDetector(cfg AnomalyDetectorConfig) *AnomalyDetector {
	threshold := cfg.StdDevThreshold
	if threshold == 0 {
		threshold = DefaultStdDevThreshold
	}

	maxSamples := cfg.MaxSamples
	if maxSamples == 0 {
		maxSamples = MaxSamplesForStats
	}

	return &AnomalyDetector{
		stats:           make(map[string]*EndpointStats),
		activeAnomalies: make(map[string]*Anomaly),
		stdDevThreshold: threshold,
		maxSamples:      maxSamples,
		onAnomaly:       cfg.OnAnomaly,
	}
}

// RecordLatency records a latency measurement and checks for anomalies.
func (ad *AnomalyDetector) RecordLatency(endpointName string, latencyMs float64) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	stats := ad.getOrCreateStats(endpointName)
	ad.addSample(stats, latencyMs)

	// Need enough samples for meaningful detection
	if stats.SampleCount < MinSamplesForDetection {
		return nil
	}

	return ad.checkLatencyAnomaly(endpointName, latencyMs, stats)
}

// CheckLatency checks if a latency value is anomalous without recording it.
func (ad *AnomalyDetector) CheckLatency(endpointName string, latencyMs float64) *Anomaly {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	stats, exists := ad.stats[endpointName]
	if !exists || stats.SampleCount < MinSamplesForDetection {
		return nil
	}

	return ad.checkLatencyAnomaly(endpointName, latencyMs, stats)
}

// checkLatencyAnomaly checks if a latency value is anomalous.
func (ad *AnomalyDetector) checkLatencyAnomaly(endpointName string, latencyMs float64, stats *EndpointStats) *Anomaly {
	if stats.StdDev == 0 {
		return nil
	}

	deviation := (latencyMs - stats.Mean) / stats.StdDev

	// Check for high latency spike
	if deviation > ad.stdDevThreshold {
		anomaly := &Anomaly{
			ID:           fmt.Sprintf("%s-%d", endpointName, time.Now().UnixNano()),
			EndpointName: endpointName,
			Type:         AnomalyTypeLatencySpike,
			Severity:     ad.getSeverity(deviation),
			Message:      fmt.Sprintf("Latency %.0fms exceeds normal range (mean: %.0fms, stddev: %.0fms)", latencyMs, stats.Mean, stats.StdDev),
			Value:        latencyMs,
			Expected:     stats.Mean,
			Deviation:    deviation,
			DetectedAt:   time.Now(),
		}

		ad.activeAnomalies[anomaly.ID] = anomaly
		if ad.onAnomaly != nil {
			ad.onAnomaly(anomaly)
		}

		return anomaly
	}

	// Check for significant improvement (negative anomaly)
	if deviation < -ad.stdDevThreshold {
		anomaly := &Anomaly{
			ID:           fmt.Sprintf("%s-%d", endpointName, time.Now().UnixNano()),
			EndpointName: endpointName,
			Type:         AnomalyTypeLatencyDrop,
			Severity:     SeverityInfo,
			Message:      fmt.Sprintf("Latency %.0fms significantly below normal (mean: %.0fms)", latencyMs, stats.Mean),
			Value:        latencyMs,
			Expected:     stats.Mean,
			Deviation:    deviation,
			DetectedAt:   time.Now(),
		}

		// Don't track positive anomalies as active
		if ad.onAnomaly != nil {
			ad.onAnomaly(anomaly)
		}

		return anomaly
	}

	return nil
}

// RecordAvailability records availability and checks for anomalies.
func (ad *AnomalyDetector) RecordAvailability(endpointName string, successRate float64) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	statsKey := endpointName + "_availability"
	stats := ad.getOrCreateStats(statsKey)
	ad.addSample(stats, successRate)

	if stats.SampleCount < MinSamplesForDetection {
		return nil
	}

	return ad.checkAvailabilityAnomaly(endpointName, successRate, stats)
}

// checkAvailabilityAnomaly checks if availability is anomalously low.
func (ad *AnomalyDetector) checkAvailabilityAnomaly(endpointName string, successRate float64, stats *EndpointStats) *Anomaly {
	if stats.StdDev == 0 {
		return nil
	}

	deviation := (successRate - stats.Mean) / stats.StdDev

	// Only check for availability drops (negative deviations)
	if deviation < -ad.stdDevThreshold {
		anomaly := &Anomaly{
			ID:           fmt.Sprintf("%s-avail-%d", endpointName, time.Now().UnixNano()),
			EndpointName: endpointName,
			Type:         AnomalyTypeAvailabilityDip,
			Severity:     ad.getSeverity(math.Abs(deviation)),
			Message:      fmt.Sprintf("Availability %.1f%% below normal (mean: %.1f%%)", successRate*100, stats.Mean*100),
			Value:        successRate,
			Expected:     stats.Mean,
			Deviation:    deviation,
			DetectedAt:   time.Now(),
		}

		ad.activeAnomalies[anomaly.ID] = anomaly
		if ad.onAnomaly != nil {
			ad.onAnomaly(anomaly)
		}

		return anomaly
	}

	return nil
}

// GetStats returns the current statistics for an endpoint.
func (ad *AnomalyDetector) GetStats(endpointName string) (*EndpointStats, bool) {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	stats, exists := ad.stats[endpointName]
	if !exists {
		return nil, false
	}

	// Return a copy without the internal samples slice
	return &EndpointStats{
		EndpointName: stats.EndpointName,
		Mean:         stats.Mean,
		StdDev:       stats.StdDev,
		Min:          stats.Min,
		Max:          stats.Max,
		SampleCount:  stats.SampleCount,
		LastValue:    stats.LastValue,
		LastUpdate:   stats.LastUpdate,
	}, true
}

// GetAllStats returns statistics for all endpoints.
func (ad *AnomalyDetector) GetAllStats() map[string]*EndpointStats {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	result := make(map[string]*EndpointStats, len(ad.stats))
	for name, stats := range ad.stats {
		result[name] = &EndpointStats{
			EndpointName: stats.EndpointName,
			Mean:         stats.Mean,
			StdDev:       stats.StdDev,
			Min:          stats.Min,
			Max:          stats.Max,
			SampleCount:  stats.SampleCount,
			LastValue:    stats.LastValue,
			LastUpdate:   stats.LastUpdate,
		}
	}

	return result
}

// GetActiveAnomalies returns all currently active (unresolved) anomalies.
func (ad *AnomalyDetector) GetActiveAnomalies() []*Anomaly {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	anomalies := make([]*Anomaly, 0, len(ad.activeAnomalies))
	for _, a := range ad.activeAnomalies {
		anomalies = append(anomalies, a)
	}

	return anomalies
}

// ResolveAnomaly marks an anomaly as resolved.
func (ad *AnomalyDetector) ResolveAnomaly(anomalyID string) bool {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	anomaly, exists := ad.activeAnomalies[anomalyID]
	if !exists {
		return false
	}

	anomaly.IsResolved = true
	anomaly.ResolvedAt = time.Now()
	delete(ad.activeAnomalies, anomalyID)

	return true
}

// ResolveEndpointAnomalies resolves all active anomalies for an endpoint.
func (ad *AnomalyDetector) ResolveEndpointAnomalies(endpointName string) int {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	resolved := 0
	for id, anomaly := range ad.activeAnomalies {
		if anomaly.EndpointName == endpointName {
			anomaly.IsResolved = true
			anomaly.ResolvedAt = time.Now()
			delete(ad.activeAnomalies, id)
			resolved++
		}
	}

	return resolved
}

// ClearStats clears all statistics for an endpoint.
func (ad *AnomalyDetector) ClearStats(endpointName string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	delete(ad.stats, endpointName)
	delete(ad.stats, endpointName+"_availability")
}

// ClearAllStats clears all statistics.
func (ad *AnomalyDetector) ClearAllStats() {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.stats = make(map[string]*EndpointStats)
	ad.activeAnomalies = make(map[string]*Anomaly)
}

// getOrCreateStats gets or creates stats for an endpoint.
func (ad *AnomalyDetector) getOrCreateStats(name string) *EndpointStats {
	stats, exists := ad.stats[name]
	if !exists {
		stats = &EndpointStats{
			EndpointName: name,
			samples:      make([]float64, 0, ad.maxSamples),
			Min:          math.MaxFloat64,
			Max:          -math.MaxFloat64,
		}
		ad.stats[name] = stats
	}
	return stats
}

// addSample adds a sample to the rolling statistics.
func (ad *AnomalyDetector) addSample(stats *EndpointStats, value float64) {
	// Add to rolling window
	if len(stats.samples) >= ad.maxSamples {
		// Remove oldest sample
		stats.samples = stats.samples[1:]
	}
	stats.samples = append(stats.samples, value)

	// Update statistics
	stats.LastValue = value
	stats.LastUpdate = time.Now()
	stats.SampleCount = len(stats.samples)

	// Update min/max
	if value < stats.Min {
		stats.Min = value
	}
	if value > stats.Max {
		stats.Max = value
	}

	// Recalculate mean and stddev
	ad.recalculateStats(stats)
}

// recalculateStats recalculates mean and standard deviation.
func (ad *AnomalyDetector) recalculateStats(stats *EndpointStats) {
	n := len(stats.samples)
	if n == 0 {
		stats.Mean = 0
		stats.StdDev = 0
		return
	}

	// Calculate mean
	sum := 0.0
	for _, v := range stats.samples {
		sum += v
	}
	stats.Mean = sum / float64(n)

	// Calculate standard deviation
	if n < MinSamplesForStdDev {
		stats.StdDev = 0
		return
	}

	sumSquares := 0.0
	for _, v := range stats.samples {
		diff := v - stats.Mean
		sumSquares += diff * diff
	}
	stats.StdDev = math.Sqrt(sumSquares / float64(n-1))
}

// getSeverity determines anomaly severity based on deviation.
func (ad *AnomalyDetector) getSeverity(deviation float64) string {
	absDeviation := math.Abs(deviation)

	switch {
	case absDeviation > SeverityCriticalDeviation:
		return SeverityCritical
	case absDeviation > SeverityWarningDeviation:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

// AutoResolveStaleAnomalies resolves anomalies older than the specified duration.
func (ad *AnomalyDetector) AutoResolveStaleAnomalies(_ context.Context, maxAge time.Duration) int {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	resolved := 0
	cutoff := time.Now().Add(-maxAge)

	for id, anomaly := range ad.activeAnomalies {
		if anomaly.DetectedAt.Before(cutoff) {
			anomaly.IsResolved = true
			anomaly.ResolvedAt = time.Now()
			delete(ad.activeAnomalies, id)
			resolved++
		}
	}

	return resolved
}
