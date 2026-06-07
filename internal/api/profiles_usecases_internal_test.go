package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

// TestProfileOptimisticConcurrencyHTTP exercises the Phase-5 ETag/If-Match flow
// through the handlers: GET emits an ETag, a stale If-Match yields 412, and the
// current ETag updates and returns a fresh ETag.
func TestProfileOptimisticConcurrencyHTTP(t *testing.T) {
	s, db := newProfilesTestServer(t)
	ctx := t.Context()

	if err := db.Profiles().Create(ctx, &database.Profile{ID: "p1", Name: "orig", ConfigJSON: "{}"}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	// GET emits an ETag.
	getRec := httptest.NewRecorder()
	s.handleGetProfile(getRec, httptest.NewRequest(http.MethodGet, "/", http.NoBody), "p1")
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", getRec.Code)
	}
	etag := getRec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("GET should emit an ETag header")
	}

	body := `{"name":"renamed","config":{"x":1}}`

	// A stale If-Match is rejected with 412 and the row is untouched.
	staleRec := httptest.NewRecorder()
	staleReq := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	staleReq.Header.Set("If-Match", `"1999-01-01T00:00:00Z"`)
	s.handleUpdateProfile(staleRec, staleReq, "p1")
	if staleRec.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale If-Match status = %d, want 412", staleRec.Code)
	}
	if cur, _ := db.Profiles().Get(ctx, "p1"); cur.Name != "orig" {
		t.Fatalf("row should be unchanged after 412, got name=%q", cur.Name)
	}

	// The current ETag succeeds and returns a fresh ETag.
	okRec := httptest.NewRecorder()
	okReq := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	okReq.Header.Set("If-Match", etag)
	s.handleUpdateProfile(okRec, okReq, "p1")
	if okRec.Code != http.StatusOK {
		t.Fatalf("matching If-Match status = %d, want 200 (body=%s)", okRec.Code, okRec.Body.String())
	}
	if okRec.Header().Get("ETag") == "" {
		t.Fatal("successful PUT should emit a fresh ETag")
	}
	if cur, _ := db.Profiles().Get(ctx, "p1"); cur.Name != "renamed" {
		t.Fatalf("update should have persisted, got name=%q", cur.Name)
	}
}
