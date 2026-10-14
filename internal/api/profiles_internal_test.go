package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
)

// newProfilesTestServer builds a Server wired to a real database (which seeds a
// default profile) with the profiles use-case initialised.
func newProfilesTestServer(t *testing.T) (*Server, *database.DB) {
	t.Helper()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s := &Server{}
	s.dbConn = db
	// A real config + path so activating a profile (which applies its saved config
	// to the running config and persists) has somewhere to write.
	s.config = config.DefaultConfig()
	s.configPath = filepath.Join(t.TempDir(), "config.yaml")
	s.profiles = app.NewProfiles(s.db, s.config, s.configPath)
	// Wire the identity use-cases (ADR-0024) so writeGated→requireRole→callerRole
	// resolves through the repository port instead of nil-panicking.
	s.initIdentityUseCases()
	return s, db
}

// TestProfilesUseCaseAdapterCRUD exercises the ADR-0016 phase-3 adapter against a
// real database: the database.Profile <-> catalog.Profile mapping and the
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
	created, err := s.profiles.Create(ctx, catalog.NewProfile{Name: "field-site", ConfigJSON: `{"k":1}`})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created profile missing ID")
	}

	// Duplicate name maps to the sentinel.
	if _, dupErr := s.profiles.Create(ctx, catalog.NewProfile{Name: "field-site"}); !errors.Is(
		dupErr,
		catalog.ErrNameExists,
	) {
		t.Fatalf("duplicate create should map to ErrNameExists, got %v", dupErr)
	}

	// Get round-trips the row.
	got, err := s.profiles.Get(ctx, created.ID)
	if err != nil || got.Name != "field-site" || got.ConfigJSON != `{"k":1}` {
		t.Fatalf("get mismatch: %+v err=%v", got, err)
	}

	// Missing id maps to ErrNotFound.
	if _, missErr := s.profiles.Get(ctx, "nope"); !errors.Is(missErr, catalog.ErrNotFound) {
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
	if delErr := s.profiles.Delete(ctx, created.ID); !errors.Is(delErr, catalog.ErrDeleteActive) {
		t.Fatalf("deleting active should be ErrDeleteActive, got %v", delErr)
	}

	// Cannot delete the default profile.
	if defErr := s.profiles.Delete(ctx, list[0].ID); !errors.Is(defErr, catalog.ErrDeleteDefault) {
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
	staleReq.Header.Set("If-Match", `"999999"`) // a row_version that cannot match
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
	if fresh := okRec.Header().Get("ETag"); fresh == "" || fresh == etag {
		t.Fatalf("successful PUT should emit a fresh ETag different from %q, got %q", etag, fresh)
	}
	if cur, _ := db.Profiles().Get(ctx, "p1"); cur.Name != "renamed" {
		t.Fatalf("update should have persisted, got name=%q", cur.Name)
	}
}

// routedProfilesServer is newProfilesTestServer with the route table built and
// an admin user for Mux's session, so requests reach the handlers through the
// same patterns the daemon serves.
func routedProfilesServer(t *testing.T) *Server {
	t.Helper()
	s, db := newProfilesTestServer(t)
	if _, err := db.CreateUser(t.Context(), "admin", "$2a$10$x", roles.Admin); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	s.withRouteDeps(t)
	s.setupRoutes()
	return s
}

// The UI duplicates a profile by posting to /profiles/{id}/duplicate. The
// handler trimmed a stale "/api/profiles/" prefix from the versioned path, so
// the store was asked for "/api/v1/profiles/<id>" and every duplicate on a
// licensed install answered 404.
func TestDuplicateProfileThroughTheRouter(t *testing.T) {
	t.Parallel()
	s := routedProfilesServer(t)

	list, err := s.profiles.List(t.Context())
	if err != nil || len(list) != 1 {
		t.Fatalf("list seeded profiles: %+v, %v", list, err)
	}
	source := list[0]

	rec := httptest.NewRecorder()
	s.Mux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		APIVersionPrefix+"/profiles/"+source.ID+"/duplicate", strings.NewReader(`{"name":"site-b"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("duplicate = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var got ProfileResponse
	if decodeErr := json.Unmarshal(rec.Body.Bytes(), &got); decodeErr != nil {
		t.Fatalf("decode: %v", decodeErr)
	}
	if got.ID == source.ID || got.Name != "site-b" {
		t.Errorf("duplicate = %+v, want a new profile named site-b", got)
	}
}

// Each by-id operation reaches its handler through its own pattern, and a path
// with more segments than any of them is not a profile id.
func TestProfileByIDRoutes(t *testing.T) {
	t.Parallel()
	s := routedProfilesServer(t)

	created, err := s.profiles.Create(t.Context(), catalog.NewProfile{Name: "field-site", ConfigJSON: `{}`})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	base := APIVersionPrefix + "/profiles/" + created.ID

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"get", http.MethodGet, base, "", http.StatusOK},
		{"update", http.MethodPut, base, `{"name":"renamed","config":{}}`, http.StatusOK},
		{"patch settings", http.MethodPatch, base + "/settings", `{"theme":"dark"}`, http.StatusOK},
		{"settings is patch only", http.MethodPut, base + "/settings", `{}`, http.StatusMethodNotAllowed},
		{"duplicate is post only", http.MethodGet, base + "/duplicate", "", http.StatusMethodNotAllowed},
		{"nested path", http.MethodGet, base + "/settings/extra", "", http.StatusNotFound},
		{"delete", http.MethodDelete, base, "", http.StatusOK},
		{"gone", http.MethodGet, base, "", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		s.Mux().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if rec.Code != tc.want {
			t.Errorf("%s: %s %s = %d, want %d: %s", tc.name, tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
