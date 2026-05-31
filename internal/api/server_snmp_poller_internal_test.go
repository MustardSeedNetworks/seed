package api

import "testing"

func TestInitSNMPPoller_RegistersOnPro(t *testing.T) {
	// No license.Manager wired -> effectiveTier returns Pro,
	// so the Starter-tier snmp-poller registers.
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
