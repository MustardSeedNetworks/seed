package enumerate

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// observedService builds a Service through its constructor so every field the
// sweep path touches is real, without driving a sweep: the house style here is
// to exercise the service's own logic rather than put packets on the wire.
func observedService(t *testing.T) *Service {
	t.Helper()
	return NewService(&config.Config{NetworkDiscovery: config.NetworkDiscoveryConfig{
		Options:     config.DiscoveryOptions{ARPScan: true},
		ScanTimeout: time.Second,
	}}, realInterface(t), nil)
}

// The observer is how a router's routing table becomes a learnable target
// network (seed#2695), so it has to receive the sweep's devices, not a count.
func TestSweepObserverReceivesTheSweepsDevices(t *testing.T) {
	svc := observedService(t)
	var got []*DiscoveredDevice
	calls := 0
	svc.SetSweepObserver(func(_ context.Context, devices []*DiscoveredDevice) {
		calls++
		got = devices
	})

	want := []*DiscoveredDevice{{IP: "10.44.40.1"}, {IP: "10.44.40.2"}}
	svc.notifySweep(context.Background(), want)

	if calls != 1 {
		t.Fatalf("observer called %d times, want 1", calls)
	}
	if len(got) != len(want) || got[0].IP != "10.44.40.1" || got[1].IP != "10.44.40.2" {
		t.Errorf("observer got %+v, want the sweep's devices %+v", got, want)
	}
}

// A service with no observer registered sweeps exactly as before.
func TestSweepObserverIsOptional(t *testing.T) {
	svc := observedService(t)

	svc.notifySweep(context.Background(), []*DiscoveredDevice{{IP: "10.44.40.1"}})

	svc.SetSweepObserver(func(context.Context, []*DiscoveredDevice) {})
	svc.SetSweepObserver(nil)
	svc.notifySweep(context.Background(), []*DiscoveredDevice{{IP: "10.44.40.1"}})
}

// The observer runs outside s.mu. The learner it carries reads discovery
// settings and can end up back in the service, and notifying under the write
// lock the sweep still holds would deadlock the daemon on its first sweep.
func TestSweepObserverRunsOutsideTheServiceLock(t *testing.T) {
	svc := observedService(t)
	reentered := make(chan *ServiceStatus, 1)
	svc.SetSweepObserver(func(context.Context, []*DiscoveredDevice) {
		reentered <- svc.GetStatus() // takes s.mu
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.notifySweep(context.Background(), nil)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("notifySweep deadlocked: the observer cannot take the service lock")
	}
	select {
	case <-reentered:
	default:
		t.Fatal("the observer never ran, so the lock it must not hold was never tested")
	}
}
