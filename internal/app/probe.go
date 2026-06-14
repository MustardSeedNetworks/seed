package app

// probe.go wires the composition root to the probe engine's persistence port
// (ADR-0020 / WS-B). The probe package declares a consumer-side persistence
// port in its own domain types (probe.Probe / probe.Result) and stays
// persistence-free; the adapter below implements that port over the concrete
// *database.ProbeRepository, translating to and from the database row
// representation. The translation that used to live inside internal/probe
// (dbProbeToModel / the Result→ProbeResult build) lives here instead, so the
// probe package no longer imports internal/database.

import (
	"context"
	"encoding/json"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// ProbeStorage adapts *database.ProbeRepository to the probe engine's
// consumer-side persistence port, translating between the database row types
// and the probe package's domain types.
type ProbeStorage struct {
	repo *database.ProbeRepository
}

// NewProbeStorage builds the probe-engine persistence adapter over the concrete
// probe repository. The result satisfies the probe engine's persistence port,
// so it is passed to Engine.WithStorage in the composition root.
func NewProbeStorage(repo *database.ProbeRepository) *ProbeStorage {
	return &ProbeStorage{repo: repo}
}

// GetProbe loads one probe by ID and translates it to the domain type.
func (s *ProbeStorage) GetProbe(ctx context.Context, id string) (probe.Probe, error) {
	p, err := s.repo.GetProbe(ctx, id)
	if err != nil {
		return probe.Probe{}, err
	}
	return dbProbeToModel(p), nil
}

// ListProbes lists probes matching the client/kind filters and translates them.
func (s *ProbeStorage) ListProbes(ctx context.Context, clientID, kind string) ([]probe.Probe, error) {
	rows, err := s.repo.ListProbes(ctx, clientID, kind)
	if err != nil {
		return nil, err
	}
	out := make([]probe.Probe, 0, len(rows))
	for _, p := range rows {
		out = append(out, dbProbeToModel(p))
	}
	return out, nil
}

// RecordResult persists a dispatch result, translating it into the database row
// representation. The metadata column is TEXT and accepts opaque JSON.
func (s *ProbeStorage) RecordResult(ctx context.Context, r probe.Result) error {
	return s.repo.RecordResult(ctx, &database.ProbeResult{
		ProbeID:      r.ProbeID,
		ClientID:     r.ClientID,
		Kind:         r.Kind,
		Timestamp:    r.Timestamp,
		Success:      r.Success,
		LatencyMs:    r.LatencyMs,
		Error:        r.Error,
		MetadataJSON: string(r.Metadata),
	})
}

// modelToDBProbe converts the probe package's domain Probe into the database
// row representation. The inverse of dbProbeToModel: the [json.RawMessage] columns
// become TEXT. ID/ClientID/timestamps pass through (empty on the save path, where
// the repository fills them).
func modelToDBProbe(p probe.Probe) *database.Probe {
	return &database.Probe{
		ID:              p.ID,
		ClientID:        p.ClientID,
		Kind:            p.Kind,
		DisplayName:     p.DisplayName,
		Target:          p.Target,
		ParamsJSON:      string(p.Params),
		IntervalSeconds: p.IntervalSeconds,
		Enabled:         p.Enabled,
		WarningJSON:     string(p.Warning),
		CriticalJSON:    string(p.Critical),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// dbProbeToModel converts the database row representation into the probe
// package's Probe type. The JSON columns are passed through as
// [json.RawMessage]; consumers decode them per-Kind.
func dbProbeToModel(p *database.Probe) probe.Probe {
	return probe.Probe{
		ID:              p.ID,
		ClientID:        p.ClientID,
		Kind:            p.Kind,
		DisplayName:     p.DisplayName,
		Target:          p.Target,
		Params:          json.RawMessage(p.ParamsJSON),
		IntervalSeconds: p.IntervalSeconds,
		Enabled:         p.Enabled,
		Warning:         json.RawMessage(p.WarningJSON),
		Critical:        json.RawMessage(p.CriticalJSON),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
