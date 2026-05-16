package harvest

// services.go holds the harvest-package constants and the GeneratorService
// struct + orchestration: Generate (async entry), generateReport (the worker
// goroutine), saveReportFile, failReport, and saveReport. The PDF rendering,
// HTML/CSV/JSON renderers, report-record CRUD, bulk Export, plus the
// Template / Scheduler / Aggregator services each live in their own
// services_*.go file.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// DefaultReportsPath is the default directory for storing generated reports.
const DefaultReportsPath = "data/reports"

// DefaultReportTTL is how long reports are kept before expiring.
const DefaultReportTTL = 7 * 24 * time.Hour

// Period constants for report aggregation.
const (
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
)

// SQLite date format for grouping.
const sqliteDateFormat = "%Y-%m-%d"

// PDF layout and formatting constants.
const (
	// Page margins and spacing.
	pdfPageMarginBottom    = 15  // Bottom margin for auto page break
	pdfCoverTopMargin      = 60  // Top margin on cover page
	pdfSectionSpacingSmall = 5   // Small vertical spacing between sections
	pdfSectionSpacingMed   = 10  // Medium vertical spacing between sections
	pdfSectionSpacingLarge = 20  // Large vertical spacing between sections
	pdfLineStartX          = 10  // X coordinate for horizontal line start
	pdfLineEndX            = 200 // X coordinate for horizontal line end

	// Font sizes.
	pdfFontSizeCoverTitle = 28 // Cover page title font size
	pdfFontSizeCoverName  = 16 // Cover page report name font size
	pdfFontSizeCoverMeta  = 12 // Cover page metadata font size
	pdfFontSizeSection    = 14 // Section header font size
	pdfFontSizeSubsection = 11 // Subsection header font size
	pdfFontSizeBody       = 10 // Body text font size
	pdfFontSizeSmall      = 9  // Small text font size

	// Cell dimensions.
	pdfCellHeightTitle    = 15 // Height for title cells
	pdfCellHeightSubtitle = 10 // Height for subtitle/name cells
	pdfCellHeightBody     = 8  // Height for body text cells
	pdfCellHeightMetric   = 7  // Height for metric rows
	pdfCellHeightSeverity = 6  // Height for severity list items
	pdfLabelColumnWidth   = 60 // Width for label columns in metric tables
	pdfSeverityLabelWidth = 30 // Width for severity labels

	// Colors (RGB components).
	pdfColorGrayLight = 200 // Light gray for divider lines
	pdfColorGrayMid   = 100 // Medium gray for metadata text
	pdfColorGrayDark  = 80  // Dark gray for label text
)

// Date calculation constants.
const (
	hoursPerDay           = 24 // Hours in a day for period calculations
	monthlyThresholdDays  = 30 // Days threshold for monthly period
	daysInWeek            = 7  // Days in a week for modular arithmetic
	topIssuesDisplayLimit = 5  // Maximum top issues to display in reports
)

// Harvest entity-name constants used as switch discriminators across
// export types, GetTrends metrics, and template section IDs.
const (
	entityDevices = "devices"
)

// Template section order constants.
const (
	sectionOrderOverview    = 1 // Order for overview/summary sections
	sectionOrderSecondary   = 2 // Order for secondary sections (security, critical, devices)
	sectionOrderTertiary    = 3 // Order for tertiary sections (performance, high severity, software)
	sectionOrderQuaternary  = 4 // Order for fourth sections (recommendations, medium severity, changes)
	sectionOrderQuinary     = 5 // Order for fifth sections (low severity)
	sectionOrderRemediation = 6 // Order for remediation section
)

// GeneratorService generates reports in various formats.
type GeneratorService struct {
	cfg         *config.Config
	db          *database.DB
	templates   *TemplateService
	aggregator  *AggregatorService
	reportsPath string
	mu          sync.RWMutex
}

// NewGeneratorService creates a new generator service.
func NewGeneratorService(
	cfg *config.Config,
	db *database.DB,
	templates *TemplateService,
	aggregator *AggregatorService,
) *GeneratorService {
	return &GeneratorService{
		cfg:         cfg,
		db:          db,
		templates:   templates,
		aggregator:  aggregator,
		reportsPath: DefaultReportsPath,
	}
}

// Generate creates a report with the given parameters.
func (s *GeneratorService) Generate(
	ctx context.Context,
	reportType ReportType,
	format ExportFormat,
	params *ReportParams,
) (*Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create report record
	report := &Report{
		ID:        uuid.New().String(),
		Name:      fmt.Sprintf("%s Report - %s", reportType, time.Now().Format("2006-01-02")),
		Type:      reportType,
		Format:    format,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
	if params != nil {
		report.Parameters = *params
	}

	// Save initial report record
	if err := s.saveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("saving report: %w", err)
	}

	// Generate report asynchronously. Return a snapshot to the caller so they
	// can observe the initial pending state without racing the goroutine's
	// later writes to Status / CompletedAt / ExpiresAt / FileSize. Callers
	// wanting to observe progress should use GetReport(id), which reads
	// through s.mu.
	go s.generateReport(context.Background(), report)

	snapshot := *report
	return &snapshot, nil
}

// generateReport performs the actual report generation.
func (s *GeneratorService) generateReport(ctx context.Context, report *Report) {
	s.mu.Lock()
	report.Status = StatusGenerating
	_ = s.saveReport(ctx, report)
	s.mu.Unlock()

	// Aggregate data for the report
	dateRange := PeriodWeekly
	if report.Parameters.DateRange != nil {
		days := report.Parameters.DateRange.End.Sub(report.Parameters.DateRange.Start).Hours() / hoursPerDay
		if days > monthlyThresholdDays {
			dateRange = PeriodMonthly
		} else if days <= 1 {
			dateRange = PeriodDaily
		}
	}

	data, err := s.aggregator.Aggregate(ctx, dateRange, "", "")
	if err != nil {
		s.failReport(ctx, report, fmt.Sprintf("aggregation failed: %v", err))
		return
	}

	// Generate based on format
	var content []byte
	switch report.Format {
	case FormatPDF:
		content, err = s.generatePDF(report, data)
	case FormatHTML:
		content, err = s.generateHTML(report, data)
	case FormatCSV:
		content, err = s.generateCSV(report, data)
	case FormatJSON:
		content, err = s.generateJSON(report, data)
	case FormatExcel, FormatMarkdown:
		err = fmt.Errorf("unsupported format: %s", report.Format)
	}

	if err != nil {
		s.failReport(ctx, report, fmt.Sprintf("generation failed: %v", err))
		return
	}

	// Save to file
	if saveErr := s.saveReportFile(report, content); saveErr != nil {
		s.failReport(ctx, report, fmt.Sprintf("save failed: %v", saveErr))
		return
	}

	// Update report status
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expires := now.Add(DefaultReportTTL)
	report.Status = StatusComplete
	report.CompletedAt = &now
	report.ExpiresAt = &expires
	report.FileSize = int64(len(content))

	_ = s.saveReport(ctx, report)
}

func (s *GeneratorService) saveReportFile(report *Report, content []byte) error {
	// Ensure reports directory exists
	if err := os.MkdirAll(s.reportsPath, 0o750); err != nil {
		return fmt.Errorf("creating reports directory: %w", err)
	}

	// Generate filename
	ext := string(report.Format)
	filename := fmt.Sprintf("%s.%s", report.ID, ext)
	filepath := filepath.Join(s.reportsPath, filename)

	// Write file
	if err := os.WriteFile(filepath, content, 0o600); err != nil {
		return fmt.Errorf("writing report file: %w", err)
	}

	report.FilePath = filepath
	return nil
}

func (s *GeneratorService) failReport(ctx context.Context, report *Report, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	report.Status = StatusFailed
	report.Error = errMsg
	_ = s.saveReport(ctx, report)
}

func (s *GeneratorService) saveReport(ctx context.Context, report *Report) error {
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

	_, err := s.db.Exec(ctx, `
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
