package api

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

type fakeDeviceScanService struct {
	devices []*discovery.DiscoveredDevice
	err     error
	release chan struct{}
}

func (f *fakeDeviceScanService) Scan(ctx context.Context) error {
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.err
}

func (f *fakeDeviceScanService) GetDevices() []*discovery.DiscoveredDevice { return f.devices }
func (f *fakeDeviceScanService) Count() int                                { return len(f.devices) }

func TestDeviceScanKindRunsToSuccess(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeDeviceScanService{devices: devicesAt("10.0.0.1", "10.0.0.2", "10.0.0.3")}
	handler := newDeviceScanHandler(func() deviceScanService { return fake })
	if err := runner.Register(deviceScanJobKind, handler); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(deviceScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	res, ok := j.Result.(DeviceScanJobResult)
	if !ok || res.Count != 3 || len(res.Devices) != 3 {
		t.Fatalf("result = %+v (ok=%v), want count 3", res, ok)
	}
}

func TestDeviceScanKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeDeviceScanService{release: make(chan struct{})}
	handler := newDeviceScanHandler(func() deviceScanService { return fake })
	_ = runner.Register(deviceScanJobKind, handler)

	id, _ := runner.Submit(deviceScanJobKind, nil)
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

func TestDeviceScanKindFailure(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeDeviceScanService{err: errors.New("scan failed")}
	handler := newDeviceScanHandler(func() deviceScanService { return fake })
	_ = runner.Register(deviceScanJobKind, handler)

	id, _ := runner.Submit(deviceScanJobKind, nil)
	waitFor(t, "job failed", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
	j, _ := runner.Get(id)
	if j.Err == "" {
		t.Fatal("failed device-scan job has empty Err")
	}
}

func TestRegisterDeviceScanKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeDeviceScanService{devices: devicesAt("10.0.0.9")}
	srv.registerDeviceScanKind(func() deviceScanService { return fake })

	id, err := runner.Submit(deviceScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit after registerDeviceScanKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
}
