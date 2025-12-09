// Package gateway provides gateway ping testing functionality.
package gateway

import (
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
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
}

// NewTester creates a new gateway tester.
func NewTester(thresholds Thresholds) *Tester {
	return &Tester{
		thresholds:  thresholds,
		pingCount:   3, // 3 pings matches the Min/Avg/Max display
		pingTimeout: 2 * time.Second,
		stats:       &PingStats{Status: StatusUnknown},
		stopCh:      make(chan struct{}),
	}
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
func DetectGateway() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectGatewayDarwin()
	case "linux":
		return detectGatewayLinux()
	default:
		return "", nil
	}
}

// detectGatewayDarwin detects the default gateway on macOS.
func detectGatewayDarwin() (string, error) {
	// Use netstat -rn to get the default route
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output looking for default route
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "default" {
			// The gateway is typically the second field
			gateway := fields[1]
			// Validate it's an IP address
			if net.ParseIP(gateway) != nil {
				return gateway, nil
			}
		}
	}

	// Try route -n get default
	cmd = exec.Command("route", "-n", "get", "default")
	output, err = cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse "gateway:" line
	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			gateway := strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			if net.ParseIP(gateway) != nil {
				return gateway, nil
			}
		}
	}

	return "", nil
}

// detectGatewayLinux detects the default gateway on Linux.
func detectGatewayLinux() (string, error) {
	// Try ip route
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to reading /proc/net/route
		return detectGatewayFromProc()
	}

	// Parse "default via X.X.X.X"
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "via" && i+1 < len(fields) {
			gateway := fields[i+1]
			if net.ParseIP(gateway) != nil {
				return gateway, nil
			}
		}
	}

	return "", nil
}

// detectGatewayFromProc reads gateway from /proc/net/route on Linux.
func detectGatewayFromProc() (string, error) {
	// This is a fallback - in production we'd parse /proc/net/route
	return "", nil
}

// DetectGatewayIPv6 attempts to detect the default IPv6 gateway.
func DetectGatewayIPv6() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectGatewayIPv6Darwin()
	case "linux":
		return detectGatewayIPv6Linux()
	default:
		return "", nil
	}
}

// detectGatewayIPv6Darwin detects the default IPv6 gateway on macOS.
func detectGatewayIPv6Darwin() (string, error) {
	// Use netstat -rn to get the default IPv6 route
	cmd := exec.Command("netstat", "-rn", "-f", "inet6")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output looking for default route
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "default" {
			gateway := fields[1]
			// Remove interface scope suffix (e.g., %en0)
			if idx := strings.Index(gateway, "%"); idx > 0 {
				gateway = gateway[:idx]
			}
			// Validate it's an IPv6 address
			if ip := net.ParseIP(gateway); ip != nil && ip.To4() == nil {
				return gateway, nil
			}
		}
	}

	return "", nil
}

// detectGatewayIPv6Linux detects the default IPv6 gateway on Linux.
func detectGatewayIPv6Linux() (string, error) {
	// Try ip -6 route
	cmd := exec.Command("ip", "-6", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	// Parse "default via XXXX:XXXX::X"
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "via" && i+1 < len(fields) {
			gateway := fields[i+1]
			if ip := net.ParseIP(gateway); ip != nil && ip.To4() == nil {
				return gateway, nil
			}
		}
	}

	return "", nil
}

// Ping performs a single ping to the gateway.
func (t *Tester) Ping() *PingResult {
	t.mu.RLock()
	gateway := t.gateway
	timeout := t.pingTimeout
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

	start := time.Now()

	if net.ParseIP(gateway) == nil {
		result.Success = false
		result.Error = "invalid gateway address"
		return result
	}

	// Use system ping command for ICMP (requires no special privileges on most systems)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// #nosec G204 - gateway validated above
		cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(int(timeout.Milliseconds())), gateway)
	case "linux":
		// #nosec G204 - gateway validated above
		cmd = exec.Command("ping", "-c", "1", "-W", strconv.Itoa(int(timeout.Seconds())), gateway)
	default:
		// #nosec G204 - gateway validated above
		cmd = exec.Command("ping", "-c", "1", gateway)
	}

	output, err := cmd.Output()
	elapsed := time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = "ping failed"
		result.Time = elapsed
		result.TimeMs = float64(elapsed.Milliseconds())
		return result
	}

	// Parse the RTT from ping output
	rtt := parsePingRTT(string(output))
	if rtt > 0 {
		result.Time = time.Duration(rtt * float64(time.Millisecond))
		result.TimeMs = rtt
	} else {
		result.Time = elapsed
		result.TimeMs = float64(elapsed.Milliseconds())
	}
	result.Success = true

	return result
}

// parsePingRTT extracts the RTT value from ping output.
func parsePingRTT(output string) float64 {
	// Look for patterns like "time=X.XX ms" or "time=X ms"
	re := regexp.MustCompile(`time[=<](\d+\.?\d*)\s*ms`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		rtt, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return rtt
		}
	}
	return 0
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

// IsRunning returns whether continuous testing is active.
func (t *Tester) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}
