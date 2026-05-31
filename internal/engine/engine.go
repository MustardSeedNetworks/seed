// Package engine defines the minimal lifecycle contract every
// long-running seed subsystem implements (probe engine, retention
// engine, snmp poller, listeners, discovery). The interface is
// deliberately small — Name + Start + Stop — because the behaviors
// differ too much under the hood to force them through a richer
// shared API.
//
// The payoff is in lifecycle, not behavior:
//
//   - server.go registers every subsystem with a Registry and
//     starts them in one loop, stops them in reverse order on
//     shutdown.
//   - Per-tier gating reads as Registry.Enable("snmp-poller") for
//     Pro, off for Free, instead of bespoke conditionals at every
//     init site.
//   - Structured logging context (engine.name) is injected once
//     in Registry.Start instead of by every engine's lifecycle.
//   - A future GET /api/engines admin endpoint enumerates the
//     Registry rather than chasing fields on the server struct.
package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

// Engine is the lifecycle contract. Engines self-manage their
// internal scheduler, subscribers, repositories — Registry only
// cares about identity + start/stop.
type Engine interface {
	// Name is the stable identifier for logs, metrics, and the
	// /api/engines admin surface. Use kebab-case; one word
	// preferred (probe, retention, snmp-poller).
	Name() string

	// Start brings the engine up. Must be idempotent — registering
	// twice and starting twice cannot leave duplicate goroutines.
	// Returning an error aborts the rest of the registry's start
	// sequence; callers should treat partial-start as failure and
	// invoke Stop to clean up engines that did come up.
	Start(ctx context.Context) error

	// Stop tears the engine down. Must be safe to call repeatedly
	// and safe to call when the engine never started.
	Stop(ctx context.Context) error
}

// ErrAlreadyStarted is returned by Registry.Start when invoked on
// an already-running registry.
var ErrAlreadyStarted = errors.New("engine registry already started")

// Registry is a small, ordered collection of named engines. Start
// brings them up in registration order; Stop tears them down in
// reverse order so dependencies wind down cleanly (e.g. the
// retention engine depends on the probe engine's results being
// flushed first).
type Registry struct {
	logger *slog.Logger

	mu       sync.Mutex
	engines  []Engine
	names    map[string]struct{}
	started  bool
	startedN int // how many engines actually came up — used by Stop
}

// NewRegistry returns an empty registry. Pass nil logger to use
// [slog.Default].
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	return &Registry{
		logger: logger,
		names:  make(map[string]struct{}),
	}
}

// Register adds an engine to the registry. Duplicate names return
// an error — engine names are global and must be unique. Returns
// an error if invoked after Start (the registry's order is
// established at registration time).
func (r *Registry) Register(e Engine) error {
	if e == nil {
		return errors.New("engine: nil engine cannot be registered")
	}
	name := e.Name()
	if name == "" {
		return errors.New("engine: registered engine has empty Name()")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return errors.New("engine: cannot Register after Start")
	}
	if _, dup := r.names[name]; dup {
		return fmt.Errorf("engine: duplicate engine name %q", name)
	}
	r.engines = append(r.engines, e)
	r.names[name] = struct{}{}
	return nil
}

// Engines returns a snapshot of the registered engines in
// registration order. Useful for /api/engines listings.
func (r *Registry) Engines() []Engine {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Engine, len(r.engines))
	copy(out, r.engines)
	return out
}

// Start brings every registered engine up in registration order.
// Returns [ErrAlreadyStarted] if invoked on a running registry. If
// any engine returns an error, every engine that already started
// is stopped before Start returns the error.
func (r *Registry) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return ErrAlreadyStarted
	}
	r.started = true
	snapshot := make([]Engine, len(r.engines))
	copy(snapshot, r.engines)
	r.mu.Unlock()

	for i, e := range snapshot {
		r.logger.InfoContext(ctx, "engine starting", "engine", e.Name())
		if err := e.Start(ctx); err != nil {
			r.logger.ErrorContext(ctx, "engine start failed",
				"engine", e.Name(), "error", err)
			// Roll back: stop everything we already started.
			r.rollback(ctx, snapshot[:i])
			r.mu.Lock()
			r.started = false
			r.startedN = 0
			r.mu.Unlock()
			return fmt.Errorf("engine %q: %w", e.Name(), err)
		}
		r.mu.Lock()
		r.startedN++
		r.mu.Unlock()
	}
	return nil
}

// Stop tears engines down in reverse registration order. Safe to
// call when Start was never invoked or when a previous Start
// failed partway. Returns the first stop error encountered but
// always continues stopping the rest.
func (r *Registry) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return nil
	}
	snapshot := make([]Engine, r.startedN)
	copy(snapshot, r.engines[:r.startedN])
	r.started = false
	r.startedN = 0
	r.mu.Unlock()

	return r.stopAll(ctx, snapshot)
}

// rollback stops engines that came up before a Start error,
// without flipping the started flag (caller already cleared it).
func (r *Registry) rollback(ctx context.Context, started []Engine) {
	_ = r.stopAll(ctx, started)
}

// stopAll iterates engines in reverse order, logging and tracking
// the first error but always running every Stop.
func (r *Registry) stopAll(ctx context.Context, engines []Engine) error {
	var firstErr error
	for _, e := range slices.Backward(engines) {
		r.logger.InfoContext(ctx, "engine stopping", "engine", e.Name())
		if err := e.Stop(ctx); err != nil {
			r.logger.ErrorContext(ctx, "engine stop failed",
				"engine", e.Name(), "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("engine %q: %w", e.Name(), err)
			}
		}
	}
	return firstErr
}
