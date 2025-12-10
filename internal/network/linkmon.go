// Package network handles network interface management.
package network

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
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
	//nolint:gosec // G304: carrierPath is constructed from validated interface name in sysfs
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

// checkLinkStateDarwin checks link state on macOS using net.Interface.
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
