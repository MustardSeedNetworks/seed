package gateway

// Implements ICMP-based ping tests to verify gateway connectivity, measure round-trip times,
// and detect gateway availability issues. Supports sequential and continuous gateway monitoring.

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/services/discovery"
)

// Status represents the status of a gateway ping operation.
type Status string

// Gateway ping status constants.
const (
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

// Time conversion and threshold constants.
const (
	// microsecondsPerMillisecond is the number of microseconds in one millisecond,
	// used for converting [time.Duration.Microseconds]() to milliseconds.
	microsecondsPerMillisecond = 1000.0

	// pingIntervalDelay is the delay between consecutive ping packets
	// to avoid flooding the network and allow for accurate timing measurements.
	pingIntervalDelay = 200 * time.Millisecond

	// defaultWarningLatencyMs is the default latency threshold (in ms) for warning status.
	defaultWarningLatencyMs = 50

	// defaultCriticalLatencyMs is the default latency threshold (in ms) for critical status.
	defaultCriticalLatencyMs = 200

	// defaultLossWarningPercent is the default packet loss percentage for warning status.
	defaultLossWarningPercent = 5.0

	// defaultLossCriticalPercent is the default packet loss percentage for critical status.
	defaultLossCriticalPercent = 20.0

	// defaultPingCount is the number of ping probes sent per test cycle.
	defaultPingCount = 3

	// defaultPingTimeoutSec is the timeout in seconds for each ping probe.
	defaultPingTimeoutSec = 2

	// percentMultiplier converts a ratio to percentage (0-100 scale).
	percentMultiplier = 100
)

// PingResult contains the result of a single ping.
type PingResult struct {
	Sequence int           `json:"sequence"`
	Time     time.Duration `json:"time"`
	TimeMs   float64       `json:"timeMs"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
}

// PingStats contains aggregated ping statistics.
type PingStats struct {
	Gateway     string       `json:"gateway"`
	Sent        int          `json:"sent"`
	Received    int          `json:"received"`
	Lost        int          `json:"lost"`
	LossPercent float64      `json:"lossPercent"`
	MinTime     float64      `json:"minTime"`  // ms
	MaxTime     float64      `json:"maxTime"`  // ms
	AvgTime     float64      `json:"avgTime"`  // ms
	LastTime    float64      `json:"lastTime"` // ms
	Status      Status       `json:"status"`
	Reachable   bool         `json:"reachable"`
	Results     []PingResult `json:"results,omitempty"`
	LastUpdated time.Time    `json:"lastUpdated"`
	IPv6        *PingStats   `json:"ipv6,omitempty"` // IPv6 gateway stats
}

// Thresholds defines timing thresholds for gateway ping.
type Thresholds struct {
	Warning  time.Duration // Latency threshold for warning
	Critical time.Duration // Latency threshold for error
	LossWarn float64       // Packet loss % threshold for warning
	LossCrit float64       // Packet loss % threshold for error
}

// DefaultThresholds returns reasonable default thresholds.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Warning:  defaultWarningLatencyMs * time.Millisecond,
		Critical: defaultCriticalLatencyMs * time.Millisecond,
		LossWarn: defaultLossWarningPercent,
		LossCrit: defaultLossCriticalPercent,
	}
}

// Tester performs gateway ping tests.
type Tester struct {
	gateway     string
	thresholds  Thresholds
	pingCount   int
	pingTimeout time.Duration
	stats       *PingStats
	mu          sync.RWMutex
	stopCh      chan struct{}
	running     bool
	stopOnce    sync.Once             // Prevents double-close panic on stopCh (fixes #854)
	pinger      *discovery.ICMPPinger // Raw ICMP pinger (nil if unavailable)
}

// NewTester creates a new gateway tester.
func NewTester(thresholds Thresholds) *Tester {
	t := &Tester{
		thresholds:  thresholds,
		pingCount:   defaultPingCount,
		pingTimeout: defaultPingTimeoutSec * time.Second,
		stats:       &PingStats{Status: StatusUnknown},
		stopCh:      make(chan struct{}),
	}

	// Try to create ICMP pinger (requires CAP_NET_RAW or root)
	pinger, err := discovery.NewICMPPinger(t.pingTimeout)
	if err == nil {
		t.pinger = pinger
	}

	return t
}

// SetGateway updates the gateway address to ping.
func (t *Tester) SetGateway(gateway string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gateway = gateway
}

// GetGateway returns the current gateway address.
func (t *Tester) GetGateway() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.gateway
}

// GetStats returns the current ping statistics.
func (t *Tester) GetStats() *PingStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.stats == nil {
		return &PingStats{Status: StatusUnknown}
	}
	// Return a copy
	statsCopy := *t.stats
	return &statsCopy
}

// DetectGateway attempts to detect the default gateway.
// Uses netlink on Linux, exec commands on macOS.
func DetectGateway() (string, error) {
	return detectGatewayPlatform()
}

// DetectGatewayIPv6 attempts to detect the default IPv6 gateway.
// Uses netlink on Linux, exec commands on macOS.
func DetectGatewayIPv6() (string, error) {
	return detectGatewayIPv6Platform()
}

// Ping performs a single ping to the gateway using raw ICMP sockets.
func (t *Tester) Ping() *PingResult {
	t.mu.RLock()
	gateway := t.gateway
	timeout := t.pingTimeout
	pinger := t.pinger
	t.mu.RUnlock()

	if gateway == "" {
		return &PingResult{
			Success: false,
			Error:   "no gateway configured",
		}
	}

	result := &PingResult{
		Sequence: 1,
	}

	if net.ParseIP(gateway) == nil {
		result.Success = false
		result.Error = "invalid gateway address"
		return result
	}

	// Fallback: pinger unavailable (no CAP_NET_RAW)
	if pinger == nil {
		result.Success = false
		result.Error = "ICMP pinger unavailable - requires CAP_NET_RAW"
		return result
	}

	// Use raw ICMP pinger
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pingResult := pinger.Ping(ctx, gateway)
	if !pingResult.Reachable {
		result.Success = false
		result.Error = pingErrorMessage(pingResult.Error)
		return result
	}

	result.Success = true
	result.Time = pingResult.RTT
	result.TimeMs = float64(pingResult.RTT.Microseconds()) / microsecondsPerMillisecond
	return result
}

// pingErrorMessage returns an appropriate error message for a failed ping.
func pingErrorMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return "ping timeout"
}

// Test performs a complete ping test with multiple packets.
func (t *Tester) Test() *PingStats {
	t.mu.RLock()
	gateway := t.gateway
	count := t.pingCount
	t.mu.RUnlock()

	stats := &PingStats{
		Gateway:     gateway,
		Sent:        0,
		Received:    0,
		Lost:        0,
		Results:     make([]PingResult, 0, count),
		LastUpdated: time.Now(),
	}

	if gateway == "" {
		// Try to detect gateway
		detected, err := DetectGateway()
		if err == nil && detected != "" {
			t.SetGateway(detected)
			gateway = detected
			stats.Gateway = gateway
		} else {
			stats.Status = StatusError
			stats.Reachable = false
			t.mu.Lock()
			t.stats = stats
			t.mu.Unlock()
			return stats
		}
	}

	var totalTime float64
	var minTime, maxTime float64 = -1, 0

	for i := range count {
		result := t.Ping()
		result.Sequence = i + 1
		stats.Sent++

		if result.Success {
			stats.Received++
			totalTime += result.TimeMs

			if minTime < 0 || result.TimeMs < minTime {
				minTime = result.TimeMs
			}
			if result.TimeMs > maxTime {
				maxTime = result.TimeMs
			}
			stats.LastTime = result.TimeMs
		} else {
			stats.Lost++
		}

		stats.Results = append(stats.Results, *result)

		// Small delay between pings
		if i < count-1 {
			time.Sleep(pingIntervalDelay)
		}
	}

	// Calculate statistics
	if stats.Received > 0 {
		stats.AvgTime = totalTime / float64(stats.Received)
		stats.MinTime = minTime
		stats.MaxTime = maxTime
		stats.Reachable = true
	}

	if stats.Sent > 0 {
		stats.LossPercent = float64(stats.Lost) / float64(stats.Sent) * percentMultiplier
	}

	// Determine status
	stats.Status = t.determineStatus(stats)

	// Store stats
	t.mu.Lock()
	t.stats = stats
	t.mu.Unlock()

	return stats
}

// determineStatus calculates the overall status based on thresholds.
func (t *Tester) determineStatus(stats *PingStats) Status {
	if !stats.Reachable || stats.Received == 0 {
		return StatusError
	}

	// Check packet loss first
	if stats.LossPercent >= t.thresholds.LossCrit {
		return StatusError
	}
	if stats.LossPercent >= t.thresholds.LossWarn {
		return StatusWarning
	}

	// Check latency
	avgDuration := time.Duration(stats.AvgTime * float64(time.Millisecond))
	if avgDuration >= t.thresholds.Critical {
		return StatusError
	}
	if avgDuration >= t.thresholds.Warning {
		return StatusWarning
	}

	return StatusSuccess
}

// StartContinuous starts continuous ping testing in the background.
func (t *Tester) StartContinuous(interval time.Duration, callback func(*PingStats)) {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running = true
	t.stopCh = make(chan struct{})
	t.stopOnce = sync.Once{} // Reset Once for new channel (fixes #854)
	stopCh := t.stopCh
	t.mu.Unlock()

	// Capture stopCh into a local so the goroutine doesn't race with a
	// future StartContinuous reassigning t.stopCh.
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run initial test
		stats := t.Test()
		if callback != nil {
			callback(stats)
		}

		for {
			select {
			case <-ticker.C:
				tickStats := t.Test()
				if callback != nil {
					callback(tickStats)
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// StopContinuous stops continuous ping testing.
// Uses [sync.Once] to prevent double-close panic (fixes #854).
func (t *Tester) StopContinuous() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		t.stopOnce.Do(func() {
			close(t.stopCh)
		})
		t.running = false
	}
}

// Close closes the tester and releases resources.
func (t *Tester) Close() {
	t.StopContinuous()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pinger != nil {
		_ = t.pinger.Close()
		t.pinger = nil
	}
}

// IsRunning returns whether continuous testing is active.
func (t *Tester) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}
