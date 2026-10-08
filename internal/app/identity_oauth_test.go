package app_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	ssosync "github.com/MustardSeedNetworks/seed/internal/identity/oauth"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

// newSSOSync builds the SSO identity-sync use-case the way the composition
// root does, over a fresh database.
func newSSOSync(t *testing.T) *ssosync.Service {
	t.Helper()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return app.NewIdentityOAuth(func() *database.DB { return db })
}

// The three tests below pin the SSO identity-sync seam end to end. The OAuth
// callback relies on every behaviour asserted here: the bootstrap role, the
// synthetic username, the idempotent upsert that refreshes the IdP profile,
// and the refusals.

func TestIdentityOAuthSyncUserWithoutDatabase(t *testing.T) {
	t.Parallel()
	svc := app.NewIdentityOAuth(func() *database.DB { return nil })
	_, err := svc.SyncUser(t.Context(), ssosync.Identity{Provider: "google", ExternalID: "sub-1"})
	if !errors.Is(err, ssosync.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestIdentityOAuthSyncUserUpserts(t *testing.T) {
	t.Parallel()
	svc := newSSOSync(t)

	first, err := svc.SyncUser(t.Context(), ssosync.Identity{
		Provider: "google", ExternalID: "sub-1", Email: "a@example.com", DisplayName: "A",
	})
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if first.Username != "google:sub-1" || first.Role != roles.Admin || first.AuthProvider != "google" ||
		first.ExternalID != "sub-1" || first.TokenVersion != 1 || !first.IsActive {
		t.Fatalf("first sync user = %+v, want bootstrap admin google:sub-1", first)
	}
	if first.PasswordHash != "!sso-no-password" {
		t.Fatalf("password hash = %q, want the unmatchable sentinel", first.PasswordHash)
	}

	again, err := svc.SyncUser(t.Context(), ssosync.Identity{
		Provider: "google", ExternalID: "sub-1", Email: "b@example.com", DisplayName: "B",
	})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if again.ID != first.ID || again.Email != "b@example.com" || again.DisplayName != "B" {
		t.Fatalf("second sync = %+v, want id %d with the refreshed profile", again, first.ID)
	}

	other, err := svc.SyncUser(t.Context(), ssosync.Identity{Provider: "github", ExternalID: "42"})
	if err != nil {
		t.Fatalf("other sync: %v", err)
	}
	if other.ID == first.ID || other.Role != roles.Viewer {
		t.Fatalf("other sync = %+v, want a new viewer", other)
	}
}

func TestIdentityOAuthSyncUserRefuses(t *testing.T) {
	t.Parallel()
	svc := newSSOSync(t)

	for name, in := range map[string]ssosync.Identity{
		"unsupported provider": {Provider: "okta", ExternalID: "sub-1"},
		"no external id":       {Provider: "google"},
		"no provider":          {ExternalID: "sub-1"},
	} {
		if u, err := svc.SyncUser(t.Context(), in); err == nil {
			t.Errorf("%s: got user %+v, want an error", name, u)
		}
	}
}
