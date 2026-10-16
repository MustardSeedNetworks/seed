package lifecycle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

// RefreshInterval is how often an enabled feed asks about the inventory.
// Cisco revises an EoX bulletin rarely; once a day is ample.
const RefreshInterval = 24 * time.Hour

// FeedSource is the operator's answer and the credentials to fetch with.
type FeedSource interface {
	// Credentials reports false unless the operator has turned the feed on;
	// the refresher then makes no request at all.
	Credentials(ctx context.Context) (Credentials, bool, error)
}

// Inventory lists the product IDs Seed has inventoried.
type Inventory func() []string

// FeedStatus is the outcome of the last refresh that ran.
type FeedStatus struct {
	LastAttempt time.Time
	LastSuccess time.Time
	LastError   string
	// Fetched is how many products the last successful refresh returned.
	Fetched int
}

// Refresher fetches EoX data for the inventory while the feed is on, and
// merges it into the stored table. Its embedded loop implements engine.Engine.
type Refresher struct {
	*outbound.Loop

	source    FeedSource
	inventory Inventory
	store     Store
	fetcher   *Fetcher
	logger    *slog.Logger

	mu     sync.Mutex
	status FeedStatus
}

// NewRefresher returns a refresher that checks the feed every interval and
// whenever [Refresher.Wake] is called.
func NewRefresher(
	source FeedSource, inventory Inventory, store Store, fetcher *Fetcher,
	interval time.Duration, logger *slog.Logger,
) *Refresher {
	r := &Refresher{source: source, inventory: inventory, store: store, fetcher: fetcher, logger: logger}
	r.Loop = outbound.NewLoop("lifecycle-feed", interval, r.refresh)
	return r
}

// Status returns the outcome of the last refresh.
func (r *Refresher) Status() FeedStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// refresh fetches and stores once, if the feed is on and there is anything
// to ask about. Off, it returns before any request is built.
func (r *Refresher) refresh(ctx context.Context) {
	creds, on, err := r.source.Credentials(ctx)
	if err != nil {
		r.logger.WarnContext(ctx, "EoX feed settings unreadable", "error", err)
		return
	}
	if !on {
		return
	}
	productIDs := r.inventory()
	if len(productIDs) == 0 {
		return
	}

	fetched, err := r.fetch(ctx, creds, productIDs)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status.LastAttempt = time.Now()
	if err != nil {
		r.status.LastError = err.Error()
		r.logger.WarnContext(ctx, "EoX feed refresh failed", "error", err)
		return
	}
	r.status.LastSuccess = r.status.LastAttempt
	r.status.LastError = ""
	r.status.Fetched = fetched
	r.logger.InfoContext(ctx, "EoX feed refreshed", "products", len(productIDs), "records", fetched)
}

func (r *Refresher) fetch(ctx context.Context, creds Credentials, productIDs []string) (int, error) {
	fetched, err := r.fetcher.Fetch(ctx, creds, productIDs)
	if err != nil {
		return 0, err
	}
	if mergeErr := r.store.MergeEoX(ctx, fetched); mergeErr != nil {
		return 0, mergeErr
	}
	return fetched.Len(), nil
}
