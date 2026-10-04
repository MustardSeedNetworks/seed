package api

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/license/licensetest"
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
		{"snmp-trap", license.TierPro},
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

// TestRegisterEngineIfLicensed_GatesOnTheLiveGrant drives the gate through a
// real manager. The persisted tier is what was once granted: an expired trial
// still records Pro, and a state no signature backs records TierInvalid, which
// sits below Free and used to gate out even the Free engines (D-SEED-13).
func TestRegisterEngineIfLicensed_GatesOnTheLiveGrant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state license.ActivationState
		want  []string
	}{
		{
			name:  "live trial runs every tier",
			state: trialState(t, 1),
			want:  []string{"probe", "snmp-poller", "alert-listener-pipeline"},
		},
		{
			name:  "expired trial runs Free only",
			state: trialState(t, license.TrialDays+1),
			want:  []string{"probe"},
		},
		{
			name: "unbacked Pro state runs Free only",
			state: license.ActivationState{
				Tier:     int(license.TierPro),
				Features: license.FeaturesForTier(license.TierPro),
			},
			want: []string{"probe"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{engines: engine.NewRegistry(nil), licenseMgr: managerWithState(t, tc.state)}
			for _, name := range []string{"probe", "snmp-poller", "alert-listener-pipeline"} {
				if err := s.registerEngineIfLicensed(&gatingTestEngine{name: name}); err != nil {
					t.Fatalf("register %s: %v", name, err)
				}
			}
			var got []string
			for _, eng := range s.engines.Engines() {
				got = append(got, eng.Name())
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("registered %v, want %v", got, tc.want)
			}
		})
	}
}

// trialState is a trial on this device that started daysAgo days ago.
func trialState(t *testing.T, daysAgo int) license.ActivationState {
	t.Helper()
	return license.ActivationState{
		DeviceHash:     thisDevice(t),
		Tier:           int(license.TierPro),
		TrialStartedAt: time.Now().AddDate(0, 0, -daysAgo),
		IsTrialMode:    true,
	}
}

// managerWithState loads a manager over st sealed on disk.
func managerWithState(t *testing.T, st license.ActivationState) *license.Manager {
	t.Helper()
	dir := t.TempDir()
	licensetest.WriteState(t, dir, st)
	mgr, err := license.NewManagerWithDir(dir)
	if err != nil {
		t.Fatalf("license manager: %v", err)
	}
	return mgr
}
