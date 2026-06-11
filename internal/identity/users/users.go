// Package users holds the user-management application (use-case) layer
// (ADR-0020, ADR-0024). It owns the user-CRUD orchestration that previously
// lived in the api.Server identity handlers — listing, creating, getting,
// updating passwords, updating roles, deactivating, and deleting users —
// behind a narrow consumer-defined Repository port over the concrete database.
// Handlers keep transport concerns: request decode, authorization, response
// shaping, and error-to-status mapping. The adapter satisfying the port lives
// in the composition root (internal/app) and resolves the database lazily, so a
// nil database degrades every method to ErrUnavailable (the pre-strangle 503)
// rather than panicking.
//
// Domain sentinels (database.ErrUserExists, database.ErrUserNotFound,
// database.ErrLastAdmin) are NOT remapped here — they pass through verbatim so
// handlers keep their existing [errors.Is] switches mapping to 409/404/etc.
package users

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	// ErrUnavailable signals the user store is not wired (handlers map it to
	// 503, the pre-strangle degraded behavior).
	ErrUnavailable = errors.New("user store not available")
)

// Repository is the user-store surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *database.DB in
// internal/app. Available reports whether a user store is wired (resolved per
// call so the use-case can degrade gracefully); the remaining methods are only
// invoked once availability is confirmed.
//
// The port returns *database.User directly (ADR-0024: thin CRUD stays thin;
// no new domain DTO is introduced because it would be churn without a security
// or clarity gain).
type Repository interface {
	Available() bool
	List(ctx context.Context) ([]*database.User, error)
	Create(ctx context.Context, username, hash, role string) (*database.User, error)
	Get(ctx context.Context, username string) (*database.User, error)
	UpdatePassword(ctx context.Context, username, hash string) error
	UpdateRole(ctx context.Context, username, role string) error
	Deactivate(ctx context.Context, username string) error
	Delete(ctx context.Context, username string) error
}

// Service is the user-management use-case.
type Service struct {
	repo Repository
}

// NewService builds the use-case over its narrow repository dependency.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns all users. Returns ErrUnavailable when the store is not wired.
func (s *Service) List(ctx context.Context) ([]*database.User, error) {
	if !s.repo.Available() {
		return nil, ErrUnavailable
	}
	return s.repo.List(ctx)
}

// Create inserts a new user. Returns ErrUnavailable when the store is not
// wired; domain errors (database.ErrUserExists) pass through verbatim.
func (s *Service) Create(ctx context.Context, username, hash, role string) (*database.User, error) {
	if !s.repo.Available() {
		return nil, ErrUnavailable
	}
	return s.repo.Create(ctx, username, hash, role)
}

// Get returns a user by username. Returns ErrUnavailable when the store is not
// wired; database.ErrUserNotFound passes through verbatim.
func (s *Service) Get(ctx context.Context, username string) (*database.User, error) {
	if !s.repo.Available() {
		return nil, ErrUnavailable
	}
	return s.repo.Get(ctx, username)
}

// UpdatePassword persists a new password hash for the given user. Returns
// ErrUnavailable when the store is not wired; domain errors pass through.
func (s *Service) UpdatePassword(ctx context.Context, username, hash string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.UpdatePassword(ctx, username, hash)
}

// UpdateRole sets the user's role. Returns ErrUnavailable when the store is not
// wired; database.ErrUserNotFound and database.ErrLastAdmin pass through.
func (s *Service) UpdateRole(ctx context.Context, username, role string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.UpdateRole(ctx, username, role)
}

// Deactivate marks the user inactive. Returns ErrUnavailable when the store is
// not wired; database.ErrUserNotFound passes through verbatim.
func (s *Service) Deactivate(ctx context.Context, username string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.Deactivate(ctx, username)
}

// Delete removes the user. Returns ErrUnavailable when the store is not wired;
// database.ErrUserNotFound and database.ErrLastAdmin pass through verbatim.
func (s *Service) Delete(ctx context.Context, username string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.Delete(ctx, username)
}
