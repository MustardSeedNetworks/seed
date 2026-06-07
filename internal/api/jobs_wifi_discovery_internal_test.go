package api

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

type fakeWiFiDiscoveryBridge struct {
	result  *discovery.WiFiScanResult
	err     error
	release chan struct{}
}

func (f *fakeWiFiDiscoveryBridge) Scan(ctx context.Context) (*discovery.WiFiScanResult, error) {
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

func TestWiFiDiscoveryScanKindRunsToSuccess(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeWiFiDiscoveryBridge{result: &discovery.WiFiScanResult{Interface: "wlan0"}}
	handler := newWiFiDiscoveryScanHandler(func() wifiDiscoveryBridge { return fake })
	if err := runner.Register(wifiDiscoveryScanJobKind, handler); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(wifiDiscoveryScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	res, ok := j.Result.(WiFiDiscoveryScanResponse)
	if !ok || res.Interface != "wlan0" {
		t.Fatalf("result = %+v (ok=%v), want interface wlan0", res, ok)
	}
}

func TestWiFiDiscoveryScanKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeWiFiDiscoveryBridge{release: make(chan struct{})}
	handler := newWiFiDiscoveryScanHandler(func() wifiDiscoveryBridge { return fake })
	_ = runner.Register(wifiDiscoveryScanJobKind, handler)

	id, _ := runner.Submit(wifiDiscoveryScanJobKind, nil)
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

func TestWiFiDiscoveryScanKindFailure(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeWiFiDiscoveryBridge{err: errors.New("no wifi interface")}
	handler := newWiFiDiscoveryScanHandler(func() wifiDiscoveryBridge { return fake })
	_ = runner.Register(wifiDiscoveryScanJobKind, handler)

	id, _ := runner.Submit(wifiDiscoveryScanJobKind, nil)
	waitFor(t, "job failed", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateFailed
	})
	j, _ := runner.Get(id)
	if j.Err == "" {
		t.Fatal("failed wifi-discovery-scan job has empty Err")
	}
}

func TestRegisterWiFiDiscoveryScanKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeWiFiDiscoveryBridge{result: &discovery.WiFiScanResult{Interface: "wlan0"}}
	srv.registerWiFiDiscoveryScanKind(func() wifiDiscoveryBridge { return fake })

	id, err := runner.Submit(wifiDiscoveryScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit after registerWiFiDiscoveryScanKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
}
