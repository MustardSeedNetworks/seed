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
// violated rule. V1.0 supports latency_ms thresholds; kind-specific
// thresholds (cert days_remaining, BGP state, etc.) are added per-
// checker in A1.4+ via custom JSON shapes the Checker reads
// directly from p.Critical / p.Warning before returning Result.
func evaluateThresholds(p Probe, r Result) []Breach {
	var breaches []Breach
	if warn := parseGenericThreshold(p.Warning); warn != nil {
		if warn.LatencyMs > 0 && r.LatencyMs > warn.LatencyMs {
			breaches = append(breaches, Breach{
				ProbeID:   p.ID,
				ClientID:  p.ClientID,
				Severity:  "warning",
				Field:     "latency_ms",
				Threshold: warn.LatencyMs,
				Actual:    r.LatencyMs,
				Timestamp: r.Timestamp,
			})
		}
	}
	if crit := parseGenericThreshold(p.Critical); crit != nil {
		if crit.LatencyMs > 0 && r.LatencyMs > crit.LatencyMs {
			breaches = append(breaches, Breach{
				ProbeID:   p.ID,
				ClientID:  p.ClientID,
				Severity:  "critical",
				Field:     "latency_ms",
				Threshold: crit.LatencyMs,
				Actual:    r.LatencyMs,
				Timestamp: r.Timestamp,
			})
		}
	}
	// Success=false always breaches at critical when no other
	// threshold matched. Lets every probe at minimum surface "this
	// failed" to the alerts pipeline.
	if !r.Success && len(breaches) == 0 {
		breaches = append(breaches, Breach{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Severity:  "critical",
			Field:     "success",
			Threshold: true,
			Actual:    false,
			Timestamp: r.Timestamp,
		})
	}
	return breaches
}

// genericThreshold is the V1.0 base threshold shape. Checkers that
// need kind-specific fields (e.g. cert_days_remaining for TLS)
// embed this and add their own.
type genericThreshold struct {
	LatencyMs float64 `json:"latency_ms,omitempty"`
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
