package pipeline

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

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
	Match func(evt *listener.EventRecord) bool

	// Build returns the alert to write. The returned Source is used
	// in the suppression fingerprint alongside the rule ID.
	Build func(evt *listener.EventRecord) *alerts.Alert

	// Threshold is how many matching events must accrue inside
	// Window before this rule fires. Threshold=1 / Window=0 is the
	// pre-#1379 fire-on-first-match path.
	Threshold int

	// Window is the time window over which Threshold accrues.
	// Window=0 means "no window" (fire on first match).
	Window time.Duration

	// counter is the optional per-(rule, entity) rolling counter.
	// nil when Threshold <= 1.
	counter *windowCounter
}

// staticRulesView reads p.staticRules without taking the mutex — safe
// because it's set once at construction and never mutated.
func (p *ListenerPipeline) staticRulesView() bool { return p.staticRules || p.alertRules == nil }

// reloadLoop ticks every reloadInterval and pulls the latest enabled
// rules from alertRules into the pipeline's ruleset.
func (p *ListenerPipeline) reloadLoop(ctx context.Context) {
	defer p.wg.Done()
	t := time.NewTicker(p.reloadInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := p.ReloadRules(ctx); err != nil {
				p.logger.WarnContext(ctx, "alert_rules reload failed; keeping previous ruleset",
					"error", err)
			}
		}
	}
}

// ReloadRules pulls every enabled alert_rules row, compiles them into
// runtime rules, and swaps them in atomically. Falls back to
// DefaultListenerRules when the DB has no enabled rows. Returns the
// list error from the repo unchanged so callers can decide to log /
// retry. Safe to call concurrently with ScanOnce.
func (p *ListenerPipeline) ReloadRules(ctx context.Context) error {
	if p.alertRules == nil || p.staticRules {
		return nil
	}
	rows, err := p.alertRules.List(ctx, true)
	if err != nil {
		return err
	}
	compiled := CompileRulesFromDB(rows)
	var next []Rule
	var usingDefaults bool
	if len(compiled) == 0 {
		next = p.defaults
		usingDefaults = true
	} else {
		next = compiled
	}
	p.mu.Lock()
	p.rules = next
	p.mu.Unlock()
	p.logger.DebugContext(ctx, "alert_rules reloaded",
		"db_rows", len(rows), "compiled", len(compiled),
		"active_rules", len(next), "defaults_fallback", usingDefaults)
	return nil
}

// snapshotRules returns the current ruleset under p.mu so concurrent
// reloads can't observe a torn iteration. The slice is shared, not
// cloned, because Rule entries are immutable after compilation.
func (p *ListenerPipeline) snapshotRules() []Rule {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rules
}
