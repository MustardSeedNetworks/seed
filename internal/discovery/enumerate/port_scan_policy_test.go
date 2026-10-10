package enumerate_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// TestPortScanSettingReachesTheProfiler pins seed#2926: the drawer's port-scan
// toggle and port list change which ports the profiler probes on every
// discovered device, at construction and after a settings reload.
func TestPortScanSettingReachesTheProfiler(t *testing.T) {
	cfg := testutil.NewConfigBuilder().WithInterface("lo").Build()
	cfg.NetworkDiscovery.Options.PortScan = config.PortScanConfig{
		Enabled:  true,
		Preset:   config.PortPresetCustom,
		TCPPorts: "9100,5000-5001",
	}

	service := enumerate.NewService(cfg, enumerate.NewDeviceDiscovery("lo"), nil)
	intensity, ports, _ := service.GetProfiler().ScanConfigSnapshot()
	if intensity != discovery.PortScanCustom {
		t.Fatalf("intensity = %q with port scanning on, want %q", intensity, discovery.PortScanCustom)
	}
	for _, port := range []int{9100, 5000, 5001} {
		if !slices.Contains(ports, port) {
			t.Errorf("configured port %d is not scanned; profiler ports = %v", port, ports)
		}
	}
	// The quick list still classifies the device (seed#2674).
	for _, port := range discovery.GetQuickPorts() {
		if !slices.Contains(ports, port) {
			t.Errorf("quick classification port %d dropped; profiler ports = %v", port, ports)
		}
	}

	cfg.NetworkDiscovery.Options.PortScan.Enabled = false
	if err := service.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	intensity, ports, _ = service.GetProfiler().ScanConfigSnapshot()
	if intensity != discovery.PortScanQuick || len(ports) != 0 {
		t.Errorf("after turning port scanning off: intensity %q ports %v, want %q and none",
			intensity, ports, discovery.PortScanQuick)
	}

	cfg.NetworkDiscovery.Options.PortScan = config.PortScanConfig{Enabled: true, Preset: config.PortPresetSecure}
	if err := service.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	_, ports, _ = service.GetProfiler().ScanConfigSnapshot()
	if !slices.Contains(ports, 9443) {
		t.Errorf("the secure preset's port 9443 is not scanned after reload; profiler ports = %v", ports)
	}
}
