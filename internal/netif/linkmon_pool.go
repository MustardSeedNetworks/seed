// SPDX-License-Identifier: BUSL-1.1

package netif

// LinkMonitorPool multiplexes per-interface LinkMonitor instances so the
// runtime can observe state changes across N interfaces concurrently —
// the multi_interface Pro feature (seed#1192, follow-up #1214).
//
// The pool is the new public entry point; LinkMonitor remains as the
// per-interface primitive. The pool keeps reconciliation cheap: monitors
// for unchanged interfaces survive a Reconcile call, only added/removed
// names trigger Start/Stop. Callbacks are fanned out from every
// underlying monitor through a single shared OnStateChange registration
// — callers see one event stream keyed by event.Interface.
//
// Concurrency: the pool's own mutex guards the map. Each child monitor
// has its own mutex; the pool never reaches inside them.

import (
	"fmt"
	"slices"
	"sort"
	"sync"
)

// LinkMonitorPool watches link state on a set of interfaces and emits a
// single, unified event stream. Use Reconcile to declaratively set the
// active interface set; the pool handles diff + start/stop internally.
type LinkMonitorPool struct {
	mu        sync.RWMutex
	monitors  map[string]*LinkMonitor
	callbacks []LinkStateCallback
	running   bool
}

// NewLinkMonitorPool returns an empty pool. Call Reconcile to populate
// it with interface names, then Start to begin polling.
func NewLinkMonitorPool() *LinkMonitorPool {
	return &LinkMonitorPool{
		monitors: make(map[string]*LinkMonitor),
	}
}

// Reconcile sets the monitored interface set to exactly `names`. Empty
// or whitespace-only names are dropped. Monitors for names that were
// already running are left untouched (no restart); monitors for newly
// added names are created and started if the pool itself is Running;
// monitors for removed names are stopped and dropped.
//
// Returns the post-reconcile interface count.
func (p *LinkMonitorPool) Reconcile(names []string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	wanted := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		wanted[n] = struct{}{}
	}

	// Stop + drop monitors for removed names.
	for name, mon := range p.monitors {
		if _, keep := wanted[name]; !keep {
			mon.Stop()
			delete(p.monitors, name)
		}
	}

	// Add new monitors. Wire the shared callback registry to each so a
	// state change on any interface reaches every registered callback
	// without the caller knowing about per-interface monitors.
	for name := range wanted {
		if _, exists := p.monitors[name]; exists {
			continue
		}
		mon := NewLinkMonitor(name)
		for _, cb := range p.callbacks {
			mon.OnStateChange(cb)
		}
		if p.running {
			// Best-effort start. A failed Start logs internally; we
			// don't propagate because pool semantics are "monitor what
			// you can." The Interfaces page in Settings already shows
			// per-interface state read separately.
			_ = mon.Start()
		}
		p.monitors[name] = mon
	}

	return len(p.monitors)
}

// OnStateChange registers cb against every current and future child
// monitor. Calling after Reconcile attaches cb to existing monitors
// retroactively so call order does not matter.
func (p *LinkMonitorPool) OnStateChange(cb LinkStateCallback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.callbacks = append(p.callbacks, cb)
	for _, mon := range p.monitors {
		mon.OnStateChange(cb)
	}
}

// Start polls all current monitors. Calling Start a second time is a
// no-op. Monitors added by a later Reconcile call inherit the running
// state automatically.
func (p *LinkMonitorPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return nil
	}
	p.running = true
	var firstErr error
	for name, mon := range p.monitors {
		if err := mon.Start(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("start %s: %w", name, err)
		}
	}
	return firstErr
}

// Stop halts polling on every monitor and clears the running flag so a
// subsequent Reconcile leaves new monitors idle. Existing callbacks are
// kept registered.
func (p *LinkMonitorPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	for _, mon := range p.monitors {
		mon.Stop()
	}
}

// GetState returns the last observed LinkState for the named interface,
// or LinkStateUnknown if the pool is not monitoring that interface.
func (p *LinkMonitorPool) GetState(name string) LinkState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if mon, ok := p.monitors[name]; ok {
		return mon.GetState()
	}
	return LinkStateUnknown
}

// Interfaces returns the de-duplicated, alphabetically sorted list of
// interface names currently in the pool. Useful for tests and admin
// surfaces that want a snapshot.
func (p *LinkMonitorPool) Interfaces() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.monitors))
	for name := range p.monitors {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Monitor returns the underlying per-interface LinkMonitor, or nil if
// the pool is not watching that name. Exposed so callers that need
// per-interface APIs (history, IsUp) can reach in without the pool
// re-exporting every method.
func (p *LinkMonitorPool) Monitor(name string) *LinkMonitor {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.monitors[name]
}

// Count returns the number of monitored interfaces.
func (p *LinkMonitorPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.monitors)
}

// Has reports whether the named interface is being monitored.
func (p *LinkMonitorPool) Has(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.monitors[name]
	return ok
}

// DiffNames returns the added + removed names for the transition from
// the pool's current set to `next`. Useful for callers that want to
// log or audit the reconcile delta before invoking Reconcile.
func (p *LinkMonitorPool) DiffNames(next []string) ([]string, []string) {
	current := p.Interfaces()
	wantedSet := make(map[string]struct{}, len(next))
	for _, n := range next {
		if n == "" {
			continue
		}
		wantedSet[n] = struct{}{}
	}
	var added, removed []string
	for name := range wantedSet {
		if !slices.Contains(current, name) {
			added = append(added, name)
		}
	}
	for _, name := range current {
		if _, keep := wantedSet[name]; !keep {
			removed = append(removed, name)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}
