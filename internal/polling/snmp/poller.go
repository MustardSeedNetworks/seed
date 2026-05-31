package snmp

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/scheduler"
)

// pollStatusOK / pollStatusError are the values written into
// polling_targets.last_status by the poll job.
const (
	pollStatusOK    = "ok"
	pollStatusError = "error"
)

// ErrCollectorNotRegistered is returned when a target's collector
// chain references a collector that no implementation has registered
// for. The chain entry is skipped (logged warn) but the rest of the
// chain runs.
var ErrCollectorNotRegistered = errors.New("collector not registered")

// pollerStorage is the narrowed surface the Poller needs from the
// database layer. Tests inject a fake.
type pollerStorage interface {
	List(ctx context.Context, clientID string) ([]*database.PollingTarget, error)
	UpdateLastPoll(ctx context.Context, id, status, errMsg string) error
}

// pollerScheduler is the narrowed scheduler surface for tests.
type pollerScheduler interface {
	Register(j scheduler.Job)
	Unregister(id string) bool
	Start(ctx context.Context)
	Stop()
}

// Poller orchestrates per-target SNMP polling. On Start it loads
// every enabled target, registers one scheduler.Job per target,
// and the scheduler dispatches the per-target collector chain at
// each target's configured cadence.
type Poller struct {
	logger    *slog.Logger
	storage   pollerStorage
	scheduler pollerScheduler

	mu         sync.RWMutex
	collectors map[string]Collector

	runMu   sync.Mutex
	started bool
	jobIDs  []string
}

// NewPoller returns an unstarted Poller. Pass nil logger to use
// [slog.Default].
func NewPoller(storage pollerStorage, sched pollerScheduler, logger *slog.Logger) *Poller {
	if logger == nil {
		logger = slog.Default()
	}
	return &Poller{
		logger:     logger,
		storage:    storage,
		scheduler:  sched,
		collectors: make(map[string]Collector),
	}
}

// RegisterCollector adds a Collector. Replaces any prior collector
// registered with the same Name.
func (p *Poller) RegisterCollector(c Collector) {
	if c == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.collectors[c.Name()] = c
}

// Name returns "snmp-poller". Implements [engine.Engine] so the
// poller registers in the lifecycle registry alongside the probe +
// retention engines.
func (*Poller) Name() string { return "snmp-poller" }

// Start loads all enabled polling_targets and registers one job
// per target with the scheduler. Idempotent.
func (p *Poller) Start(ctx context.Context) error {
	p.runMu.Lock()
	defer p.runMu.Unlock()

	if p.started {
		return nil
	}

	targets, err := p.storage.List(ctx, "")
	if err != nil {
		return err
	}

	for _, t := range targets {
		job := &targetJob{poller: p, target: t}
		p.scheduler.Register(job)
		p.jobIDs = append(p.jobIDs, job.ID())
	}

	p.scheduler.Start(ctx)
	p.started = true
	p.logger.InfoContext(ctx, "snmp poller started",
		"targets", len(p.jobIDs),
		"collectors", len(p.collectors),
	)
	return nil
}

// Stop unregisters all jobs and stops the scheduler.
func (p *Poller) Stop(ctx context.Context) error {
	p.runMu.Lock()
	defer p.runMu.Unlock()

	if !p.started {
		return nil
	}

	for _, id := range p.jobIDs {
		p.scheduler.Unregister(id)
	}
	p.jobIDs = nil
	p.scheduler.Stop()
	p.started = false
	p.logger.InfoContext(ctx, "snmp poller stopped")
	return nil
}

// runChain walks the target's collector chain. Each collector is
// invoked sequentially; failures are logged and the chain
// continues. After the chain runs, last_status / last_error /
// last_polled_at are recorded.
func (p *Poller) runChain(ctx context.Context, target *database.PollingTarget) {
	creds := credentialsForTarget(target)

	wireTarget := wireTarget(target)
	var firstErr error
	for _, name := range target.CollectorChain {
		p.mu.RLock()
		c, ok := p.collectors[name]
		p.mu.RUnlock()

		if !ok {
			p.logger.WarnContext(ctx, "snmp poller: collector not registered, skipping",
				"target_id", target.ID, "collector", name)
			if firstErr == nil {
				firstErr = ErrCollectorNotRegistered
			}
			continue
		}

		if err := c.Collect(ctx, wireTarget, creds); err != nil {
			p.logger.WarnContext(ctx, "snmp poller: collector failed",
				"target_id", target.ID, "collector", name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	status := pollStatusOK
	errMsg := ""
	if firstErr != nil {
		status = pollStatusError
		errMsg = firstErr.Error()
	}
	if updErr := p.storage.UpdateLastPoll(ctx, target.ID, status, errMsg); updErr != nil {
		p.logger.WarnContext(ctx, "snmp poller: update last_poll failed",
			"target_id", target.ID, "error", updErr)
	}
}

// credentialsForTarget resolves the decrypted credentials for a
// polling target. V1.0 ships with a no-credentials stub — Stage
// A3.x adds the device_credentials decryption via
// license.Manager.DecryptSecret.
func credentialsForTarget(_ *database.PollingTarget) ResolvedCredentials {
	return ResolvedCredentials{}
}

// wireTarget converts a database.PollingTarget into the
// internal/polling/snmp Target shape that Collectors consume.
func wireTarget(t *database.PollingTarget) Target {
	return Target{
		ID:              t.ID,
		ClientID:        t.ClientID,
		Name:            t.Name,
		IPAddress:       t.IPAddress,
		SNMPVersion:     t.SNMPVersion,
		CredentialsID:   t.CredentialsID,
		PollIntervalSec: t.PollIntervalSec,
		CollectorChain:  t.CollectorChain,
		Enabled:         t.Enabled,
		LastPolledAt:    t.LastPolledAt,
		LastStatus:      t.LastStatus,
		LastError:       t.LastError,
	}
}

// targetJob is the scheduler.Job adapter for a single polling
// target. Implements scheduler.Job by reaching into the poller for
// dispatch.
type targetJob struct {
	poller  *Poller
	target  *database.PollingTarget
	lastRun time.Time
}

// ID is the scheduler key for this target.
func (j *targetJob) ID() string {
	return "snmp:" + j.target.ID
}

// NextRun returns now on first call, then lastRun + interval.
func (j *targetJob) NextRun(now time.Time) time.Time {
	if j.lastRun.IsZero() {
		return now
	}
	return j.lastRun.Add(time.Duration(j.target.PollIntervalSec) * time.Second)
}

// Run dispatches the target's collector chain.
func (j *targetJob) Run(ctx context.Context) error {
	j.lastRun = time.Now().UTC()
	j.poller.runChain(ctx, j.target)
	return nil
}
