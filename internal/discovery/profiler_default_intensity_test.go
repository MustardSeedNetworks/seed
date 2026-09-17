package discovery_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// seed#2674, owner decision 2026-09-15: a fresh install profiles at the light
// intensity, so a discovered device gets a type instead of a bare address. The
// separate full port scan (Options.PortScan) stays opt-in.
func TestDefaultProfilerProfilesAtTheLightIntensity(t *testing.T) {
	cfg := discovery.DefaultProfilerConfig()

	if cfg.PortScanIntensity != discovery.PortScanQuick {
		t.Errorf("default port scan intensity = %q, want %q",
			cfg.PortScanIntensity, discovery.PortScanQuick)
	}
	if len(cfg.GetPortsForIntensity()) == 0 {
		t.Error("the light intensity must probe a port list; an empty list profiles nothing")
	}
}
