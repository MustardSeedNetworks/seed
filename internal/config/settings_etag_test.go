package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestSettingsETag covers the optimistic-concurrency version token for the
// file-backed settings resource (ADR re-arch Phase 5). The token is a strong
// ETag: a content-hash of the mutable-settings subset (config.ProfileExportFields
// via ToProfileJSON), quoted per RFC 9110. It must be deterministic, change when
// a covered field changes, and stay stable when an excluded GLOBAL field changes.
func TestSettingsETag(t *testing.T) {
	cfg := config.DefaultConfig()

	base := cfg.SettingsETag()

	// Quoted strong validator (RFC 9110), non-empty hash body.
	if !strings.HasPrefix(base, `"`) || !strings.HasSuffix(base, `"`) {
		t.Fatalf("ETag not quoted: %q", base)
	}
	if len(strings.Trim(base, `"`)) == 0 {
		t.Fatalf("ETag has empty hash body: %q", base)
	}

	// Deterministic: a second call on an unchanged config returns the same token.
	if again := cfg.SettingsETag(); again != base {
		t.Fatalf("ETag not deterministic: %q vs %q", base, again)
	}

	// Changing a COVERED field (a threshold) must change the token.
	cfg.Thresholds.DNS.Warning += 5 * time.Millisecond
	if changed := cfg.SettingsETag(); changed == base {
		t.Fatalf("ETag unchanged after covered field changed: %q", changed)
	}

	// Changing an EXCLUDED global field (Auth) must NOT change the token: a
	// JWT-secret rotation must not invalidate a pending settings-edit token.
	afterCovered := cfg.SettingsETag()
	cfg.Auth.JWTSecret = "a-completely-different-secret-value"
	if global := cfg.SettingsETag(); global != afterCovered {
		t.Fatalf("ETag changed after excluded global field changed: %q vs %q", afterCovered, global)
	}
}

// TestSettingsETagLockedMatches asserts the no-lock variant (caller holds the
// lock) agrees with the locking accessor, so the GET (RLock) and PUT (write Lock)
// paths and the response accessor all derive the same token.
func TestSettingsETagLockedMatches(t *testing.T) {
	cfg := config.DefaultConfig()
	if locked, unlocked := cfg.SettingsETagLocked(), cfg.SettingsETag(); locked != unlocked {
		t.Fatalf("SettingsETagLocked %q != SettingsETag %q", locked, unlocked)
	}
}
