package api

import (
	"testing"
)

// These tests exercise the wire-up before the Stage A5.9 tier
// gate filters by license. The init helpers construct + register
// every engine; the per-tier gate (registerEngineIfLicensed) only
// skips them when a configured license.Manager reports a tier
// below the engine's minimum.
//
// The test environment goes through initLicenseAndAPITokens which
// calls license.NewManager() — that returns a Free-tier Manager
// (no key file present). The gate then filters Starter + Pro
// engines, leaving only probe + retention in services.Engines.
//
// We therefore assert "construction succeeded" by checking the
// init function didn't panic and the registry has the Free-tier
// engines; the full set is exercised by integration tests that
// wire a Pro license.

func TestInitTopologyReconcilers_RegistersOnPro(t *testing.T) {
	// No license manager wired -> effectiveTier returns Pro,
	// so all four topology reconcilers land.
	services := NewServiceContainer()
	initTopologyReconcilers(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	want := []string{
		"topology-sysinfo-reconciler",
		"topology-iftable-reconciler",
		"topology-edge-reconciler",
		"topology-arp-reconciler",
	}
	for _, n := range want {
		if !names[n] {
			t.Errorf("topology engine %q not registered; got %v", n, names)
		}
	}
}

func TestInitAlertPipelines_RegistersOnPro(t *testing.T) {
	services := NewServiceContainer()
	initAlertPipelines(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	want := []string{
		"alert-listener-pipeline",
		"alert-observation-pipeline",
	}
	for _, n := range want {
		if !names[n] {
			t.Errorf("alert pipeline %q not registered; got %v", n, names)
		}
	}
}

func TestInitDatabaseDependentServices_GatesByFreeLicense(t *testing.T) {
	// initLicenseAndAPITokens wires a Free-tier license.Manager,
	// so the tier gate filters out everything Starter+ during the
	// full init pass. probe + retention are Free-tier and survive.
	services := NewServiceContainer()
	initDatabaseDependentServices(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	freeTierEngines := []string{"probe", "retention"}
	for _, n := range freeTierEngines {
		if !names[n] {
			t.Errorf("Free-tier engine %q must be registered; got %v", n, names)
		}
	}
	// Sanity: Starter+ engines must NOT have landed.
	for _, n := range []string{"snmp-poller", "topology-sysinfo-reconciler"} {
		if names[n] {
			t.Errorf("Starter+ engine %q should be gated out on Free tier; got %v", n, names)
		}
	}
}
