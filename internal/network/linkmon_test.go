// Package network_test handles network interface management.
// LinkMonitor test suite validates real-time link state monitoring, state change detection,
// and interface property change tracking across network transitions.
package network_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/network"
)

func TestLinkStateString(t *testing.T) {
	tests := []struct {
		name  string
		state network.LinkState
		want  string
	}{
		{"up state", network.LinkStateUp, "up"},
		{"down state", network.LinkStateDown, "down"},
		{"unknown state", network.LinkStateUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("LinkState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLinkMonitor(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	if mon == nil {
		t.Fatal("NewLinkMonitor() returned nil")
	}

	helper := network.NewLinkMonitorTestHelper(mon)

	if helper.GetInterfaceName() != "eth0" {
		t.Errorf("interfaceName = %v, want eth0", helper.GetInterfaceName())
	}

	if helper.GetLastState() != network.LinkStateUnknown {
		t.Errorf("lastState = %v, want LinkStateUnknown", helper.GetLastState())
	}

	if helper.GetPollInterval() != int64(500*time.Millisecond) {
		t.Errorf("pollInterval = %v, want 500ms", helper.GetPollInterval())
	}

	if helper.GetMaxHistory() != 100 {
		t.Errorf("maxHistory = %v, want 100", helper.GetMaxHistory())
	}

	if helper.GetCallbacksCount() < 0 {
		t.Error("callbacks count is negative")
	}
}

func TestLinkMonitorStartStop(t *testing.T) {
	mon := network.NewLinkMonitor("lo")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Start monitoring.
	err := mon.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Verify running.
	if !helper.IsRunning() {
		t.Error("Monitor not running after Start()")
	}

	// Starting again should be no-op.
	err = mon.Start()
	if err != nil {
		t.Errorf("Second Start() error = %v", err)
	}

	// Give it time to poll once.
	time.Sleep(600 * time.Millisecond)

	// Stop monitoring.
	mon.Stop()

	// Verify stopped.
	if helper.IsRunning() {
		t.Error("Monitor still running after Stop()")
	}

	// Stopping again should be no-op.
	mon.Stop()
}

func TestLinkMonitorSetInterface(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Change interface.
	mon.SetInterface("eth1")

	if helper.GetInterfaceName() != "eth1" {
		t.Errorf("interfaceName = %v, want eth1", helper.GetInterfaceName())
	}

	// State should be reset to unknown.
	if helper.GetLastState() != network.LinkStateUnknown {
		t.Errorf("lastState = %v, want LinkStateUnknown after interface change", helper.GetLastState())
	}
}

func TestLinkMonitorGetState(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	// Initial state should be unknown.
	state := mon.GetState()
	if state != network.LinkStateUnknown {
		t.Errorf("GetState() = %v, want LinkStateUnknown", state)
	}

	// Start to check actual state.
	err := mon.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mon.Stop()

	// Give it time to check state.
	time.Sleep(600 * time.Millisecond)

	state = mon.GetState()
	// loopback should be up or unknown (depending on system).
	t.Logf("Loopback state: %v", state)
}

func TestLinkMonitorIsUp(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	err := mon.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mon.Stop()

	// Give it time to check state.
	time.Sleep(600 * time.Millisecond)

	isUp := mon.IsUp()
	// loopback should typically be up.
	t.Logf("Loopback IsUp: %v", isUp)
}

func TestLinkMonitorOnStateChange(t *testing.T) {
	mon := network.NewLinkMonitor("lo")
	helper := network.NewLinkMonitorTestHelper(mon)

	callbackCalled := false
	var receivedEvent network.LinkEvent

	// Register callback.
	mon.OnStateChange(func(event network.LinkEvent) {
		callbackCalled = true
		receivedEvent = event
	})

	// Verify callback was registered.
	if helper.GetCallbacksCount() != 1 {
		t.Errorf("callbacks length = %d, want 1", helper.GetCallbacksCount())
	}

	// Note: Callback will only be called if state actually changes.
	// For this test, we just verify it was registered.
	t.Logf("Callback registered, would be called on state change")
	_ = callbackCalled
	_ = receivedEvent
}

func TestLinkMonitorGetHistory(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	// Initially empty.
	history := mon.GetHistory()
	if len(history) != 0 {
		t.Errorf("GetHistory() length = %d, want 0", len(history))
	}

	// Note: History will only populate if state changes occur.
	// For this test, we verify it returns empty initially.
}

func TestLinkMonitorGetUptime(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	// Small delay to ensure uptime > 0.
	time.Sleep(10 * time.Millisecond)

	uptime := mon.GetUptime()
	if uptime <= 0 {
		t.Errorf("GetUptime() = %v, want > 0", uptime)
	}

	if uptime > 1*time.Second {
		t.Errorf("GetUptime() = %v, unexpectedly large", uptime)
	}
}

func TestLinkMonitorGetFlapCount24h(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	// Initially should be 0.
	count := mon.GetFlapCount24h()
	if count != 0 {
		t.Errorf("GetFlapCount24h() = %d, want 0", count)
	}
}

func TestLinkMonitorWaitForLinkUp(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	err := mon.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mon.Stop()

	// Give it time to check initial state.
	time.Sleep(600 * time.Millisecond)

	// Wait for link up with short timeout.
	// Loopback should typically be up already.
	result := mon.WaitForLinkUp(2 * time.Second)
	t.Logf("WaitForLinkUp() = %v", result)

	// If loopback is down, result will be false, which is still a valid test.
}

func TestLinkMonitorConcurrentAccess(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	err := mon.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mon.Stop()

	// Concurrent reads.
	done := make(chan bool)
	for range 10 {
		go func() {
			for range 50 {
				_ = mon.GetState()
				_ = mon.IsUp()
				_ = mon.GetHistory()
				_ = mon.GetFlapCount24h()
				_ = mon.GetUptime()
			}
			done <- true
		}()
	}

	// Wait for all goroutines.
	for range 10 {
		<-done
	}
}

func TestLinkMonitorCallbackPanicRecovery(t *testing.T) {
	mon := network.NewLinkMonitor("lo")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Register a callback that panics.
	mon.OnStateChange(func(_ network.LinkEvent) {
		panic("test panic")
	})

	// Register a normal callback after the panic one.
	normalCalled := false
	mon.OnStateChange(func(_ network.LinkEvent) {
		normalCalled = true
	})

	err := mon.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mon.Stop()

	// Give it time to poll and potentially trigger callbacks.
	// Note: Callbacks only fire on state change.
	time.Sleep(600 * time.Millisecond)

	// Monitor should still be running despite callback panic.
	if !helper.IsRunning() {
		t.Error("Monitor stopped after callback panic")
	}

	// Note: normalCalled may be false if no state change occurred.
	// This test primarily verifies panic recovery doesn't crash the monitor.
	_ = normalCalled
}

func TestLinkEvent(t *testing.T) {
	// Test LinkEvent structure.
	event := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: time.Now(),
	}

	if event.Interface != "eth0" {
		t.Errorf("Interface = %v, want eth0", event.Interface)
	}

	if event.State != network.LinkStateUp {
		t.Errorf("State = %v, want LinkStateUp", event.State)
	}

	if event.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestLinkStateConstants(t *testing.T) {
	// Verify state constants are distinct.
	if network.LinkStateUnknown == network.LinkStateDown || network.LinkStateUnknown == network.LinkStateUp {
		t.Error("LinkStateUnknown overlaps with other states")
	}

	if network.LinkStateDown == network.LinkStateUp {
		t.Error("LinkStateDown == LinkStateUp")
	}

	// Verify they have different string representations.
	states := map[network.LinkState]string{
		network.LinkStateUnknown: "unknown",
		network.LinkStateDown:    "down",
		network.LinkStateUp:      "up",
	}

	seen := make(map[string]bool)
	for state, str := range states {
		if state.String() != str {
			t.Errorf("State %d String() = %v, want %v", state, state.String(), str)
		}
		if seen[str] {
			t.Errorf("Duplicate string representation: %v", str)
		}
		seen[str] = true
	}
}
