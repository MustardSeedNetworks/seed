package harvest

// services_reports.go contains the report-record CRUD on GeneratorService:
// list, get, scan, download, delete, plus the template-driven Generate
// convenience entry point.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"
)

// GenerateFromTemplate creates a report using a template.
func (s *GeneratorService) GenerateFromTemplate(
	ctx context.Context,
	templateID string,
	format ExportFormat,
	params *ReportParams,
) (*Report, error) {
	tmpl, ok := s.templates.Get(templateID)
	if !ok {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Validate format is supported by template
	formatSupported := slices.Contains(tmpl.Formats, format)
	if !formatSupported {
		return nil, fmt.Errorf("format %s not supported by template %s", format, templateID)
	}

	return s.Generate(ctx, tmpl.Type, format, params)
}

// GetReport retrieves a report by ID.
func (s *GeneratorService) GetReport(ctx context.Context, id string) (*Report, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at
		FROM reports WHERE id = ?
	`, id)

	return s.scanReport(row)
}

// ListReports returns all generated reports.
func (s *GeneratorService) ListReports(ctx context.Context) ([]Report, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, type, format, template, status, file_path, file_size, parameters_json, error, created_at, completed_at, expires_at
		FROM reports ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying reports: %w", err)
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		report, scanErr := s.scanReportFromRows(rows)
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

func (s *GeneratorService) scanReport(row interface{ Scan(...any) error }) (*Report, error) {
	var r Report
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

func (s *GeneratorService) scanReportFromRows(
	rows interface{ Scan(...any) error },
) (*Report, error) {
	return s.scanReport(rows)
}

// DownloadReport returns the report file content.
func (s *GeneratorService) DownloadReport(ctx context.Context, id string) (io.ReadCloser, error) {
	report, err := s.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	if report.FilePath == "" {
		return nil, errors.New("report has no file")
	}

	file, err := os.Open(report.FilePath)
	if err != nil {
		return nil, fmt.Errorf("opening report file: %w", err)
	}

	return file, nil
}

// DeleteReport removes a report.
func (s *GeneratorService) DeleteReport(ctx context.Context, id string) error {
	report, err := s.GetReport(ctx, id)
	if err != nil {
		return err
	}

	// Delete file if exists
	if report.FilePath != "" {
		_ = os.Remove(report.FilePath)
	}

	_, err = s.db.Exec(ctx, "DELETE FROM reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting report from database: %w", err)
	}
	return nil
}
