// Package tokens holds the personal-access-token application (use-case) layer
// (ADR-0020, ADR-0024). It owns the PAT mint/list/revoke orchestration that
// previously lived in the api.Server identity handlers, behind narrow
// consumer-defined Store and LicenseGate ports. Handlers keep transport
// concerns: request decode, token-material generation, scope/authorization
// checks, response shaping, and error-to-status mapping.
//
// Per ADR-0024: the per-token scope cap (scope ≤ ownerRole) is an
// authorization decision resolved at the edge (callerRole) and validated in
// the handler — the use-case never performs authorization. Service.Mint only
// enforces the license gate (ErrMintingNotAllowed) and the store seam
// (ErrUnavailable). [sql.ErrNoRows] from Revoke passes through verbatim so the
// handler maps it to 404.
package tokens

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	// ErrUnavailable signals the token store is not wired (handlers map it
	// to 503, the pre-strangle degraded behavior).
	ErrUnavailable = errors.New("token store not available")

	// ErrMintingNotAllowed signals the active license does not permit PAT
	// minting (handlers map it to 402 TIER_TOO_LOW).
	ErrMintingNotAllowed = errors.New("API token minting requires the Pro tier")
)

// Store is the token-storage surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *database.APITokenRepository
// in internal/app. Available reports whether a store is wired.
type Store interface {
	Available() bool
	Insert(ctx context.Context, t database.APITokenRecord) error
	ListByOwner(ctx context.Context, owner string) ([]database.APITokenRecord, error)
	Revoke(ctx context.Context, id, owner string) error
}

// LicenseGate is the license-feature surface the use-case drives. AllowsMinting
// reports whether the active license tier permits PAT minting.
type LicenseGate interface {
	AllowsMinting() bool
}

// Service is the personal-access-token use-case.
type Service struct {
	store Store
	gate  LicenseGate
}

// NewService builds the use-case over its narrow store and license-gate
// dependencies.
func NewService(store Store, gate LicenseGate) *Service {
	return &Service{store: store, gate: gate}
}

// Mint persists a pre-built token record. The handler is responsible for
// token-material generation (id/secret/hash/prefix) and scope/authorization
// checks before calling Mint. Mint enforces the license gate and the store
// seam only. Returns ErrMintingNotAllowed when the license does not permit
// minting; ErrUnavailable when the store is not wired.
func (s *Service) Mint(ctx context.Context, rec database.APITokenRecord) error {
	if !s.gate.AllowsMinting() {
		return ErrMintingNotAllowed
	}
	if !s.store.Available() {
		return ErrUnavailable
	}
	return s.store.Insert(ctx, rec)
}

// List returns the active tokens owned by the given user. Returns an empty
// slice (not ErrUnavailable) when the store is not wired — the pre-strangle
// list handler returned an empty array when repo was nil.
func (s *Service) List(ctx context.Context, owner string) ([]database.APITokenRecord, error) {
	if !s.store.Available() {
		return []database.APITokenRecord{}, nil
	}
	return s.store.ListByOwner(ctx, owner)
}

// Revoke removes the token identified by (id, owner). Returns ErrUnavailable
// when the store is not wired; [sql.ErrNoRows] passes through verbatim so the
// handler can map it to 404.
func (s *Service) Revoke(ctx context.Context, id, owner string) error {
	if !s.store.Available() {
		return ErrUnavailable
	}
	err := s.store.Revoke(ctx, id, owner)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return err
}
