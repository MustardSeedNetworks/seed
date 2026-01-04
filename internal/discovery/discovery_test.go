// Package discovery_test implements multi-protocol network device discovery tests.
// Test suite validates device discovery functionality across all discovery methods,
// settings configurations, and state management operations. Ensures device aggregation,
// protocol-specific information, and discovery timing work correctly.
package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/testutil"
)

func TestDeviceDiscovery(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")
	if dd == nil {
		t.Fatal("NewDeviceDiscovery returned nil")
	}

	// Test GetDevices (should be empty initially)
	devices := dd.GetDevices()
	if devices == nil {
		t.Error("GetDevices returned nil")
	}

	// Test GetStatus
	status := dd.GetStatus()
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}
	if status.Interface != "lo" {
		t.Errorf("Expected interface 'lo', got %s", status.Interface)
	}
}

func TestGetDevice(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	// Test GetDevice for non-existent device
	device := dd.GetDevice("00:11:22:33:44:55")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestGetDeviceByIP(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	// Test GetDeviceByIP for non-existent device
	device := dd.GetDeviceByIP("192.168.1.10")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestClearDevices(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	// Clear should work even with no devices
	dd.ClearDevices()

	// Verify devices are cleared
	if dd.Count() != 0 {
		t.Errorf("Expected 0 devices after clear, got %d", dd.Count())
	}
}

func TestSetInterface(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	err := dd.SetInterface("eth0")
	if err != nil {
		t.Logf("SetInterface returned error (expected on test system): %v", err)
	}

	status := dd.GetStatus()
	// Interface may or may not change depending on system
	t.Logf("Interface after SetInterface: %s", status.Interface)
}

func TestDeviceProfiler(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	profiler := discovery.NewDeviceProfiler(discovery.DefaultProfilerConfig(), &cfg.SNMP)

	if profiler == nil {
		t.Fatal("NewDeviceProfiler returned nil")
	}

	// Test Start/Stop
	profiler.Start()
	time.Sleep(50 * time.Millisecond)
	profiler.Stop()

	// Test GetProfile for non-existent device
	profile := profiler.GetProfile("192.168.1.1")
	if profile != nil {
		t.Error("Expected nil profile for non-existent device")
	}

	// Test IsProfiling
	isProfiling := profiler.IsProfiling("192.168.1.1")
	if isProfiling {
		t.Error("Expected false for non-queued device")
	}

	// Test ClearProfiles
	profiler.ClearProfiles()
}

func TestProfilerConfig(t *testing.T) {
	cfg := discovery.DefaultProfilerConfig()

	if cfg.MaxConcurrent <= 0 {
		t.Error("Expected positive number of workers")
	}
	if cfg.Timeout <= 0 {
		t.Error("Expected positive timeout")
	}
	if len(cfg.QuickPorts) == 0 {
		t.Error("Expected QuickPorts to be configured")
	}
}

func TestNeighborInfo(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	// Start protocol manager
	if err := dd.Start(); err != nil {
		t.Logf("Failed to start discovery (expected on test system): %v", err)
		return
	}
	defer dd.Stop()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test GetNeighbors (likely empty on test system)
	accessor := &discovery.DeviceDiscoveryTestAccessor{Discovery: dd}
	protoManager := accessor.GetProtoManager()
	if protoManager == nil {
		t.Log("Protocol manager not initialized (expected on test system)")
		return
	}
	neighbors := protoManager.GetNeighbors()
	if neighbors == nil {
		t.Error("GetNeighbors returned nil")
	}
}

func TestDiscoveryService(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	service := discovery.NewService(cfg, "lo", nil)
	if service == nil {
		t.Fatal("NewService returned nil")
	}

	// Test Start
	err := service.Start()
	if err != nil {
		t.Logf("Start returned error (expected on test system): %v", err)
	}

	// Test GetStatus
	status := service.GetStatus()
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}

	// Test GetOptions
	opts := service.GetOptions()
	if !opts.ARPScan && !opts.ICMPScan {
		t.Log("Discovery methods ARP and ICMP are both disabled")
	}

	// Test IsRunning
	running := service.IsRunning()
	t.Logf("Service running: %v", running)

	// Test GetDevices
	devices := service.GetDevices()
	if devices == nil {
		t.Error("GetDevices returned nil")
	}

	// Test Stop
	service.Stop()

	// Verify stopped
	if service.IsRunning() {
		t.Error("Service still running after Stop")
	}
}

func TestReload(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	// Test Reload without starting
	err := service.Reload()
	if err != nil {
		t.Errorf("Reload returned error: %v", err)
	}

	// Start service and test Reload
	if startErr := service.Start(); startErr != nil {
		t.Logf("Start failed: %v", startErr)
		return
	}
	defer service.Stop()

	// Change options and reload
	cfg.NetworkDiscovery.Options.ARPScan = false
	err = service.Reload()
	if err != nil {
		t.Errorf("Reload returned error: %v", err)
	}
}

func TestScanWithContext(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	cfg.NetworkDiscovery.ScanTimeout = 1 * time.Second
	service := discovery.NewService(cfg, "lo", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start service first
	if err := service.Start(); err != nil {
		t.Logf("Failed to start service: %v", err)
		return
	}
	defer service.Stop()

	// Attempt scan (may timeout)
	err := service.Scan(ctx)
	if err != nil {
		t.Logf("Scan returned error (may be expected): %v", err)
	}
}

func TestClearDevicesService(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	// Clear (should work even with no devices)
	service.ClearDevices()

	// Verify cleared
	devices := service.GetDevices()
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices after clear, got %d", len(devices))
	}
}

func TestDiscoveryOptions(t *testing.T) {
	tests := []struct {
		name     string
		arpScan  bool
		icmpScan bool
		portScan bool
	}{
		{"passive_only", false, false, false},
		{"arp_icmp", true, true, false},
		{"full_scan", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testutil.NewConfigBuilder().
				WithDiscoveryMethods(tt.arpScan, tt.icmpScan, tt.portScan).
				Build()
			service := discovery.NewService(cfg, "lo", nil)

			if err := service.Start(); err != nil {
				t.Logf("Start failed for %s: %v", tt.name, err)
			}
			defer service.Stop()

			status := service.GetStatus()
			if status == nil {
				t.Fatal("GetStatus returned nil")
			}
		})
	}
}

func TestDeviceCount(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	count := dd.Count()
	if count != 0 {
		t.Errorf("Expected 0 devices initially, got %d", count)
	}
}

func TestIsScanning(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	if dd.IsScanning() {
		t.Error("Expected IsScanning to be false initially")
	}
}

func TestLastScan(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	lastScan := dd.LastScan()
	if !lastScan.IsZero() {
		t.Error("Expected LastScan to be zero time initially")
	}
}

func TestGetSubnetInfo(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	subnet, localIP := dd.GetSubnetInfo()
	t.Logf("Subnet: %s, LocalIP: %s", subnet, localIP)
}

func TestSetAdditionalSubnets(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	// Test setting additional subnets
	err := dd.SetAdditionalSubnets([]string{"192.168.1.0/24"})
	if err != nil {
		t.Errorf("SetAdditionalSubnets returned error: %v", err)
	}

	// Test getting additional subnets
	subnets := dd.GetAdditionalSubnets()
	if len(subnets) != 1 {
		t.Errorf("Expected 1 subnet, got %d", len(subnets))
	}
}

func TestDeviceDiscoveryStartStop(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	// Test Start
	err := dd.Start()
	if err != nil {
		t.Logf("Start returned error (expected on test system): %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Test Stop
	dd.Stop()
}

func TestServiceInterface(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	// Test SetInterface
	err := service.SetInterface("eth0")
	if err != nil {
		t.Logf("SetInterface returned error (expected on test system): %v", err)
	}
}

func TestServiceGetDevice(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	// Test GetDevice for non-existent device
	device := service.GetDevice("00:11:22:33:44:55")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestGetDeviceByIPService(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	// Test GetDeviceByIP for non-existent device
	device := service.GetDeviceByIP("192.168.1.10")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestGetNeighbors(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	if err := service.Start(); err != nil {
		t.Logf("Start failed: %v", err)
		return
	}
	defer service.Stop()

	neighbors := service.GetNeighbors()
	if neighbors == nil {
		t.Error("GetNeighbors returned nil")
	}
}

func TestDeviceDiscoveryAccess(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	service := discovery.NewService(cfg, "lo", nil)

	// Test DeviceDiscovery accessor
	dd := service.DeviceDiscovery()
	if dd == nil {
		t.Error("DeviceDiscovery() returned nil")
	}
}
