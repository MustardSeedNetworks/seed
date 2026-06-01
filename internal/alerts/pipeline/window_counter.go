package pipeline

// window_counter implements the time-windowed alert-rule threshold
// described in #1379. Each (rule, entity) pair gets a rolling counter
// that increments per match. When the counter reaches threshold
// inside window_seconds, the rule fires once and the counter resets.
// Old hits outside the window are pruned lazily on each Hit call.
//
// V1.0 keeps the counter in-memory. A restart clears all counters,
// which means a rule mid-accrual ("2 of 3") loses its progress.
// That's the same restart-safety tradeoff every in-memory subsystem
// has and is acceptable for V1.0; persistent counters land in a
// follow-up if customer demand surfaces.

import (
	"sync"
	"time"
)

// windowCounter is the per-rule state holder. One counter per Rule;
// the rule's Match closure decides whether to call Hit. windowCounter
// is safe for concurrent use.
type windowCounter struct {
	window    time.Duration
	threshold int

	mu     sync.Mutex
	events map[string][]time.Time
}

// newWindowCounter constructs a counter with the given window +
// threshold. Both must be positive (validated by the caller).
func newWindowCounter(window time.Duration, threshold int) *windowCounter {
	return &windowCounter{
		window:    window,
		threshold: threshold,
		events:    make(map[string][]time.Time),
	}
}

// Hit records one matching event for entityKey at now. Returns true
// when this hit caused the counter to cross the threshold; the
// counter then resets so a subsequent steady stream doesn't keep
// firing on every hit past N.
func (c *windowCounter) Hit(entityKey string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	cutoff := now.Add(-c.window)
	existing := c.events[entityKey]
	hits := make([]time.Time, 0, len(existing)+1)
	for _, t := range existing {
		if t.After(cutoff) {
			hits = append(hits, t)
		}
	}
	hits = append(hits, now)
	fresh := hits
	if len(fresh) >= c.threshold {
		// Threshold met — reset so the next stream of hits starts
		// fresh rather than rapid-firing.
		delete(c.events, entityKey)
		return true
	}
	c.events[entityKey] = fresh
	return false
}
