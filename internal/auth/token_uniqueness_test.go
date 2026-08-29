package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// Two logins by the same user inside the same second must not produce the same
// token.
//
// The claims carry no unique id and jwt.NewNumericDate truncates to seconds, so
// every claim is identical for logins in the same second. The blacklist is keyed
// on sha256 of the whole token string, so identical tokens share a key -- and
// logging out one session revokes every other session minted in that second.
//
// Found via an E2E flake: two real logouts produced 72 "Token is blacklisted"
// 401s across a parallel run, because the shared session token collided with the
// tokens the logout tests minted.
func TestTokensAreUniquePerMint(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager("", time.Hour, defaults.Auth.Username, defaults.Auth.PasswordHash)
	ctx := context.Background()

	first, err := m.GenerateToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("first GenerateToken: %v", err)
	}
	second, err := m.GenerateToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("second GenerateToken: %v", err)
	}

	if first == second {
		t.Fatal("two tokens minted in the same second are byte-identical; " +
			"per-session revocation cannot distinguish them")
	}
}

// Revoking one session must not revoke another session for the same user.
func TestRevokingOneSessionLeavesTheOtherValid(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager("", time.Hour, defaults.Auth.Username, defaults.Auth.PasswordHash)
	ctx := context.Background()

	keep, err := m.GenerateToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("GenerateToken (keep): %v", err)
	}
	revoke, err := m.GenerateToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("GenerateToken (revoke): %v", err)
	}

	m.RevokeToken(revoke)

	if _, revokedErr := m.ValidateToken(ctx, revoke); revokedErr == nil {
		t.Error("the revoked token still validates")
	}
	// The other session was never logged out and must survive.
	if _, keepErr := m.ValidateToken(ctx, keep); keepErr != nil {
		t.Errorf("logging out one session revoked another: %v", keepErr)
	}
}
