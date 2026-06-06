package anomaly

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// defaultEscalateAfter is the recurrence count at which a persistent detection
// is escalated one severity level (info→warning→critical). A repeatedly
// re-observed problem is more important than a one-off blip.
const defaultEscalateAfter = 5

// tracked is the live state of one coalesced detection instance.
type tracked struct {
	det          Detection
	baseSeverity Severity
	firstSeen    time.Time
	lastSeen     time.Time
	count        int
}

// Engine coalesces detections from any source into one deduped, escalating,
// ageable stream. It is source-neutral: it knows only the catalog and the
// detections fed to it. Safe for concurrent Observe/Snapshot/Prune.
type Engine struct {
	mu            sync.RWMutex
	catalog       *Catalog
	active        map[instanceKey]*tracked
	capabilities  map[string]struct{}
	escalateAfter int
}

// Option configures an Engine.
type Option func(*Engine)

// WithCapabilities registers the platform capabilities available for auto
// follow-ups. An "auto" follow-up whose Capability is not registered is
// degraded to a guided prompt in Snapshot (ADR-0011).
func WithCapabilities(caps ...string) Option {
	return func(e *Engine) {
		for _, c := range caps {
			e.capabilities[c] = struct{}{}
		}
	}
}

// WithEscalateAfter sets the recurrence count that escalates severity one level.
// Zero disables escalation.
func WithEscalateAfter(n int) Option {
	return func(e *Engine) { e.escalateAfter = n }
}

// NewEngine returns an engine backed by catalog.
func NewEngine(catalog *Catalog, opts ...Option) *Engine {
	e := &Engine{
		catalog:       catalog,
		active:        make(map[instanceKey]*tracked),
		capabilities:  make(map[string]struct{}),
		escalateAfter: defaultEscalateAfter,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Observe folds one detection into the stream at observation time `at`. A new
// (type, subject) pair creates an instance; a repeat updates lastSeen and the
// recurrence count (which can escalate severity). It errors if the detection's
// DefKey is not in the catalog or its severity override is invalid — a rule
// source must only emit defined anomalies.
func (e *Engine) Observe(d Detection, at time.Time) error {
	if _, ok := e.catalog.Lookup(d.DefKey); !ok {
		return fmt.Errorf("anomaly: detection references unknown def %q", d.DefKey)
	}
	if d.Severity != "" && !d.Severity.valid() {
		return fmt.Errorf("anomaly: detection for %q has invalid severity %q", d.DefKey, d.Severity)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	k := d.key()
	t, ok := e.active[k]
	if !ok {
		e.active[k] = &tracked{
			det:          d,
			baseSeverity: e.baseSeverity(d),
			firstSeen:    at,
			lastSeen:     at,
			count:        1,
		}
		return nil
	}
	// Coalesce: refresh evidence + lifecycle, keep the earliest firstSeen.
	t.det = d
	t.baseSeverity = e.baseSeverity(d)
	t.count++
	if at.After(t.lastSeen) {
		t.lastSeen = at
	}
	return nil
}

// baseSeverity is the detection's explicit severity, else the catalog default.
func (e *Engine) baseSeverity(d Detection) Severity {
	if d.Severity != "" {
		return d.Severity
	}
	def, _ := e.catalog.Lookup(d.DefKey)
	return def.DefaultSeverity
}

// effectiveSeverity applies recurrence escalation: a base severity bumps one
// level once count reaches escalateAfter, capped at critical.
func (e *Engine) effectiveSeverity(t *tracked) Severity {
	if e.escalateAfter <= 0 || t.count < e.escalateAfter {
		return t.baseSeverity
	}
	switch t.baseSeverity {
	case SeverityInfo:
		return SeverityWarning
	case SeverityWarning:
		return SeverityCritical
	case SeverityCritical:
		return SeverityCritical
	default:
		return t.baseSeverity
	}
}

// Prune clears instances not re-observed since cutoff (the anomaly's condition
// is considered resolved). Returns the number cleared.
func (e *Engine) Prune(cutoff time.Time) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for k, t := range e.active {
		if t.lastSeen.Before(cutoff) {
			delete(e.active, k)
			n++
		}
	}
	return n
}

// Snapshot projects the live stream as deterministically-ordered Anomaly views,
// merging catalog copy with instance evidence/lifecycle and degrading
// capability-gated auto follow-ups to prompts. Ordering: severity (most urgent
// first), then defKey, then subject — stable for tests and UI.
func (e *Engine) Snapshot() []Anomaly {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]Anomaly, 0, len(e.active))
	for _, t := range e.active {
		def, _ := e.catalog.Lookup(t.det.DefKey)
		out = append(out, Anomaly{
			DefKey:         def.ID,
			Category:       def.Category,
			Severity:       e.effectiveSeverity(t),
			Subject:        t.det.Subject,
			Title:          def.Title,
			Description:    def.Description,
			Impact:         def.Impact,
			Recommendation: def.Recommendation,
			Standards:      def.Standards,
			Evidence:       t.det.Evidence,
			FollowUps:      e.projectFollowUps(def.FollowUps),
			FirstSeen:      t.firstSeen,
			LastSeen:       t.lastSeen,
			Count:          t.count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if ri, rj := out[i].Severity.rank(), out[j].Severity.rank(); ri != rj {
			return ri > rj // most urgent first
		}
		if out[i].DefKey != out[j].DefKey {
			return out[i].DefKey < out[j].DefKey
		}
		return out[i].Subject.ID < out[j].Subject.ID
	})
	return out
}

// projectFollowUps degrades an "auto" follow-up to a prompt when its required
// capability is not registered, so the guidance is always actionable.
func (e *Engine) projectFollowUps(fus []FollowUp) []FollowUp {
	if len(fus) == 0 {
		return nil
	}
	out := make([]FollowUp, 0, len(fus))
	for _, f := range fus {
		if f.Kind == FollowUpAuto && f.Capability != "" {
			if _, ok := e.capabilities[f.Capability]; !ok {
				f.Kind = FollowUpPrompt
			}
		}
		out = append(out, f)
	}
	return out
}

// Len is the number of live anomaly instances.
func (e *Engine) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.active)
}
