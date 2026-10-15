package database

// The repository_report*.go files implement the persistence ports declared by
// internal/reporting. The database imports reporting, never the reverse;
// internal/app wires the two (ADR-0020, D-SEED-30).

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// ReportRepository implements reporting.ReportRepo over the SQLite reports table. The
// SQL and row scanning were lifted verbatim from the reporting package
// (services_reports.go + services.go:saveReport) when reporting was made
// I/O-free — Phase 3 slice 1b-iv.
type ReportRepository struct {
	db *DB
}

// NewReportRepository constructs a ReportRepository backed by db.
func NewReportRepository(db *DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Compile-time assertion that the adapter satisfies reporting's port.
var _ reporting.ReportRepo = (*ReportRepository)(nil)

// GetReport retrieves a report by ID.
func (r *ReportRepository) GetReport(ctx context.Context, id string) (*reporting.Report, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at
		FROM reports WHERE id = ?
	`, id)

	return scanReport(row)
}

// ListReports returns all generated reports.
func (r *ReportRepository) ListReports(ctx context.Context) ([]reporting.Report, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at
		FROM reports ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying reports: %w", err)
	}
	defer rows.Close()

	var reports []reporting.Report
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
func (r *ReportRepository) SaveReport(ctx context.Context, report *reporting.Report) error {
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

// UpdateReport updates an existing report row, reporting whether one matched.
// Unlike SaveReport this never inserts: the generator writes status updates
// after Generate has returned, and an INSERT there would resurrect a report the
// caller had already deleted.
func (r *ReportRepository) UpdateReport(ctx context.Context, report *reporting.Report) (bool, error) {
	paramsJSON, err := json.Marshal(report.Parameters)
	if err != nil {
		return false, fmt.Errorf("marshaling report parameters: %w", err)
	}

	var completedAt, expiresAt *string
	if report.CompletedAt != nil {
		t := report.CompletedAt.Format(time.RFC3339)
		completedAt = &t
	}
	if report.ExpiresAt != nil {
		t := report.ExpiresAt.Format(time.RFC3339)
		expiresAt = &t
	}

	res, err := r.db.Exec(ctx, `
		UPDATE reports
		SET name = ?, type = ?, format = ?, template = ?, status = ?, file_path = ?,
		    file_size = ?, parameters_json = ?, error = ?, completed_at = ?, expires_at = ?
		WHERE id = ?
	`, report.Name, report.Type, report.Format, report.Template, report.Status,
		report.FilePath, report.FileSize, string(paramsJSON), report.Error,
		completedAt, expiresAt, report.ID)
	if err != nil {
		return false, fmt.Errorf("updating report in database: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("reading update result for report %s: %w", report.ID, err)
	}

	return affected > 0, nil
}

// DeleteReport removes the report row (the file is removed by the service).
func (r *ReportRepository) DeleteReport(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting report from database: %w", err)
	}
	return nil
}

// scanReport materializes a Report from a QueryRow or Rows scanner.
func scanReport(row interface{ Scan(...any) error }) (*reporting.Report, error) {
	var r reporting.Report
	var paramsJSON, createdAt, completedAt, expiresAt *string

	// created_at is written as an RFC3339 string (see Save), so it has to be
	// read back as one. Scanning it straight into a time.Time made every read
	// fail with "unsupported Scan, storing driver.Value type string into type
	// *time.Time" -- the two sibling columns below already did this correctly.
	err := row.Scan(&r.ID, &r.Name, &r.Type, &r.Format, &r.Template, &r.Status,
		&r.FilePath, &r.FileSize, &paramsJSON, &r.Error, &createdAt, &completedAt, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("scanning report: %w", err)
	}

	if createdAt != nil {
		t, parseErr := time.Parse(time.RFC3339, *createdAt)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing report created_at %q: %w", *createdAt, parseErr)
		}
		r.CreatedAt = t
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
