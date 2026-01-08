package discovery_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestDefaultPipelineConfig_AllFields(t *testing.T) {
	cfg := discovery.DefaultPipelineConfig()

	// Test all phase defaults
	t.Run("phases", func(t *testing.T) {
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
	})

	// Test timing defaults
	t.Run("timing", func(t *testing.T) {
		if cfg.Timing.ProbeDelay <= 0 {
			t.Error("ProbeDelay should be positive")
		}
		if cfg.Timing.HostDelay <= 0 {
			t.Error("HostDelay should be positive")
		}
		if cfg.Timing.MaxConcurrentHosts <= 0 {
			t.Error("MaxConcurrentHosts should be positive")
		}
		if cfg.Timing.PhaseTimeout <= 0 {
			t.Error("PhaseTimeout should be positive")
		}
	})

	// Test port scan defaults
	t.Run("portScan", func(t *testing.T) {
		if cfg.PortScan.Intensity != discovery.PortScanOff {
			t.Errorf("PortScan.Intensity should be off by default, got %q", cfg.PortScan.Intensity)
		}
		if cfg.PortScan.ConnectTimeout <= 0 {
			t.Error("PortScan.ConnectTimeout should be positive")
		}
	})

	// Test SNMP defaults
	t.Run("snmpCollection", func(t *testing.T) {
		if !cfg.SNMPCollection.Enabled {
			t.Error("SNMPCollection should be enabled by default")
		}
		if !cfg.SNMPCollection.MIBs.System {
			t.Error("System MIB should be enabled by default")
		}
		if !cfg.SNMPCollection.MIBs.Interfaces {
			t.Error("Interfaces MIB should be enabled by default")
		}
	})

	// Test resolution defaults
	t.Run("resolution", func(t *testing.T) {
		if !cfg.Resolution.DNS {
			t.Error("DNS resolution should be enabled by default")
		}
		if !cfg.Resolution.NetBIOS {
			t.Error("NetBIOS resolution should be enabled by default")
		}
		if !cfg.Resolution.MDNS {
			t.Error("MDNS resolution should be enabled by default")
		}
	})

	// Test persistence defaults
	t.Run("persistence", func(t *testing.T) {
		if !cfg.Persistence.StoreHistory {
			t.Error("StoreHistory should be enabled by default")
		}
		if cfg.Persistence.StalenessThreshold <= 0 {
			t.Error("StalenessThreshold should be positive")
		}
	})
}

func TestPipelineConfig_CustomValues(t *testing.T) {
	cfg := discovery.PipelineConfig{
		Phases: discovery.PipelinePhaseConfig{
			Enumeration:      true,
			NameResolution:   false,
			ServiceDiscovery: true,
			VulnAssessment:   true,
		},
		Timing: discovery.PipelineTiming{
			ProbeDelay:         200 * time.Millisecond,
			HostDelay:          100 * time.Millisecond,
			MaxConcurrentHosts: 50,
			PhaseTimeout:       30 * time.Minute,
			Profile:            discovery.ScanProfileAggressive,
		},
		PortScan: discovery.PipelinePortScanConfig{
			Intensity:      discovery.PortScanComprehensive,
			CustomPorts:    []int{22, 80, 443, 8080},
			BannerGrab:     true,
			ConnectTimeout: 5 * time.Second,
		},
		SNMPCollection: discovery.SNMPCollectionConfig{
			Enabled:           true,
			WalkTimeout:       60 * time.Second,
			MaxOIDsPerRequest: 20,
			MIBs: discovery.SNMPMIBSelection{
				System:      true,
				Interfaces:  true,
				IPAddresses: true,
				Routing:     true,
				Bridge:      true,
				Entity:      true,
				LLDP:        true,
				VLAN:        true,
			},
		},
		Resolution: discovery.PipelineResolutionConfig{
			DNS:     true,
			NetBIOS: false,
			MDNS:    true,
		},
		Persistence: discovery.PipelinePersistenceConfig{
			StoreHistory:       true,
			StalenessThreshold: 48 * time.Hour,
			PurgeAfter:         168 * time.Hour,
		},
	}

	if !cfg.Phases.ServiceDiscovery {
		t.Error("ServiceDiscovery should be enabled")
	}
	if cfg.Timing.MaxConcurrentHosts != 50 {
		t.Errorf("MaxConcurrentHosts should be 50, got %d", cfg.Timing.MaxConcurrentHosts)
	}
	if cfg.PortScan.Intensity != discovery.PortScanComprehensive {
		t.Errorf("PortScan.Intensity should be comprehensive, got %q", cfg.PortScan.Intensity)
	}
	if len(cfg.PortScan.CustomPorts) != 4 {
		t.Errorf("CustomPorts should have 4 entries, got %d", len(cfg.PortScan.CustomPorts))
	}
	if cfg.SNMPCollection.MaxOIDsPerRequest != 20 {
		t.Errorf("SNMP MaxOIDsPerRequest should be 20, got %d", cfg.SNMPCollection.MaxOIDsPerRequest)
	}
	if cfg.Resolution.NetBIOS {
		t.Error("NetBIOS resolution should be disabled")
	}
}

func TestPortLists(t *testing.T) {
	t.Run("quick_ports_minimal", func(t *testing.T) {
		ports := discovery.GetQuickPorts()
		if len(ports) < 3 {
			t.Errorf("Quick ports should have at least 3 ports, got %d", len(ports))
		}
		if len(ports) > 20 {
			t.Errorf("Quick ports should have at most 20 ports, got %d", len(ports))
		}
	})

	t.Run("standard_ports_comprehensive", func(t *testing.T) {
		ports := discovery.GetStandardPorts()
		if len(ports) < 30 {
			t.Errorf("Standard ports should have at least 30 ports, got %d", len(ports))
		}
		if len(ports) > 200 {
			t.Errorf("Standard ports should have at most 200 ports, got %d", len(ports))
		}
	})

	t.Run("comprehensive_ports_extensive", func(t *testing.T) {
		ports := discovery.GetComprehensivePorts()
		if len(ports) < 500 {
			t.Errorf("Comprehensive ports should have at least 500 ports, got %d", len(ports))
		}
	})

	t.Run("quick_subset_of_standard", func(t *testing.T) {
		quickPorts := discovery.GetQuickPorts()
		stdPorts := discovery.GetStandardPorts()

		stdPortsMap := make(map[int]bool)
		for _, p := range stdPorts {
			stdPortsMap[p] = true
		}

		// Most quick ports should be in standard ports
		matchCount := 0
		for _, p := range quickPorts {
			if stdPortsMap[p] {
				matchCount++
			}
		}
		if matchCount < len(quickPorts)/2 {
			t.Errorf(
				"Most quick ports should be in standard ports, only %d of %d matched",
				matchCount,
				len(quickPorts),
			)
		}
	})
}

func TestScanTimingPresets_Profiles(t *testing.T) {
	presets := discovery.GetScanTimingPresets()

	profiles := []discovery.ScanTimingProfile{
		discovery.ScanProfilePolite,
		discovery.ScanProfileNormal,
		discovery.ScanProfileAggressive,
	}

	for _, profile := range profiles {
		t.Run(string(profile), func(t *testing.T) {
			preset, ok := presets[profile]
			if !ok {
				t.Fatalf("Missing preset for profile %q", profile)
			}

			if preset.ProbeDelay < 0 {
				t.Errorf("ProbeDelay should be non-negative for %q", profile)
			}
			if preset.HostDelay < 0 {
				t.Errorf("HostDelay should be non-negative for %q", profile)
			}
			if preset.MaxConcurrentHosts <= 0 {
				t.Errorf("MaxConcurrentHosts should be positive for %q", profile)
			}
			if preset.PhaseTimeout <= 0 {
				t.Errorf("PhaseTimeout should be positive for %q", profile)
			}
		})
	}
}

func TestScanTimingPresets_Ordering(t *testing.T) {
	presets := discovery.GetScanTimingPresets()

	polite := presets[discovery.ScanProfilePolite]
	normal := presets[discovery.ScanProfileNormal]
	aggressive := presets[discovery.ScanProfileAggressive]

	// Polite should have smaller concurrency
	if polite.MaxConcurrentHosts >= normal.MaxConcurrentHosts {
		t.Error("Polite should have fewer concurrent hosts than normal")
	}
	if normal.MaxConcurrentHosts >= aggressive.MaxConcurrentHosts {
		t.Error("Normal should have fewer concurrent hosts than aggressive")
	}

	// Aggressive should have shorter delays
	if aggressive.ProbeDelay >= normal.ProbeDelay {
		t.Error("Aggressive should have shorter probe delay than normal")
	}
	if aggressive.HostDelay >= normal.HostDelay {
		t.Error("Aggressive should have shorter host delay than normal")
	}
}

func TestPipelineRun_AllFields(t *testing.T) {
	now := time.Now()
	completed := now.Add(10 * time.Minute)

	run := discovery.PipelineRun{
		ID:           "run-test-123",
		StartedAt:    now,
		CompletedAt:  &completed,
		Status:       discovery.PipelineStateComplete,
		Trigger:      "manual",
		CurrentPhase: "complete",
		PhaseDurations: map[string]time.Duration{
			"enumeration": 2 * time.Minute,
			"resolution":  3 * time.Minute,
			"scanning":    5 * time.Minute,
		},
		DevicesFound: 100,
		Errors:       []string{"warning: timeout on one device"},
	}

	if run.ID != "run-test-123" {
		t.Errorf("ID mismatch: got %q", run.ID)
	}
	if run.Status != discovery.PipelineStateComplete {
		t.Errorf("Status should be complete, got %q", run.Status)
	}
	if len(run.PhaseDurations) != 3 {
		t.Errorf("PhaseDurations should have 3 entries, got %d", len(run.PhaseDurations))
	}
	if run.DevicesFound != 100 {
		t.Errorf("DevicesFound should be 100, got %d", run.DevicesFound)
	}
	if len(run.Errors) != 1 {
		t.Errorf("Errors should have 1 entry, got %d", len(run.Errors))
	}

	// Verify duration calculation
	enumDur := run.PhaseDurations["enumeration"]
	if enumDur != 2*time.Minute {
		t.Errorf("enumeration duration should be 2m, got %v", enumDur)
	}
}

func TestPipelineEvent_Types(t *testing.T) {
	eventTypes := []discovery.PipelineEventType{
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

	for _, eventType := range eventTypes {
		event := discovery.PipelineEvent{
			Type:      eventType,
			Timestamp: time.Now(),
			RunID:     "test-run",
			Payload:   map[string]any{"test": "data"},
		}

		if event.Type != eventType {
			t.Errorf("Event type mismatch: expected %q, got %q", eventType, event.Type)
		}
		if event.RunID != "test-run" {
			t.Errorf("RunID should be 'test-run', got %q", event.RunID)
		}
	}
}

func TestPipelineStartedPayload(t *testing.T) {
	payload := discovery.PipelineStartedPayload{
		TotalPhases: 4,
		Phases:      []string{"enumeration", "resolution", "scanning", "assessment"},
	}

	if payload.TotalPhases != 4 {
		t.Errorf("TotalPhases should be 4, got %d", payload.TotalPhases)
	}
	if len(payload.Phases) != 4 {
		t.Errorf("Phases should have 4 entries, got %d", len(payload.Phases))
	}
	if payload.Phases[0] != "enumeration" {
		t.Errorf("First phase should be 'enumeration', got %q", payload.Phases[0])
	}
}

func TestPipelineFailedPayload(t *testing.T) {
	// Test error scenarios
	payloads := []discovery.PhaseCompletedPayload{
		{
			Phase:    "enumeration",
			Duration: 30 * time.Second,
			Errors:   []string{"network timeout", "permission denied"},
		},
		{
			Phase:         "resolution",
			NamesResolved: 0,
			Duration:      10 * time.Second,
			Errors:        []string{"DNS server unreachable"},
		},
	}

	for _, p := range payloads {
		if len(p.Errors) == 0 {
			t.Errorf("Expected errors for phase %q", p.Phase)
		}
	}
}

func TestSNMPMIBSelection_AllFields(t *testing.T) {
	// Test all MIBs enabled
	allEnabled := discovery.SNMPMIBSelection{
		System:      true,
		Interfaces:  true,
		IPAddresses: true,
		Routing:     true,
		Bridge:      true,
		Entity:      true,
		LLDP:        true,
		VLAN:        true,
	}

	fields := []bool{
		allEnabled.System,
		allEnabled.Interfaces,
		allEnabled.IPAddresses,
		allEnabled.Routing,
		allEnabled.Bridge,
		allEnabled.Entity,
		allEnabled.LLDP,
		allEnabled.VLAN,
	}

	enabledCount := 0
	for _, f := range fields {
		if f {
			enabledCount++
		}
	}

	if enabledCount != 8 {
		t.Errorf("Expected all 8 MIBs enabled, got %d", enabledCount)
	}

	// Test minimal MIBs
	minimal := discovery.SNMPMIBSelection{
		System:     true,
		Interfaces: true,
	}

	if !minimal.System || !minimal.Interfaces {
		t.Error("Minimal config should have System and Interfaces enabled")
	}
	if minimal.Bridge || minimal.Entity {
		t.Error("Minimal config should not have Bridge or Entity enabled")
	}
}

func TestSNMPCollectionConfig_Fields(t *testing.T) {
	cfg := discovery.SNMPCollectionConfig{
		Enabled:           true,
		WalkTimeout:       45 * time.Second,
		MaxOIDsPerRequest: 15,
		MIBs: discovery.SNMPMIBSelection{
			System:     true,
			Interfaces: true,
		},
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.WalkTimeout != 45*time.Second {
		t.Errorf("WalkTimeout should be 45s, got %v", cfg.WalkTimeout)
	}
	if cfg.MaxOIDsPerRequest != 15 {
		t.Errorf("MaxOIDsPerRequest should be 15, got %d", cfg.MaxOIDsPerRequest)
	}
}

func TestPipelinePersistenceConfig_Fields(t *testing.T) {
	cfg := discovery.PipelinePersistenceConfig{
		StoreHistory:       true,
		StalenessThreshold: 12 * time.Hour,
		PurgeAfter:         7 * 24 * time.Hour, // 7 days
	}

	if !cfg.StoreHistory {
		t.Error("StoreHistory should be true")
	}
	if cfg.StalenessThreshold != 12*time.Hour {
		t.Errorf("StalenessThreshold should be 12h, got %v", cfg.StalenessThreshold)
	}
	if cfg.PurgeAfter != 168*time.Hour {
		t.Errorf("PurgeAfter should be 168h (7 days), got %v", cfg.PurgeAfter)
	}
}

func TestPipelinePortScanConfig_Fields(t *testing.T) {
	tests := []struct {
		name      string
		intensity discovery.PortScanIntensity
	}{
		{"off", discovery.PortScanOff},
		{"quick", discovery.PortScanQuick},
		{"standard", discovery.PortScanStandard},
		{"comprehensive", discovery.PortScanComprehensive},
		{"custom", discovery.PortScanCustom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := discovery.PipelinePortScanConfig{
				Intensity:      tt.intensity,
				CustomPorts:    []int{22, 80, 443},
				BannerGrab:     true,
				ConnectTimeout: 3 * time.Second,
			}

			if cfg.Intensity != tt.intensity {
				t.Errorf("Intensity mismatch: expected %q, got %q", tt.intensity, cfg.Intensity)
			}
		})
	}
}
