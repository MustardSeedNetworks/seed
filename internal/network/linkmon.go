// Package network handles network interface management.
package network

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/netscope/internal/validation"
)

// LinkState represents the current link state.
type LinkState int

const (
	LinkStateUnknown LinkState = iota
	LinkStateDown
	LinkStateUp
)

func (s LinkState) String() string {
	switch s {
	case LinkStateUp:
		return "up"
	case LinkStateDown:
		return "down"
	default:
		return "unknown"
	}
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
}

// NewLinkMonitor creates a new link state monitor.
func NewLinkMonitor(interfaceName string) *LinkMonitor {
	return &LinkMonitor{
		interfaceName: interfaceName,
		callbacks:     make([]LinkStateCallback, 0),
		lastState:     LinkStateUnknown,
		pollInterval:  500 * time.Millisecond, // Check every 500ms
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
	m.running = true
	m.mu.Unlock()

	// Get initial state
	m.lastState = m.checkLinkState()

	go m.pollLoop()
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

// pollLoop continuously checks link state.
func (m *LinkMonitor) pollLoop() {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			newState := m.checkLinkState()

			m.mu.Lock()
			oldState := m.lastState
			if newState != oldState {
				m.lastState = newState
				callbacks := make([]LinkStateCallback, len(m.callbacks))
				copy(callbacks, m.callbacks)
				m.mu.Unlock()

				// Notify callbacks
				event := LinkEvent{
					Interface: m.interfaceName,
					State:     newState,
					Timestamp: time.Now(),
				}
				for _, cb := range callbacks {
					go cb(event)
				}
			} else {
				m.mu.Unlock()
			}
		}
	}
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

// checkLinkStateDarwin checks link state on macOS using ifconfig.
func (m *LinkMonitor) checkLinkStateDarwin() LinkState {
	if err := validation.ValidateInterface(m.interfaceName); err != nil {
		return LinkStateUnknown
	}

	// #nosec G204 - interface name validated above
	cmd := exec.Command("ifconfig", m.interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return LinkStateUnknown
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for "status: active" or "status: inactive"
		if strings.Contains(line, "status:") {
			if strings.Contains(line, "active") {
				return LinkStateUp
			}
			return LinkStateDown
		}
	}

	// Fallback: check if interface has "RUNNING" flag
	for _, line := range lines {
		if strings.Contains(line, "flags=") && strings.Contains(line, "RUNNING") {
			return LinkStateUp
		}
	}

	return LinkStateUnknown
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
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
