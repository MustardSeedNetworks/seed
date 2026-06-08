package vuln_test

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
)

func newTestRegistry() *discovery.DeviceRegistry {
	return discovery.NewDeviceRegistry(
		discovery.NewEventBus(&discovery.EventBusConfig{}),
		&discovery.RegistryConfig{EmitEvents: false},
	)
}

// TestStageNilScannerIsNoop verifies the assess stage with a nil scanner is a
// safe no-op (ADR-0018): no panic, no vulnerabilities recorded.
func TestStageNilScannerIsNoop(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	reg.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "de:ad:be:ef:00:02", IP: "10.0.0.2"})
	stage := vuln.NewStage(nil, reg, discovery.NewEventBus(&discovery.EventBusConfig{}))
	stats := &discovery.ScanStats{}

	stage.Assess(context.Background(), stats)

	if stats.VulnerableDevices != 0 {
		t.Fatalf("VulnerableDevices = %d, want 0 with nil scanner", stats.VulnerableDevices)
	}
	if d := reg.GetDevice("de:ad:be:ef:00:02"); d == nil || d.Vulnerabilities != nil {
		t.Fatalf("device unexpectedly assessed: %+v", d)
	}
}
