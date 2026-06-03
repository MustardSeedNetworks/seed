// Package jobs is seed's unified async job runner (ADR-0005).
//
// Every long-running operation — speedtest, iperf, discovery scan, vuln scan,
// survey, traceroute, pipeline — used to reinvent run/status/cancel/progress as
// its own pair of endpoints. This package collapses them into one model: a Job
// of some Kind, advanced through a fixed lifecycle by a registered Handler, with
// uniform cancellation, progress, and result capture. The HTTP surface
// (POST /jobs, GET /jobs/{id}, DELETE /jobs/{id}, the single SSE stream) and the
// refactor of the real long-ops into job kinds are layered on separately; this
// is the in-memory core.
//
// Lifecycle:
//
//	queued ──▶ running ──▶ succeeded   (handler returned nil error)
//	                  └──▶ failed      (handler returned an error or panicked)
//	                  └──▶ cancelled   (Cancel or shutdown, handler observed ctx)
//
// Semantics:
//   - Bounded concurrency with a hard cap. Submit rejects with ErrAtCapacity
//     once the cap of active (queued or running) jobs is reached — appliance
//     backpressure, never an unbounded queue. ErrAtCapacity is the 503 signal.
//   - Cancellation via context: Cancel cancels the job's context; a well-behaved
//     Handler observes ctx.Done() and returns, landing the job in cancelled.
//   - Progress reporting: a Handler calls its report callback with a fraction in
//     [0,1]; the value is clamped and visible through Get.
//   - Result capture: a Handler's return value becomes Job.Result on success.
//   - Panic isolation: a panicking Handler fails just that job (recovered and
//     logged) without crashing the runner.
//   - State-change facts: every transition publishes a JobEvent on the events
//     bus, so the frontend can follow one job/event stream.
//   - Retention: terminal jobs are kept in memory until Cleanup removes those
//     older than the configured retention window (driven by the scheduler).
//
// Durability is optional via Config.Store (Phase 5c): when set, the runner
// write-throughs each state transition and reads through on a Get miss, and
// Recover reconciles jobs left in-flight by a previous process at startup. With
// no store the runner is in-memory only — the correct fail-cleanly v1 (ADR-0005).
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/platform/events"
)

// defaultMaxConcurrent bounds in-flight jobs when Config leaves it unset. Sized
// for an appliance: enough parallelism for independent scans, low enough that
// overload is rejected rather than thrashing.
const defaultMaxConcurrent = 8

// State is a point in a job's lifecycle.
type State string

// The five lifecycle states (ADR-0005). queued and running are active; the rest
// are terminal.
const (
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateSucceeded State = "succeeded"
	StateFailed    State = "failed"
	StateCancelled State = "cancelled"
)

// Sentinel errors returned by the runner. Callers match with [errors.Is]; the
// HTTP layer maps ErrAtCapacity to 503, ErrNotFound to 404, ErrUnknownKind to
// 400, ErrClosed to 503.
var (
	ErrAtCapacity  = errors.New("jobs: runner at capacity")
	ErrUnknownKind = errors.New("jobs: unknown job kind")
	ErrNotFound    = errors.New("jobs: job not found")
	ErrClosed      = errors.New("jobs: runner closed")
	errPanic       = errors.New("jobs: handler panicked")
)

// Topic returns the events-bus topic a state change is published on, e.g.
// "job.running". Subscribers register per state.
func Topic(s State) string { return "job." + string(s) }

// States returns the five lifecycle states in canonical order. A consumer that
// must observe every transition — an SSE bridge, an audit sink — subscribes to
// Topic(s) for each, rather than hardcoding the set at the call site.
func States() []State {
	return []State{StateQueued, StateRunning, StateSucceeded, StateFailed, StateCancelled}
}

// Job is an immutable snapshot of a unit of work. Get and JobEvent hand out
// copies; mutating one never affects the runner's state.
type Job struct {
	ID       string
	Kind     string
	State    State
	Progress float64 // fraction in [0,1]
	Result   any     // set on success; nil otherwise
	Err      string  // failure detail; empty unless State is failed
}

// JobEvent is the domain fact published on every state change. It carries a
// snapshot of the job as of that transition.
type JobEvent struct {
	Job Job
}

// Topic routes the event by the job's new state, e.g. "job.succeeded".
func (e JobEvent) Topic() string { return Topic(e.Job.State) }

// Store is the durable backing for jobs (ADR-0005, Phase 5c). The Runner
// write-throughs a snapshot on every state transition and falls back to it on a
// Get miss, so a job survives a restart. nil disables persistence — the
// in-memory map stays the source of truth for active jobs (the fail-cleanly v1).
// Progress ticks are deliberately not persisted (too frequent); only state
// changes are. Implementations must be safe for concurrent use.
type Store interface {
	// Save persists the current snapshot of j (upsert by ID). Best-effort from
	// the Runner's perspective: a Save error is logged, never fails the job.
	Save(ctx context.Context, j Job) error
	// Load returns the persisted job and whether it exists.
	Load(ctx context.Context, id string) (Job, bool, error)
	// MarkInterrupted transitions every persisted non-terminal job (queued or
	// running) to failed — called once at startup via Recover, since a restart's
	// lost handler goroutines can't resume them. Returns the count transitioned.
	MarkInterrupted(ctx context.Context) (int, error)
}

// Handler executes one job of a kind. It receives a context cancelled when the
// job is cancelled or the runner shuts down, the submit-time params, and a
// report callback for progress in [0,1]. Its return value becomes Job.Result; a
// non-nil error fails the job. A well-behaved handler returns promptly once ctx
// is done.
type Handler func(ctx context.Context, params any, report func(float64)) (any, error)

// Config tunes a Runner. The zero value is valid: an 8-job cap, zero
// retention (terminal jobs are eligible for Cleanup immediately), and no
// persistence.
type Config struct {
	// MaxConcurrent caps active (queued or running) jobs. <= 0 uses the default.
	MaxConcurrent int
	// Retention is how long a terminal job is kept before Cleanup may remove it.
	Retention time.Duration
	// Store, when non-nil, durably backs the runner: snapshots are written
	// through on each state transition and read on a Get miss. nil keeps the
	// in-memory-only fail-cleanly behavior.
	Store Store
}

// Runner schedules and tracks jobs. The zero value is not usable; call New. A
// Runner is safe for concurrent use.
type Runner struct {
	bus       *events.Bus
	logger    *slog.Logger
	store     Store // nil when persistence is disabled
	cap       int
	retention time.Duration

	mu       sync.Mutex
	handlers map[string]Handler
	jobs     map[string]*entry
	active   int
	closed   bool
	wg       sync.WaitGroup
}

// entry is a job's mutable state behind the runner's mutex. The job snapshot,
// its cancel func, and bookkeeping all live here; nothing outside this file
// touches it without holding Runner.mu.
type entry struct {
	job         Job
	cancel      context.CancelFunc
	cancelled   bool      // a cancel was requested (vs. a plain handler error)
	completedAt time.Time // when it reached a terminal state; zero while active
}

// New returns a runner that publishes state changes to bus and logs handler
// panics to logger. bus and logger must be non-nil.
func New(bus *events.Bus, logger *slog.Logger, cfg Config) *Runner {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = defaultMaxConcurrent
	}
	return &Runner{
		bus:       bus,
		logger:    logger,
		store:     cfg.Store,
		cap:       cfg.MaxConcurrent,
		retention: cfg.Retention,
		handlers:  make(map[string]Handler),
		jobs:      make(map[string]*entry),
	}
}

// Register binds a handler to a job kind. It must be called before Submit of
// that kind (typically at boot). Re-registering a kind, an empty kind, or a nil
// handler is an error.
func (r *Runner) Register(kind string, h Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	if kind == "" {
		return errors.New("jobs: empty kind")
	}
	if h == nil {
		return errors.New("jobs: nil handler")
	}
	if _, dup := r.handlers[kind]; dup {
		return fmt.Errorf("jobs: kind %q already registered", kind)
	}
	r.handlers[kind] = h
	return nil
}

// Submit enqueues a job of kind with params and returns its id. It rejects with
// ErrUnknownKind if no handler is registered, ErrAtCapacity if the concurrency
// cap is reached, or ErrClosed after shutdown. The job runs asynchronously.
func (r *Runner) Submit(kind string, params any) (string, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return "", ErrClosed
	}
	h, ok := r.handlers[kind]
	if !ok {
		r.mu.Unlock()
		return "", fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	if r.active >= r.cap {
		r.mu.Unlock()
		return "", ErrAtCapacity
	}

	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	e := &entry{
		job:    Job{ID: id, Kind: kind, State: StateQueued},
		cancel: cancel,
	}
	r.jobs[id] = e
	r.active++
	r.wg.Add(1)
	snap := e.job
	r.mu.Unlock()

	r.bus.Publish(JobEvent{Job: snap})
	r.persist(snap)
	go r.execute(ctx, e, h, params)
	return id, nil
}

// Get returns a snapshot of the job and whether it exists. Active and recently
// terminal jobs are served from memory; on a miss it falls back to the durable
// store (if configured), so a job that completed before the last restart, or
// was evicted by Cleanup but not yet by retention, is still retrievable.
func (r *Runner) Get(id string) (Job, bool) {
	r.mu.Lock()
	e, ok := r.jobs[id]
	if ok {
		snap := e.job
		r.mu.Unlock()
		return snap, true
	}
	r.mu.Unlock()

	if r.store == nil {
		return Job{}, false
	}
	j, found, err := r.store.Load(context.Background(), id)
	if err != nil {
		r.logger.ErrorContext(context.Background(), "load job from store", "job", id, "err", err)
		return Job{}, false
	}
	return j, found
}

// Cancel requests cancellation of a job, cancelling its context so a
// well-behaved handler unwinds into the cancelled state. It is idempotent:
// cancelling a terminal or already-cancelling job is a no-op. An unknown id
// returns ErrNotFound.
func (r *Runner) Cancel(id string) error {
	r.mu.Lock()
	e, ok := r.jobs[id]
	if !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	if isTerminal(e.job.State) {
		r.mu.Unlock()
		return nil
	}
	e.cancelled = true
	cancel := e.cancel
	r.mu.Unlock()

	cancel()
	return nil
}

// Cleanup removes terminal jobs whose age exceeds the retention window and
// returns the count removed. It is intended to be driven by the scheduler on a
// retention tick; active jobs are never removed.
func (r *Runner) Cleanup() int {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	removed := 0
	for id, e := range r.jobs {
		if isTerminal(e.job.State) && now.Sub(e.completedAt) >= r.retention {
			delete(r.jobs, id)
			removed++
		}
	}
	return removed
}

// Close stops accepting new jobs, cancels every in-flight job, and waits for
// their handlers to return. It returns nil once drained, or ctx.Err() if ctx is
// cancelled first. Close is idempotent.
func (r *Runner) Close(ctx context.Context) error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	cancels := make([]context.CancelFunc, 0, len(r.jobs))
	for _, e := range r.jobs {
		if !isTerminal(e.job.State) {
			e.cancelled = true
			cancels = append(cancels, e.cancel)
		}
	}
	r.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// execute runs a job's handler to completion and records the terminal state.
func (r *Runner) execute(ctx context.Context, e *entry, h Handler, params any) {
	defer r.wg.Done()
	defer e.cancel()

	r.transition(e, func(j *Job) { j.State = StateRunning })

	report := func(p float64) {
		r.mu.Lock()
		e.job.Progress = clamp(p)
		r.mu.Unlock()
	}

	result, err := safeRun(ctx, r.logger, h, params, report)
	r.finish(e, result, err)
}

// finish records a job's terminal state, frees its slot, and publishes the fact.
func (r *Runner) finish(e *entry, result any, err error) {
	r.mu.Lock()
	switch {
	case err == nil:
		e.job.State = StateSucceeded
		e.job.Result = result
		e.job.Progress = 1
	case e.cancelled:
		e.job.State = StateCancelled
	default:
		e.job.State = StateFailed
		e.job.Err = err.Error()
	}
	e.completedAt = time.Now()
	r.active--
	snap := e.job
	r.mu.Unlock()

	r.bus.Publish(JobEvent{Job: snap})
	r.persist(snap)
}

// transition mutates a job's state under the lock and publishes the new fact.
func (r *Runner) transition(e *entry, mutate func(*Job)) {
	r.mu.Lock()
	mutate(&e.job)
	snap := e.job
	r.mu.Unlock()

	r.bus.Publish(JobEvent{Job: snap})
	r.persist(snap)
}

// persist write-throughs a snapshot to the durable store, if one is configured.
// It is best-effort: a store error is logged, never propagated — the in-memory
// runner stays authoritative for the live job, so a transient persistence
// failure never stalls or fails execution. Called outside the lock (after the
// bus publish) so it never extends the critical section.
func (r *Runner) persist(snap Job) {
	if r.store == nil {
		return
	}
	if err := r.store.Save(context.Background(), snap); err != nil {
		r.logger.ErrorContext(context.Background(), "persist job snapshot",
			"job", snap.ID, "state", snap.State, "err", err)
	}
}

// Recover reconciles the durable store with reality at startup: any persisted
// job still marked queued or running is from a previous process whose handler
// goroutine is gone, so it can never complete — MarkInterrupted transitions it
// to failed. It returns the number of jobs reconciled. A no-op (0, nil) when no
// store is configured. Call once after Register, before serving.
func (r *Runner) Recover(ctx context.Context) (int, error) {
	if r.store == nil {
		return 0, nil
	}
	return r.store.MarkInterrupted(ctx)
}

// safeRun invokes a handler with panic recovery, converting a panic into a
// failing error so one bad job never crashes the runner.
func safeRun(
	ctx context.Context,
	logger *slog.Logger,
	h Handler,
	params any,
	report func(float64),
) (any, error) {
	var (
		result any
		err    error
	)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(ctx, "job handler panicked", "panic", rec)
				result = nil
				err = fmt.Errorf("%w: %v", errPanic, rec)
			}
		}()
		result, err = h(ctx, params, report)
	}()
	return result, err
}

// isTerminal reports whether s is a final state.
func isTerminal(s State) bool {
	switch s {
	case StateSucceeded, StateFailed, StateCancelled:
		return true
	case StateQueued, StateRunning:
		return false
	default:
		return false
	}
}

// clamp bounds a progress fraction to [0,1].
func clamp(p float64) float64 {
	switch {
	case p < 0:
		return 0
	case p > 1:
		return 1
	default:
		return p
	}
}
