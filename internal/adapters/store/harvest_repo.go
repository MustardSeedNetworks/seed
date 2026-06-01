// Package store holds persistence adapters: concrete SQLite implementations of
// the repository ports declared by the domain modules. Adapters depend on
// modules (importing their types and satisfying their ports), never the
// reverse — the inward-only direction enforced by depguard.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/modules/harvest"
)

// ReportRepo implements harvest.ReportRepo over the SQLite reports table. The
// SQL and row scanning were lifted verbatim from the harvest module
// (services_reports.go + services.go:saveReport) when the module was made
// I/O-free — Phase 3 slice 1b-iv.
type ReportRepo struct {
	db *database.DB
}

// NewReportRepo constructs a ReportRepo backed by db.
func NewReportRepo(db *database.DB) *ReportRepo {
	return &ReportRepo{db: db}
}

// Compile-time assertion that the adapter satisfies the module's port.
var _ harvest.ReportRepo = (*ReportRepo)(nil)

// GetReport retrieves a report by ID.
func (r *ReportRepo) GetReport(ctx context.Context, id string) (*harvest.Report, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at
		FROM reports WHERE id = ?
	`, id)

	return scanReport(row)
}

// ListReports returns all generated reports.
func (r *ReportRepo) ListReports(ctx context.Context) ([]harvest.Report, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at
		FROM reports ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying reports: %w", err)
	}
	defer rows.Close()

	var reports []harvest.Report
	for rows.Next() {
		report, scanErr := scanReport(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		reports = append(reports, *report)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating reports: %w", rowsErr)
	}

	return reports, nil
}

// SaveReport upserts a report record.
func (r *ReportRepo) SaveReport(ctx context.Context, report *harvest.Report) error {
	paramsJSON, _ := json.Marshal(report.Parameters)

	var completedAt, expiresAt *string
	if report.CompletedAt != nil {
		t := report.CompletedAt.Format(time.RFC3339)
		completedAt = &t
	}
	if report.ExpiresAt != nil {
		t := report.ExpiresAt.Format(time.RFC3339)
		expiresAt = &t
	}

	_, err := r.db.Exec(ctx, `
		INSERT OR REPLACE INTO reports (id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, report.ID, report.Name, report.Type, report.Format, report.Template, report.Status,
		report.FilePath, report.FileSize, string(paramsJSON), report.Error,
		report.CreatedAt.Format(time.RFC3339), completedAt, expiresAt)
	if err != nil {
		return fmt.Errorf("saving report to database: %w", err)
	}

	return nil
}

// DeleteReport removes the report row (the file is removed by the service).
func (r *ReportRepo) DeleteReport(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting report from database: %w", err)
	}
	return nil
}

// scanReport materializes a Report from a QueryRow or Rows scanner.
func scanReport(row interface{ Scan(...any) error }) (*harvest.Report, error) {
	var r harvest.Report
	var paramsJSON, completedAt, expiresAt *string

	err := row.Scan(&r.ID, &r.Name, &r.Type, &r.Format, &r.Template, &r.Status,
		&r.FilePath, &r.FileSize, &paramsJSON, &r.Error, &r.CreatedAt, &completedAt, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("scanning report: %w", err)
	}

	if paramsJSON != nil {
		_ = json.Unmarshal([]byte(*paramsJSON), &r.Parameters)
	}
	if completedAt != nil {
		t, _ := time.Parse(time.RFC3339, *completedAt)
		r.CompletedAt = &t
	}
	if expiresAt != nil {
		t, _ := time.Parse(time.RFC3339, *expiresAt)
		r.ExpiresAt = &t
	}

	return &r, nil
}
