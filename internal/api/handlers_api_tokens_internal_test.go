// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

// apiTokenTestSetup wires up the minimum surface needed by the
// token handlers: a temp SQLite DB + license manager rooted at a
// tmpdir so test runs don't touch the developer's real license.
func apiTokenTestSetup(t *testing.T) (*Server, *license.Manager) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-token-*.db")
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

	// Seed the usernames these tests mint tokens for. The hardening
	// migration added FOREIGN KEY (owner_username) REFERENCES users
	// ON DELETE CASCADE on api_tokens, so any owner referenced by an
	// inserted token must exist as a users row first.
	for _, name := range []string{"alice", "bob", "carol"} {
		_, createErr := db.CreateUser(t.Context(), name, "$2a$10$x", database.RoleAdmin)
		if createErr != nil {
			t.Fatalf("seed user %q: %v", name, createErr)
		}
	}

	licenseDir := t.TempDir()
	mgr, mgrErr := license.NewManagerWithDir(licenseDir)
	if mgrErr != nil {
		t.Fatalf("license manager: %v", mgrErr)
	}

	s := &Server{
		mux:      http.NewServeMux(),
		services: NewServiceContainer(),
	}
	s.services.Database.DB = db
	s.services.Auth.APITokens = database.NewAPITokenRepository(db)
	s.services.Auth.License = mgr
	// Wire the discovery use-cases so routed handlers (e.g. the
	// /discovery/engine/events SSE policy test) have a non-nil use-case; the
	// engine stays nil here, so they degrade to the unavailable (503) path.
	s.initDiscoveryUseCases()
	// Wire the identity use-cases (ADR-0024) so the token handlers and
	// callerRole resolve through the repository ports against the seeded store.
	s.initIdentityUseCases()
	return s, mgr
}

func newAuthedRequest(method, path string, body []byte, username string) *http.Request {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	var req *http.Request
	if bodyReader != nil {
		req = httptest.NewRequest(method, path, bodyReader)
	} else {
		req = httptest.NewRequest(method, path, http.NoBody)
	}
	// X-Username is the contract between the JWT/token middleware and
	// downstream handlers; setting it directly bypasses the middleware
	// for unit tests, which is what we want here.
	req.Header.Set("X-Username", username)
	return req
}

func TestMintRequiresPro_NoLicense(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	// No StartTrial → license state is nil → the tokens use-case LicenseGate
	// (HasFeature "rest_api") returns false → mint is rejected with 402.

	body, _ := json.Marshal(MintTokenRequest{Name: "ci"})
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens", body, "alice")
	w := httptest.NewRecorder()
	s.handleAPITokens(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402; body=%s", w.Code, w.Body.String())
	}
}

func TestMintRequiresPro_TrialAllowed(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}

	body, _ := json.Marshal(MintTokenRequest{Name: "ci-bot"})
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens", body, "alice")
	w := httptest.NewRecorder()
	s.handleAPITokens(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var out MintTokenResponse
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(out.Token, APITokenPrefix) {
		t.Errorf("token missing prefix: %q", out.Token)
	}
	if out.Prefix == "" || !strings.HasPrefix(out.Token, out.Prefix) {
		t.Errorf("prefix mismatch: token=%q prefix=%q", out.Token, out.Prefix)
	}
}

func TestMintRejectsEmptyName(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	body, _ := json.Marshal(MintTokenRequest{Name: "   "})
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens", body, "alice")
	w := httptest.NewRecorder()
	s.handleAPITokens(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestMintRequiresAuthentication(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	body, _ := json.Marshal(MintTokenRequest{Name: "no-user"})
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens", body, "")
	w := httptest.NewRecorder()
	s.handleAPITokens(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestListAndRevokeRoundTrip(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	// Mint.
	body, _ := json.Marshal(MintTokenRequest{Name: "tok1"})
	w := httptest.NewRecorder()
	s.handleAPITokens(w, newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens", body, "alice"))
	if w.Code != http.StatusCreated {
		t.Fatalf("mint failed: %d %s", w.Code, w.Body.String())
	}
	var minted MintTokenResponse
	_ = json.NewDecoder(w.Body).Decode(&minted)

	// List shows it.
	w = httptest.NewRecorder()
	s.handleAPITokens(w, newAuthedRequest(http.MethodGet, APIVersionPrefix+"/tokens", nil, "alice"))
	if w.Code != http.StatusOK {
		t.Fatalf("list failed: %d", w.Code)
	}
	var list []TokenListItem
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].ID != minted.ID {
		t.Errorf("list mismatch: %+v", list)
	}

	// Revoke.
	w = httptest.NewRecorder()
	s.handleAPITokenByID(w,
		newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/tokens/"+minted.ID, nil, "alice"))
	if w.Code != http.StatusNoContent {
		t.Errorf("revoke status = %d, want 204; body=%s", w.Code, w.Body.String())
	}

	// Revoke again is 404.
	w = httptest.NewRecorder()
	s.handleAPITokenByID(w,
		newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/tokens/"+minted.ID, nil, "alice"))
	if w.Code != http.StatusNotFound {
		t.Errorf("second revoke status = %d, want 404", w.Code)
	}
}

func TestRevokeForeignTokenFails(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	// Alice mints.
	body, _ := json.Marshal(MintTokenRequest{Name: "alice-tok"})
	w := httptest.NewRecorder()
	s.handleAPITokens(w, newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens", body, "alice"))
	var minted MintTokenResponse
	_ = json.NewDecoder(w.Body).Decode(&minted)

	// Bob tries to revoke Alice's token.
	w = httptest.NewRecorder()
	s.handleAPITokenByID(w,
		newAuthedRequest(http.MethodDelete, APIVersionPrefix+"/tokens/"+minted.ID, nil, "bob"))
	if w.Code != http.StatusNotFound {
		t.Errorf("bob revoking alice's token: status = %d, want 404 (silent reject)", w.Code)
	}
}

func TestAPITokenMiddlewareResolvesValidToken(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	// Mint a token directly via the repo so we have the plaintext.
	id, secret, err := mintTokenMaterial()
	if err != nil {
		t.Fatalf("mintTokenMaterial: %v", err)
	}
	plaintext := APITokenPrefix + secret
	rec := database.APITokenRecord{
		ID: id, OwnerUsername: "carol", Name: "ci",
		TokenHash: hashAPIToken(plaintext),
		Prefix:    plaintext[:apiTokenDisplayPrefix],
	}
	if insErr := s.services.Auth.APITokens.Insert(context.Background(), rec); insErr != nil {
		t.Fatalf("insert: %v", insErr)
	}

	var capturedUser string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedUser = r.Header.Get("X-Username")
	})
	mw := apiTokenMiddleware(s.services.Auth.APITokens, next)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if capturedUser != "carol" {
		t.Errorf("captured user = %q, want %q (status=%d)", capturedUser, "carol", w.Code)
	}
}

func TestAPITokenMiddlewareRejectsBadToken(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	mw := apiTokenMiddleware(s.services.Auth.APITokens, next)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+APITokenPrefix+"deadbeef")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if called {
		t.Error("next handler should not run when token is invalid")
	}
}

func TestAPITokenMiddlewareFallsThroughForJWT(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	mw := apiTokenMiddleware(s.services.Auth.APITokens, next)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.fake.fake")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("non-PAT bearer should fall through to next middleware")
	}
}

func TestAPITokenMiddlewareSkipsNonAPI(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	mw := apiTokenMiddleware(s.services.Auth.APITokens, next)

	req := httptest.NewRequest(http.MethodGet, "/static/app.js", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+APITokenPrefix+"anything")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("non-API path should bypass token middleware")
	}
}
