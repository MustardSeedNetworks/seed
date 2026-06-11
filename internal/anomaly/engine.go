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

// observeResult reports the effect of one Observe on the live stream so the
// persistence Coordinator (store.go) can tell a material change — written
// through immediately — from mere recurrence, which is batched into the next
// Flush. Package-internal: external callers use Observe and ignore it.
type observeResult struct {
	key      instanceKey
	created  bool // a new instance was inserted
	material bool // created, OR base severity changed, OR escalation threshold crossed
}

// observe folds one detection into the stream and reports what changed. A new
// (type, subject) pair creates an instance; a repeat updates lastSeen and the
// recurrence count (which can escalate severity). It errors if the detection's
// DefKey is not in the catalog or its severity override is invalid — a rule
// source must only emit defined anomalies.
func (e *Engine) observe(d Detection, at time.Time) (observeResult, error) {
	if _, ok := e.catalog.Lookup(d.DefKey); !ok {
		return observeResult{}, fmt.Errorf("anomaly: detection references unknown def %q", d.DefKey)
	}
	if d.Severity != "" && !d.Severity.valid() {
		return observeResult{}, fmt.Errorf(
			"anomaly: detection for %q has invalid severity %q", d.DefKey, d.Severity)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	k := d.key()
	newBase := e.baseSeverity(d)
	t, ok := e.active[k]
	if !ok {
		e.active[k] = &tracked{
			det:          d,
			baseSeverity: newBase,
			firstSeen:    at,
			lastSeen:     at,
			count:        1,
		}
		return observeResult{key: k, created: true, material: true}, nil
	}
	// Coalesce: refresh evidence + lifecycle, keep the earliest firstSeen. A
	// material change — a base-severity change or crossing the escalation
	// threshold — warrants a durable write; a plain recurrence does not.
	beforeEff := e.effectiveSeverityOf(t.baseSeverity, t.count)
	baseChanged := t.baseSeverity != newBase
	t.det = d
	t.baseSeverity = newBase
	t.count++
	if at.After(t.lastSeen) {
		t.lastSeen = at
	}
	afterEff := e.effectiveSeverityOf(t.baseSeverity, t.count)
	return observeResult{key: k, material: baseChanged || afterEff != beforeEff}, nil
}

// Observe folds one detection into the stream at observation time `at`. A new
// (type, subject) pair creates an instance; a repeat updates lastSeen and the
// recurrence count (which can escalate severity). It errors if the detection's
// DefKey is not in the catalog or its severity override is invalid — a rule
// source must only emit defined anomalies.
func (e *Engine) Observe(d Detection, at time.Time) error {
	_, err := e.observe(d, at)
	return err
}

// Restore seeds the live set from persisted records (ADR-0021 load-on-start), so
// a restart continues coalescing onto the same instances instead of re-detecting
// them as new. Each record's persisted (effective) severity becomes the restored
// base severity (ADR-0021: the store holds the effective value); a live
// re-detection then overrides it from the catalog default on the next Observe.
// Records whose DefKey is no longer in the catalog are skipped — a catalog change
// can orphan an old row. Intended to be called once, before Observe traffic
// begins. Returns the number of instances restored.
func (e *Engine) Restore(records []Record) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for _, r := range records {
		a := r.Anomaly
		if _, ok := e.catalog.Lookup(a.DefKey); !ok {
			continue
		}
		k := instanceKey{def: a.DefKey, kind: a.Subject.Kind, id: a.Subject.ID}
		e.active[k] = &tracked{
			det: Detection{
				DefKey:   a.DefKey,
				Subject:  a.Subject,
				Severity: a.Severity,
				Evidence: a.Evidence,
			},
			baseSeverity: a.Severity,
			firstSeen:    a.FirstSeen,
			lastSeen:     a.LastSeen,
			count:        a.Count,
		}
		n++
	}
	return n
}

// baseSeverity is the detection's explicit severity, else the catalog default.
func (e *Engine) baseSeverity(d Detection) Severity {
	if d.Severity != "" {
		return d.Severity
	}
	def, _ := e.catalog.Lookup(d.DefKey)
	return def.DefaultSeverity
}

// effectiveSeverity applies recurrence escalation to a tracked instance.
func (e *Engine) effectiveSeverity(t *tracked) Severity {
	return e.effectiveSeverityOf(t.baseSeverity, t.count)
}

// effectiveSeverityOf bumps base one level once count reaches escalateAfter,
// capped at critical (escalateAfter <= 0 disables escalation).
func (e *Engine) effectiveSeverityOf(base Severity, count int) Severity {
	if e.escalateAfter <= 0 || count < e.escalateAfter {
		return base
	}
	switch base {
	case SeverityInfo:
		return SeverityWarning
	case SeverityWarning:
		return SeverityCritical
	case SeverityCritical:
		return SeverityCritical
	default:
		return base
	}
}

// Prune clears instances not re-observed since cutoff (the anomaly's condition
// is considered resolved). Returns the number cleared.
func (e *Engine) Prune(cutoff time.Time) int {
	return len(e.pruneKeys(cutoff))
}

// pruneKeys clears instances not re-observed since cutoff and returns their
// keys, so the persistence Coordinator can mark exactly those resolved.
func (e *Engine) pruneKeys(cutoff time.Time) []instanceKey {
	e.mu.Lock()
	defer e.mu.Unlock()
	var cleared []instanceKey
	for k, t := range e.active {
		if t.lastSeen.Before(cutoff) {
			cleared = append(cleared, k)
			delete(e.active, k)
		}
	}
	return cleared
}

// project builds the Anomaly view of one tracked instance, merging catalog copy
// with instance evidence/lifecycle and degrading capability-gated auto
// follow-ups to prompts. The caller holds e.mu.
func (e *Engine) project(t *tracked) Anomaly {
	def, _ := e.catalog.Lookup(t.det.DefKey)
	return Anomaly{
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
	}
}

// snapshotKeys projects the named live instances, skipping any no longer
// present. Used by the persistence Coordinator to build write-through and Flush
// records without re-projecting the whole stream.
func (e *Engine) snapshotKeys(keys []instanceKey) []Anomaly {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Anomaly, 0, len(keys))
	for _, k := range keys {
		if t, ok := e.active[k]; ok {
			out = append(out, e.project(t))
		}
	}
	return out
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
		out = append(out, e.project(t))
	}
	SortAnomalies(out)
	return out
}

// SortAnomalies orders anomalies canonically in place: most-urgent severity
// first, then defKey, then subject id. It is the single stable ordering shared by
// the engine's live Snapshot and by store-backed reads (ADR-0021 §4), so the API
// presents the same order whether anomalies come from memory or SQL.
func SortAnomalies(a []Anomaly) {
	sort.Slice(a, func(i, j int) bool {
		if ri, rj := a[i].Severity.rank(), a[j].Severity.rank(); ri != rj {
			return ri > rj // most urgent first
		}
		if a[i].DefKey != a[j].DefKey {
			return a[i].DefKey < a[j].DefKey
		}
		return a[i].Subject.ID < a[j].Subject.ID
	})
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
