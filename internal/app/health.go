package app

// health.go wires the composition root to the health-monitoring application
// (use-case) service (ADR-0020). After the dead health_check_results read-path
// was deleted (ADR-0026), the use-case's only concern is reporting active
// anomalies from the unified store (ADR-0021), produced by the active-monitoring
// probe engine (source=probe, ADR-0025). The adapter below implements the narrow
// AnomalyReader port over the concrete anomaly repository, resolved lazily per
// call so a later-set value (the api test harness) is honored; a nil store
// degrades to monitoring.ErrUnavailable rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
)

// NewHealthMonitoring builds the health-monitoring use-case (ADR-0020) over a
// lazy accessor for the unified anomaly store. A nil accessor result makes
// Anomalies degrade to monitoring.ErrUnavailable (the golden-pinned 503 path).
func NewHealthMonitoring(anomalyStore func() *database.AnomalyRepository) *monitoring.Service {
	return monitoring.NewService(healthAnomaly{store: anomalyStore})
}

// healthAnomaly implements monitoring.AnomalyReader over the unified anomaly
// store (ADR-0021), reading the source=probe slice — the active-monitoring probe
// engine is the producer of these anomalies (ADR-0025).
type healthAnomaly struct {
	store func() *database.AnomalyRepository
}

func (a healthAnomaly) Available() bool { return a.store() != nil }
func (a healthAnomaly) ActiveAnomalies(ctx context.Context) ([]anomaly.Anomaly, error) {
	return a.store().ActiveBySource(ctx, anomaly.SourceProbe)
}
