// SPDX-License-Identifier: BUSL-1.1

package auth_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestFirstRun_NoUsableDefaultPassword locks the cross-repo first-run invariant
// (#1242): seed never ships a usable default password. A fresh install carries
// either an empty hash or the setup placeholder; both MUST be flagged as
// needing setup AND authenticate no password, so an operator is forced through
// the wizard. The legacy "seed" default must also be flagged so an old install
// that still carries it is detected and rotated. Mirrors the same guarantee
// stem (TestFirstRun_PlaceholderHashRejectsAllPasswords) and niac make.
func TestFirstRun_NoUsableDefaultPassword(t *testing.T) {
	t.Parallel()

	// 1. Fresh-install markers must be flagged as "needs setup".
	for _, hash := range []string{"", auth.SetupModePlaceholder} {
		if !auth.IsDefaultPasswordHash(hash) {
			t.Errorf("IsDefaultPasswordHash(%q) = false, want true (fresh install must require setup)", hash)
		}
	}

	// The legacy "seed" default must be detected too, so an old install that
	// still carries it is caught and rotated rather than silently accepted.
	seedHash, err := auth.HashPassword("seed")
	if err != nil {
		t.Fatalf("HashPassword(seed): %v", err)
	}
	if !auth.IsDefaultPasswordHash(seedHash) {
		t.Error("IsDefaultPasswordHash(hash of \"seed\") = false, want true — legacy default not detected")
	}

	// 2. No usable default: neither the empty hash nor the setup placeholder may
	// authenticate ANY password. These are the hashes a fresh install actually
	// ships, so this is the load-bearing assertion.
	guesses := []string{"", "admin", "password", "changeme", "seed", "default", auth.SetupModePlaceholder}
	for _, freshHash := range []string{"", auth.SetupModePlaceholder} {
		for _, guess := range guesses {
			matched, _, _ := auth.VerifyPassword(freshHash, guess)
			if matched {
				t.Errorf("VerifyPassword(%q, %q) matched — a usable default password leaked", freshHash, guess)
			}
		}
	}

	// 3. A securely-set password is NOT flagged as default (the inverse — so the
	// gate doesn't false-positive after the operator completes setup).
	realHash, err := auth.HashPassword("a-strong-unique-operator-password-2026")
	if err != nil {
		t.Fatalf("HashPassword(real): %v", err)
	}
	if auth.IsDefaultPasswordHash(realHash) {
		t.Error("IsDefaultPasswordHash(real password) = true, want false")
	}
}
