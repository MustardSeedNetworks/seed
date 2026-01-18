// Package link provides network interface link status monitoring and utilities.
// It wraps the lower-level network.LinkMonitor to provide a higher-level API
// for the sap module.
package link

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

// State represents the link operational state.
type State string

// Link state constants.
const (
	StateUp      State = "up"
	StateDown    State = "down"
	StateDormant State = "dormant"
	StateUnknown State = "unknown"
)

// Duplex represents the link duplex mode.
type Duplex string

// Duplex mode constants.
const (
	DuplexFull    Duplex = "full"
	DuplexHalf    Duplex = "half"
	DuplexUnknown Duplex = "unknown"
)

// Speed represents common link speeds.
type Speed int

// Link speed constants in Mbps.
const (
	Speed10     Speed = 10
	Speed100    Speed = 100
	Speed1000   Speed = 1000
	Speed2500   Speed = 2500
	Speed5000   Speed = 5000
	Speed10000  Speed = 10000
	Speed25000  Speed = 25000
	Speed40000  Speed = 40000
	Speed100000 Speed = 100000
)

const (
	bpsPerMbps  = 1_000_000
	mbpsPerGbps = 1000
)

// Bps converts speed to bits per second.
func (s Speed) Bps() int64 {
	return int64(s) * bpsPerMbps
}

// String returns a human-readable speed string.
func (s Speed) String() string {
	switch {
	case s >= mbpsPerGbps:
		return fmt.Sprintf("%d Gbps", s/mbpsPerGbps)
	default:
		return fmt.Sprintf("%d Mbps", s)
	}
}

// Time-related constants for link monitoring.
const (
	// DefaultPollInterval is the default interval for polling link state.
	DefaultPollInterval = 500 * time.Millisecond

	// DefaultEventBufferSize is the maximum number of events to retain.
	DefaultEventBufferSize = 100

	// WaitPollInterval is the polling interval when waiting for link state changes.
	WaitPollInterval = 100 * time.Millisecond
)

// Status contains detailed link status information.
type Status struct {
	Interface  string    `json:"interface"`
	State      State     `json:"state"`
	Speed      Speed     `json:"speed"`
	SpeedStr   string    `json:"speedStr"`
	Duplex     Duplex    `json:"duplex"`
	MTU        int       `json:"mtu"`
	MACAddress string    `json:"macAddress"`
	Carrier    bool      `json:"carrier"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Event represents a link state change event.
type Event struct {
	Interface string    `json:"interface"`
	OldState  State     `json:"oldState"`
	NewState  State     `json:"newState"`
	Timestamp time.Time `json:"timestamp"`
}

// EventCallback is invoked when a link state changes.
type EventCallback func(Event)

// Monitor watches for link state changes on a network interface.
type Monitor struct {
	interfaceName string
	state         State
	callbacks     []EventCallback
	history       []Event
	maxHistory    int
	pollInterval  time.Duration
	running       bool
	stopCh        chan struct{}
	startTime     time.Time
	mu            sync.RWMutex
}

// NewMonitor creates a new link monitor for the specified interface.
func NewMonitor(interfaceName string) *Monitor {
	return &Monitor{
		interfaceName: interfaceName,
		state:         StateUnknown,
		callbacks:     make([]EventCallback, 0),
		history:       make([]Event, 0),
		maxHistory:    DefaultEventBufferSize,
		pollInterval:  DefaultPollInterval,
	}
}

// SetInterface changes the monitored interface.
func (m *Monitor) SetInterface(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
	m.state = StateUnknown
}

// GetInterface returns the currently monitored interface name.
func (m *Monitor) GetInterface() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.interfaceName
}

// GetState returns the current link state.
func (m *Monitor) GetState() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// IsUp returns true if the link is in the up state.
func (m *Monitor) IsUp() bool {
	return m.GetState() == StateUp
}

// IsDown returns true if the link is in the down state.
func (m *Monitor) IsDown() bool {
	return m.GetState() == StateDown
}

// OnStateChange registers a callback for link state changes.
func (m *Monitor) OnStateChange(callback EventCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// Start begins monitoring the interface for link state changes.
func (m *Monitor) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.stopCh = make(chan struct{})
	m.running = true
	m.startTime = time.Now()
	m.state = checkLinkState(m.interfaceName)
	m.mu.Unlock()

	go m.pollLoop()
	return nil
}

// Stop halts link monitoring.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false
}

// IsRunning returns whether the monitor is actively running.
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetHistory returns the event history.
func (m *Monitor) GetHistory() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Event, len(m.history))
	copy(result, m.history)
	return result
}

// GetFlapCount returns the number of state changes in the given duration.
func (m *Monitor) GetFlapCount(duration time.Duration) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().Add(-duration)
	count := 0
	for _, event := range m.history {
		if event.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

// GetUptime returns how long the monitor has been running.
func (m *Monitor) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.startTime.IsZero() {
		return 0
	}
	return time.Since(m.startTime)
}

// WaitForState blocks until the link reaches the specified state or timeout.
func (m *Monitor) WaitForState(target State, timeout time.Duration) bool {
	if m.GetState() == target {
		return true
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if checkLinkState(m.interfaceName) == target {
			m.mu.Lock()
			m.state = target
			m.mu.Unlock()
			return true
		}
		time.Sleep(WaitPollInterval)
	}
	return false
}

// WaitForUp blocks until the link comes up or timeout.
func (m *Monitor) WaitForUp(timeout time.Duration) bool {
	return m.WaitForState(StateUp, timeout)
}

// WaitForDown blocks until the link goes down or timeout.
func (m *Monitor) WaitForDown(timeout time.Duration) bool {
	return m.WaitForState(StateDown, timeout)
}

// pollLoop continuously checks link state.
func (m *Monitor) pollLoop() {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAndNotify()
		}
	}
}

// checkAndNotify checks current link state and notifies callbacks if changed.
func (m *Monitor) checkAndNotify() {
	newState := checkLinkState(m.interfaceName)

	m.mu.Lock()
	oldState := m.state
	if newState == oldState {
		m.mu.Unlock()
		return
	}

	m.state = newState

	event := Event{
		Interface: m.interfaceName,
		OldState:  oldState,
		NewState:  newState,
		Timestamp: time.Now(),
	}

	// Record in history
	m.history = append(m.history, event)
	if len(m.history) > m.maxHistory {
		m.history = m.history[1:]
	}

	// Copy callbacks for safe iteration outside lock
	callbacks := make([]EventCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.Unlock()

	// Notify callbacks
	for _, cb := range callbacks {
		go func(callback EventCallback) {
			defer func() { _ = recover() }()
			callback(event)
		}(cb)
	}
}

// checkLinkState reads the current link state from the system.
func checkLinkState(interfaceName string) State {
	return checkLinkStatePlatform(interfaceName)
}

// GetStatus returns detailed status information for an interface.
func GetStatus(interfaceName string) (*Status, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %w", interfaceName, err)
	}

	status := &Status{
		Interface:  iface.Name,
		MTU:        iface.MTU,
		MACAddress: iface.HardwareAddr.String(),
		UpdatedAt:  time.Now(),
	}

	// Determine state from flags
	switch {
	case iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0:
		status.State = StateUp
		status.Carrier = true
	case iface.Flags&net.FlagUp != 0:
		status.State = StateDown
		status.Carrier = false
	default:
		status.State = StateUnknown
		status.Carrier = false
	}

	// Get speed and duplex from platform-specific methods
	status.Speed, status.Duplex = getSpeedDuplex(interfaceName)
	status.SpeedStr = status.Speed.String()

	return status, nil
}

// ListInterfaces returns status for all network interfaces.
func ListInterfaces() ([]Status, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("listing interfaces: %w", err)
	}

	result := make([]Status, 0, len(ifaces))
	for _, iface := range ifaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		status := Status{
			Interface:  iface.Name,
			MTU:        iface.MTU,
			MACAddress: iface.HardwareAddr.String(),
			UpdatedAt:  time.Now(),
		}

		switch {
		case iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0:
			status.State = StateUp
			status.Carrier = true
		case iface.Flags&net.FlagUp != 0:
			status.State = StateDown
			status.Carrier = false
		default:
			status.State = StateUnknown
			status.Carrier = false
		}

		status.Speed, status.Duplex = getSpeedDuplex(iface.Name)
		status.SpeedStr = status.Speed.String()

		result = append(result, status)
	}

	return result, nil
}

// IsPhysicalInterface returns true if the interface appears to be physical hardware.
func IsPhysicalInterface(name string) bool {
	return isPhysicalInterfacePlatform(name)
}

// ParseSpeed parses a speed string into a Speed value.
func ParseSpeed(s string) Speed {
	return parseSpeedPlatform(s)
}

// ParseDuplex parses a duplex string into a Duplex value.
func ParseDuplex(s string) Duplex {
	switch s {
	case "full", "Full":
		return DuplexFull
	case "half", "Half":
		return DuplexHalf
	default:
		return DuplexUnknown
	}
}

// ParseState parses a state string into a State value.
func ParseState(s string) State {
	switch s {
	case "up", "UP":
		return StateUp
	case "down", "DOWN":
		return StateDown
	case "dormant", "DORMANT":
		return StateDormant
	default:
		return StateUnknown
	}
}

// Platform returns the current operating system platform.
func Platform() string {
	return runtime.GOOS
}
