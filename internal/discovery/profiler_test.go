// Package discovery provides profiler tests.
package discovery

import (
	"testing"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/config"
)

func TestDefaultProfilerConfig(t *testing.T) {
	cfg := DefaultProfilerConfig()

	if cfg == nil {
		t.Fatal("DefaultProfilerConfig returned nil")
	}
	if !cfg.Enabled {
		t.Error("Expected Enabled=true by default")
	}
	if cfg.Timeout <= 0 {
		t.Error("Expected positive Timeout")
	}
	if cfg.MaxConcurrent <= 0 {
		t.Error("Expected positive MaxConcurrent")
	}
	if len(cfg.QuickPorts) == 0 {
		t.Error("Expected QuickPorts to have entries")
	}

	// Verify common ports are included
	portMap := make(map[int]bool)
	for _, p := range cfg.QuickPorts {
		portMap[p] = true
	}
	if !portMap[22] {
		t.Error("Expected port 22 (SSH) in QuickPorts")
	}
	if !portMap[80] {
		t.Error("Expected port 80 (HTTP) in QuickPorts")
	}
	if !portMap[443] {
		t.Error("Expected port 443 (HTTPS) in QuickPorts")
	}
}

func TestDeviceProfile_Fields(t *testing.T) {
	profile := DeviceProfile{
		ProfiledAt: time.Now(),
		OpenPorts: []OpenPort{
			{Port: 22, Protocol: "tcp", Service: "ssh", IsOpen: true},
			{Port: 80, Protocol: "tcp", Service: "http", IsOpen: true},
		},
		HTTPInfo: &HTTPInfo{
			Port:       80,
			StatusCode: 200,
			Title:      "Test Server",
			Server:     "nginx",
			IsHTTPS:    false,
		},
		SNMPInfo: &SNMPInfo{
			SysDescr:    "Test Device",
			SysName:     "test-device",
			SysContact:  "admin@test.com",
			SysLocation: "Server Room",
		},
		MDNSServices: []MDNSService{
			{Name: "Test", Type: "_http._tcp", Port: 80},
		},
		DeviceType:  "server",
		DeviceIcons: []string{"web", "ssh"},
	}

	if profile.ProfiledAt.IsZero() {
		t.Error("ProfiledAt should be set")
	}
	if len(profile.OpenPorts) != 2 {
		t.Errorf("Expected 2 open ports, got %d", len(profile.OpenPorts))
	}
	if profile.HTTPInfo == nil {
		t.Error("HTTPInfo should be set")
	}
	if profile.SNMPInfo == nil {
		t.Error("SNMPInfo should be set")
	}
	if len(profile.MDNSServices) != 1 {
		t.Errorf("Expected 1 mDNS service, got %d", len(profile.MDNSServices))
	}
	if profile.DeviceType != "server" {
		t.Errorf("Expected DeviceType 'server', got %q", profile.DeviceType)
	}
	if len(profile.DeviceIcons) != 2 {
		t.Errorf("Expected 2 DeviceIcons, got %d", len(profile.DeviceIcons))
	}
}

func TestOpenPort_Fields(t *testing.T) {
	port := OpenPort{
		Port:     443,
		Protocol: "tcp",
		Service:  "https",
		Banner:   "nginx/1.18.0",
		IsOpen:   true,
	}

	if port.Port != 443 {
		t.Errorf("Expected Port 443, got %d", port.Port)
	}
	if port.Protocol != "tcp" {
		t.Errorf("Expected Protocol 'tcp', got %q", port.Protocol)
	}
	if port.Service != "https" {
		t.Errorf("Expected Service 'https', got %q", port.Service)
	}
	if port.Banner != "nginx/1.18.0" {
		t.Errorf("Expected Banner 'nginx/1.18.0', got %q", port.Banner)
	}
	if !port.IsOpen {
		t.Error("Expected IsOpen=true")
	}
}

func TestHTTPInfo_Fields(t *testing.T) {
	info := HTTPInfo{
		Port:       8080,
		StatusCode: 301,
		Title:      "Redirect",
		Server:     "Apache",
		IsHTTPS:    true,
	}

	if info.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", info.Port)
	}
	if info.StatusCode != 301 {
		t.Errorf("Expected StatusCode 301, got %d", info.StatusCode)
	}
	if info.Title != "Redirect" {
		t.Errorf("Expected Title 'Redirect', got %q", info.Title)
	}
	if info.Server != "Apache" {
		t.Errorf("Expected Server 'Apache', got %q", info.Server)
	}
	if !info.IsHTTPS {
		t.Error("Expected IsHTTPS=true")
	}
}

func TestSNMPInfo_Fields(t *testing.T) {
	info := SNMPInfo{
		SysDescr:    "Cisco IOS",
		SysName:     "router01",
		SysContact:  "noc@company.com",
		SysLocation: "Data Center 1",
	}

	if info.SysDescr != "Cisco IOS" {
		t.Errorf("Expected SysDescr 'Cisco IOS', got %q", info.SysDescr)
	}
	if info.SysName != "router01" {
		t.Errorf("Expected SysName 'router01', got %q", info.SysName)
	}
	if info.SysContact != "noc@company.com" {
		t.Errorf("Expected SysContact 'noc@company.com', got %q", info.SysContact)
	}
	if info.SysLocation != "Data Center 1" {
		t.Errorf("Expected SysLocation 'Data Center 1', got %q", info.SysLocation)
	}
}

func TestMDNSService_Fields(t *testing.T) {
	svc := MDNSService{
		Name: "My Printer",
		Type: "_ipp._tcp",
		Port: 631,
		TXT:  map[string]string{"product": "HP LaserJet"},
	}

	if svc.Name != "My Printer" {
		t.Errorf("Expected Name 'My Printer', got %q", svc.Name)
	}
	if svc.Type != "_ipp._tcp" {
		t.Errorf("Expected Type '_ipp._tcp', got %q", svc.Type)
	}
	if svc.Port != 631 {
		t.Errorf("Expected Port 631, got %d", svc.Port)
	}
	if svc.TXT["product"] != "HP LaserJet" {
		t.Errorf("Expected TXT product='HP LaserJet', got %q", svc.TXT["product"])
	}
}

func TestNewDeviceProfiler(t *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Timeout:     5 * time.Second,
	}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	if profiler == nil {
		t.Fatal("NewDeviceProfiler returned nil")
	}
}

func TestDeviceProfiler_QueueProfile(t *testing.T) {
	cfg := DefaultProfilerConfig()
	cfg.Timeout = 100 * time.Millisecond // Short timeout for test
	snmpCfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Timeout:     100 * time.Millisecond,
	}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	// Queue should work even without starting
	profiler.QueueProfile("192.0.2.1") // TEST-NET-1, non-routable

	// Check not profiled yet (hasn't started)
	if profiler.IsProfiled("192.0.2.1") {
		t.Error("Should not be profiled yet without Start()")
	}
}

func TestDeviceProfiler_IsProfiled(t *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{Communities: []string{"public"}}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	// Not profiled initially
	if profiler.IsProfiled("10.0.0.1") {
		t.Error("Should not be profiled initially")
	}
}

func TestDeviceProfiler_IsProfiling(t *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{Communities: []string{"public"}}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	// Not profiling initially
	if profiler.IsProfiling("10.0.0.1") {
		t.Error("Should not be profiling initially")
	}
}

func TestDeviceProfiler_GetAllProfiles(t *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{Communities: []string{"public"}}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	profiles := profiler.GetAllProfiles()
	if len(profiles) != 0 {
		t.Errorf("Expected 0 profiles initially, got %d", len(profiles))
	}
}

func TestDeviceProfiler_ClearProfiles(t *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{Communities: []string{"public"}}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	// Clear should work even when empty
	profiler.ClearProfiles()

	profiles := profiler.GetAllProfiles()
	if len(profiles) != 0 {
		t.Errorf("Expected 0 profiles after clear, got %d", len(profiles))
	}
}

func TestDeviceProfiler_StartStop(_ *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{Communities: []string{"public"}}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	// Start profiler
	profiler.Start()

	// Stop profiler
	profiler.Stop()

	// Should be safe to stop multiple times
	profiler.Stop()
}

func TestDeviceProfiler_GetProfile_NonExistent(t *testing.T) {
	cfg := DefaultProfilerConfig()
	snmpCfg := &config.SNMPConfig{Communities: []string{"public"}}

	profiler := NewDeviceProfiler(cfg, snmpCfg)

	profile := profiler.GetProfile("10.255.255.1")
	if profile != nil {
		t.Error("Expected nil for non-existent profile")
	}
}

func TestDeviceProfiler_ProfilerConfig(t *testing.T) {
	cfg := &ProfilerConfig{
		Enabled:       false,
		Timeout:       5 * time.Second,
		MaxConcurrent: 20,
		QuickPorts:    []int{22, 80},
	}

	if cfg.Enabled {
		t.Error("Expected Enabled=false")
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Expected Timeout=5s, got %v", cfg.Timeout)
	}
	if cfg.MaxConcurrent != 20 {
		t.Errorf("Expected MaxConcurrent=20, got %d", cfg.MaxConcurrent)
	}
	if len(cfg.QuickPorts) != 2 {
		t.Errorf("Expected 2 QuickPorts, got %d", len(cfg.QuickPorts))
	}
}
