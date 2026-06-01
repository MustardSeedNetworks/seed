package harvest

import (
	"context"
	"time"
)

// Export internal functions and methods for testing.

// ExportCalculateNextRun exposes the internal calculateNextRun function for testing.
func ExportCalculateNextRun(schedule *Schedule) *time.Time {
	return calculateNextRun(schedule)
}

// ExportGenerateHTML exposes the generateHTML method for testing.
func (s *GeneratorService) ExportGenerateHTML(report *Report, data *AggregatedData) ([]byte, error) {
	return s.generateHTML(report, data)
}

// ExportGenerateCSV exposes the generateCSV method for testing.
func (s *GeneratorService) ExportGenerateCSV(report *Report, data *AggregatedData) ([]byte, error) {
	return s.generateCSV(report, data)
}

// ExportGenerateJSON exposes the generateJSON method for testing.
func (s *GeneratorService) ExportGenerateJSON(report *Report, data *AggregatedData) ([]byte, error) {
	return s.generateJSON(report, data)
}

// ExportGeneratePDF exposes the generatePDF method for testing.
func (s *GeneratorService) ExportGeneratePDF(report *Report, data *AggregatedData) ([]byte, error) {
	return s.generatePDF(report, data)
}

// ExportDataToCSV exposes the dataToCSV method for testing.
func (s *GeneratorService) ExportDataToCSV(data any) ([]byte, error) {
	return s.dataToCSV(data)
}

// ExportExportDevices exposes the exportDevices method for testing.
func (s *GeneratorService) ExportExportDevices(
	ctx context.Context,
	req *ExportRequest,
) (any, int, error) {
	return s.exportDevices(ctx, req)
}

// ExportExportVulnerabilities exposes the exportVulnerabilities method for testing.
func (s *GeneratorService) ExportExportVulnerabilities(
	ctx context.Context,
	req *ExportRequest,
) (any, int, error) {
	return s.exportVulnerabilities(ctx, req)
}

// ExportSaveReportFile exposes the saveReportFile method for testing.
func (s *GeneratorService) ExportSaveReportFile(report *Report, content []byte) error {
	return s.saveReportFile(report, content)
}

// ExportFailReport exposes the failReport method for testing.
func (s *GeneratorService) ExportFailReport(ctx context.Context, report *Report, errMsg string) {
	s.failReport(ctx, report, errMsg)
}

// ExportSaveReport exposes the saveReport method for testing.
func (s *GeneratorService) ExportSaveReport(ctx context.Context, report *Report) error {
	return s.saveReport(ctx, report)
}

// ExportAggregateVulnerabilities exposes the aggregateVulnerabilities method for testing.
func (s *AggregatorService) ExportAggregateVulnerabilities(
	ctx context.Context,
	data *AggregatedData,
	since time.Time,
) {
	s.aggregateVulnerabilities(ctx, data, since)
}

// ExportAggregatePerformance exposes the aggregatePerformance method for testing.
func (s *AggregatorService) ExportAggregatePerformance(
	ctx context.Context,
	data *AggregatedData,
	since time.Time,
) {
	s.aggregatePerformance(ctx, data, since)
}

// ExportAggregateTopIssues exposes the aggregateTopIssues method for testing.
func (s *AggregatorService) ExportAggregateTopIssues(ctx context.Context, data *AggregatedData) {
	s.aggregateTopIssues(ctx, data)
}

// ExportLoadSchedules exposes the loadSchedules method for testing.
func (s *SchedulerService) ExportLoadSchedules(ctx context.Context) error {
	return s.loadSchedules(ctx)
}

// ExportCheckSchedules exposes the checkSchedules method for testing.
func (s *SchedulerService) ExportCheckSchedules(ctx context.Context) {
	s.checkSchedules(ctx)
}

// ExportRunScheduledReport exposes the runScheduledReport method for testing.
func (s *SchedulerService) ExportRunScheduledReport(ctx context.Context, schedule *ScheduledReport) {
	s.runScheduledReport(ctx, schedule)
}

// ExportSaveSchedule exposes the saveSchedule method for testing.
func (s *SchedulerService) ExportSaveSchedule(ctx context.Context, sr *ScheduledReport) error {
	return s.saveSchedule(ctx, sr)
}

// ExportLoadBuiltInTemplates exposes the loadBuiltInTemplates method for testing.
func (s *TemplateService) ExportLoadBuiltInTemplates() {
	s.loadBuiltInTemplates()
}

// SetReportsPath allows setting a custom reports path for testing.
func (s *GeneratorService) SetReportsPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reportsPath = path
}

// GetReportsPath returns the current reports path.
func (s *GeneratorService) GetReportsPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reportsPath
}
