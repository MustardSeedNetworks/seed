package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/config"
)

func TestDeviceDiscovery(t *testing.T) {
	dd := NewDeviceDiscovery("lo")
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
	dd := NewDeviceDiscovery("lo")

	// Test GetDevice for non-existent device
	device := dd.GetDevice("00:11:22:33:44:55")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestGetDeviceByIP(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	// Test GetDeviceByIP for non-existent device
	device := dd.GetDeviceByIP("192.168.1.10")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestClearDevices(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	// Clear should work even with no devices
	dd.ClearDevices()

	// Verify devices are cleared
	if dd.Count() != 0 {
		t.Errorf("Expected 0 devices after clear, got %d", dd.Count())
	}
}

func TestSetInterface(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	err := dd.SetInterface("eth0")
	if err != nil {
		t.Logf("SetInterface returned error (expected on test system): %v", err)
	}

	status := dd.GetStatus()
	// Interface may or may not change depending on system
	t.Logf("Interface after SetInterface: %s", status.Interface)
}

func TestDeviceProfiler(t *testing.T) {
	cfg := config.DefaultConfig()
	profiler := NewDeviceProfiler(DefaultProfilerConfig(), &cfg.SNMP)

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
	cfg := DefaultProfilerConfig()

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
	dd := NewDeviceDiscovery("lo")

	// Start protocol manager
	if err := dd.Start(); err != nil {
		t.Logf("Failed to start discovery (expected on test system): %v", err)
		return
	}
	defer dd.Stop()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test GetNeighbors (likely empty on test system)
	neighbors := dd.protoManager.GetNeighbors()
	if neighbors == nil {
		t.Error("GetNeighbors returned nil")
	}
}

func TestDiscoveryService(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Interface.Default = "lo"
	cfg.NetworkDiscovery.Profile = "standard"

	service := NewService(cfg, "lo")
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

	// Test GetProfile
	profile := service.GetProfile()
	if profile != "standard" {
		t.Errorf("Expected profile 'standard', got %s", profile)
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

func TestSetProfile(t *testing.T) {
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

	tests := []struct {
		profile config.DiscoveryProfile
	}{
		{"stealth"},
		{"standard"},
		{"full_scan"},
		{"custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			err := service.SetProfile(tt.profile)
			if err != nil {
				t.Errorf("SetProfile(%s) returned error: %v", tt.profile, err)
			}

			if service.GetProfile() != tt.profile {
				t.Errorf("Expected profile %s, got %s", tt.profile, service.GetProfile())
			}
		})
	}
}

func TestScanWithContext(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.NetworkDiscovery.ScanTimeout = 1 * time.Second
	service := NewService(cfg, "lo")

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
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

	// Clear (should work even with no devices)
	service.ClearDevices()

	// Verify cleared
	devices := service.GetDevices()
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices after clear, got %d", len(devices))
	}
}

func TestDiscoveryProfiles(t *testing.T) {
	profiles := []config.DiscoveryProfile{
		"stealth",
		"standard",
		"full_scan",
		"custom",
	}

	for _, profile := range profiles {
		t.Run(string(profile), func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.NetworkDiscovery.Profile = profile
			service := NewService(cfg, "lo")

			if err := service.Start(); err != nil {
				t.Logf("Start failed for profile %s: %v", profile, err)
			}
			defer service.Stop()

			status := service.GetStatus()
			if status.Profile != profile {
				t.Errorf("Expected profile %s, got %s", profile, status.Profile)
			}
		})
	}
}

func TestDeviceCount(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	count := dd.Count()
	if count != 0 {
		t.Errorf("Expected 0 devices initially, got %d", count)
	}
}

func TestIsScanning(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	if dd.IsScanning() {
		t.Error("Expected IsScanning to be false initially")
	}
}

func TestLastScan(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	lastScan := dd.LastScan()
	if !lastScan.IsZero() {
		t.Error("Expected LastScan to be zero time initially")
	}
}

func TestGetSubnetInfo(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

	subnet, localIP := dd.GetSubnetInfo()
	t.Logf("Subnet: %s, LocalIP: %s", subnet, localIP)
}

func TestSetAdditionalSubnets(t *testing.T) {
	dd := NewDeviceDiscovery("lo")

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
	dd := NewDeviceDiscovery("lo")

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
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

	// Test SetInterface
	err := service.SetInterface("eth0")
	if err != nil {
		t.Logf("SetInterface returned error (expected on test system): %v", err)
	}
}

func TestServiceGetDevice(t *testing.T) {
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

	// Test GetDevice for non-existent device
	device := service.GetDevice("00:11:22:33:44:55")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestGetDeviceByIPService(t *testing.T) {
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

	// Test GetDeviceByIP for non-existent device
	device := service.GetDeviceByIP("192.168.1.10")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

func TestGetNeighbors(t *testing.T) {
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

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
	cfg := config.DefaultConfig()
	service := NewService(cfg, "lo")

	// Test DeviceDiscovery accessor
	dd := service.DeviceDiscovery()
	if dd == nil {
		t.Error("DeviceDiscovery() returned nil")
	}
}
