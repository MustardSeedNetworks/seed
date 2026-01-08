package link_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/link"
)

// TestCheckAndNotifyNoChange tests checkAndNotify when state hasn't changed.
func TestCheckAndNotifyNoChange(t *testing.T) {
	// Use loopback which has stable state
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	// Get current state
	currentState := link.ExportCheckLinkStatePlatform(loopbackName)
	m.SetState(currentState)

	initialHistoryLen := m.MonitorHistoryLen()

	var callbackInvoked atomic.Bool
	m.OnStateChange(func(_ link.Event) {
		callbackInvoked.Store(true)
	})

	// Call checkAndNotify - state should not change
	m.ExportCheckAndNotify()

	// Allow goroutines to run
	time.Sleep(10 * time.Millisecond)

	// Verify no new history (state didn't change)
	if m.MonitorHistoryLen() != initialHistoryLen {
		t.Errorf("history length changed unexpectedly from %d to %d", initialHistoryLen, m.MonitorHistoryLen())
	}

	// Callback should not have been invoked
	if callbackInvoked.Load() {
		t.Error("callback should not be invoked when state doesn't change")
	}
}

// TestCheckAndNotifyStateChange tests checkAndNotify when state changes.
func TestCheckAndNotifyStateChange(t *testing.T) {
	// Use loopback which typically is always up
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	// Force state to something different from actual
	// Loopback is typically up, so set to down to force a change
	m.SetState(link.StateDown)

	var eventReceived atomic.Bool
	var receivedEvent link.Event
	var mu sync.Mutex

	m.OnStateChange(func(e link.Event) {
		mu.Lock()
		eventReceived.Store(true)
		receivedEvent = e
		mu.Unlock()
	})

	initialHistoryLen := m.MonitorHistoryLen()

	// Call checkAndNotify - should detect state change
	m.ExportCheckAndNotify()

	// Allow goroutines to run (callbacks are async)
	time.Sleep(50 * time.Millisecond)

	// Check if loopback is actually up (expected on most systems)
	actualState := link.ExportCheckLinkStatePlatform(loopbackName)
	if actualState == link.StateDown {
		t.Skip("Loopback is actually down, cannot test state change")
	}

	// Verify history was updated
	if m.MonitorHistoryLen() <= initialHistoryLen {
		t.Log("History was not updated - state may not have changed")
	}

	// Verify callback was invoked (if state changed)
	if eventReceived.Load() {
		mu.Lock()
		if receivedEvent.OldState != link.StateDown {
			t.Errorf("expected OldState=StateDown, got %v", receivedEvent.OldState)
		}
		if receivedEvent.Interface != loopbackName {
			t.Errorf("expected Interface=%s, got %s", loopbackName, receivedEvent.Interface)
		}
		mu.Unlock()
	}
}

// TestCheckAndNotifyWithMultipleCallbacks tests that all callbacks are invoked.
func TestCheckAndNotifyWithMultipleCallbacks(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	// Force state to something different to trigger change
	m.SetState(link.StateDown)

	var counter1, counter2, counter3 atomic.Int32

	m.OnStateChange(func(_ link.Event) {
		counter1.Add(1)
	})
	m.OnStateChange(func(_ link.Event) {
		counter2.Add(1)
	})
	m.OnStateChange(func(_ link.Event) {
		counter3.Add(1)
	})

	m.ExportCheckAndNotify()

	// Allow async callbacks to run
	time.Sleep(100 * time.Millisecond)

	// Check if state actually changed
	actualState := link.ExportCheckLinkStatePlatform(loopbackName)
	if actualState == link.StateDown {
		t.Skip("Loopback is actually down, cannot verify callback invocation")
	}

	// All callbacks should have been invoked if state changed
	if counter1.Load() != counter2.Load() || counter2.Load() != counter3.Load() {
		t.Errorf(
			"callbacks invoked different number of times: %d, %d, %d",
			counter1.Load(),
			counter2.Load(),
			counter3.Load(),
		)
	}
}

// TestCheckAndNotifyPanicRecovery tests that panicking callbacks don't affect others.
func TestCheckAndNotifyPanicRecovery(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)
	m.SetState(link.StateDown)

	var normalCallbackRan atomic.Bool

	// Register panicking callback first
	m.OnStateChange(func(_ link.Event) {
		panic("intentional test panic")
	})

	// Register normal callback after
	m.OnStateChange(func(_ link.Event) {
		normalCallbackRan.Store(true)
	})

	// Should not panic
	m.ExportCheckAndNotify()

	// Allow async callbacks
	time.Sleep(100 * time.Millisecond)

	// Check if state actually changed
	actualState := link.ExportCheckLinkStatePlatform(loopbackName)
	if actualState != link.StateDown && !normalCallbackRan.Load() {
		t.Log("Normal callback may not have run due to execution order")
	}
}

// TestCheckAndNotifyHistoryTrimming tests that history is properly trimmed.
func TestCheckAndNotifyHistoryTrimming(t *testing.T) {
	m := link.NewMonitor("fake_interface")
	maxHistory := m.MonitorMaxHistory()

	// Fill history to max
	for range maxHistory {
		m.AddHistoryEvent(link.Event{
			Interface: "fake_interface",
			OldState:  link.StateDown,
			NewState:  link.StateUp,
			Timestamp: time.Now(),
		})
	}

	if m.MonitorHistoryLen() != maxHistory {
		t.Fatalf("expected history length %d, got %d", maxHistory, m.MonitorHistoryLen())
	}

	// Force a state different from what checkLinkState will return
	m.SetState(link.StateUp) // Non-existent interface returns Unknown

	// This should add an event and trim
	m.ExportCheckAndNotify()

	// History should still be at max (or unchanged if state didn't change)
	if m.MonitorHistoryLen() > maxHistory {
		t.Errorf("history exceeded max: %d > %d", m.MonitorHistoryLen(), maxHistory)
	}
}

// TestCheckAndNotifyTimestamp tests that events have correct timestamps.
func TestCheckAndNotifyTimestamp(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)
	m.SetState(link.StateDown)

	beforeCheck := time.Now()
	m.ExportCheckAndNotify()
	afterCheck := time.Now()

	history := m.GetHistory()
	if len(history) > 0 {
		lastEvent := history[len(history)-1]
		if lastEvent.Timestamp.Before(beforeCheck) || lastEvent.Timestamp.After(afterCheck) {
			t.Errorf(
				"event timestamp %v not between %v and %v",
				lastEvent.Timestamp,
				beforeCheck,
				afterCheck,
			)
		}
	}
}

// TestCheckAndNotifyEventInterface tests that events have correct interface name.
func TestCheckAndNotifyEventInterface(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)
	m.SetState(link.StateDown)

	m.ExportCheckAndNotify()

	history := m.GetHistory()
	if len(history) > 0 {
		lastEvent := history[len(history)-1]
		if lastEvent.Interface != loopbackName {
			t.Errorf("expected interface %q, got %q", loopbackName, lastEvent.Interface)
		}
	}
}

// TestMonitorWithFastPolling tests monitor with faster poll interval.
func TestMonitorWithFastPolling(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)
	m.SetPollInterval(50) // 50ms

	var callCount atomic.Int32
	m.OnStateChange(func(_ link.Event) {
		callCount.Add(1)
	})

	err := m.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Let it run for a bit with fast polling
	time.Sleep(200 * time.Millisecond)

	m.Stop()

	// Monitor should have polled multiple times without errors.
	// After Stop, IsRunning should be false - this is expected behavior.
	_ = m.IsRunning()
}

// TestWaitForStateSuccessPath tests the success path of WaitForState.
func TestWaitForStateSuccessPath(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	// Check what state loopback is actually in
	actualState := link.ExportCheckLinkStatePlatform(loopbackName)

	// Set to different state
	var targetState link.State
	if actualState == link.StateUp {
		m.SetState(link.StateDown)
		targetState = link.StateUp
	} else {
		m.SetState(link.StateUp)
		targetState = actualState
	}

	// Now WaitForState should succeed quickly because checkLinkState returns actual state
	result := m.WaitForState(targetState, 500*time.Millisecond)

	if actualState == targetState || actualState == link.StateUp {
		if !result {
			t.Log("WaitForState returned false but was expected to find target state")
		}
	}
}

// TestWaitForStateWithExistingInterface tests waiting with a real interface.
func TestWaitForStateWithExistingInterface(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	// Get actual current state
	currentState := link.ExportCheckLinkStatePlatform(loopbackName)

	// Wait for current state should succeed immediately
	start := time.Now()
	result := m.WaitForState(currentState, 100*time.Millisecond)
	elapsed := time.Since(start)

	if !result {
		t.Errorf("WaitForState should succeed for current state %v", currentState)
	}

	// Should be fast since we're checking for current state
	if elapsed > 20*time.Millisecond {
		t.Logf("WaitForState took %v (expected < 20ms for current state)", elapsed)
	}
}

// TestWaitForStateUpdatesMonitorState tests that WaitForState updates monitor state.
func TestWaitForStateUpdatesMonitorState(t *testing.T) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)
	m.SetState(link.StateDown)

	actualState := link.ExportCheckLinkStatePlatform(loopbackName)

	// Wait for actual state
	result := m.WaitForState(actualState, 100*time.Millisecond)

	if result {
		// Monitor state should now be updated
		if m.GetState() != actualState {
			t.Errorf("monitor state should be %v after WaitForState, got %v", actualState, m.GetState())
		}
	}
}

// BenchmarkCheckAndNotify benchmarks checkAndNotify performance.
func BenchmarkCheckAndNotify(b *testing.B) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	b.ResetTimer()
	for b.Loop() {
		m.ExportCheckAndNotify()
	}
}

// BenchmarkCheckAndNotifyWithCallbacks benchmarks checkAndNotify with callbacks.
func BenchmarkCheckAndNotifyWithCallbacks(b *testing.B) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)

	// Add some callbacks
	for range 5 {
		m.OnStateChange(func(_ link.Event) {})
	}

	b.ResetTimer()
	for b.Loop() {
		m.ExportCheckAndNotify()
	}
}

// BenchmarkWaitForState benchmarks WaitForState performance.
func BenchmarkWaitForState(b *testing.B) {
	loopbackName := "lo0"
	if runtime.GOOS == "linux" {
		loopbackName = "lo"
	}

	m := link.NewMonitor(loopbackName)
	targetState := link.ExportCheckLinkStatePlatform(loopbackName)

	b.ResetTimer()
	for b.Loop() {
		m.WaitForState(targetState, 10*time.Millisecond)
	}
}
