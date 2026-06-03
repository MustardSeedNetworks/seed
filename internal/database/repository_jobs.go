package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrJobNotFound is returned when a job lookup misses.
var ErrJobNotFound = errors.New("job not found")

// JobRecord mirrors a jobs row — the durable projection of a
// platform/jobs Job (ADR-0005, Phase 5c). The Runner writes the full
// snapshot through Save on every lifecycle transition; ResultJSON and
// Error are the serialized success result and failure detail. CompletedAt
// is the zero time while the job is active (queued or running) and the
// terminal timestamp once it finishes.
type JobRecord struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	State       string    `json:"state"`
	Progress    float64   `json:"progress"`
	ResultJSON  string    `json:"resultJson,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CompletedAt time.Time `json:"completedAt,omitzero"`
}

// JobRepository owns durable read/write access to the jobs table. The
// Runner persists each transition; GET /jobs reads snapshots; the
// scheduler-driven retention sweep deletes terminal rows past their
// window.
type JobRepository struct {
	db *DB
}

// Jobs returns the durable job repository (Phase 5c). The platform/jobs
// Runner writes lifecycle transitions through it so jobs survive a restart.
func (db *DB) Jobs() *JobRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.jobs == nil {
		db.jobs = &JobRepository{db: db}
	}
	return db.jobs
}

// Save upserts a job snapshot. The first write (queued) inserts and fixes
// created_at + kind; later writes update the mutable lifecycle columns and
// updated_at only — created_at and kind are immutable after insert. Write the
// full current snapshot on every transition; this is the Runner's write-through.
func (r *JobRepository) Save(ctx context.Context, rec *JobRecord) error {
	if rec.ID == "" {
		return errors.New("jobs: ID required for Save")
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO jobs
		  (id, kind, state, progress, result_json, error,
		   created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			state        = excluded.state,
			progress     = excluded.progress,
			result_json  = excluded.result_json,
			error        = excluded.error,
			updated_at   = excluded.updated_at,
			completed_at = excluded.completed_at
	`,
		rec.ID, rec.Kind, rec.State, rec.Progress,
		toNullString(rec.ResultJSON), toNullString(rec.Error),
		rec.CreatedAt.UTC().Format(time.RFC3339Nano),
		rec.UpdatedAt.UTC().Format(time.RFC3339Nano),
		toNullTime(rec.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("save job: %w", err)
	}
	return nil
}

// Get returns the job by id, or ErrJobNotFound when absent.
func (r *JobRepository) Get(ctx context.Context, id string) (*JobRecord, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, kind, state, progress, result_json, error,
			created_at, updated_at, completed_at
		FROM jobs WHERE id = ?
	`, id)
	return scanJob(row.Scan)
}

// List returns all jobs, newest first (by created_at). Intended for the
// GET /jobs listing and for restart recovery (find jobs left non-terminal).
func (r *JobRepository) List(ctx context.Context) ([]*JobRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, kind, state, progress, result_json, error,
			created_at, updated_at, completed_at
		FROM jobs ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*JobRecord
	for rows.Next() {
		rec, scanErr := scanJob(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, rec)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate jobs: %w", rowsErr)
	}
	return out, nil
}

// DeleteCompletedBefore removes terminal jobs whose completed_at is before
// cutoff and returns the count removed. Active jobs (NULL completed_at) are
// never touched. This is the durable analogue of the in-memory Runner.Cleanup.
func (r *JobRepository) DeleteCompletedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `
		DELETE FROM jobs
		WHERE completed_at IS NOT NULL AND completed_at < ?
	`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("delete completed jobs: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// MarkInterrupted transitions every non-terminal job (queued or running) to
// failed, stamping an interruption error and the completion time. It is the
// startup-recovery primitive (Phase 5c): after a restart the handler goroutines
// of any in-flight job are gone, so those rows can never reach a terminal state
// on their own and are reconciled here. Returns the number of rows transitioned.
func (r *JobRepository) MarkInterrupted(ctx context.Context) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := r.db.Exec(ctx, `
		UPDATE jobs
		SET state = 'failed', error = ?, updated_at = ?, completed_at = ?
		WHERE state IN ('queued', 'running')
	`, "interrupted by restart", now, now)
	if err != nil {
		return 0, fmt.Errorf("mark interrupted jobs: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// scanJob maps one jobs row into a JobRecord, translating [sql.ErrNoRows] into
// ErrJobNotFound for the single-row Get path.
func scanJob(scan func(...any) error) (*JobRecord, error) {
	var (
		rec          JobRecord
		resultJSON   sql.NullString
		errStr       sql.NullString
		createdAtStr string
		updatedAtStr string
		completedAt  sql.NullString
	)
	err := scan(
		&rec.ID, &rec.Kind, &rec.State, &rec.Progress,
		&resultJSON, &errStr, &createdAtStr, &updatedAtStr, &completedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}

	if resultJSON.Valid {
		rec.ResultJSON = resultJSON.String
	}
	if errStr.Valid {
		rec.Error = errStr.String
	}
	rec.CreatedAt = parseTime(createdAtStr)
	rec.UpdatedAt = parseTime(updatedAtStr)
	if completedAt.Valid {
		rec.CompletedAt = parseTime(completedAt.String)
	}
	return &rec, nil
}
