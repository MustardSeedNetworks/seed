package auth_test

// client_claim_mint_test.go proves every token this manager issues carries the
// client its user belongs to, that a store which cannot answer fails the mint
// rather than issuing a token claiming the default tenant, and that the
// middleware hands the claim on as a context value.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

func mintManager(t *testing.T, store auth.UserStore) *auth.Manager {
	t.Helper()

	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret, time.Hour,
		defaults.Auth.Username, defaults.Auth.PasswordHash,
	)
	if store != nil {
		m.SetUserStore(store)
	}
	return m
}

func TestIssuedTokensCarryTheUsersClient(t *testing.T) {
	t.Parallel()

	store := newMockUserStore()
	store.clientIDs["admin"] = "acme"
	m := mintManager(t, store)
	ctx := context.Background()

	mints := map[string]func() (string, error){
		"access":  func() (string, error) { return m.GenerateAccessToken(ctx, "admin") },
		"refresh": func() (string, error) { return m.GenerateRefreshToken(ctx, "admin") },
		"session": func() (string, error) { return m.GenerateToken(ctx, "admin") },
	}
	for name, mint := range mints {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			token, err := mint()
			if err != nil {
				t.Fatalf("mint %s: %v", name, err)
			}
			claims, err := m.ValidateToken(ctx, token)
			if err != nil {
				t.Fatalf("validate %s: %v", name, err)
			}
			if claims.ClientID != "acme" {
				t.Errorf("%s token client = %q, want %q", name, claims.ClientID, "acme")
			}
		})
	}
}

func TestMintFailsWhenClientCannotBeResolved(t *testing.T) {
	t.Parallel()

	store := newMockUserStore()
	store.clientErr = errors.New("store unavailable")
	m := mintManager(t, store)

	if _, err := m.GenerateAccessToken(context.Background(), "admin"); err == nil {
		t.Error("minted a token for a user whose client could not be resolved")
	}
}

func TestStorelessMintUsesTheDefaultClient(t *testing.T) {
	t.Parallel()

	m := mintManager(t, nil)
	ctx := context.Background()

	token, err := m.GenerateAccessToken(ctx, "admin")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	claims, err := m.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.ClientID != auth.DefaultClientID {
		t.Errorf("storeless client = %q, want %q", claims.ClientID, auth.DefaultClientID)
	}
}

func TestMiddlewareHandsTheClaimOnAsContext(t *testing.T) {
	t.Parallel()

	store := newMockUserStore()
	store.clientIDs["admin"] = "acme"
	m := mintManager(t, store)

	token, err := m.GenerateAccessToken(context.Background(), "admin")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	var got string
	var gotErr error
	handler := m.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, gotErr = auth.ClientIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	// A caller-supplied client must not survive into the handler's context.
	req.Header.Set("X-Client-Id", "attacker-tenant")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotErr != nil {
		t.Fatalf("handler saw no client claim: %v", gotErr)
	}
	if got != "acme" {
		t.Errorf("handler client = %q, want %q", got, "acme")
	}
}
