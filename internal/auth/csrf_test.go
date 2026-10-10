package auth_test

import (
	"net/http"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestGetSessionIDFromRequest_HashedKeying pins the security-invariant change:
// the CSRF session key is now sha256(bearer) (foundation's SessionKey), not the
// raw JWT payload segment. It must not embed the token plaintext, must be the
// 64-char sha256 hex, and a request with no token must yield "": it has no
// session for a cross-site page to ride.
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

	// No token → no session key.
	empty, _ := http.NewRequest(http.MethodPost, "/api/v1/x", nil)
	if k := auth.GetSessionIDFromRequest(empty); k != "" {
		t.Errorf("no-token request should yield empty key, got %q", k)
	}
}
