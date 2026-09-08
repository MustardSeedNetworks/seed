package netif_test

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

// countingDetector stands in for the platform detector and reports one score,
// for the first real interface, so the test can see whether cached detection
// data still reaches the interface map.
type countingDetector struct {
	calls atomic.Int32
	delay time.Duration
	name  string
}

func (c *countingDetector) DetectAll() ([]detection.InterfaceScore, error) {
	c.calls.Add(1)
	time.Sleep(c.delay)
	return []detection.InterfaceScore{{Name: c.name, Speed: 42, SpeedDisplay: "42 bps"}}, nil
}

func firstInterfaceName(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil || len(ifaces) == 0 {
		t.Skipf("no interfaces to refresh: %v", err)
	}
	return ifaces[0].Name
}

func TestRefreshInterfacesReusesRecentDetection(t *testing.T) {
	clock := time.Unix(1_700_000_000, 0)
	det := &countingDetector{name: firstInterfaceName(t)}
	mgr := netif.NewManagerForDetector(det, func() time.Time { return clock })

	for range 3 {
		if err := mgr.RefreshInterfaces(); err != nil {
			t.Fatalf("RefreshInterfaces() error = %v", err)
		}
	}
	if got := det.calls.Load(); got != 1 {
		t.Fatalf("DetectAll ran %d times across 3 refreshes inside the TTL, want 1", got)
	}

	info, err := mgr.GetInterface(det.name)
	if err != nil {
		t.Fatalf("GetInterface(%q) error = %v", det.name, err)
	}
	if info.Speed != 42 || info.SpeedDisplay != "42 bps" {
		t.Fatalf("cached detection was not applied: speed=%d display=%q", info.Speed, info.SpeedDisplay)
	}

	clock = clock.Add(netif.DetectionTTL)
	if err = mgr.RefreshInterfaces(); err != nil {
		t.Fatalf("RefreshInterfaces() error = %v", err)
	}
	if got := det.calls.Load(); got != 2 {
		t.Fatalf("DetectAll ran %d times after the TTL elapsed, want 2", got)
	}
}

func TestRefreshInterfacesBurstDetectsOnce(t *testing.T) {
	det := &countingDetector{name: firstInterfaceName(t), delay: 50 * time.Millisecond}
	mgr := netif.NewManagerForDetector(det, time.Now)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := mgr.RefreshInterfaces(); err != nil {
				t.Errorf("RefreshInterfaces() error = %v", err)
			}
		})
	}
	wg.Wait()

	if got := det.calls.Load(); got != 1 {
		t.Fatalf("DetectAll ran %d times for 8 concurrent refreshes, want 1", got)
	}
}
