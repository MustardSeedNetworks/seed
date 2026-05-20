package auth_test

// Wave 2 / task #84 — Argon2id hashing + bcrypt-migration verification.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/krisarmstrong/seed/internal/auth"
)

// knownBcryptPlaintext is the plaintext that the bcrypt fixture
// corresponds to. Declared as a typed constant (gochecknoglobals
// exempts constants).
const knownBcryptPlaintext = "legacypass-1234"

// bcryptFixtureCost is the cost factor used when generating the legacy
// fixture hash; matches the production cost-12 threshold the code path
// migrates away from.
const bcryptFixtureCost = 12

// bcryptFixture returns a cost-12 bcrypt hash of knownBcryptPlaintext.
// It replaces the previous package-level init() (banned by
// gochecknoinits) and the previous package-level knownBcryptCost12Hash
// var (banned by gochecknoglobals). bcrypt at cost 12 takes ~50ms on
// the build hardware, so calling once per test is fine for the suite's
// throughput and avoids any package-level mutable state.
func bcryptFixture(t *testing.T) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(knownBcryptPlaintext), bcryptFixtureCost)
	if err != nil {
		t.Fatalf("bcrypt fixture: %v", err)
	}
	return string(h)
}

func TestHashPassword_ProducesArgon2id(t *testing.T) {
	hash, err := auth.HashPassword("CorrectHorseBattery1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("expected hash to start with $argon2id$v=19$, got %q", hash)
	}
	// Sanity: must contain the parameter segment.
	if !strings.Contains(hash, "m=65536,t=3,p=4") {
		t.Errorf("expected hash to embed RFC 9106 params, got %q", hash)
	}
}

func TestHashPassword_DifferentSaltsEachCall(t *testing.T) {
	h1, err := auth.HashPassword("SamePasswordEachTime1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := auth.HashPassword("SamePasswordEachTime1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Errorf("two argon2id hashes of the same password must differ (random salt)")
	}
}

func TestVerifyPassword_Argon2id_Match(t *testing.T) {
	hash, err := auth.HashPassword("MatchingPassword1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	matched, needsRehash, err := auth.VerifyPassword(hash, "MatchingPassword1!")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !matched {
		t.Error("expected matched=true for correct password")
	}
	if needsRehash {
		t.Error("expected needsRehash=false for an argon2id hash")
	}
}

func TestVerifyPassword_Argon2id_NoMatch(t *testing.T) {
	hash, err := auth.HashPassword("OriginalPassword1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	matched, needsRehash, err := auth.VerifyPassword(hash, "WrongPassword2!")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if matched {
		t.Error("expected matched=false for wrong password")
	}
	if needsRehash {
		t.Error("expected needsRehash=false on no-match")
	}
}

func TestVerifyPassword_Bcrypt_FlagsRehash(t *testing.T) {
	matched, needsRehash, err := auth.VerifyPassword(bcryptFixture(t), knownBcryptPlaintext)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !matched {
		t.Error("expected matched=true for correct bcrypt password")
	}
	if !needsRehash {
		t.Error("expected needsRehash=true for legacy bcrypt hash")
	}
}

func TestVerifyPassword_Bcrypt_WrongPassword(t *testing.T) {
	matched, needsRehash, err := auth.VerifyPassword(bcryptFixture(t), "wrong-plaintext")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if matched {
		t.Error("expected matched=false for wrong bcrypt password")
	}
	if needsRehash {
		t.Error("expected needsRehash=false on wrong-password bcrypt")
	}
}

func TestVerifyPassword_UnsupportedHash(t *testing.T) {
	tests := []string{
		"",
		"$pbkdf2$something",
		"plaintext-not-a-hash",
		"$argon2i$v=19$m=4096,t=3,p=1$xxx$yyy", // wrong variant
	}
	for _, h := range tests {
		matched, needsRehash, err := auth.VerifyPassword(h, "anything")
		if !errors.Is(err, auth.ErrUnsupportedHash) {
			t.Errorf("hash=%q: expected ErrUnsupportedHash, got %v", h, err)
		}
		if matched || needsRehash {
			t.Errorf("hash=%q: expected matched=false, needsRehash=false; got %v/%v", h, matched, needsRehash)
		}
	}
}

func TestDetectHashAlgorithm(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want auth.HashAlgorithm
	}{
		{"argon2id", "$argon2id$v=19$m=65536,t=3,p=4$xxx$yyy", auth.HashAlgorithmArgon2id},
		{"bcrypt 2a", "$2a$12$abc", auth.HashAlgorithmBcrypt},
		{"bcrypt 2b", "$2b$12$abc", auth.HashAlgorithmBcrypt},
		{"bcrypt 2y", "$2y$12$abc", auth.HashAlgorithmBcrypt},
		{"empty", "", auth.HashAlgorithmUnknown},
		{"random", "deadbeef", auth.HashAlgorithmUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.DetectHashAlgorithm(tt.hash)
			if got != tt.want {
				t.Errorf("DetectHashAlgorithm(%q) = %q; want %q", tt.hash, got, tt.want)
			}
		})
	}
}

// TestAuthenticateMigratesBcryptToArgon2id verifies that a successful
// login against a legacy bcrypt hash transparently rewrites the stored
// hash to argon2id via the UserStore.
func TestAuthenticateMigratesBcryptToArgon2id(t *testing.T) {
	ctx := context.Background()
	store := newMockUserStore()
	store.passwords["legacy-user"] = bcryptFixture(t)

	m := auth.NewManager("test-secret-please-change", 0, "legacy-user", "")
	m.SetUserStore(store)

	tok, err := m.Authenticate(ctx, "legacy-user", knownBcryptPlaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token after legacy login")
	}

	// The stored hash should now be argon2id.
	newHash := store.passwords["legacy-user"]
	if auth.DetectHashAlgorithm(newHash) != auth.HashAlgorithmArgon2id {
		t.Errorf("expected post-login hash to be argon2id, got %q", newHash)
	}

	// A second login with the new hash must continue to succeed.
	if _, reloginErr := m.Authenticate(ctx, "legacy-user", knownBcryptPlaintext); reloginErr != nil {
		t.Errorf("second Authenticate after rehash failed: %v", reloginErr)
	}
}
