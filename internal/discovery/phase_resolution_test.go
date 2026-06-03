package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestDefaultResolutionConfig(t *testing.T) {
	cfg := discovery.DefaultResolutionConfig()

	if cfg == nil {
		t.Fatal("DefaultResolutionConfig returned nil")
	}
	if !cfg.DNS {
		t.Error("Expected DNS=true by default")
	}
	if !cfg.NetBIOS {
		t.Error("Expected NetBIOS=true by default")
	}
	if !cfg.MDNS {
		t.Error("Expected MDNS=true by default")
	}
	if cfg.Timing.DNSTimeout <= 0 {
		t.Error("Expected positive DNSTimeout")
	}
	if cfg.Timing.NetBIOSTimeout <= 0 {
		t.Error("Expected positive NetBIOSTimeout")
	}
	if cfg.Timing.MDNSTimeout <= 0 {
		t.Error("Expected positive MDNSTimeout")
	}
	if cfg.Timing.PhaseTimeout <= 0 {
		t.Error("Expected positive PhaseTimeout")
	}
	if cfg.Timing.MaxConcurrentDNS <= 0 {
		t.Error("Expected positive MaxConcurrentDNS")
	}
	if cfg.Timing.MaxConcurrentNetBIOS <= 0 {
		t.Error("Expected positive MaxConcurrentNetBIOS")
	}
	if cfg.Timing.MaxConcurrentMDNS <= 0 {
		t.Error("Expected positive MaxConcurrentMDNS")
	}
}

func TestResolutionConfig_Fields(t *testing.T) {
	cfg := &discovery.ResolutionConfig{
		DNS:     false,
		NetBIOS: true,
		MDNS:    false,
		Timing: discovery.ResolutionTiming{
			DNSTimeout:           1 * time.Second,
			NetBIOSTimeout:       2 * time.Second,
			MDNSTimeout:          3 * time.Second,
			PhaseTimeout:         10 * time.Minute,
			MaxConcurrentDNS:     100,
			MaxConcurrentNetBIOS: 50,
			MaxConcurrentMDNS:    25,
		},
	}

	if cfg.DNS {
		t.Error("Expected DNS=false")
	}
	if !cfg.NetBIOS {
		t.Error("Expected NetBIOS=true")
	}
	if cfg.MDNS {
		t.Error("Expected MDNS=false")
	}
	if cfg.Timing.DNSTimeout != 1*time.Second {
		t.Errorf("Expected DNSTimeout=1s, got %v", cfg.Timing.DNSTimeout)
	}
	if cfg.Timing.NetBIOSTimeout != 2*time.Second {
		t.Errorf("Expected NetBIOSTimeout=2s, got %v", cfg.Timing.NetBIOSTimeout)
	}
	if cfg.Timing.MDNSTimeout != 3*time.Second {
		t.Errorf("Expected MDNSTimeout=3s, got %v", cfg.Timing.MDNSTimeout)
	}
	if cfg.Timing.PhaseTimeout != 10*time.Minute {
		t.Errorf("Expected PhaseTimeout=10m, got %v", cfg.Timing.PhaseTimeout)
	}
	if cfg.Timing.MaxConcurrentDNS != 100 {
		t.Errorf("Expected MaxConcurrentDNS=100, got %d", cfg.Timing.MaxConcurrentDNS)
	}
	if cfg.Timing.MaxConcurrentNetBIOS != 50 {
		t.Errorf("Expected MaxConcurrentNetBIOS=50, got %d", cfg.Timing.MaxConcurrentNetBIOS)
	}
	if cfg.Timing.MaxConcurrentMDNS != 25 {
		t.Errorf("Expected MaxConcurrentMDNS=25, got %d", cfg.Timing.MaxConcurrentMDNS)
	}
}

func TestNewResolutionPhase(t *testing.T) {
	cfg := discovery.DefaultResolutionConfig()

	// Test with default config
	phase := discovery.NewResolutionPhase("lo", cfg, nil)
	if phase == nil {
		t.Fatal("NewResolutionPhase returned nil")
	}
	if phase.Name() != "resolution" {
		t.Errorf("Expected Name='resolution', got %s", phase.Name())
	}

	// Test with nil config (should use defaults)
	phaseDefaultCfg := discovery.NewResolutionPhase("lo", nil, nil)
	if phaseDefaultCfg == nil {
		t.Fatal("NewResolutionPhase with nil config returned nil")
	}
}

func TestResolutionPhase_Name(t *testing.T) {
	phase := discovery.NewResolutionPhase("lo", nil, nil)

	if phase.Name() != "resolution" {
		t.Errorf("Expected Name='resolution', got %s", phase.Name())
	}
}

func TestResolutionPhase_RunEmptyDevices(t *testing.T) {
	phase := discovery.NewResolutionPhase("lo", nil, nil)
	ctx := context.Background()

	// Run with empty devices list
	devices, err := phase.Run(ctx, nil, nil)
	if err != nil {
		t.Errorf("Run returned error for empty devices: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("Expected empty devices, got %d", len(devices))
	}

	// Run with empty slice
	devices, err = phase.Run(ctx, []*discovery.DiscoveredDevice{}, nil)
	if err != nil {
		t.Errorf("Run returned error for empty slice: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("Expected empty devices, got %d", len(devices))
	}
}

func TestResolutionProgress(t *testing.T) {
	var progress discovery.ResolutionProgress

	// Test Start
	progress.Start(10)

	// Test initial state
	if progress.Resolved() != 0 {
		t.Errorf("Expected Resolved=0 initially, got %d", progress.Resolved())
	}
	if progress.CurrentTarget() != "" {
		t.Errorf("Expected empty CurrentTarget initially, got %s", progress.CurrentTarget())
	}

	// Test PercentComplete with no devices resolved
	if progress.PercentComplete() != 0 {
		t.Errorf("Expected PercentComplete=0 initially, got %f", progress.PercentComplete())
	}
}

func TestResolutionProgress_MarkResolved(t *testing.T) {
	var progress discovery.ResolutionProgress
	progress.Start(5)

	// First resolution should return true
	if !progress.MarkResolved("192.168.1.1") {
		t.Error("Expected MarkResolved to return true for first resolution")
	}
	if progress.Resolved() != 1 {
		t.Errorf("Expected Resolved=1, got %d", progress.Resolved())
	}

	// Second resolution of same IP should return false (idempotent)
	if progress.MarkResolved("192.168.1.1") {
		t.Error("Expected MarkResolved to return false for duplicate resolution")
	}
	if progress.Resolved() != 1 {
		t.Errorf("Expected Resolved=1 (no duplicate count), got %d", progress.Resolved())
	}

	// Different IP should return true
	if !progress.MarkResolved("192.168.1.2") {
		t.Error("Expected MarkResolved to return true for different IP")
	}
	if progress.Resolved() != 2 {
		t.Errorf("Expected Resolved=2, got %d", progress.Resolved())
	}
}

func TestResolutionProgress_SetCurrentTarget(t *testing.T) {
	var progress discovery.ResolutionProgress
	progress.Start(5)

	progress.SetCurrentTarget("192.168.1.1")
	if progress.CurrentTarget() != "192.168.1.1" {
		t.Errorf("Expected CurrentTarget='192.168.1.1', got %s", progress.CurrentTarget())
	}

	progress.SetCurrentTarget("10.0.0.1")
	if progress.CurrentTarget() != "10.0.0.1" {
		t.Errorf("Expected CurrentTarget='10.0.0.1', got %s", progress.CurrentTarget())
	}
}

func TestResolutionProgress_PercentComplete(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		resolved     int
		expectedPct  float64
		tolerancePct float64
	}{
		{"no_devices", 0, 0, 100.0, 0.1}, // 0 devices = 100% complete
		{"none_resolved", 10, 0, 0.0, 0.1},
		{"half_resolved", 10, 5, 50.0, 0.1},
		{"all_resolved", 10, 10, 100.0, 0.1},
		{"over_resolved", 10, 15, 100.0, 0.1}, // Capped at 100%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var progress discovery.ResolutionProgress
			progress.Start(tt.total)

			for i := range tt.resolved {
				progress.MarkResolved("192.168.1." + string(rune('0'+i)))
			}

			pct := progress.PercentComplete()
			diff := pct - tt.expectedPct
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerancePct {
				t.Errorf("Expected PercentComplete~=%.1f%%, got %.1f%%", tt.expectedPct, pct)
			}
		})
	}
}

func TestResolutionProgress_Concurrency(t *testing.T) {
	var progress discovery.ResolutionProgress
	progress.Start(1000)

	// Test concurrent MarkResolved
	done := make(chan struct{})
	for i := range 10 {
		go func(base int) {
			for j := range 100 {
				ip := "192.168." + string(rune('0'+base)) + "." + string(rune('0'+j))
				progress.MarkResolved(ip)
			}
			done <- struct{}{}
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Should have 1000 unique IPs resolved
	if progress.Resolved() != 1000 {
		t.Errorf("Expected Resolved=1000, got %d", progress.Resolved())
	}
}

func TestNewDNSResolver(t *testing.T) {
	// Test with defaults
	resolver := discovery.NewDNSResolver(0, 0)
	if resolver == nil {
		t.Fatal("NewDNSResolver returned nil")
	}

	// Test with custom values
	resolver = discovery.NewDNSResolver(2*time.Second, 100)
	if resolver == nil {
		t.Fatal("NewDNSResolver returned nil with custom values")
	}
}

func TestDNSResolver_ResolveReverse(t *testing.T) {
	resolver := discovery.NewDNSResolver(100*time.Millisecond, 10)
	ctx := context.Background()

	// Test with localhost - should resolve
	hostname, err := resolver.ResolveReverse(ctx, "127.0.0.1")
	// May or may not resolve depending on system config
	t.Logf("ResolveReverse(127.0.0.1): hostname=%q, err=%v", hostname, err)
}

func TestDNSResolver_ResolveForward(t *testing.T) {
	resolver := discovery.NewDNSResolver(100*time.Millisecond, 10)
	ctx := context.Background()

	// Test with localhost
	ips, err := resolver.ResolveForward(ctx, "localhost")
	// May or may not resolve depending on system config
	t.Logf("ResolveForward(localhost): ips=%v, err=%v", ips, err)
}

func TestDNSResolver_ResolveBatch(t *testing.T) {
	resolver := discovery.NewDNSResolver(100*time.Millisecond, 10)
	ctx := context.Background()

	// Test with multiple IPs
	ips := []string{"127.0.0.1", "192.0.2.1"} // TEST-NET-1 won't resolve
	results := resolver.ResolveBatch(ctx, ips)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for i, result := range results {
		if result.IP != ips[i] {
			t.Errorf("Expected result[%d].IP=%s, got %s", i, ips[i], result.IP)
		}
	}
}

func TestDNSResolver_ResolveBatch_ContextCancellation(t *testing.T) {
	resolver := discovery.NewDNSResolver(5*time.Second, 10)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	ips := []string{"127.0.0.1", "192.0.2.1"}
	results := resolver.ResolveBatch(ctx, ips)

	// Should still return results (with errors)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestDNSResult_Fields(t *testing.T) {
	result := discovery.DNSResult{
		IP:       "192.168.1.1",
		Hostname: "device.local",
		Err:      nil,
	}

	if result.IP != "192.168.1.1" {
		t.Errorf("Expected IP='192.168.1.1', got %s", result.IP)
	}
	if result.Hostname != "device.local" {
		t.Errorf("Expected Hostname='device.local', got %s", result.Hostname)
	}
	if result.Err != nil {
		t.Errorf("Expected Err=nil, got %v", result.Err)
	}
}

func TestResolutionTiming_Fields(t *testing.T) {
	timing := discovery.ResolutionTiming{
		DNSTimeout:           500 * time.Millisecond,
		NetBIOSTimeout:       1 * time.Second,
		MDNSTimeout:          2 * time.Second,
		PhaseTimeout:         5 * time.Minute,
		MaxConcurrentDNS:     200,
		MaxConcurrentNetBIOS: 100,
		MaxConcurrentMDNS:    50,
	}

	if timing.DNSTimeout != 500*time.Millisecond {
		t.Errorf("Expected DNSTimeout=500ms, got %v", timing.DNSTimeout)
	}
	if timing.NetBIOSTimeout != 1*time.Second {
		t.Errorf("Expected NetBIOSTimeout=1s, got %v", timing.NetBIOSTimeout)
	}
	if timing.MDNSTimeout != 2*time.Second {
		t.Errorf("Expected MDNSTimeout=2s, got %v", timing.MDNSTimeout)
	}
	if timing.PhaseTimeout != 5*time.Minute {
		t.Errorf("Expected PhaseTimeout=5m, got %v", timing.PhaseTimeout)
	}
	if timing.MaxConcurrentDNS != 200 {
		t.Errorf("Expected MaxConcurrentDNS=200, got %d", timing.MaxConcurrentDNS)
	}
	if timing.MaxConcurrentNetBIOS != 100 {
		t.Errorf("Expected MaxConcurrentNetBIOS=100, got %d", timing.MaxConcurrentNetBIOS)
	}
	if timing.MaxConcurrentMDNS != 50 {
		t.Errorf("Expected MaxConcurrentMDNS=50, got %d", timing.MaxConcurrentMDNS)
	}
}

func TestResolutionPhase_RunWithDevices(t *testing.T) {
	cfg := discovery.DefaultResolutionConfig()
	cfg.Timing.DNSTimeout = 50 * time.Millisecond
	cfg.Timing.NetBIOSTimeout = 50 * time.Millisecond
	cfg.Timing.MDNSTimeout = 50 * time.Millisecond
	cfg.Timing.PhaseTimeout = 500 * time.Millisecond

	phase := discovery.NewResolutionPhase("lo", cfg, nil)
	ctx := context.Background()

	// Create test devices
	devices := []*discovery.DiscoveredDevice{
		{IP: "127.0.0.1", MAC: "00:11:22:33:44:55"},
		{IP: "192.0.2.1", MAC: "AA:BB:CC:DD:EE:FF"}, // TEST-NET-1
	}

	result, err := phase.Run(ctx, devices, nil)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 devices returned, got %d", len(result))
	}

	// Verify devices have DisplayName computed
	for _, d := range result {
		if d.DisplayName == "" {
			t.Errorf("Expected DisplayName to be set for device %s", d.IP)
		}
	}
}
