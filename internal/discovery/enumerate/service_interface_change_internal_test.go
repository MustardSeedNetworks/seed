package enumerate

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func discoveringService(t *testing.T) (*Service, chan string) {
	t.Helper()
	scans := make(chan string, 4)
	svc := &Service{
		cfg: &config.Config{NetworkDiscovery: config.NetworkDiscoveryConfig{
			Options:     config.DiscoveryOptions{ARPScan: true},
			ScanTimeout: time.Second,
		}},
		interfaceName:   "lo0",
		deviceDiscovery: NewDeviceDiscovery("lo0"),
		scanFunc: func(context.Context) error {
			scans <- "scan"
			return nil
		},
	}
	return svc, scans
}

// seed#2674, owner decision 2026-09-15: the sweep follows the interface. Until
// now SetInterface only swapped the name, so a laptop moved from Ethernet to
// Wi-Fi kept showing the old subnet's devices until the rescan timer fired.
func TestSetInterfaceRescansTheNewInterface(t *testing.T) {
	svc, scans := discoveringService(t)
	svc.running = true

	if err := svc.SetInterface("en0"); err != nil {
		t.Fatalf("SetInterface: %v", err)
	}

	select {
	case <-scans:
	case <-time.After(2 * time.Second):
		t.Fatal("interface change did not trigger a discovery sweep")
	}
}

// A service that is not running has nothing to rescan; an interface change
// must not start traffic behind a stopped service.
func TestSetInterfaceDoesNotScanWhileStopped(t *testing.T) {
	svc, scans := discoveringService(t)

	if err := svc.SetInterface("en0"); err != nil {
		t.Fatalf("SetInterface: %v", err)
	}

	select {
	case <-scans:
		t.Fatal("a stopped service scanned on an interface change")
	case <-time.After(250 * time.Millisecond):
	}
}

// With every active method off there is nothing to sweep, so an interface
// change stays quiet even while the service runs.
func TestSetInterfaceDoesNotScanWithoutAnActiveMethod(t *testing.T) {
	svc, scans := discoveringService(t)
	svc.running = true
	svc.cfg.NetworkDiscovery.Options = config.DiscoveryOptions{}

	if err := svc.SetInterface("en0"); err != nil {
		t.Fatalf("SetInterface: %v", err)
	}

	select {
	case <-scans:
		t.Fatal("scanned with every active discovery method off")
	case <-time.After(250 * time.Millisecond):
	}
}
