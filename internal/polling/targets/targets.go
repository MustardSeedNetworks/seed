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
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/polling"
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
	ListAll(ctx context.Context, clientID string) ([]*polling.Target, error)
	Get(ctx context.Context, clientID, id string) (*polling.Target, error)
	Create(ctx context.Context, t *polling.Target) error
	Update(ctx context.Context, clientID string, t *polling.Target) error
	Delete(ctx context.Context, clientID, id string) error
}

// Service is the polling-targets CRUD use-case.
type Service struct {
	repo Repository

	// createMu serialises CreateMissing. Its read-then-create is only
	// idempotent while nothing else is doing the same read.
	createMu sync.Mutex
}

// NewService builds the use-case over its Repository port.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// ListAll returns every polling target owned by clientID, enabled or not.
// There is no "all clients" spelling: the scheduler reads its own seam on the
// repository, and a management caller only ever sees its own tenant.
func (s *Service) ListAll(ctx context.Context, clientID string) ([]*polling.Target, error) {
	return s.repo.ListAll(ctx, clientID)
}

// Get returns one target, mapping the repository's not-found to ErrNotFound.
func (s *Service) Get(ctx context.Context, clientID, id string) (*polling.Target, error) {
	t, err := s.repo.Get(ctx, clientID, id)
	return t, mapNotFound(err)
}

// Create persists a new target. A repository validation error (the
// "polling_targets:" prefix is the repo's user-input signal) is returned as a
// ValidationError; everything else propagates as-is.
func (s *Service) Create(ctx context.Context, t *polling.Target) error {
	err := s.repo.Create(ctx, t)
	if err != nil && strings.HasPrefix(err.Error(), "polling_targets:") {
		return ValidationError{Msg: err.Error()}
	}
	return err
}

// Update applies a full update and returns the freshly-read row so the caller
// sees the refreshed audit columns; on a re-read miss it returns the written
// value. Maps the repository's not-found to ErrNotFound.
func (s *Service) Update(
	ctx context.Context, clientID string, t *polling.Target,
) (*polling.Target, error) {
	if err := mapNotFound(s.repo.Update(ctx, clientID, t)); err != nil {
		return nil, err
	}
	if current, err := s.repo.Get(ctx, clientID, t.ID); err == nil && current != nil {
		return current, nil
	}
	return t, nil
}

// CreateMissing persists each candidate whose address has no target yet, and
// returns how many were created.
//
// It exists because discovery promotes the devices a sweep found answering
// SNMP (seed#2692), and a sweep runs again every minute: the operation the
// promoter needs is "add what is not there", not "create". The read and the
// writes are taken under one lock because polling_targets has no unique index
// on the address — two callers that each read the list before either wrote
// would each create a row for the same device, and the poller would then poll
// it twice.
//
// An address that already has a target is left exactly as it is, enabled or
// not: disabling a target is how an operator says "do not poll this", and a
// promoter that re-enabled it would overrule them once a minute.
func (s *Service) CreateMissing(
	ctx context.Context, clientID string, candidates []*polling.Target,
) (int, error) {
	s.createMu.Lock()
	defer s.createMu.Unlock()

	existing, err := s.repo.ListAll(ctx, clientID)
	if err != nil {
		return 0, err
	}
	taken := make(map[string]struct{}, len(existing))
	for _, target := range existing {
		if target != nil {
			taken[target.IPAddress] = struct{}{}
		}
	}

	created := 0
	var failed error
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if _, known := taken[candidate.IPAddress]; known {
			continue
		}
		taken[candidate.IPAddress] = struct{}{}
		if createErr := s.Create(ctx, candidate); createErr != nil {
			// One unusable candidate must not cost the rest their
			// targets; the caller reports what could not be written.
			failed = errors.Join(failed, createErr)
			continue
		}
		created++
	}
	return created, failed
}

// Delete removes a target, mapping the repository's not-found to ErrNotFound.
func (s *Service) Delete(ctx context.Context, clientID, id string) error {
	return mapNotFound(s.repo.Delete(ctx, clientID, id))
}

// mapNotFound normalizes the repository's not-found sentinel to ErrNotFound,
// leaving other errors (including ErrUnavailable from the adapter) untouched.
func mapNotFound(err error) error {
	if errors.Is(err, polling.ErrTargetNotFound) {
		return ErrNotFound
	}
	return err
}
