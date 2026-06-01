// Package roots provides network path analysis, topology mapping, and IP enrichment.
// Color: Amber #b45309
package roots

import (
	"context"
	"sync"

	"github.com/krisarmstrong/seed/internal/config"
)

// Module is the main Roots module providing path analysis and topology services.
// It is persistence-free: none of its services touch the database (topology
// persistence will arrive behind a repo port when it is implemented).
type Module struct {
	mu         sync.RWMutex
	cfg        *config.Config
	traceroute *TracerouteService
	topology   *TopologyService
	enrichment *EnrichmentService
	analysis   *AnalysisService
}

// New creates a new Roots module instance.
func New(cfg *config.Config) *Module {
	m := &Module{cfg: cfg}

	m.traceroute = NewTracerouteService(cfg)
	m.topology = NewTopologyService(cfg)
	m.enrichment = NewEnrichmentService(cfg)
	m.analysis = NewAnalysisService(cfg)

	return m
}

// Traceroute returns the traceroute service.
func (m *Module) Traceroute() *TracerouteService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.traceroute
}

// Topology returns the topology service.
func (m *Module) Topology() *TopologyService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.topology
}

// Enrichment returns the IP enrichment service.
func (m *Module) Enrichment() *EnrichmentService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enrichment
}

// Analysis returns the path analysis service.
func (m *Module) Analysis() *AnalysisService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.analysis
}

// Start initializes and starts the Roots module services.
func (m *Module) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Add Roots module config and start topology discovery if enabled
	_ = ctx

	return nil
}

// Stop gracefully shuts down all Roots module services.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.topology != nil {
		m.topology.Stop()
	}

	return nil
}
