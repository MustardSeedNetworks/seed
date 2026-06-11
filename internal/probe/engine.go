package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// defaultSubscriberBufferSize is the per-subscriber channel capacity.
// Subscribers that fall behind have events dropped (with a counter
// increment) rather than blocking the engine.
const defaultSubscriberBufferSize = 64

// ErrCheckerNotRegistered is returned when the engine is asked to
// dispatch a probe whose Kind has no registered Checker.
var ErrCheckerNotRegistered = errors.New("no checker registered for kind")

// Engine schedules and dispatches probes, evaluates thresholds, and
// emits ResultEvents to subscribers. Storage + scheduling fields are
// optional — wire them via WithStorage to enable Start/Stop/RunNow.
// Without storage the Engine still supports in-memory dispatch via
// RunDefinition.
type Engine struct {
	logger *slog.Logger

	mu       sync.RWMutex
	checkers map[string]Checker
	subs     []chan ResultEvent
	dropped  uint64 // total events dropped due to full subscriber buffers

	// Storage + scheduling (Stage A1.3b). Both nil = in-memory mode
	// (RunDefinition + subscribers work; Start/Stop/RunNow do not).
	storage   probeStorage
	scheduler probeScheduler

	// runMu guards lifecycle state.
	runMu   sync.Mutex
	started bool
	jobIDs  []string
}

// NewEngine returns a freshly-constructed Engine with no Checkers
// registered. Callers register Checkers via RegisterChecker before
// dispatching. Pass nil logger to use [slog.Default].
func NewEngine(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		logger:   logger,
		checkers: make(map[string]Checker),
	}
}

// RegisterChecker adds a Checker. Re-registering the same Kind
// replaces the previous Checker. Safe for concurrent calls.
func (e *Engine) RegisterChecker(c Checker) {
	if c == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkers[c.Kind()] = c
}

// Kinds returns the sorted list of registered Checker kinds. Used by
// UI/handler layers to populate the new-probe kind selector.
func (e *Engine) Kinds() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, 0, len(e.checkers))
	for k := range e.checkers {
		out = append(out, k)
	}
	// Sort for stable output; small N makes the in-place sort cheap.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// RunDefinition dispatches an ad-hoc probe to its registered Checker
// and returns the Result. The probe is NOT persisted; this is the
// primitive that AutoTest sequences and one-shot UI invocations
// build on. Thresholds in p.Warning / p.Critical are evaluated and
// the resulting ResultEvent is emitted to subscribers.
//
// If no Checker is registered for p.Kind, the returned Result has
// Success=false and Error=ErrCheckerNotRegistered.
func (e *Engine) RunDefinition(ctx context.Context, p Probe) Result {
	e.mu.RLock()
	checker, ok := e.checkers[p.Kind]
	e.mu.RUnlock()

	if !ok {
		r := Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error:     fmt.Sprintf("%v: %q", ErrCheckerNotRegistered, p.Kind),
		}
		e.emit(ResultEvent{Result: r})
		return r
	}

	r := checker.Run(ctx, p)
	// Honor the engine's clock for Timestamp when the Checker did
	// not set one. Checkers that capture per-step timing can still
	// set their own; we only fill in zero values.
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now().UTC()
	}
	if r.ProbeID == "" {
		r.ProbeID = p.ID
	}
	if r.ClientID == "" {
		r.ClientID = p.ClientID
	}
	if r.Kind == "" {
		r.Kind = p.Kind
	}

	breaches := evaluateThresholds(p, r)
	e.emit(ResultEvent{Result: r, Breaches: breaches})
	return r
}

// Subscribe returns a channel that receives every ResultEvent emitted
// by the engine. The channel is buffered; subscribers that fall
// behind have events dropped (the engine increments a counter and
// continues). Callers must drain the channel.
//
// There is no explicit Unsubscribe — subscribers go away when the
// engine is garbage collected. For long-lived deployments this is
// fine; A1.4 may add explicit Unsubscribe if alerts pipeline
// lifecycle requires it.
func (e *Engine) Subscribe() <-chan ResultEvent {
	ch := make(chan ResultEvent, defaultSubscriberBufferSize)
	e.mu.Lock()
	e.subs = append(e.subs, ch)
	e.mu.Unlock()
	return ch
}

// DroppedEvents returns the cumulative count of ResultEvents that
// were dropped because a subscriber's buffer was full. Useful for
// monitoring engine health and back-pressure issues.
func (e *Engine) DroppedEvents() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dropped
}

// emit fans a ResultEvent out to all current subscribers. Drops the
// event for any subscriber whose buffer is full and increments the
// dropped counter; never blocks.
func (e *Engine) emit(re ResultEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ch := range e.subs {
		select {
		case ch <- re:
		default:
			e.dropped++
			e.logger.Warn("probe engine dropped ResultEvent (subscriber buffer full)",
				"kind", re.Result.Kind,
				"probe_id", re.Result.ProbeID,
				"dropped_total", e.dropped,
			)
		}
	}
}

// evaluateThresholds compares a Result against the Warning and
// Critical threshold JSON on the Probe. Returns one Breach per
// violated rule. Two metrics are gated centrally today: latency_ms
// (breach when actual EXCEEDS the bound — higher is worse) and
// cert_days_remaining (breach when actual falls BELOW the bound —
// fewer days is worse). The cert metric is read from
// Result.Metadata.days_remaining, which only TLS-family checkers
// publish, so it auto-scopes to cert-bearing probes. Further
// kind-specific metrics (BGP state, etc.) add a field to
// genericThreshold and a gate here as their checkers expose them.
func evaluateThresholds(p Probe, r Result) []Breach {
	var breaches []Breach
	warn := parseGenericThreshold(p.Warning)
	crit := parseGenericThreshold(p.Critical)

	// latency_ms: breach when the measured value exceeds the bound.
	if warn != nil && warn.LatencyMs > 0 && r.LatencyMs > warn.LatencyMs {
		breaches = append(breaches, breachOf(p, r, "warning", "latency_ms", warn.LatencyMs, r.LatencyMs))
	}
	if crit != nil && crit.LatencyMs > 0 && r.LatencyMs > crit.LatencyMs {
		breaches = append(breaches, breachOf(p, r, "critical", "latency_ms", crit.LatencyMs, r.LatencyMs))
	}

	// cert_days_remaining: breach when the certificate's remaining days
	// fall below the bound (inverted from latency). Only evaluated when the
	// probe published days_remaining in its metadata — a non-TLS probe has
	// none and skips the gate.
	if days, ok := certDaysRemaining(r); ok {
		if warn != nil && warn.CertDaysRemaining > 0 && days < warn.CertDaysRemaining {
			breaches = append(breaches, breachOf(p, r, "warning", "cert_days_remaining", warn.CertDaysRemaining, days))
		}
		if crit != nil && crit.CertDaysRemaining > 0 && days < crit.CertDaysRemaining {
			breaches = append(breaches, breachOf(p, r, "critical", "cert_days_remaining", crit.CertDaysRemaining, days))
		}
	}

	// Success=false always breaches at critical when no other
	// threshold matched. Lets every probe at minimum surface "this
	// failed" to the alerts pipeline.
	if !r.Success && len(breaches) == 0 {
		breaches = append(breaches, breachOf(p, r, "critical", "success", true, false))
	}
	return breaches
}

// breachOf builds a Breach for a violated threshold, stamping the probe/client
// identity and result timestamp shared by every breach. Threshold and actual are
// any so each metric supplies its own value type (float latency, int days, bool
// success).
func breachOf(p Probe, r Result, severity, field string, threshold, actual any) Breach {
	return Breach{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Severity:  severity,
		Field:     field,
		Threshold: threshold,
		Actual:    actual,
		Timestamp: r.Timestamp,
	}
}

// genericThreshold is the threshold shape parsed from a probe's Warning/Critical
// JSON. Each field is one centrally-gated metric (see evaluateThresholds); a
// zero value means that metric is unconfigured for this probe.
type genericThreshold struct {
	LatencyMs         float64 `json:"latency_ms,omitempty"`
	CertDaysRemaining int     `json:"cert_days_remaining,omitempty"`
}

// certMetadata is the slice of a TLS-family probe's Result.Metadata the cert
// gate reads. The JSON field name is the contract with the TLS checker's
// TLSCertInfo (internal/probe/checkers); a pointer distinguishes "absent" (a
// non-TLS probe) from a real zero/negative days-remaining (an expired cert).
type certMetadata struct {
	DaysRemaining *int `json:"days_remaining"`
}

// certDaysRemaining extracts the certificate days-remaining a TLS-family probe
// publishes in Result.Metadata. The bool is false when the field is absent, so a
// probe that does not report a certificate skips the cert gate entirely.
func certDaysRemaining(r Result) (int, bool) {
	if len(r.Metadata) == 0 {
		return 0, false
	}
	var m certMetadata
	if err := json.Unmarshal(r.Metadata, &m); err != nil || m.DaysRemaining == nil {
		return 0, false
	}
	return *m.DaysRemaining, true
}

// parseGenericThreshold decodes a threshold JSON blob into the
// V1.0 shape. Returns nil on empty input or unparseable JSON
// (logged at debug elsewhere; a malformed threshold simply
// disables that side of the gate rather than failing the probe).
func parseGenericThreshold(raw json.RawMessage) *genericThreshold {
	if len(raw) == 0 {
		return nil
	}
	var t genericThreshold
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil
	}
	return &t
}
