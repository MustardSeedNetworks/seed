package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// dbJobStore adapts the durable database.JobRepository to the jobs.Store seam
// the runner write-throughs (ADR-0005, Phase 5c). It is the composition-root
// bridge between platform/jobs (which must not know about persistence) and the
// jobs table: it marshals the job's result to JSON and derives the
// created/updated/completed timestamps the in-memory Job snapshot does not carry.
type dbJobStore struct {
	repo *database.JobRepository
	now  func() time.Time
}

// newDBJobStore builds the adapter over db's jobs repository.
func newDBJobStore(db *database.DB) *dbJobStore {
	return &dbJobStore{
		repo: db.Jobs(),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// Save write-throughs a snapshot. created_at is set on every call but kept by
// the repository's upsert only on first insert; completed_at is stamped once the
// job reaches a terminal state and left NULL while active.
func (s *dbJobStore) Save(ctx context.Context, j jobs.Job) error {
	now := s.now()
	rec := &database.JobRecord{
		ID:        j.ID,
		Kind:      j.Kind,
		State:     string(j.State),
		Progress:  j.Progress,
		Error:     j.Err,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if j.Result != nil {
		if b, err := json.Marshal(j.Result); err == nil {
			rec.ResultJSON = string(b)
		}
	}
	if jobStateTerminal(j.State) {
		rec.CompletedAt = now
	}
	return s.repo.Save(ctx, rec)
}

// Load returns the persisted job. A missing row is (zero, false, nil), not an
// error, so the runner's Get fallback treats it as a plain miss. The stored
// result comes back as [json.RawMessage] — the HTTP layer re-marshals it
// transparently, so a store-served job is wire-identical to a memory-served one.
func (s *dbJobStore) Load(ctx context.Context, id string) (jobs.Job, bool, error) {
	rec, err := s.repo.Get(ctx, id)
	if errors.Is(err, database.ErrJobNotFound) {
		return jobs.Job{}, false, nil
	}
	if err != nil {
		return jobs.Job{}, false, err
	}
	j := jobs.Job{
		ID:       rec.ID,
		Kind:     rec.Kind,
		State:    jobs.State(rec.State),
		Progress: rec.Progress,
		Err:      rec.Error,
	}
	if rec.ResultJSON != "" {
		j.Result = json.RawMessage(rec.ResultJSON)
	}
	return j, true, nil
}

// MarkInterrupted reconciles persisted in-flight jobs at startup.
func (s *dbJobStore) MarkInterrupted(ctx context.Context) (int, error) {
	return s.repo.MarkInterrupted(ctx)
}

// dbJobIdempotency is the durable Idempotency-Key store backing POST /jobs
// (Phase 5c-4), satisfying jobIdempotencyStore over database.JobRepository.
// Best-effort, matching the in-memory cache: a backend error degrades a check to
// idemMiss (create afresh) and a store failure is logged, never surfaced.
type dbJobIdempotency struct {
	repo   *database.JobRepository
	logger *slog.Logger
}

func newDBJobIdempotency(db *database.DB, logger *slog.Logger) *dbJobIdempotency {
	return &dbJobIdempotency{repo: db.Jobs(), logger: logger}
}

func (d *dbJobIdempotency) check(ctx context.Context, key string, req CreateJobRequest) idemResult {
	jobID, storedHash, found, err := d.repo.LookupIdempotency(ctx, key)
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
	if err := d.repo.RecordIdempotency(ctx, key, requestHash(req), jobID); err != nil {
		d.logger.ErrorContext(ctx, "idempotency record failed", "err", err)
	}
}

// jobStateTerminal reports whether a job state is final. It mirrors the runner's
// own (unexported) notion so the adapter can decide when to stamp completed_at.
func jobStateTerminal(state jobs.State) bool {
	switch state {
	case jobs.StateSucceeded, jobs.StateFailed, jobs.StateCancelled:
		return true
	case jobs.StateQueued, jobs.StateRunning:
		return false
	default:
		return false
	}
}
