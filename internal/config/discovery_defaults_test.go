package config_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// seed#2674: an install that never visits Settings must still discover its
// network, and it must rescan often enough that the pages track reality. The
// owner found a 10-minute rescan on a fresh install; these cases pin the
// shipped answer so a later default change cannot quietly turn discovery off
// or slow it back down.
func TestFreshInstallDiscoversByDefault(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("Load with no config file: %v", err)
	}

	opts := cfg.NetworkDiscovery.Options
	for name, on := range map[string]bool{
		"lldp":    opts.PassiveProtocols.LLDP,
		"cdp":     opts.PassiveProtocols.CDP,
		"edp":     opts.PassiveProtocols.EDP,
		"arpScan": opts.ARPScan,
		"icmp":    opts.ICMPScan,
	} {
		if !on {
			t.Errorf("fresh install ships %s off; the owner decision is scan by default", name)
		}
	}

	// Port scanning and traceroute stay opt-in: they act on hosts rather than
	// listening for them.
	if opts.PortScan.Enabled {
		t.Error("port scanning must stay opt-in")
	}
	if opts.Traceroute {
		t.Error("traceroute must stay opt-in")
	}

	if got, want := cfg.NetworkDiscovery.Timing.RescanInterval, time.Minute; got != want {
		t.Errorf("rescan interval = %s, want %s", got, want)
	}
	if !cfg.NetworkDiscovery.Enabled || !cfg.NetworkDiscovery.AutoScan {
		t.Error("discovery must be enabled and auto-scanning on a fresh install")
	}
}

// The shipped sample config is what an operator copies; it must not contradict
// the built-in defaults.
func TestSampleConfigMatchesTheDiscoveryDefaults(t *testing.T) {
	cfg, err := config.Load("../../configs/seed.json")
	if err != nil {
		t.Fatalf("Load configs/seed.json: %v", err)
	}
	if got, want := cfg.NetworkDiscovery.Timing.RescanInterval, time.Minute; got != want {
		t.Errorf("configs/seed.json rescan interval = %s, want %s", got, want)
	}
	if !cfg.NetworkDiscovery.Options.ARPScan || !cfg.NetworkDiscovery.Options.ICMPScan {
		t.Error("configs/seed.json must ship the active sweeps on")
	}
}
