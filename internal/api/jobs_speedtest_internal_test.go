package api

import (
	"context"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/diagnostics/speedtest"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// fakeSpeedTester is a controllable speedTester for the kind tests: it never
// touches the network.
type fakeSpeedTester struct {
	result   *speedtest.Result
	err      error
	progress float64
	release  chan struct{} // if non-nil, RunTest blocks until closed or ctx is done
}

func (f *fakeSpeedTester) GetStatus() speedtest.Status {
	return speedtest.Status{Running: true, Progress: f.progress}
}

func (f *fakeSpeedTester) RunTest(ctx context.Context) (*speedtest.Result, error) {
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestSpeedtestKindRunsToSuccess(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeSpeedTester{
		result:   &speedtest.Result{Download: 100, Upload: 20, Latency: 5, Server: "s1"},
		progress: 100,
	}
	if err := runner.Register(speedtestJobKind, newSpeedtestHandler(func() speedTester { return fake })); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(speedtestJobKind, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})

	j, _ := runner.Get(id)
	res, ok := j.Result.(SpeedtestResponse)
	if !ok {
		t.Fatalf("Result type = %T, want SpeedtestResponse", j.Result)
	}
	if res.Download != 100 || res.Upload != 20 || res.Server != "s1" {
		t.Fatalf("result = %+v, want download 100 / upload 20 / server s1", res)
	}
	if j.Progress != 1 {
		t.Fatalf("progress = %v, want 1", j.Progress)
	}
}

func TestSpeedtestKindReportsProgress(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	release := make(chan struct{})
	defer close(release)
	fake := &fakeSpeedTester{result: &speedtest.Result{}, progress: 50, release: release}
	_ = runner.Register(speedtestJobKind, newSpeedtestHandler(func() speedTester { return fake }))

	id, _ := runner.Submit(speedtestJobKind, nil)
	// The handler samples the tester's phase progress (50/100) onto the job.
	waitFor(t, "progress sampled to 0.5", func() bool {
		j, ok := runner.Get(id)
		return ok && j.Progress == 0.5
	})
}

func TestSpeedtestKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	// release is never closed: RunTest blocks until the job context is cancelled.
	fake := &fakeSpeedTester{release: make(chan struct{})}
	_ = runner.Register(speedtestJobKind, newSpeedtestHandler(func() speedTester { return fake }))

	id, _ := runner.Submit(speedtestJobKind, nil)
	waitFor(t, "job running", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateRunning
	})

	if err := runner.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	waitFor(t, "job cancelled", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateCancelled
	})
}

func TestSpeedtestKindFailure(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeSpeedTester{err: errors.New("network down")}
	_ = runner.Register(speedtestJobKind, newSpeedtestHandler(func() speedTester { return fake }))

	id, _ := runner.Submit(speedtestJobKind, nil)
	waitFor(t, "job failed", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
	j, _ := runner.Get(id)
	if j.Err == "" {
		t.Fatal("failed speedtest job has empty Err, want the tester error")
	}
}

// TestRegisterSpeedtestKindWiresRunner exercises the Server-level registration
// path with an injected fake, proving registerSpeedtestKind makes the kind
// runnable on the server's own runner without hitting the network.
func TestRegisterSpeedtestKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeSpeedTester{result: &speedtest.Result{Download: 42}, progress: 100}
	srv.registerSpeedtestKind(func() speedTester { return fake })

	id, err := runner.Submit(speedtestJobKind, nil)
	if err != nil {
		t.Fatalf("Submit after registerSpeedtestKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	res, ok := j.Result.(SpeedtestResponse)
	if !ok || res.Download != 42 {
		t.Fatalf("result = %+v (ok=%v), want download 42", res, ok)
	}
}
