package api

// health_checks_ping.go contains ping-based reachability tests for health checks.

import (
	"context"
	"errors"
	"math"
	"net"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// PingStats holds extended ping statistics.
type PingStats struct {
	AvgLatency float64 // ms
	MinLatency float64 // ms
	MaxLatency float64 // ms
	PacketLoss float64 // percentage
	Jitter     float64 // ms (standard deviation)
}

// runPingTests runs all configured ping tests and returns results.
func (s *Server) runPingTests() []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.PingTargets))
	threshold := s.config.Thresholds.CustomTests.Ping

	for _, target := range s.config.HealthChecks.PingTargets {
		if !target.Enabled {
			continue
		}

		name := target.Name
		if name == "" {
			name = target.Host
		}

		testResult := CustomTestResult{Name: name, Host: target.Host}
		pingStats, err := runExtendedPing(target.Host, defaultPingCount)

		if err != nil {
			testResult.Success = false
			testResult.Error = "Ping test failed"
			testResult.TestStatus = statusError
		} else {
			testResult.Success = pingStats.PacketLoss < packetLossThresholdFull
			testResult.Latency = pingStats.AvgLatency
			testResult.MinLatency = pingStats.MinLatency
			testResult.MaxLatency = pingStats.MaxLatency
			testResult.PacketLoss = pingStats.PacketLoss
			testResult.Jitter = pingStats.Jitter
			testResult.TestStatus = s.evaluatePingStatus(pingStats, threshold)
		}
		results = append(results, testResult)
	}
	return results
}

// evaluatePingStatus determines ping test status based on packet loss and latency.
func (s *Server) evaluatePingStatus(stats *PingStats, threshold config.Threshold) string {
	switch {
	case stats.PacketLoss > packetLossThresholdHigh:
		return statusError
	case stats.PacketLoss > packetLossThresholdLow:
		return statusWarning
	default:
		return getTestStatus(
			stats.AvgLatency,
			threshold.Warning.Milliseconds(),
			threshold.Critical.Milliseconds(),
		)
	}
}

// runExtendedPing runs multiple pings and returns statistics.
func runExtendedPing(host string, count int) (*PingStats, error) {
	var latencies []float64
	sent := 0
	received := 0

	for i := range count {
		sent++
		ctx, cancel := context.WithTimeout(context.Background(), pingProbeTimeoutSec*time.Second)

		start := time.Now()
		// Try TCP 80/443 as ping alternative (actual ICMP requires root)
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host+":80")
		if err != nil {
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", host+":443")
		}
		cancel()

		if err == nil {
			latency := time.Since(start).Seconds() * millisecondsPerSecond
			latencies = append(latencies, latency)
			received++
			_ = conn.Close()
		}

		// Small delay between pings
		if i < count-1 {
			time.Sleep(pingProbeDelayMs * time.Millisecond)
		}
	}

	if len(latencies) == 0 {
		return &PingStats{PacketLoss: packetLossThresholdFull}, errors.New("host unreachable")
	}

	// Calculate statistics
	stats := &PingStats{
		PacketLoss: float64(sent-received) / float64(sent) * percentageDivisor,
	}

	// Min, max, avg
	stats.MinLatency = latencies[0]
	stats.MaxLatency = latencies[0]
	var sum float64
	for _, lat := range latencies {
		sum += lat
		if lat < stats.MinLatency {
			stats.MinLatency = lat
		}
		if lat > stats.MaxLatency {
			stats.MaxLatency = lat
		}
	}
	stats.AvgLatency = sum / float64(len(latencies))

	// Jitter (standard deviation)
	if len(latencies) > 1 {
		var variance float64
		for _, lat := range latencies {
			diff := lat - stats.AvgLatency
			variance += diff * diff
		}
		stats.Jitter = math.Sqrt(variance / float64(len(latencies)))
	}

	return stats, nil
}
