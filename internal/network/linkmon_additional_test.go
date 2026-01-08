package network_test

// Additional LinkMonitor tests for comprehensive coverage - focuses on
// state change handling, rate limiting, callbacks, and history management.

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/network"
)

func TestLinkStateStringDefaultCase(t *testing.T) {
	// Test the default case in String() by using an invalid state value.
	// Since LinkState is an int, we can create an invalid value.
	invalidState := network.LinkState(99)
	got := invalidState.String()
	if got != "unknown" {
		t.Errorf("LinkState(99).String() = %v, want unknown", got)
	}

	// Test with negative value
	negativeState := network.LinkState(-1)
	got = negativeState.String()
	if got != "unknown" {
		t.Errorf("LinkState(-1).String() = %v, want unknown", got)
	}
}

func TestLinkMonitorRecordEvent(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Add events and verify history
	event1 := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: time.Now().Add(-1 * time.Hour),
	}
	event2 := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateDown,
		Timestamp: time.Now(),
	}

	helper.RecordEvent(event1)
	helper.RecordEvent(event2)

	history := helper.GetHistory()
	if len(history) != 2 {
		t.Errorf("History length = %d, want 2", len(history))
	}

	if history[0].State != network.LinkStateUp {
		t.Errorf("First event state = %v, want Up", history[0].State)
	}
	if history[1].State != network.LinkStateDown {
		t.Errorf("Second event state = %v, want Down", history[1].State)
	}
}

func TestLinkMonitorRecordEventMaxHistory(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Set small max history for testing
	helper.SetMaxHistory(3)

	// Add more events than max history
	for i := range 5 {
		event := network.LinkEvent{
			Interface: "eth0",
			State:     network.LinkStateUp,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		helper.RecordEvent(event)
	}

	history := helper.GetHistory()
	if len(history) != 3 {
		t.Errorf("History length = %d, want 3 (max)", len(history))
	}

	// Oldest events should be trimmed
	// The history should contain the last 3 events
}

func TestLinkMonitorCheckAndUpdateRateLimit(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Set a known gap
	helper.SetMinCallbackGap(int64(100 * time.Millisecond))
	helper.SetLastCallbackTime(time.Time{}) // Zero time

	// First call should succeed (enough time has passed since zero time)
	result := helper.CheckAndUpdateRateLimit()
	if !result {
		t.Error("First CheckAndUpdateRateLimit() = false, want true")
	}

	// Immediate second call should be rate limited
	result = helper.CheckAndUpdateRateLimit()
	if result {
		t.Error("Immediate second CheckAndUpdateRateLimit() = true, want false")
	}

	// Wait for gap and try again
	time.Sleep(150 * time.Millisecond)
	result = helper.CheckAndUpdateRateLimit()
	if !result {
		t.Error("CheckAndUpdateRateLimit() after gap = false, want true")
	}
}

func TestLinkMonitorProcessStateChangeNoChange(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Set initial state
	helper.SetLastState(network.LinkStateUp)

	// Process same state - should not change
	shouldNotify, changed := helper.ProcessStateChange(network.LinkStateUp)
	if changed {
		t.Error("ProcessStateChange() changed = true when state didn't change")
	}
	if shouldNotify {
		t.Error("ProcessStateChange() shouldNotify = true when state didn't change")
	}
}

func TestLinkMonitorProcessStateChangeWithChange(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Set initial state
	helper.SetLastState(network.LinkStateDown)
	helper.SetLastCallbackTime(time.Time{}) // Ensure rate limit passes

	// Process different state - should change
	shouldNotify, changed := helper.ProcessStateChange(network.LinkStateUp)
	if !changed {
		t.Error("ProcessStateChange() changed = false when state changed")
	}
	if !shouldNotify {
		t.Error("ProcessStateChange() shouldNotify = false, want true")
	}

	// Verify state was updated
	if helper.GetLastState() != network.LinkStateUp {
		t.Errorf("State = %v, want Up", helper.GetLastState())
	}
}

func TestLinkMonitorProcessStateChangeRateLimited(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Set initial state and recent callback time
	helper.SetLastState(network.LinkStateDown)
	helper.SetMinCallbackGap(int64(1 * time.Hour))               // Long gap to ensure rate limiting
	helper.SetLastCallbackTime(time.Now().Add(-1 * time.Second)) // Recent callback

	// Process different state - should change but be rate limited
	shouldNotify, changed := helper.ProcessStateChange(network.LinkStateUp)
	if !changed {
		t.Error("ProcessStateChange() changed = false when state changed")
	}
	if shouldNotify {
		t.Error("ProcessStateChange() shouldNotify = true, want false (rate limited)")
	}
}

func TestLinkMonitorSafeInvokeCallback(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	event := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: time.Now(),
	}

	// Test normal callback
	var called atomic.Bool
	callback := func(e network.LinkEvent) {
		called.Store(true)
		if e.Interface != "eth0" {
			t.Errorf("Callback received wrong interface: %s", e.Interface)
		}
	}

	helper.SafeInvokeCallback(callback, event)
	time.Sleep(10 * time.Millisecond) // Allow async execution

	if !called.Load() {
		t.Error("Normal callback was not called")
	}
}

func TestLinkMonitorSafeInvokeCallbackWithPanic(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	event := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: time.Now(),
	}

	// Test callback that panics - should be recovered
	panicCallback := func(_ network.LinkEvent) {
		panic("test panic in callback")
	}

	// This should not panic
	helper.SafeInvokeCallback(panicCallback, event)
	time.Sleep(10 * time.Millisecond)

	// If we reach here, panic was recovered
	t.Log("Panic was successfully recovered")
}

func TestLinkMonitorNotifyCallbacks(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	event := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: time.Now(),
	}

	var callCount atomic.Int32
	var wg sync.WaitGroup

	callbacks := make([]network.LinkStateCallback, 3)
	for i := range 3 {
		wg.Add(1)
		callbacks[i] = func(_ network.LinkEvent) {
			callCount.Add(1)
			wg.Done()
		}
	}

	helper.NotifyCallbacks(event, callbacks)

	// Wait for callbacks with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for callbacks")
	}

	if callCount.Load() != 3 {
		t.Errorf("Callback count = %d, want 3", callCount.Load())
	}
}

func TestLinkMonitorGetFlapCount24hWithEvents(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Add events at various times
	now := time.Now()

	// Event within 24h (should count)
	helper.AddHistoryEvent(network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: now.Add(-1 * time.Hour),
	})

	// Event within 24h (should count)
	helper.AddHistoryEvent(network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateDown,
		Timestamp: now.Add(-2 * time.Hour),
	})

	// Event outside 24h (should not count)
	helper.AddHistoryEvent(network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: now.Add(-25 * time.Hour),
	})

	// Event exactly at 24h boundary (should not count - using after)
	helper.AddHistoryEvent(network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateDown,
		Timestamp: now.Add(-24 * time.Hour),
	})

	count := mon.GetFlapCount24h()
	if count != 2 {
		t.Errorf("GetFlapCount24h() = %d, want 2", count)
	}
}

func TestLinkMonitorGetFlapCount24hEmpty(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	count := mon.GetFlapCount24h()
	if count != 0 {
		t.Errorf("GetFlapCount24h() on new monitor = %d, want 0", count)
	}
}

func TestLinkMonitorWaitForLinkUpAlreadyUp(t *testing.T) {
	mon := network.NewLinkMonitor("lo")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Set state to up
	helper.SetLastState(network.LinkStateUp)

	// Should return immediately
	start := time.Now()
	result := mon.WaitForLinkUp(5 * time.Second)
	elapsed := time.Since(start)

	if !result {
		t.Error("WaitForLinkUp() = false when already up")
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("WaitForLinkUp() took %v, should return immediately", elapsed)
	}
}

func TestLinkMonitorWaitForLinkUpTimeout(t *testing.T) {
	// Use a non-existent interface to ensure it stays unknown/down
	mon := network.NewLinkMonitor("nonexistent999")

	start := time.Now()
	result := mon.WaitForLinkUp(200 * time.Millisecond)
	elapsed := time.Since(start)

	if result {
		t.Error("WaitForLinkUp() = true for non-existent interface")
	}

	// Should have waited close to the timeout
	if elapsed < 150*time.Millisecond {
		t.Errorf("WaitForLinkUp() returned too early: %v", elapsed)
	}
}

func TestLinkMonitorCheckLinkState(t *testing.T) {
	// Test with loopback interface (should exist on all systems)
	mon := network.NewLinkMonitor("lo")
	helper := network.NewLinkMonitorTestHelper(mon)

	state := helper.CheckLinkState()
	// State should be Up, Down, or Unknown depending on the system
	validStates := []network.LinkState{
		network.LinkStateUp,
		network.LinkStateDown,
		network.LinkStateUnknown,
	}

	if !slices.Contains(validStates, state) {
		t.Errorf("CheckLinkState() returned invalid state: %v", state)
	}

	t.Logf("Loopback interface state: %v", state)
}

func TestLinkMonitorCheckLinkStateNonExistent(t *testing.T) {
	mon := network.NewLinkMonitor("nonexistent999")
	helper := network.NewLinkMonitorTestHelper(mon)

	state := helper.CheckLinkState()
	if state != network.LinkStateUnknown {
		t.Errorf("CheckLinkState() for non-existent interface = %v, want Unknown", state)
	}
}

func TestLinkMonitorMultipleCallbacks(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Register multiple callbacks
	var callOrder []int
	var mu sync.Mutex

	for i := range 5 {
		idx := i
		mon.OnStateChange(func(_ network.LinkEvent) {
			mu.Lock()
			callOrder = append(callOrder, idx)
			mu.Unlock()
		})
	}

	if helper.GetCallbacksCount() != 5 {
		t.Errorf("Callbacks count = %d, want 5", helper.GetCallbacksCount())
	}
}

func TestLinkMonitorHistoryRetention(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Default max history is 100
	if helper.GetMaxHistory() != 100 {
		t.Errorf("Default maxHistory = %d, want 100", helper.GetMaxHistory())
	}

	// Set smaller max for testing
	helper.SetMaxHistory(5)

	// Add 10 events
	for i := range 10 {
		helper.AddHistoryEvent(network.LinkEvent{
			Interface: "eth0",
			State:     network.LinkState(i % 2),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	// Should only retain 5 events
	history := helper.GetHistory()
	if len(history) != 5 {
		t.Errorf("History length = %d, want 5", len(history))
	}
}

func TestLinkMonitorMinCallbackGap(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	// Default is 100ms
	defaultGap := helper.GetMinCallbackGap()
	if defaultGap != int64(100*time.Millisecond) {
		t.Errorf("Default minCallbackGap = %d, want %d", defaultGap, 100*time.Millisecond)
	}

	// Set custom gap
	helper.SetMinCallbackGap(int64(500 * time.Millisecond))
	newGap := helper.GetMinCallbackGap()
	if newGap != int64(500*time.Millisecond) {
		t.Errorf("minCallbackGap after set = %d, want %d", newGap, 500*time.Millisecond)
	}
}

func TestLinkMonitorStartStopIdempotent(t *testing.T) {
	mon := network.NewLinkMonitor("lo")

	// Start multiple times
	for range 3 {
		err := mon.Start()
		if err != nil {
			t.Errorf("Start() error = %v", err)
		}
	}

	// Stop multiple times
	for range 3 {
		mon.Stop() // Should not panic
	}
}

func TestLinkEventTimestamp(t *testing.T) {
	before := time.Now()

	event := network.LinkEvent{
		Interface: "eth0",
		State:     network.LinkStateUp,
		Timestamp: time.Now(),
	}

	after := time.Now()

	// Verify all fields are properly set
	if event.Interface != "eth0" {
		t.Errorf("Interface = %v, want eth0", event.Interface)
	}
	if event.State != network.LinkStateUp {
		t.Errorf("State = %v, want Up", event.State)
	}
	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("Event timestamp is outside expected range")
	}
}

func TestLinkMonitorGetUptimeProgresses(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")

	uptime1 := mon.GetUptime()
	time.Sleep(50 * time.Millisecond)
	uptime2 := mon.GetUptime()

	if uptime2 <= uptime1 {
		t.Errorf("Uptime did not progress: %v -> %v", uptime1, uptime2)
	}
}

func TestLinkMonitorSetInterfaceWhileRunning(t *testing.T) {
	mon := network.NewLinkMonitor("eth0")
	helper := network.NewLinkMonitorTestHelper(mon)

	err := mon.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer mon.Stop()

	// Set state to known value
	helper.SetLastState(network.LinkStateUp)

	// Change interface while running
	mon.SetInterface("eth1")

	if helper.GetInterfaceName() != "eth1" {
		t.Errorf("Interface name = %v, want eth1", helper.GetInterfaceName())
	}

	// State should be reset
	if helper.GetLastState() != network.LinkStateUnknown {
		t.Errorf("State after SetInterface = %v, want Unknown", helper.GetLastState())
	}
}
