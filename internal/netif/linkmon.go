package netif

// Link monitor module provides real-time network interface state tracking, detecting changes
// in link status, speed, duplex, and other physical layer attributes across Linux and macOS.

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// LinkState represents the current link state.
type LinkState int

// Link state constants for interface monitoring.
const (
	LinkStateUnknown LinkState = iota
	LinkStateDown
	LinkStateUp
)

// Link monitor configuration constants.
const (
	// defaultPollIntervalMs is the default polling interval in milliseconds
	// for checking link state changes. 500ms provides responsive detection
	// without excessive CPU usage.
	defaultPollIntervalMs = 500

	// defaultEventBufferSize is the maximum number of link events to retain
	// in history. Keeps memory bounded while providing useful diagnostics.
	defaultEventBufferSize = 100

	// defaultMinCallbackGapMs is the minimum interval in milliseconds between
	// callback invocations to prevent callback storms during rapid state changes.
	defaultMinCallbackGapMs = 100

	// linkCheckPollIntervalMs is the polling interval in milliseconds used when
	// actively waiting for link state changes (e.g., in WaitForLinkUp).
	linkCheckPollIntervalMs = 100
)

func (s LinkState) String() string {
	switch s {
	case LinkStateUp:
		return "up"
	case LinkStateDown:
		return "down"
	case LinkStateUnknown:
		return "unknown"
	}
	return "unknown"
}

// LinkEvent represents a link state change event.
type LinkEvent struct {
	Interface string
	State     LinkState
	Timestamp time.Time
}

// LinkStateCallback is called when link state changes.
type LinkStateCallback func(event LinkEvent)

// LinkMonitor watches for link state changes on an interface.
type LinkMonitor struct {
	interfaceName string
	callbacks     []LinkStateCallback
	lastState     LinkState
	mu            sync.RWMutex
	stopCh        chan struct{}
	running       bool
	pollInterval  time.Duration
	// History tracking
	history    []LinkEvent
	maxHistory int
	startTime  time.Time
	// Rate limiting for callback goroutines (fixes #857)
	lastCallbackTime time.Time
	minCallbackGap   time.Duration // Minimum time between callback bursts
}

// NewLinkMonitor creates a new link state monitor.
func NewLinkMonitor(interfaceName string) *LinkMonitor {
	return &LinkMonitor{
		interfaceName:  interfaceName,
		callbacks:      make([]LinkStateCallback, 0),
		lastState:      LinkStateUnknown,
		pollInterval:   defaultPollIntervalMs * time.Millisecond,
		history:        make([]LinkEvent, 0),
		maxHistory:     defaultEventBufferSize,
		startTime:      time.Now(),
		minCallbackGap: defaultMinCallbackGapMs * time.Millisecond,
	}
}

// OnStateChange registers a callback for link state changes.
func (m *LinkMonitor) OnStateChange(callback LinkStateCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// Start begins monitoring link state.
func (m *LinkMonitor) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.stopCh = make(chan struct{})
	stopCh := m.stopCh
	m.running = true
	m.mu.Unlock()

	// Get initial state
	m.lastState = m.checkLinkState()

	// pollLoop reads stopCh as a parameter, not via m.stopCh, so Stop's
	// close-then-reassign cycle on a future Start can't race the read.
	go m.pollLoop(stopCh)
	return nil
}

// Stop stops monitoring.
func (m *LinkMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false
}

// SetInterface changes the monitored interface.
func (m *LinkMonitor) SetInterface(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
	m.lastState = LinkStateUnknown
}

// GetState returns the current link state.
func (m *LinkMonitor) GetState() LinkState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastState
}

// IsUp returns true if the link is up.
func (m *LinkMonitor) IsUp() bool {
	return m.GetState() == LinkStateUp
}

// stateChangeResult holds the result of processing a link state change.
type stateChangeResult struct {
	event        LinkEvent
	callbacks    []LinkStateCallback
	shouldNotify bool
}

// pollLoop continuously checks link state. stopCh is a parameter (not read
// from m.stopCh) so the goroutine works on its captured channel.
func (m *LinkMonitor) pollLoop(stopCh <-chan struct{}) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			m.handlePollTick()
		}
	}
}

// handlePollTick processes a single poll tick, checking for state changes.
func (m *LinkMonitor) handlePollTick() {
	newState := m.checkLinkState()

	result, changed := m.processStateChange(newState)
	if !changed {
		return
	}

	if !result.shouldNotify {
		logging.GetLogger().Debug("Link state change rate limited",
			"interface", result.event.Interface,
			"state", result.event.State.String())
		return
	}

	m.notifyCallbacks(result.event, result.callbacks)
}

// processStateChange handles state transition logic under lock.
// Returns the result and whether a state change occurred.
func (m *LinkMonitor) processStateChange(newState LinkState) (stateChangeResult, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if newState == m.lastState {
		return stateChangeResult{}, false
	}

	m.lastState = newState

	event := LinkEvent{
		Interface: m.interfaceName,
		State:     newState,
		Timestamp: time.Now(),
	}

	m.recordEvent(event)

	callbacks := make([]LinkStateCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)

	shouldNotify := m.checkAndUpdateRateLimit()

	return stateChangeResult{
		event:        event,
		callbacks:    callbacks,
		shouldNotify: shouldNotify,
	}, true
}

// recordEvent adds an event to history, maintaining max size.
// Must be called with mu held.
func (m *LinkMonitor) recordEvent(event LinkEvent) {
	m.history = append(m.history, event)
	if len(m.history) > m.maxHistory {
		m.history = m.history[1:]
	}
}

// checkAndUpdateRateLimit checks if callbacks should be notified based on rate limiting.
// Returns true if notification is allowed, and updates the last callback time.
// Must be called with mu held.
func (m *LinkMonitor) checkAndUpdateRateLimit() bool {
	now := time.Now()
	if now.Sub(m.lastCallbackTime) < m.minCallbackGap {
		return false
	}
	m.lastCallbackTime = now
	return true
}

// notifyCallbacks invokes all callbacks asynchronously with panic recovery.
func (m *LinkMonitor) notifyCallbacks(event LinkEvent, callbacks []LinkStateCallback) {
	for _, cb := range callbacks {
		go m.safeInvokeCallback(cb, event)
	}
}

// safeInvokeCallback invokes a callback with panic recovery (fixes #790).
func (m *LinkMonitor) safeInvokeCallback(callback LinkStateCallback, event LinkEvent) {
	defer func() {
		if r := recover(); r != nil {
			logging.GetLogger().Error("Panic in link monitor callback",
				"panic", r,
				"interface", event.Interface,
				"state", event.State.String(),
				"stack", string(debug.Stack()))
		}
	}()
	callback(event)
}

// checkLinkState reads the current link state from the system.
func (m *LinkMonitor) checkLinkState() LinkState {
	switch runtime.GOOS {
	case "linux":
		return m.checkLinkStateLinux()
	case "darwin":
		return m.checkLinkStateDarwin()
	default:
		return LinkStateUnknown
	}
}

// checkLinkStateLinux reads carrier state from sysfs.
func (m *LinkMonitor) checkLinkStateLinux() LinkState {
	// Read /sys/class/net/<iface>/carrier
	carrierPath := filepath.Join("sys", "class", "net", m.interfaceName, "carrier")
	carrierPath = string(os.PathSeparator) + carrierPath

	data, err := os.ReadFile(carrierPath)
	if err != nil {
		// Interface might not exist or no carrier file
		return LinkStateUnknown
	}

	carrier := strings.TrimSpace(string(data))
	if carrier == "1" {
		return LinkStateUp
	}
	return LinkStateDown
}

// checkLinkStateDarwin checks link state on macOS using [net.Interface].
func (m *LinkMonitor) checkLinkStateDarwin() LinkState {
	iface, err := net.InterfaceByName(m.interfaceName)
	if err != nil {
		return LinkStateUnknown
	}

	// Check if interface is UP and RUNNING (has carrier)
	// net.FlagUp means administratively up
	// net.FlagRunning means operationally up (link active)
	if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0 {
		return LinkStateUp
	}

	// If interface is up but not running, link is down
	if iface.Flags&net.FlagUp != 0 {
		return LinkStateDown
	}

	return LinkStateUnknown
}

// GetHistory returns the recent link state change events.
func (m *LinkMonitor) GetHistory() []LinkEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]LinkEvent, len(m.history))
	copy(result, m.history)
	return result
}

// GetFlapCount24h returns the number of link state changes in the last 24 hours.
func (m *LinkMonitor) GetFlapCount24h() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	count := 0
	for _, event := range m.history {
		if event.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

// GetUptime returns how long the monitor has been running.
func (m *LinkMonitor) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.startTime)
}

// WaitForLinkUp blocks until link comes up or timeout.
func (m *LinkMonitor) WaitForLinkUp(timeout time.Duration) bool {
	if m.IsUp() {
		return true
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.checkLinkState() == LinkStateUp {
			return true
		}
		time.Sleep(linkCheckPollIntervalMs * time.Millisecond)
	}
	return false
}
