package enumerate

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// realInterface names an interface that exists on this host. SetInterface
// validates the name (#840), and the loopback is spelled lo0 on darwin and lo
// on linux, so the name cannot be a literal.
func realInterface(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil || len(ifaces) == 0 {
		t.Skipf("no interfaces to switch to: %v", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface.Name
		}
	}
	return ifaces[0].Name
}

func discoveringService(t *testing.T) (*Service, chan string, string) {
	t.Helper()
	scans := make(chan string, 4)
	iface := realInterface(t)
	svc := &Service{
		cfg: &config.Config{NetworkDiscovery: config.NetworkDiscoveryConfig{
			Options:     config.DiscoveryOptions{ARPScan: true},
			ScanTimeout: time.Second,
		}},
		interfaceName:   iface,
		deviceDiscovery: NewDeviceDiscovery(iface),
		scanFunc: func(context.Context) error {
			scans <- "scan"
			return nil
		},
	}
	return svc, scans, iface
}

// seed#2674, owner decision 2026-09-15: the sweep follows the interface. Until
// now SetInterface only swapped the name, so a laptop moved from Ethernet to
// Wi-Fi kept showing the old subnet's devices until the rescan timer fired.
func TestSetInterfaceRescansTheNewInterface(t *testing.T) {
	svc, scans, iface := discoveringService(t)
	svc.running = true

	if err := svc.SetInterface(iface); err != nil {
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
	svc, scans, iface := discoveringService(t)

	if err := svc.SetInterface(iface); err != nil {
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
	svc, scans, iface := discoveringService(t)
	svc.running = true
	svc.cfg.NetworkDiscovery.Options = config.DiscoveryOptions{}

	if err := svc.SetInterface(iface); err != nil {
		t.Fatalf("SetInterface: %v", err)
	}

	select {
	case <-scans:
		t.Fatal("scanned with every active discovery method off")
	case <-time.After(250 * time.Millisecond):
	}
}
