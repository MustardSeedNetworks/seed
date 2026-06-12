// Package troubleshooting holds the Wi-Fi troubleshooting application (use-case)
// services the API handlers call (ADR-0020 clean-hexagonal). It exposes three
// cohesive use-cases over one Wi-Fi capability — Queries (airspace/anomaly
// visibility reads), Management (radio/interface settings, scan, status,
// connect), and Discovery (enhanced vendor/authorization/channel scan) — plus
// AnalyzeBSSes for the survey path. The consumer-defined ports live here at the
// consumer (interface-segregation); the composition root (internal/app) builds
// the adapters and injects the use-cases, keeping internal/api pure transport.
package troubleshooting

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// VisibilitySource is the narrow capability the read use-cases need from the
// live visibility component for the airspace tree and status summary. It is
// defined here, at the consumer (ADR-0016 interface-segregation), and satisfied
// by *visibility.Service. The anomaly list no longer comes from here — after the
// engine convergence (ADR-0029 §4) it is read from the unified store via
// AnomalyStore, mirroring the probe endpoint.
type VisibilitySource interface {
	Tree() []airspace.SSIDGroup
	Status() visibility.Status
}

// AnomalyStore reads the Wi-Fi slice of the unified anomaly store (ADR-0029 §4).
// Available reports whether the store is wired; ActiveWiFi returns the active
// source=wifi instances. Defined at the consumer; the composition root satisfies
// it over the anomaly repository, symmetric with the health endpoint's reader.
type AnomalyStore interface {
	Available() bool
	ActiveWiFi(ctx context.Context) ([]anomaly.Anomaly, error)
}

// AirspaceResult is the airspace read use-case output: the tree plus the
// capture/entity status. HTTP-agnostic — the handler maps it to the wire DTO.
type AirspaceResult struct {
	SSIDs  []airspace.SSIDGroup
	Status visibility.Status
}

// AnomaliesResult is the anomaly read use-case output: the stream plus status.
type AnomaliesResult struct {
	Anomalies []anomaly.Anomaly
	Status    visibility.Status
}

// Queries is the read use-case for Wi-Fi visibility. The API handlers depend on
// it instead of on the background components directly. The airspace tree and
// status come from the live visibility component; the anomaly list comes from the
// unified store (ADR-0029 §4).
type Queries struct {
	src   VisibilitySource
	store AnomalyStore
}

// NewQueries builds the read use-case over the visibility source (tree/status)
// and the anomaly store (the source=wifi anomaly list). Either may be nil (no
// capture component / no DB, e.g. the test harness): the queries then degrade to
// empty-but-valid results.
func NewQueries(src VisibilitySource, store AnomalyStore) *Queries {
	return &Queries{src: src, store: store}
}

// Airspace returns the current airspace tree and status, degrading to an empty
// tree (never nil) when no visibility source is present.
func (q *Queries) Airspace() AirspaceResult {
	if q == nil || q.src == nil {
		return AirspaceResult{SSIDs: []airspace.SSIDGroup{}}
	}
	return AirspaceResult{SSIDs: q.src.Tree(), Status: q.src.Status()}
}

// Anomalies returns the active Wi-Fi anomalies from the unified store plus the
// live status summary (ADR-0029 §4). The status always comes from the visibility
// source; the list comes from the store. An unwired store degrades to an empty
// list (never nil) — Wi-Fi's historic graceful contract — while a genuine store
// error is surfaced to the caller (mapped to 500 by the handler).
func (q *Queries) Anomalies(ctx context.Context) (AnomaliesResult, error) {
	status := visibility.Status{}
	if q != nil && q.src != nil {
		status = q.src.Status()
	}
	if q == nil || q.store == nil || !q.store.Available() {
		return AnomaliesResult{Anomalies: []anomaly.Anomaly{}, Status: status}, nil
	}
	list, err := q.store.ActiveWiFi(ctx)
	if err != nil {
		return AnomaliesResult{}, err
	}
	if list == nil {
		list = []anomaly.Anomaly{}
	}
	return AnomaliesResult{Anomalies: list, Status: status}, nil
}
