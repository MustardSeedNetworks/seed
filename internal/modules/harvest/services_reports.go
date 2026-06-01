package harvest

// services_reports.go contains the report-record CRUD on GeneratorService:
// list, get, scan, download, delete, plus the template-driven Generate
// convenience entry point.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
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
	return s.reports.GetReport(ctx, id)
}

// ListReports returns all generated reports.
func (s *GeneratorService) ListReports(ctx context.Context) ([]Report, error) {
	return s.reports.ListReports(ctx)
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

	return s.reports.DeleteReport(ctx, id)
}
