package harvest

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-pdf/fpdf"
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

	// Generate report asynchronously
	go s.generateReport(context.Background(), report)

	return report, nil
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
		days := report.Parameters.DateRange.End.Sub(report.Parameters.DateRange.Start).Hours() / 24
		if days > 30 {
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

// generatePDF creates a PDF report.
func (s *GeneratorService) generatePDF(report *Report, data *AggregatedData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	// Cover page
	pdf.AddPage()
	s.addPDFCover(pdf, report)

	// Executive summary
	pdf.AddPage()
	s.addPDFExecutiveSummary(pdf, data)

	// Device inventory section
	pdf.AddPage()
	s.addPDFDeviceSection(pdf, data)

	// Vulnerability section (if applicable)
	if report.Type == ReportTypeVulnerability || report.Type == ReportTypeExecutive ||
		report.Type == ReportTypeDetailed {
		pdf.AddPage()
		s.addPDFVulnerabilitySection(pdf, data)
	}

	// Performance section
	if report.Type == ReportTypePerformance || report.Type == ReportTypeExecutive || report.Type == ReportTypeDetailed {
		pdf.AddPage()
		s.addPDFPerformanceSection(pdf, data)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *GeneratorService) addPDFCover(pdf *fpdf.Fpdf, report *Report) {
	pdf.SetFont("Arial", "B", 28)
	pdf.Ln(60)
	pdf.CellFormat(0, 15, "Network Report", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 16)
	pdf.Ln(10)
	pdf.CellFormat(0, 10, report.Name, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 12)
	pdf.Ln(20)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(
		0,
		8,
		fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 15:04")),
		"",
		1,
		"C",
		false,
		0,
		"",
	)
	pdf.CellFormat(0, 8, fmt.Sprintf("Type: %s", report.Type), "", 1, "C", false, 0, "")
}

func (s *GeneratorService) addPDFExecutiveSummary(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Executive Summary")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)

	metrics := []struct {
		label string
		value string
	}{
		{"Report Period", fmt.Sprintf("%s to %s", data.StartDate.Format("Jan 2"), data.EndDate.Format("Jan 2, 2006"))},
		{"Total Devices", strconv.Itoa(data.DeviceCount)},
		{"Total Vulnerabilities", strconv.Itoa(data.VulnCount.Total)},
		{"Critical Issues", strconv.Itoa(data.VulnCount.Critical)},
		{"Average Latency", fmt.Sprintf("%.1f ms", data.Performance.AvgLatencyMs)},
		{"Uptime", fmt.Sprintf("%.1f%%", data.Performance.UptimePercent)},
	}

	for _, m := range metrics {
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(60, 7, m.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 7, m.value, "", 1, "L", false, 0, "")
	}
}

func (s *GeneratorService) addPDFDeviceSection(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Device Inventory")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 8, fmt.Sprintf("Total devices discovered: %d", data.DeviceCount), "", 1, "L", false, 0, "")
}

func (s *GeneratorService) addPDFVulnerabilitySection(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Vulnerability Assessment")

	// Severity breakdown
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(0, 8, "Severity Distribution", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	severities := []struct {
		label string
		count int
		color []int
	}{
		{"Critical", data.VulnCount.Critical, []int{220, 53, 69}},
		{"High", data.VulnCount.High, []int{255, 128, 0}},
		{"Medium", data.VulnCount.Medium, []int{255, 193, 7}},
		{"Low", data.VulnCount.Low, []int{40, 167, 69}},
	}

	for _, sev := range severities {
		pdf.SetTextColor(sev.color[0], sev.color[1], sev.color[2])
		pdf.CellFormat(30, 6, sev.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 6, strconv.Itoa(sev.count), "", 1, "L", false, 0, "")
	}

	// Top issues
	if len(data.TopIssues) > 0 {
		pdf.Ln(5)
		pdf.SetFont("Arial", "B", 11)
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 8, "Top Issues", "", 1, "L", false, 0, "")

		pdf.SetFont("Arial", "", 9)
		for i, issue := range data.TopIssues {
			if i >= 5 {
				break
			}
			pdf.CellFormat(
				0,
				6,
				fmt.Sprintf("%d. %s (%d occurrences)", i+1, issue.Description, issue.Count),
				"",
				1,
				"L",
				false,
				0,
				"",
			)
		}
	}
}

func (s *GeneratorService) addPDFPerformanceSection(pdf *fpdf.Fpdf, data *AggregatedData) {
	s.addPDFSectionHeader(pdf, "Performance Metrics")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)

	metrics := []struct {
		label string
		value string
	}{
		{"Average Latency", fmt.Sprintf("%.2f ms", data.Performance.AvgLatencyMs)},
		{"Packet Loss", fmt.Sprintf("%.2f%%", data.Performance.AvgPacketLoss)},
		{"Average Bandwidth", fmt.Sprintf("%.2f Mbps", data.Performance.AvgBandwidthMbps)},
		{"Uptime", fmt.Sprintf("%.2f%%", data.Performance.UptimePercent)},
	}

	for _, m := range metrics {
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(60, 7, m.label+":", "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 7, m.value, "", 1, "L", false, 0, "")
	}
}

func (s *GeneratorService) addPDFSectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 10, title, "", 1, "L", false, 0, "")

	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)
}

// generateHTML creates an HTML report.
//
//nolint:unparam // error return kept for interface consistency with other generators
func (s *GeneratorService) generateHTML(report *Report, data *AggregatedData) ([]byte, error) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; border-bottom: 2px solid #d4a017; padding-bottom: 10px; }
        h2 { color: #666; margin-top: 30px; }
        .metric { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee; }
        .metric-label { color: #666; }
        .metric-value { font-weight: bold; }
        .severity-critical { color: #dc3545; }
        .severity-high { color: #ff8000; }
        .severity-medium { color: #ffc107; }
        .severity-low { color: #28a745; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <p>Generated: %s</p>
    <p>Period: %s to %s</p>

    <h2>Executive Summary</h2>
    <div class="metric"><span class="metric-label">Total Devices</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label">Total Vulnerabilities</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label">Average Latency</span><span class="metric-value">%.2f ms</span></div>
    <div class="metric"><span class="metric-label">Uptime</span><span class="metric-value">%.2f%%</span></div>

    <h2>Vulnerability Summary</h2>
    <div class="metric"><span class="metric-label severity-critical">Critical</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label severity-high">High</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label severity-medium">Medium</span><span class="metric-value">%d</span></div>
    <div class="metric"><span class="metric-label severity-low">Low</span><span class="metric-value">%d</span></div>

    <h2>Performance Metrics</h2>
    <div class="metric"><span class="metric-label">Average Latency</span><span class="metric-value">%.2f ms</span></div>
    <div class="metric"><span class="metric-label">Packet Loss</span><span class="metric-value">%.2f%%</span></div>
    <div class="metric"><span class="metric-label">Bandwidth</span><span class="metric-value">%.2f Mbps</span></div>
</body>
</html>`,
		report.Name, report.Name,
		time.Now().Format("January 2, 2006 15:04"),
		data.StartDate.Format("Jan 2"), data.EndDate.Format("Jan 2, 2006"),
		data.DeviceCount, data.VulnCount.Total,
		data.Performance.AvgLatencyMs, data.Performance.UptimePercent,
		data.VulnCount.Critical, data.VulnCount.High, data.VulnCount.Medium, data.VulnCount.Low,
		data.Performance.AvgLatencyMs, data.Performance.AvgPacketLoss, data.Performance.AvgBandwidthMbps,
	)

	return []byte(html), nil
}

// generateCSV creates a CSV report.
func (s *GeneratorService) generateCSV(report *Report, data *AggregatedData) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header
	_ = writer.Write([]string{"Metric", "Value"})

	// Data rows
	rows := [][]string{
		{"Report Name", report.Name},
		{"Generated", time.Now().Format(time.RFC3339)},
		{"Period Start", data.StartDate.Format(time.RFC3339)},
		{"Period End", data.EndDate.Format(time.RFC3339)},
		{"Total Devices", strconv.Itoa(data.DeviceCount)},
		{"Total Vulnerabilities", strconv.Itoa(data.VulnCount.Total)},
		{"Critical Vulnerabilities", strconv.Itoa(data.VulnCount.Critical)},
		{"High Vulnerabilities", strconv.Itoa(data.VulnCount.High)},
		{"Medium Vulnerabilities", strconv.Itoa(data.VulnCount.Medium)},
		{"Low Vulnerabilities", strconv.Itoa(data.VulnCount.Low)},
		{"Average Latency (ms)", fmt.Sprintf("%.2f", data.Performance.AvgLatencyMs)},
		{"Packet Loss (%)", fmt.Sprintf("%.2f", data.Performance.AvgPacketLoss)},
		{"Bandwidth (Mbps)", fmt.Sprintf("%.2f", data.Performance.AvgBandwidthMbps)},
		{"Uptime (%)", fmt.Sprintf("%.2f", data.Performance.UptimePercent)},
	}

	for _, row := range rows {
		_ = writer.Write(row)
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// generateJSON creates a JSON report.
func (s *GeneratorService) generateJSON(report *Report, data *AggregatedData) ([]byte, error) {
	output := map[string]any{
		"report": map[string]any{
			"id":        report.ID,
			"name":      report.Name,
			"type":      report.Type,
			"generated": time.Now().Format(time.RFC3339),
		},
		"data": data,
	}

	result, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON report: %w", err)
	}
	return result, nil
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

func (s *GeneratorService) scanReportFromRows(rows interface{ Scan(...any) error }) (*Report, error) {
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

// Export exports data in the specified format.
func (s *GeneratorService) Export(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	if req == nil {
		return nil, errors.New("export request is nil")
	}

	// Create export ID
	exportID := uuid.New().String()

	// Query data based on type
	var data any
	var recordCount int
	var err error

	switch req.Type {
	case "devices":
		data, recordCount, err = s.exportDevices(ctx, req)
	case "vulnerabilities":
		data, recordCount, err = s.exportVulnerabilities(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported export type: %s", req.Type)
	}

	if err != nil {
		return nil, err
	}

	// Generate export file
	var content []byte
	switch req.Format {
	case FormatJSON:
		content, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling export data: %w", err)
		}
	case FormatCSV:
		content, err = s.dataToCSV(data)
	case FormatPDF, FormatHTML, FormatExcel, FormatMarkdown:
		err = fmt.Errorf("unsupported export format: %s", req.Format)
	}

	if err != nil {
		return nil, err
	}

	// Save file
	filename := fmt.Sprintf("export-%s.%s", exportID, req.Format)
	filepath := filepath.Join(s.reportsPath, filename)

	if mkdirErr := os.MkdirAll(s.reportsPath, 0o750); mkdirErr != nil {
		return nil, fmt.Errorf("creating export directory: %w", mkdirErr)
	}
	if writeErr := os.WriteFile(filepath, content, 0o600); writeErr != nil {
		return nil, fmt.Errorf("writing export file: %w", writeErr)
	}

	return &ExportResult{
		ID:          exportID,
		FilePath:    filepath,
		FileSize:    int64(len(content)),
		RecordCount: recordCount,
		Format:      req.Format,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(DefaultReportTTL),
	}, nil
}

func (s *GeneratorService) exportDevices(ctx context.Context, _ *ExportRequest) (any, int, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ip_address, mac_address, hostname, vendor, device_type, first_seen, last_seen
		FROM devices ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("querying devices: %w", err)
	}
	defer rows.Close()

	var devices []map[string]any
	for rows.Next() {
		var id, ip, mac, hostname, vendor, deviceType, firstSeen, lastSeen string
		if scanErr := rows.Scan(&id, &ip, &mac, &hostname, &vendor, &deviceType, &firstSeen, &lastSeen); scanErr != nil {
			continue
		}
		devices = append(devices, map[string]any{
			"id":          id,
			"ip_address":  ip,
			"mac_address": mac,
			"hostname":    hostname,
			"vendor":      vendor,
			"device_type": deviceType,
			"first_seen":  firstSeen,
			"last_seen":   lastSeen,
		})
	}

	return devices, len(devices), nil
}

func (s *GeneratorService) exportVulnerabilities(ctx context.Context, _ *ExportRequest) (any, int, error) {
	rows, err := s.db.Query(ctx, `
		SELECT dv.id, dv.device_id, dv.cve_id, dv.severity, dv.description, dv.discovered_at, d.ip_address
		FROM device_vulnerabilities dv
		LEFT JOIN devices d ON dv.device_id = d.id
		ORDER BY dv.severity DESC, dv.discovered_at DESC
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("querying vulnerabilities: %w", err)
	}
	defer rows.Close()

	var vulns []map[string]any
	for rows.Next() {
		var id int
		var deviceID, cveID, severity, desc, discoveredAt string
		var ipAddr *string
		if scanErr := rows.Scan(&id, &deviceID, &cveID, &severity, &desc, &discoveredAt, &ipAddr); scanErr != nil {
			continue
		}
		vulns = append(vulns, map[string]any{
			"id":            id,
			"device_id":     deviceID,
			"cve_id":        cveID,
			"severity":      severity,
			"description":   desc,
			"discovered_at": discoveredAt,
			"device_ip":     ipAddr,
		})
	}

	return vulns, len(vulns), nil
}

func (s *GeneratorService) dataToCSV(data any) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if d, ok := data.([]map[string]any); ok && len(d) > 0 {
		// Get headers from first record
		var headers []string
		for k := range d[0] {
			headers = append(headers, k)
		}
		sort.Strings(headers)
		_ = writer.Write(headers)

		// Write rows
		for _, record := range d {
			var row []string
			for _, h := range headers {
				row = append(row, fmt.Sprintf("%v", record[h]))
			}
			_ = writer.Write(row)
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// TemplateService manages report templates.
type TemplateService struct {
	cfg       *config.Config
	templates map[string]*Template
	mu        sync.RWMutex
}

// NewTemplateService creates a new template service.
func NewTemplateService(cfg *config.Config) *TemplateService {
	return &TemplateService{
		cfg:       cfg,
		templates: make(map[string]*Template),
	}
}

// Load loads all available templates.
func (s *TemplateService) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadBuiltInTemplates()
	return nil
}

func (s *TemplateService) loadBuiltInTemplates() {
	now := time.Now()

	// Executive Summary template
	s.templates["executive"] = &Template{
		ID:          "executive",
		Name:        "Executive Summary",
		Description: "High-level network health and security overview for management",
		Type:        ReportTypeExecutive,
		Formats:     []ExportFormat{FormatPDF, FormatHTML},
		Sections: []TemplateSection{
			{ID: "overview", Name: "Overview", Title: "Network Overview", Order: 1},
			{ID: "security", Name: "Security", Title: "Security Posture", Order: 2},
			{ID: "performance", Name: "Performance", Title: "Performance Summary", Order: 3},
			{ID: "recommendations", Name: "Recommendations", Title: "Recommendations", Order: 4},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Vulnerability Report template
	s.templates["vulnerability"] = &Template{
		ID:          "vulnerability",
		Name:        "Vulnerability Report",
		Description: "Detailed vulnerability assessment and remediation guidance",
		Type:        ReportTypeVulnerability,
		Formats:     []ExportFormat{FormatPDF, FormatHTML, FormatCSV},
		Sections: []TemplateSection{
			{ID: "summary", Name: "Summary", Title: "Vulnerability Summary", Order: 1},
			{ID: "critical", Name: "Critical", Title: "Critical Vulnerabilities", Order: 2},
			{ID: "high", Name: "High", Title: "High Severity", Order: 3},
			{ID: "medium", Name: "Medium", Title: "Medium Severity", Order: 4, Optional: true},
			{ID: "low", Name: "Low", Title: "Low Severity", Order: 5, Optional: true},
			{ID: "remediation", Name: "Remediation", Title: "Remediation Plan", Order: 6},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Device Inventory template
	s.templates["inventory"] = &Template{
		ID:          "inventory",
		Name:        "Device Inventory",
		Description: "Complete network device inventory with details",
		Type:        ReportTypeInventory,
		Formats:     []ExportFormat{FormatPDF, FormatHTML, FormatCSV, FormatExcel},
		Sections: []TemplateSection{
			{ID: "summary", Name: "Summary", Title: "Inventory Summary", Order: 1},
			{ID: "devices", Name: "Devices", Title: "Device List", Order: 2},
			{ID: "software", Name: "Software", Title: "Software Inventory", Order: 3, Optional: true},
			{ID: "changes", Name: "Changes", Title: "Recent Changes", Order: 4, Optional: true},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Performance Report template
	s.templates["performance"] = &Template{
		ID:          "performance",
		Name:        "Performance Report",
		Description: "Network performance metrics and trends analysis",
		Type:        ReportTypePerformance,
		Formats:     []ExportFormat{FormatPDF, FormatHTML, FormatJSON},
		Sections: []TemplateSection{
			{ID: "overview", Name: "Overview", Title: "Performance Overview", Order: 1},
			{ID: "latency", Name: "Latency", Title: "Latency Analysis", Order: 2},
			{ID: "throughput", Name: "Throughput", Title: "Throughput Metrics", Order: 3},
			{ID: "availability", Name: "Availability", Title: "Service Availability", Order: 4},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Get retrieves a template by ID.
func (s *TemplateService) Get(id string) (*Template, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.templates[id]
	return t, ok
}

// List returns all available templates.
func (s *TemplateService) List() []Template {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Template, 0, len(s.templates))
	for _, t := range s.templates {
		result = append(result, *t)
	}
	return result
}

// Create adds a custom template.
func (s *TemplateService) Create(tmpl *Template) error {
	if tmpl == nil {
		return errors.New("template is nil")
	}
	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.templates[tmpl.ID]; exists {
		return fmt.Errorf("template already exists: %s", tmpl.ID)
	}

	tmpl.IsBuiltIn = false
	tmpl.CreatedAt = time.Now()
	tmpl.UpdatedAt = time.Now()
	s.templates[tmpl.ID] = tmpl

	return nil
}

// Update modifies a custom template.
func (s *TemplateService) Update(tmpl *Template) error {
	if tmpl == nil {
		return errors.New("template is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.templates[tmpl.ID]
	if !ok {
		return fmt.Errorf("template not found: %s", tmpl.ID)
	}
	if existing.IsBuiltIn {
		return errors.New("cannot modify built-in template")
	}

	tmpl.UpdatedAt = time.Now()
	s.templates[tmpl.ID] = tmpl

	return nil
}

// Delete removes a custom template.
func (s *TemplateService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, ok := s.templates[id]
	if !ok {
		return fmt.Errorf("template not found: %s", id)
	}
	if tmpl.IsBuiltIn {
		return errors.New("cannot delete built-in template")
	}

	delete(s.templates, id)
	return nil
}

// SchedulerService manages scheduled reports.
type SchedulerService struct {
	cfg       *config.Config
	db        *database.DB
	generator *GeneratorService
	cancel    context.CancelFunc
	mu        sync.RWMutex
	schedules map[string]*ScheduledReport
}

// NewSchedulerService creates a new scheduler service.
func NewSchedulerService(cfg *config.Config, db *database.DB, generator *GeneratorService) *SchedulerService {
	return &SchedulerService{
		cfg:       cfg,
		db:        db,
		generator: generator,
		schedules: make(map[string]*ScheduledReport),
	}
}

// Start begins the scheduler.
func (s *SchedulerService) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

	// Load existing schedules
	if err := s.loadSchedules(ctx); err != nil {
		return fmt.Errorf("loading schedules: %w", err)
	}

	// Start scheduler loop
	go s.runScheduler(ctx)

	return nil
}

func (s *SchedulerService) loadSchedules(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, template, format, schedule_json, parameters_json, recipients_json, enabled, last_run, next_run, created_at, updated_at
		FROM scheduled_reports
	`)
	if err != nil {
		return fmt.Errorf("querying scheduled reports: %w", err)
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for rows.Next() {
		var sr ScheduledReport
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

		s.schedules[sr.ID] = &sr
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return fmt.Errorf("iterating scheduled reports: %w", rowsErr)
	}

	return nil
}

func (s *SchedulerService) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkSchedules(ctx)
		}
	}
}

func (s *SchedulerService) checkSchedules(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}
		if schedule.NextRun != nil && now.After(*schedule.NextRun) {
			go s.runScheduledReport(ctx, schedule)
		}
	}
}

func (s *SchedulerService) runScheduledReport(ctx context.Context, schedule *ScheduledReport) {
	// Generate report
	_, _ = s.generator.GenerateFromTemplate(ctx, schedule.Template, schedule.Format, &schedule.Parameters)

	// Update last run and calculate next run
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	schedule.LastRun = &now
	schedule.NextRun = calculateNextRun(&schedule.Schedule)
	schedule.UpdatedAt = now

	_ = s.saveSchedule(ctx, schedule)
}

func calculateNextRun(schedule *Schedule) *time.Time {
	now := time.Now()

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.Local
	}

	var next time.Time
	switch schedule.Frequency {
	case FrequencyDaily:
		next = time.Date(now.Year(), now.Month(), now.Day()+1, schedule.Hour, schedule.Minute, 0, 0, loc)
	case FrequencyWeekly:
		next = now
		if schedule.DayOfWeek != nil {
			daysUntil := (*schedule.DayOfWeek - int(now.Weekday()) + 7) % 7
			if daysUntil == 0 {
				daysUntil = 7
			}
			next = next.AddDate(0, 0, daysUntil)
		}
		next = time.Date(next.Year(), next.Month(), next.Day(), schedule.Hour, schedule.Minute, 0, 0, loc)
	case FrequencyMonthly:
		day := 1
		if schedule.DayOfMonth != nil {
			day = *schedule.DayOfMonth
		}
		next = time.Date(now.Year(), now.Month()+1, day, schedule.Hour, schedule.Minute, 0, 0, loc)
	}

	return &next
}

// Stop halts the scheduler.
func (s *SchedulerService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// Create adds a scheduled report.
func (s *SchedulerService) Create(ctx context.Context, sr *ScheduledReport) error {
	if sr == nil {
		return errors.New("scheduled report is nil")
	}
	if sr.ID == "" {
		sr.ID = uuid.New().String()
	}

	sr.CreatedAt = time.Now()
	sr.UpdatedAt = time.Now()
	sr.NextRun = calculateNextRun(&sr.Schedule)

	if err := s.saveSchedule(ctx, sr); err != nil {
		return err
	}

	s.mu.Lock()
	s.schedules[sr.ID] = sr
	s.mu.Unlock()

	return nil
}

func (s *SchedulerService) saveSchedule(ctx context.Context, sr *ScheduledReport) error {
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

	_, err := s.db.Exec(ctx, `
		INSERT OR REPLACE INTO scheduled_reports (id, name, template, format, schedule_json, parameters_json, recipients_json, enabled, last_run, next_run, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sr.ID, sr.Name, sr.Template, sr.Format, string(scheduleJSON), string(paramsJSON), string(recipientsJSON),
		sr.Enabled, lastRun, nextRun, sr.CreatedAt.Format(time.RFC3339), sr.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("saving scheduled report: %w", err)
	}

	return nil
}

// Get retrieves a scheduled report.
func (s *SchedulerService) Get(_ context.Context, id string) (*ScheduledReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sr, ok := s.schedules[id]
	if !ok {
		return nil, fmt.Errorf("scheduled report not found: %s", id)
	}
	return sr, nil
}

// List returns all scheduled reports.
func (s *SchedulerService) List(_ context.Context) ([]ScheduledReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ScheduledReport, 0, len(s.schedules))
	for _, sr := range s.schedules {
		result = append(result, *sr)
	}
	return result, nil
}

// Update modifies a scheduled report.
func (s *SchedulerService) Update(ctx context.Context, sr *ScheduledReport) error {
	if sr == nil {
		return errors.New("scheduled report is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schedules[sr.ID]; !ok {
		return fmt.Errorf("scheduled report not found: %s", sr.ID)
	}

	sr.UpdatedAt = time.Now()
	sr.NextRun = calculateNextRun(&sr.Schedule)
	s.schedules[sr.ID] = sr

	return s.saveSchedule(ctx, sr)
}

// Delete removes a scheduled report.
func (s *SchedulerService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schedules[id]; !ok {
		return fmt.Errorf("scheduled report not found: %s", id)
	}

	delete(s.schedules, id)
	_, err := s.db.Exec(ctx, "DELETE FROM scheduled_reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting scheduled report: %w", err)
	}
	return nil
}

// AggregatorService aggregates data for reports.
type AggregatorService struct {
	cfg *config.Config
	db  *database.DB
}

// NewAggregatorService creates a new aggregator service.
func NewAggregatorService(cfg *config.Config, db *database.DB) *AggregatorService {
	return &AggregatorService{
		cfg: cfg,
		db:  db,
	}
}

// Aggregate collects and aggregates data for a time period.
func (s *AggregatorService) Aggregate(ctx context.Context, period, _, _ string) (*AggregatedData, error) {
	// Calculate date range based on period
	now := time.Now()
	var startDate time.Time

	switch period {
	case PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
	case PeriodWeekly:
		startDate = now.AddDate(0, 0, -7)
	case PeriodMonthly:
		startDate = now.AddDate(0, -1, 0)
	default:
		startDate = now.AddDate(0, 0, -7) // Default to weekly
	}

	data := &AggregatedData{
		Period:    period,
		StartDate: startDate,
		EndDate:   now,
	}

	// Aggregate device count
	row := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM devices")
	_ = row.Scan(&data.DeviceCount)

	// Aggregate vulnerability counts
	s.aggregateVulnerabilities(ctx, data, startDate)

	// Aggregate performance metrics
	s.aggregatePerformance(ctx, data, startDate)

	// Get top issues
	s.aggregateTopIssues(ctx, data)

	return data, nil
}

func (s *AggregatorService) aggregateVulnerabilities(ctx context.Context, data *AggregatedData, since time.Time) {
	rows, err := s.db.Query(ctx, `
		SELECT severity, COUNT(*) as count
		FROM device_vulnerabilities
		WHERE discovered_at >= ?
		GROUP BY severity
	`, since.Format(time.RFC3339))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var count int
		if scanErr := rows.Scan(&severity, &count); scanErr != nil {
			continue
		}

		switch severity {
		case "critical":
			data.VulnCount.Critical = count
		case "high":
			data.VulnCount.High = count
		case "medium":
			data.VulnCount.Medium = count
		case "low":
			data.VulnCount.Low = count
		}
		data.VulnCount.Total += count
	}
}

func (s *AggregatorService) aggregatePerformance(ctx context.Context, data *AggregatedData, since time.Time) {
	// Get average latency from gateway results
	row := s.db.QueryRow(ctx, `
		SELECT AVG(latency_ms), AVG(packet_loss)
		FROM gateway_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var avgLatency, avgPacketLoss *float64
	_ = row.Scan(&avgLatency, &avgPacketLoss)

	if avgLatency != nil {
		data.Performance.AvgLatencyMs = *avgLatency
	}
	if avgPacketLoss != nil {
		data.Performance.AvgPacketLoss = *avgPacketLoss
	}

	// Get average bandwidth from speedtest results
	row = s.db.QueryRow(ctx, `
		SELECT AVG((download_mbps + upload_mbps) / 2)
		FROM speedtest_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var avgBandwidth *float64
	_ = row.Scan(&avgBandwidth)
	if avgBandwidth != nil {
		data.Performance.AvgBandwidthMbps = *avgBandwidth
	}

	// Calculate uptime (simplified: based on successful gateway checks)
	row = s.db.QueryRow(ctx, `
		SELECT
			COUNT(CASE WHEN success = 1 THEN 1 END) * 100.0 / COUNT(*)
		FROM gateway_results
		WHERE timestamp >= ?
	`, since.Format(time.RFC3339))

	var uptime *float64
	_ = row.Scan(&uptime)
	if uptime != nil {
		data.Performance.UptimePercent = *uptime
	} else {
		data.Performance.UptimePercent = 100.0 // Default to 100% if no data
	}
}

func (s *AggregatorService) aggregateTopIssues(ctx context.Context, data *AggregatedData) {
	rows, err := s.db.Query(ctx, `
		SELECT severity, description, COUNT(*) as count
		FROM device_vulnerabilities
		GROUP BY description
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				ELSE 4
			END,
			count DESC
		LIMIT 10
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var issue IssueSummary
		if scanErr := rows.Scan(&issue.Severity, &issue.Description, &issue.Count); scanErr != nil {
			continue
		}
		issue.Category = "vulnerability"
		data.TopIssues = append(data.TopIssues, issue)
	}
}

// GetTrends retrieves trend data for a metric.
func (s *AggregatorService) GetTrends(ctx context.Context, metric, period string) ([]DataPoint, error) {
	// Determine time range and grouping
	now := time.Now()
	var startDate time.Time
	var groupFormat string

	switch period {
	case PeriodDaily:
		startDate = now.AddDate(0, 0, -1)
		groupFormat = "%Y-%m-%d %H:00"
	case PeriodWeekly:
		startDate = now.AddDate(0, 0, -7)
		groupFormat = sqliteDateFormat
	case PeriodMonthly:
		startDate = now.AddDate(0, -1, 0)
		groupFormat = sqliteDateFormat
	default:
		startDate = now.AddDate(0, 0, -7)
		groupFormat = sqliteDateFormat
	}

	var query string
	switch metric {
	case "latency":
		query = fmt.Sprintf(`
			SELECT strftime('%s', timestamp) as period, AVG(latency_ms)
			FROM gateway_results
			WHERE timestamp >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
	case "bandwidth":
		query = fmt.Sprintf(`
			SELECT strftime('%s', timestamp) as period, AVG(download_mbps)
			FROM speedtest_results
			WHERE timestamp >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
	case "devices":
		query = fmt.Sprintf(`
			SELECT strftime('%s', last_seen) as period, COUNT(*)
			FROM devices
			WHERE last_seen >= ?
			GROUP BY period
			ORDER BY period
		`, groupFormat)
	default:
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}

	rows, err := s.db.Query(ctx, query, startDate.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("querying trends: %w", err)
	}
	defer rows.Close()

	var points []DataPoint
	for rows.Next() {
		var periodStr string
		var value float64
		if scanErr := rows.Scan(&periodStr, &value); scanErr != nil {
			continue
		}

		t, _ := time.Parse("2006-01-02", periodStr)
		points = append(points, DataPoint{
			Timestamp: t,
			Value:     value,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating trend data: %w", rowsErr)
	}

	return points, nil
}
