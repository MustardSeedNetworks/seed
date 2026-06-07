package api

import (
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// TestSettingsUseCasePersistsToDefaultProfile exercises the ADR-0016 phase-3
// adapter wiring end-to-end against a real database: SaveToActiveProfile must
// write the live config's profile JSON onto the seeded default profile.
func TestSettingsUseCasePersistsToDefaultProfile(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Capture the seeded default profile's original config for comparison.
	def, err := db.Profiles().GetDefault(t.Context())
	if err != nil {
		t.Fatalf("get default profile: %v", err)
	}
	original := def.ConfigJSON

	cfg := config.DefaultConfig()
	cfg.DisplayOptions.UnitSystem = "metric" // a distinctive, serialized value

	s := &Server{
		config:   cfg,
		services: NewServiceContainer(),
	}
	s.services.Database.DB = db
	s.initSettingsUseCase()

	if saveErr := s.settingsStore.SaveToActiveProfile(t.Context()); saveErr != nil {
		t.Fatalf("SaveToActiveProfile: %v", saveErr)
	}

	got, err := db.Profiles().GetDefault(t.Context())
	if err != nil {
		t.Fatalf("re-get default profile: %v", err)
	}
	if got.ConfigJSON == original {
		t.Fatal("profile ConfigJSON was not updated")
	}

	want, err := cfg.ToProfileJSON()
	if err != nil {
		t.Fatalf("ToProfileJSON: %v", err)
	}
	if got.ConfigJSON != want {
		t.Fatalf("persisted config mismatch:\n got=%s\nwant=%s", got.ConfigJSON, want)
	}
}

// TestSettingsUseCaseNoDatabaseIsNoOp verifies the use-case tolerates a missing
// database (the historic "persist to config file only" path) without error.
func TestSettingsUseCaseNoDatabaseIsNoOp(t *testing.T) {
	s := &Server{
		config:   config.DefaultConfig(),
		services: NewServiceContainer(),
	}
	s.initSettingsUseCase()

	if err := s.settingsStore.SaveToActiveProfile(t.Context()); err != nil {
		t.Fatalf("expected no-op without a db, got %v", err)
	}
}
