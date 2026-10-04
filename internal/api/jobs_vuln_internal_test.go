package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// fakeVulnScanService is a controllable vulnScanService for the kind tests: it
// never touches the real scanner or device registry.
type fakeVulnScanService struct {
	devices []*discovery.DiscoveredDevice
	results []*discovery.DeviceVulnerabilities
	scanErr error
	gate    chan struct{} // if non-nil, each ScanDevice waits for one token (or ctx)
}

func (f *fakeVulnScanService) Devices(targetIP string) []*discovery.DiscoveredDevice {
	if targetIP == "" {
		return f.devices
	}
	for _, d := range f.devices {
		if d.IP == targetIP {
			return []*discovery.DiscoveredDevice{d}
		}
	}
	return nil
}

func (f *fakeVulnScanService) ScanDevice(
	ctx context.Context, device *discovery.DiscoveredDevice,
) (*discovery.DeviceVulnerabilities, error) {
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.scanErr != nil {
		return nil, f.scanErr
	}
	return &discovery.DeviceVulnerabilities{DeviceIP: device.IP}, nil
}

func (f *fakeVulnScanService) Results() []*discovery.DeviceVulnerabilities {
	return f.results
}

func devicesAt(ips ...string) []*discovery.DiscoveredDevice {
	out := make([]*discovery.DiscoveredDevice, len(ips))
	for i, ip := range ips {
		out[i] = &discovery.DiscoveredDevice{IP: ip}
	}
	return out
}

func TestVulnScanKindScansAllDevices(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeVulnScanService{
		devices: devicesAt("10.0.0.1", "10.0.0.2", "10.0.0.3"),
		results: []*discovery.DeviceVulnerabilities{{DeviceIP: "10.0.0.1"}, {DeviceIP: "10.0.0.2"}},
	}
	if err := runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake })); err != nil {
		t.Fatalf("Register: %v", err)
	}

	id, err := runner.Submit(vulnScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})

	j, _ := runner.Get(id)
	res, ok := j.Result.(VulnScanJobResult)
	if !ok {
		t.Fatalf("Result type = %T, want VulnScanJobResult", j.Result)
	}
	if res.Scanned != 3 || res.Failed != 0 || res.Count != 2 {
		t.Fatalf("result = %+v, want scanned 3 / failed 0 / count 2", res)
	}
}

func TestVulnScanKindTargetsSingleDevice(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeVulnScanService{devices: devicesAt("10.0.0.1", "10.0.0.2")}
	_ = runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake }))

	id, _ := runner.Submit(vulnScanJobKind, json.RawMessage(`{"ip":"10.0.0.2"}`))
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	if res := j.Result.(VulnScanJobResult); res.Scanned != 1 {
		t.Fatalf("scanned = %d, want 1 (single-device target)", res.Scanned)
	}
}

// A request the kind does not understand must fail, not fall back to scanning
// every device. The UI posted {"targets":[ip]} for months and each auto-trigger
// scanned the whole estate (#2960).
func TestVulnScanKindRefusesParamsItCannotHonour(t *testing.T) {
	t.Parallel()

	for _, params := range []string{
		`{"targets":["10.0.0.2"]}`,
		`{"ip":"not-an-ip"}`,
	} {
		t.Run(params, func(t *testing.T) {
			t.Parallel()

			_, runner := newJobsTestServer(t, jobs.Config{})
			fake := &fakeVulnScanService{devices: devicesAt("10.0.0.1", "10.0.0.2")}
			_ = runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake }))

			id, err := runner.Submit(vulnScanJobKind, json.RawMessage(params))
			if err != nil {
				t.Fatalf("Submit: %v", err)
			}
			waitFor(t, "job reaches a terminal state", func() bool {
				j, ok := runner.Get(id)
				return ok && (j.State == jobs.StateFailed || j.State == jobs.StateSucceeded)
			})
			if j, _ := runner.Get(id); j.State != jobs.StateFailed {
				t.Fatalf("state = %s, result = %+v; want failed with nothing scanned", j.State, j.Result)
			}
		})
	}
}

// The scan is sold at Pro (compliance_advanced). It arrives through POST /jobs,
// which every tier reaches, so the kind is the boundary. Before #2960 only the
// retired legacy route carried the gate, and the job kind was open to Free.
func TestVulnScanKindRequiresComplianceAdvanced(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, key string
		gated     bool
	}{
		{"free", "", true},
		{"starter", prodSeedStarterVector, true},
		{"pro", prodSeedProVector, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := newLicensedJobsServer(t, tc.key)
			srv.registerVulnScanKind(func() vulnScanService { return &fakeVulnScanService{} })
			body, _ := json.Marshal(CreateJobRequest{
				Kind:   vulnScanJobKind,
				Params: json.RawMessage(`{"ip":"10.0.0.2"}`),
			})
			w := httptest.NewRecorder()
			srv.handleJobs(w, httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/jobs", bytes.NewReader(body)))

			if !tc.gated {
				if w.Code != http.StatusCreated {
					t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
				}
				return
			}
			if w.Code != http.StatusPaymentRequired {
				t.Fatalf("status = %d, want 402; body=%s", w.Code, w.Body.String())
			}
			var gate FeatureGateResponse
			if err := json.NewDecoder(w.Body).Decode(&gate); err != nil {
				t.Fatalf("decode 402 body: %v", err)
			}
			if gate.RequiredFeature != vulnScanFeature {
				t.Errorf("requiredFeature = %q, want %q", gate.RequiredFeature, vulnScanFeature)
			}
		})
	}
}

func TestVulnScanKindReportsProgress(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	gate := make(chan struct{})
	fake := &fakeVulnScanService{devices: devicesAt("10.0.0.1", "10.0.0.2"), gate: gate}
	_ = runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake }))

	id, _ := runner.Submit(vulnScanJobKind, nil)
	gate <- struct{}{} // let the first of two devices complete -> progress 0.5
	waitFor(t, "progress 0.5 after first device", func() bool {
		j, ok := runner.Get(id)
		return ok && j.Progress == 0.5
	})
	gate <- struct{}{} // release the second device
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
}

func TestVulnScanKindCancellation(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	// gate never receives tokens: ScanDevice blocks until the job ctx is cancelled.
	fake := &fakeVulnScanService{devices: devicesAt("10.0.0.1"), gate: make(chan struct{})}
	_ = runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake }))

	id, _ := runner.Submit(vulnScanJobKind, nil)
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

func TestVulnScanKindCountsPerDeviceFailures(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	// Every device errors (but not via cancellation), so the scan completes with
	// a non-zero failed count rather than failing the whole job.
	fake := &fakeVulnScanService{devices: devicesAt("10.0.0.1", "10.0.0.2"), scanErr: errors.New("offline")}
	_ = runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake }))

	id, _ := runner.Submit(vulnScanJobKind, nil)
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	res := j.Result.(VulnScanJobResult)
	if res.Failed != 2 || res.Scanned != 0 {
		t.Fatalf("result = %+v, want failed 2 / scanned 0", res)
	}
}

func TestVulnScanKindNoDevices(t *testing.T) {
	t.Parallel()

	_, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeVulnScanService{} // no devices
	_ = runner.Register(vulnScanJobKind, newVulnScanHandler(func() vulnScanService { return fake }))

	id, _ := runner.Submit(vulnScanJobKind, nil)
	waitFor(t, "job succeeded with no devices", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
	j, _ := runner.Get(id)
	if res := j.Result.(VulnScanJobResult); res.Scanned != 0 || res.Count != 0 {
		t.Fatalf("result = %+v, want empty", res)
	}
}

func TestRegisterVulnScanKindWiresRunner(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	fake := &fakeVulnScanService{devices: devicesAt("10.0.0.1")}
	srv.registerVulnScanKind(func() vulnScanService { return fake })

	id, err := runner.Submit(vulnScanJobKind, nil)
	if err != nil {
		t.Fatalf("Submit after registerVulnScanKind: %v", err)
	}
	waitFor(t, "job succeeded", func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == jobs.StateSucceeded
	})
}
