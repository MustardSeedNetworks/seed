package app

// health.go wires the composition root to the health-monitoring application
// (use-case) service (ADR-0020): the read-and-control surface over the
// health-check subsystem — results/history persistence, scoring, SLA tracking,
// alerting, and anomaly detection. The adapters below implement the narrow ports
// declared in internal/health/monitoring over the concrete collaborators (the
// health-check repository, scorer, SLA tracker, alert manager, and anomaly
// detector), so the API handlers depend on the use-case instead of reaching into
// the service container directly. Each collaborator is resolved through a lazy
// accessor per call so a later-set value (the api test harness) is honored, and
// a nil collaborator degrades the relevant method to ErrUnavailable rather than
// panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/health"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
)

// NewHealthMonitoring builds the health-monitoring use-case (ADR-0020) over lazy
// accessors for its five collaborators. A nil accessor result makes the matching
// methods degrade to monitoring.ErrUnavailable (the golden-pinned 503 path).
func NewHealthMonitoring(
	repo func() *database.HealthCheckRepository,
	scorer func() *health.ScoringService,
	sla func() *health.SLATracker,
	alertMgr func() *alerts.AlertManager,
	anomalyStore func() *database.AnomalyRepository,
) *monitoring.Service {
	return monitoring.NewService(
		healthResultStore{repo: repo},
		healthScorer{scorer: scorer},
		healthSLA{sla: sla},
		healthAlerts{mgr: alertMgr},
		healthAnomaly{store: anomalyStore},
	)
}

// healthResultStore implements monitoring.ResultStore over the health-check
// repository, resolving it lazily. Methods beyond Available are only invoked by
// the use-case once Available reports true, so they assume a non-nil repository.
type healthResultStore struct {
	repo func() *database.HealthCheckRepository
}

func (a healthResultStore) Available() bool { return a.repo() != nil }

func (a healthResultStore) Query(
	ctx context.Context, opts database.HealthCheckQueryOptions,
) ([]*database.HealthCheckResult, error) {
	return a.repo().Query(ctx, opts)
}

func (a healthResultStore) LatestForAllEndpoints(
	ctx context.Context,
) ([]*database.HealthCheckResult, error) {
	return a.repo().GetLatestForAllEndpoints(ctx)
}

func (a healthResultStore) DailyRollups(
	ctx context.Context, checkType, endpoint string, tr database.TimeRange,
) ([]*database.HealthCheckDailyRollup, error) {
	return a.repo().GetDailyRollups(ctx, checkType, endpoint, tr)
}

func (a healthResultStore) HourlyRollups(
	ctx context.Context, checkType, endpoint string, tr database.TimeRange,
) ([]*database.HealthCheckHourlyRollup, error) {
	return a.repo().GetHourlyRollups(ctx, checkType, endpoint, tr)
}

// healthScorer implements monitoring.Scorer over the scoring service.
type healthScorer struct {
	scorer func() *health.ScoringService
}

func (a healthScorer) Available() bool { return a.scorer() != nil }
func (a healthScorer) AllScores(ctx context.Context) ([]*health.EndpointHealthScore, error) {
	return a.scorer().CalculateAllScores(ctx)
}

// healthSLA implements monitoring.SLAReporter over the SLA tracker.
type healthSLA struct {
	sla func() *health.SLATracker
}

func (a healthSLA) Available() bool { return a.sla() != nil }
func (a healthSLA) CurrentPeriodReport(
	ctx context.Context, endpoint string,
) (*health.SLAReport, error) {
	return a.sla().GenerateCurrentPeriodReport(ctx, endpoint)
}

func (a healthSLA) Summary(ctx context.Context, period string) (*health.SLASummary, error) {
	return a.sla().GenerateSummary(ctx, period)
}

// healthAlerts implements monitoring.AlertReader over the alert manager.
type healthAlerts struct {
	mgr func() *alerts.AlertManager
}

func (a healthAlerts) Available() bool                     { return a.mgr() != nil }
func (a healthAlerts) ActiveAlerts() []*alerts.HealthAlert { return a.mgr().GetActiveAlerts() }
func (a healthAlerts) Stats() alerts.AlertStats            { return a.mgr().GetAlertStats() }
func (a healthAlerts) Acknowledge(alertID, acknowledgedBy string) bool {
	return a.mgr().AcknowledgeAlert(alertID, acknowledgedBy)
}

// healthAnomaly implements monitoring.AnomalyReader over the unified anomaly
// store (ADR-0021), reading the source=probe slice — the active-monitoring probe
// engine is the producer of these anomalies (ADR-0025). It replaced the bespoke,
// never-fed health.AnomalyDetector that was deleted when anomalies converged on
// the single engine; per-endpoint rolling statistics went away with it.
type healthAnomaly struct {
	store func() *database.AnomalyRepository
}

func (a healthAnomaly) Available() bool { return a.store() != nil }
func (a healthAnomaly) ActiveAnomalies(ctx context.Context) ([]anomaly.Anomaly, error) {
	return a.store().ActiveBySource(ctx, anomaly.SourceProbe)
}
