package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/reporting"
	"github.com/krisarmstrong/seed/internal/wifi/visibility"
)

// BackgroundComponents holds the long-lived components that own real background
// lifecycle (Start/Stop). Stateless request/response logic lives in the handlers
// and the api service groupings, built directly from the feature packages; only
// components that own background work belong here: the report scheduler
// (internal/reporting) and the Wi-Fi airspace visibility loop
// (internal/wifi/visibility).
type BackgroundComponents struct {
	Reporting      *reporting.Service
	WiFiVisibility *visibility.Service
}

// Start initializes and starts all background components.
func (b *BackgroundComponents) Start(ctx context.Context) error {
	if b.Reporting != nil {
		if err := b.Reporting.Start(ctx); err != nil {
			return err
		}
	}
	if b.WiFiVisibility != nil {
		if err := b.WiFiVisibility.Start(ctx); err != nil {
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
	if b.WiFiVisibility != nil {
		_ = b.WiFiVisibility.Stop()
	}
	return nil
}
