package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
)

// autotestTimeFormat is fixed-width with nanoseconds, so started_at sorts as
// text and a re-imported record keeps its exact start time.
const autotestTimeFormat = "2006-01-02T15:04:05.000000000Z"

// AutotestResultsRepository owns autotest_results (P-E2).
type AutotestResultsRepository struct {
	db *DB
}

// Insert stores a new record; an existing id is autotest.ErrExists.
func (r *AutotestResultsRepository) Insert(ctx context.Context, rep *autotest.Report) error {
	stages, err := json.Marshal(rep.Stages)
	if err != nil {
		return fmt.Errorf("encode autotest stages: %w", err)
	}
	res, err := r.db.Exec(ctx, `
		INSERT INTO autotest_results
		  (id, interface, jack_label, asset_tag, verdict, started_at, duration_ms, stages)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING`,
		rep.ID, rep.Interface, rep.JackLabel, rep.AssetTag, string(rep.Verdict),
		rep.StartedAt.UTC().Format(autotestTimeFormat), rep.DurationMs, string(stages))
	if err != nil {
		return fmt.Errorf("insert autotest result: %w", err)
	}
	if res == nil {
		return errors.New("insert autotest result: nil sql.Result")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("insert autotest result: %w", err)
	}
	if n == 0 {
		return autotest.ErrExists
	}
	return nil
}

const autotestColumns = `id, interface, jack_label, asset_tag, verdict, started_at, duration_ms, stages`

// Get returns one record; a miss is autotest.ErrNotFound.
func (r *AutotestResultsRepository) Get(ctx context.Context, id string) (*autotest.Report, error) {
	rep, err := scanAutotestResult(r.db.QueryRow(ctx,
		`SELECT `+autotestColumns+` FROM autotest_results WHERE id = ?`, id))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, autotest.ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("get autotest result: %w", err)
	}
	return rep, nil
}

// List returns every record, newest first.
func (r *AutotestResultsRepository) List(ctx context.Context) ([]autotest.Report, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+autotestColumns+` FROM autotest_results ORDER BY started_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list autotest results: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []autotest.Report
	for rows.Next() {
		rep, scanErr := scanAutotestResult(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *rep)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list autotest results: %w", err)
	}
	return out, nil
}

func scanAutotestResult(row interface{ Scan(dest ...any) error }) (*autotest.Report, error) {
	var (
		rep              autotest.Report
		verdict, started string
		stages           string
	)
	if err := row.Scan(&rep.ID, &rep.Interface, &rep.JackLabel, &rep.AssetTag, &verdict,
		&started, &rep.DurationMs, &stages); err != nil {
		return nil, err
	}
	rep.Verdict = autotest.Outcome(verdict)
	t, err := time.Parse(autotestTimeFormat, started)
	if err != nil {
		return nil, fmt.Errorf("autotest result %s started_at %q: %w", rep.ID, started, err)
	}
	rep.StartedAt = t
	if err = json.Unmarshal([]byte(stages), &rep.Stages); err != nil {
		return nil, fmt.Errorf("autotest result %s stages: %w", rep.ID, err)
	}
	return &rep, nil
}
