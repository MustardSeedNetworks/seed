package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/scheduler"
)

// ErrStorageNotConfigured is returned by Start / Stop / RunNow when
// the Engine was constructed without WithStorage. In-memory dispatch
// via RunDefinition still works without storage.
var ErrStorageNotConfigured = errors.New("probe.Engine has no storage configured (call WithStorage)")

// probeStorage is the subset of *database.ProbeRepository the Engine
// needs. Narrowed interface so tests can inject fakes without
// constructing a real DB.
type probeStorage interface {
	GetProbe(ctx context.Context, id string) (*database.Probe, error)
	ListProbes(ctx context.Context, clientID, kind string) ([]*database.Probe, error)
	RecordResult(ctx context.Context, pr *database.ProbeResult) error
}

// probeScheduler is the subset of *scheduler.Scheduler the Engine
// needs. Narrowed interface for test injection.
type probeScheduler interface {
	Register(j scheduler.Job)
	Unregister(id string) bool
	Start(ctx context.Context) error
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

// Start loads every enabled probe from the storage layer, registers
// each as a scheduler job, and begins the scheduler's tick loop.
// Returns ErrStorageNotConfigured if WithStorage was not called.
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

	if startErr := sched.Start(ctx); startErr != nil {
		return fmt.Errorf("scheduler start: %w", startErr)
	}

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

	r := e.RunDefinition(ctx, dbProbeToModel(p))
	if persistErr := persistResult(ctx, storage, r); persistErr != nil {
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

// dbProbeToModel converts the database row representation into the
// probe package's Probe type. The JSON columns are passed through as
// [json.RawMessage]; consumers decode them per-Kind.
func dbProbeToModel(p *database.Probe) Probe {
	return Probe{
		ID:              p.ID,
		ClientID:        p.ClientID,
		Kind:            p.Kind,
		DisplayName:     p.DisplayName,
		Target:          p.Target,
		Params:          json.RawMessage(p.ParamsJSON),
		IntervalSeconds: p.IntervalSeconds,
		Enabled:         p.Enabled,
		Warning:         json.RawMessage(p.WarningJSON),
		Critical:        json.RawMessage(p.CriticalJSON),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// persistResult writes a dispatch Result into probe_results via the
// storage layer. Metadata is preserved as-is; the database column is
// TEXT and accepts opaque JSON.
func persistResult(ctx context.Context, storage probeStorage, r Result) error {
	pr := &database.ProbeResult{
		ProbeID:      r.ProbeID,
		ClientID:     r.ClientID,
		Kind:         r.Kind,
		Timestamp:    r.Timestamp,
		Success:      r.Success,
		LatencyMs:    r.LatencyMs,
		Error:        r.Error,
		MetadataJSON: string(r.Metadata),
	}
	return storage.RecordResult(ctx, pr)
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
