package devices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/devices"
)

// fakeEngine drives the use-case in tests: availability/in-progress are
// togglable, and the scan hook records the options it was called with.
type fakeEngine struct {
	available  bool
	scanning   bool
	devices    []*discovery.DiscoveredDevice
	stats      *discovery.EngineStats
	lastScan   *discovery.ScanResult
	caps       map[string]bool
	device     *discovery.DiscoveredDevice
	scanResult *discovery.ScanResult
	scanErr    error
	scanOpts   *discovery.ScanOptions
	subID      string
	unsubbed   string
}

func (f *fakeEngine) Available() bool                            { return f.available }
func (f *fakeEngine) Scanning() bool                             { return f.scanning }
func (f *fakeEngine) Devices() []*discovery.DiscoveredDevice     { return f.devices }
func (f *fakeEngine) Device(string) *discovery.DiscoveredDevice  { return f.device }
func (f *fakeEngine) Stats() *discovery.EngineStats              { return f.stats }
func (f *fakeEngine) LastScan() *discovery.ScanResult            { return f.lastScan }
func (f *fakeEngine) Capabilities() map[string]bool              { return f.caps }
func (f *fakeEngine) SubscribeAll(func(*discovery.Event)) string { return f.subID }
func (f *fakeEngine) Unsubscribe(id string)                      { f.unsubbed = id }

func (f *fakeEngine) Scan(
	_ context.Context, opts *discovery.ScanOptions,
) (*discovery.ScanResult, error) {
	f.scanOpts = opts
	return f.scanResult, f.scanErr
}

func TestSnapshotUnavailable(t *testing.T) {
	t.Parallel()
	svc := devices.NewService(&fakeEngine{available: false})
	if _, err := svc.Snapshot(); !errors.Is(err, devices.ErrUnavailable) {
		t.Fatalf("Snapshot() err = %v, want ErrUnavailable", err)
	}
}

func TestSnapshotReturnsInventory(t *testing.T) {
	t.Parallel()
	eng := &fakeEngine{
		available: true,
		devices:   []*discovery.DiscoveredDevice{{}},
		stats:     &discovery.EngineStats{},
		lastScan:  &discovery.ScanResult{},
		caps:      map[string]bool{"wired": true},
	}
	svc := devices.NewService(eng)

	snap, err := svc.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() err = %v", err)
	}
	if len(snap.Devices) != 1 || snap.Stats == nil || snap.ScanResult == nil || !snap.Capabilities["wired"] {
		t.Fatalf("Snapshot() = %+v, missing inventory fields", snap)
	}
}

func TestScanGuardsInOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		available bool
		scanning  bool
		want      error
	}{
		{"unavailable beats in-progress", false, true, devices.ErrUnavailable},
		{"in-progress when available", true, true, devices.ErrScanInProgress},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := devices.NewService(&fakeEngine{available: tt.available, scanning: tt.scanning})
			if _, err := svc.Scan(context.Background(), &discovery.ScanOptions{}); !errors.Is(err, tt.want) {
				t.Fatalf("Scan() err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestScanSurfacesEngineError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	svc := devices.NewService(&fakeEngine{available: true, scanErr: sentinel})
	if _, err := svc.Scan(context.Background(), &discovery.ScanOptions{}); !errors.Is(err, sentinel) {
		t.Fatalf("Scan() err = %v, want the engine error verbatim", err)
	}
}

func TestScanReturnsFreshResult(t *testing.T) {
	t.Parallel()
	fresh := &discovery.ScanResult{}
	eng := &fakeEngine{
		available:  true,
		lastScan:   &discovery.ScanResult{}, // different pointer than the fresh result
		scanResult: fresh,
	}
	svc := devices.NewService(eng)

	snap, err := svc.Scan(context.Background(), &discovery.ScanOptions{})
	if err != nil {
		t.Fatalf("Scan() err = %v", err)
	}
	if snap.ScanResult != fresh {
		t.Fatalf("Scan() ScanResult = %p, want the freshly produced result %p", snap.ScanResult, fresh)
	}
}

func TestQuickAndFullScanUseEngineDefaults(t *testing.T) {
	t.Parallel()
	eng := &fakeEngine{available: true, scanResult: &discovery.ScanResult{}}
	svc := devices.NewService(eng)

	if _, err := svc.QuickScan(context.Background()); err != nil {
		t.Fatalf("QuickScan() err = %v", err)
	}
	if eng.scanOpts == nil {
		t.Fatal("QuickScan() did not pass scan options to the engine")
	}

	if _, err := svc.FullScan(context.Background()); err != nil {
		t.Fatalf("FullScan() err = %v", err)
	}
	if eng.scanOpts == nil {
		t.Fatal("FullScan() did not pass scan options to the engine")
	}
}

func TestDeviceNotFound(t *testing.T) {
	t.Parallel()
	svc := devices.NewService(&fakeEngine{available: true, device: nil})
	if _, err := svc.Device("AA:BB:CC:DD:EE:FF"); !errors.Is(err, devices.ErrDeviceNotFound) {
		t.Fatalf("Device() err = %v, want ErrDeviceNotFound", err)
	}
}

func TestDeviceFound(t *testing.T) {
	t.Parallel()
	want := &discovery.DiscoveredDevice{}
	svc := devices.NewService(&fakeEngine{available: true, device: want})
	got, err := svc.Device("AA:BB:CC:DD:EE:FF")
	if err != nil || got != want {
		t.Fatalf("Device() = %v, %v; want the device, nil", got, err)
	}
}

func TestSubscribeUnavailable(t *testing.T) {
	t.Parallel()
	svc := devices.NewService(&fakeEngine{available: false})
	if _, err := svc.Subscribe(func(*discovery.Event) {}); !errors.Is(err, devices.ErrUnavailable) {
		t.Fatalf("Subscribe() err = %v, want ErrUnavailable", err)
	}
}

func TestSubscribeReturnsIDAndUnsubscribes(t *testing.T) {
	t.Parallel()
	eng := &fakeEngine{available: true, subID: "sub-1"}
	svc := devices.NewService(eng)

	id, err := svc.Subscribe(func(*discovery.Event) {})
	if err != nil || id != "sub-1" {
		t.Fatalf("Subscribe() = %q, %v; want sub-1, nil", id, err)
	}
	svc.Unsubscribe(id)
	if eng.unsubbed != "sub-1" {
		t.Fatalf("Unsubscribe propagated %q, want sub-1", eng.unsubbed)
	}
}

func TestScanningProbeRequiresAvailable(t *testing.T) {
	t.Parallel()
	// A scanning-but-unavailable engine reports not-scanning: the probe must not
	// claim a scan is running when there is no engine to run it.
	svc := devices.NewService(&fakeEngine{available: false, scanning: true})
	if svc.Scanning() {
		t.Fatal("Scanning() = true for an unavailable engine, want false")
	}
}
