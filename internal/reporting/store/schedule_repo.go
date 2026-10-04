package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// ScheduleRepo implements reporting.ScheduleRepo over the scheduled_reports table.
// The SQL and row scanning were lifted verbatim from the reporting package
// (services_scheduler.go) when reporting was made I/O-free — Phase 3 slice 1b-v.
type ScheduleRepo struct {
	db *database.DB
}

// NewScheduleRepo constructs a ScheduleRepo backed by db.
func NewScheduleRepo(db *database.DB) *ScheduleRepo {
	return &ScheduleRepo{db: db}
}

// Compile-time assertion that the adapter satisfies reporting's port.
var _ reporting.ScheduleRepo = (*ScheduleRepo)(nil)

// ListSchedules returns every persisted scheduled report. A row that cannot be
// read is an error, not a skip: the skip hid that no row had ever loaded
// (created_at was scanned into a [time.Time]), so every schedule was dropped on
// restart without a word.
func (r *ScheduleRepo) ListSchedules(ctx context.Context) ([]reporting.ScheduledReport, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, template, format, schedule_json, parameters_json, recipients_json, enabled, last_run, next_run, created_at, updated_at
		FROM scheduled_reports
	`)
	if err != nil {
		return nil, fmt.Errorf("querying scheduled reports: %w", err)
	}
	defer rows.Close()

	var schedules []reporting.ScheduledReport
	for rows.Next() {
		sr, scanErr := scanSchedule(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		schedules = append(schedules, *sr)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating scheduled reports: %w", rowsErr)
	}

	return schedules, nil
}

// scanSchedule materializes one scheduled_reports row. The timestamps are
// written as RFC3339 strings (see SaveSchedule) and read back the same way.
func scanSchedule(row interface{ Scan(...any) error }) (*reporting.ScheduledReport, error) {
	var sr reporting.ScheduledReport
	var scheduleJSON, paramsJSON, recipientsJSON, createdAt, updatedAt string
	var lastRun, nextRun *string

	if err := row.Scan(
		&sr.ID,
		&sr.Name,
		&sr.Template,
		&sr.Format,
		&scheduleJSON,
		&paramsJSON,
		&recipientsJSON,
		&sr.Enabled,
		&lastRun,
		&nextRun,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("scanning scheduled report: %w", err)
	}

	if err := json.Unmarshal([]byte(scheduleJSON), &sr.Schedule); err != nil {
		return nil, fmt.Errorf("decoding schedule of scheduled report %s: %w", sr.ID, err)
	}
	_ = json.Unmarshal([]byte(paramsJSON), &sr.Parameters)
	_ = json.Unmarshal([]byte(recipientsJSON), &sr.Recipients)

	var err error
	if sr.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("parsing created_at of scheduled report %s: %w", sr.ID, err)
	}
	if sr.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("parsing updated_at of scheduled report %s: %w", sr.ID, err)
	}
	if lastRun != nil {
		t, _ := time.Parse(time.RFC3339, *lastRun)
		sr.LastRun = &t
	}
	if nextRun != nil {
		t, _ := time.Parse(time.RFC3339, *nextRun)
		sr.NextRun = &t
	}

	return &sr, nil
}

// SaveSchedule upserts a scheduled-report row.
func (r *ScheduleRepo) SaveSchedule(ctx context.Context, sr *reporting.ScheduledReport) error {
	scheduleJSON, _ := json.Marshal(sr.Schedule)
	paramsJSON, _ := json.Marshal(sr.Parameters)
	recipientsJSON, _ := json.Marshal(sr.Recipients)

	var lastRun, nextRun *string
	if sr.LastRun != nil {
		t := sr.LastRun.Format(time.RFC3339)
		lastRun = &t
	}
	if sr.NextRun != nil {
		t := sr.NextRun.Format(time.RFC3339)
		nextRun = &t
	}

	_, err := r.db.Exec(
		ctx,
		`
		INSERT OR REPLACE INTO scheduled_reports (id, name, template, format, schedule_json, parameters_json, recipients_json, enabled, last_run, next_run, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sr.ID,
		sr.Name,
		sr.Template,
		sr.Format,
		string(scheduleJSON),
		string(paramsJSON),
		string(recipientsJSON),
		sr.Enabled,
		lastRun,
		nextRun,
		sr.CreatedAt.Format(time.RFC3339),
		sr.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("saving scheduled report: %w", err)
	}

	return nil
}

// DeleteSchedule removes a scheduled-report row.
func (r *ScheduleRepo) DeleteSchedule(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM scheduled_reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting scheduled report: %w", err)
	}
	return nil
}
