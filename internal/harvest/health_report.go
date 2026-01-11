package harvest

// This file adds health check monitoring data integration for harvest reports.

import (
	"context"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/health"
)

// Health report constants.
const (
	// criticalityScale is the divisor for converting criticality score to 1-10 scale.
	criticalityScale = 10
	// maxCriticality is the maximum criticality value.
	maxCriticality = 10
	// percentUptime is the multiplier for converting ratio to percentage.
	percentUptime = 100
	// queryLimitLarge is the limit for large queries.
	queryLimitLarge = 10000
	// percentile95 is the percentile value for P95 calculations.
	percentile95 = 0.95
	// slaComplianceTarget is the target SLA compliance percentage.
	slaComplianceTarget = 99.0
	// latencyThreshold is the latency threshold in ms for recommendations.
	latencyThreshold = 100
)

// Health status constants.
const (
	statusHealthy  = "healthy"
	statusDegraded = "degraded"
	statusCritical = "critical"
)

// HealthReportData contains aggregated health check data for reports.
type HealthReportData struct {
	// Summary statistics
	TotalEndpoints  int     `json:"totalEndpoints"`
	HealthyCount    int     `json:"healthyCount"`
	DegradedCount   int     `json:"degradedCount"`
	CriticalCount   int     `json:"criticalCount"`
	OverallUptime   float64 `json:"overallUptime"`   // Percentage
	AvgLatencyMs    float64 `json:"avgLatencyMs"`    // Average latency across all endpoints
	AvgLatencyP95Ms float64 `json:"avgLatencyP95Ms"` // P95 latency

	// SLA compliance
	SLACompliance   float64             `json:"slaCompliance"` // Percentage of endpoints meeting SLA
	EndpointsMet    int                 `json:"endpointsMet"`
	EndpointsMissed int                 `json:"endpointsMissed"`
	SLAViolations   []SLAViolationEntry `json:"slaViolations,omitempty"`

	// Health scores per endpoint
	EndpointScores []EndpointScoreEntry `json:"endpointScores"`

	// Active alerts and anomalies
	ActiveAlerts   int `json:"activeAlerts"`
	AlertsCritical int `json:"alertsCritical"`
	AlertsWarning  int `json:"alertsWarning"`
	Anomalies      int `json:"anomalies"`

	// Time range
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// SLAViolationEntry represents an SLA violation for reporting.
type SLAViolationEntry struct {
	EndpointName   string    `json:"endpointName"`
	ViolationType  string    `json:"violationType"` // "uptime" or "latency"
	TargetValue    float64   `json:"targetValue"`
	ActualValue    float64   `json:"actualValue"`
	ViolationStart time.Time `json:"violationStart"`
}

// EndpointScoreEntry contains health score data for a single endpoint.
type EndpointScoreEntry struct {
	Name             string  `json:"name"`
	CheckType        string  `json:"checkType"`
	AvailabilityPct  float64 `json:"availabilityPct"`
	LatencyAvgMs     float64 `json:"latencyAvgMs"`
	LatencyP95Ms     float64 `json:"latencyP95Ms"`
	CompositeScore   float64 `json:"compositeScore"`
	Status           string  `json:"status"` // healthy, degraded, critical
	Criticality      int     `json:"criticality"`
	ChecksTotal      int     `json:"checksTotal"`
	ChecksSuccessful int     `json:"checksSuccessful"`
}

// HealthReportBuilder generates health check data for reports.
type HealthReportBuilder struct {
	db         *database.DB
	repository *database.HealthCheckRepository
	scorer     *health.ScoringService
	slaTracker *health.SLATracker
}

// NewHealthReportBuilder creates a new health report builder.
func NewHealthReportBuilder(
	db *database.DB,
	repository *database.HealthCheckRepository,
	scorer *health.ScoringService,
	slaTracker *health.SLATracker,
) *HealthReportBuilder {
	return &HealthReportBuilder{
		db:         db,
		repository: repository,
		scorer:     scorer,
		slaTracker: slaTracker,
	}
}

// BuildHealthReportData generates health report data for the specified time range.
func (b *HealthReportBuilder) BuildHealthReportData(ctx context.Context, start, end time.Time) (*HealthReportData, error) {
	data := &HealthReportData{
		PeriodStart: start,
		PeriodEnd:   end,
		GeneratedAt: time.Now().UTC(),
	}

	// Get health scores if scorer is available
	if b.scorer != nil {
		scores, err := b.scorer.CalculateAllScores(ctx)
		if err == nil {
			data.TotalEndpoints = len(scores)
			for _, score := range scores {
				// Convert criticality score (0-100) back to 1-10 scale
				criticality := int(score.CriticalityScore / criticalityScale)
				if criticality < 1 {
					criticality = 1
				}
				if criticality > maxCriticality {
					criticality = maxCriticality
				}

				entry := EndpointScoreEntry{
					Name:             score.EndpointName,
					CheckType:        score.CheckType,
					AvailabilityPct:  score.AvailabilityPct,
					LatencyAvgMs:     score.P95LatencyMs, // Use actual P95 from score
					LatencyP95Ms:     score.P95LatencyMs,
					CompositeScore:   score.CompositeScore,
					Status:           score.Status,
					Criticality:      criticality,
					ChecksTotal:      int(score.TotalChecks),
					ChecksSuccessful: int(score.SuccessfulChecks),
				}

				switch score.Status {
				case statusHealthy:
					data.HealthyCount++
				case statusDegraded:
					data.DegradedCount++
				case statusCritical:
					data.CriticalCount++
				}

				data.EndpointScores = append(data.EndpointScores, entry)
			}
		}
	}

	// Get SLA compliance if SLA tracker is available
	if b.slaTracker != nil {
		summary, err := b.slaTracker.GenerateSummary(ctx, "daily")
		if err == nil && summary != nil {
			data.SLACompliance = summary.ComplianceRate
			data.EndpointsMet = summary.EndpointsMet
			data.EndpointsMissed = summary.EndpointsMissed
		}
	}

	// Calculate aggregate statistics from repository
	if b.repository != nil {
		timeRange := database.TimeRange{Start: start, End: end}

		// Get all results in the time range
		results, err := b.repository.Query(ctx, database.HealthCheckQueryOptions{
			TimeRange: timeRange,
			Limit:     queryLimitLarge,
		})
		if err == nil && len(results) > 0 {
			var totalLatency float64
			var successCount int
			latencies := make([]float64, 0, len(results))

			for _, r := range results {
				if r.Success {
					successCount++
				}
				if r.LatencyMs > 0 {
					totalLatency += r.LatencyMs
					latencies = append(latencies, r.LatencyMs)
				}
			}

			if len(results) > 0 {
				data.OverallUptime = float64(successCount) / float64(len(results)) * percentUptime
			}
			if len(latencies) > 0 {
				data.AvgLatencyMs = totalLatency / float64(len(latencies))
				data.AvgLatencyP95Ms = calculateP95(latencies)
			}
		}
	}

	return data, nil
}

// calculateP95 calculates the 95th percentile of a slice of values.
func calculateP95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sortFloat64s(sorted)

	// Calculate P95 index
	index := int(float64(len(sorted)-1) * percentile95)
	return sorted[index]
}

// sortFloat64s sorts a slice of float64 in ascending order.
func sortFloat64s(values []float64) {
	for i := 1; i < len(values); i++ {
		key := values[i]
		j := i - 1
		for j >= 0 && values[j] > key {
			values[j+1] = values[j]
			j--
		}
		values[j+1] = key
	}
}

// FormatHealthSummary returns a formatted text summary for reports.
func (d *HealthReportData) FormatHealthSummary() string {
	return fmt.Sprintf(
		"Health Check Summary\n"+
			"====================\n"+
			"Period: %s to %s\n\n"+
			"Endpoints: %d total (%d healthy, %d degraded, %d critical)\n"+
			"Overall Uptime: %.2f%%\n"+
			"Average Latency: %.2fms (P95: %.2fms)\n"+
			"SLA Compliance: %.2f%% (%d met, %d missed)\n"+
			"Active Alerts: %d (%d critical, %d warning)\n"+
			"Anomalies Detected: %d\n",
		d.PeriodStart.Format("2006-01-02"),
		d.PeriodEnd.Format("2006-01-02"),
		d.TotalEndpoints,
		d.HealthyCount,
		d.DegradedCount,
		d.CriticalCount,
		d.OverallUptime,
		d.AvgLatencyMs,
		d.AvgLatencyP95Ms,
		d.SLACompliance,
		d.EndpointsMet,
		d.EndpointsMissed,
		d.ActiveAlerts,
		d.AlertsCritical,
		d.AlertsWarning,
		d.Anomalies,
	)
}

// GetStatusBreakdown returns a map of status counts for charting.
func (d *HealthReportData) GetStatusBreakdown() map[string]int {
	return map[string]int{
		"healthy":  d.HealthyCount,
		"degraded": d.DegradedCount,
		"critical": d.CriticalCount,
	}
}

// GetTopCriticalEndpoints returns endpoints with critical status or low scores.
func (d *HealthReportData) GetTopCriticalEndpoints(limit int) []EndpointScoreEntry {
	critical := make([]EndpointScoreEntry, 0)
	for _, e := range d.EndpointScores {
		if e.Status == statusCritical || e.Status == statusDegraded {
			critical = append(critical, e)
		}
	}

	// Sort by composite score (lowest first)
	for i := 1; i < len(critical); i++ {
		key := critical[i]
		j := i - 1
		for j >= 0 && critical[j].CompositeScore > key.CompositeScore {
			critical[j+1] = critical[j]
			j--
		}
		critical[j+1] = key
	}

	if len(critical) > limit {
		return critical[:limit]
	}
	return critical
}

// GetRecommendations generates recommendations based on health data.
func (d *HealthReportData) GetRecommendations() []string {
	recommendations := make([]string, 0)

	if d.CriticalCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("CRITICAL: %d endpoint(s) in critical state require immediate attention", d.CriticalCount))
	}

	if d.SLACompliance < slaComplianceTarget {
		recommendations = append(recommendations,
			fmt.Sprintf("SLA compliance (%.2f%%) is below target. Review failing endpoints.", d.SLACompliance))
	}

	if d.AvgLatencyMs > latencyThreshold {
		recommendations = append(recommendations,
			fmt.Sprintf("Average latency (%.2fms) is elevated. Consider infrastructure optimization.", d.AvgLatencyMs))
	}

	if d.AlertsCritical > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d critical alert(s) active. Review alert dashboard for details.", d.AlertsCritical))
	}

	if d.Anomalies > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d anomaly(ies) detected. Investigate unusual patterns.", d.Anomalies))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"All health checks are within normal parameters. Continue monitoring.")
	}

	return recommendations
}
