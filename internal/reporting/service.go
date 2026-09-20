// Package reporting provides report generation and data export capabilities.
package reporting

import (
	"context"
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// Service is the reporting component providing report generation services.
type Service struct {
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
func New(cfg *config.Config, deps Deps) *Service {
	m := &Service{cfg: cfg}

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
func (m *Service) Generator() *GeneratorService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.generator
}

// Templates returns the template management service.
func (m *Service) Templates() *TemplateService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.templates
}

// Scheduler returns the scheduled report service.
func (m *Service) Scheduler() *SchedulerService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scheduler
}

// Aggregator returns the data aggregation service.
func (m *Service) Aggregator() *AggregatorService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.aggregator
}

// Load reads the report templates and the persisted schedules. It is separate
// from Run and called synchronously at startup (#2748): a load failure is a
// misconfigured install the operator must see immediately, and a Run that
// returned it instead would be retried four times in microseconds and then
// take the outbox relay and the Wi-Fi loops down with it, while the daemon
// carried on serving HTTP.
func (m *Service) Load(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.templates.Load(); err != nil {
		return err
	}
	return m.scheduler.Load(ctx)
}

// Run ticks the schedule loop until ctx is cancelled. It blocks so the
// component can be a supervised worker; Load must have succeeded first.
func (m *Service) Run(ctx context.Context) error {
	m.mu.RLock()
	scheduler := m.scheduler
	m.mu.RUnlock()

	return scheduler.Run(ctx)
}
