package anomaly

import (
	"context"
	"sync"
	"time"
)

// Coordinator couples the in-memory Engine to a persistent Store (ADR-0021).
// Material state changes — a newly detected anomaly or a severity escalation —
// are written through to the Store immediately, so durable events survive a
// restart. High-frequency recurrence (lastSeen/count) is NOT written per
// observation; it is accumulated and persisted by Flush on a periodic tick, so a
// scan burst costs one batched write, not one per detection. Resolution (Prune)
// is written through. The pure Engine is unchanged and remains usable
// standalone; the Coordinator is the long-lived, persisted instance.
type Coordinator struct {
	engine *Engine
	store  Store

	mu    sync.Mutex
	dirty map[instanceKey]struct{}
}

// NewCoordinator wraps engine with write-through persistence to store. Each
// persisted instance is tagged with the Source carried on its detection
// (ADR-0029), so one shared Coordinator serves every producer.
func NewCoordinator(engine *Engine, store Store) *Coordinator {
	return &Coordinator{
		engine: engine,
		store:  store,
		dirty:  make(map[instanceKey]struct{}),
	}
}

// Observe folds a detection into the engine and persists per the write cadence:
// write-through on a material change, deferred (marked for the next Flush) on
// mere recurrence. A Store error after a successful in-memory update is returned
// but does not roll back the engine — the next Flush re-persists the instance.
func (c *Coordinator) Observe(ctx context.Context, d Detection, at time.Time) error {
	res, err := c.engine.observe(d, at)
	if err != nil {
		return err
	}
	if res.material {
		return c.store.Upsert(ctx, c.engine.recordsForKeys([]instanceKey{res.key}))
	}
	c.mu.Lock()
	c.dirty[res.key] = struct{}{}
	c.mu.Unlock()
	return nil
}

// Flush persists every instance touched by recurrence since the last Flush — the
// batched write path for the high-frequency lastSeen/count updates. It is a
// no-op when nothing is pending.
func (c *Coordinator) Flush(ctx context.Context) error {
	c.mu.Lock()
	keys := make([]instanceKey, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}
	c.dirty = make(map[instanceKey]struct{})
	c.mu.Unlock()

	recs := c.engine.recordsForKeys(keys)
	if len(recs) == 0 {
		return nil
	}
	return c.store.Upsert(ctx, recs)
}

// Prune clears the given source's instances idle since cutoff and marks exactly
// those resolved in the store as of cutoff (the idle boundary). Returns the
// number cleared. This is the TTL-on-silence resolution path; ResolveSubject is
// the explicit fast-path. Source-scoped because resolution windows differ per
// producer (ADR-0029 §3) — a shared engine must not resolve one source's
// instances on another source's cadence.
func (c *Coordinator) Prune(ctx context.Context, source Source, cutoff time.Time) (int, error) {
	keys := c.engine.pruneKeysForSource(source, cutoff)
	return len(keys), c.resolveKeys(ctx, keys, cutoff)
}

// ResolveSubject clears every live instance for subject (across all defs) and
// marks exactly those resolved in the store as of `at`. It is the explicit
// recovery fast-path (ADR-0025 §3): a source that observes a subject return to
// health resolves its anomalies at once, instead of waiting for Prune's
// TTL-on-silence. Returns the number cleared; a subject with no active instances
// is a no-op.
func (c *Coordinator) ResolveSubject(ctx context.Context, subject SubjectRef, at time.Time) (int, error) {
	keys := c.engine.clearSubject(subject)
	return len(keys), c.resolveKeys(ctx, keys, at)
}

// resolveKeys marks already-cleared instance keys resolved in the store as of
// `at` and drops any pending dirty marks, so a later Flush cannot resurrect a
// resolved row. Shared by Prune (idle boundary) and ResolveSubject (explicit
// recovery). A no-op when keys is empty.
func (c *Coordinator) resolveKeys(ctx context.Context, keys []instanceKey, at time.Time) error {
	if len(keys) == 0 {
		return nil
	}
	ids := make([]string, len(keys))
	for i, k := range keys {
		ids[i] = k.recordID()
	}
	c.mu.Lock()
	for _, k := range keys {
		delete(c.dirty, k)
	}
	c.mu.Unlock()
	return c.store.MarkResolved(ctx, ids, at)
}

// Load seeds the engine from the store's active (unresolved) instances (ADR-0021
// load-on-start). Call once before observation begins; returns the number
// restored. A store error is returned without mutating the engine.
func (c *Coordinator) Load(ctx context.Context) (int, error) {
	recs, err := c.store.LoadActive(ctx)
	if err != nil {
		return 0, err
	}
	return c.engine.Restore(recs), nil
}

// Engine returns the underlying in-memory engine for read-only projection
// (Snapshot) by consumers that also read live state.
func (c *Coordinator) Engine() *Engine { return c.engine }
