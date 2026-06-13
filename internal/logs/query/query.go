// Package query is the log-query use-case (ADR-0020 clean-hexagonal, WS-A11b): it
// owns the "persisted store first, fall back to the in-memory ring buffer"
// decision the /api/logs/query handler used to carry inline. Both log sources are
// reached through consumer-defined ports (Store for the database, Buffer for the
// process-local broadcaster), satisfied by adapters in the composition root. The
// filter/paginate logic for the memory path lives here so the transport layer only
// parses the request and encodes the result.
package query

import (
	"context"
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// ErrNoSource is returned when neither the persisted store nor the memory buffer
// is available to serve a query (mapped to 503 by the transport layer).
var ErrNoSource = errors.New("logquery: no log source available")

// Params is the parsed, transport-neutral log-query filter.
type Params struct {
	Levels     []string
	Layers     []string
	Components []string
	Search     string
	Limit      int
	Offset     int
}

// Store is the persisted-log source. Available reports whether a database is
// wired; List returns the entries matching params (already filtered/paginated by
// the database). A List error makes the use-case fall back to the memory buffer.
type Store interface {
	Available() bool
	List(ctx context.Context, p Params) ([]*logging.LogEntry, error)
}

// Buffer is the in-memory ring buffer of recent logs. Available reports whether
// the broadcaster is initialized; All returns every retained entry (newest-last),
// which the use-case filters and paginates itself.
type Buffer interface {
	Available() bool
	All() []*logging.LogEntry
}

// Result is a log-query outcome: the page of entries plus the total matched
// before pagination (for the persisted path the total is the page length, the
// original handler's behavior).
type Result struct {
	Entries    []*logging.LogEntry
	TotalCount int
}

// Service is the log-query use-case.
type Service struct {
	store  Store
	buffer Buffer
}

// NewService builds the use-case over its log-source ports.
func NewService(store Store, buffer Buffer) *Service {
	return &Service{store: store, buffer: buffer}
}

// Query returns the page of logs for params, preferring the persisted store and
// falling back to the memory buffer on an absent/erroring store. Returns
// ErrNoSource when neither source can serve.
func (s *Service) Query(ctx context.Context, p Params) (Result, error) {
	if s.store.Available() {
		if entries, err := s.store.List(ctx, p); err == nil {
			return Result{Entries: entries, TotalCount: len(entries)}, nil
		}
		// Fall through to the memory buffer on a store error.
	}

	if !s.buffer.Available() {
		return Result{}, ErrNoSource
	}

	all := s.buffer.All()
	filtered := make([]*logging.LogEntry, 0, len(all))
	for _, entry := range all {
		if matches(entry, p) {
			filtered = append(filtered, entry)
		}
	}
	total := len(filtered)
	return Result{Entries: paginate(filtered, p.Offset, p.Limit), TotalCount: total}, nil
}

// matches reports whether entry passes every set filter in p.
func matches(entry *logging.LogEntry, p Params) bool {
	if len(p.Levels) > 0 && !containsFold(p.Levels, entry.Level) {
		return false
	}
	if len(p.Layers) > 0 && !containsFold(p.Layers, entry.Layer) {
		return false
	}
	if len(p.Components) > 0 && !containsFold(p.Components, entry.Component) {
		return false
	}
	if p.Search != "" && !strings.Contains(strings.ToLower(entry.Message), p.Search) {
		return false
	}
	return true
}

// paginate returns the [offset, offset+limit) window of logs, or nil past the end.
func paginate(logs []*logging.LogEntry, offset, limit int) []*logging.LogEntry {
	if offset >= len(logs) {
		return nil
	}
	end := min(offset+limit, len(logs))
	return logs[offset:end]
}

func containsFold(slice []string, target string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}
