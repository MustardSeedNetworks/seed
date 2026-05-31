package api

import "testing"

func TestInitSNMPPoller_RegistersEngine(t *testing.T) {
	services := NewServiceContainer()
	initSNMPPoller(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	if !names["snmp-poller"] {
		t.Errorf("snmp-poller not registered; got engines = %v", names)
	}
}

// TestInitDatabaseDependentServices_RegistersEverythingIncludingPoller
// verifies the full server init lands all 9 expected engines in the
// registry: probe, retention, 4 topology reconcilers, 2 alert
// pipelines, and the snmp poller.
func TestInitDatabaseDependentServices_RegistersEverythingIncludingPoller(t *testing.T) {
	services := NewServiceContainer()
	initDatabaseDependentServices(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	want := []string{
		"probe",
		"retention",
		"topology-sysinfo-reconciler",
		"topology-iftable-reconciler",
		"topology-edge-reconciler",
		"topology-arp-reconciler",
		"alert-listener-pipeline",
		"alert-observation-pipeline",
		"snmp-poller",
	}
	for _, n := range want {
		if !names[n] {
			t.Errorf("engine %q not registered after full init; got %v", n, names)
		}
	}
}
