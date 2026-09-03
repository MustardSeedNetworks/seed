package api

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// reportsPathPrefix is the collection path with its trailing slash, so
// handleReportByID can recover the id (and optional /download suffix).
const reportsPathPrefix = APIVersionPrefix + "/reports/"

// downloadSuffix marks the variant of GET /reports/{id} that streams the file
// rather than returning the record.
const downloadSuffix = "/download"

// reportIDPattern matches the uuid.New().String() ids Generate assigns.
//
// r.URL.Path arrives percent-decoded, so without this an id could carry a
// newline and forge log lines. It is also plain input validation: nothing but a
// UUID can name a report.
var reportIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ReportInfo is a generated report as the API presents it.
//
// Deliberately not reporting.Report: that carries FilePath, an absolute path on
// the server's filesystem. It is useful to the generator and to nobody else, so
// it does not cross the wire.
type ReportInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	Status      string `json:"status"`
	FileSize    int64  `json:"fileSize,omitempty"`
	CreatedAt   string `json:"createdAt"`
	CompletedAt string `json:"completedAt,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ReportsResponse is the payload of GET /api/v1/reports.
type ReportsResponse struct {
	Reports []ReportInfo `json:"reports"`
}

// GenerateReportRequest is the body of POST /api/v1/reports/generate.
type GenerateReportRequest struct {
	Type   string `json:"type"   validate:"required"`
	Format string `json:"format" validate:"required"`
}

func toReportInfo(r *reporting.Report) ReportInfo {
	info := ReportInfo{
		ID:          r.ID,
		Name:        r.Name,
		Type:        string(r.Type),
		Format:      string(r.Format),
		Status:      string(r.Status),
		FileSize:    r.FileSize,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		CompletedAt: "",
		ExpiresAt:   "",
		Error:       r.Error,
	}
	if r.CompletedAt != nil {
		info.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}
	if r.ExpiresAt != nil {
		info.ExpiresAt = r.ExpiresAt.Format(time.RFC3339)
	}
	return info
}

// reportGenerator resolves the generator, or writes the unavailable response
// and reports false. Reporting is optional at construction (cmd_serve wires it,
// tests often do not), so every handler goes through here.
func (s *Server) reportGenerator(
	w http.ResponseWriter,
	r *http.Request,
) (*reporting.GeneratorService, bool) {
	if s.background == nil || s.background.Reporting == nil {
		sendErrorResponseWithDetails(
			w,
			logging.FromContext(r.Context()),
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			i18n.FromRequest(r).T("errors.reports.unavailable"),
			"",
		)
		return nil, false
	}
	return s.background.Reporting.Generator(), true
}

// handleReports serves GET /api/v1/reports.
func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	gen, ok := s.reportGenerator(w, r)
	if !ok {
		return
	}

	reports, err := gen.ListReports(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "listing reports failed",
			"event", "report.list.failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, i18n.FromRequest(r).T("errors.reports.listFailed"), "")
		return
	}

	resp := ReportsResponse{Reports: make([]ReportInfo, 0, len(reports))}
	for i := range reports {
		resp.Reports = append(resp.Reports, toReportInfo(&reports[i]))
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleReportGenerate serves POST /api/v1/reports/generate.
//
// Generation is asynchronous: the record is written before Generate returns and
// the content is produced afterwards, so this answers 202 with the pending
// record. Clients re-read the collection rather than holding a request open.
func (s *Server) handleReportGenerate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	var req GenerateReportRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, i18n.FromRequest(r)) {
		return
	}

	reportType := reporting.ReportType(req.Type)
	format := reporting.ExportFormat(req.Format)
	if !reporting.IsValidReportType(reportType) || !reporting.IsValidExportFormat(format) {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, i18n.FromRequest(r).T("errors.reports.invalidRequest"), "")
		return
	}

	// The route is gated on export_csv_json, which is Starter. The PDF audit
	// report is sold as Pro (audit_pdf), and PDF is a value of `format` rather
	// than a path of its own, so the route gate cannot express it and a Starter
	// licence could generate one (#2327).
	if format == reporting.FormatPDF && !s.hasFeature("audit_pdf") {
		s.sendFeatureGate(w, r, "audit_pdf")

		return
	}

	// Availability is resolved after entitlement so the answer to "may I" does
	// not depend on whether the reporting service happens to be up.
	gen, ok := s.reportGenerator(w, r)
	if !ok {
		return
	}

	report, err := gen.Generate(r.Context(), reportType, format, nil)
	if err != nil {
		logger.ErrorContext(r.Context(), "generating report failed",
			"event", "report.generate.failed",
			"report_type", string(reportType),
			"format", string(format),
			"error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, i18n.FromRequest(r).T("errors.reports.generateFailed"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusAccepted, toReportInfo(report))
}

// handleReportByID serves GET and DELETE on /api/v1/reports/{id}, plus the
// /download variant of GET.
func (s *Server) handleReportByID(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	rest := strings.TrimPrefix(r.URL.Path, reportsPathPrefix)
	download := strings.HasSuffix(rest, downloadSuffix)
	id := strings.TrimSuffix(rest, downloadSuffix)
	if !reportIDPattern.MatchString(id) {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, i18n.FromRequest(r).T("errors.reports.invalidID"), "")
		return
	}

	gen, ok := s.reportGenerator(w, r)
	if !ok {
		return
	}

	switch {
	case r.Method == http.MethodGet && download:
		s.downloadReport(w, r, gen, id)
	case r.Method == http.MethodGet:
		s.getReport(w, r, gen, id)
	case r.Method == http.MethodDelete:
		s.deleteReport(w, r, gen, id)
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeValidation, i18n.FromRequest(r).T("errors.methodNotAllowed"), "")
	}
}

func (s *Server) getReport(
	w http.ResponseWriter,
	r *http.Request,
	gen *reporting.GeneratorService,
	id string,
) {
	logger := logging.FromContext(r.Context())

	report, err := gen.GetReport(r.Context(), id)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, i18n.FromRequest(r).T("errors.reports.notFound"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, toReportInfo(report))
}

func (s *Server) deleteReport(
	w http.ResponseWriter,
	r *http.Request,
	gen *reporting.GeneratorService,
	id string,
) {
	logger := logging.FromContext(r.Context())

	if err := gen.DeleteReport(r.Context(), id); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, i18n.FromRequest(r).T("errors.reports.notFound"), "")
		return
	}

	// No report_id field: the logging middleware already records r.URL.Path,
	// which is where the id came from.
	logger.InfoContext(r.Context(), "report deleted", "event", "report.deleted")

	w.WriteHeader(http.StatusNoContent)
}

// downloadReport streams the generated file.
func (s *Server) downloadReport(
	w http.ResponseWriter,
	r *http.Request,
	gen *reporting.GeneratorService,
	id string,
) {
	logger := logging.FromContext(r.Context())

	report, err := gen.GetReport(r.Context(), id)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, i18n.FromRequest(r).T("errors.reports.notFound"), "")
		return
	}

	body, err := gen.DownloadReport(r.Context(), id)
	if err != nil {
		// A report that is still generating has no file yet, which is a
		// different answer from one whose file is missing.
		status := http.StatusNotFound
		if report.Status != reporting.StatusComplete {
			status = http.StatusConflict
		}
		logger.WarnContext(r.Context(), "report download failed",
			"event", "report.download.failed",
			"status", string(report.Status),
			"error", err)
		sendErrorResponseWithDetails(w, logger, status,
			ErrCodeNotFound, i18n.FromRequest(r).T("errors.reports.notReady"), "")
		return
	}
	defer func() { _ = body.Close() }()

	// The filename is built from the report's UUID and format, never from user
	// input, so it is safe to echo back.
	w.Header().Set("Content-Type", reportContentType(report.Format))
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+report.ID+`.`+string(report.Format)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if _, copyErr := io.Copy(w, body); copyErr != nil && !errors.Is(copyErr, http.ErrBodyNotAllowed) {
		// Headers are already out, so there is no error response left to send.
		logger.WarnContext(r.Context(), "report download truncated",
			"event", "report.download.truncated", "error", copyErr)
	}
}

// reportContentType maps a report format to its MIME type.
func reportContentType(format reporting.ExportFormat) string {
	switch format {
	case reporting.FormatPDF:
		return "application/pdf"
	case reporting.FormatHTML:
		return "text/html; charset=utf-8"
	case reporting.FormatCSV:
		return "text/csv; charset=utf-8"
	case reporting.FormatJSON:
		return "application/json"
	case reporting.FormatExcel:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case reporting.FormatMarkdown:
		return "text/markdown; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
