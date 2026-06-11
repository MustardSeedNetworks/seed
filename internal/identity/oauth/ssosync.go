// Package ssosync holds the SSO identity-sync application (use-case) layer
// (ADR-0020, ADR-0024). Its single responsibility is the UpsertSSOUser
// operation: syncing an IdP-authenticated identity into the local users table
// before session-token issuance. The handler keeps all transport concerns
// (cookie management, state/CSRF validation, provider code exchange, token
// issuance via authManager, redirects).
//
// Package name rationale: the directory is internal/identity/oauth to locate
// it near the other identity use-cases; the package name is ssosync rather
// than oauth to avoid an import-alias clash with the existing
// internal/oauth provider-registry package that handlers_oauth.go also
// imports. The composition root and api import this package as ssosync,
// unambiguously distinct from internal/oauth.
package ssosync

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP behavior.
var (
	// ErrUnavailable signals the user store is not wired (handlers redirect
	// with "User store unavailable." matching the pre-strangle path).
	ErrUnavailable = errors.New("user store not available")
)

// Repository is the user-store surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *database.DB in
// internal/app. Available reports whether a store is wired; SyncUser upserts
// the IdP identity into the users table.
type Repository interface {
	Available() bool
	SyncUser(ctx context.Context, in database.SSOUserInput) (*database.User, error)
}

// Service is the SSO identity-sync use-case.
type Service struct {
	repo Repository
}

// NewService builds the use-case over its narrow repository dependency.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SyncUser upserts the IdP-authenticated identity into the local user store.
// Returns ErrUnavailable when the store is not wired; other errors from the
// repository pass through verbatim.
func (s *Service) SyncUser(ctx context.Context, in database.SSOUserInput) (*database.User, error) {
	if !s.repo.Available() {
		return nil, ErrUnavailable
	}
	return s.repo.SyncUser(ctx, in)
}
