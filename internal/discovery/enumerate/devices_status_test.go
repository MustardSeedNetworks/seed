package enumerate_test

import (
	"context"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// loopbackName returns this host's loopback interface: "lo" on Linux, "lo0" on
// Darwin. Hard-coding either passes on one of CI and the Mac only.
func loopbackName(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces: %v", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface.Name
		}
	}
	t.Fatal("no loopback interface on this host")
	return ""
}

// A fresh install says "discovering…" rather than "no devices found" by reading
// the status this GetStatus serves over /api/v1/security/devices (seed#2674).
// Two properties carry that state, and neither is asserted anywhere else:
// LastScan is the zero time until a sweep has run, and it stops being zero once
// one has — including a sweep that failed, or the page would sit on
// "discovering…" for ever on a host discovery cannot scan.
func TestGetStatusLastScanTracksFirstSweep(t *testing.T) {
	t.Parallel()

	dd := enumerate.NewDeviceDiscovery(loopbackName(t))
	t.Cleanup(dd.Stop)

	if before := dd.GetStatus(); !before.LastScan.IsZero() {
		t.Fatalf("LastScan before any scan = %v, want the zero time", before.LastScan)
	}

	// The ARP scanner skips loopback addresses on every platform (arp.go's
	// `!ipNet.IP.IsLoopback()`), so this is the failing-sweep path on Linux and
	// Darwin alike. The outcome is deliberately not asserted: only the status is.
	_ = dd.Scan(context.Background())

	after := dd.GetStatus()
	if after.LastScan.IsZero() {
		t.Error("LastScan is still the zero time after a sweep returned")
	}
	if after.Scanning {
		t.Error("Scanning is still true after Scan returned")
	}
}
