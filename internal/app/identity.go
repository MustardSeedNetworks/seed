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
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/identity/mfa"
	ssosync "github.com/MustardSeedNetworks/seed/internal/identity/oauth"
	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

// NewIdentityUsers builds the user-management use-case (ADR-0020) over a
// lazy accessor for the database. A nil database makes every method degrade
// to users.ErrUnavailable (the pre-strangle 503 path).
func NewIdentityUsers(persist func() *Persistence) *users.Service {
	return users.NewService(dbUsersAdapter{db: lazyHandle(persist)})
}

// NewIdentityTokens builds the PAT use-case (ADR-0020) over lazy accessors
// for the database (its api_tokens table) and the license manager. A nil
// database degrades every method to tokens.ErrUnavailable; a nil license manager permits minting
// (the pre-strangle dev/test behavior: builds without a license manager).
func NewIdentityTokens(
	persist func() *Persistence,
	mgr func() *license.Manager,
) *tokens.Service {
	return tokens.NewService(
		dbTokensAdapter{repo: lazyRepo(lazyHandle(persist), database.NewAPITokenRepository)},
		licenseTokenGate{mgr: mgr},
	)
}

// NewIdentityOAuth builds the SSO identity-sync use-case (ADR-0020) over a
// lazy accessor for the database. A nil database degrades to ssosync.ErrUnavailable
// (the pre-strangle "User store unavailable" redirect path).
func NewIdentityOAuth(persist func() *Persistence) *ssosync.Service {
	return ssosync.NewService(dbSSOAdapter{db: lazyHandle(persist)})
}

// NewMFAStore builds the second-factor store the MFA handlers enrol and verify
// through, over a lazy accessor for the database. It reports unavailable while
// no database is wired.
func NewMFAStore(persist func() *Persistence) mfa.Store {
	return dbMFAAdapter{db: lazyHandle(persist)}
}

// ResolveClientID answers which client owns a user, for the PAT seam. It fails
// closed when there is no database: a request that cannot be attributed to a
// tenant is unauthenticated, not a request belonging to the default tenant.
func ResolveClientID(ctx context.Context, persist *Persistence, username string) (string, error) {
	db := persist.handle()
	if db == nil {
		return "", errors.New("client identity unavailable: no database")
	}
	return db.GetClientID(ctx, username)
}

// AuthUserStore is the login-path user store, plus the startup migration of the
// config file's admin into the users table.
type AuthUserStore interface {
	auth.UserStore
	MigrateUserFromConfig(ctx context.Context, username, passwordHash string) error
}

// NewAuthUserStore builds the auth manager's user store over persist.
func NewAuthUserStore(persist *Persistence) AuthUserStore {
	return database.NewUserStoreAdapter(persist.handle())
}

// ── users adapter ────────────────────────────────────────────────────────────

// dbUsersAdapter implements users.Repository over *database.DB, resolving it
// lazily. Methods beyond Available are only invoked by the use-case once
// Available reports true, so they assume a non-nil database.
type dbUsersAdapter struct {
	db func() *database.DB
}

func (a dbUsersAdapter) Available() bool { return a.db() != nil }

func (a dbUsersAdapter) List(ctx context.Context) ([]*users.User, error) {
	return a.db().ListUsers(ctx)
}

func (a dbUsersAdapter) Create(ctx context.Context, username, hash, role string) (*users.User, error) {
	return a.db().CreateUser(ctx, username, hash, role)
}

func (a dbUsersAdapter) Get(ctx context.Context, username string) (*users.User, error) {
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

func (a dbUsersAdapter) GetDashboard(ctx context.Context, username string) ([]string, bool, error) {
	return a.db().GetUserDashboard(ctx, username)
}

func (a dbUsersAdapter) SetDashboard(ctx context.Context, username string, widgets []string) error {
	return a.db().SetUserDashboard(ctx, username, widgets)
}

// ── tokens adapters ──────────────────────────────────────────────────────────

// dbTokensAdapter implements tokens.Store over *database.APITokenRepository,
// resolving it lazily.
type dbTokensAdapter struct {
	repo func() *database.APITokenRepository
}

func (a dbTokensAdapter) Available() bool { return a.repo() != nil }

func (a dbTokensAdapter) Insert(ctx context.Context, t tokens.Record) error {
	return a.repo().Insert(ctx, t)
}

func (a dbTokensAdapter) ListByOwner(ctx context.Context, owner string) ([]tokens.Record, error) {
	return a.repo().ListByOwner(ctx, owner)
}

func (a dbTokensAdapter) Revoke(ctx context.Context, id, owner string) error {
	return a.repo().Revoke(ctx, id, owner)
}

func (a dbTokensAdapter) FindActiveByHash(ctx context.Context, hash string) (tokens.Record, error) {
	return a.repo().FindActiveByHash(ctx, hash)
}

func (a dbTokensAdapter) TouchLastUsed(ctx context.Context, id string) error {
	return a.repo().TouchLastUsed(ctx, id)
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

func (a dbSSOAdapter) SyncUser(ctx context.Context, in ssosync.Identity) (*users.User, error) {
	return a.db().UpsertSSOUser(ctx, in)
}

// ── MFA adapter ──────────────────────────────────────────────────────────────

// dbMFAAdapter implements mfa.Store over *database.DB, resolving it lazily.
type dbMFAAdapter struct {
	db func() *database.DB
}

func (a dbMFAAdapter) Available() bool { return a.db() != nil }

func (a dbMFAAdapter) SetTOTPSecret(ctx context.Context, username, secret string) error {
	return a.db().SetTOTPSecret(ctx, username, secret)
}

func (a dbMFAAdapter) EnableTOTP(ctx context.Context, username string) error {
	return a.db().EnableTOTP(ctx, username)
}

func (a dbMFAAdapter) DisableTOTP(ctx context.Context, username string) error {
	return a.db().DisableTOTP(ctx, username)
}

func (a dbMFAAdapter) GetTOTP(ctx context.Context, username string) (string, bool, error) {
	return a.db().GetTOTP(ctx, username)
}

func (a dbMFAAdapter) AddWebAuthnCredential(
	ctx context.Context, userID int64, cred mfa.WebAuthnCredential,
) (int64, error) {
	return a.db().AddWebAuthnCredential(ctx, userID, cred)
}

func (a dbMFAAdapter) ListWebAuthnCredentials(ctx context.Context, userID int64) ([]mfa.WebAuthnCredential, error) {
	return a.db().ListWebAuthnCredentials(ctx, userID)
}

func (a dbMFAAdapter) WebAuthnCredentialOwner(ctx context.Context, credentialID []byte) (string, error) {
	return a.db().WebAuthnCredentialOwner(ctx, credentialID)
}

func (a dbMFAAdapter) UpdateWebAuthnCredential(ctx context.Context, credentialID, record []byte) error {
	return a.db().UpdateWebAuthnCredential(ctx, credentialID, record)
}
