package api

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	wificapture "github.com/MustardSeedNetworks/seed/internal/wifi/capture"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// BackgroundComponents holds the long-lived components that own real background
// lifecycle (Start/Stop). Stateless request/response logic lives in the handlers
// and the api service groupings, built directly from the feature packages; only
// components that own background work belong here: the report scheduler
// (internal/reporting), the Wi-Fi airspace visibility loop
// (internal/wifi/visibility), the monitor-mode capture producer that feeds it
// (internal/wifi/capture), and the transactional-outbox relay that drains durable
// events to the bus (internal/platform/outbox, ADR-0017).
//
// The outbox relay is created during server init (it needs the event bus, which
// is built there) rather than in the cmd-layer constructor, so it is wired onto
// this struct after construction.
type BackgroundComponents struct {
	Reporting      *reporting.Service
	WiFiVisibility *visibility.Service
	WiFiCapture    *wificapture.Capture
	Outbox         *outbox.Relay
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
	if b.Outbox != nil {
		// The relay owns its own lifecycle context (it polls until Stop), so it
		// is detached from the start ctx; Stop tears it down on shutdown.
		b.Outbox.Start(context.WithoutCancel(ctx))
	}
	return nil
}

// Stop gracefully shuts down all background components. Capture stops before the
// visibility loop it feeds (stop producing, then consuming).
func (b *BackgroundComponents) Stop() error {
	if b.Outbox != nil {
		b.Outbox.Stop()
	}
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
