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
// The package also owns the user entity and its domain sentinels (ErrUserExists,
// ErrUserNotFound, ErrLastAdmin): the database returns them and the use-case
// passes them through verbatim, so handlers map them to 409/404 without
// importing the database (seed#2750).
package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// User is a local or SSO account as the user store holds it.
type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	Role           string
	IsActive       bool
	LastLogin      *time.Time
	FailedAttempts int
	LockedUntil    *time.Time
	TokenVersion   int
	AuthProvider   string // 'local' | 'google' | 'microsoft' | 'github'
	ExternalID     string // IdP subject claim (OIDC 'sub' / MS Graph 'id'); empty for local users
	Email          string // display + cross-provider matching
	DisplayName    string // optional human name returned by the IdP
	ClientID       string // owning tenant; the session's client claim is minted from this
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	// ErrUnavailable signals the user store is not wired (handlers map it to
	// 503, the pre-strangle degraded behavior).
	ErrUnavailable = errors.New("user store not available")
)

// Domain sentinels the user store returns; the use-case passes them through.
var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
	ErrInvalidRole  = errors.New("invalid role")
	ErrLastAdmin    = errors.New("cannot demote or delete the last admin")
	// ErrInvalidDashboard wraps the reason a dashboard layout was refused.
	ErrInvalidDashboard = errors.New("invalid dashboard layout")
)

// MaxDashboardWidgets bounds a saved layout. The UI catalog is far smaller;
// the bound only stops a client from storing an unbounded list.
const MaxDashboardWidgets = 32

// dashboardWidgetID is the shape of a widget id. The catalog itself is the
// UI's: the store keeps ids, and the dashboard skips one it no longer knows.
var dashboardWidgetID = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,31}$`)

// ValidateDashboard checks a layout's shape: at most MaxDashboardWidgets ids,
// each well formed and none repeated. Order is the layout, so it is kept.
func ValidateDashboard(widgets []string) error {
	if len(widgets) > MaxDashboardWidgets {
		return fmt.Errorf("%w: %d widgets, at most %d", ErrInvalidDashboard, len(widgets), MaxDashboardWidgets)
	}
	seen := make(map[string]bool, len(widgets))
	for _, id := range widgets {
		if !dashboardWidgetID.MatchString(id) {
			return fmt.Errorf("%w: widget id %q is not a widget name", ErrInvalidDashboard, id)
		}
		if seen[id] {
			return fmt.Errorf("%w: widget %q appears twice", ErrInvalidDashboard, id)
		}
		seen[id] = true
	}
	return nil
}

// Repository is the user-store surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *database.DB in
// internal/app. Available reports whether a user store is wired (resolved per
// call so the use-case can degrade gracefully); the remaining methods are only
// invoked once availability is confirmed.
type Repository interface {
	Available() bool
	List(ctx context.Context) ([]*User, error)
	Create(ctx context.Context, username, hash, role string) (*User, error)
	Get(ctx context.Context, username string) (*User, error)
	UpdatePassword(ctx context.Context, username, hash string) error
	UpdateRole(ctx context.Context, username, role string) error
	Deactivate(ctx context.Context, username string) error
	Delete(ctx context.Context, username string) error
	// GetDashboard returns the user's saved widget ids and whether they have
	// ever saved a layout.
	GetDashboard(ctx context.Context, username string) ([]string, bool, error)
	SetDashboard(ctx context.Context, username string, widgets []string) error
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
func (s *Service) List(ctx context.Context) ([]*User, error) {
	if !s.repo.Available() {
		return nil, ErrUnavailable
	}
	return s.repo.List(ctx)
}

// Create inserts a new user. Returns ErrUnavailable when the store is not
// wired; domain errors (ErrUserExists) pass through verbatim.
func (s *Service) Create(ctx context.Context, username, hash, role string) (*User, error) {
	if !s.repo.Available() {
		return nil, ErrUnavailable
	}
	return s.repo.Create(ctx, username, hash, role)
}

// Get returns a user by username. Returns ErrUnavailable when the store is not
// wired; ErrUserNotFound passes through verbatim.
func (s *Service) Get(ctx context.Context, username string) (*User, error) {
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
// wired; ErrUserNotFound and ErrLastAdmin pass through.
func (s *Service) UpdateRole(ctx context.Context, username, role string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.UpdateRole(ctx, username, role)
}

// Deactivate marks the user inactive. Returns ErrUnavailable when the store is
// not wired; ErrUserNotFound passes through verbatim.
func (s *Service) Deactivate(ctx context.Context, username string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.Deactivate(ctx, username)
}

// Delete removes the user. Returns ErrUnavailable when the store is not wired;
// ErrUserNotFound and ErrLastAdmin pass through verbatim.
func (s *Service) Delete(ctx context.Context, username string) error {
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.Delete(ctx, username)
}

// GetDashboard returns the user's own dashboard layout and whether they have
// ever saved one, so the UI can tell "no layout yet" (show the default) from
// "saved an empty dashboard". Returns ErrUnavailable when the
// store is not wired; ErrUserNotFound passes through.
func (s *Service) GetDashboard(ctx context.Context, username string) ([]string, bool, error) {
	if !s.repo.Available() {
		return nil, false, ErrUnavailable
	}
	return s.repo.GetDashboard(ctx, username)
}

// SetDashboard replaces the user's dashboard layout after ValidateDashboard.
// Returns ErrUnavailable when the store is not wired, a wrapped
// ErrInvalidDashboard for a malformed layout, and ErrUserNotFound verbatim.
func (s *Service) SetDashboard(ctx context.Context, username string, widgets []string) error {
	if err := ValidateDashboard(widgets); err != nil {
		return err
	}
	if !s.repo.Available() {
		return ErrUnavailable
	}
	return s.repo.SetDashboard(ctx, username, widgets)
}
