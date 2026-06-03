package api

import (
	"context"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/platform/jobs"
	"github.com/krisarmstrong/seed/internal/services/discovery"
)

type fakeBluetoothScanner struct {
	result  *discovery.BluetoothScanResult
	err     error
	release chan struct{}
}

func (f *fakeBluetoothScanner) Scan(ctx context.Context) (*discovery.BluetoothScanResult, error) {
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

func (f *fakeBluetoothScanner) GetStats() *discovery.BluetoothDiscoveryStats {
	return &discovery.BluetoothDiscoveryStats{}
}

func TestBluetoothScanKindRunsToSuccess(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeBluetoothScanner{result: &discovery.BluetoothScanResult{ScanType: "le", AdapterName: "hci0"}}
	handler := newBluetoothScanHandler(func() bluetoothScannerService { return fake })
	if err := runner.Register(bluetoothScanJobKind, handler); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(bluetoothScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	res, ok := j.Result.(BluetoothScanResponse)
	if !ok || res.ScanType != "le" || res.AdapterName != "hci0" {
		t.Fatalf("result = %+v (ok=%v), want scanType le / adapter hci0", res, ok)
	}
}

func TestBluetoothScanKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeBluetoothScanner{release: make(chan struct{})}
	handler := newBluetoothScanHandler(func() bluetoothScannerService { return fake })
	_ = runner.Register(bluetoothScanJobKind, handler)

	id, _ := runner.Submit(bluetoothScanJobKind, nil)
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

func TestBluetoothScanKindFailure(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeBluetoothScanner{err: errors.New("no adapter")}
	handler := newBluetoothScanHandler(func() bluetoothScannerService { return fake })
	_ = runner.Register(bluetoothScanJobKind, handler)

	id, _ := runner.Submit(bluetoothScanJobKind, nil)
	waitFor(t, "job failed", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
	j, _ := runner.Get(id)
	if j.Err == "" {
		t.Fatal("failed bluetooth-scan job has empty Err")
	}
}

func TestRegisterBluetoothScanKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeBluetoothScanner{result: &discovery.BluetoothScanResult{ScanType: "le"}}
	srv.registerBluetoothScanKind(func() bluetoothScannerService { return fake })

	id, err := runner.Submit(bluetoothScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit after registerBluetoothScanKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
}
