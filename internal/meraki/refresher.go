package meraki

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

// RefreshInterval is how often an enabled integration reads device state.
// The dashboard allows ten requests a second per organization; a read costs
// three requests per organization plus paging.
const RefreshInterval = 5 * time.Minute

// Source is the operator's answer and the credentials to read with.
type Source interface {
	// Credentials reports false unless the operator has turned the
	// integration on; the refresher then makes no request at all.
	Credentials(ctx context.Context) (Credentials, bool, error)
}

// Snapshot is the outcome of the refreshes since the daemon started.
type Snapshot struct {
	LastAttempt time.Time
	LastSuccess time.Time
	LastError   string
	// Devices is what the last successful read returned. It is dropped when
	// the operator turns the integration off.
	Devices []Device
}

// Refresher reads device state while the integration is on. Its embedded loop
// implements engine.Engine.
type Refresher struct {
	*outbound.Loop

	source Source
	client *Client
	logger *slog.Logger

	mu       sync.Mutex
	snapshot Snapshot
}

// NewRefresher returns a refresher that reads every interval and whenever
// [Refresher.Wake] is called.
func NewRefresher(source Source, client *Client, interval time.Duration, logger *slog.Logger) *Refresher {
	r := &Refresher{source: source, client: client, logger: logger}
	r.Loop = outbound.NewLoop("meraki", interval, r.refresh)
	return r
}

// Snapshot returns the last read.
func (r *Refresher) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot
}

// refresh reads once if the integration is on. Off, it returns before any
// request is built and forgets what an earlier read returned.
func (r *Refresher) refresh(ctx context.Context) {
	creds, on, err := r.source.Credentials(ctx)
	if err != nil {
		r.logger.WarnContext(ctx, "Meraki settings unreadable", "error", err)
		return
	}
	if !on {
		r.mu.Lock()
		r.snapshot.Devices = nil
		r.mu.Unlock()
		return
	}

	devices, err := r.client.Devices(ctx, creds)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.LastAttempt = time.Now()
	if err != nil {
		r.snapshot.LastError = err.Error()
		r.logger.WarnContext(ctx, "Meraki read failed", "error", err)
		return
	}
	r.snapshot.LastSuccess = r.snapshot.LastAttempt
	r.snapshot.LastError = ""
	r.snapshot.Devices = devices
	r.logger.InfoContext(ctx, "Meraki device state read", "devices", len(devices))
}
