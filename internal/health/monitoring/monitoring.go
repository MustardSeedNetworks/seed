// Package monitoring holds the health-monitoring application (use-case) layer
// (ADR-0020). After the dead health_check_results read-path was deleted
// (ADR-0026), its single remaining concern is reporting active anomalies from
// the unified store (ADR-0021), produced by the active-monitoring probe engine
// (source=probe, ADR-0025). The handler keeps transport concerns (request
// decode, query-parameter parsing, response shaping, error-to-status mapping);
// the adapter satisfying the port lives in the composition root (internal/app)
// and resolves its collaborator lazily, so a nil collaborator degrades to
// ErrUnavailable (the golden-pinned 503) rather than panicking.
package monitoring

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
)

// ErrUnavailable signals the anomaly collaborator is not wired (handlers map it
// to 503, the golden-pinned degraded behavior).
var ErrUnavailable = errors.New("health monitoring service not available")

// AnomalyReader is the anomaly surface the use-case reads. It reads the
// source=probe slice of the unified anomaly store (ADR-0021/0025). Available
// reports whether the store is wired; ActiveAnomalies is only invoked once
// availability is confirmed.
type AnomalyReader interface {
	Available() bool
	ActiveAnomalies(ctx context.Context) ([]anomaly.Anomaly, error)
}

// Service is the health-monitoring use-case.
type Service struct {
	anomaly AnomalyReader
}

// NewService builds the use-case over its narrow dependency.
func NewService(anomalyReader AnomalyReader) *Service {
	return &Service{anomaly: anomalyReader}
}

// Anomalies is the use-case read model for detected anomalies: the (optionally
// endpoint-filtered) anomaly list and the total active count across all subjects.
type Anomalies struct {
	Anomalies   []anomaly.Anomaly
	ActiveCount int
}

// Anomalies returns the active anomalies, optionally filtered to a single
// endpoint (matched against the anomaly subject id). ActiveCount always reflects
// the total active anomalies independent of the filter.
func (s *Service) Anomalies(ctx context.Context, endpoint string) (Anomalies, error) {
	if !s.anomaly.Available() {
		return Anomalies{}, ErrUnavailable
	}
	active, err := s.anomaly.ActiveAnomalies(ctx)
	if err != nil {
		return Anomalies{}, err
	}
	out := Anomalies{ActiveCount: len(active)}

	if endpoint == "" {
		out.Anomalies = active
		return out, nil
	}
	for _, a := range active {
		if a.Subject.ID == endpoint {
			out.Anomalies = append(out.Anomalies, a)
		}
	}
	return out, nil
}
