package api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// fakeEngineScanner is a controllable engineScanner for the kind tests: it never
// runs the real discovery engine.
type fakeEngineScanner struct {
	result  *discovery.ScanResult
	err     error
	devices []*discovery.DiscoveredDevice
	gotOpts *discovery.ScanOptions // captures the opts Scan was called with
	release chan struct{}          // if non-nil, Scan blocks until closed or ctx is done
}

func (f *fakeEngineScanner) Scan(
	ctx context.Context, opts *discovery.ScanOptions,
) (*discovery.ScanResult, error) {
	f.gotOpts = opts
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

func (f *fakeEngineScanner) GetDevices() []*discovery.DiscoveredDevice { return f.devices }
func (f *fakeEngineScanner) GetStats() *discovery.EngineStats          { return &discovery.EngineStats{} }
func (f *fakeEngineScanner) GetCapabilities() map[string]bool          { return map[string]bool{"wired": true} }

func TestEngineScanKindRunsToSuccess(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeEngineScanner{
		result:  &discovery.ScanResult{},
		devices: devicesAt("10.0.0.1", "10.0.0.2"),
	}
	handler := newEngineScanHandler(func() engineScanner { return fake })
	if err := runner.Register(engineScanJobKind, handler); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(engineScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})

	j, _ := runner.Get(id)
	res, ok := j.Result.(EngineDiscoveryResponse)
	if !ok {
		t.Fatalf("Result type = %T, want EngineDiscoveryResponse", j.Result)
	}
	if len(res.Devices) != 2 || res.Capabilities["wired"] != true {
		t.Fatalf("result = %+v, want 2 devices + wired capability", res)
	}
}

func TestEngineScanKindAppliesParams(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeEngineScanner{result: &discovery.ScanResult{}}
	_ = runner.Register(engineScanJobKind, newEngineScanHandler(func() engineScanner { return fake }))

	id, _ := runner.Submit(engineScanJobKind, json.RawMessage(`{"scanType":"full","includeVulnScan":true}`))
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	if fake.gotOpts == nil || !fake.gotOpts.IncludeVulnScan {
		t.Fatalf("Scan opts = %+v, want IncludeVulnScan true from params", fake.gotOpts)
	}
}

func TestEngineScanKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeEngineScanner{release: make(chan struct{})} // never closed; cancel via ctx
	_ = runner.Register(engineScanJobKind, newEngineScanHandler(func() engineScanner { return fake }))

	id, _ := runner.Submit(engineScanJobKind, nil)
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

func TestEngineScanKindFailure(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeEngineScanner{err: errors.New("scan failed")}
	_ = runner.Register(engineScanJobKind, newEngineScanHandler(func() engineScanner { return fake }))

	id, _ := runner.Submit(engineScanJobKind, nil)
	waitFor(t, "job failed", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
	j, _ := runner.Get(id)
	if j.Err == "" {
		t.Fatal("failed engine-scan job has empty Err, want the engine error")
	}
}

func TestEngineScanKindRejectsMalformedParams(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeEngineScanner{result: &discovery.ScanResult{}}
	_ = runner.Register(engineScanJobKind, newEngineScanHandler(func() engineScanner { return fake }))

	id, _ := runner.Submit(engineScanJobKind, json.RawMessage(`{not json`))
	waitFor(t, "job failed on malformed params", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
}

func TestRegisterEngineScanKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeEngineScanner{result: &discovery.ScanResult{}, devices: devicesAt("10.0.0.9")}
	srv.registerEngineScanKind(func() engineScanner { return fake })

	id, err := runner.Submit(engineScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit after registerEngineScanKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
}
