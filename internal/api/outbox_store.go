package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
)

// outboxRetention is how long a delivered (published) outbox row is kept before
// the maintenance loop prunes it. Delivered rows have no further use; the window
// is a short grace period for debugging, mirroring jobs retention.
const outboxRetention = time.Hour

// dbOutboxStore adapts the durable database.OutboxRepository to the outbox.Store
// seam the relay drains through (ADR-0017). It is the composition-root bridge
// between internal/platform/outbox (which must not know about persistence) and
// the outbox table. The repository keys rows by the int64 autoincrement id; the
// relay's dedup token is an opaque string, so the adapter is the one place that
// translates between the two.
type dbOutboxStore struct {
	repo *database.OutboxRepository
}

// newDBOutboxStore builds the adapter over db's outbox repository.
func newDBOutboxStore(db *database.DB) *dbOutboxStore {
	return &dbOutboxStore{repo: db.Outbox()}
}

// FetchUnpublished returns pending rows as relay records, stringifying the id.
func (s *dbOutboxStore) FetchUnpublished(ctx context.Context, limit int) ([]outbox.Record, error) {
	recs, err := s.repo.FetchUnpublished(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]outbox.Record, len(recs))
	for i, r := range recs {
		out[i] = outbox.Record{
			ID:      strconv.FormatInt(r.ID, 10),
			Topic:   r.Topic,
			Payload: r.Payload,
		}
	}
	return out, nil
}

// MarkPublished parses the relay's string ids back to the repository's int64 keys.
func (s *dbOutboxStore) MarkPublished(ctx context.Context, ids []string) error {
	intIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		n, parseErr := strconv.ParseInt(id, 10, 64)
		if parseErr != nil {
			return fmt.Errorf("outbox store: invalid id %q: %w", id, parseErr)
		}
		intIDs = append(intIDs, n)
	}
	return s.repo.MarkPublished(ctx, intIDs)
}

// DeletePublishedBefore prunes delivered rows older than cutoff.
func (s *dbOutboxStore) DeletePublishedBefore(ctx context.Context, cutoff time.Time) (int, error) {
	n, err := s.repo.DeletePublishedBefore(ctx, cutoff)
	return int(n), err
}
