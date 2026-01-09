package discovery_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestDefaultEnumerationConfig(t *testing.T) {
	cfg := discovery.DefaultEnumerationConfig()

	if cfg == nil {
		t.Fatal("DefaultEnumerationConfig returned nil")
	}
	if !cfg.ARPScan {
		t.Error("Expected ARPScan=true by default")
	}
	if !cfg.ICMPScan {
		t.Error("Expected ICMPScan=true by default")
	}
	if !cfg.NDPScan {
		t.Error("Expected NDPScan=true by default")
	}
	if !cfg.PassiveProtocols.LLDP {
		t.Error("Expected PassiveProtocols.LLDP=true by default")
	}
	if !cfg.PassiveProtocols.CDP {
		t.Error("Expected PassiveProtocols.CDP=true by default")
	}
	if !cfg.PassiveProtocols.EDP {
		t.Error("Expected PassiveProtocols.EDP=true by default")
	}
	if cfg.Timing.PingTimeout <= 0 {
		t.Error("Expected positive PingTimeout")
	}
	if cfg.Timing.ScanTimeout <= 0 {
		t.Error("Expected positive ScanTimeout")
	}
	if cfg.Timing.PingWorkers <= 0 {
		t.Error("Expected positive PingWorkers")
	}
}

func TestEnumerationConfig_Fields(t *testing.T) {
	cfg := &discovery.EnumerationConfig{
		ARPScan:  false,
		ICMPScan: true,
		NDPScan:  false,
		Timing: discovery.EnumerationTiming{
			PingTimeout: 2 * time.Second,
			ScanTimeout: 10 * time.Minute,
			PingWorkers: 25,
			ARPDelay:    10 * time.Millisecond,
		},
		AdditionalSubnets: []string{"10.0.0.0/8", "172.16.0.0/12"},
	}

	if cfg.ARPScan {
		t.Error("Expected ARPScan=false")
	}
	if !cfg.ICMPScan {
		t.Error("Expected ICMPScan=true")
	}
	if cfg.NDPScan {
		t.Error("Expected NDPScan=false")
	}
	if cfg.Timing.PingTimeout != 2*time.Second {
		t.Errorf("Expected PingTimeout=2s, got %v", cfg.Timing.PingTimeout)
	}
	if cfg.Timing.ScanTimeout != 10*time.Minute {
		t.Errorf("Expected ScanTimeout=10m, got %v", cfg.Timing.ScanTimeout)
	}
	if cfg.Timing.PingWorkers != 25 {
		t.Errorf("Expected PingWorkers=25, got %d", cfg.Timing.PingWorkers)
	}
	if cfg.Timing.ARPDelay != 10*time.Millisecond {
		t.Errorf("Expected ARPDelay=10ms, got %v", cfg.Timing.ARPDelay)
	}
	if len(cfg.AdditionalSubnets) != 2 {
		t.Errorf("Expected 2 AdditionalSubnets, got %d", len(cfg.AdditionalSubnets))
	}
}

func TestNewEnumerationPhase(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")
	cfg := discovery.DefaultEnumerationConfig()

	// Test with default config
	phase := discovery.NewEnumerationPhase(dd, cfg, nil)
	if phase == nil {
		t.Fatal("NewEnumerationPhase returned nil")
	}
	if phase.Name() != "enumeration" {
		t.Errorf("Expected Name='enumeration', got %s", phase.Name())
	}

	// Test with nil config (should use defaults)
	phaseDefaultCfg := discovery.NewEnumerationPhase(dd, nil, nil)
	if phaseDefaultCfg == nil {
		t.Fatal("NewEnumerationPhase with nil config returned nil")
	}
}

func TestEnumerationPhase_Name(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")
	phase := discovery.NewEnumerationPhase(dd, nil, nil)

	if phase.Name() != "enumeration" {
		t.Errorf("Expected Name='enumeration', got %s", phase.Name())
	}
}

func TestEnumerationProgress(t *testing.T) {
	var progress discovery.EnumerationProgress

	// Test Start
	progress.Start()

	// Test initial state
	if progress.DevicesFound() != 0 {
		t.Errorf("Expected DevicesFound=0 initially, got %d", progress.DevicesFound())
	}
	if progress.CurrentTarget() != "" {
		t.Errorf("Expected empty CurrentTarget initially, got %s", progress.CurrentTarget())
	}

	// Test IncrementDevices
	progress.IncrementDevices()
	progress.IncrementDevices()
	progress.IncrementDevices()

	if progress.DevicesFound() != 3 {
		t.Errorf("Expected DevicesFound=3, got %d", progress.DevicesFound())
	}

	// Test SetPhase/SetCurrentTarget
	progress.SetPhase("arp_scan")
	progress.SetCurrentTarget("192.168.1.1")

	if progress.CurrentTarget() != "192.168.1.1" {
		t.Errorf("Expected CurrentTarget='192.168.1.1', got %s", progress.CurrentTarget())
	}

	// Test EstimatedTotal (should return constant for /24)
	total := progress.EstimatedTotal()
	if total <= 0 {
		t.Errorf("Expected positive EstimatedTotal, got %d", total)
	}

	// Test PercentComplete
	pct := progress.PercentComplete()
	if pct < 0 || pct > 100 {
		t.Errorf("Expected PercentComplete between 0-100, got %f", pct)
	}
}

func TestEnumerationProgress_Errors(t *testing.T) {
	var progress discovery.EnumerationProgress
	progress.Start()

	// Test AddError
	progress.AddError("arp_scan", context.DeadlineExceeded)
	progress.AddError("icmp_scan", context.Canceled)

	errors := progress.Errors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	if errors[0].Phase != "arp_scan" {
		t.Errorf("Expected first error phase='arp_scan', got %s", errors[0].Phase)
	}
	if errors[1].Phase != "icmp_scan" {
		t.Errorf("Expected second error phase='icmp_scan', got %s", errors[1].Phase)
	}
}

func TestEnumerationProgress_MarkComplete(t *testing.T) {
	var progress discovery.EnumerationProgress
	progress.Start()

	progress.MarkComplete("arp_scan")
	progress.MarkComplete("icmp_scan")
	progress.MarkComplete("ndp_scan")

	// Errors list should return a copy
	errors := progress.Errors()
	if errors == nil {
		t.Error("Expected non-nil errors slice")
	}
}

func TestEnhancedEnumerationConfig_Fields(t *testing.T) {
	cfg := &discovery.EnhancedEnumerationConfig{
		EnumerationConfig: *discovery.DefaultEnumerationConfig(),
		MultiPassARP:      true,
		ARPPasses:         3,
		GratuitousARP:     true,
		SlowScan:          true,
		RetryUnresponsive: true,
		RetryCount:        5,
	}

	if !cfg.MultiPassARP {
		t.Error("Expected MultiPassARP=true")
	}
	if cfg.ARPPasses != 3 {
		t.Errorf("Expected ARPPasses=3, got %d", cfg.ARPPasses)
	}
	if !cfg.GratuitousARP {
		t.Error("Expected GratuitousARP=true")
	}
	if !cfg.SlowScan {
		t.Error("Expected SlowScan=true")
	}
	if !cfg.RetryUnresponsive {
		t.Error("Expected RetryUnresponsive=true")
	}
	if cfg.RetryCount != 5 {
		t.Errorf("Expected RetryCount=5, got %d", cfg.RetryCount)
	}
}

func TestGenerateHostIPs(t *testing.T) {
	tests := []struct {
		name          string
		cidr          string
		expectedCount int
	}{
		{"slash24", "192.168.1.0/24", 254},
		{"slash30", "10.0.0.0/30", 2},
		{"slash31", "10.0.0.0/31", 0}, // /31 has 0 usable hosts in traditional calculation
		{"slash32", "10.0.0.0/32", 0}, // Single IP
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, subnet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR %s: %v", tt.cidr, err)
			}

			hosts := discovery.ExportGenerateHostIPs(subnet)
			if len(hosts) != tt.expectedCount {
				t.Errorf("Expected %d hosts for %s, got %d", tt.expectedCount, tt.cidr, len(hosts))
			}
		})
	}
}

func TestGenerateHostIPs_LargeSubnet(t *testing.T) {
	// Test that large subnets are capped to prevent overflow
	_, subnet, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("Failed to parse CIDR: %v", err)
	}

	hosts := discovery.ExportGenerateHostIPs(subnet)
	// Should be capped at 65534 (16 host bits max)
	if len(hosts) > 65534 {
		t.Errorf("Expected hosts to be capped, got %d", len(hosts))
	}
}

func TestEnumerationTiming_Fields(t *testing.T) {
	timing := discovery.EnumerationTiming{
		PingTimeout: 500 * time.Millisecond,
		ScanTimeout: 3 * time.Minute,
		PingWorkers: 100,
		ARPDelay:    5 * time.Millisecond,
	}

	if timing.PingTimeout != 500*time.Millisecond {
		t.Errorf("Expected PingTimeout=500ms, got %v", timing.PingTimeout)
	}
	if timing.ScanTimeout != 3*time.Minute {
		t.Errorf("Expected ScanTimeout=3m, got %v", timing.ScanTimeout)
	}
	if timing.PingWorkers != 100 {
		t.Errorf("Expected PingWorkers=100, got %d", timing.PingWorkers)
	}
	if timing.ARPDelay != 5*time.Millisecond {
		t.Errorf("Expected ARPDelay=5ms, got %v", timing.ARPDelay)
	}
}

func TestEnumerationError_Fields(t *testing.T) {
	now := time.Now()
	err := discovery.EnumerationError{
		Phase: "arp_scan",
		Error: context.DeadlineExceeded,
		Time:  now,
	}

	if err.Phase != "arp_scan" {
		t.Errorf("Expected Phase='arp_scan', got %s", err.Phase)
	}
	if !errors.Is(err.Error, context.DeadlineExceeded) {
		t.Errorf("Expected Error=context.DeadlineExceeded, got %v", err.Error)
	}
	if !err.Time.Equal(now) {
		t.Errorf("Expected Time=%v, got %v", now, err.Time)
	}
}

func TestEnumerationProgress_Concurrency(t *testing.T) {
	var progress discovery.EnumerationProgress
	progress.Start()

	// Test concurrent increments
	done := make(chan struct{})
	for range 10 {
		go func() {
			for range 100 {
				progress.IncrementDevices()
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	if progress.DevicesFound() != 1000 {
		t.Errorf("Expected DevicesFound=1000, got %d", progress.DevicesFound())
	}
}

func TestMethodsToStrings(t *testing.T) {
	methods := []discovery.Method{
		discovery.MethodARP,
		discovery.MethodLLDP,
		discovery.MethodPING,
	}

	// Use exported function through test accessor
	strings := discovery.ExportMethodsToStrings(methods)

	if len(strings) != 3 {
		t.Errorf("Expected 3 strings, got %d", len(strings))
	}
	if strings[0] != "arp" {
		t.Errorf("Expected strings[0]='arp', got %s", strings[0])
	}
	if strings[1] != "lldp" {
		t.Errorf("Expected strings[1]='lldp', got %s", strings[1])
	}
	if strings[2] != "ping" {
		t.Errorf("Expected strings[2]='ping', got %s", strings[2])
	}
}
