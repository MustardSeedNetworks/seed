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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/engine"
)

// ListenerPipelineName is the engine identifier.
const ListenerPipelineName = "alert-listener-pipeline"

// listenerHighWaterKey is the settings key holding the latest
// ObservedAt the listener alert pipeline has already processed.
const listenerHighWaterKey = "alerts.listener.high_water"

// Tunables — production defaults.
const (
	defaultBatch       = 500
	defaultInterval    = 15 * time.Second
	minInterval        = 100 * time.Millisecond
	defaultSuppression = 5 * time.Minute
)

// listenerReader is the narrowed surface the listener pipeline
// needs from the listener_events repo. Tests inject a fake.
type listenerReader interface {
	List(ctx context.Context, opts database.ListenerEventListOptions) ([]*database.ListenerEvent, error)
}

// alertWriter is the narrowed surface for writing alerts.
type alertWriter interface {
	Create(ctx context.Context, alert *database.Alert) error
}

// settingsKV is the high-water-mark store. Same shape as
// internal/topology — duplicate type rather than a circular import.
type settingsKV interface {
	GetWithDefault(ctx context.Context, key, defaultValue string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// Rule is one alert-emitting predicate. Match returns true when the
// event should fire an alert; Build constructs the alert payload.
// Both functions receive the same event so Build can pull whichever
// fields the title/message wants without re-decoding.
type Rule struct {
	// ID is a stable identifier used in the suppression fingerprint
	// so two events firing the same rule against the same source
	// don't spam (e.g. linkDown trap arriving every 30s).
	ID string

	// Match returns true when this rule applies to evt.
	Match func(evt *database.ListenerEvent) bool

	// Build returns the alert to write. The returned Source is used
	// in the suppression fingerprint alongside the rule ID.
	Build func(evt *database.ListenerEvent) *database.Alert
}

// ListenerPipeline scans listener_events on a tick and writes alerts
// for every event matching any built-in rule. Suppression dedupes
// repeated alerts for the same (rule, source) within the window.
type ListenerPipeline struct {
	events      listenerReader
	alerts      alertWriter
	settings    settingsKV
	logger      *slog.Logger
	now         func() time.Time
	interval    time.Duration
	rules       []Rule
	suppression time.Duration

	mu         sync.Mutex
	started    bool
	stopped    bool
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	emitted    map[string]time.Time
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

	// Rules override the built-in set; when nil, DefaultListenerRules
	// is used.
	Rules []Rule
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
	rules := cfg.Rules
	if rules == nil {
		rules = DefaultListenerRules()
	}
	return &ListenerPipeline{
		events:      cfg.Events,
		alerts:      cfg.Alerts,
		settings:    cfg.Settings,
		logger:      cfg.Logger,
		now:         cfg.Now,
		interval:    cfg.Interval,
		suppression: cfg.Suppression,
		rules:       rules,
		emitted:     make(map[string]time.Time),
	}, nil
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
	p.logger.InfoContext(ctx, "listener alert pipeline started",
		"interval", p.interval, "rules", len(p.rules))
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
	events, err := p.events.List(ctx, database.ListenerEventListOptions{
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

// evaluate applies every rule to evt and writes any alerts that
// match + pass suppression. Returns the count of alerts emitted.
func (p *ListenerPipeline) evaluate(ctx context.Context, evt *database.ListenerEvent) int {
	now := p.now()
	count := 0
	for _, rule := range p.rules {
		if !rule.Match(evt) {
			continue
		}
		fingerprint := fingerprintFor(rule.ID, evt.SourceAddr, evt.Kind)
		if p.suppressed(fingerprint, now) {
			continue
		}
		alert := rule.Build(evt)
		if alert == nil {
			continue
		}
		if writeErr := p.alerts.Create(ctx, alert); writeErr != nil {
			p.logger.WarnContext(ctx, "alert create failed",
				"rule", rule.ID, "source", evt.SourceAddr, "error", writeErr)
			continue
		}
		p.markEmitted(fingerprint, now)
		count++
	}
	return count
}

// fingerprintFor builds the suppression key.
func fingerprintFor(ruleID, source, kind string) string {
	h := sha256.New()
	h.Write([]byte(ruleID))
	h.Write([]byte{0x00})
	h.Write([]byte(source))
	h.Write([]byte{0x00})
	h.Write([]byte(kind))
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// suppressed returns true when this fingerprint has fired within the
// suppression window.
func (p *ListenerPipeline) suppressed(fingerprint string, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	last, ok := p.emitted[fingerprint]
	if !ok {
		return false
	}
	return now.Sub(last) < p.suppression
}

// markEmitted records the fire time for a fingerprint. Old entries
// (>2x suppression window) are evicted lazily to keep the map
// bounded.
func (p *ListenerPipeline) markEmitted(fingerprint string, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.emitted[fingerprint] = now
	// Lazy eviction: walk the map only when it gets noticeably big.
	const evictThreshold = 4096
	if len(p.emitted) <= evictThreshold {
		return
	}
	cutoff := now.Add(-2 * p.suppression)
	for k, v := range p.emitted {
		if v.Before(cutoff) {
			delete(p.emitted, k)
		}
	}
}

func (p *ListenerPipeline) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := p.settings.GetWithDefault(ctx, listenerHighWaterKey, "")
	if err != nil {
		return time.Time{}, err
	}
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse high-water %q: %w", raw, err)
	}
	return parsed, nil
}

func (p *ListenerPipeline) saveHighWater(ctx context.Context, t time.Time) error {
	return p.settings.Set(ctx, listenerHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}

// DefaultListenerRules returns the V1.0 built-in rule set:
//
//   - syslog severity emergency/alert/critical/error -> alert
//   - snmp trap linkDown -> alert (warning)
//   - snmp trap authenticationFailure -> alert (error)
//   - snmp trap any other event -> alert (info) when an explicit
//     trap OID is present
//
// The function returns a fresh slice each call so a pipeline can
// safely append or filter without affecting other callers.
func DefaultListenerRules() []Rule {
	return []Rule{
		ruleSyslogSevereLogged(),
		ruleTrapLinkDown(),
		ruleTrapAuthFailure(),
	}
}

func ruleSyslogSevereLogged() Rule {
	severeSeverities := map[string]bool{
		"emergency": true,
		"alert":     true,
		"critical":  true,
		"error":     true,
	}
	return Rule{
		ID: "syslog.severe",
		Match: func(evt *database.ListenerEvent) bool {
			return evt.Kind == "syslog-udp" && severeSeverities[evt.Severity]
		},
		Build: func(evt *database.ListenerEvent) *database.Alert {
			return &database.Alert{
				Type:     database.AlertTypeSystem,
				Severity: mapSyslogSeverity(evt.Severity),
				Title:    fmt.Sprintf("Syslog %s from %s", evt.Severity, evt.SourceAddr),
				Message:  summarize(evt.PayloadJSON, "message"),
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

func ruleTrapLinkDown() Rule {
	return Rule{
		ID: "trap.linkdown",
		Match: func(evt *database.ListenerEvent) bool {
			if evt.Kind != "snmp-trap-v2c" {
				return false
			}
			return strings.Contains(evt.PayloadJSON, `"1.3.6.1.6.3.1.1.5.3"`)
		},
		Build: func(evt *database.ListenerEvent) *database.Alert {
			return &database.Alert{
				Type:     database.AlertTypeConnectivity,
				Severity: database.AlertSeverityWarning,
				Title:    "Link down trap from " + evt.SourceAddr,
				Message:  "SNMP linkDown trap received",
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

func ruleTrapAuthFailure() Rule {
	return Rule{
		ID: "trap.authfail",
		Match: func(evt *database.ListenerEvent) bool {
			if evt.Kind != "snmp-trap-v2c" {
				return false
			}
			return strings.Contains(evt.PayloadJSON, `"1.3.6.1.6.3.1.1.5.5"`)
		},
		Build: func(evt *database.ListenerEvent) *database.Alert {
			return &database.Alert{
				Type:     database.AlertTypeSecurity,
				Severity: database.AlertSeverityError,
				Title:    "SNMP authentication failure from " + evt.SourceAddr,
				Message:  "Repeated authentication failures may indicate a credential probe",
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

// mapSyslogSeverity bridges syslog severity names to the alerts
// table's enum.
func mapSyslogSeverity(s string) string {
	switch s {
	case "emergency", "alert", "critical":
		return database.AlertSeverityCritical
	case "error":
		return database.AlertSeverityError
	case "warning":
		return database.AlertSeverityWarning
	default:
		return database.AlertSeverityInfo
	}
}

// summarize pulls one string field out of a JSON payload without
// fully unmarshalling. Returns "" when the field is absent or the
// payload isn't valid JSON. Used to grab e.g. the syslog "message"
// for an alert title without forcing a struct definition per kind.
func summarize(payload, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return ""
	}
	raw, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}
