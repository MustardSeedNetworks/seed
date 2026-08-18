package api

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/engine"
)

func TestInitSNMPPoller_RegistersOnPro(t *testing.T) {
	// No license.Manager wired -> effectiveTier returns Pro,
	// so the Starter-tier snmp-poller registers.
	//
	// A config is required now that the poller resolves credentials: it supplies
	// the keyring the resolver decrypts with. An empty Config is enough — it
	// lazily creates an ephemeral keyring — but a nil one means there is no
	// keyring at all, and the poller declines to register rather than come up
	// unable to authenticate to anything.
	s := &Server{engines: engine.NewRegistry(nil), config: &config.Config{}}
	s.initSNMPPoller(newTestDB(t))

	names := make(map[string]bool)
	for _, e := range s.engines.Engines() {
		names[e.Name()] = true
	}
	if !names["snmp-poller"] {
		t.Errorf("snmp-poller not registered; got engines = %v", names)
	}
}
