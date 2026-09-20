package api

import (
	"context"

	"github.com/MustardSeedNetworks/foundation/pkg/supervise"

	"github.com/MustardSeedNetworks/seed/internal/logging"
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

	group *supervise.Group
}

// backgroundRestarts is how many times a faulted background loop is restarted
// before the group gives up on it. All four loops are resumable — tickers and a
// frame reader, none of them carrying state between iterations — so a transient
// fault should not cost the daemon its reporting, its airspace view or its
// durable-event delivery for the rest of its life. A loop that faults every
// time stops after this many tries rather than spinning, leaving the one line
// the supervisor logs.
const backgroundRestarts = 3

// Start runs every configured component as a supervised worker (#2748). Before
// this each component spawned its own goroutine, so a panic anywhere in a
// background loop took the daemon down with it; now the supervisor recovers it,
// logs one line naming the worker, and restarts the loop.
//
// Registration order is the reverse of the stop order the components need,
// because supervise.Group.Stop cancels workers last-registered-first: capture
// (the producer) stops before the visibility loop it feeds, and the outbox
// relay — the last thing still able to deliver an event — stops first.
func (b *BackgroundComponents) Start(ctx context.Context) error {
	// Loading is synchronous and still fails startup: an unreadable template
	// directory or schedule table is a misconfigured install, not a fault to
	// recover from.
	if b.Reporting != nil {
		if err := b.Reporting.Load(ctx); err != nil {
			return err
		}
	}

	group := supervise.New(logging.GetLogger())
	if b.WiFiVisibility != nil {
		group.Add("wifi-visibility", supervise.RestartN(backgroundRestarts), b.WiFiVisibility.Run)
	}
	if b.Reporting != nil {
		group.Add("reporting", supervise.RestartN(backgroundRestarts), b.Reporting.Run)
	}
	if b.WiFiCapture != nil {
		group.Add("wifi-capture", supervise.RestartN(backgroundRestarts), b.WiFiCapture.Run)
	}
	if b.Outbox != nil {
		group.Add("outbox", supervise.RestartN(backgroundRestarts), b.Outbox.Run)
	}
	b.group = group
	group.Start(ctx)
	return nil
}

// Stop cancels the supervised workers in reverse registration order and blocks
// until each has exited. It runs after the HTTP drain (slice 1 of #2748), so a
// request in flight still reaches live components. Safe when Start was never
// called.
func (b *BackgroundComponents) Stop() error {
	group := b.group
	if group == nil {
		return nil
	}
	b.group = nil
	// The caller's deadline is the shutdown context cmd/seed applies around
	// this call; a bound of its own would only invent a second one to disagree
	// with it.
	if err := group.Stop(context.Background()); err != nil {
		logging.GetLogger().Warn("background components did not stop cleanly", "error", err.Error())
	}
	return nil
}
