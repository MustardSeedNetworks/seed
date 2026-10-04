// Package inbox is the alert-inbox use-case (ADR-0020, WS-A8): the /api/v1/alerts
// endpoints' application service over a narrow Repository port, so the transport
// layer depends on a use-case instead of reaching into the database directly. The
// Repository is satisfied by an adapter in the composition root over the alert
// repository (the store the NMS pipelines write to).
package inbox

import (
	"cmp"
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
)

// ErrUnavailable is returned when the store is not wired (handler → 503).
var ErrUnavailable = errors.New("inbox: store unavailable")

// Repository is the alert-store surface the use-case needs: list the alerts the
// pipelines have written and the alerts each one caused, and mark one
// acknowledged or resolved.
type Repository interface {
	List(ctx context.Context, opts alerts.ListOptions) ([]*alerts.Alert, error)
	ListEffects(ctx context.Context, causeIDs []int64) ([]*alerts.Alert, error)
	// DeviceNames maps an alert Source that is a polling target id to the
	// target's name.
	DeviceNames(ctx context.Context) (map[string]string, error)
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

// Narratives explains each alert in page that heads a cluster, keyed by alert
// id. The effects are read by cause rather than taken from the page, so a
// cause still names an effect the page's filter or pagination left out.
func (s *Service) Narratives(
	ctx context.Context, page []*alerts.Alert,
) (map[int64]narrative.Narrative, error) {
	var causeIDs []int64
	for _, a := range page {
		if a.RootCauseID == nil {
			causeIDs = append(causeIDs, a.ID)
		}
	}
	effects, err := s.repo.ListEffects(ctx, causeIDs)
	if err != nil {
		return nil, err
	}
	names, err := s.repo.DeviceNames(ctx)
	if err != nil {
		return nil, err
	}
	byCause := make(map[int64][]*alerts.Alert)
	for _, e := range effects {
		byCause[*e.RootCauseID] = append(byCause[*e.RootCauseID], e)
	}

	out := make(map[int64]narrative.Narrative)
	for _, a := range page {
		// A source that is not a polling target (a syslog sender's address)
		// is already the best name Seed has for it.
		device := cmp.Or(names[a.Source], a.Source)
		if n, ok := narrative.Explain(a, byCause[a.ID], device); ok {
			out[a.ID] = n
		}
	}
	return out, nil
}

// Acknowledge marks alert id acknowledged by username.
func (s *Service) Acknowledge(ctx context.Context, id int64, username string) error {
	return s.repo.Acknowledge(ctx, id, username)
}

// Resolve marks alert id resolved.
func (s *Service) Resolve(ctx context.Context, id int64) error {
	return s.repo.Resolve(ctx, id)
}
