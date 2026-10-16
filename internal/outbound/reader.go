package outbound

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Source is a vendor API integration's answer and the credentials to read
// with.
type Source interface {
	// Credentials reports false unless the operator has turned the
	// integration on; the reader then makes no request at all.
	Credentials(ctx context.Context) (Credentials, bool, error)
}

// Status is the outcome of the reads since the daemon started.
type Status struct {
	LastAttempt time.Time
	LastSuccess time.Time
	LastError   string
}

// Reader reads a vendor API's device state while the integration is on and
// holds the last successful read. Its embedded loop implements engine.Engine.
type Reader[T any] struct {
	*Loop

	source Source
	read   func(context.Context, Credentials) ([]T, error)
	logger *slog.Logger

	mu     sync.Mutex
	status Status
	items  []T
}

// NewReader returns a reader named name that calls read every interval and
// whenever [Loop.Wake] is called, as long as source says the integration is
// on.
func NewReader[T any](
	name string, source Source, read func(context.Context, Credentials) ([]T, error),
	interval time.Duration, logger *slog.Logger,
) *Reader[T] {
	r := &Reader[T]{source: source, read: read, logger: logger.With("integration", name)}
	r.Loop = NewLoop(name, interval, r.refresh)
	return r
}

// Status returns the outcome of the reads so far.
func (r *Reader[T]) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// Items returns what the last successful read returned, or nil when there has
// been none since the operator last turned the integration on.
func (r *Reader[T]) Items() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.items
}

// refresh reads once if the integration is on. Off, it returns before any
// request is built and forgets what an earlier read returned.
func (r *Reader[T]) refresh(ctx context.Context) {
	creds, on, err := r.source.Credentials(ctx)
	if err != nil {
		r.logger.WarnContext(ctx, "Vendor API settings unreadable", "error", err)
		return
	}
	if !on {
		r.mu.Lock()
		r.items = nil
		r.mu.Unlock()
		return
	}

	items, err := r.read(ctx, creds)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status.LastAttempt = time.Now()
	if err != nil {
		r.status.LastError = err.Error()
		r.logger.WarnContext(ctx, "Vendor API read failed", "error", err)
		return
	}
	r.status.LastSuccess = r.status.LastAttempt
	r.status.LastError = ""
	r.items = items
	r.logger.InfoContext(ctx, "Vendor API device state read", "devices", len(items))
}
