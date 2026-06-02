package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/reporting"
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

// ListSchedules returns every persisted scheduled report. Rows that fail to
// scan are skipped (best-effort load, preserved from the original loader).
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
		var sr reporting.ScheduledReport
		var scheduleJSON, paramsJSON, recipientsJSON string
		var lastRun, nextRun *string

		scanErr := rows.Scan(
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
			&sr.CreatedAt,
			&sr.UpdatedAt,
		)
		if scanErr != nil {
			continue
		}

		_ = json.Unmarshal([]byte(scheduleJSON), &sr.Schedule)
		_ = json.Unmarshal([]byte(paramsJSON), &sr.Parameters)
		_ = json.Unmarshal([]byte(recipientsJSON), &sr.Recipients)

		if lastRun != nil {
			t, _ := time.Parse(time.RFC3339, *lastRun)
			sr.LastRun = &t
		}
		if nextRun != nil {
			t, _ := time.Parse(time.RFC3339, *nextRun)
			sr.NextRun = &t
		}

		schedules = append(schedules, sr)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating scheduled reports: %w", rowsErr)
	}

	return schedules, nil
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
