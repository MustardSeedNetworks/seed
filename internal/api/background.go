package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/reporting"
)

// BackgroundComponents holds the long-lived components that own real background
// lifecycle (Start/Stop). Stateless request/response logic lives in the handlers
// and the api service groupings, built directly from the feature packages; only
// components that own background work belong here. Currently that is the report
// scheduler (internal/reporting).
type BackgroundComponents struct {
	Reporting *reporting.Service
}

// Start initializes and starts all background components.
func (b *BackgroundComponents) Start(ctx context.Context) error {
	if b.Reporting != nil {
		if err := b.Reporting.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all background components.
func (b *BackgroundComponents) Stop() error {
	if b.Reporting != nil {
		_ = b.Reporting.Stop()
	}
	return nil
}
