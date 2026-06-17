// Package pipeline consumes the observation + event streams that
// Stage A3.5 emits (listener_events for syslog/traps,
// snmp_observations for collector deltas) and emits structured
// alerts into the existing alerts table.
//
// Stage A4.5 ships the listener-event pipeline (syslog + traps);
// Stage A4.6 adds the observation-delta pipeline (interface
// transitions, BGP peer state, storage % thresholds).
//
// Rules are hardcoded for V1.0 — a small built-in set covering the
// "always alert on these" cases. User-configurable rules + a UI
// land in a follow-up stage; the engine + suppression machinery
// here is already shaped for that future.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// ListenerPipelineName is the engine identifier.
const ListenerPipelineName = "alert-listener-pipeline"

// Tunables — production defaults.
const (
	defaultBatch          = 500
	defaultInterval       = 15 * time.Second
	minInterval           = 100 * time.Millisecond
	defaultSuppression    = 5 * time.Minute
	defaultReloadInterval = 60 * time.Second
)

// listenerReader is the narrowed surface the listener pipeline
// needs from the listener_events repo. Tests inject a fake.
type listenerReader interface {
	List(ctx context.Context, opts listener.EventListOptions) ([]*listener.EventRecord, error)
}

// alertWriter is the narrowed surface for writing alerts.
type alertWriter interface {
	Create(ctx context.Context, alert *alerts.Alert) error
}

// settingsKV is the high-water-mark store. Same shape as
// internal/topology — duplicate type rather than a circular import.
type settingsKV interface {
	GetWithDefault(ctx context.Context, key, defaultValue string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// ListenerPipeline scans listener_events on a tick and writes alerts
// for every event matching any built-in rule. Suppression dedupes
// repeated alerts for the same (rule, source) within the window.
type ListenerPipeline struct {
	events         listenerReader
	alerts         alertWriter
	settings       settingsKV
	alertRules     alertRulesReader
	logger         *slog.Logger
	now            func() time.Time
	interval       time.Duration
	reloadInterval time.Duration
	staticRules    bool // true when Config.Rules was supplied; reloading disabled
	defaults       []Rule
	suppression    time.Duration

	mu         sync.Mutex
	started    bool
	stopped    bool
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	suppress   suppressionStore
	rules      []Rule
	lastTickAt time.Time
	lastError  string
}

// ListenerConfig wires the pipeline.
type ListenerConfig struct {
	Events      listenerReader
	Alerts      alertWriter
	Settings    settingsKV
	Logger      *slog.Logger
	Now         func() time.Time
	Interval    time.Duration
	Suppression time.Duration

	// AlertRules pulls operator-configured rows from alert_rules.
	// When set, the pipeline reloads rules from the table on each
	// ReloadInterval tick and falls back to DefaultListenerRules
	// only when the table is empty / has no enabled rows.
	// Ignored when Rules is also set (Rules wins for static-config
	// tests).
	AlertRules alertRulesReader

	// ReloadInterval controls how often AlertRules is re-queried.
	// Defaults to 60s; floored at 1s.
	ReloadInterval time.Duration

	// Rules override every other source; when set, the pipeline uses
	// exactly these rules and never reloads. Primary use is tests.
	// When nil and AlertRules is also nil, DefaultListenerRules() is
	// used permanently.
	Rules []Rule

	// Suppressions is the persistence backend for the per-(rule, entity)
	// "don't re-fire within the window" check. When nil, an in-memory
	// store is used (legacy — restart loses state). Production wires
	// NewDBSuppressionStore(db.AlertSuppressions()) for restart-safety
	// (#1380).
	Suppressions suppressionStore
}

// NewListenerPipeline returns an unstarted pipeline.
func NewListenerPipeline(cfg ListenerConfig) (*ListenerPipeline, error) {
	if cfg.Events == nil {
		return nil, errors.New("alerts: listener Events required")
	}
	if cfg.Alerts == nil {
		return nil, errors.New("alerts: listener Alerts writer required")
	}
	if cfg.Settings == nil {
		return nil, errors.New("alerts: listener Settings required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.Interval < minInterval {
		cfg.Interval = minInterval
	}
	if cfg.Suppression <= 0 {
		cfg.Suppression = defaultSuppression
	}
	if cfg.ReloadInterval <= 0 {
		cfg.ReloadInterval = defaultReloadInterval
	}
	if cfg.ReloadInterval < time.Second {
		cfg.ReloadInterval = time.Second
	}
	defaults := DefaultListenerRules()
	staticRules := cfg.Rules != nil
	initial := cfg.Rules
	if initial == nil {
		initial = defaults
	}
	suppress := cfg.Suppressions
	if suppress == nil {
		suppress = newInMemorySuppressionStore()
	}
	p := &ListenerPipeline{
		events:         cfg.Events,
		alerts:         cfg.Alerts,
		settings:       cfg.Settings,
		alertRules:     cfg.AlertRules,
		logger:         cfg.Logger,
		now:            cfg.Now,
		interval:       cfg.Interval,
		reloadInterval: cfg.ReloadInterval,
		staticRules:    staticRules,
		defaults:       defaults,
		suppression:    cfg.Suppression,
		rules:          initial,
		suppress:       suppress,
	}
	// Prime ruleset from the DB before returning so a caller that
	// jumps straight to ScanOnce (tests, single-shot use) sees the
	// operator's configured rules rather than DefaultListenerRules.
	// Failure here is non-fatal — the pipeline retains DefaultListenerRules
	// and the next reload tick gets another chance.
	if !staticRules && cfg.AlertRules != nil {
		if reloadErr := p.ReloadRules(context.Background()); reloadErr != nil {
			cfg.Logger.Warn("initial alert_rules load failed; using defaults",
				"error", reloadErr)
		}
	}
	return p, nil
}

// Name implements [engine.Engine].
func (*ListenerPipeline) Name() string { return ListenerPipelineName }

// Status implements [engine.Reporter]. See observation_pipeline.go
// for the state model — same rules apply here.
func (p *ListenerPipeline) Status() engine.Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := engine.Status{
		LastTickAt: p.lastTickAt,
		LastError:  p.lastError,
	}
	switch {
	case p.stopped:
		s.State = engine.StateStopped
	case p.lastTickAt.IsZero():
		s.State = engine.StateOK
	case p.now().Sub(p.lastTickAt) > degradedTickMultiplier*p.interval:
		s.State = engine.StateDegraded
	default:
		s.State = engine.StateOK
	}
	return s
}

// Start kicks off the scan loop. Idempotent.
func (p *ListenerPipeline) Start(ctx context.Context) error {
	// Prime the ruleset from DB before announcing started, so the
	// first scan reflects operator config rather than the default
	// rules for the first reload interval. A reload failure here is
	// non-fatal — the pipeline retains DefaultListenerRules until a
	// later reload succeeds.
	if !p.staticRulesView() {
		if reloadErr := p.ReloadRules(ctx); reloadErr != nil {
			p.logger.WarnContext(ctx, "initial alert_rules reload failed; using defaults",
				"error", reloadErr)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.started = true
	p.wg.Add(1)
	go p.loop(loopCtx)
	if !p.staticRules && p.alertRules != nil {
		p.wg.Add(1)
		go p.reloadLoop(loopCtx)
	}
	p.logger.InfoContext(ctx, "listener alert pipeline started",
		"interval", p.interval, "rules", len(p.rules),
		"reload_interval", p.reloadInterval, "db_rules_active", p.alertRules != nil && !p.staticRules)
	return nil
}

// Stop terminates the scan loop. Honors ctx deadline.
func (p *ListenerPipeline) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	p.started = false
	p.stopped = true
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Unlock()

	doneCh := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	p.logger.InfoContext(ctx, "listener alert pipeline stopped")
	return nil
}

func (p *ListenerPipeline) loop(ctx context.Context) {
	defer p.wg.Done()
	t := time.NewTicker(p.interval)
	defer t.Stop()
	if err := p.ScanOnce(ctx); err != nil {
		p.logger.WarnContext(ctx, "listener alert scan failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := p.ScanOnce(ctx); err != nil {
				p.logger.WarnContext(ctx, "listener alert scan failed", "error", err)
			}
		}
	}
}

// ScanOnce processes one batch of listener_events.
func (p *ListenerPipeline) ScanOnce(ctx context.Context) error {
	err := p.scanOnceInner(ctx)
	p.recordScan(err)
	return err
}

// recordScan stamps lastTickAt + lastError for engine.Reporter.
func (p *ListenerPipeline) recordScan(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastTickAt = p.now()
	if err != nil {
		p.lastError = err.Error()
		return
	}
	p.lastError = ""
}

func (p *ListenerPipeline) scanOnceInner(ctx context.Context) error {
	since, err := p.loadHighWater(ctx)
	if err != nil {
		return fmt.Errorf("load high-water: %w", err)
	}
	events, err := p.events.List(ctx, listener.EventListOptions{
		Since: since,
		Limit: defaultBatch,
	})
	if err != nil {
		return fmt.Errorf("list listener_events: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	var maxObservedAt time.Time
	alertsEmitted := 0
	for _, evt := range events {
		if evt.ObservedAt.After(maxObservedAt) {
			maxObservedAt = evt.ObservedAt
		}
		alertsEmitted += p.evaluate(ctx, evt)
	}
	p.logger.DebugContext(ctx, "listener alert pass",
		"batch", len(events), "alerts", alertsEmitted,
		"max_observed_at", maxObservedAt)

	if !maxObservedAt.IsZero() {
		if saveErr := p.saveHighWater(ctx, maxObservedAt); saveErr != nil {
			return fmt.Errorf("save high-water: %w", saveErr)
		}
	}
	return nil
}
