package management_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
)

// fakeStore is an in-memory management.Store over a config value; Write applies
// the mutation but does not persist (no disk in unit tests).
type fakeStore struct{ cfg *config.Config }

func (f *fakeStore) Read(fn func(*config.Config))              { fn(f.cfg) }
func (f *fakeStore) Write(fn func(*config.Config) error) error { return fn(f.cfg) }

func TestGetReturnsSettingsAndETag(t *testing.T) {
	cfg := &config.Config{}
	cfg.HealthChecks.RunSpeedtest = true
	svc := management.NewService(&fakeStore{cfg: cfg})

	settings, etag := svc.Get()
	if etag == "" {
		t.Error("Get should return a non-empty ETag")
	}
	for _, key := range []string{
		"interface", "vlan", "ip", "thresholds",
		"healthChecks", "speedtest", "iperf", "displayOptions",
	} {
		if _, ok := settings[key]; !ok {
			t.Errorf("settings map missing key %q", key)
		}
	}
	hc, ok := settings["healthChecks"].(map[string]any)
	if !ok || hc["runSpeedtest"] != true {
		t.Errorf("healthChecks read model did not reflect config: %+v", settings["healthChecks"])
	}
}

func TestUpdateAppliesChange(t *testing.T) {
	cfg := &config.Config{}
	svc := management.NewService(&fakeStore{cfg: cfg})

	err := svc.Update(map[string]any{
		"healthChecks": map[string]any{"runSpeedtest": true, "runDiscovery": true},
	}, "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !cfg.HealthChecks.RunSpeedtest || !cfg.HealthChecks.RunDiscovery {
		t.Errorf("Update did not apply health-check toggles: %+v", cfg.HealthChecks)
	}
}

func TestUpdateETagMismatchConflicts(t *testing.T) {
	svc := management.NewService(&fakeStore{cfg: &config.Config{}})
	err := svc.Update(map[string]any{"healthChecks": map[string]any{"runSpeedtest": true}}, "stale-etag")
	if !errors.Is(err, management.ErrConflict) {
		t.Errorf("stale If-Match: want ErrConflict, got %v", err)
	}
}

func TestUpdateInvalidTypeValidation(t *testing.T) {
	cfg := &config.Config{}
	svc := management.NewService(&fakeStore{cfg: cfg})
	// A string where a boolean is expected must surface as a validation error.
	err := svc.Update(map[string]any{
		"healthChecks": map[string]any{"runSpeedtest": "not-a-bool"},
	}, "")
	if !errors.Is(err, management.ErrValidation) {
		t.Errorf("wrong-typed update: want ErrValidation, got %v", err)
	}
	if cfg.HealthChecks.RunSpeedtest {
		t.Error("invalid update should not have mutated config")
	}
}
