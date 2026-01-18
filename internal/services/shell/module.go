// Package shell provides security posture assessment, device discovery, and vulnerability scanning.
// Color: Orange #ea580c
package shell

import (
	"context"
	"sync"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// Module is the main Shell module providing security services.
type Module struct {
	mu            sync.RWMutex
	cfg           *config.Config
	db            *database.DB
	discovery     *DiscoveryService
	vulnerability *VulnerabilityService
	posture       *PostureService
	rogue         *RogueService
}

// New creates a new Shell module instance.
func New(cfg *config.Config, db *database.DB) *Module {
	m := &Module{
		cfg: cfg,
		db:  db,
	}

	m.discovery = NewDiscoveryService(cfg, db)
	m.vulnerability = NewVulnerabilityService(cfg, db)
	m.posture = NewPostureService(cfg, db, m.discovery, m.vulnerability)
	m.rogue = NewRogueService(cfg)

	return m
}

// Discovery returns the device discovery service.
func (m *Module) Discovery() *DiscoveryService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.discovery
}

// Vulnerability returns the vulnerability scanning service.
func (m *Module) Vulnerability() *VulnerabilityService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vulnerability
}

// Posture returns the security posture service.
func (m *Module) Posture() *PostureService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.posture
}

// Rogue returns the rogue device detection service.
func (m *Module) Rogue() *RogueService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rogue
}

// Start initializes and starts the Shell module services.
func (m *Module) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Add Shell module config and start discovery/rogue detection if enabled
	_ = ctx

	return nil
}

// Stop gracefully shuts down all Shell module services.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.discovery != nil {
		m.discovery.Stop()
	}

	if m.rogue != nil {
		m.rogue.Stop()
	}

	if m.vulnerability != nil {
		m.vulnerability.Stop()
	}

	return nil
}
