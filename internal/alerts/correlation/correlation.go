// Package correlation names the alert that probably caused another one.
//
// Seed raises one alert per condition it observes, which is right for the
// record and wrong for the operator: when a BGP session drops because the
// interface carrying it went down, two alerts arrive seconds apart and the
// second one explains nothing about the first. This package annotates the
// later alert with the id of the earlier one.
//
// It annotates; it does not aggregate. The inbox stays the complete record of
// what happened and every alert is still stored and still delivered — the
// principle delivery/writer.go already states, that a receiver must not learn
// about an alert an operator cannot then find. Suppressing symptoms in favour
// of one synthetic parent would break exactly that, and would need the
// pipeline to hold alerts back while it waited to see what else arrived.
//
// Not to be confused with pipeline/suppression.go, which answers a different
// question: whether this rule already fired for this entity recently. That is
// de-duplication of one condition over time; this is the relationship between
// two different conditions at the same moment.
package correlation

import (
	"context"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// Writer is the alert-store surface this decorator wraps — the same narrow
// interface the pipelines and the delivery decorator already speak.
type Writer interface {
	Create(ctx context.Context, alert *alerts.Alert) error
}

// DefaultWindow is how long an alert stays eligible to explain a later one.
// A BGP hold timer is 90 s by default, so a session drop caused by a link
// failure is observable within roughly two SNMP poll intervals of it; five
// minutes covers that with room for a slow poller without stretching so far
// that two unrelated faults on one device look like one story.
const DefaultWindow = 5 * time.Minute

// causeRuleFor returns the rule whose alert, on the same device and inside the
// window, is the probable cause of an alert raised by rule.
//
// One pair, deliberately. A correlation that is merely plausible is worse than
// none: it puts a confident wrong sentence in front of an engineer who is
// mid-incident. This pair is mechanical — a BGP peer leaves Established
// because the interface carrying the session went down — and both alerts come
// from real SNMP observations of the same target today.
//
// Deliberately absent: storage.high -> storage.critical is the same condition
// escalating, not one condition causing another, and suppression already owns
// it; operator-defined listener rules are open-ended, so nothing general can
// be asserted about what causes them.
func causeRuleFor(rule string) (string, bool) {
	switch rule {
	case ruleBGPFlap:
		return ruleInterfaceDown, true
	default:
		return "", false
	}
}

// isCause reports whether alerts from this rule can explain a later alert, and
// so are worth remembering. It is the range of causeRuleFor.
func isCause(rule string) bool {
	return rule == ruleInterfaceDown
}

// The pipeline rule ids this package reasons about, spelled once.
const (
	ruleBGPFlap       = "bgp.flap"
	ruleInterfaceDown = "iface.down"
)

// Config configures the decorator. The zero value is usable: Window falls back
// to [DefaultWindow] and Now to [time.Now].
type Config struct {
	// Window is how long a stored alert may explain a later one.
	Window time.Duration
	// Now is the clock, injectable so tests own it.
	Now func() time.Time
}

// WrapWriter returns a Writer that annotates each alert with its probable
// cause before storing it, so the annotation is on disk and in the payload the
// delivery decorator sends. Wire it inside delivery:
//
//	delivery.WrapWriter(correlation.WrapWriter(repo, cfg), notifier)
func WrapWriter(store Writer, cfg Config) Writer {
	if cfg.Window <= 0 {
		cfg.Window = DefaultWindow
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &correlatingWriter{
		store:  store,
		window: cfg.Window,
		now:    cfg.Now,
		recent: make(map[string]stored),
	}
}

// stored is what the decorator remembers about an alert that was written: its
// id, so a later alert can point at it, and when it was written, so the window
// can be judged.
type stored struct {
	id int64
	at time.Time
}

type correlatingWriter struct {
	store  Writer
	window time.Duration
	now    func() time.Time

	// recent holds only alerts whose rule appears as some rule's cause, keyed
	// by rule and device. It is process-local, so a restart forgets the window
	// in flight and the first alert after it names no cause — the same
	// trade-off inMemorySuppressionStore makes, and acceptable for the same
	// reason: the cost is one un-annotated alert, not a missed one.
	mu     sync.Mutex
	recent map[string]stored
}

func (w *correlatingWriter) Create(ctx context.Context, alert *alerts.Alert) error {
	if id, ok := w.causeOf(alert); ok {
		alert.RootCauseID = &id
	}
	if err := w.store.Create(ctx, alert); err != nil {
		return err
	}
	w.remember(alert)
	return nil
}

// causeOf returns the id of the alert that probably caused this one.
func (w *correlatingWriter) causeOf(alert *alerts.Alert) (int64, bool) {
	causeRule, ok := causeRuleFor(alert.Rule)
	if !ok {
		return 0, false
	}
	now := w.now()

	w.mu.Lock()
	defer w.mu.Unlock()
	cause, ok := w.recent[key(causeRule, alert.Source)]
	if !ok || now.Sub(cause.at) > w.window {
		return 0, false
	}
	return cause.id, true
}

// remember records an alert that was actually stored, so it can explain a
// later one. Only alerts some rule names as a cause are kept, which bounds the
// map by the number of distinct (cause rule, device) pairs rather than by the
// alert rate.
func (w *correlatingWriter) remember(alert *alerts.Alert) {
	if !isCause(alert.Rule) {
		return
	}
	now := w.now()

	w.mu.Lock()
	defer w.mu.Unlock()
	w.recent[key(alert.Rule, alert.Source)] = stored{id: alert.ID, at: now}
	w.evictExpired(now)
}

// evictExpired drops entries that can no longer explain anything. Called under
// w.mu on every remembered alert; the map holds one entry per (cause rule,
// device), so the scan is over devices, not over alerts.
func (w *correlatingWriter) evictExpired(now time.Time) {
	for k, s := range w.recent {
		if now.Sub(s.at) > w.window {
			delete(w.recent, k)
		}
	}
}

// key namespaces the memory by rule and device. \x00 cannot appear in either,
// so no pair of values can collide onto one key.
func key(rule, source string) string {
	return rule + "\x00" + source
}
