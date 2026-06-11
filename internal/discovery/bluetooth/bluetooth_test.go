package bluetooth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/bluetooth"
)

type fakeScanner struct {
	available bool
	result    *discovery.BluetoothScanResult
	scanErr   error
	last      *discovery.BluetoothScanResult
	stats     *discovery.BluetoothDiscoveryStats
}

func (f *fakeScanner) Available() bool                           { return f.available }
func (f *fakeScanner) LastScan() *discovery.BluetoothScanResult  { return f.last }
func (f *fakeScanner) Stats() *discovery.BluetoothDiscoveryStats { return f.stats }
func (f *fakeScanner) Scan(context.Context) (*discovery.BluetoothScanResult, error) {
	return f.result, f.scanErr
}

func TestUnavailablePaths(t *testing.T) {
	t.Parallel()
	svc := bluetooth.NewService(&fakeScanner{available: false})

	if _, err := svc.Scan(context.Background()); !errors.Is(err, bluetooth.ErrUnavailable) {
		t.Errorf("Scan() err = %v, want ErrUnavailable", err)
	}
	if _, err := svc.Devices(); !errors.Is(err, bluetooth.ErrUnavailable) {
		t.Errorf("Devices() err = %v, want ErrUnavailable", err)
	}
	if _, err := svc.Stats(); !errors.Is(err, bluetooth.ErrUnavailable) {
		t.Errorf("Stats() err = %v, want ErrUnavailable", err)
	}
}

func TestScanReturnsResultAndStats(t *testing.T) {
	t.Parallel()
	result := &discovery.BluetoothScanResult{}
	stats := &discovery.BluetoothDiscoveryStats{}
	svc := bluetooth.NewService(&fakeScanner{available: true, result: result, stats: stats})

	scan, err := svc.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() err = %v", err)
	}
	if scan.Result != result || scan.Stats != stats {
		t.Fatalf("Scan() = %+v, want the result + post-scan stats", scan)
	}
}

func TestScanSurfacesScannerError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("adapter down")
	svc := bluetooth.NewService(&fakeScanner{available: true, scanErr: sentinel})
	if _, err := svc.Scan(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("Scan() err = %v, want the scanner error verbatim", err)
	}
}

func TestDevicesReturnsLastScan(t *testing.T) {
	t.Parallel()
	last := &discovery.BluetoothScanResult{}
	svc := bluetooth.NewService(&fakeScanner{available: true, last: last})
	got, err := svc.Devices()
	if err != nil || got != last {
		t.Fatalf("Devices() = %v, %v; want the last scan, nil", got, err)
	}
}

func TestStatusUnavailable(t *testing.T) {
	t.Parallel()
	status := bluetooth.NewService(&fakeScanner{available: false}).Status()
	if status.Available || status.LastScan != nil {
		t.Fatalf("Status() = %+v, want available=false, no last scan", status)
	}
}

func TestStatusAvailable(t *testing.T) {
	t.Parallel()
	last := &discovery.BluetoothScanResult{}
	status := bluetooth.NewService(&fakeScanner{available: true, last: last}).Status()
	if !status.Available || status.LastScan != last {
		t.Fatalf("Status() = %+v, want available=true with the last scan", status)
	}
}
