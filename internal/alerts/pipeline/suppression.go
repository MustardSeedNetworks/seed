package pipeline

// suppression.go isolates the per-(rule, entity) "don't re-fire
// inside this window" check both alert pipelines use. Operators
// see one alert per rule+entity per suppression window rather than
// a flood when the same condition persists across multiple scans.
//
// Two implementations:
//
//   - inMemorySuppressionStore — process-local map, lost on restart.
//     Backward-compatible default for tests that don't wire a DB.
//
//   - DBSuppressionStore       — sqlite-backed, restart-safe (#1380).
//     Production wiring; server.go passes db.AlertSuppressions() into
//     each pipeline's config.
//
// Both implementations honor the same narrow contract so the
// pipelines stay agnostic to where the marks live.

import (
	"context"
	"sync"
	"time"
)

// suppressionStore is the narrow surface alert pipelines need.
// Both the in-memory and DB-backed implementations satisfy it.
type suppressionStore interface {
	// IsSuppressed reports whether fingerprint has a fire mark that
	// has not yet expired.
	IsSuppressed(ctx context.Context, fingerprint string, now time.Time) (bool, error)

	// Mark records that fingerprint just fired and should be
	// suppressed until "until".
	Mark(ctx context.Context, fingerprint, ruleID, entityKey string, until time.Time) error
}

// inMemorySuppressionStore is the legacy backend — same behavior the
// pipelines had before #1380, kept for tests that don't wire a DB.
type inMemorySuppressionStore struct {
	mu      sync.Mutex
	emitted map[string]time.Time
}

func newInMemorySuppressionStore() *inMemorySuppressionStore {
	return &inMemorySuppressionStore{emitted: make(map[string]time.Time)}
}

func (s *inMemorySuppressionStore) IsSuppressed(
	_ context.Context, fingerprint string, now time.Time,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.emitted[fingerprint]
	if !ok {
		return false, nil
	}
	return now.Before(until), nil
}

func (s *inMemorySuppressionStore) Mark(
	_ context.Context, fingerprint, _, _ string, until time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitted[fingerprint] = until
	// Lazy eviction at the noticeable-size threshold mirrors the
	// pre-#1380 behavior so the map stays bounded under sustained
	// rule fire-rate.
	const evictThreshold = 4096
	if len(s.emitted) <= evictThreshold {
		return nil
	}
	for k, v := range s.emitted {
		if v.Before(until.Add(-time.Hour)) {
			delete(s.emitted, k)
		}
	}
	return nil
}

// NewDBSuppressionStore wires a database-backed suppression store
// by returning the repo directly — *database.AlertSuppressionsRepository
// already satisfies suppressionStore. Exported as a constructor so
// server.go has a stable seam if we ever need to layer behavior
// (metrics, retries) above the raw repo.
func NewDBSuppressionStore(repo suppressionStore) suppressionStore {
	return repo
}
