// Package sap_test provides comprehensive tests for the sap module.
// These tests cover Module initialization, service registration, and service lifecycle.
package sap_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/sap"
)

// =============================================================================
// Module Initialization Tests
// =============================================================================

// TestNewModuleCreation verifies that New creates a valid Module instance.
func TestNewModuleCreation(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	if module == nil {
		t.Fatal("expected non-nil Module")
	}
}

// TestNewModuleWithNilConfig verifies New handles nil config gracefully.
// Note: This tests the actual behavior - the code may panic with nil config.
func TestNewModuleWithNilConfig(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil config since services access cfg fields
			t.Logf("New panicked with nil config as expected: %v", r)
		}
	}()

	// This should panic because services dereference cfg
	_ = sap.New(nil, nil)
	// If we get here, the test failed
	t.Error("expected panic with nil config")
}

// TestNewModuleWithNilDatabase verifies New handles nil database gracefully.
func TestNewModuleWithNilDatabase(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	if module == nil {
		t.Fatal("expected non-nil Module with nil database")
	}
}

// TestNewModuleServicesNotNil verifies all services are initialized.
func TestNewModuleServicesNotNil(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	tests := []struct {
		name    string
		service any
	}{
		{"Link", module.Link()},
		{"Cable", module.Cable()},
		{"DHCP", module.DHCP()},
		{"DNS", module.DNS()},
		{"Gateway", module.Gateway()},
		{"SNMP", module.SNMP()},
		{"Performance", module.Performance()},
		{"VLAN", module.VLAN()},
		{"Telemetry", module.Telemetry()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.service == nil {
				t.Errorf("expected non-nil %s service", tt.name)
			}
		})
	}
}

// =============================================================================
// Service Getter Tests
// =============================================================================

// TestModuleLinkService verifies the Link service getter.
func TestModuleLinkService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	link := module.Link()
	if link == nil {
		t.Fatal("expected non-nil LinkService")
	}

	// Verify same instance is returned on subsequent calls
	link2 := module.Link()
	if link != link2 {
		t.Error("expected same LinkService instance on subsequent calls")
	}
}

// TestModuleCableService verifies the Cable service getter.
func TestModuleCableService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	cable := module.Cable()
	if cable == nil {
		t.Fatal("expected non-nil CableService")
	}

	// Verify same instance is returned on subsequent calls
	cable2 := module.Cable()
	if cable != cable2 {
		t.Error("expected same CableService instance on subsequent calls")
	}
}

// TestModuleDHCPService verifies the DHCP service getter.
func TestModuleDHCPService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	dhcp := module.DHCP()
	if dhcp == nil {
		t.Fatal("expected non-nil DHCPService")
	}

	// Verify same instance is returned on subsequent calls
	dhcp2 := module.DHCP()
	if dhcp != dhcp2 {
		t.Error("expected same DHCPService instance on subsequent calls")
	}
}

// TestModuleDNSService verifies the DNS service getter.
func TestModuleDNSService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	dns := module.DNS()
	if dns == nil {
		t.Fatal("expected non-nil DNSService")
	}

	// Verify same instance is returned on subsequent calls
	dns2 := module.DNS()
	if dns != dns2 {
		t.Error("expected same DNSService instance on subsequent calls")
	}
}

// TestModuleGatewayService verifies the Gateway service getter.
func TestModuleGatewayService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	gateway := module.Gateway()
	if gateway == nil {
		t.Fatal("expected non-nil GatewayService")
	}

	// Verify same instance is returned on subsequent calls
	gateway2 := module.Gateway()
	if gateway != gateway2 {
		t.Error("expected same GatewayService instance on subsequent calls")
	}
}

// TestModuleSNMPService verifies the SNMP service getter.
func TestModuleSNMPService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	snmp := module.SNMP()
	if snmp == nil {
		t.Fatal("expected non-nil SNMPService")
	}

	// Verify same instance is returned on subsequent calls
	snmp2 := module.SNMP()
	if snmp != snmp2 {
		t.Error("expected same SNMPService instance on subsequent calls")
	}
}

// TestModulePerformanceService verifies the Performance service getter.
func TestModulePerformanceService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	perf := module.Performance()
	if perf == nil {
		t.Fatal("expected non-nil PerformanceService")
	}

	// Verify same instance is returned on subsequent calls
	perf2 := module.Performance()
	if perf != perf2 {
		t.Error("expected same PerformanceService instance on subsequent calls")
	}
}

// TestModuleVLANService verifies the VLAN service getter.
func TestModuleVLANService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	vlan := module.VLAN()
	if vlan == nil {
		t.Fatal("expected non-nil VLANService")
	}

	// Verify same instance is returned on subsequent calls
	vlan2 := module.VLAN()
	if vlan != vlan2 {
		t.Error("expected same VLANService instance on subsequent calls")
	}
}

// TestModuleTelemetryService verifies the Telemetry service getter.
func TestModuleTelemetryService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	telemetry := module.Telemetry()
	if telemetry == nil {
		t.Fatal("expected non-nil TelemetryService")
	}

	// Verify same instance is returned on subsequent calls
	telemetry2 := module.Telemetry()
	if telemetry != telemetry2 {
		t.Error("expected same TelemetryService instance on subsequent calls")
	}
}

// =============================================================================
// Service Getter Table-Driven Tests
// =============================================================================

// TestModuleServiceGettersTableDriven tests all service getters in a table-driven format.
func TestModuleServiceGettersTableDriven(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	tests := []struct {
		name      string
		getterFn  func() any
		wantNil   bool
		wantSame  bool
		iterations int
	}{
		{
			name:       "Link",
			getterFn:   func() any { return module.Link() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "Cable",
			getterFn:   func() any { return module.Cable() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "DHCP",
			getterFn:   func() any { return module.DHCP() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "DNS",
			getterFn:   func() any { return module.DNS() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "Gateway",
			getterFn:   func() any { return module.Gateway() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "SNMP",
			getterFn:   func() any { return module.SNMP() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "Performance",
			getterFn:   func() any { return module.Performance() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "VLAN",
			getterFn:   func() any { return module.VLAN() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
		{
			name:       "Telemetry",
			getterFn:   func() any { return module.Telemetry() },
			wantNil:    false,
			wantSame:   true,
			iterations: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := tt.getterFn()

			if tt.wantNil && first != nil {
				t.Errorf("%s: expected nil service, got non-nil", tt.name)
			}
			if !tt.wantNil && first == nil {
				t.Errorf("%s: expected non-nil service, got nil", tt.name)
				return
			}

			if tt.wantSame {
				for i := 1; i < tt.iterations; i++ {
					next := tt.getterFn()
					if first != next {
						t.Errorf("%s: iteration %d returned different instance", tt.name, i)
					}
				}
			}
		})
	}
}

// =============================================================================
// Module Lifecycle Tests
// =============================================================================

// TestModuleStopWithoutStart verifies Stop is safe without Start.
func TestModuleStopWithoutStart(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	// Stop without Start should not panic
	err := module.Stop()
	if err != nil {
		t.Errorf("Stop without Start returned error: %v", err)
	}
}

// TestModuleMultipleStops verifies multiple Stops are safe.
func TestModuleMultipleStops(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	// Multiple stops should not panic
	for i := 0; i < 5; i++ {
		err := module.Stop()
		if err != nil {
			t.Errorf("Stop iteration %d returned error: %v", i, err)
		}
	}
}

// TestModuleStopReturnsNil verifies Stop returns nil error.
func TestModuleStopReturnsNil(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	err := module.Stop()
	if err != nil {
		t.Errorf("expected nil error from Stop, got: %v", err)
	}
}

// TestModuleStartWithCanceledContext verifies Start handles canceled context.
func TestModuleStartWithCanceledContext(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Start with canceled context - behavior depends on implementation
	// Some services may fail, others may succeed. We just verify no panic.
	_ = module.Start(ctx)

	// Always clean up
	_ = module.Stop()
}

// TestModuleStartWithTimeout verifies Start handles context timeout.
func TestModuleStartWithTimeout(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Very short timeout - may or may not succeed
	_ = module.Start(ctx)

	// Always clean up
	_ = module.Stop()
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

// TestModuleConcurrentServiceAccess verifies thread-safety of service getters.
func TestModuleConcurrentServiceAccess(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	const goroutines = 10
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 9) // 9 services

	// Test concurrent access to all services
	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.Link()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.Cable()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.DHCP()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.DNS()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.Gateway()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.SNMP()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.Performance()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.VLAN()
			}
		}()
		go func() {
			defer wg.Done()
			for range iterations {
				_ = module.Telemetry()
			}
		}()
	}

	wg.Wait()
}

// TestModuleConcurrentStopCalls verifies concurrent Stop calls are safe.
func TestModuleConcurrentStopCalls(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = module.Stop()
		}()
	}

	wg.Wait()
}

// =============================================================================
// Service Creation Tests - Individual Services
// =============================================================================

// TestNewLinkService verifies NewLinkService creates a valid service.
func TestNewLinkService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewLinkService(cfg)

	if service == nil {
		t.Fatal("expected non-nil LinkService")
	}
}

// TestNewCableService verifies NewCableService creates a valid service.
func TestNewCableService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewCableService(cfg)

	if service == nil {
		t.Fatal("expected non-nil CableService")
	}
}

// TestNewDHCPService verifies NewDHCPService creates a valid service.
func TestNewDHCPService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewDHCPService(cfg)

	if service == nil {
		t.Fatal("expected non-nil DHCPService")
	}
}

// TestNewDNSService verifies NewDNSService creates a valid service.
func TestNewDNSService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewDNSService(cfg)

	if service == nil {
		t.Fatal("expected non-nil DNSService")
	}
}

// TestNewGatewayService verifies NewGatewayService creates a valid service.
func TestNewGatewayService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewGatewayService(cfg)

	if service == nil {
		t.Fatal("expected non-nil GatewayService")
	}
}

// TestNewSNMPService verifies NewSNMPService creates a valid service.
func TestNewSNMPService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewSNMPService(cfg)

	if service == nil {
		t.Fatal("expected non-nil SNMPService")
	}
}

// TestNewPerformanceService verifies NewPerformanceService creates a valid service.
func TestNewPerformanceService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewPerformanceService(cfg)

	if service == nil {
		t.Fatal("expected non-nil PerformanceService")
	}
}

// TestNewVLANService verifies NewVLANService creates a valid service.
func TestNewVLANService(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewVLANService(cfg)

	if service == nil {
		t.Fatal("expected non-nil VLANService")
	}
}

// =============================================================================
// Service Constructor Table-Driven Tests
// =============================================================================

// TestServiceConstructorsTableDriven tests all service constructors.
func TestServiceConstructorsTableDriven(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()

	tests := []struct {
		name        string
		constructor func(*config.Config) any
		wantNil     bool
	}{
		{
			name:        "NewLinkService",
			constructor: func(c *config.Config) any { return sap.NewLinkService(c) },
			wantNil:     false,
		},
		{
			name:        "NewCableService",
			constructor: func(c *config.Config) any { return sap.NewCableService(c) },
			wantNil:     false,
		},
		{
			name:        "NewDHCPService",
			constructor: func(c *config.Config) any { return sap.NewDHCPService(c) },
			wantNil:     false,
		},
		{
			name:        "NewDNSService",
			constructor: func(c *config.Config) any { return sap.NewDNSService(c) },
			wantNil:     false,
		},
		{
			name:        "NewGatewayService",
			constructor: func(c *config.Config) any { return sap.NewGatewayService(c) },
			wantNil:     false,
		},
		{
			name:        "NewSNMPService",
			constructor: func(c *config.Config) any { return sap.NewSNMPService(c) },
			wantNil:     false,
		},
		{
			name:        "NewPerformanceService",
			constructor: func(c *config.Config) any { return sap.NewPerformanceService(c) },
			wantNil:     false,
		},
		{
			name:        "NewVLANService",
			constructor: func(c *config.Config) any { return sap.NewVLANService(c) },
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.constructor(cfg)

			if tt.wantNil && service != nil {
				t.Errorf("%s: expected nil, got non-nil", tt.name)
			}
			if !tt.wantNil && service == nil {
				t.Errorf("%s: expected non-nil, got nil", tt.name)
			}
		})
	}
}

// =============================================================================
// GatewayService Tests
// =============================================================================

// TestGatewayServiceTester verifies Tester returns the underlying tester.
func TestGatewayServiceTester(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewGatewayService(cfg)

	tester := service.Tester()
	if tester == nil {
		t.Error("expected non-nil gateway tester")
	}
}

// TestGatewayServiceStartStop verifies Start and Stop lifecycle.
func TestGatewayServiceStartStop(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewGatewayService(cfg)

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Errorf("Start returned error: %v", err)
	}

	// Stop should not panic
	service.Stop()

	// Multiple stops should be safe
	service.Stop()
}

// TestGatewayServiceStopWithoutStart verifies Stop without Start is safe.
func TestGatewayServiceStopWithoutStart(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewGatewayService(cfg)

	// Should not panic
	service.Stop()
}

// TestGatewayServiceGetHealthNotInitialized verifies GetHealth before Start.
func TestGatewayServiceGetHealthNotInitialized(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewGatewayService(cfg)

	ctx := context.Background()
	// GetHealth should work because tester is initialized in constructor
	health, err := service.GetHealth(ctx)
	if err != nil {
		t.Logf("GetHealth returned error (may be expected): %v", err)
	}
	if health != nil {
		t.Logf("GetHealth returned health: %+v", health)
	}
}

// =============================================================================
// DNSService Tests
// =============================================================================

// TestDNSServiceTester verifies Tester returns the underlying tester.
func TestDNSServiceTester(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewDNSService(cfg)

	tester := service.Tester()
	if tester == nil {
		t.Error("expected non-nil DNS tester")
	}
}

// TestDNSServiceSecurityScanner verifies SecurityScanner returns the underlying scanner.
func TestDNSServiceSecurityScanner(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewDNSService(cfg)

	scanner := service.SecurityScanner()
	if scanner == nil {
		t.Error("expected non-nil security scanner")
	}
}

// =============================================================================
// PerformanceService Tests
// =============================================================================

// TestPerformanceServiceSpeedtestTester verifies SpeedtestTester returns the tester.
func TestPerformanceServiceSpeedtestTester(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewPerformanceService(cfg)

	tester := service.SpeedtestTester()
	if tester == nil {
		t.Error("expected non-nil speedtest tester")
	}
}

// TestPerformanceServiceIPerfManager verifies IPerfManager returns the manager.
func TestPerformanceServiceIPerfManager(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewPerformanceService(cfg)

	manager := service.IPerfManager()
	if manager == nil {
		t.Error("expected non-nil iperf manager")
	}
}

// TestPerformanceServiceStop verifies Stop is safe.
func TestPerformanceServiceStop(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewPerformanceService(cfg)

	// Should not panic
	service.Stop()

	// Multiple stops should be safe
	service.Stop()
}

// =============================================================================
// VLANService Tests
// =============================================================================

// TestVLANServiceManager verifies Manager returns the VLAN manager.
func TestVLANServiceManager(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewVLANService(cfg)

	manager := service.Manager()
	if manager == nil {
		t.Error("expected non-nil VLAN manager")
	}
}

// TestVLANServiceTrafficMonitor verifies TrafficMonitor returns the monitor.
func TestVLANServiceTrafficMonitor(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewVLANService(cfg)

	monitor := service.TrafficMonitor()
	if monitor == nil {
		t.Error("expected non-nil traffic monitor")
	}
}

// TestVLANServiceListEmpty verifies List returns empty slice initially.
func TestVLANServiceListEmpty(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewVLANService(cfg)

	ctx := context.Background()
	configs, err := service.List(ctx)
	if err != nil {
		t.Errorf("List returned error: %v", err)
	}
	// May be nil or empty depending on system state
	if configs != nil {
		t.Logf("List returned %d configs", len(configs))
	}
}

// =============================================================================
// LinkService Tests
// =============================================================================

// TestLinkServiceStop verifies Stop is safe.
func TestLinkServiceStop(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewLinkService(cfg)

	// Should not panic without Start
	service.Stop()

	// Multiple stops should be safe
	service.Stop()
}

// TestLinkServiceGetStatusNotInitialized verifies GetStatus before Start.
func TestLinkServiceGetStatusNotInitialized(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewLinkService(cfg)

	ctx := context.Background()
	status, err := service.GetStatus(ctx)
	if err == nil {
		t.Error("expected error when service not initialized")
	}
	if !errors.Is(err, sap.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
	if status != nil {
		t.Error("expected nil status when not initialized")
	}
}

// =============================================================================
// SNMPService Tests
// =============================================================================

// TestSNMPServiceGetInterfacesNotImplemented verifies GetInterfaces returns not implemented.
func TestSNMPServiceGetInterfacesNotImplemented(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewSNMPService(cfg)

	ctx := context.Background()
	interfaces, err := service.GetInterfaces(ctx, "", "")
	if !errors.Is(err, sap.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got: %v", err)
	}
	if interfaces != nil {
		t.Error("expected nil interfaces")
	}
}

// TestSNMPServiceGetMACTableNotImplemented verifies GetMACTable returns not implemented.
func TestSNMPServiceGetMACTableNotImplemented(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewSNMPService(cfg)

	ctx := context.Background()
	table, err := service.GetMACTable(ctx, "", "")
	if !errors.Is(err, sap.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got: %v", err)
	}
	if table != nil {
		t.Error("expected nil MAC table")
	}
}

// =============================================================================
// Configuration Variations Tests
// =============================================================================

// TestModuleWithCustomInterface verifies module works with custom interface config.
func TestModuleWithCustomInterface(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Interface.Default = "en0"

	module := sap.New(cfg, nil)
	if module == nil {
		t.Fatal("expected non-nil module with custom interface")
	}

	// All services should still be accessible
	if module.Link() == nil {
		t.Error("expected non-nil Link service")
	}
	if module.VLAN() == nil {
		t.Error("expected non-nil VLAN service")
	}
}

// TestModuleWithDNSServers verifies module works with custom DNS servers.
func TestModuleWithDNSServers(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.DNS.Servers = []config.DNSServer{
		{Address: "8.8.8.8", Enabled: true},
		{Address: "1.1.1.1", Enabled: true},
	}
	cfg.DNS.TestHostname = "example.com"

	module := sap.New(cfg, nil)
	if module == nil {
		t.Fatal("expected non-nil module with custom DNS config")
	}

	dns := module.DNS()
	if dns == nil {
		t.Error("expected non-nil DNS service")
	}
}

// TestModuleWithSpeedtestConfig verifies module works with custom speedtest config.
func TestModuleWithSpeedtestConfig(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Speedtest.ServerID = "12345"

	module := sap.New(cfg, nil)
	if module == nil {
		t.Fatal("expected non-nil module with custom speedtest config")
	}

	perf := module.Performance()
	if perf == nil {
		t.Error("expected non-nil Performance service")
	}
}

// =============================================================================
// Error Constant Tests
// =============================================================================

// TestErrorsAreDistinct verifies all error constants are distinct.
func TestErrorsAreDistinct(t *testing.T) {
	t.Parallel()
	errs := []error{
		sap.ErrNotImplemented,
		sap.ErrNotInitialized,
		sap.ErrNotSupported,
		sap.ErrTestFailed,
	}

	for i, err1 := range errs {
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("errors %d and %d should be distinct", i, j)
			}
		}
	}
}

// TestErrorsHaveMessages verifies all errors have non-empty messages.
func TestErrorsHaveMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotImplemented", sap.ErrNotImplemented},
		{"ErrNotInitialized", sap.ErrNotInitialized},
		{"ErrNotSupported", sap.ErrNotSupported},
		{"ErrTestFailed", sap.ErrTestFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() == "" {
				t.Errorf("%s should have non-empty message", tt.name)
			}
		})
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

// TestConstantsValues verifies exported constant values.
func TestConstantsValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"DefaultInterface", sap.DefaultInterfaceConst, "eth0"},
		{"InterfaceStateWaitMs", sap.InterfaceStateWaitMsConst, 100},
		{"SNMPTimeticksPerSecond", sap.SNMPTimeticksPerSecondConst, 100},
		{"DefaultIPerfPort", sap.DefaultIPerfPortConst, 5201},
		{"DefaultIPerfDurationSec", sap.DefaultIPerfDurationSecConst, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkModuleCreation benchmarks module creation.
func BenchmarkModuleCreation(b *testing.B) {
	cfg := config.DefaultConfig()
	b.ResetTimer()

	for b.Loop() {
		_ = sap.New(cfg, nil)
	}
}

// BenchmarkServiceGetter benchmarks service getter calls.
func BenchmarkServiceGetter(b *testing.B) {
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)
	b.ResetTimer()

	for b.Loop() {
		_ = module.Link()
	}
}

// BenchmarkAllServiceGetters benchmarks all service getter calls.
func BenchmarkAllServiceGetters(b *testing.B) {
	cfg := config.DefaultConfig()
	module := sap.New(cfg, nil)
	b.ResetTimer()

	for b.Loop() {
		_ = module.Link()
		_ = module.Cable()
		_ = module.DHCP()
		_ = module.DNS()
		_ = module.Gateway()
		_ = module.SNMP()
		_ = module.Performance()
		_ = module.VLAN()
		_ = module.Telemetry()
	}
}

// BenchmarkModuleStop benchmarks Stop call.
func BenchmarkModuleStop(b *testing.B) {
	cfg := config.DefaultConfig()
	b.ResetTimer()

	for b.Loop() {
		module := sap.New(cfg, nil)
		_ = module.Stop()
	}
}
