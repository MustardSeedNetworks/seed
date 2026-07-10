package auth_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestCSRFManagerGenerateAndValidate exercises Seed's thin wrapper over
// foundation's manager: a token generated for a session validates, and a wrong
// session or wrong token is rejected with foundation's ErrTokenInvalid. The
// per-session store mechanics themselves are covered in foundation's tests.
func TestCSRFManagerGenerateAndValidate(t *testing.T) {
	manager := auth.NewCSRFManager()
	defer manager.Stop()

	sessionID := "test-session"

	token, err := manager.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if validateErr := manager.ValidateToken(sessionID, token); validateErr != nil {
		t.Errorf("failed to validate token: %v", validateErr)
	}

	if wrongSessionErr := manager.ValidateToken("wrong-session", token); !errors.Is(
		wrongSessionErr, csrf.ErrTokenInvalid,
	) {
		t.Errorf("expected ErrTokenInvalid, got %v", wrongSessionErr)
	}

	if wrongTokenErr := manager.ValidateToken(sessionID, "wrong-token"); !errors.Is(
		wrongTokenErr, csrf.ErrTokenInvalid,
	) {
		t.Errorf("expected ErrTokenInvalid, got %v", wrongTokenErr)
	}
}

// TestCSRFManagerStop confirms Stop shuts the manager down cleanly (delegating
// to foundation's cleanup goroutine) without hanging or panicking.
func TestCSRFManagerStop(t *testing.T) {
	manager := auth.NewCSRFManager()
	if _, err := manager.GenerateToken("session1"); err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	manager.Stop()
}

// TestGetSessionIDFromRequest_HashedKeying pins the security-invariant change:
// the CSRF session key is now sha256(bearer) (foundation's SessionKey), not the
// raw JWT payload segment. It must not embed the token plaintext, must be the
// 64-char sha256 hex, and a request with no token must yield "" so the
// middleware rejects with 401.
func TestGetSessionIDFromRequest_HashedKeying(t *testing.T) {
	bearer := "header.payload-segment.signature"
	r, _ := http.NewRequest(http.MethodPost, "/api/v1/x", nil)
	r.Header.Set("Authorization", "Bearer "+bearer)

	key := auth.GetSessionIDFromRequest(r)
	if key == "" {
		t.Fatal("expected a session key for an authenticated request")
	}
	if key == bearer || key == "payload-segment" {
		t.Errorf("session key must not be the raw token/payload segment (got %q)", key)
	}
	if len(key) != 64 { // sha256 hex
		t.Errorf("session key len = %d, want 64 (sha256 hex)", len(key))
	}

	// No token → empty key so the middleware can 401.
	empty, _ := http.NewRequest(http.MethodPost, "/api/v1/x", nil)
	if k := auth.GetSessionIDFromRequest(empty); k != "" {
		t.Errorf("no-token request should yield empty key, got %q", k)
	}
}
