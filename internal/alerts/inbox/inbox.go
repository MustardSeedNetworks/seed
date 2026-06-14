// Package inbox is the alert-inbox use-case (ADR-0020, WS-A8): the /api/v1/alerts
// endpoints' application service over a narrow Repository port, so the transport
// layer depends on a use-case instead of reaching into the database directly. The
// Repository is satisfied by an adapter in the composition root over the alert
// repository (the store the NMS pipelines write to).
package inbox

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// ErrUnavailable is returned when the store is not wired (handler → 503).
var ErrUnavailable = errors.New("inbox: store unavailable")

// Repository is the alert-store surface the use-case needs: list the alerts the
// pipelines have written, and mark one acknowledged or resolved.
type Repository interface {
	List(ctx context.Context, opts alerts.ListOptions) ([]*alerts.Alert, error)
	Acknowledge(ctx context.Context, id int64, username string) error
	Resolve(ctx context.Context, id int64) error
}

// Service is the alert-inbox use-case.
type Service struct {
	repo Repository
}

// NewService builds the use-case over its Repository port.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// List returns the alerts matching opts.
func (s *Service) List(ctx context.Context, opts alerts.ListOptions) ([]*alerts.Alert, error) {
	return s.repo.List(ctx, opts)
}

// Acknowledge marks alert id acknowledged by username.
func (s *Service) Acknowledge(ctx context.Context, id int64, username string) error {
	return s.repo.Acknowledge(ctx, id, username)
}

// Resolve marks alert id resolved.
func (s *Service) Resolve(ctx context.Context, id int64) error {
	return s.repo.Resolve(ctx, id)
}
