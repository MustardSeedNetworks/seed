package probe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

// ErrStorageNotConfigured is returned by Start / Stop / RunNow when
// the Engine was constructed without WithStorage. In-memory dispatch
// via RunDefinition still works without storage.
var ErrStorageNotConfigured = errors.New("probe.Engine has no storage configured (call WithStorage)")

// probeStorage is the persistence port the Engine needs, expressed in the
// probe package's own domain types. The concrete adapter (internal/app)
// translates to/from the database row representation, so the probe package
// itself stays persistence-free. Narrowed so tests can inject fakes without
// constructing a real DB.
type probeStorage interface {
	GetProbe(ctx context.Context, id string) (Probe, error)
	ListProbes(ctx context.Context, clientID, kind string) ([]Probe, error)
	RecordResult(ctx context.Context, r Result) error
}

// probeScheduler is the subset of *scheduler.Scheduler the Engine
// needs. Narrowed interface for test injection.
type probeScheduler interface {
	Register(j scheduler.Job)
	Unregister(id string) bool
	Start(ctx context.Context)
	Stop()
}

// WithStorage wires the database + scheduler used by Start, Stop,
// and RunNow. Returns the receiver so calls can chain after
// NewEngine. Pass nil to clear the wiring (used in tests).
func (e *Engine) WithStorage(storage probeStorage, sched probeScheduler) *Engine {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.storage = storage
	e.scheduler = sched
	return e
}

// Name returns "probe". Implements [engine.Engine] so the probe
// engine can register in the lifecycle registry alongside other
// long-running subsystems.
func (*Engine) Name() string { return "probe" }

// Start dispatches the scheduled probes. Returns
// ErrStorageNotConfigured if WithStorage was not called.
//
// Start is idempotent at the second-call level: a second Start
// without an intervening Stop returns nil immediately. Honors ctx
// for both the initial probe load and the scheduler's tick loop.
func (e *Engine) Start(ctx context.Context) error {
	e.runMu.Lock()
	defer e.runMu.Unlock()

	if e.started {
		return nil
	}

	storage, sched, err := e.requireStorage()
	if err != nil {
		return err
	}

	if err = e.loadAndRegisterProbes(ctx, storage, sched); err != nil {
		return err
	}

	sched.Start(ctx)
	e.started = true
	e.logger.InfoContext(ctx, "probe engine started",
		"probes_scheduled", len(e.jobIDs),
		"checkers_registered", len(e.Kinds()),
	)
	return nil
}

// Stop unregisters all scheduled probes and stops the scheduler.
// Idempotent; safe to call multiple times. Returns
// ErrStorageNotConfigured if WithStorage was not called.
func (e *Engine) Stop(ctx context.Context) error {
	e.runMu.Lock()
	defer e.runMu.Unlock()

	if !e.started {
		return nil
	}

	_, sched, err := e.requireStorage()
	if err != nil {
		return err
	}

	for _, id := range e.jobIDs {
		sched.Unregister(id)
	}
	e.jobIDs = nil

	sched.Stop()
	e.started = false
	e.logger.InfoContext(ctx, "probe engine stopped")
	return nil
}

// Reschedule re-reads probe definitions from storage and rebuilds the
// scheduler's job set, so changes made after Start — a settings save
// that adds, removes, enables, or disables probes — take effect on the
// running engine without a restart. It is a no-op if the engine has not
// been started (Start will pick up the current definitions). Safe for
// concurrent use with Start/Stop; all serialize on runMu.
func (e *Engine) Reschedule(ctx context.Context) error {
	e.runMu.Lock()
	defer e.runMu.Unlock()

	if !e.started {
		return nil
	}

	storage, sched, err := e.requireStorage()
	if err != nil {
		return err
	}

	for _, id := range e.jobIDs {
		sched.Unregister(id)
	}
	e.jobIDs = nil

	if err = e.loadAndRegisterProbes(ctx, storage, sched); err != nil {
		return err
	}

	e.logger.InfoContext(ctx, "probe engine rescheduled", "probes_scheduled", len(e.jobIDs))
	return nil
}

// loadAndRegisterProbes lists the enabled probes from storage and
// registers a scheduler job for each, recording the job IDs on the
// engine. Callers must hold runMu. Shared by Start and Reschedule.
func (e *Engine) loadAndRegisterProbes(ctx context.Context, storage probeStorage, sched probeScheduler) error {
	probes, err := storage.ListProbes(ctx, "", "")
	if err != nil {
		return fmt.Errorf("load probes: %w", err)
	}
	for _, p := range probes {
		if !p.Enabled {
			continue
		}
		job := &probeJob{engine: e, probeID: p.ID, interval: time.Duration(p.IntervalSeconds) * time.Second}
		sched.Register(job)
		e.jobIDs = append(e.jobIDs, job.ID())
	}
	return nil
}

// RunNow loads the named probe from storage, dispatches it through
// RunDefinition, and persists the Result to probe_results. Used by
// AutoTest sequences, manual UI triggers, and webhook callbacks.
// Returns ErrStorageNotConfigured if WithStorage was not called.
func (e *Engine) RunNow(ctx context.Context, probeID string) (Result, error) {
	storage, _, err := e.requireStorage()
	if err != nil {
		return Result{}, err
	}

	p, err := storage.GetProbe(ctx, probeID)
	if err != nil {
		return Result{}, fmt.Errorf("get probe %q: %w", probeID, err)
	}

	r := e.RunDefinition(ctx, p)
	if persistErr := storage.RecordResult(ctx, r); persistErr != nil {
		return r, fmt.Errorf("persist result for probe %q: %w", probeID, persistErr)
	}
	return r, nil
}

// requireStorage returns the configured storage + scheduler or
// ErrStorageNotConfigured.
func (e *Engine) requireStorage() (probeStorage, probeScheduler, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.storage == nil || e.scheduler == nil {
		return nil, nil, ErrStorageNotConfigured
	}
	return e.storage, e.scheduler, nil
}

// probeJob is the scheduler.Job adapter that drives one probe at its
// configured interval. Implements scheduler.Job by reaching into the
// engine to fetch the latest probe config + dispatch.
type probeJob struct {
	engine   *Engine
	probeID  string
	interval time.Duration
	lastRun  time.Time
}

// ID returns the probe ID. Used by the scheduler as a job key.
func (j *probeJob) ID() string {
	return "probe:" + j.probeID
}

// NextRun returns the next scheduled execution time. First call
// returns now (immediate first run); subsequent calls return
// lastRun+interval.
func (j *probeJob) NextRun(now time.Time) time.Time {
	if j.lastRun.IsZero() {
		return now
	}
	return j.lastRun.Add(j.interval)
}

// Run is called by the scheduler at NextRun. It loads the latest
// probe config (so in-flight edits take effect on the next tick)
// and dispatches via Engine.RunNow which persists the result.
func (j *probeJob) Run(ctx context.Context) error {
	j.lastRun = time.Now().UTC()
	_, err := j.engine.RunNow(ctx, j.probeID)
	if err != nil {
		j.engine.logger.WarnContext(ctx, "probe dispatch failed",
			"probe_id", j.probeID,
			"error", err,
		)
	}
	return err
}
