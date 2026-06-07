package discovery_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// These cover the port-scan config vocabulary relocated into scan_config.go
// when the pipeline orchestrator was retired (Phase 7). The engine's
// DeviceProfiler.GetPortsForIntensity depends on them.

func TestPortScanIntensity_Constants(t *testing.T) {
	intensities := []discovery.PortScanIntensity{
		discovery.PortScanOff,
		discovery.PortScanQuick,
		discovery.PortScanStandard,
		discovery.PortScanComprehensive,
		discovery.PortScanCustom,
	}

	seen := make(map[discovery.PortScanIntensity]bool)
	for _, intensity := range intensities {
		if seen[intensity] {
			t.Errorf("Duplicate intensity: %q", intensity)
		}
		seen[intensity] = true
	}
}

func TestScanTimingProfile_Constants(t *testing.T) {
	profiles := []discovery.ScanTimingProfile{
		discovery.ScanProfilePolite,
		discovery.ScanProfileNormal,
		discovery.ScanProfileAggressive,
	}

	seen := make(map[discovery.ScanTimingProfile]bool)
	for _, profile := range profiles {
		if seen[profile] {
			t.Errorf("Duplicate profile: %q", profile)
		}
		seen[profile] = true
	}
}

func TestGetQuickPorts(t *testing.T) {
	ports := discovery.GetQuickPorts()
	if len(ports) == 0 {
		t.Fatal("GetQuickPorts should return ports")
	}
	for _, expected := range []int{22, 80, 443} {
		if !slices.Contains(ports, expected) {
			t.Errorf("Quick ports should include %d", expected)
		}
	}
}

func TestGetStandardPorts(t *testing.T) {
	ports := discovery.GetStandardPorts()
	if len(ports) < 30 {
		t.Errorf("Standard ports should have at least 30 ports, got %d", len(ports))
	}
	for _, expected := range []int{22, 23, 80, 443, 3306, 5432} {
		if !slices.Contains(ports, expected) {
			t.Errorf("Standard ports should include %d", expected)
		}
	}
}

func TestGetComprehensivePorts(t *testing.T) {
	ports := discovery.GetComprehensivePorts()
	if len(ports) < 500 {
		t.Errorf("Comprehensive ports should have at least 500 ports, got %d", len(ports))
	}
}
