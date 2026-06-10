// Package monitoring holds the health-monitoring application (use-case) layer
// (ADR-0020). It owns the read-and-control surface over the health-check
// subsystem that previously lived in the api.Server health handlers — querying
// the latest check results and their history, computing endpoint health scores,
// reporting SLA compliance, surfacing active alerts (and acknowledging them),
// and reporting detected anomalies — behind narrow consumer-defined ports over
// the persistence repository, scorer, SLA tracker, alert manager, and anomaly
// detector. Handlers keep transport concerns: request decode, query-parameter
// parsing, response shaping, and error-to-status mapping. The adapters
// satisfying the ports live in the composition root (internal/app) and resolve
// each collaborator lazily, so a nil collaborator degrades the relevant method
// to ErrUnavailable (the golden-pinned 503) rather than panicking.
//
// The existing internal/health package is the domain core (scoring, anomaly,
// SLA, dependencies); this subpackage is the application layer that orchestrates
// it for the API, keeping the two cleanly separated.
package monitoring

import (
	"context"
	"errors"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/health"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	// ErrUnavailable signals the relevant collaborator is not wired (handlers
	// map it to 503, the golden-pinned degraded behavior).
	ErrUnavailable = errors.New("health monitoring service not available")
	// ErrAlertNotFound signals the alert is unknown or already acknowledged
	// (handlers map it to 404).
	ErrAlertNotFound = errors.New("alert not found or already acknowledged")
)

// Query limits and history-period vocabulary owned by the use-case.
const (
	resultsQueryLimit = 100
	historyQueryLimit = 1000

	period1h  = "1h"
	period6h  = "6h"
	period24h = "24h"
	period7d  = "7d"
	period30d = "30d"

	defaultPeriod = 24 * time.Hour
	hoursIn6h     = 6
	daysIn7d      = 7
	daysIn30d     = 30

	defaultSLAPeriod = "daily"
)

// periodDuration returns the look-back window for a period string, defaulting to
// 24h for unknown values (the pre-strangle behavior).
func periodDuration(period string) time.Duration {
	switch period {
	case period1h:
		return time.Hour
	case period6h:
		return hoursIn6h * time.Hour
	case period24h:
		return defaultPeriod
	case period7d:
		return daysIn7d * defaultPeriod
	case period30d:
		return daysIn30d * defaultPeriod
	default:
		return defaultPeriod
	}
}

// ResultStore is the health-check persistence surface the use-case reads,
// defined at the consumer (ADR-0020) and satisfied by an adapter over
// *database.HealthCheckRepository in internal/app. Available reports whether the
// repository is wired; the remaining methods are only invoked once availability
// is confirmed.
type ResultStore interface {
	Available() bool
	Query(ctx context.Context, opts database.HealthCheckQueryOptions) ([]*database.HealthCheckResult, error)
	LatestForAllEndpoints(ctx context.Context) ([]*database.HealthCheckResult, error)
	DailyRollups(
		ctx context.Context, checkType, endpoint string, tr database.TimeRange,
	) ([]*database.HealthCheckDailyRollup, error)
	HourlyRollups(
		ctx context.Context, checkType, endpoint string, tr database.TimeRange,
	) ([]*database.HealthCheckHourlyRollup, error)
}

// Scorer is the health-scoring surface the use-case reads.
type Scorer interface {
	Available() bool
	AllScores(ctx context.Context) ([]*health.EndpointHealthScore, error)
}

// SLAReporter is the SLA-tracking surface the use-case reads.
type SLAReporter interface {
	Available() bool
	CurrentPeriodReport(ctx context.Context, endpoint string) (*health.SLAReport, error)
	Summary(ctx context.Context, period string) (*health.SLASummary, error)
}

// AlertReader is the health-alert surface the use-case reads and controls.
type AlertReader interface {
	Available() bool
	ActiveAlerts() []*alerts.HealthAlert
	Stats() alerts.AlertStats
	Acknowledge(alertID, acknowledgedBy string) bool
}

// AnomalyReader is the anomaly-detection surface the use-case reads.
type AnomalyReader interface {
	Available() bool
	ActiveAnomalies() []*health.Anomaly
	AllStats() map[string]*health.EndpointStats
}

// Service is the health-monitoring use-case.
type Service struct {
	results ResultStore
	scorer  Scorer
	sla     SLAReporter
	alerts  AlertReader
	anomaly AnomalyReader
}

// NewService builds the use-case over its narrow dependencies.
func NewService(
	results ResultStore,
	scorer Scorer,
	sla SLAReporter,
	alertReader AlertReader,
	anomaly AnomalyReader,
) *Service {
	return &Service{results: results, scorer: scorer, sla: sla, alerts: alertReader, anomaly: anomaly}
}

// Results returns the latest check results, optionally filtered by endpoint
// and/or check type. With no filter it returns the latest result per endpoint.
func (s *Service) Results(
	ctx context.Context, endpoint, checkType string,
) ([]*database.HealthCheckResult, error) {
	if !s.results.Available() {
		return nil, ErrUnavailable
	}
	if endpoint != "" || checkType != "" {
		return s.results.Query(ctx, database.HealthCheckQueryOptions{
			CheckType:    checkType,
			EndpointName: endpoint,
			Limit:        resultsQueryLimit,
		})
	}
	return s.results.LatestForAllEndpoints(ctx)
}

// HistoryKind discriminates the payload carried by a History read model.
type HistoryKind string

const (
	// HistoryDailyRollups is returned for the 7d/30d periods.
	HistoryDailyRollups HistoryKind = "daily_rollups"
	// HistoryHourlyRollups is returned for the 6h/24h periods.
	HistoryHourlyRollups HistoryKind = "hourly_rollups"
	// HistoryRaw is returned for short/unknown periods.
	HistoryRaw HistoryKind = "raw"
)

// History is the use-case read model for historical health-check data: the
// resolved time window plus exactly one of daily rollups, hourly rollups, or raw
// results, selected by the requested period.
type History struct {
	Kind          HistoryKind
	Period        string
	Start         time.Time
	End           time.Time
	DailyRollups  []*database.HealthCheckDailyRollup
	HourlyRollups []*database.HealthCheckHourlyRollup
	Results       []*database.HealthCheckResult
}

// History returns historical data for a period, choosing daily rollups for long
// periods, hourly rollups for medium periods, and raw results otherwise.
func (s *Service) History(ctx context.Context, endpoint, checkType, period string) (History, error) {
	if !s.results.Available() {
		return History{}, ErrUnavailable
	}
	end := time.Now()
	start := end.Add(-periodDuration(period))
	tr := database.TimeRange{Start: start, End: end}
	out := History{Period: period, Start: start, End: end}

	switch period {
	case period7d, period30d:
		rollups, err := s.results.DailyRollups(ctx, checkType, endpoint, tr)
		if err != nil {
			return History{}, err
		}
		out.Kind = HistoryDailyRollups
		out.DailyRollups = rollups
	case period6h, period24h:
		rollups, err := s.results.HourlyRollups(ctx, checkType, endpoint, tr)
		if err != nil {
			return History{}, err
		}
		out.Kind = HistoryHourlyRollups
		out.HourlyRollups = rollups
	default:
		results, err := s.results.Query(ctx, database.HealthCheckQueryOptions{
			CheckType:    checkType,
			EndpointName: endpoint,
			TimeRange:    tr,
			Limit:        historyQueryLimit,
		})
		if err != nil {
			return History{}, err
		}
		out.Kind = HistoryRaw
		out.Results = results
	}
	return out, nil
}

// ScoreSummary tallies endpoint scores by status.
type ScoreSummary struct {
	TotalEndpoints int
	Healthy        int
	Degraded       int
	Critical       int
	Unknown        int
}

// Scores is the use-case read model for computed health scores plus their
// status tally.
type Scores struct {
	Scores  []*health.EndpointHealthScore
	Summary ScoreSummary
}

// Scores returns computed health scores for all endpoints with a status tally.
func (s *Service) Scores(ctx context.Context) (Scores, error) {
	if !s.scorer.Available() {
		return Scores{}, ErrUnavailable
	}
	scores, err := s.scorer.AllScores(ctx)
	if err != nil {
		return Scores{}, err
	}
	summary := ScoreSummary{TotalEndpoints: len(scores)}
	for _, score := range scores {
		switch score.Status {
		case "healthy":
			summary.Healthy++
		case "degraded":
			summary.Degraded++
		case "critical":
			summary.Critical++
		default:
			summary.Unknown++
		}
	}
	return Scores{Scores: scores, Summary: summary}, nil
}

// SLAReport returns the current-period SLA report for a single endpoint.
func (s *Service) SLAReport(ctx context.Context, endpoint string) (*health.SLAReport, error) {
	if !s.sla.Available() {
		return nil, ErrUnavailable
	}
	return s.sla.CurrentPeriodReport(ctx, endpoint)
}

// SLASummary returns the SLA summary across all endpoints for a period,
// defaulting the period to "daily" when empty.
func (s *Service) SLASummary(ctx context.Context, period string) (*health.SLASummary, error) {
	if !s.sla.Available() {
		return nil, ErrUnavailable
	}
	if period == "" {
		period = defaultSLAPeriod
	}
	return s.sla.Summary(ctx, period)
}

// Alerts is the use-case read model for active health-check alerts.
type Alerts struct {
	Alerts []*alerts.HealthAlert
	Stats  alerts.AlertStats
}

// Alerts returns the active alerts and their aggregate statistics.
func (s *Service) Alerts() (Alerts, error) {
	if !s.alerts.Available() {
		return Alerts{}, ErrUnavailable
	}
	return Alerts{Alerts: s.alerts.ActiveAlerts(), Stats: s.alerts.Stats()}, nil
}

// AcknowledgeAlert acknowledges an alert, returning ErrAlertNotFound when the
// alert is unknown or already acknowledged.
func (s *Service) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	if !s.alerts.Available() {
		return ErrUnavailable
	}
	if !s.alerts.Acknowledge(alertID, acknowledgedBy) {
		return ErrAlertNotFound
	}
	return nil
}

// Anomalies is the use-case read model for detected anomalies: the (optionally
// endpoint-filtered) anomaly list, the total active count across all endpoints,
// and, when requested, per-endpoint statistics (also endpoint-filtered).
type Anomalies struct {
	Anomalies   []*health.Anomaly
	ActiveCount int
	Stats       []*health.EndpointStats
}

// Anomalies returns detected anomalies, optionally filtered to one endpoint.
// ActiveCount always reflects the total active anomalies (independent of the
// filter). Stats is populated only when includeStats is set.
func (s *Service) Anomalies(endpoint string, includeStats bool) (Anomalies, error) {
	if !s.anomaly.Available() {
		return Anomalies{}, ErrUnavailable
	}
	active := s.anomaly.ActiveAnomalies()
	out := Anomalies{ActiveCount: len(active)}

	if endpoint == "" {
		out.Anomalies = active
	} else {
		for _, a := range active {
			if a.EndpointName == endpoint {
				out.Anomalies = append(out.Anomalies, a)
			}
		}
	}

	if includeStats {
		for name, st := range s.anomaly.AllStats() {
			if endpoint == "" || name == endpoint {
				out.Stats = append(out.Stats, st)
			}
		}
	}
	return out, nil
}
