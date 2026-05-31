package api

import (
	"testing"
)

func TestInitTopologyReconcilers_RegistersFourEngines(t *testing.T) {
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

func TestInitAlertPipelines_RegistersBothPipelines(t *testing.T) {
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

func TestInitDatabaseDependentServices_RegistersAllStageAEngines(t *testing.T) {
	services := NewServiceContainer()
	initDatabaseDependentServices(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	// Probe + retention from earlier stages, plus the four
	// topology reconcilers and two alert pipelines from A4.
	want := []string{
		"probe",
		"retention",
		"topology-sysinfo-reconciler",
		"topology-iftable-reconciler",
		"topology-edge-reconciler",
		"topology-arp-reconciler",
		"alert-listener-pipeline",
		"alert-observation-pipeline",
	}
	for _, n := range want {
		if !names[n] {
			t.Errorf("engine %q not registered after full init; got %v", n, names)
		}
	}
}
