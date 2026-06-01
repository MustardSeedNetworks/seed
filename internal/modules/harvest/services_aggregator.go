package harvest

// services_aggregator.go contains AggregatorService: rolls up devices,
// vulnerabilities, performance, and top issues over a period; also exposes
// GetTrends for time-series chart data.

import (
	"context"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// AggregatorService aggregates data for reports. It reads metrics through the
// MetricsRepo port and owns the severity-bucket / category semantics.
type AggregatorService struct {
	cfg     *config.Config
	metrics MetricsRepo
}

// NewAggregatorService creates a new aggregator service.
func NewAggregatorService(cfg *config.Config, metrics MetricsRepo) *AggregatorService {
	return &AggregatorService{
		cfg:     cfg,
		metrics: metrics,
	}
}

// Aggregate collects and aggregates data for a time period.
func (s *AggregatorService) Aggregate(
	ctx context.Context,
	period, _, _ string,
) (*AggregatedData, error) {
	// Calculate date range based on period
	now := time.Now()
	var startDate time.Time

	switch period {
	case PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
	case PeriodWeekly:
		startDate = now.AddDate(0, 0, -7)
	case PeriodMonthly:
		startDate = now.AddDate(0, -1, 0)
	default:
		startDate = now.AddDate(0, 0, -7) // Default to weekly
	}

	data := &AggregatedData{
		Period:    period,
		StartDate: startDate,
		EndDate:   now,
	}

	// Aggregate device count (best-effort: a query error leaves the zero value)
	data.DeviceCount, _ = s.metrics.CountDevices(ctx)

	// Aggregate vulnerability counts
	s.aggregateVulnerabilities(ctx, data, startDate)

	// Aggregate performance metrics
	s.aggregatePerformance(ctx, data, startDate)

	// Get top issues
	s.aggregateTopIssues(ctx, data)

	return data, nil
}

func (s *AggregatorService) aggregateVulnerabilities(
	ctx context.Context,
	data *AggregatedData,
	since time.Time,
) {
	// The domain owns the meaning of each severity bucket; the repo returns the
	// raw severity → count. A query error leaves VulnCount at its zero value.
	counts, _ := s.metrics.VulnerabilitySeverityCounts(ctx, since)
	for severity, count := range counts {
		switch severity {
		case statusCritical:
			data.VulnCount.Critical = count
		case "high":
			data.VulnCount.High = count
		case "medium":
			data.VulnCount.Medium = count
		case "low":
			data.VulnCount.Low = count
		}
		data.VulnCount.Total += count
	}
}

func (s *AggregatorService) aggregatePerformance(
	ctx context.Context,
	data *AggregatedData,
	since time.Time,
) {
	data.Performance, _ = s.metrics.PerformanceMetrics(ctx, since)
}

func (s *AggregatorService) aggregateTopIssues(ctx context.Context, data *AggregatedData) {
	issues, _ := s.metrics.TopIssues(ctx)
	data.TopIssues = append(data.TopIssues, issues...)
}

// GetTrends retrieves trend data for a metric.
func (s *AggregatorService) GetTrends(
	ctx context.Context,
	metric, period string,
) ([]DataPoint, error) {
	return s.metrics.Trends(ctx, metric, period)
}
