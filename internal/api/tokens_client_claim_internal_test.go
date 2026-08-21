package api

// tokens_client_claim_internal_test.go covers the PAT half of the session
// client claim: an automation token carries its owner's tenancy, resolved at
// request time, and a token whose owner has no resolvable client is rejected
// rather than silently attributed to the default tenant.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// insertPAT mints a token owned by owner and returns its plaintext.
func insertPAT(t *testing.T, s *Server, owner string) string {
	t.Helper()

	id, secret, err := mintTokenMaterial()
	if err != nil {
		t.Fatalf("mintTokenMaterial: %v", err)
	}
	plaintext := APITokenPrefix + secret
	rec := database.APITokenRecord{
		ID: id, OwnerUsername: owner, Name: "ci",
		TokenHash: hashAPIToken(plaintext),
		Prefix:    plaintext[:apiTokenDisplayPrefix],
	}
	if insErr := s.apiTokens.Insert(context.Background(), rec); insErr != nil {
		t.Fatalf("insert token: %v", insErr)
	}
	return plaintext
}

func TestAPITokenMiddlewareCarriesOwnersClient(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	db := s.db()
	if _, err := db.Exec(context.Background(),
		`INSERT INTO clients (id, name, slug, created_at, updated_at)
		 VALUES ('acme', 'Acme', 'acme', '2026-08-20T00:00:00Z', '2026-08-20T00:00:00Z')`,
	); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	if _, err := db.Exec(context.Background(),
		`UPDATE users SET client_id = 'acme' WHERE username = 'carol'`,
	); err != nil {
		t.Fatalf("assign client: %v", err)
	}

	plaintext := insertPAT(t, s, "carol")

	var got string
	var gotErr error
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, gotErr = auth.ClientIDFromContext(r.Context())
	})
	mw := apiTokenMiddleware(s.apiTokens, s.resolveClientID, next)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	if gotErr != nil {
		t.Fatalf("handler saw no client claim: %v", gotErr)
	}
	if got != "acme" {
		t.Errorf("client = %q, want %q — a PAT must carry its owner's tenancy", got, "acme")
	}
}

func TestAPITokenMiddlewareRejectsUnresolvableClient(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	mgr.StartTrial()

	plaintext := insertPAT(t, s, "carol")

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	unresolvable := func(_ context.Context, _ string) (string, error) {
		return "", errors.New("client identity unavailable")
	}
	mw := apiTokenMiddleware(s.apiTokens, unresolvable, next)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if called {
		t.Error("handler ran for a request that could not be attributed to a tenant")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestResolveClientIDFailsClosedWithoutDatabase(t *testing.T) {
	t.Parallel()

	var s Server
	if _, err := s.resolveClientID(context.Background(), "carol"); err == nil {
		t.Error("resolved a client with no database; the seam must fail closed")
	}
}
