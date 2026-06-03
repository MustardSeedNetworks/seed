package jobs_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// fakeStore is a concurrency-safe in-memory jobs.Store for exercising the
// runner's durability seam without a database.
type fakeStore struct {
	mu       sync.Mutex
	jobs     map[string]jobs.Job
	saves    int
	loadErr  error
	failSave bool
}

func newFakeStore() *fakeStore { return &fakeStore{jobs: make(map[string]jobs.Job)} }

func (s *fakeStore) Save(_ context.Context, j jobs.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failSave {
		return errors.New("fakeStore: save failed")
	}
	s.jobs[j.ID] = j
	s.saves++
	return nil
}

func (s *fakeStore) Load(_ context.Context, id string) (jobs.Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return jobs.Job{}, false, s.loadErr
	}
	j, ok := s.jobs[id]
	return j, ok, nil
}

func (s *fakeStore) MarkInterrupted(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, j := range s.jobs {
		if j.State == jobs.StateQueued || j.State == jobs.StateRunning {
			j.State = jobs.StateFailed
			j.Err = "interrupted by restart"
			s.jobs[id] = j
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) snapshot(id string) (jobs.Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	return j, ok
}

// waitForStoreState polls the store until id reaches state, failing on timeout.
// The runner persists after publishing, so a test that awaits the bus event
// must still wait for the (best-effort, slightly later) write-through.
func waitForStoreState(t *testing.T, s *fakeStore, id string, state jobs.State) jobs.Job {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		if j, ok := s.snapshot(id); ok && j.State == state {
			return j
		}
		select {
		case <-deadline:
			j, _ := s.snapshot(id)
			t.Fatalf("store never saw job %s in state %q (last %q)", id, state, j.State)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// TestRunnerWritesThroughToStore verifies a successful job's terminal snapshot
// is persisted, with the success result captured.
func TestRunnerWritesThroughToStore(t *testing.T) {
	t.Parallel()

	fs := newFakeStore()
	f := newFixture(t, jobs.Config{Store: fs})
	if err := f.runner.Register("noop", okHandler("payload")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, err := f.runner.Submit("noop", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	awaitJob(t, succeeded, id)

	persisted := waitForStoreState(t, fs, id, jobs.StateSucceeded)
	if persisted.Kind != "noop" {
		t.Errorf("persisted kind = %q, want noop", persisted.Kind)
	}
	if persisted.Result != "payload" {
		t.Errorf("persisted result = %v, want payload", persisted.Result)
	}
}

// TestRunnerGetFallsBackToStore proves a Get miss in memory is served from the
// store — the post-restart / post-eviction retrieval path.
func TestRunnerGetFallsBackToStore(t *testing.T) {
	t.Parallel()

	fs := newFakeStore()
	fs.jobs["ghost"] = jobs.Job{
		ID: "ghost", Kind: "speedtest", State: jobs.StateSucceeded,
		Progress: 1, Result: "from-store",
	}
	f := newFixture(t, jobs.Config{Store: fs})

	got, ok := f.runner.Get("ghost")
	if !ok {
		t.Fatal("Get returned !ok; expected fallback to store")
	}
	if got.State != jobs.StateSucceeded || got.Result != "from-store" {
		t.Errorf("got %+v, want succeeded/from-store", got)
	}
}

// TestRunnerGetStoreMissReturnsFalse: unknown id, store also misses.
func TestRunnerGetStoreMissReturnsFalse(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{Store: newFakeStore()})
	if _, ok := f.runner.Get("nope"); ok {
		t.Error("Get returned ok for an id absent from memory and store")
	}
}

// TestRunnerRecoverMarksInterrupted: jobs left non-terminal by a prior process
// are transitioned to failed at startup.
func TestRunnerRecoverMarksInterrupted(t *testing.T) {
	t.Parallel()

	fs := newFakeStore()
	fs.jobs["a"] = jobs.Job{ID: "a", Kind: "k", State: jobs.StateRunning}
	fs.jobs["b"] = jobs.Job{ID: "b", Kind: "k", State: jobs.StateQueued}
	fs.jobs["c"] = jobs.Job{ID: "c", Kind: "k", State: jobs.StateSucceeded}
	f := newFixture(t, jobs.Config{Store: fs})

	n, err := f.runner.Recover(context.Background())
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if n != 2 {
		t.Errorf("recovered = %d, want 2 (the running + queued)", n)
	}
	if j, _ := fs.snapshot("c"); j.State != jobs.StateSucceeded {
		t.Errorf("terminal job c mutated to %q", j.State)
	}
}

// TestRunnerStoreSaveErrorDoesNotFailJob proves persistence is best-effort: a
// store that always errors must not stop a job from running to success.
func TestRunnerStoreSaveErrorDoesNotFailJob(t *testing.T) {
	t.Parallel()

	fs := newFakeStore()
	fs.failSave = true
	f := newFixture(t, jobs.Config{Store: fs})
	if err := f.runner.Register("noop", okHandler("ok")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	succeeded := f.watch(t, jobs.StateSucceeded)

	id, err := f.runner.Submit("noop", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	done := awaitJob(t, succeeded, id)
	if done.State != jobs.StateSucceeded {
		t.Errorf("state = %q, want succeeded despite store errors", done.State)
	}
}

// TestRunnerNilStoreRecoverNoOp: Recover is a no-op without a store.
func TestRunnerNilStoreRecoverNoOp(t *testing.T) {
	t.Parallel()

	f := newFixture(t, jobs.Config{})
	n, err := f.runner.Recover(context.Background())
	if err != nil || n != 0 {
		t.Errorf("Recover with nil store = (%d, %v), want (0, nil)", n, err)
	}
}
