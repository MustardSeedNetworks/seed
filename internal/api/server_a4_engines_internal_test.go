package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/license"
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
// engines, leaving only probe + retention in s.engines.
//
// We therefore assert "construction succeeded" by checking the
// init function didn't panic and the registry has the Free-tier
// engines; the full set is exercised by integration tests that
// wire a Pro license.

func TestInitTopologyReconcilers_RegistersOnPro(t *testing.T) {
	// No license manager wired -> effectiveTier returns Pro,
	// so all four topology reconcilers land.
	s := &Server{engines: engine.NewRegistry(nil)}
	s.initTopologyReconcilers(newTestDB(t))

	names := make(map[string]bool)
	for _, e := range s.engines.Engines() {
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
	s := &Server{engines: engine.NewRegistry(nil)}
	s.initAlertPipelines(newTestDB(t))

	names := make(map[string]bool)
	for _, e := range s.engines.Engines() {
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
	// An empty licenseDir means no activation, so the manager reports Free and
	// the tier gate filters out everything Starter+. probe + retention are
	// Free-tier and survive.
	//
	// The directory is set explicitly because the default is the developer's
	// real ~/.config/seed: this assertion used to depend on whether the machine
	// running it happened to have a licence activated, and was green in CI only
	// because runners start clean (#2155).
	s := &Server{engines: engine.NewRegistry(nil), licenseDir: t.TempDir()}
	s.initDatabaseDependentServices(newTestDB(t))

	names := make(map[string]bool)
	for _, e := range s.engines.Engines() {
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

// The gate above only means something if licenseDir is actually honoured --
// an ignored field would leave every run on the ambient state it is meant to
// replace. Start a trial in a temp directory, point the server at it, and the
// tier gate must open for Starter+ engines.
func TestInitDatabaseDependentServices_HonoursLicenseDir(t *testing.T) {
	dir := t.TempDir()

	mgr, mgrErr := license.NewManagerWithDir(dir)
	if mgrErr != nil {
		t.Fatalf("NewManagerWithDir: %v", mgrErr)
	}
	if res := mgr.StartTrial(); res == nil || !res.Success {
		t.Fatalf("StartTrial did not activate: %+v", res)
	}

	s := &Server{engines: engine.NewRegistry(nil), licenseDir: dir}
	s.initDatabaseDependentServices(newTestDB(t))

	names := make(map[string]bool)
	for _, e := range s.engines.Engines() {
		names[e.Name()] = true
	}
	// The Free-tier test above asserts this same engine is gated OUT, so the
	// pair brackets the behaviour: absent without a licence, present with one.
	if !names["topology-sysinfo-reconciler"] {
		t.Errorf("a trial licence must admit Starter+ engines; got %v", names)
	}

	// And the real config directory was left alone.
	home, homeErr := os.UserHomeDir()
	if homeErr == nil {
		if _, statErr := os.Stat(filepath.Join(home, ".config", "seed", ".license")); statErr == nil {
			t.Error("the test wrote activation state to the real user config directory")
		}
	}
}
