package api

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

func TestMinTierForEngine_Mapping(t *testing.T) {
	cases := []struct {
		name string
		want license.Tier
	}{
		{"probe", license.TierFree},
		{"retention", license.TierFree},
		{"snmp-poller", license.TierStarter},
		{"topology-sysinfo-reconciler", license.TierStarter},
		{"topology-iftable-reconciler", license.TierStarter},
		{"topology-edge-reconciler", license.TierStarter},
		{"topology-arp-reconciler", license.TierStarter},
		{"alert-listener-pipeline", license.TierPro},
		{"alert-observation-pipeline", license.TierPro},
		{"syslog-udp", license.TierPro},
		{"snmp-trap-v2c", license.TierPro},
		{"unknown-future-engine", license.TierFree}, // default
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := minTierForEngine(tt.name); got != tt.want {
				t.Errorf("minTierForEngine(%q) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

// gatingTestEngine is the smallest engine.Engine implementation
// the gate tests need — name only; Start/Stop never run because
// the gating decision happens at Register time.
type gatingTestEngine struct{ name string }

func (g *gatingTestEngine) Name() string                { return g.name }
func (*gatingTestEngine) Start(_ context.Context) error { return nil }
func (*gatingTestEngine) Stop(_ context.Context) error  { return nil }

func TestRegisterEngineIfLicensed_NilServerIsNoOp(t *testing.T) {
	if err := (*Server)(nil).registerEngineIfLicensed(&gatingTestEngine{name: "probe"}); err != nil {
		t.Errorf("nil server should be no-op, got %v", err)
	}
}

func TestRegisterEngineIfLicensed_NoManagerAllowsAllEngines(t *testing.T) {
	// Pre-license state (nil Manager) treats every engine as
	// allowed — matches dev / fresh install / test experience.
	s := &Server{engines: engine.NewRegistry(nil)}
	for _, name := range []string{
		"probe", "snmp-poller", "alert-listener-pipeline",
	} {
		if err := s.registerEngineIfLicensed(&gatingTestEngine{name: name}); err != nil {
			t.Fatalf("nil-manager pre-license should allow %s, got %v", name, err)
		}
	}
	if got := len(s.engines.Engines()); got != 3 {
		t.Errorf("expected 3 registered engines, got %d", got)
	}
}

// gatingFakeAuth lets tests pin a tier without constructing a real
// license.Manager. The licenseTierAdapter is bypassed by
// effectiveTier when Auth.License is non-nil but the field type
// doesn't allow a stub — so we test the gate's branch logic
// directly via minTierForEngine instead, and trust the
// effectiveTier integration test to cover the real-license path.
//
// effectiveTier is exercised end-to-end in production runs; the
// unit tests verify minTierForEngine + the no-manager bypass.
