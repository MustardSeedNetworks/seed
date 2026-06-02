// Package reporting provides report generation and data export capabilities.
package reporting

import (
	"context"
	"sync"

	"github.com/krisarmstrong/seed/internal/config"
)

// Module is the reporting component providing report generation services.
type Module struct {
	mu         sync.RWMutex
	cfg        *config.Config
	generator  *GeneratorService
	templates  *TemplateService
	scheduler  *SchedulerService
	aggregator *AggregatorService
}

// Deps holds the persistence adapters the reporting component depends on. The
// composition root (internal/app) implements these ports with the SQLite
// adapters in internal/reporting/store; the component itself is persistence-free.
type Deps struct {
	Reports  ReportRepo
	Schedule ScheduleRepo
	Metrics  MetricsRepo
	Export   ExportRepo
}

// New creates a new reporting component instance from its port dependencies.
func New(cfg *config.Config, deps Deps) *Module {
	m := &Module{cfg: cfg}

	// Create services in dependency order:
	// 1. Templates (no dependencies)
	// 2. Aggregator (needs the metrics repo)
	// 3. Generator (needs report + export repos + templates + aggregator)
	// 4. Scheduler (needs the schedule repo + generator)
	m.templates = NewTemplateService(cfg)
	m.aggregator = NewAggregatorService(cfg, deps.Metrics)
	m.generator = NewGeneratorService(cfg, deps.Reports, deps.Export, m.templates, m.aggregator)
	m.scheduler = NewSchedulerService(cfg, deps.Schedule, m.generator)

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

// Start initializes and starts the reporting component services.
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

// Stop gracefully shuts down all reporting component services.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scheduler != nil {
		m.scheduler.Stop()
	}

	return nil
}
