// Package gateway_test provides gateway reachability testing and latency measurement.
// Test suite validates gateway detection, ping testing, threshold evaluation,
// and packet loss calculation for gateway health monitoring.
package gateway_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/gateway"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := gateway.DefaultThresholds()

	if thresholds.Warning != 50*time.Millisecond {
		t.Errorf("expected Warning threshold 50ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 200*time.Millisecond {
		t.Errorf("expected Critical threshold 200ms, got %v", thresholds.Critical)
	}
	if thresholds.LossWarn != 5.0 {
		t.Errorf("expected LossWarn 5.0, got %v", thresholds.LossWarn)
	}
	if thresholds.LossCrit != 20.0 {
		t.Errorf("expected LossCrit 20.0, got %v", thresholds.LossCrit)
	}
}

func TestNewTester(t *testing.T) {
	thresholds := gateway.DefaultThresholds()
	tester := gateway.NewTester(thresholds)

	if tester == nil {
		t.Fatal("expected non-nil tester")
	}
	if tester.TesterPingCount() != 3 {
		t.Errorf("expected pingCount 3, got %d", tester.TesterPingCount())
	}
	if tester.TesterPingTimeout() != 2*time.Second {
		t.Errorf("expected pingTimeout 2s, got %v", tester.TesterPingTimeout())
	}
	if tester.TesterStats() == nil {
		t.Error("expected non-nil stats")
	}
	if tester.TesterStats().Status != gateway.StatusUnknown {
		t.Errorf("expected initial status Unknown, got %v", tester.TesterStats().Status)
	}
}

func TestTesterSetGetGateway(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Initially empty.
	if got := tester.GetGateway(); got != "" {
		t.Errorf("expected empty gateway, got %q", got)
	}

	// Set gateway.
	tester.SetGateway("192.168.1.1")
	if got := tester.GetGateway(); got != "192.168.1.1" {
		t.Errorf("expected gateway '192.168.1.1', got %q", got)
	}

	// Update gateway.
	tester.SetGateway("10.0.0.1")
	if got := tester.GetGateway(); got != "10.0.0.1" {
		t.Errorf("expected gateway '10.0.0.1', got %q", got)
	}
}

func TestTesterGetStats(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	stats := tester.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Status != gateway.StatusUnknown {
		t.Errorf("expected initial status Unknown, got %v", stats.Status)
	}

	// Ensure it returns a copy (modifying returned stats shouldn't affect internal state).
	stats.Gateway = "modified"
	internalStats := tester.GetStats()
	if internalStats.Gateway == "modified" {
		t.Error("GetStats should return a copy, not the internal reference")
	}
}

// Note: parsePingRTT tests removed - now using raw ICMP pinger instead of exec.Command ping.

func TestTesterPingNoGateway(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())
	// Don't set a gateway.

	result := tester.Ping()
	if result.Success {
		t.Error("expected failure when no gateway configured")
	}
	if result.Error != "no gateway configured" {
		t.Errorf("expected 'no gateway configured' error, got %q", result.Error)
	}
}

func TestTesterDetermineStatus(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  50 * time.Millisecond,
		Critical: 200 * time.Millisecond,
		LossWarn: 5.0,
		LossCrit: 20.0,
	}
	tester := gateway.NewTester(thresholds)

	tests := []struct {
		name     string
		stats    *gateway.PingStats
		expected gateway.Status
	}{
		{
			name:     "unreachable",
			stats:    &gateway.PingStats{Reachable: false, Received: 0},
			expected: gateway.StatusError,
		},
		{
			name:     "no packets received",
			stats:    &gateway.PingStats{Reachable: true, Received: 0},
			expected: gateway.StatusError,
		},
		{
			name:     "critical packet loss",
			stats:    &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 25.0, AvgTime: 10},
			expected: gateway.StatusError,
		},
		{
			name:     "warning packet loss",
			stats:    &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 10.0, AvgTime: 10},
			expected: gateway.StatusWarning,
		},
		{
			name:     "critical latency",
			stats:    &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 0, AvgTime: 250},
			expected: gateway.StatusError,
		},
		{
			name:     "warning latency",
			stats:    &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 0, AvgTime: 75},
			expected: gateway.StatusWarning,
		},
		{
			name:     "success - good latency no loss",
			stats:    &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 0, AvgTime: 10},
			expected: gateway.StatusSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tester.DetermineStatus(tt.stats)
			if result != tt.expected {
				t.Errorf("DetermineStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	if gateway.StatusSuccess != "success" {
		t.Errorf("expected StatusSuccess = 'success', got %q", gateway.StatusSuccess)
	}
	if gateway.StatusWarning != "warning" {
		t.Errorf("expected StatusWarning = 'warning', got %q", gateway.StatusWarning)
	}
	if gateway.StatusError != "error" {
		t.Errorf("expected StatusError = 'error', got %q", gateway.StatusError)
	}
	if gateway.StatusUnknown != "unknown" {
		t.Errorf("expected StatusUnknown = 'unknown', got %q", gateway.StatusUnknown)
	}
}

func TestPingResultFields(t *testing.T) {
	result := gateway.PingResult{
		Sequence: 1,
		Time:     50 * time.Millisecond,
		TimeMs:   50.0,
		Success:  true,
		Error:    "",
	}

	if result.Sequence != 1 {
		t.Errorf("expected Sequence 1, got %d", result.Sequence)
	}
	if result.Time != 50*time.Millisecond {
		t.Errorf("expected Time 50ms, got %v", result.Time)
	}
	if result.TimeMs != 50.0 {
		t.Errorf("expected TimeMs 50.0, got %v", result.TimeMs)
	}
	if !result.Success {
		t.Error("expected Success true")
	}
	if result.Error != "" {
		t.Errorf("expected empty Error, got %q", result.Error)
	}
}

func TestPingStatsFields(t *testing.T) {
	now := time.Now()
	stats := gateway.PingStats{
		Gateway:     "192.168.1.1",
		Sent:        5,
		Received:    4,
		Lost:        1,
		LossPercent: 20.0,
		MinTime:     10.5,
		MaxTime:     50.3,
		AvgTime:     25.2,
		LastTime:    22.1,
		Status:      gateway.StatusSuccess,
		Reachable:   true,
		Results:     []gateway.PingResult{},
		LastUpdated: now,
	}

	if stats.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", stats.Gateway)
	}
	if stats.Sent != 5 {
		t.Errorf("expected Sent 5, got %d", stats.Sent)
	}
	if stats.Received != 4 {
		t.Errorf("expected Received 4, got %d", stats.Received)
	}
	if stats.Lost != 1 {
		t.Errorf("expected Lost 1, got %d", stats.Lost)
	}
	if stats.LossPercent != 20.0 {
		t.Errorf("expected LossPercent 20.0, got %v", stats.LossPercent)
	}
	if stats.MinTime != 10.5 {
		t.Errorf("expected MinTime 10.5, got %v", stats.MinTime)
	}
	if stats.MaxTime != 50.3 {
		t.Errorf("expected MaxTime 50.3, got %v", stats.MaxTime)
	}
	if stats.AvgTime != 25.2 {
		t.Errorf("expected AvgTime 25.2, got %v", stats.AvgTime)
	}
	if stats.LastTime != 22.1 {
		t.Errorf("expected LastTime 22.1, got %v", stats.LastTime)
	}
	if stats.Status != gateway.StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", stats.Status)
	}
	if !stats.Reachable {
		t.Error("expected Reachable true")
	}
	if len(stats.Results) != 0 {
		t.Errorf("expected empty Results, got %d", len(stats.Results))
	}
	if stats.LastUpdated != now {
		t.Errorf("expected LastUpdated %v, got %v", now, stats.LastUpdated)
	}
}

func TestTesterIsRunning(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	if tester.IsRunning() {
		t.Error("expected IsRunning() to be false initially")
	}
}

func TestTesterStartStopContinuousNotRunning(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Should not panic when stopping a non-running tester.
	tester.StopContinuous()

	if tester.IsRunning() {
		t.Error("expected IsRunning() to be false after StopContinuous")
	}
}

func TestDetectGateway(_ *testing.T) {
	// This may or may not find a gateway depending on system config.
	gw, err := gateway.DetectGateway()
	// Just verify it doesn't panic.
	_ = gw
	_ = err
}

// Note: Platform-specific gateway detection tests removed - now in platform files.

func TestTesterPingWithGateway(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ping test in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	// Use localhost which should always be reachable.
	tester.SetGateway("127.0.0.1")

	result := tester.Ping()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// 127.0.0.1 should be reachable.
	if !result.Success {
		t.Logf("ping to localhost failed: %v (may be blocked by firewall)", result.Error)
	}
}

func TestTesterTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	stats := tester.Test()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Gateway != "127.0.0.1" {
		t.Errorf("expected Gateway '127.0.0.1', got %q", stats.Gateway)
	}
	if stats.Sent == 0 {
		t.Error("expected Sent > 0")
	}
}

func TestTesterTestWithAutoDetect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	// Don't set gateway - let it auto-detect.

	stats := tester.Test()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	// Should have tried to detect or returned error status.
	if stats.Status == "" {
		t.Error("expected non-empty status")
	}
}

func TestTesterStartStopContinuous(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping continuous test in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	callbackCh := make(chan struct{}, 10)
	callback := func(_ *gateway.PingStats) {
		select {
		case callbackCh <- struct{}{}:
		default:
		}
	}

	// Start continuous testing.
	tester.StartContinuous(100*time.Millisecond, callback)

	if !tester.IsRunning() {
		t.Error("expected IsRunning() to be true after StartContinuous")
	}

	// Wait for at least one callback with timeout.
	select {
	case <-callbackCh:
		// Got callback.
	case <-time.After(5 * time.Second):
		// Timeout is okay - test is about start/stop mechanism.
	}

	// Starting again should be no-op.
	tester.StartContinuous(100*time.Millisecond, callback)

	// Stop.
	tester.StopContinuous()

	if tester.IsRunning() {
		t.Error("expected IsRunning() to be false after StopContinuous")
	}
}

func TestThresholdsFields(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  75 * time.Millisecond,
		Critical: 300 * time.Millisecond,
		LossWarn: 10.0,
		LossCrit: 30.0,
	}

	if thresholds.Warning != 75*time.Millisecond {
		t.Errorf("expected Warning 75ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 300*time.Millisecond {
		t.Errorf("expected Critical 300ms, got %v", thresholds.Critical)
	}
	if thresholds.LossWarn != 10.0 {
		t.Errorf("expected LossWarn 10.0, got %v", thresholds.LossWarn)
	}
	if thresholds.LossCrit != 30.0 {
		t.Errorf("expected LossCrit 30.0, got %v", thresholds.LossCrit)
	}
}

// Note: parsePingRTTMoreCases tests removed - now using raw ICMP pinger.

func TestDetermineStatusEdgeCases(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  50 * time.Millisecond,
		Critical: 200 * time.Millisecond,
		LossWarn: 5.0,
		LossCrit: 20.0,
	}
	tester := gateway.NewTester(thresholds)

	// At exactly warning loss threshold.
	stats := &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 5.0, AvgTime: 10}
	status := tester.DetermineStatus(stats)
	if status != gateway.StatusWarning {
		t.Errorf("expected StatusWarning at loss warning threshold, got %v", status)
	}

	// At exactly critical loss threshold.
	stats = &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 20.0, AvgTime: 10}
	status = tester.DetermineStatus(stats)
	if status != gateway.StatusError {
		t.Errorf("expected StatusError at loss critical threshold, got %v", status)
	}

	// At exactly warning latency threshold.
	stats = &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 0, AvgTime: 50}
	status = tester.DetermineStatus(stats)
	if status != gateway.StatusWarning {
		t.Errorf("expected StatusWarning at latency warning threshold, got %v", status)
	}

	// At exactly critical latency threshold.
	stats = &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 0, AvgTime: 200}
	status = tester.DetermineStatus(stats)
	if status != gateway.StatusError {
		t.Errorf("expected StatusError at latency critical threshold, got %v", status)
	}
}

func TestConcurrentTesterAccess(_ *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for range 50 {
				tester.SetGateway("192.168.1." + string(rune('0'+id)))
				_ = tester.GetGateway()
				_ = tester.GetStats()
				_ = tester.IsRunning()
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
}

func TestGetStatsWithNilStats(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())
	// Force nil stats.
	tester.TesterMu().Lock()
	tester.TesterSetStats(nil)
	tester.TesterMu().Unlock()

	stats := tester.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats even when internal stats is nil")
	}
	if stats.Status != gateway.StatusUnknown {
		t.Errorf("expected StatusUnknown, got %v", stats.Status)
	}
}

func TestPingResultWithError(t *testing.T) {
	result := gateway.PingResult{
		Sequence: 3,
		Time:     0,
		TimeMs:   0,
		Success:  false,
		Error:    "request timeout",
	}

	if result.Success {
		t.Error("expected Success false")
	}
	if result.Error != "request timeout" {
		t.Errorf("expected Error 'request timeout', got %q", result.Error)
	}
	if result.Sequence != 3 {
		t.Errorf("expected Sequence 3, got %d", result.Sequence)
	}
	if result.Time != 0 {
		t.Errorf("expected Time 0, got %v", result.Time)
	}
	if result.TimeMs != 0 {
		t.Errorf("expected TimeMs 0, got %v", result.TimeMs)
	}
}

func TestPingStatsZeroValues(t *testing.T) {
	stats := gateway.PingStats{}

	if stats.Gateway != "" {
		t.Error("expected empty Gateway")
	}
	if stats.Sent != 0 {
		t.Error("expected Sent 0")
	}
	if stats.Received != 0 {
		t.Error("expected Received 0")
	}
	if stats.Lost != 0 {
		t.Error("expected Lost 0")
	}
	if stats.LossPercent != 0 {
		t.Error("expected LossPercent 0")
	}
	if stats.Reachable {
		t.Error("expected Reachable false")
	}
	if stats.Status != "" {
		t.Error("expected empty Status")
	}
}

func TestThresholdsZeroValues(t *testing.T) {
	thresholds := gateway.Thresholds{}

	if thresholds.Warning != 0 {
		t.Error("expected Warning 0")
	}
	if thresholds.Critical != 0 {
		t.Error("expected Critical 0")
	}
	if thresholds.LossWarn != 0 {
		t.Error("expected LossWarn 0")
	}
	if thresholds.LossCrit != 0 {
		t.Error("expected LossCrit 0")
	}
}

func TestTesterWithZeroThresholds(t *testing.T) {
	// Test with zero thresholds - any latency/loss should trigger error.
	thresholds := gateway.Thresholds{}
	tester := gateway.NewTester(thresholds)

	// Zero thresholds mean any latency >= 0 should be critical.
	stats := &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 0, AvgTime: 1}
	status := tester.DetermineStatus(stats)
	// With zero thresholds, any latency should be >= Critical (0).
	if status != gateway.StatusError {
		t.Errorf("expected StatusError with zero thresholds, got %v", status)
	}
}

func TestMultipleGatewayChanges(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	gateways := []string{"192.168.1.1", "10.0.0.1", "172.16.0.1", "8.8.8.8", ""}
	for _, gw := range gateways {
		tester.SetGateway(gw)
		if got := tester.GetGateway(); got != gw {
			t.Errorf("expected gateway %q, got %q", gw, got)
		}
	}
}

func TestPingStatsLastUpdated(t *testing.T) {
	now := time.Now()
	stats := gateway.PingStats{
		LastUpdated: now,
	}

	if stats.LastUpdated != now {
		t.Errorf("expected LastUpdated %v, got %v", now, stats.LastUpdated)
	}
}

func TestPingStatsWithResults(t *testing.T) {
	results := []gateway.PingResult{
		{Sequence: 1, TimeMs: 10.5, Success: true},
		{Sequence: 2, TimeMs: 15.2, Success: true},
		{Sequence: 3, TimeMs: 0, Success: false, Error: "timeout"},
	}

	stats := gateway.PingStats{
		Gateway:     "192.168.1.1",
		Sent:        3,
		Received:    2,
		Lost:        1,
		LossPercent: 33.33,
		Results:     results,
	}

	if stats.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", stats.Gateway)
	}
	if stats.Sent != 3 {
		t.Errorf("expected Sent 3, got %d", stats.Sent)
	}
	if stats.Received != 2 {
		t.Errorf("expected Received 2, got %d", stats.Received)
	}
	if stats.Lost != 1 {
		t.Errorf("expected Lost 1, got %d", stats.Lost)
	}
	if stats.LossPercent != 33.33 {
		t.Errorf("expected LossPercent 33.33, got %v", stats.LossPercent)
	}
	if len(stats.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(stats.Results))
	}
	if stats.Results[0].TimeMs != 10.5 {
		t.Errorf("expected first result TimeMs 10.5, got %v", stats.Results[0].TimeMs)
	}
	if stats.Results[2].Success {
		t.Error("expected third result to be unsuccessful")
	}
}

func TestDetermineStatusPriorityOrder(t *testing.T) {
	// Test that loss thresholds are checked before latency.
	thresholds := gateway.Thresholds{
		Warning:  50 * time.Millisecond,
		Critical: 200 * time.Millisecond,
		LossWarn: 5.0,
		LossCrit: 20.0,
	}
	tester := gateway.NewTester(thresholds)

	// High loss + low latency should still be error (loss checked first).
	stats := &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 25.0, AvgTime: 5}
	status := tester.DetermineStatus(stats)
	if status != gateway.StatusError {
		t.Errorf("expected StatusError due to high loss, got %v", status)
	}

	// Warning loss + success latency should be warning.
	stats = &gateway.PingStats{Reachable: true, Received: 1, LossPercent: 10.0, AvgTime: 5}
	status = tester.DetermineStatus(stats)
	if status != gateway.StatusWarning {
		t.Errorf("expected StatusWarning due to loss, got %v", status)
	}
}

func TestStopContinuousMultipleTimes(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Stopping multiple times should not panic.
	tester.StopContinuous()
	tester.StopContinuous()
	tester.StopContinuous()

	if tester.IsRunning() {
		t.Error("expected IsRunning() to be false")
	}
}

func TestStartContinuousWithNilCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	// Start with nil callback - should not panic.
	tester.StartContinuous(100*time.Millisecond, nil)

	if !tester.IsRunning() {
		t.Error("expected IsRunning() to be true")
	}

	time.Sleep(150 * time.Millisecond)
	tester.StopContinuous()
}

func TestTesterCopyStats(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Get initial stats.
	stats1 := tester.GetStats()
	stats1.Gateway = "modified1"
	stats1.Sent = 999

	// Get stats again - should be unmodified.
	stats2 := tester.GetStats()
	if stats2.Gateway == "modified1" {
		t.Error("stats should be a copy, not shared reference")
	}
	if stats2.Sent == 999 {
		t.Error("stats should be a copy, not shared reference")
	}
}

// Note: parsePingRTTEdgeCases tests removed - now using raw ICMP pinger.

func TestTesterPingToInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	// Use an invalid/unreachable IP that will fail quickly.
	tester.SetGateway("192.0.2.1") // TEST-NET-1, not routable

	result := tester.Ping()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// This should fail (or timeout quickly).
	// We just verify the error path doesn't panic.
	if result.Success {
		t.Log("ping unexpectedly succeeded - skip this check")
	} else if result.Error == "" {
		t.Error("expected non-empty error for failed ping")
	}
}

func TestTesterTestWithFailedPings(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	thresholds := gateway.DefaultThresholds()
	tester := gateway.NewTester(thresholds)
	// Use an unreachable IP.
	tester.SetGateway("192.0.2.1")
	// Reduce ping count for faster test.
	tester.TesterMu().Lock()
	tester.TesterSetPingCount(2)
	tester.TesterSetPingTimeout(1 * time.Second)
	tester.TesterMu().Unlock()

	stats := tester.Test()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	// Should have attempted pings.
	if stats.Sent == 0 {
		t.Error("expected Sent > 0")
	}
	// Status should reflect the failure.
	if stats.Received == stats.Sent {
		t.Log("all pings succeeded unexpectedly")
	}
}

func TestPingStatsTiming(t *testing.T) {
	stats := gateway.PingStats{
		MinTime:  5.5,
		MaxTime:  50.3,
		AvgTime:  25.2,
		LastTime: 22.1,
	}

	if stats.MinTime != 5.5 {
		t.Errorf("expected MinTime 5.5, got %v", stats.MinTime)
	}
	if stats.MaxTime != 50.3 {
		t.Errorf("expected MaxTime 50.3, got %v", stats.MaxTime)
	}
	if stats.AvgTime != 25.2 {
		t.Errorf("expected AvgTime 25.2, got %v", stats.AvgTime)
	}
	if stats.LastTime != 22.1 {
		t.Errorf("expected LastTime 22.1, got %v", stats.LastTime)
	}
}

func TestPingResultTimeDuration(t *testing.T) {
	result := gateway.PingResult{
		Time:   125 * time.Millisecond,
		TimeMs: 125.0,
	}

	if result.Time != 125*time.Millisecond {
		t.Errorf("expected Time 125ms, got %v", result.Time)
	}
	if result.TimeMs != 125.0 {
		t.Errorf("expected TimeMs 125.0, got %v", result.TimeMs)
	}
}

func TestStartContinuousAlreadyRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	called := 0
	callback := func(_ *gateway.PingStats) {
		called++
	}

	// Start once.
	tester.StartContinuous(50*time.Millisecond, callback)
	if !tester.IsRunning() {
		t.Error("expected IsRunning true after first start")
	}

	// Start again - should be no-op (already running).
	tester.StartContinuous(50*time.Millisecond, callback)
	if !tester.IsRunning() {
		t.Error("expected IsRunning still true after second start")
	}

	// Stop.
	tester.StopContinuous()
	if tester.IsRunning() {
		t.Error("expected IsRunning false after stop")
	}
}

func TestTesterPingTimeout(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Verify initial timeout.
	tester.TesterMu().RLock()
	timeout := tester.TesterPingTimeout()
	tester.TesterMu().RUnlock()

	if timeout != 2*time.Second {
		t.Errorf("expected default pingTimeout 2s, got %v", timeout)
	}
}

func TestTesterPingCount(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Verify initial ping count.
	tester.TesterMu().RLock()
	count := tester.TesterPingCount()
	tester.TesterMu().RUnlock()

	if count != 3 {
		t.Errorf("expected default pingCount 3, got %d", count)
	}
}
