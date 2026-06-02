package jobs_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/platform/events"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// --- test harness ---------------------------------------------------------

// fixture bundles a runner with the bus it publishes to so a test can both
// drive jobs and observe the state-change facts they emit.
type fixture struct {
	runner *jobs.Runner
	bus    *events.Bus
}

func newFixture(t *testing.T, cfg jobs.Config) *fixture {
	t.Helper()
	bus := events.New(slog.New(slog.DiscardHandler))
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), cfg)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runner.Close(ctx); err != nil {
			t.Errorf("runner.Close: %v", err)
		}
		if err := bus.Close(ctx); err != nil {
			t.Errorf("bus.Close: %v", err)
		}
	})
	return &fixture{runner: runner, bus: bus}
}

// watch subscribes to the job event for a single terminal/intermediate state
// and returns a channel of the jobs seen in that state. It lets tests wait on a
// concrete fact instead of sleeping.
func (f *fixture) watch(t *testing.T, state jobs.State) <-chan jobs.Job {
	t.Helper()
	ch := make(chan jobs.Job, 16)
	f.bus.Subscribe(jobs.Topic(state), func(_ context.Context, ev events.Event) {
		if je, ok := ev.(jobs.JobEvent); ok {
			ch <- je.Job
		}
	})
	return ch
}

// awaitJob blocks until a job with id appears on ch, failing on timeout.
func awaitJob(t *testing.T, ch <-chan jobs.Job, id string) jobs.Job {
	t.Helper()
	for {
		select {
		case j := <-ch:
			if j.ID == id {
				return j
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("job %s never reached the awaited state", id)
		}
	}
}

// okHandler succeeds immediately, returning result.
func okHandler(result any) jobs.Handler {
	return func(context.Context, any, func(float64)) (any, error) {
		return result, nil
	}
}

// blockHandler signals on started, then blocks until release is closed or the
// job's context is cancelled. It reports which happened via the returned error.
func blockHandler(started chan<- struct{}, release <-chan struct{}) jobs.Handler {
	return func(ctx context.Context, _ any, _ func(float64)) (any, error) {
		close(started)
		select {
		case <-release:
			return "released", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// --- tests ----------------------------------------------------------------

func TestRunnerRunsJobToSuccess(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	if err := f.runner.Register("noop", okHandler("payload")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, err := f.runner.Submit("noop", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	done := awaitJob(t, succeeded, id)
	if done.State != jobs.StateSucceeded {
		t.Fatalf("state = %q, want succeeded", done.State)
	}
	if done.Result != "payload" {
		t.Fatalf("result = %v, want payload", done.Result)
	}
	if done.Progress != 1 {
		t.Fatalf("progress = %v, want 1", done.Progress)
	}

	got, ok := f.runner.Get(id)
	if !ok || got.State != jobs.StateSucceeded || got.Result != "payload" {
		t.Fatalf("Get = %+v, ok=%v; want succeeded/payload", got, ok)
	}
}

func TestSubmitUnknownKindIsRejected(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	_, err := f.runner.Submit("does-not-exist", nil)
	if !errors.Is(err, jobs.ErrUnknownKind) {
		t.Fatalf("Submit err = %v, want ErrUnknownKind", err)
	}
}

func TestFailingHandlerMarksJobFailed(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	wantErr := errors.New("disk on fire")
	_ = f.runner.Register("boom", func(context.Context, any, func(float64)) (any, error) {
		return nil, wantErr
	})
	failed := f.watch(t, jobs.StateFailed)

	id, _ := f.runner.Submit("boom", nil)
	j := awaitJob(t, failed, id)
	if j.State != jobs.StateFailed {
		t.Fatalf("state = %q, want failed", j.State)
	}
	if j.Err == "" {
		t.Fatal("failed job has empty Err, want the handler error message")
	}
}

func TestRunnerRejectsAtCapacity(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{MaxConcurrent: 1})
	started := make(chan struct{})
	release := make(chan struct{})
	_ = f.runner.Register("slow", blockHandler(started, release))
	_ = f.runner.Register("quick", okHandler("done"))
	succeeded := f.watch(t, jobs.StateSucceeded)

	first, err := f.runner.Submit("slow", nil)
	if err != nil {
		t.Fatalf("first Submit: %v", err)
	}

	// The single slot is now occupied: a second submit must be rejected with a
	// distinct at-capacity error, not queued.
	_, err = f.runner.Submit("slow", nil)
	if !errors.Is(err, jobs.ErrAtCapacity) {
		t.Fatalf("second Submit err = %v, want ErrAtCapacity", err)
	}

	close(release)
	awaitJob(t, succeeded, first) // slot frees only after completion

	// With the slot freed, a fresh submit is accepted and runs to completion;
	// an at-capacity error here would mean the slot never freed.
	third, err := f.runner.Submit("quick", nil)
	if err != nil {
		t.Fatalf("third Submit after slot freed = %v, want accepted", err)
	}
	awaitJob(t, succeeded, third)
}

func TestCancelStopsRunningJob(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	_ = f.runner.Register("slow", blockHandler(started, release))
	cancelled := f.watch(t, jobs.StateCancelled)

	id, _ := f.runner.Submit("slow", nil)
	<-started // handler is now blocked inside the runner

	if err := f.runner.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	j := awaitJob(t, cancelled, id)
	if j.State != jobs.StateCancelled {
		t.Fatalf("state = %q, want cancelled", j.State)
	}
}

func TestCancelUnknownJob(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	if err := f.runner.Cancel("nope"); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("Cancel err = %v, want ErrNotFound", err)
	}
}

func TestCancelTerminalJobIsNoop(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	_ = f.runner.Register("noop", okHandler(nil))
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, _ := f.runner.Submit("noop", nil)
	awaitJob(t, succeeded, id)

	if err := f.runner.Cancel(id); err != nil {
		t.Fatalf("Cancel of terminal job = %v, want nil (idempotent no-op)", err)
	}
	got, _ := f.runner.Get(id)
	if got.State != jobs.StateSucceeded {
		t.Fatalf("state after cancel = %q, want still succeeded", got.State)
	}
}

func TestProgressReporting(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	reported := make(chan struct{})
	release := make(chan struct{})
	_ = f.runner.Register("progressing", func(_ context.Context, _ any, report func(float64)) (any, error) {
		report(0.5)
		close(reported)
		<-release
		return "ok", nil
	})
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, _ := f.runner.Submit("progressing", nil)
	<-reported // 0.5 has been recorded, job still running

	mid, ok := f.runner.Get(id)
	if !ok || mid.Progress != 0.5 {
		t.Fatalf("mid-flight progress = %v (ok=%v), want 0.5", mid.Progress, ok)
	}

	close(release)
	final := awaitJob(t, succeeded, id)
	if final.Progress != 1 {
		t.Fatalf("final progress = %v, want 1", final.Progress)
	}
}

func TestProgressIsClamped(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	reported := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	_ = f.runner.Register("wild", func(_ context.Context, _ any, report func(float64)) (any, error) {
		report(-0.5)
		report(7)
		close(reported)
		<-release
		return "ok", nil
	})

	id, _ := f.runner.Submit("wild", nil)
	<-reported
	got, _ := f.runner.Get(id)
	if got.Progress != 1 {
		t.Fatalf("progress = %v, want clamped to 1", got.Progress)
	}
}

func TestPanicInHandlerFailsJobCleanly(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	_ = f.runner.Register("panicker", func(context.Context, any, func(float64)) (any, error) {
		panic("handler boom")
	})
	_ = f.runner.Register("noop", okHandler("fine"))
	failed := f.watch(t, jobs.StateFailed)
	succeeded := f.watch(t, jobs.StateSucceeded)

	bad, _ := f.runner.Submit("panicker", nil)
	j := awaitJob(t, failed, bad)
	if j.State != jobs.StateFailed {
		t.Fatalf("panicking job state = %q, want failed", j.State)
	}
	if j.Err == "" {
		t.Fatal("panicking job has empty Err, want recovered panic detail")
	}

	// The runner must still be alive and able to run further jobs.
	good, err := f.runner.Submit("noop", nil)
	if err != nil {
		t.Fatalf("Submit after panic: %v", err)
	}
	if done := awaitJob(t, succeeded, good); done.Result != "fine" {
		t.Fatalf("post-panic job result = %v, want fine", done.Result)
	}
}

func TestStateChangeEventsEmitted(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	_ = f.runner.Register("noop", okHandler(nil))

	// A subscriber on each lifecycle topic records which transitions fired. The
	// bus guarantees order per subscriber, not across these three independent
	// subscriptions, so this asserts the set of facts emitted, not their
	// interleaving — lifecycle ordering itself is enforced by the runner.
	var (
		mu   sync.Mutex
		seen = map[jobs.State]bool{}
	)
	record := func(_ context.Context, ev events.Event) {
		if je, ok := ev.(jobs.JobEvent); ok {
			mu.Lock()
			seen[je.Job.State] = true
			mu.Unlock()
		}
	}
	for _, s := range []jobs.State{jobs.StateQueued, jobs.StateRunning, jobs.StateSucceeded} {
		f.bus.Subscribe(jobs.Topic(s), record)
	}
	done := f.watch(t, jobs.StateSucceeded)

	id, _ := f.runner.Submit("noop", nil)
	awaitJob(t, done, id)

	// Drain the bus so every queued event is delivered before asserting.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.bus.Close(ctx); err != nil {
		t.Fatalf("bus drain: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, s := range []jobs.State{jobs.StateQueued, jobs.StateRunning, jobs.StateSucceeded} {
		if !seen[s] {
			t.Fatalf("no %q event emitted; saw %v", s, seen)
		}
	}
}

func TestGracefulShutdownCancelsInFlight(t *testing.T) {
	t.Parallel()

	bus := events.New(slog.New(slog.DiscardHandler))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = bus.Close(ctx)
	}()
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), jobs.Config{MaxConcurrent: 8})

	const n = 4
	starts := make([]chan struct{}, n)
	ids := make([]string, n)
	for i := range n {
		starts[i] = make(chan struct{})
		_ = runner.Register("slow"+string(rune('a'+i)), blockHandler(starts[i], make(chan struct{})))
		ids[i], _ = runner.Submit("slow"+string(rune('a'+i)), nil)
		<-starts[i] // ensure all are actually running and blocked on ctx
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runner.Close(ctx); err != nil {
		t.Fatalf("Close drained incompletely: %v", err)
	}

	for _, id := range ids {
		got, ok := runner.Get(id)
		if !ok || got.State != jobs.StateCancelled {
			t.Fatalf("in-flight job %s = %+v (ok=%v), want cancelled after shutdown", id, got, ok)
		}
	}
}

func TestSubmitAfterCloseIsRejected(t *testing.T) {
	t.Parallel()

	bus := events.New(slog.New(slog.DiscardHandler))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = bus.Close(ctx)
	}()
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), jobs.Config{})
	_ = runner.Register("noop", okHandler(nil))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runner.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := runner.Submit("noop", nil); !errors.Is(err, jobs.ErrClosed) {
		t.Fatalf("Submit after Close err = %v, want ErrClosed", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{}) // Cleanup will Close once more
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.runner.Close(ctx); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := f.runner.Close(ctx); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestRetentionCleanupRemovesTerminalJobs(t *testing.T) {
	t.Parallel()

	// Retention 0 → every terminal job is immediately eligible for cleanup.
	f := newFixture(t, jobs.Config{Retention: 0})
	_ = f.runner.Register("noop", okHandler(nil))
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, _ := f.runner.Submit("noop", nil)
	awaitJob(t, succeeded, id)

	if removed := f.runner.Cleanup(); removed != 1 {
		t.Fatalf("Cleanup removed %d, want 1", removed)
	}
	if _, ok := f.runner.Get(id); ok {
		t.Fatal("terminal job still present after Cleanup with zero retention")
	}
}

func TestRetentionKeepsRecentTerminalJobs(t *testing.T) {
	t.Parallel()

	// A long retention window keeps just-finished jobs around.
	f := newFixture(t, jobs.Config{Retention: time.Hour})
	_ = f.runner.Register("noop", okHandler(nil))
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, _ := f.runner.Submit("noop", nil)
	awaitJob(t, succeeded, id)

	if removed := f.runner.Cleanup(); removed != 0 {
		t.Fatalf("Cleanup removed %d, want 0 within retention window", removed)
	}
	if _, ok := f.runner.Get(id); !ok {
		t.Fatal("recent terminal job purged despite long retention window")
	}
}

func TestCleanupLeavesActiveJobs(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{Retention: 0})
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	_ = f.runner.Register("slow", blockHandler(started, release))

	id, _ := f.runner.Submit("slow", nil)
	<-started

	if removed := f.runner.Cleanup(); removed != 0 {
		t.Fatalf("Cleanup removed %d active jobs, want 0", removed)
	}
	if _, ok := f.runner.Get(id); !ok {
		t.Fatal("active job purged by Cleanup")
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	if err := f.runner.Register("", okHandler(nil)); err == nil {
		t.Fatal("Register with empty kind = nil, want error")
	}
	if err := f.runner.Register("k", nil); err == nil {
		t.Fatal("Register with nil handler = nil, want error")
	}
	if err := f.runner.Register("dup", okHandler(nil)); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := f.runner.Register("dup", okHandler(nil)); err == nil {
		t.Fatal("duplicate Register = nil, want error")
	}
}

func TestConcurrentSubmitGetCancelIsRaceFree(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{MaxConcurrent: 64})
	_ = f.runner.Register("fast", okHandler("x"))

	const workers = 16
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		go func(w int) {
			defer wg.Done()
			for i := range 50 {
				id, err := f.runner.Submit("fast", i)
				if errors.Is(err, jobs.ErrAtCapacity) {
					continue
				}
				if err != nil {
					return
				}
				_, _ = f.runner.Get(id)
				_ = f.runner.Cancel(id)
				_ = w
			}
		}(w)
	}
	wg.Wait()
	// Cleanup is exercised concurrently with the tail of in-flight jobs.
	f.runner.Cleanup()
}
