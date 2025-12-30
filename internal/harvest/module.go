// Package harvest provides report generation and data export capabilities.
// Color: Gold #d4a017
package harvest

import (
	"context"
	"sync"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// Module is the main Harvest module providing reporting services.
type Module struct {
	mu         sync.RWMutex
	cfg        *config.Config
	db         *database.DB
	generator  *GeneratorService
	templates  *TemplateService
	scheduler  *SchedulerService
	aggregator *AggregatorService
}

// New creates a new Harvest module instance.
func New(cfg *config.Config, db *database.DB) *Module {
	m := &Module{
		cfg: cfg,
		db:  db,
	}

	// Create services in dependency order:
	// 1. Templates (no dependencies)
	// 2. Aggregator (needs db)
	// 3. Generator (needs templates + aggregator)
	// 4. Scheduler (needs generator)
	m.templates = NewTemplateService(cfg)
	m.aggregator = NewAggregatorService(cfg, db)
	m.generator = NewGeneratorService(cfg, db, m.templates, m.aggregator)
	m.scheduler = NewSchedulerService(cfg, db, m.generator)

	return m
}

// Generator returns the report generator service.
func (m *Module) Generator() *GeneratorService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.generator
}

// Templates returns the template management service.
func (m *Module) Templates() *TemplateService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.templates
}

// Scheduler returns the scheduled report service.
func (m *Module) Scheduler() *SchedulerService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scheduler
}

// Aggregator returns the data aggregation service.
func (m *Module) Aggregator() *AggregatorService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.aggregator
}

// Start initializes and starts the Harvest module services.
func (m *Module) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load templates
	if err := m.templates.Load(); err != nil {
		return err
	}

	// Start scheduler for recurring reports
	if err := m.scheduler.Start(ctx); err != nil {
		return err
	}

	return nil
}

// Stop gracefully shuts down all Harvest module services.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scheduler != nil {
		m.scheduler.Stop()
	}

	return nil
}
