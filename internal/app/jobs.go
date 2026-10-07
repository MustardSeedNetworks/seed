package app

// jobs.go is the durable side of the job runner (ADR-0005 Phase 5c, ADR-0020).
// JobStore implements jobs.Store over the jobs table, and carries the two other
// jobs-table concerns the api needs: Idempotency-Key records for POST /jobs and
// the retention prune. platform/jobs stays persistence-free.

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// JobStore persists job snapshots: it marshals the result to JSON and derives
// the created/updated/completed timestamps the in-memory Job does not carry.
type JobStore struct {
	repo *database.JobRepository
	now  func() time.Time
}

// NewJobStore builds the store over db's jobs repository.
func NewJobStore(db *database.DB) *JobStore {
	return &JobStore{
		repo: db.Jobs(),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// Save write-throughs a snapshot. created_at is set on every call but kept by
// the repository's upsert only on first insert; completed_at is stamped once the
// job reaches a terminal state and left NULL while active.
func (s *JobStore) Save(ctx context.Context, j jobs.Job) error {
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
	if j.State.Terminal() {
		rec.CompletedAt = now
	}
	return s.repo.Save(ctx, rec)
}

// Load returns the persisted job. A missing row is (zero, false, nil), not an
// error, so the runner's Get fallback treats it as a plain miss. The stored
// result comes back as [json.RawMessage] — the HTTP layer re-marshals it
// transparently, so a store-served job is wire-identical to a memory-served one.
func (s *JobStore) Load(ctx context.Context, id string) (jobs.Job, bool, error) {
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
func (s *JobStore) MarkInterrupted(ctx context.Context) (int, error) {
	return s.repo.MarkInterrupted(ctx)
}

// DeleteCompletedBefore prunes terminal jobs completed before cutoff.
func (s *JobStore) DeleteCompletedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.repo.DeleteCompletedBefore(ctx, cutoff)
}

// LookupIdempotency returns the job an Idempotency-Key was recorded against and
// the request hash stored with it; found is false for an unseen key.
func (s *JobStore) LookupIdempotency(ctx context.Context, key string) (string, string, bool, error) {
	return s.repo.LookupIdempotency(ctx, key)
}

// RecordIdempotency binds an Idempotency-Key and its request hash to jobID.
func (s *JobStore) RecordIdempotency(ctx context.Context, key, requestHash, jobID string) error {
	return s.repo.RecordIdempotency(ctx, key, requestHash, jobID)
}
