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
	source Source

	mu    sync.Mutex
	dirty map[instanceKey]struct{}
}

// NewCoordinator wraps engine with write-through persistence to store, tagging
// every persisted instance with source.
func NewCoordinator(engine *Engine, store Store, source Source) *Coordinator {
	return &Coordinator{
		engine: engine,
		store:  store,
		source: source,
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
		return c.store.Upsert(ctx, c.records([]instanceKey{res.key}))
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

	recs := c.records(keys)
	if len(recs) == 0 {
		return nil
	}
	return c.store.Upsert(ctx, recs)
}

// Prune clears instances idle since cutoff and marks exactly those resolved in
// the store as of cutoff (the idle boundary). Returns the number cleared.
func (c *Coordinator) Prune(ctx context.Context, cutoff time.Time) (int, error) {
	keys := c.engine.pruneKeys(cutoff)
	if len(keys) == 0 {
		return 0, nil
	}
	ids := make([]string, len(keys))
	for i, k := range keys {
		ids[i] = k.recordID()
	}
	// The pruned instances are gone from the engine; drop any pending dirty
	// marks so a later Flush does not resurrect a resolved row.
	c.mu.Lock()
	for _, k := range keys {
		delete(c.dirty, k)
	}
	c.mu.Unlock()

	if err := c.store.MarkResolved(ctx, ids, cutoff); err != nil {
		return len(keys), err
	}
	return len(keys), nil
}

// Engine returns the underlying in-memory engine for read-only projection
// (Snapshot) by consumers that also read live state.
func (c *Coordinator) Engine() *Engine { return c.engine }

// records projects keys into persistable Records, tagging source and the stable
// id and skipping keys no longer live.
func (c *Coordinator) records(keys []instanceKey) []Record {
	anoms := c.engine.snapshotKeys(keys)
	out := make([]Record, 0, len(anoms))
	for _, a := range anoms {
		out = append(out, Record{
			ID:      RecordID(a.DefKey, a.Subject),
			Source:  c.source,
			Anomaly: a,
		})
	}
	return out
}
