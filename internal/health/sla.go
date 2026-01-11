package health

import (
	"context"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

// SLA reporting periods.
const (
	ReportingPeriodDaily   = "daily"
	ReportingPeriodWeekly  = "weekly"
	ReportingPeriodMonthly = "monthly"
)

// Default SLA targets.
const (
	DefaultTargetUptime     = 99.9  // 99.9% uptime
	DefaultTargetLatencyP95 = 500.0 // 500ms P95 latency

	// PercentageMultiplier converts ratios to percentages.
	PercentageMultiplier = 100.0

	// P95Percentile is the percentile for P95 calculations.
	P95Percentile = 0.95
)

// SLAConfig defines the SLA targets for an endpoint.
type SLAConfig struct {
	EndpointName     string  `json:"name" yaml:"name"`
	TargetUptime     float64 `json:"targetUptime" yaml:"target_uptime"`          // e.g., 99.9 for 99.9%
	TargetLatencyP95 float64 `json:"targetLatencyP95" yaml:"target_latency_p95"` // ms
	ReportingPeriod  string  `json:"reportingPeriod" yaml:"reporting_period"`    // daily, weekly, monthly
	Enabled          bool    `json:"enabled" yaml:"enabled"`
}

// SLAViolation represents a single SLA violation event.
type SLAViolation struct {
	Timestamp   time.Time     `json:"timestamp"`
	Type        string        `json:"type"` // "uptime" or "latency"
	TargetValue float64       `json:"targetValue"`
	ActualValue float64       `json:"actualValue"`
	Duration    time.Duration `json:"duration,omitempty"`
	Description string        `json:"description"`
}

// SLAReport contains the SLA compliance report for an endpoint.
type SLAReport struct {
	EndpointName     string         `json:"endpointName"`
	Period           string         `json:"period"`
	PeriodStart      time.Time      `json:"periodStart"`
	PeriodEnd        time.Time      `json:"periodEnd"`
	ActualUptime     float64        `json:"actualUptime"` // Percentage
	TargetUptime     float64        `json:"targetUptime"` // Percentage
	UptimeMet        bool           `json:"uptimeMet"`
	ActualLatencyP95 float64        `json:"actualLatencyP95"` // ms
	TargetLatencyP95 float64        `json:"targetLatencyP95"` // ms
	LatencyMet       bool           `json:"latencyMet"`
	OverallMet       bool           `json:"overallMet"`
	TotalChecks      int64          `json:"totalChecks"`
	SuccessfulChecks int64          `json:"successfulChecks"`
	Violations       []SLAViolation `json:"violations,omitempty"`
	GeneratedAt      time.Time      `json:"generatedAt"`
}

// SLASummary provides an overview of SLA compliance across all endpoints.
type SLASummary struct {
	Period          string       `json:"period"`
	PeriodStart     time.Time    `json:"periodStart"`
	PeriodEnd       time.Time    `json:"periodEnd"`
	TotalEndpoints  int          `json:"totalEndpoints"`
	EndpointsMet    int          `json:"endpointsMet"`
	EndpointsMissed int          `json:"endpointsMissed"`
	ComplianceRate  float64      `json:"complianceRate"` // Percentage
	Reports         []*SLAReport `json:"reports"`
	GeneratedAt     time.Time    `json:"generatedAt"`
}

// SLATracker tracks and reports on SLA compliance.
type SLATracker struct {
	mu      sync.RWMutex
	repo    *database.HealthCheckRepository
	configs map[string]*SLAConfig // endpoint name -> config
}

// SLATrackerConfig configures the SLA tracker.
type SLATrackerConfig struct {
	Repository *database.HealthCheckRepository
	Configs    []SLAConfig
}

// NewSLATracker creates a new SLA tracker.
func NewSLATracker(cfg SLATrackerConfig) *SLATracker {
	tracker := &SLATracker{
		repo:    cfg.Repository,
		configs: make(map[string]*SLAConfig),
	}

	for i := range cfg.Configs {
		if cfg.Configs[i].Enabled {
			tracker.configs[cfg.Configs[i].EndpointName] = &cfg.Configs[i]
		}
	}

	return tracker
}

// SetConfig sets or updates the SLA configuration for an endpoint.
func (st *SLATracker) SetConfig(cfg SLAConfig) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if cfg.Enabled {
		st.configs[cfg.EndpointName] = &cfg
	} else {
		delete(st.configs, cfg.EndpointName)
	}
}

// GetConfig returns the SLA configuration for an endpoint.
func (st *SLATracker) GetConfig(endpointName string) (*SLAConfig, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	cfg, exists := st.configs[endpointName]
	return cfg, exists
}

// GetAllConfigs returns all SLA configurations.
func (st *SLATracker) GetAllConfigs() []SLAConfig {
	st.mu.RLock()
	defer st.mu.RUnlock()

	configs := make([]SLAConfig, 0, len(st.configs))
	for _, cfg := range st.configs {
		configs = append(configs, *cfg)
	}

	return configs
}

// GenerateReport generates an SLA report for an endpoint over a time period.
func (st *SLATracker) GenerateReport(ctx context.Context, endpointName string, start, end time.Time) (*SLAReport, error) {
	st.mu.RLock()
	cfg, exists := st.configs[endpointName]
	if !exists {
		// Use defaults if no config
		cfg = &SLAConfig{
			EndpointName:     endpointName,
			TargetUptime:     DefaultTargetUptime,
			TargetLatencyP95: DefaultTargetLatencyP95,
			ReportingPeriod:  ReportingPeriodDaily,
		}
	}
	st.mu.RUnlock()

	// Get historical data from repository
	history, err := st.repo.Query(ctx, database.HealthCheckQueryOptions{
		EndpointName: endpointName,
		TimeRange:    database.TimeRange{Start: start, End: end},
	})
	if err != nil {
		return nil, err
	}

	report := &SLAReport{
		EndpointName:     endpointName,
		Period:           cfg.ReportingPeriod,
		PeriodStart:      start,
		PeriodEnd:        end,
		TargetUptime:     cfg.TargetUptime,
		TargetLatencyP95: cfg.TargetLatencyP95,
		GeneratedAt:      time.Now(),
	}

	// Calculate metrics from history
	var totalLatency float64
	var latencies []float64
	var violations []SLAViolation

	for _, result := range history {
		report.TotalChecks++
		if result.Success {
			report.SuccessfulChecks++
		}
		if result.LatencyMs > 0 {
			totalLatency += result.LatencyMs
			latencies = append(latencies, result.LatencyMs)
		}
	}

	// Calculate uptime
	if report.TotalChecks > 0 {
		report.ActualUptime = float64(report.SuccessfulChecks) / float64(report.TotalChecks) * PercentageMultiplier
	}
	report.UptimeMet = report.ActualUptime >= cfg.TargetUptime

	// Calculate P95 latency
	if len(latencies) > 0 {
		report.ActualLatencyP95 = calculateP95(latencies)
	}
	report.LatencyMet = report.ActualLatencyP95 <= cfg.TargetLatencyP95

	// Overall compliance
	report.OverallMet = report.UptimeMet && report.LatencyMet

	// Record violations
	if !report.UptimeMet {
		violations = append(violations, SLAViolation{
			Timestamp:   end,
			Type:        "uptime",
			TargetValue: cfg.TargetUptime,
			ActualValue: report.ActualUptime,
			Description: "Uptime below target",
		})
	}

	if !report.LatencyMet {
		violations = append(violations, SLAViolation{
			Timestamp:   end,
			Type:        "latency",
			TargetValue: cfg.TargetLatencyP95,
			ActualValue: report.ActualLatencyP95,
			Description: "P95 latency above target",
		})
	}

	report.Violations = violations

	return report, nil
}

// GenerateCurrentPeriodReport generates a report for the current reporting period.
func (st *SLATracker) GenerateCurrentPeriodReport(ctx context.Context, endpointName string) (*SLAReport, error) {
	st.mu.RLock()
	cfg, exists := st.configs[endpointName]
	st.mu.RUnlock()

	period := ReportingPeriodDaily
	if exists {
		period = cfg.ReportingPeriod
	}

	start, end := getPeriodBounds(period, time.Now())
	return st.GenerateReport(ctx, endpointName, start, end)
}

// GenerateSummary generates an SLA summary for all configured endpoints.
func (st *SLATracker) GenerateSummary(ctx context.Context, period string) (*SLASummary, error) {
	st.mu.RLock()
	endpoints := make([]string, 0, len(st.configs))
	for name := range st.configs {
		endpoints = append(endpoints, name)
	}
	st.mu.RUnlock()

	start, end := getPeriodBounds(period, time.Now())

	summary := &SLASummary{
		Period:         period,
		PeriodStart:    start,
		PeriodEnd:      end,
		TotalEndpoints: len(endpoints),
		GeneratedAt:    time.Now(),
	}

	for _, name := range endpoints {
		report, err := st.GenerateReport(ctx, name, start, end)
		if err != nil {
			continue
		}

		summary.Reports = append(summary.Reports, report)

		if report.OverallMet {
			summary.EndpointsMet++
		} else {
			summary.EndpointsMissed++
		}
	}

	if summary.TotalEndpoints > 0 {
		summary.ComplianceRate = float64(summary.EndpointsMet) / float64(summary.TotalEndpoints) * PercentageMultiplier
	}

	return summary, nil
}

// CheckCompliance checks if an endpoint is currently meeting its SLA.
func (st *SLATracker) CheckCompliance(ctx context.Context, endpointName string) (bool, *SLAReport, error) {
	report, err := st.GenerateCurrentPeriodReport(ctx, endpointName)
	if err != nil {
		return false, nil, err
	}

	return report.OverallMet, report, nil
}

// getPeriodBounds returns the start and end times for a reporting period.
func getPeriodBounds(period string, now time.Time) (time.Time, time.Time) {
	now = now.UTC()
	end := now

	switch period {
	case ReportingPeriodDaily:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, end

	case ReportingPeriodWeekly:
		// Start from Monday of current week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		return start, end

	case ReportingPeriodMonthly:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, end

	default:
		// Default to daily
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, end
	}
}

// calculateP95 calculates the 95th percentile of a slice of values.
func calculateP95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values (simple insertion sort for small slices)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	// Calculate P95 index
	idx := int(float64(len(sorted)-1) * P95Percentile)
	return sorted[idx]
}

// GetViolationsForPeriod returns all SLA violations for a period.
func (st *SLATracker) GetViolationsForPeriod(ctx context.Context, period string) ([]SLAViolation, error) {
	summary, err := st.GenerateSummary(ctx, period)
	if err != nil {
		return nil, err
	}

	var allViolations []SLAViolation
	for _, report := range summary.Reports {
		allViolations = append(allViolations, report.Violations...)
	}

	return allViolations, nil
}
