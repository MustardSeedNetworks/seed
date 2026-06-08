// Package outbox is seed's transactional-outbox relay (ADR-0017): the durability
// layer for the in-process event bus (platform/events).
//
// A producer that needs cross-restart durable delivery writes an outbox row in
// the SAME database transaction as its domain change (database.OutboxRepository
// .Enqueue runs on the caller's *sql.Tx). The row and the domain row commit or
// roll back together — a rolled-back operation never leaves an event behind. A
// Relay then drains pending rows post-commit, republishes each onto the bus as a
// Message on its original topic, and marks the batch published.
//
// Delivery is at-least-once: the relay publishes first and marks second, so a
// crash between the two replays the row on the next drain, and rows enqueued by a
// prior process are redelivered on Start. Consumers must therefore be idempotent,
// keyed on Message.ID (the stable outbox row id) — Dedupe wraps a handler with a
// bounded most-recently-seen window to make that a one-liner.
//
// This package is pure: it knows the Store port and the bus, never database/sql.
// The composition root supplies the Store adapter over the real repository.
package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/platform/events"
)

const (
	// defaultBatchSize bounds how many pending rows one Drain publishes — a
	// large backlog drains across several ticks rather than one long stall.
	defaultBatchSize = 100
	// defaultInterval is the relay's poll cadence. Short enough that durable
	// events feel live, long enough that an empty table costs almost nothing.
	defaultInterval = 2 * time.Second
)

// Record is one pending outbox row handed to the relay by the Store. ID is the
// stable, monotonic dedup token; Payload is the opaque serialized event.
type Record struct {
	ID      string
	Topic   string
	Payload []byte
}

// Store is the relay's persistence seam (satisfied in the composition root over
// database.OutboxRepository). It exposes only what the relay needs: read the
// backlog, mark a batch delivered, and prune delivered rows. Enqueue is
// deliberately absent — it must run on the producer's transaction, so it lives on
// the repository, not here.
type Store interface {
	// FetchUnpublished returns up to limit pending rows in insert order.
	FetchUnpublished(ctx context.Context, limit int) ([]Record, error)
	// MarkPublished stamps the given rows delivered (idempotent per id).
	MarkPublished(ctx context.Context, ids []string) error
	// DeletePublishedBefore prunes delivered rows older than cutoff.
	DeletePublishedBefore(ctx context.Context, cutoff time.Time) (int, error)
}

// Message is the bus event the relay republishes for an outbox row. It carries
// the stable ID (the dedup key) and the raw Payload; its Topic is the row's
// original topic, so an ordinary bus subscriber receives it transparently.
type Message struct {
	ID         string
	EventTopic string
	Payload    []byte
}

// Topic implements events.Event, routing the message on its original topic.
func (m Message) Topic() string { return m.EventTopic }

// Relay drains the outbox to the bus. The zero value is not usable; call
// NewRelay. A Relay is safe for concurrent use.
type Relay struct {
	store    Store
	bus      *events.Bus
	logger   *slog.Logger
	batch    int
	interval time.Duration

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

// Option tunes a Relay.
type Option func(*Relay)

// WithBatchSize sets the maximum rows published per drain (<= 0 keeps the default).
func WithBatchSize(n int) Option {
	return func(r *Relay) {
		if n > 0 {
			r.batch = n
		}
	}
}

// WithInterval sets the poll cadence (<= 0 keeps the default).
func WithInterval(d time.Duration) Option {
	return func(r *Relay) {
		if d > 0 {
			r.interval = d
		}
	}
}

// NewRelay builds a relay over store publishing to bus. store, bus and logger
// must be non-nil.
func NewRelay(store Store, bus *events.Bus, logger *slog.Logger, opts ...Option) *Relay {
	r := &Relay{
		store:    store,
		bus:      bus,
		logger:   logger,
		batch:    defaultBatchSize,
		interval: defaultInterval,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Drain publishes one batch of pending rows to the bus and marks them published,
// returning how many it published. It publishes before marking: a failure to
// mark (or a crash in between) leaves the rows pending so the next Drain replays
// them — at-least-once. A fetch or mark error is returned to the caller after the
// publishes already happened.
func (r *Relay) Drain(ctx context.Context) (int, error) {
	recs, err := r.store.FetchUnpublished(ctx, r.batch)
	if err != nil {
		return 0, fmt.Errorf("outbox: fetch unpublished: %w", err)
	}
	if len(recs) == 0 {
		return 0, nil
	}

	ids := make([]string, 0, len(recs))
	for _, rec := range recs {
		r.bus.Publish(Message{ID: rec.ID, EventTopic: rec.Topic, Payload: rec.Payload})
		ids = append(ids, rec.ID)
	}
	if markErr := r.store.MarkPublished(ctx, ids); markErr != nil {
		return len(recs), fmt.Errorf("outbox: mark published: %w", markErr)
	}
	return len(recs), nil
}

// Cleanup prunes delivered rows older than retention, returning the count
// removed. Driven by the maintenance loop, like jobs retention.
func (r *Relay) Cleanup(ctx context.Context, retention time.Duration) (int, error) {
	n, err := r.store.DeletePublishedBefore(ctx, time.Now().UTC().Add(-retention))
	if err != nil {
		return 0, fmt.Errorf("outbox: cleanup: %w", err)
	}
	return n, nil
}

// Start drains the backlog once (the across-restart replay) and then polls on the
// configured interval until Stop. It is idempotent: a second Start while running
// is a no-op.
func (r *Relay) Start(ctx context.Context) {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	r.started = true
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.done = make(chan struct{})
	r.mu.Unlock()

	if _, err := r.Drain(runCtx); err != nil {
		r.logger.ErrorContext(runCtx, "outbox initial drain failed", "err", err)
	}
	go r.loop(runCtx)
}

// Stop halts the poll loop and waits for it to exit. Idempotent.
func (r *Relay) Stop() {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	r.started = false
	cancel := r.cancel
	done := r.done
	r.cancel = nil
	r.mu.Unlock()

	cancel()
	<-done
}

// loop polls the store until ctx is cancelled.
func (r *Relay) loop(ctx context.Context) {
	defer close(r.done)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.Drain(ctx); err != nil {
				r.logger.ErrorContext(ctx, "outbox drain failed", "err", err)
			}
		}
	}
}

// Deduper is a bounded most-recently-seen set of Message IDs for idempotent
// consumers. It is safe for concurrent use. Size it above the relay's redelivery
// horizon: an id evicted once capacity is exceeded is forgotten and would be
// reprocessed.
type Deduper struct {
	mu       sync.Mutex
	capacity int
	seen     map[string]struct{}
	order    []string
}

// NewDeduper returns a Deduper retaining the most recent capacity ids (a
// capacity <= 0 is treated as 1).
func NewDeduper(capacity int) *Deduper {
	if capacity <= 0 {
		capacity = 1
	}
	return &Deduper{
		capacity: capacity,
		seen:     make(map[string]struct{}, capacity),
	}
}

// Seen records id and reports whether it was already in the window. The first
// call for an id returns false (process it); a repeat within the window returns
// true (skip it). Evicts the oldest id once capacity is exceeded.
func (d *Deduper) Seen(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[id]; ok {
		return true
	}
	if len(d.order) >= d.capacity {
		oldest := d.order[0]
		d.order = d.order[1:]
		delete(d.seen, oldest)
	}
	d.seen[id] = struct{}{}
	d.order = append(d.order, id)
	return false
}

// Dedupe wraps h so a redelivered Message (same ID, within d's window) is
// skipped. Non-Message events pass through untouched.
func Dedupe(d *Deduper, h events.Handler) events.Handler {
	return func(ctx context.Context, ev events.Event) {
		if m, ok := ev.(Message); ok && d.Seen(m.ID) {
			return
		}
		h(ctx, ev)
	}
}
