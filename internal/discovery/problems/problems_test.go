package problems_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/problems"
)

type fakeDetector struct {
	available  bool
	active     []discovery.NetworkProblem
	summary    *discovery.ProblemSummary
	result     *discovery.ProblemDetectionResult
	scanErr    error
	scanned    []*discovery.DiscoveredDevice
	thresholds discovery.ProblemThresholds
	setCalled  bool
}

func (f *fakeDetector) Available() bool                            { return f.available }
func (f *fakeDetector) ActiveProblems() []discovery.NetworkProblem { return f.active }
func (f *fakeDetector) Summary() *discovery.ProblemSummary         { return f.summary }
func (f *fakeDetector) Thresholds() discovery.ProblemThresholds    { return f.thresholds }

func (f *fakeDetector) Scan(
	_ context.Context, devs []*discovery.DiscoveredDevice,
) (*discovery.ProblemDetectionResult, error) {
	f.scanned = devs
	return f.result, f.scanErr
}

func (f *fakeDetector) SetThresholds(t discovery.ProblemThresholds) {
	f.setCalled = true
	f.thresholds = t
}

type fakeDeviceSource struct {
	devices []*discovery.DiscoveredDevice
}

func (f fakeDeviceSource) Devices() []*discovery.DiscoveredDevice { return f.devices }

func TestUnavailablePaths(t *testing.T) {
	t.Parallel()
	svc := problems.NewService(&fakeDetector{available: false}, fakeDeviceSource{})

	if _, err := svc.Active(); !errors.Is(err, problems.ErrUnavailable) {
		t.Errorf("Active() err = %v, want ErrUnavailable", err)
	}
	if _, err := svc.Scan(context.Background()); !errors.Is(err, problems.ErrUnavailable) {
		t.Errorf("Scan() err = %v, want ErrUnavailable", err)
	}
	if _, err := svc.Thresholds(); !errors.Is(err, problems.ErrUnavailable) {
		t.Errorf("Thresholds() err = %v, want ErrUnavailable", err)
	}
	if err := svc.SetThresholds(discovery.ProblemThresholds{}); !errors.Is(err, problems.ErrUnavailable) {
		t.Errorf("SetThresholds() err = %v, want ErrUnavailable", err)
	}
}

func TestActiveReturnsProblems(t *testing.T) {
	t.Parallel()
	det := &fakeDetector{
		available: true,
		active:    []discovery.NetworkProblem{{}, {}},
		summary:   &discovery.ProblemSummary{},
	}
	svc := problems.NewService(det, fakeDeviceSource{})

	active, err := svc.Active()
	if err != nil {
		t.Fatalf("Active() err = %v", err)
	}
	if len(active.Problems) != 2 || active.Summary == nil {
		t.Fatalf("Active() = %+v, want 2 problems + summary", active)
	}
}

func TestScanFeedsDiscoveredDevices(t *testing.T) {
	t.Parallel()
	devs := []*discovery.DiscoveredDevice{{}, {}, {}}
	det := &fakeDetector{available: true, result: &discovery.ProblemDetectionResult{}}
	svc := problems.NewService(det, fakeDeviceSource{devices: devs})

	if _, err := svc.Scan(context.Background()); err != nil {
		t.Fatalf("Scan() err = %v", err)
	}
	if len(det.scanned) != len(devs) {
		t.Fatalf("Scan() ran over %d devices, want %d from the source", len(det.scanned), len(devs))
	}
}

func TestSetThresholdsApplied(t *testing.T) {
	t.Parallel()
	det := &fakeDetector{available: true}
	svc := problems.NewService(det, fakeDeviceSource{})

	if err := svc.SetThresholds(discovery.ProblemThresholds{}); err != nil {
		t.Fatalf("SetThresholds() err = %v", err)
	}
	if !det.setCalled {
		t.Fatal("SetThresholds() did not reach the detector")
	}
}
