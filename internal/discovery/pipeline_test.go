package discovery_test

import (
	"slices"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestDefaultPipelineConfig(t *testing.T) {
	cfg := discovery.DefaultPipelineConfig()

	// Check phases
	if !cfg.Phases.Enumeration {
		t.Error("Enumeration should be enabled by default")
	}
	if !cfg.Phases.NameResolution {
		t.Error("NameResolution should be enabled by default")
	}
	if cfg.Phases.ServiceDiscovery {
		t.Error("ServiceDiscovery should be disabled by default")
	}
	if cfg.Phases.VulnAssessment {
		t.Error("VulnAssessment should be disabled by default")
	}

	// Check timing
	if cfg.Timing.Profile != discovery.ScanProfileNormal {
		t.Errorf("Expected normal timing profile, got %q", cfg.Timing.Profile)
	}
	if cfg.Timing.MaxConcurrentHosts <= 0 {
		t.Error("MaxConcurrentHosts should be positive")
	}

	// Check port scan
	if cfg.PortScan.Intensity != discovery.PortScanOff {
		t.Errorf("Expected port scan off by default, got %q", cfg.PortScan.Intensity)
	}

	// Check SNMP collection
	if !cfg.SNMPCollection.Enabled {
		t.Error("SNMP collection should be enabled by default")
	}
	if !cfg.SNMPCollection.MIBs.System {
		t.Error("System MIB should be enabled by default")
	}

	// Check resolution
	if !cfg.Resolution.DNS {
		t.Error("DNS resolution should be enabled by default")
	}
	if !cfg.Resolution.NetBIOS {
		t.Error("NetBIOS resolution should be enabled by default")
	}
	if !cfg.Resolution.MDNS {
		t.Error("mDNS resolution should be enabled by default")
	}

	// Check persistence
	if !cfg.Persistence.StoreHistory {
		t.Error("StoreHistory should be enabled by default")
	}
}

func TestPipelineState_Constants(t *testing.T) {
	states := []discovery.PipelineState{
		discovery.PipelineStateIdle,
		discovery.PipelineStateEnumerating,
		discovery.PipelineStateResolving,
		discovery.PipelineStateScanning,
		discovery.PipelineStateAssessing,
		discovery.PipelineStateComplete,
		discovery.PipelineStateFailed,
		discovery.PipelineStateCanceled,
	}

	// All states should be distinct
	seen := make(map[discovery.PipelineState]bool)
	for _, state := range states {
		if seen[state] {
			t.Errorf("Duplicate state: %q", state)
		}
		seen[state] = true
	}
}

func TestPipelineEventType_Constants(t *testing.T) {
	events := []discovery.PipelineEventType{
		discovery.EventPipelineStarted,
		discovery.EventPhaseStarted,
		discovery.EventPhaseProgress,
		discovery.EventPhaseCompleted,
		discovery.EventPhaseFailed,
		discovery.EventDeviceDiscovered,
		discovery.EventDeviceUpdated,
		discovery.EventPipelineCompleted,
		discovery.EventPipelineFailed,
		discovery.EventPipelineCanceled,
	}

	// All events should be distinct
	seen := make(map[discovery.PipelineEventType]bool)
	for _, event := range events {
		if seen[event] {
			t.Errorf("Duplicate event type: %q", event)
		}
		seen[event] = true
	}
}

func TestPortScanIntensity_Constants(t *testing.T) {
	intensities := []discovery.PortScanIntensity{
		discovery.PortScanOff,
		discovery.PortScanQuick,
		discovery.PortScanStandard,
		discovery.PortScanComprehensive,
		discovery.PortScanCustom,
	}

	// All should be distinct
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

	// All should be distinct
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

	// Quick scan should include essential web ports
	expectedPorts := []int{22, 80, 443}
	for _, expected := range expectedPorts {
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

	// Check for common enterprise ports
	expectedPorts := []int{22, 23, 80, 443, 3306, 5432}
	for _, expected := range expectedPorts {
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

func TestGetScanTimingPresets(t *testing.T) {
	presets := discovery.GetScanTimingPresets()

	if len(presets) != 3 {
		t.Errorf("Expected 3 timing presets, got %d", len(presets))
	}

	// Check polite profile
	polite, ok := presets[discovery.ScanProfilePolite]
	if !ok {
		t.Fatal("Missing polite profile")
	}
	if polite.MaxConcurrentHosts <= 0 {
		t.Error("Polite profile should have positive MaxConcurrentHosts")
	}

	// Check normal profile
	normal, ok := presets[discovery.ScanProfileNormal]
	if !ok {
		t.Fatal("Missing normal profile")
	}

	// Check aggressive profile
	aggressive, ok := presets[discovery.ScanProfileAggressive]
	if !ok {
		t.Fatal("Missing aggressive profile")
	}

	// Aggressive should have more concurrent hosts than polite
	if aggressive.MaxConcurrentHosts <= polite.MaxConcurrentHosts {
		t.Error("Aggressive should have more concurrent hosts than polite")
	}

	// Aggressive should have shorter delays
	if aggressive.ProbeDelay >= normal.ProbeDelay {
		t.Error("Aggressive should have shorter probe delay")
	}
}

func TestPipelineConfig_Fields(t *testing.T) {
	cfg := discovery.PipelineConfig{
		Phases: discovery.PipelinePhaseConfig{
			Enumeration:      true,
			NameResolution:   true,
			ServiceDiscovery: true,
			VulnAssessment:   true,
		},
		Timing: discovery.PipelineTiming{
			ProbeDelay:         100 * time.Millisecond,
			HostDelay:          50 * time.Millisecond,
			MaxConcurrentHosts: 30,
			PhaseTimeout:       15 * time.Minute,
			Profile:            discovery.ScanProfileAggressive,
		},
		PortScan: discovery.PipelinePortScanConfig{
			Intensity:      discovery.PortScanStandard,
			CustomPorts:    []int{22, 80},
			BannerGrab:     true,
			ConnectTimeout: 3 * time.Second,
		},
		SNMPCollection: discovery.SNMPCollectionConfig{
			Enabled: true,
			MIBs: discovery.SNMPMIBSelection{
				System:      true,
				Interfaces:  true,
				IPAddresses: true,
			},
		},
		Resolution: discovery.PipelineResolutionConfig{
			DNS:     true,
			NetBIOS: true,
			MDNS:    true,
		},
		Persistence: discovery.PipelinePersistenceConfig{
			StoreHistory:       true,
			StalenessThreshold: 24 * time.Hour,
			PurgeAfter:         720 * time.Hour,
		},
	}

	if !cfg.Phases.ServiceDiscovery {
		t.Error("ServiceDiscovery should be enabled")
	}
	if cfg.Timing.MaxConcurrentHosts != 30 {
		t.Errorf("MaxConcurrentHosts should be 30, got %d", cfg.Timing.MaxConcurrentHosts)
	}
	if cfg.PortScan.Intensity != discovery.PortScanStandard {
		t.Errorf("Intensity should be standard, got %q", cfg.PortScan.Intensity)
	}
	if len(cfg.PortScan.CustomPorts) != 2 {
		t.Errorf("CustomPorts should have 2 entries, got %d", len(cfg.PortScan.CustomPorts))
	}
}

func TestPipelineEvent_Fields(t *testing.T) {
	payload := map[string]string{"test": "data"}
	event := discovery.PipelineEvent{
		Type:      discovery.EventPhaseStarted,
		Timestamp: time.Now(),
		RunID:     "test-run-123",
		Payload:   payload,
	}

	if event.Type != discovery.EventPhaseStarted {
		t.Errorf("Type mismatch, got %q", event.Type)
	}
	if event.RunID != "test-run-123" {
		t.Errorf("RunID mismatch, got %q", event.RunID)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	if event.Payload == nil {
		t.Error("Payload should be set")
	}
}

func TestPipelineRun_Fields(t *testing.T) {
	now := time.Now()
	completed := now.Add(5 * time.Minute)

	run := discovery.PipelineRun{
		ID:             "run-abc123",
		StartedAt:      now,
		CompletedAt:    &completed,
		Status:         discovery.PipelineStateComplete,
		Trigger:        "manual",
		CurrentPhase:   "scanning",
		PhaseDurations: map[string]time.Duration{"enumeration": 30 * time.Second},
		DevicesFound:   42,
		Errors:         []string{"warning: timeout on device"},
	}

	if run.ID != "run-abc123" {
		t.Errorf("ID mismatch, got %q", run.ID)
	}
	if !run.StartedAt.Equal(now) {
		t.Error("StartedAt mismatch")
	}
	if run.CompletedAt == nil || !run.CompletedAt.Equal(completed) {
		t.Error("CompletedAt mismatch")
	}
	if run.Status != discovery.PipelineStateComplete {
		t.Errorf("Status mismatch, got %q", run.Status)
	}
	if run.Trigger != "manual" {
		t.Errorf("Trigger should be 'manual', got %q", run.Trigger)
	}
	if run.CurrentPhase != "scanning" {
		t.Errorf("CurrentPhase should be 'scanning', got %q", run.CurrentPhase)
	}
	if run.DevicesFound != 42 {
		t.Errorf("DevicesFound should be 42, got %d", run.DevicesFound)
	}
	if len(run.Errors) != 1 {
		t.Errorf("Errors should have 1 entry, got %d", len(run.Errors))
	}
	if run.PhaseDurations["enumeration"] != 30*time.Second {
		t.Errorf("PhaseDurations[enumeration] mismatch")
	}
}

func TestPhaseStartedPayload_Fields(t *testing.T) {
	payload := discovery.PhaseStartedPayload{
		Phase:       "enumeration",
		PhaseNumber: 1,
		TotalPhases: 4,
		DeviceCount: 0,
	}

	if payload.Phase != "enumeration" {
		t.Errorf("Phase mismatch, got %q", payload.Phase)
	}
	if payload.PhaseNumber != 1 {
		t.Errorf("PhaseNumber should be 1, got %d", payload.PhaseNumber)
	}
	if payload.TotalPhases != 4 {
		t.Errorf("TotalPhases should be 4, got %d", payload.TotalPhases)
	}
	if payload.DeviceCount != 0 {
		t.Errorf("DeviceCount should be 0, got %d", payload.DeviceCount)
	}
}

func TestPhaseProgressPayload_Fields(t *testing.T) {
	payload := discovery.PhaseProgressPayload{
		Phase:             "scanning",
		ProcessedCount:    25,
		TotalCount:        50,
		PercentComplete:   50.0,
		CurrentTarget:     "192.168.1.10",
		ElapsedMs:         5000,
		EstimatedRemainMs: 5000,
	}

	if payload.Phase != "scanning" {
		t.Errorf("Phase should be 'scanning', got %q", payload.Phase)
	}
	if payload.ProcessedCount != 25 {
		t.Errorf("ProcessedCount should be 25, got %d", payload.ProcessedCount)
	}
	if payload.TotalCount != 50 {
		t.Errorf("TotalCount should be 50, got %d", payload.TotalCount)
	}
	if payload.PercentComplete != 50.0 {
		t.Errorf("PercentComplete should be 50.0, got %f", payload.PercentComplete)
	}
	if payload.CurrentTarget != "192.168.1.10" {
		t.Errorf("CurrentTarget mismatch, got %q", payload.CurrentTarget)
	}
	if payload.ElapsedMs != 5000 {
		t.Errorf("ElapsedMs should be 5000, got %d", payload.ElapsedMs)
	}
	if payload.EstimatedRemainMs != 5000 {
		t.Errorf("EstimatedRemainMs should be 5000, got %d", payload.EstimatedRemainMs)
	}
}

func TestPhaseCompletedPayload_Fields(t *testing.T) {
	payload := discovery.PhaseCompletedPayload{
		Phase:         "enumeration",
		DevicesFound:  42,
		NamesResolved: 35,
		PortsOpen:     120,
		VulnsFound:    5,
		Duration:      30 * time.Second,
		Errors:        []string{"timeout on device"},
	}

	if payload.Phase != "enumeration" {
		t.Errorf("Phase should be 'enumeration', got %q", payload.Phase)
	}
	if payload.DevicesFound != 42 {
		t.Errorf("DevicesFound should be 42, got %d", payload.DevicesFound)
	}
	if payload.NamesResolved != 35 {
		t.Errorf("NamesResolved should be 35, got %d", payload.NamesResolved)
	}
	if payload.PortsOpen != 120 {
		t.Errorf("PortsOpen should be 120, got %d", payload.PortsOpen)
	}
	if payload.VulnsFound != 5 {
		t.Errorf("VulnsFound should be 5, got %d", payload.VulnsFound)
	}
	if payload.Duration != 30*time.Second {
		t.Errorf("Duration mismatch, got %v", payload.Duration)
	}
	if len(payload.Errors) != 1 {
		t.Errorf("Errors should have 1 entry, got %d", len(payload.Errors))
	}
}

func TestDeviceDiscoveredPayload_Fields(t *testing.T) {
	payload := discovery.DeviceDiscoveredPayload{
		IP:      "192.168.1.10",
		MAC:     "00:11:22:33:44:55",
		Vendor:  "Apple",
		Methods: []string{"arp", "icmp"},
		IsNew:   true,
	}

	if payload.IP != "192.168.1.10" {
		t.Errorf("IP mismatch, got %q", payload.IP)
	}
	if payload.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC mismatch, got %q", payload.MAC)
	}
	if payload.Vendor != "Apple" {
		t.Errorf("Vendor should be 'Apple', got %q", payload.Vendor)
	}
	if !payload.IsNew {
		t.Error("IsNew should be true")
	}
	if len(payload.Methods) != 2 {
		t.Errorf("Methods should have 2 entries, got %d", len(payload.Methods))
	}
}

func TestPipelineCompletedPayload_Fields(t *testing.T) {
	payload := discovery.PipelineCompletedPayload{
		TotalDevices:   50,
		NewDevices:     5,
		UpdatedDevices: 10,
		StaleDevices:   2,
		TotalDuration:  5 * time.Minute,
		PhaseDurations: map[string]time.Duration{
			"enumeration": 30 * time.Second,
			"resolution":  60 * time.Second,
		},
	}

	if payload.TotalDevices != 50 {
		t.Errorf("TotalDevices should be 50, got %d", payload.TotalDevices)
	}
	if payload.NewDevices != 5 {
		t.Errorf("NewDevices should be 5, got %d", payload.NewDevices)
	}
	if payload.UpdatedDevices != 10 {
		t.Errorf("UpdatedDevices should be 10, got %d", payload.UpdatedDevices)
	}
	if payload.StaleDevices != 2 {
		t.Errorf("StaleDevices should be 2, got %d", payload.StaleDevices)
	}
	if payload.TotalDuration != 5*time.Minute {
		t.Errorf("TotalDuration mismatch, got %v", payload.TotalDuration)
	}
	if len(payload.PhaseDurations) != 2 {
		t.Errorf("PhaseDurations should have 2 entries, got %d", len(payload.PhaseDurations))
	}
}

func TestSNMPMIBSelection_Fields(t *testing.T) {
	mibs := discovery.SNMPMIBSelection{
		System:      true,
		Interfaces:  true,
		IPAddresses: true,
		Routing:     true,
		Bridge:      false,
		Entity:      false,
		LLDP:        true,
		VLAN:        true,
	}

	if !mibs.System {
		t.Error("System should be enabled")
	}
	if !mibs.Interfaces {
		t.Error("Interfaces should be enabled")
	}
	if !mibs.IPAddresses {
		t.Error("IPAddresses should be enabled")
	}
	if !mibs.Routing {
		t.Error("Routing should be enabled")
	}
	if mibs.Bridge {
		t.Error("Bridge should be disabled")
	}
	if mibs.Entity {
		t.Error("Entity should be disabled")
	}
	if !mibs.LLDP {
		t.Error("LLDP should be enabled")
	}
	if !mibs.VLAN {
		t.Error("VLAN should be enabled")
	}
}
