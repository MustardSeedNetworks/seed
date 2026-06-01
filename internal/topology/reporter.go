package topology

// reporter.go provides the shared status accounting the four
// reconcilers (sys_info, if_table, edge, arp) layer on top of their
// Engine implementation to satisfy engine.Reporter (#1383).
//
// Each reconciler embeds *tickTracker. The tracker maintains
// lastTickAt + lastError under its own mutex and surfaces them via
// status(stopped). Engines call tracker.recordTick(err) at the end
// of every ReconcileOnce pass.
//
// Kept package-private — the engine.Reporter interface is the
// public seam. Operators see the rolled-up Status in /api/v1/engines,
// not the raw tracker.

import (
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/engine"
)

// degradedTickMultiplier mirrors the alerts/pipeline constant: a
// reconciler that hasn't ticked in 2x its configured interval is
// reported as degraded.
const degradedTickMultiplier = 2

type tickTracker struct {
	interval time.Duration
	now      func() time.Time

	mu         sync.Mutex
	lastTickAt time.Time
	lastError  string
}

func newTickTracker(interval time.Duration, now func() time.Time) *tickTracker {
	return &tickTracker{interval: interval, now: now}
}

func (t *tickTracker) recordTick(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastTickAt = t.now()
	if err != nil {
		t.lastError = err.Error()
		return
	}
	t.lastError = ""
}

func (t *tickTracker) status(stopped bool) engine.Status {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := engine.Status{LastTickAt: t.lastTickAt, LastError: t.lastError}
	switch {
	case stopped:
		s.State = engine.StateStopped
	case t.lastTickAt.IsZero():
		s.State = engine.StateOK
	case t.now().Sub(t.lastTickAt) > degradedTickMultiplier*t.interval:
		s.State = engine.StateDegraded
	default:
		s.State = engine.StateOK
	}
	return s
}
