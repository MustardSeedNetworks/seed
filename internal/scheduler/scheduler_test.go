package scheduler_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/scheduler"
)

// fakeJob is a controllable Job for tests.
type fakeJob struct {
	id       string
	next     time.Time
	runCount atomic.Int32
	runErr   error
	mu       sync.Mutex
	onRun    func(ctx context.Context)
}

func (j *fakeJob) ID() string { return j.id }

func (j *fakeJob) NextRun(_ time.Time) time.Time {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.next
}

func (j *fakeJob) Run(ctx context.Context) error {
	j.runCount.Add(1)
	if j.onRun != nil {
		j.onRun(ctx)
	}
	return j.runErr
}

func TestScheduler_RegisterListUnregister(t *testing.T) {
	t.Parallel()

	s := scheduler.New(time.Hour)
	defer s.Stop()

	j := &fakeJob{id: "test-1"}
	s.Register(j)

	got, ok := s.Get("test-1")
	if !ok {
		t.Fatal("Get returned !ok after Register")
	}
	if got.ID() != "test-1" {
		t.Errorf("Get returned id %q, want test-1", got.ID())
	}

	snap := s.Snapshot()
	if len(snap) != 1 {
		t.Errorf("Snapshot returned %d jobs, want 1", len(snap))
	}

	if !s.Unregister("test-1") {
		t.Error("Unregister returned false for present job")
	}
	if _, present := s.Get("test-1"); present {
		t.Error("Get returned ok after Unregister")
	}
	if s.Unregister("test-1") {
		t.Error("second Unregister returned true")
	}
}

func TestScheduler_RegisterReplacesExisting(t *testing.T) {
	t.Parallel()

	s := scheduler.New(time.Hour)
	defer s.Stop()

	first := &fakeJob{id: "same-id", runErr: nil}
	second := &fakeJob{id: "same-id"}

	s.Register(first)
	s.Register(second)

	got, _ := s.Get("same-id")
	if got != second {
		t.Error("Register did not replace existing job")
	}
	if len(s.Snapshot()) != 1 {
		t.Errorf("Snapshot length = %d, want 1 after re-register", len(s.Snapshot()))
	}
}

// The Scheduler's clock interface is unexported, so these tests
// exercise the public API via the real clock with short tick
// intervals. A future test-seam refactor could expose the clock for
// fully deterministic timing — for Phase 0, real-clock with short
// intervals is sufficient.

func TestScheduler_FiresDueJob(t *testing.T) {
	t.Parallel()

	s := scheduler.New(20 * time.Millisecond)

	var fired atomic.Int32
	done := make(chan struct{}, 1)
	j := &fakeJob{
		id:   "due",
		next: time.Now().Add(-time.Second), // already past
		onRun: func(_ context.Context) {
			if fired.Add(1) == 1 {
				done <- struct{}{}
			}
		},
	}
	s.Register(j)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.Start(ctx)

	select {
	case <-done:
		// good
	case <-ctx.Done():
		t.Fatal("job did not fire within 2s")
	}

	s.Stop()
	if got := fired.Load(); got < 1 {
		t.Errorf("job fired %d times, want >= 1", got)
	}
}

func TestScheduler_SkipsZeroNextRun(t *testing.T) {
	t.Parallel()

	s := scheduler.New(20 * time.Millisecond)

	j := &fakeJob{
		id:   "paused",
		next: time.Time{}, // zero => never run
	}
	s.Register(j)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	<-ctx.Done()
	s.Stop()

	if got := j.runCount.Load(); got != 0 {
		t.Errorf("paused job fired %d times, want 0", got)
	}
}

func TestScheduler_StopWaitsForInflight(t *testing.T) {
	t.Parallel()

	s := scheduler.New(20 * time.Millisecond)

	released := make(chan struct{})
	started := make(chan struct{}, 1)
	j := &fakeJob{
		id:   "slow",
		next: time.Now().Add(-time.Second),
		onRun: func(_ context.Context) {
			select {
			case started <- struct{}{}:
			default:
			}
			<-released
		},
	}
	s.Register(j)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.Start(ctx)

	// Wait for the job to start.
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("slow job didn't start within 2s")
	}

	// Stop in a goroutine; it should block until released is closed.
	stopped := make(chan struct{})
	go func() {
		s.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("Stop returned before in-flight job released")
	case <-time.After(80 * time.Millisecond):
		// good — Stop is waiting
	}

	close(released)
	select {
	case <-stopped:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return after job released")
	}
}

func TestScheduler_RunErrorDoesNotKillScheduler(t *testing.T) {
	t.Parallel()

	s := scheduler.New(15 * time.Millisecond)

	var fired atomic.Int32
	j := &fakeJob{
		id:     "errs",
		next:   time.Now().Add(-time.Second),
		runErr: context.Canceled, // arbitrary non-nil
		onRun: func(_ context.Context) {
			fired.Add(1)
		},
	}
	s.Register(j)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	<-ctx.Done()
	s.Stop()

	if got := fired.Load(); got < 2 {
		t.Errorf("scheduler stopped firing after error; fire count = %d, want >= 2", got)
	}
}

func TestScheduler_ConcurrentRegisterUnregister(t *testing.T) {
	t.Parallel()

	s := scheduler.New(time.Hour)
	defer s.Stop()

	var wg sync.WaitGroup
	for i := range 50 {
		id := i
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.Register(&fakeJob{id: idOf(id)})
		}()
		go func() {
			defer wg.Done()
			_ = s.Unregister(idOf(id))
		}()
	}
	wg.Wait()
	// no deadlock + race detector happy is the test
}

func idOf(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	return string(digits[n/10]) + string(digits[n%10])
}
