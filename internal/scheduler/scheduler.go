// Package scheduler provides a generic, persistence-agnostic job
// scheduler. It is the chassis primitive for periodic work across
// seed — harvest scheduled reports, DNS endpoint monitoring, SSL cert
// expiry sweeps, microburst sampling, estate-wide SNMP polling,
// management-frame capture, VoIP/BGP collection, and any future
// collector that needs a "wake up periodically and check what's due"
// behavior.
//
// Design properties:
//
//   - Persistence-agnostic: Scheduler holds no database handle. Each
//     Job is responsible for its own state and persistence. The
//     scheduler only knows "when does this job want to run next" and
//     "run it."
//
//   - Time-source-injectable: tests pass a fake clock via NewWithClock.
//     Production callers use New.
//
//   - Fire-and-forget: each Run is dispatched on its own goroutine.
//     Stop() waits for in-flight Run calls to return via WaitGroup so
//     shutdown is graceful.
//
//   - Configurable tick: callers choose the tick interval. Tick is the
//     worst-case precision for NextRun. Harvest reports use 1 minute.
//     Microburst sampling uses 100ms baseline. DNS monitors use 1
//     minute. Each collector instantiates its own Scheduler with the
//     cadence it needs.
//
// V1.0 NMS expansion — Phase 0 extraction from
// internal/harvest/services_scheduler.go. See
// msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job is something the scheduler can run. Implementations supply their
// own identity, their own next-run calculation, and the action to
// take when due. Jobs are responsible for their own persistence.
type Job interface {
	// ID returns the job's stable identity. Two jobs with the same
	// ID cannot be registered at the same time.
	ID() string

	// NextRun returns the next absolute time at which Run should be
	// invoked. The scheduler calls this on every tick and dispatches
	// Run when now is at or past the returned time. Returning the
	// zero time signals "do not run again until further notice" —
	// the job stays registered but is skipped on subsequent ticks.
	NextRun(now time.Time) time.Time

	// Run performs the job's work. It executes on its own goroutine.
	// Errors are logged and do not stop the scheduler. The context
	// is the same one Start was called with — Run should respect
	// cancellation.
	Run(ctx context.Context) error
}

// clock abstracts [time.Now] and [time.Ticker] for tests.
type clock interface {
	Now() time.Time
	NewTicker(d time.Duration) ticker
}

// ticker abstracts [time.Ticker] for tests.
type ticker interface {
	C() <-chan time.Time
	Stop()
}

// Scheduler dispatches Job.Run calls when each registered job's
// NextRun has elapsed. It is safe for concurrent use.
type Scheduler struct {
	interval time.Duration
	logger   *slog.Logger
	clk      clock

	mu     sync.RWMutex
	jobs   map[string]Job
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// runMu serializes calls to dispatch so tests can observe a
	// deterministic ordering. Production behavior is unaffected.
	runMu sync.Mutex
}

// New returns a Scheduler that wakes up every interval to check for
// due jobs. interval must be > 0.
func New(interval time.Duration) *Scheduler {
	return NewWithClock(interval, slog.Default(), realClock{})
}

// NewWithClock is the test seam — pass a fake clock to drive ticks
// deterministically.
func NewWithClock(interval time.Duration, logger *slog.Logger, clk clock) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{
		interval: interval,
		logger:   logger,
		clk:      clk,
		jobs:     make(map[string]Job),
	}
}

// Start begins the tick loop. The caller's ctx controls scheduler
// lifetime; cancelling it (or calling Stop) ends the loop.
// Start must be called exactly once per Scheduler instance.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	t := s.clk.NewTicker(s.interval)
	s.wg.Add(1)
	go s.loop(ctx, t)
}

func (s *Scheduler) loop(ctx context.Context, t ticker) {
	defer s.wg.Done()
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C():
			s.dispatch(ctx, now)
		}
	}
}

func (s *Scheduler) dispatch(ctx context.Context, now time.Time) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	s.mu.RLock()
	due := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		next := j.NextRun(now)
		if next.IsZero() {
			continue
		}
		if !now.Before(next) {
			due = append(due, j)
		}
	}
	s.mu.RUnlock()

	for _, j := range due {
		s.wg.Add(1)
		go func(job Job) {
			defer s.wg.Done()
			if err := job.Run(ctx); err != nil {
				s.logger.ErrorContext(ctx, "scheduler job failed",
					"job_id", job.ID(),
					"error", err)
			}
		}(j)
	}
}

// Stop cancels the tick loop and waits for in-flight Run calls to
// return. Calling Stop more than once is a no-op.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.wg.Wait()
}

// Register adds a job. If a job with the same ID is already
// registered, Register replaces it.
func (s *Scheduler) Register(j Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[j.ID()] = j
}

// Unregister removes a job by ID. It returns true if the job was
// present.
func (s *Scheduler) Unregister(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return false
	}
	delete(s.jobs, id)
	return true
}

// Get returns the registered job for id and a boolean indicating
// whether it was found.
func (s *Scheduler) Get(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

// Snapshot returns a copy of the currently registered jobs.
func (s *Scheduler) Snapshot() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

// realClock is the production clock implementation.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) NewTicker(d time.Duration) ticker {
	return &realTicker{t: time.NewTicker(d)}
}

type realTicker struct{ t *time.Ticker }

func (r *realTicker) C() <-chan time.Time { return r.t.C }
func (r *realTicker) Stop()               { r.t.Stop() }
