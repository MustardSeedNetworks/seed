// Package targets is the polling-targets CRUD use-case (ADR-0020, WS-A7): the
// /api/v1/polling-targets endpoints' application service over a narrow Repository
// port, so the transport layer depends on a use-case instead of reaching into the
// database directly. The Repository is satisfied by an adapter in the composition
// root over the polling-target repository.
package targets

import (
	"context"
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

var (
	// ErrNotFound is returned when no polling target matches the id.
	ErrNotFound = errors.New("targets: polling target not found")
	// ErrUnavailable is returned when the store is not wired (handler → 503).
	ErrUnavailable = errors.New("targets: store unavailable")
)

// ValidationError carries a user-input validation message from the repository
// (the handler maps it to 400 with the message verbatim).
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Repository is the persistence surface the use-case needs. It mirrors the
// polling-target repository; the adapter satisfies it over
// database.PollingTargetRepository and surfaces ErrUnavailable when no DB is wired.
type Repository interface {
	List(ctx context.Context, clientID string) ([]*database.PollingTarget, error)
	Get(ctx context.Context, id string) (*database.PollingTarget, error)
	Create(ctx context.Context, t *database.PollingTarget) error
	Update(ctx context.Context, t *database.PollingTarget) error
	Delete(ctx context.Context, id string) error
}

// Service is the polling-targets CRUD use-case.
type Service struct {
	repo Repository
}

// NewService builds the use-case over its Repository port.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// List returns the polling targets for clientID ("" for all).
func (s *Service) List(ctx context.Context, clientID string) ([]*database.PollingTarget, error) {
	return s.repo.List(ctx, clientID)
}

// Get returns one target, mapping the repository's not-found to ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (*database.PollingTarget, error) {
	t, err := s.repo.Get(ctx, id)
	return t, mapNotFound(err)
}

// Create persists a new target. A repository validation error (the
// "polling_targets:" prefix is the repo's user-input signal) is returned as a
// ValidationError; everything else propagates as-is.
func (s *Service) Create(ctx context.Context, t *database.PollingTarget) error {
	err := s.repo.Create(ctx, t)
	if err != nil && strings.HasPrefix(err.Error(), "polling_targets:") {
		return ValidationError{Msg: err.Error()}
	}
	return err
}

// Update applies a full update and returns the freshly-read row so the caller
// sees the refreshed audit columns; on a re-read miss it returns the written
// value. Maps the repository's not-found to ErrNotFound.
func (s *Service) Update(ctx context.Context, t *database.PollingTarget) (*database.PollingTarget, error) {
	if err := mapNotFound(s.repo.Update(ctx, t)); err != nil {
		return nil, err
	}
	if current, err := s.repo.Get(ctx, t.ID); err == nil && current != nil {
		return current, nil
	}
	return t, nil
}

// Delete removes a target, mapping the repository's not-found to ErrNotFound.
func (s *Service) Delete(ctx context.Context, id string) error {
	return mapNotFound(s.repo.Delete(ctx, id))
}

// mapNotFound normalizes the repository's not-found sentinel to ErrNotFound,
// leaving other errors (including ErrUnavailable from the adapter) untouched.
func mapNotFound(err error) error {
	if errors.Is(err, database.ErrPollingTargetNotFound) {
		return ErrNotFound
	}
	return err
}
