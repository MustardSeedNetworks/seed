package api

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/engine"
)

func TestInitSNMPPoller_RegistersOnPro(t *testing.T) {
	// No license.Manager wired -> effectiveTier returns Pro,
	// so the Starter-tier snmp-poller registers.
	s := &Server{engines: engine.NewRegistry(nil)}
	s.initSNMPPoller(newTestDB(t))

	names := make(map[string]bool)
	for _, e := range s.engines.Engines() {
		names[e.Name()] = true
	}
	if !names["snmp-poller"] {
		t.Errorf("snmp-poller not registered; got engines = %v", names)
	}
}
