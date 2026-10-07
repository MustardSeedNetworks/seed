package api

import (
	"context"
	"log/slog"

	"github.com/MustardSeedNetworks/seed/internal/app"
)

// dbJobIdempotency is the durable Idempotency-Key store backing POST /jobs
// (Phase 5c-4), satisfying jobIdempotencyStore over the app job store.
// Best-effort, matching the in-memory cache: a backend error degrades a check to
// idemMiss (create afresh) and a store failure is logged, never surfaced.
type dbJobIdempotency struct {
	jobs   *app.JobStore
	logger *slog.Logger
}

func newDBJobIdempotency(store *app.JobStore, logger *slog.Logger) *dbJobIdempotency {
	return &dbJobIdempotency{jobs: store, logger: logger}
}

func (d *dbJobIdempotency) check(ctx context.Context, key string, req CreateJobRequest) idemResult {
	jobID, storedHash, found, err := d.jobs.LookupIdempotency(ctx, key)
	if err != nil {
		d.logger.ErrorContext(ctx, "idempotency lookup failed; treating as miss", "err", err)
		return idemResult{kind: idemMiss}
	}
	if !found {
		return idemResult{kind: idemMiss}
	}
	if storedHash != requestHash(req) {
		return idemResult{kind: idemConflict}
	}
	return idemResult{id: jobID, kind: idemHit}
}

func (d *dbJobIdempotency) store(ctx context.Context, key string, req CreateJobRequest, jobID string) {
	if err := d.jobs.RecordIdempotency(ctx, key, requestHash(req), jobID); err != nil {
		d.logger.ErrorContext(ctx, "idempotency record failed", "err", err)
	}
}
