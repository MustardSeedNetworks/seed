package lifecycle

import (
	"context"
	"log/slog"
	"sync"
	"time"
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
// merges it into the stored table. It implements engine.Engine.
type Refresher struct {
	source    FeedSource
	inventory Inventory
	store     Store
	fetcher   *Fetcher
	interval  time.Duration
	logger    *slog.Logger
	wake      chan struct{}

	mu      sync.Mutex
	status  FeedStatus
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewRefresher returns a refresher that checks the feed every interval and
// whenever [Refresher.Wake] is called.
func NewRefresher(
	source FeedSource, inventory Inventory, store Store, fetcher *Fetcher,
	interval time.Duration, logger *slog.Logger,
) *Refresher {
	return &Refresher{
		source: source, inventory: inventory, store: store, fetcher: fetcher,
		interval: interval, logger: logger, wake: make(chan struct{}, 1),
	}
}

// Name implements engine.Engine.
func (*Refresher) Name() string { return "lifecycle-feed" }

// Start checks the feed once now and then on every tick or wake. It is a
// no-op when already started.
func (r *Refresher) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.started = true
	r.wg.Add(1)
	go r.loop(loopCtx)
	return nil
}

// Stop ends the loop and waits, up to ctx's deadline, for an in-flight
// refresh to finish.
func (r *Refresher) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	r.mu.Unlock()
	cancel()

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	r.mu.Lock()
	r.started = false
	r.mu.Unlock()
	return nil
}

// Wake asks for a check now, after the operator changed the feed settings.
func (r *Refresher) Wake() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// Status returns the outcome of the last refresh.
func (r *Refresher) Status() FeedStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Refresher) loop(ctx context.Context) {
	defer r.wg.Done()
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		r.refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-r.wake:
		}
	}
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
