package api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/diagnostics/iperf"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// fakeIperfClient is a controllable iperfClient for the kind tests: it never
// launches the iperf3 binary.
type fakeIperfClient struct {
	result   *iperf.Result
	err      error
	progress float64
	release  chan struct{} // if non-nil, RunClient blocks until closed or ctx is done
}

func (f *fakeIperfClient) GetClientStatus() iperf.ClientStatus {
	return iperf.ClientStatus{Running: true, Progress: f.progress}
}

func (f *fakeIperfClient) RunClient(ctx context.Context, _ *iperf.ClientConfig) (*iperf.Result, error) {
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

// iperfParams builds valid job params for a client run (the server value only
// needs to pass validation; results come from the fake).
func iperfParams() json.RawMessage {
	return json.RawMessage(`{"server":"10.0.0.2","protocol":"tcp","direction":"upload","duration":5}`)
}

func TestIperfKindRunsToSuccess(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeIperfClient{
		result:   &iperf.Result{Bandwidth: 940, Protocol: "tcp", Direction: "upload", Server: "10.0.0.2"},
		progress: 100,
	}
	if err := runner.Register(iperfJobKind, newIperfHandler(func() iperfClient { return fake })); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(iperfJobKind, iperfParams())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})

	j, _ := runner.Get(id)
	res, ok := j.Result.(IperfResultResponse)
	if !ok {
		t.Fatalf("Result type = %T, want IperfResultResponse", j.Result)
	}
	if res.Bandwidth != 940 || res.Protocol != "tcp" || res.Server != "10.0.0.2" {
		t.Fatalf("result = %+v, want bandwidth 940 / tcp / server 10.0.0.2", res)
	}
	if j.Progress != 1 {
		t.Fatalf("progress = %v, want 1", j.Progress)
	}
}

func TestIperfKindReportsProgress(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	release := make(chan struct{})
	defer close(release)
	fake := &fakeIperfClient{result: &iperf.Result{}, progress: 60, release: release}
	_ = runner.Register(iperfJobKind, newIperfHandler(func() iperfClient { return fake }))

	id, _ := runner.Submit(iperfJobKind, iperfParams())
	waitFor(t, "progress sampled to 0.6", func() bool {
		j, ok := runner.Get(id)
		return ok && j.Progress == 0.6
	})
}

func TestIperfKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeIperfClient{release: make(chan struct{})} // never closed; cancel via ctx
	_ = runner.Register(iperfJobKind, newIperfHandler(func() iperfClient { return fake }))

	id, _ := runner.Submit(iperfJobKind, iperfParams())
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

func TestIperfKindFailure(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeIperfClient{err: errors.New("connection refused")}
	_ = runner.Register(iperfJobKind, newIperfHandler(func() iperfClient { return fake }))

	id, _ := runner.Submit(iperfJobKind, iperfParams())
	waitFor(t, "job failed", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
	j, _ := runner.Get(id)
	if j.Err == "" {
		t.Fatal("failed iperf job has empty Err, want the client error")
	}
}

func TestIperfKindRejectsMissingServer(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeIperfClient{result: &iperf.Result{}}
	_ = runner.Register(iperfJobKind, newIperfHandler(func() iperfClient { return fake }))

	// Empty params (no server) must fail validation inside the handler -> failed
	// job, not a panic or a success.
	id, _ := runner.Submit(iperfJobKind, json.RawMessage(`{}`))
	waitFor(t, "job failed on missing server", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
}

func TestRegisterIperfKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeIperfClient{result: &iperf.Result{Bandwidth: 100}, progress: 100}
	srv.registerIperfKind(func() iperfClient { return fake })

	id, err := runner.Submit(iperfJobKind, iperfParams())
	if err != nil {
		t.Fatalf("Submit after registerIperfKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	res, ok := j.Result.(IperfResultResponse)
	if !ok || res.Bandwidth != 100 {
		t.Fatalf("result = %+v (ok=%v), want bandwidth 100", res, ok)
	}
}
