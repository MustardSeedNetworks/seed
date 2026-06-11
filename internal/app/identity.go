package app

// identity.go wires the composition root to the identity application
// (use-case) services (ADR-0020, ADR-0024): user management, SSO identity
// sync, and personal-access-token mint/list/revoke. The adapters below
// implement the narrow ports declared in internal/identity/{users,tokens,oauth}
// over the concrete collaborators (the database, the token repository, and the
// license manager), so the API handlers depend on use-cases instead of reaching
// the service container directly. Collaborators are resolved through lazy
// accessors on each call so a later-set value (the api test harness) is honored,
// and a nil collaborator degrades the use-case to its ErrUnavailable behavior
// rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	ssosync "github.com/MustardSeedNetworks/seed/internal/identity/oauth"
	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

// NewIdentityUsers builds the user-management use-case (ADR-0020) over a
// lazy accessor for the database. A nil database makes every method degrade
// to users.ErrUnavailable (the pre-strangle 503 path).
func NewIdentityUsers(db func() *database.DB) *users.Service {
	return users.NewService(dbUsersAdapter{db: db})
}

// NewIdentityTokens builds the PAT use-case (ADR-0020) over lazy accessors
// for the token repository and the license manager. A nil repository degrades
// every method to tokens.ErrUnavailable; a nil license manager permits minting
// (the pre-strangle dev/test behavior: builds without a license manager).
func NewIdentityTokens(
	repo func() *database.APITokenRepository,
	mgr func() *license.Manager,
) *tokens.Service {
	return tokens.NewService(
		dbTokensAdapter{repo: repo},
		licenseTokenGate{mgr: mgr},
	)
}

// NewIdentityOAuth builds the SSO identity-sync use-case (ADR-0020) over a
// lazy accessor for the database. A nil database degrades to ssosync.ErrUnavailable
// (the pre-strangle "User store unavailable" redirect path).
func NewIdentityOAuth(db func() *database.DB) *ssosync.Service {
	return ssosync.NewService(dbSSOAdapter{db: db})
}

// ── users adapter ────────────────────────────────────────────────────────────

// dbUsersAdapter implements users.Repository over *database.DB, resolving it
// lazily. Methods beyond Available are only invoked by the use-case once
// Available reports true, so they assume a non-nil database.
type dbUsersAdapter struct {
	db func() *database.DB
}

func (a dbUsersAdapter) Available() bool { return a.db() != nil }

func (a dbUsersAdapter) List(ctx context.Context) ([]*database.User, error) {
	return a.db().ListUsers(ctx)
}

func (a dbUsersAdapter) Create(ctx context.Context, username, hash, role string) (*database.User, error) {
	return a.db().CreateUser(ctx, username, hash, role)
}

func (a dbUsersAdapter) Get(ctx context.Context, username string) (*database.User, error) {
	return a.db().GetUser(ctx, username)
}

func (a dbUsersAdapter) UpdatePassword(ctx context.Context, username, hash string) error {
	return a.db().UpdateUserPassword(ctx, username, hash)
}

func (a dbUsersAdapter) UpdateRole(ctx context.Context, username, role string) error {
	return a.db().UpdateUserRole(ctx, username, role)
}

func (a dbUsersAdapter) Deactivate(ctx context.Context, username string) error {
	return a.db().DeactivateUser(ctx, username)
}

func (a dbUsersAdapter) Delete(ctx context.Context, username string) error {
	return a.db().DeleteUser(ctx, username)
}

// ── tokens adapters ──────────────────────────────────────────────────────────

// dbTokensAdapter implements tokens.Store over *database.APITokenRepository,
// resolving it lazily.
type dbTokensAdapter struct {
	repo func() *database.APITokenRepository
}

func (a dbTokensAdapter) Available() bool { return a.repo() != nil }

func (a dbTokensAdapter) Insert(ctx context.Context, t database.APITokenRecord) error {
	return a.repo().Insert(ctx, t)
}

func (a dbTokensAdapter) ListByOwner(ctx context.Context, owner string) ([]database.APITokenRecord, error) {
	return a.repo().ListByOwner(ctx, owner)
}

func (a dbTokensAdapter) Revoke(ctx context.Context, id, owner string) error {
	return a.repo().Revoke(ctx, id, owner)
}

// licenseTokenGate implements tokens.LicenseGate over *license.Manager,
// resolving it lazily. A nil manager permits minting — the pre-strangle dev/test
// behavior (builds without a license manager allow the feature so it stays usable).
type licenseTokenGate struct {
	mgr func() *license.Manager
}

func (g licenseTokenGate) AllowsMinting() bool {
	mgr := g.mgr()
	if mgr == nil {
		return true
	}
	return mgr.HasFeature("rest_api")
}

// ── SSO adapter ──────────────────────────────────────────────────────────────

// dbSSOAdapter implements ssosync.Repository over *database.DB, resolving it
// lazily.
type dbSSOAdapter struct {
	db func() *database.DB
}

func (a dbSSOAdapter) Available() bool { return a.db() != nil }

func (a dbSSOAdapter) SyncUser(ctx context.Context, in database.SSOUserInput) (*database.User, error) {
	return a.db().UpsertSSOUser(ctx, in)
}
