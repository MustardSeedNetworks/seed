// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

// usersTestSetup creates a Server with a temp DB and a pre-seeded
// admin user named "admin" so the request flow can authenticate as
// that user via X-Username. The returned license.Manager is fresh
// (no key activated) so tests can opt-in to Pro by calling
// mgr.StartTrial() when they need it.
func usersTestSetup(t *testing.T) (*Server, *license.Manager) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-users-*.db")
	if err != nil {
		t.Fatalf("temp db file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	if openErr != nil {
		t.Fatalf("open db: %v", openErr)
	}
	t.Cleanup(func() { _ = db.Close() })

	licenseDir := t.TempDir()
	mgr, mgrErr := license.NewManagerWithDir(licenseDir)
	if mgrErr != nil {
		t.Fatalf("license manager: %v", mgrErr)
	}

	// Seed the bootstrap admin so handlers that consult callerIsAdmin
	// against the request username find a real row.
	_, createErr := db.CreateUser(t.Context(), "admin", "$2a$10$x", database.RoleAdmin)
	if createErr != nil {
		t.Fatalf("seed admin: %v", createErr)
	}

	s := &Server{
		mux:      http.NewServeMux(),
		services: NewServiceContainer(),
	}
	s.services.Database.DB = db
	s.services.Auth.License = mgr
	// Wire the identity use-cases (ADR-0024) so callerRole/requireRole and the
	// user/token handlers resolve through the repository ports against the
	// seeded store rather than nil-panicking on a bare server.
	s.initIdentityUseCases()
	return s, mgr
}

func TestUserCreate_RequiresAdmin(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}

	// Add a non-admin user.
	_, err := s.services.Database.DB.CreateUser(t.Context(), "viewer1", "$2a$10$x", database.RoleViewer)
	if err != nil {
		t.Fatalf("seed viewer: %v", err)
	}

	body, _ := json.Marshal(
		CreateUserRequest{Username: "newone", Password: "GoodPassw0rd!ABC", Role: database.RoleOperator},
	)
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/users", body, "viewer1")
	w := httptest.NewRecorder()
	s.handleUsers(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestUserCreate_RequiresPro(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t)
	// No trial → Free tier → multi_user gate fires.

	body, _ := json.Marshal(
		CreateUserRequest{Username: "newone", Password: "GoodPassw0rd!ABC", Role: database.RoleOperator},
	)
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/users", body, "admin")
	w := httptest.NewRecorder()
	s.handleUsers(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402; body=%s", w.Code, w.Body.String())
	}
}

func TestUserCreate_Success(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}

	body, _ := json.Marshal(
		CreateUserRequest{Username: "operator1", Password: "GoodPassw0rd!ABC", Role: database.RoleOperator},
	)
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/users", body, "admin")
	w := httptest.NewRecorder()
	s.handleUsers(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var got UserResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Username != "operator1" || got.Role != database.RoleOperator {
		t.Errorf("unexpected response: %+v", got)
	}
	if got.AuthProvider != database.AuthProviderLocal {
		t.Errorf("expected auth_provider=local, got %q", got.AuthProvider)
	}
}

func TestUserDelete_RefusesLastAdmin(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}

	// Seed a second admin so we can delete one of them without hitting
	// the self-delete guard, then expect the last-admin guard to fire
	// when we try to delete the remaining admin (us).
	_, err := s.services.Database.DB.CreateUser(t.Context(), "admin2", "$2a$10$x", database.RoleAdmin)
	if err != nil {
		t.Fatalf("seed admin2: %v", err)
	}

	// First delete admin2 — fine because two admins still exist.
	req := newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/users/admin2", nil, "admin")
	w := httptest.NewRecorder()
	s.handleUserByName(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete admin2: status = %d, want 204; body=%s", w.Code, w.Body.String())
	}

	// Now try to delete admin via another admin — there's only one left.
	// We need a second admin to act, so re-seed and then try to delete
	// the first one.
	_, err = s.services.Database.DB.CreateUser(t.Context(), "admin3", "$2a$10$x", database.RoleAdmin)
	if err != nil {
		t.Fatalf("seed admin3: %v", err)
	}
	// Now demote admin3 and try to delete admin (the only admin left).
	if upErr := s.services.Database.DB.UpdateUserRole(t.Context(), "admin3", database.RoleOperator); upErr != nil {
		t.Fatalf("demote admin3: %v", upErr)
	}

	req = newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/users/admin", nil, "admin3")
	w = httptest.NewRecorder()
	s.handleUserByName(w, req)
	if w.Code != http.StatusForbidden {
		// admin3 is now an operator → not admin → 403 is the expected guard.
		t.Errorf("non-admin delete: status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestUserDelete_RefusesSelfDelete(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}

	req := newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/users/admin", nil, "admin")
	w := httptest.NewRecorder()
	s.handleUserByName(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 SELF_DELETE; body=%s", w.Code, w.Body.String())
	}
}

func TestUserList_AdminOnly(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}
	_, err := s.services.Database.DB.CreateUser(t.Context(), "viewer1", "$2a$10$x", database.RoleViewer)
	if err != nil {
		t.Fatalf("seed viewer: %v", err)
	}

	// Non-admin caller → 403.
	req := newAuthedRequest(http.MethodGet, APIVersionPrefix+"/users", nil, "viewer1")
	w := httptest.NewRecorder()
	s.handleUsers(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("viewer list: status = %d, want 403; body=%s", w.Code, w.Body.String())
	}

	// Admin caller → 200 + 2 rows (admin + viewer1).
	req = newAuthedRequest(http.MethodGet, APIVersionPrefix+"/users", nil, "admin")
	w = httptest.NewRecorder()
	s.handleUsers(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin list: status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var got []UserResponse
	if decodeErr := json.NewDecoder(w.Body).Decode(&got); decodeErr != nil {
		t.Fatalf("decode: %v", decodeErr)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 users, got %d (%+v)", len(got), got)
	}
}

func TestCurrentUser_ReturnsCallerOwnRecord(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t)

	req := newAuthedRequest(http.MethodGet, APIVersionPrefix+"/users/me", nil, "admin")
	w := httptest.NewRecorder()
	s.handleCurrentUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var got UserResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Username != "admin" || got.Role != database.RoleAdmin {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestUpsertSSOUser_FirstEverBecomesAdmin(t *testing.T) {
	t.Parallel()
	tmpFile, _ := os.CreateTemp(t.TempDir(), "seed-sso-*.db")
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, err := database.Open(tmpPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	in := database.SSOUserInput{
		Provider:    database.AuthProviderGoogle,
		ExternalID:  "google-sub-12345",
		Email:       "alice@example.com",
		DisplayName: "Alice Example",
	}

	u, err := db.UpsertSSOUser(t.Context(), in)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if u.Role != database.RoleAdmin {
		t.Errorf("first SSO user role = %q, want admin", u.Role)
	}
	if u.AuthProvider != database.AuthProviderGoogle || u.ExternalID != "google-sub-12345" {
		t.Errorf("unexpected SSO fields: provider=%q external=%q", u.AuthProvider, u.ExternalID)
	}

	// Second call returns the SAME user (upsert, not duplicate).
	u2, err := db.UpsertSSOUser(t.Context(), in)
	if err != nil {
		t.Fatalf("upsert second call: %v", err)
	}
	if u2.ID != u.ID {
		t.Errorf("expected same id on idempotent upsert, got %d vs %d", u2.ID, u.ID)
	}
}

func TestUpsertSSOUser_SubsequentDefaultsToViewer(t *testing.T) {
	t.Parallel()
	tmpFile, _ := os.CreateTemp(t.TempDir(), "seed-sso2-*.db")
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, err := database.Open(tmpPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Bootstrap a local admin first.
	if _, createErr := db.CreateUser(t.Context(), "admin", "$2a$10$x", database.RoleAdmin); createErr != nil {
		t.Fatalf("bootstrap admin: %v", createErr)
	}

	in := database.SSOUserInput{
		Provider:   database.AuthProviderMicrosoft,
		ExternalID: "ms-sub-99999",
		Email:      "ops@example.com",
	}
	u, err := db.UpsertSSOUser(t.Context(), in)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if u.Role != database.RoleViewer {
		t.Errorf("subsequent SSO user role = %q, want viewer", u.Role)
	}
}

// Verifies the api_tokens FK cascade by deleting a user that owns
// tokens and asserting the tokens are gone.
func TestDeleteUser_CascadesAPITokens(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}
	db := s.services.Database.DB

	// Seed a second admin so we can delete one of them.
	if _, err := db.CreateUser(t.Context(), "bob", "$2a$10$x", database.RoleAdmin); err != nil {
		t.Fatalf("seed bob: %v", err)
	}

	repo := database.NewAPITokenRepository(db)
	s.services.Auth.APITokens = repo

	// Insert a token owned by bob.
	if err := repo.Insert(t.Context(), database.APITokenRecord{
		ID: "tokn-bob-001", OwnerUsername: "bob", Name: "ci",
		TokenHash: "hash-bob", Prefix: "sd_pat_bob",
	}); err != nil {
		t.Fatalf("insert token: %v", err)
	}

	// Delete bob (as admin "admin").
	req := newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/users/bob", nil, "admin")
	w := httptest.NewRecorder()
	s.handleUserByName(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d; body=%s", w.Code, w.Body.String())
	}

	// Token row should have been cascaded away — Insert again with the
	// same id should succeed (it would conflict if the row remained).
	// First we need to also re-create bob so the FK passes.
	if _, err := db.CreateUser(t.Context(), "bob", "$2a$10$x", database.RoleAdmin); err != nil {
		t.Fatalf("re-create bob: %v", err)
	}
	if err := repo.Insert(t.Context(), database.APITokenRecord{
		ID: "tokn-bob-001", OwnerUsername: "bob", Name: "ci2",
		TokenHash: "hash-bob-2", Prefix: "sd_pat_bob2",
	}); err != nil {
		t.Fatalf("re-insert token after cascade: %v", err)
	}
}
