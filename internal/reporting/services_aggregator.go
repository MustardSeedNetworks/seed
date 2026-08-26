package reporting

// services_aggregator.go contains AggregatorService: rolls up devices,
// vulnerabilities, performance, and top issues over a period; also exposes
// GetTrends for time-series chart data.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
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

// TrendSummary describes where a metric's series has been and where it is
// heading, alongside the series itself.
//
// A bare series answers "what were the numbers"; a reader still has to work out
// whether anything is moving. These are the three questions that get asked of
// one in practice: what does the tail look like (P95), what does it look like
// with the noise taken out (Smoothed), and is it climbing (Change).
type TrendSummary struct {
	Points []DataPoint `json:"points"`

	// P50 and P95 over the whole window. Absent when the series is empty.
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`

	// Smoothed is a moving average over smoothingWindow points, or nil when
	// the series is shorter than the window.
	Smoothed []float64 `json:"smoothed,omitempty"`

	// Change is the proportional change between consecutive points, so a
	// reader can see acceleration rather than inferring it from the shape.
	Change []float64 `json:"change,omitempty"`
}

const (
	// smoothingWindow is the moving-average window for trend smoothing. Five
	// points takes the jitter out of a series without hiding a real step change.
	smoothingWindow = 5
	// medianPercentile and tailPercentile are the two the summary reports: the
	// middle of the series and its tail.
	medianPercentile = 50
	tailPercentile   = 95
)

// GetTrends retrieves trend data for a metric, with the summary statistics a
// chart or a report needs alongside it.
func (s *AggregatorService) GetTrends(
	ctx context.Context,
	metric, period string,
) (*TrendSummary, error) {
	points, err := s.metrics.Trends(ctx, metric, period)
	if err != nil {
		return nil, err
	}

	summary := &TrendSummary{Points: points}
	if len(points) == 0 {
		return summary, nil
	}

	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}

	// Percentiles are defined for any non-empty series, so an error here is
	// not possible and is not worth propagating as though it were.
	summary.P50, _ = Percentile(values, medianPercentile)
	summary.P95, _ = Percentile(values, tailPercentile)

	// The smoothing and rate calculations need a minimum number of points and
	// simply do not apply to a shorter series. A series too short to smooth is
	// not an error — it is a series you can read directly.
	if smoothed, smoothErr := MovingAverage(values, smoothingWindow); smoothErr == nil {
		summary.Smoothed = smoothed
	}
	if change, changeErr := RateOfChange(values); changeErr == nil {
		summary.Change = change
	}

	return summary, nil
}
