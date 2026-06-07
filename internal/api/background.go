package api

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
	wificapture "github.com/MustardSeedNetworks/seed/internal/wifi/capture"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// BackgroundComponents holds the long-lived components that own real background
// lifecycle (Start/Stop). Stateless request/response logic lives in the handlers
// and the api service groupings, built directly from the feature packages; only
// components that own background work belong here: the report scheduler
// (internal/reporting), the Wi-Fi airspace visibility loop
// (internal/wifi/visibility), and the monitor-mode capture producer that feeds it
// (internal/wifi/capture).
type BackgroundComponents struct {
	Reporting      *reporting.Service
	WiFiVisibility *visibility.Service
	WiFiCapture    *wificapture.Capture
}

// Start initializes and starts all background components. The visibility loop
// starts before the capture producer that feeds it.
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
	if b.WiFiCapture != nil {
		if err := b.WiFiCapture.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all background components. Capture stops before the
// visibility loop it feeds (stop producing, then consuming).
func (b *BackgroundComponents) Stop() error {
	if b.WiFiCapture != nil {
		_ = b.WiFiCapture.Stop()
	}
	if b.Reporting != nil {
		_ = b.Reporting.Stop()
	}
	if b.WiFiVisibility != nil {
		_ = b.WiFiVisibility.Stop()
	}
	return nil
}
