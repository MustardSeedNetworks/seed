// Package wifiapp holds the Wi-Fi application (use-case) services that the API
// handlers call. It is the per-domain use-case layer from ADR-0016 (strangle
// internal/api): handlers decode/encode and delegate here, instead of reaching
// into the API service container or background components. The package name is
// wifiapp (not app) so the internal/app composition root can import it without
// an alias clash.
package wifiapp

import (
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// VisibilitySource is the narrow capability the read use-cases need from the
// live visibility component. It is defined here, at the consumer (ADR-0016
// interface-segregation), and satisfied by *visibility.Service.
type VisibilitySource interface {
	Tree() []airspace.SSIDGroup
	Anomalies() []anomaly.Anomaly
	Status() visibility.Status
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
// it instead of on the background components directly.
type Queries struct {
	src VisibilitySource
}

// NewQueries builds the read use-case over src. A nil src (no capture component
// wired) is valid: the queries then return empty-but-valid results.
func NewQueries(src VisibilitySource) *Queries { return &Queries{src: src} }

// Airspace returns the current airspace tree and status, degrading to an empty
// tree (never nil) when no visibility source is present.
func (q *Queries) Airspace() AirspaceResult {
	if q == nil || q.src == nil {
		return AirspaceResult{SSIDs: []airspace.SSIDGroup{}}
	}
	return AirspaceResult{SSIDs: q.src.Tree(), Status: q.src.Status()}
}

// Anomalies returns the current anomaly stream and status, degrading to an empty
// stream (never nil) when no visibility source is present.
func (q *Queries) Anomalies() AnomaliesResult {
	if q == nil || q.src == nil {
		return AnomaliesResult{Anomalies: []anomaly.Anomaly{}}
	}
	return AnomaliesResult{Anomalies: q.src.Anomalies(), Status: q.src.Status()}
}
