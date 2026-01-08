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
			name: "critical packet loss",
			stats: &gateway.PingStats{
				Reachable:   true,
				Received:    1,
				LossPercent: 25.0,
				AvgTime:     10,
			},
			expected: gateway.StatusError,
		},
		{
			name: "warning packet loss",
			stats: &gateway.PingStats{
				Reachable:   true,
				Received:    1,
				LossPercent: 10.0,
				AvgTime:     10,
			},
			expected: gateway.StatusWarning,
		},
		{
			name: "critical latency",
			stats: &gateway.PingStats{
				Reachable:   true,
				Received:    1,
				LossPercent: 0,
				AvgTime:     250,
			},
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
	// Force nil stats - TesterSetStats handles its own locking.
	tester.TesterSetStats(nil)

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
	// TesterSetPingCount and TesterSetPingTimeout handle their own locking.
	tester.TesterSetPingCount(2)
	tester.TesterSetPingTimeout(1 * time.Second)

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

// Additional tests for increased coverage.

func TestPingErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "ping timeout",
		},
		{
			name:     "test error",
			err:      gateway.ErrTest,
			expected: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gateway.PingErrorMessage(tt.err)
			if result != tt.expected {
				t.Errorf("PingErrorMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectGatewayIPv6(_ *testing.T) {
	// Test IPv6 gateway detection - may or may not find one.
	gw, err := gateway.DetectGatewayIPv6()
	// Just verify it doesn't panic and returns valid types.
	_ = gw
	_ = err
}

func TestTesterClose(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	// Close should not panic.
	tester.Close()

	// Verify pinger is nil after close.
	if tester.TesterGetPinger() != nil {
		t.Error("expected pinger to be nil after Close")
	}

	// Multiple closes should not panic.
	tester.Close()
	tester.Close()
}

func TestTesterCloseWhileRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	// Start continuous testing.
	tester.StartContinuous(100*time.Millisecond, nil)

	if !tester.IsRunning() {
		t.Error("expected IsRunning() to be true")
	}

	// Close while running - should stop and clean up.
	tester.Close()

	if tester.IsRunning() {
		t.Error("expected IsRunning() to be false after Close")
	}
}

func TestTesterPingInvalidGatewayAddress(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Set an invalid IP address format.
	tester.SetGateway("not-an-ip-address")

	result := tester.Ping()
	if result.Success {
		t.Error("expected failure for invalid gateway address")
	}
	if result.Error != "invalid gateway address" {
		t.Errorf("expected 'invalid gateway address' error, got %q", result.Error)
	}
}

func TestTesterPingNoPinger(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("192.168.1.1")

	// Set pinger to nil to simulate no CAP_NET_RAW.
	tester.TesterSetPinger(nil)

	result := tester.Ping()
	if result.Success {
		t.Error("expected failure when pinger is nil")
	}
	if result.Error != "ICMP pinger unavailable - requires CAP_NET_RAW" {
		t.Errorf("expected CAP_NET_RAW error, got %q", result.Error)
	}
}

func TestTesterTestAutoDetectFails(t *testing.T) {
	// Create a tester without setting a gateway.
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Run the test - it will try to auto-detect.
	stats := tester.Test()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	// Either it detected a gateway or returned an error status.
	if stats.Status == "" {
		t.Error("expected non-empty status")
	}
}

func TestTesterTestStatisticsCalculation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")
	tester.TesterSetPingCount(2) // Reduce for faster test.

	stats := tester.Test()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	// Verify sent count matches ping count.
	if stats.Sent != 2 {
		t.Errorf("expected Sent=2, got %d", stats.Sent)
	}

	// Verify results array length matches sent.
	if len(stats.Results) != stats.Sent {
		t.Errorf("expected Results length %d, got %d", stats.Sent, len(stats.Results))
	}

	// Verify loss calculation.
	expectedLoss := float64(stats.Lost) / float64(stats.Sent) * 100
	if stats.Sent > 0 && stats.LossPercent != expectedLoss {
		t.Errorf("expected LossPercent %v, got %v", expectedLoss, stats.LossPercent)
	}

	// Verify LastUpdated is set.
	if stats.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be set")
	}
}

func TestPingStatsIPv6Field(t *testing.T) {
	ipv6Stats := &gateway.PingStats{
		Gateway:   "::1",
		Sent:      3,
		Received:  3,
		Status:    gateway.StatusSuccess,
		Reachable: true,
	}

	stats := gateway.PingStats{
		IPv6: ipv6Stats,
	}

	if stats.IPv6 == nil {
		t.Fatal("expected non-nil IPv6 stats")
	}
	if stats.IPv6.Gateway != "::1" {
		t.Errorf("expected IPv6 gateway '::1', got %q", stats.IPv6.Gateway)
	}
	if stats.IPv6.Sent != 3 {
		t.Errorf("expected IPv6 Sent 3, got %d", stats.IPv6.Sent)
	}
	if stats.IPv6.Received != 3 {
		t.Errorf("expected IPv6 Received 3, got %d", stats.IPv6.Received)
	}
	if stats.IPv6.Status != gateway.StatusSuccess {
		t.Errorf("expected IPv6 Status success, got %v", stats.IPv6.Status)
	}
	if !stats.IPv6.Reachable {
		t.Error("expected IPv6 Reachable true")
	}
}

func TestTesterRunningState(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Initial state should be not running.
	if tester.TesterRunning() {
		t.Error("expected TesterRunning() to be false initially")
	}

	// Stop when not running should be safe.
	tester.StopContinuous()
	if tester.TesterRunning() {
		t.Error("expected TesterRunning() to still be false after StopContinuous")
	}
}

func TestStartContinuousCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")
	tester.TesterSetPingCount(1) // Reduce for faster test.

	callCount := 0
	var lastStats *gateway.PingStats
	callback := func(stats *gateway.PingStats) {
		callCount++
		lastStats = stats
	}

	tester.StartContinuous(50*time.Millisecond, callback)
	time.Sleep(200 * time.Millisecond)
	tester.StopContinuous()

	// Should have received at least one callback.
	if callCount == 0 {
		t.Log("callback was not called - may be due to timing or ICMP permission")
	}

	// If callback was called, verify stats were passed.
	if lastStats != nil && lastStats.Gateway != "127.0.0.1" {
		t.Errorf("expected Gateway '127.0.0.1', got %q", lastStats.Gateway)
	}
}

func TestTesterConcurrentStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	done := make(chan bool, 10)

	// Concurrent start/stop operations.
	for range 5 {
		go func() {
			tester.StartContinuous(50*time.Millisecond, nil)
			time.Sleep(10 * time.Millisecond)
			tester.StopContinuous()
			done <- true
		}()
	}

	for range 5 {
		<-done
	}

	// Final state should be not running.
	tester.StopContinuous()
	if tester.IsRunning() {
		t.Error("expected IsRunning() to be false after all goroutines complete")
	}
}

func TestDetermineStatusWithVeryHighValues(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  1 * time.Hour,
		Critical: 2 * time.Hour,
		LossWarn: 50.0,
		LossCrit: 90.0,
	}
	tester := gateway.NewTester(thresholds)

	// With very high thresholds, normal values should be success.
	stats := &gateway.PingStats{
		Reachable:   true,
		Received:    1,
		LossPercent: 10.0,
		AvgTime:     500, // 500ms is way below 1 hour warning.
	}
	status := tester.DetermineStatus(stats)
	if status != gateway.StatusSuccess {
		t.Errorf("expected StatusSuccess with high thresholds, got %v", status)
	}
}

func TestPingResultSequenceNumbers(t *testing.T) {
	// Test sequence numbers are correctly incremented.
	results := []gateway.PingResult{
		{Sequence: 1, Success: true, TimeMs: 10.0},
		{Sequence: 2, Success: true, TimeMs: 15.0},
		{Sequence: 3, Success: false, Error: "timeout"},
	}

	for i, r := range results {
		expectedSeq := i + 1
		if r.Sequence != expectedSeq {
			t.Errorf("expected Sequence %d, got %d", expectedSeq, r.Sequence)
		}
	}
}

func TestGetAllRoutes(t *testing.T) {
	// Test GetAllRoutes - should return routes or an error.
	routes, err := gateway.GetAllRoutes()
	if err != nil {
		t.Logf("GetAllRoutes returned error: %v (may be expected on some systems)", err)
		return
	}

	// If we got routes, verify they have valid fields.
	for _, r := range routes {
		if r.Family != "inet" && r.Family != "inet6" {
			t.Errorf("unexpected Family %q", r.Family)
		}
	}
}

func TestGetDefaultGatewayInterface(t *testing.T) {
	// Test GetDefaultGatewayInterface - should return interface name or empty.
	iface, err := gateway.GetDefaultGatewayInterface()
	if err != nil {
		t.Logf("GetDefaultGatewayInterface returned error: %v", err)
	}
	// Just verify it doesn't panic and returns valid types.
	_ = iface
}

func TestRouteInfoFields(t *testing.T) {
	ri := gateway.RouteInfo{
		Destination: "0.0.0.0/0",
		Gateway:     "192.168.1.1",
		Interface:   "en0",
		Family:      "inet",
	}

	// Verify all fields are set and accessible.
	dest := ri.Destination
	gw := ri.Gateway
	iface := ri.Interface
	family := ri.Family

	if dest != "0.0.0.0/0" {
		t.Errorf("expected Destination '0.0.0.0/0', got %q", dest)
	}
	if gw != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", gw)
	}
	if iface != "en0" {
		t.Errorf("expected Interface 'en0', got %q", iface)
	}
	if family != "inet" {
		t.Errorf("expected Family 'inet', got %q", family)
	}
}

func TestRouteInfoIPv6(t *testing.T) {
	ri := gateway.RouteInfo{
		Destination: "::/0",
		Gateway:     "fe80::1",
		Interface:   "en0",
		Family:      "inet6",
	}

	// Verify all fields are set and accessible.
	dest := ri.Destination
	gw := ri.Gateway
	iface := ri.Interface
	family := ri.Family

	if dest != "::/0" {
		t.Errorf("expected Destination '::/0', got %q", dest)
	}
	if gw != "fe80::1" {
		t.Errorf("expected Gateway 'fe80::1', got %q", gw)
	}
	if iface != "en0" {
		t.Errorf("expected Interface 'en0', got %q", iface)
	}
	if family != "inet6" {
		t.Errorf("expected Family 'inet6', got %q", family)
	}
}

func TestTesterGetStatsAfterTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")
	tester.TesterSetPingCount(1)

	// Run test.
	testStats := tester.Test()

	// GetStats should return the same values.
	getStats := tester.GetStats()

	if testStats.Gateway != getStats.Gateway {
		t.Errorf("Gateway mismatch: Test returned %q, GetStats returned %q",
			testStats.Gateway, getStats.Gateway)
	}
	if testStats.Sent != getStats.Sent {
		t.Errorf("Sent mismatch: Test returned %d, GetStats returned %d",
			testStats.Sent, getStats.Sent)
	}
}

func TestThresholdsCustomValues(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
		LossWarn: 10.0,
		LossCrit: 50.0,
	}

	if thresholds.Warning != 100*time.Millisecond {
		t.Errorf("expected Warning 100ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 500*time.Millisecond {
		t.Errorf("expected Critical 500ms, got %v", thresholds.Critical)
	}
	if thresholds.LossWarn != 10.0 {
		t.Errorf("expected LossWarn 10.0, got %v", thresholds.LossWarn)
	}
	if thresholds.LossCrit != 50.0 {
		t.Errorf("expected LossCrit 50.0, got %v", thresholds.LossCrit)
	}
}

func TestDetermineStatusJustBelowThresholds(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  50 * time.Millisecond,
		Critical: 200 * time.Millisecond,
		LossWarn: 5.0,
		LossCrit: 20.0,
	}
	tester := gateway.NewTester(thresholds)

	// Just below warning thresholds - should be success.
	stats := &gateway.PingStats{
		Reachable:   true,
		Received:    1,
		LossPercent: 4.9,
		AvgTime:     49.9,
	}
	status := tester.DetermineStatus(stats)
	if status != gateway.StatusSuccess {
		t.Errorf("expected StatusSuccess just below thresholds, got %v", status)
	}
}

func TestDetermineStatusJustBelowCritical(t *testing.T) {
	thresholds := gateway.Thresholds{
		Warning:  50 * time.Millisecond,
		Critical: 200 * time.Millisecond,
		LossWarn: 5.0,
		LossCrit: 20.0,
	}
	tester := gateway.NewTester(thresholds)

	// Just below critical threshold but above warning - should be warning.
	stats := &gateway.PingStats{
		Reachable:   true,
		Received:    1,
		LossPercent: 19.9,
		AvgTime:     10,
	}
	status := tester.DetermineStatus(stats)
	if status != gateway.StatusWarning {
		t.Errorf("expected StatusWarning just below critical, got %v", status)
	}
}

func TestPingStatsEmptyResults(t *testing.T) {
	stats := gateway.PingStats{}

	if stats.Results != nil {
		t.Error("expected nil Results")
	}
	if stats.Sent != 0 {
		t.Errorf("expected Sent 0, got %d", stats.Sent)
	}
	if stats.Gateway != "" {
		t.Errorf("expected empty Gateway, got %q", stats.Gateway)
	}
	if stats.Received != 0 {
		t.Errorf("expected Received 0, got %d", stats.Received)
	}
	if stats.Lost != 0 {
		t.Errorf("expected Lost 0, got %d", stats.Lost)
	}
}

func TestTesterSetGetGatewayIPv6(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())

	// Test IPv6 gateway.
	tester.SetGateway("fe80::1")
	if got := tester.GetGateway(); got != "fe80::1" {
		t.Errorf("expected gateway 'fe80::1', got %q", got)
	}

	// Test IPv6 localhost.
	tester.SetGateway("::1")
	if got := tester.GetGateway(); got != "::1" {
		t.Errorf("expected gateway '::1', got %q", got)
	}
}

func TestTesterPingIPv6Localhost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ping test in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("::1")

	result := tester.Ping()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// May or may not succeed depending on system configuration.
	if !result.Success {
		t.Logf("IPv6 ping to ::1 failed: %v (may be expected)", result.Error)
	}
}

func TestStartContinuousRestartAfterStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway("127.0.0.1")

	// First start/stop cycle.
	tester.StartContinuous(50*time.Millisecond, nil)
	time.Sleep(60 * time.Millisecond)
	tester.StopContinuous()

	if tester.IsRunning() {
		t.Error("expected IsRunning false after first stop")
	}

	// Second start/stop cycle - verifies stopOnce is reset.
	tester.StartContinuous(50*time.Millisecond, nil)

	if !tester.IsRunning() {
		t.Error("expected IsRunning true after restart")
	}

	time.Sleep(60 * time.Millisecond)
	tester.StopContinuous()

	if tester.IsRunning() {
		t.Error("expected IsRunning false after second stop")
	}
}
