package api

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	profilesapp "github.com/MustardSeedNetworks/seed/internal/profiles/app"
)

// newProfilesTestServer builds a Server wired to a real database (which seeds a
// default profile) with the profiles use-case initialised.
func newProfilesTestServer(t *testing.T) (*Server, *database.DB) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s := &Server{services: NewServiceContainer()}
	s.services.Database.DB = db
	s.initProfilesUseCase()
	return s, db
}

// TestProfilesUseCaseAdapterCRUD exercises the ADR-0016 phase-3 adapter against a
// real database: the database.Profile <-> profilesapp.Profile mapping and the
// ErrProfileNotFound/ErrProfileNameExists -> sentinel translation.
func TestProfilesUseCaseAdapterCRUD(t *testing.T) {
	s, _ := newProfilesTestServer(t)
	ctx := t.Context()

	// The seeded default profile is present.
	list, err := s.profiles.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || !list[0].IsDefault {
		t.Fatalf("expected one seeded default profile, got %+v", list)
	}

	// Create.
	created, err := s.profiles.Create(ctx, profilesapp.NewProfile{Name: "field-site", ConfigJSON: `{"k":1}`})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created profile missing ID")
	}

	// Duplicate name maps to the sentinel.
	if _, dupErr := s.profiles.Create(ctx, profilesapp.NewProfile{Name: "field-site"}); !errors.Is(
		dupErr,
		profilesapp.ErrNameExists,
	) {
		t.Fatalf("duplicate create should map to ErrNameExists, got %v", dupErr)
	}

	// Get round-trips the row.
	got, err := s.profiles.Get(ctx, created.ID)
	if err != nil || got.Name != "field-site" || got.ConfigJSON != `{"k":1}` {
		t.Fatalf("get mismatch: %+v err=%v", got, err)
	}

	// Missing id maps to ErrNotFound.
	if _, missErr := s.profiles.Get(ctx, "nope"); !errors.Is(missErr, profilesapp.ErrNotFound) {
		t.Fatalf("missing get should map to ErrNotFound, got %v", missErr)
	}

	// Set active + resolve.
	if _, setErr := s.profiles.SetActiveProfile(ctx, created.ID); setErr != nil {
		t.Fatalf("set active: %v", setErr)
	}
	active, err := s.profiles.ActiveProfile(ctx)
	if err != nil || active.ID != created.ID {
		t.Fatalf("active resolve mismatch: %+v err=%v", active, err)
	}

	// Cannot delete the active profile.
	if delErr := s.profiles.Delete(ctx, created.ID); !errors.Is(delErr, profilesapp.ErrDeleteActive) {
		t.Fatalf("deleting active should be ErrDeleteActive, got %v", delErr)
	}

	// Cannot delete the default profile.
	if defErr := s.profiles.Delete(ctx, list[0].ID); !errors.Is(defErr, profilesapp.ErrDeleteDefault) {
		t.Fatalf("deleting default should be ErrDeleteDefault, got %v", defErr)
	}
}
