// Package gateway provides gateway ping testing functionality.
package gateway

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/krisarmstrong/netscope/internal/discovery"
)

// Status represents the status of a gateway ping operation.
type Status string

const (
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
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
		Warning:  50 * time.Millisecond,
		Critical: 200 * time.Millisecond,
		LossWarn: 5.0,  // 5% packet loss
		LossCrit: 20.0, // 20% packet loss
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
	pinger      *discovery.ICMPPinger // Raw ICMP pinger (nil if unavailable)
}

// NewTester creates a new gateway tester.
func NewTester(thresholds Thresholds) *Tester {
	t := &Tester{
		thresholds:  thresholds,
		pingCount:   3, // 3 pings matches the Min/Avg/Max display
		pingTimeout: 2 * time.Second,
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

	// Use raw ICMP pinger if available
	if pinger != nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		pingResult := pinger.Ping(ctx, gateway)
		if pingResult.Reachable {
			result.Success = true
			result.Time = pingResult.RTT
			result.TimeMs = float64(pingResult.RTT.Microseconds()) / 1000.0
		} else {
			result.Success = false
			if pingResult.Error != nil {
				result.Error = pingResult.Error.Error()
			} else {
				result.Error = "ping timeout"
			}
		}
		return result
	}

	// Fallback: pinger unavailable (no CAP_NET_RAW)
	result.Success = false
	result.Error = "ICMP pinger unavailable - requires CAP_NET_RAW"
	return result
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

	for i := 0; i < count; i++ {
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
			time.Sleep(200 * time.Millisecond)
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
		stats.LossPercent = float64(stats.Lost) / float64(stats.Sent) * 100
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
	t.mu.Unlock()

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
				stats := t.Test()
				if callback != nil {
					callback(stats)
				}
			case <-t.stopCh:
				return
			}
		}
	}()
}

// StopContinuous stops continuous ping testing.
func (t *Tester) StopContinuous() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		close(t.stopCh)
		t.running = false
	}
}

// Close closes the tester and releases resources.
func (t *Tester) Close() {
	t.StopContinuous()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pinger != nil {
		t.pinger.Close()
		t.pinger = nil
	}
}

// IsRunning returns whether continuous testing is active.
func (t *Tester) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}
