package api

// handlers_survey_report.go contains the survey PDF report generation
// handler (#653).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// GenerateReportRequest contains parameters for report generation.
type GenerateReportRequest struct {
	IncludeHeatmaps         bool   `json:"includeHeatmaps"`
	IncludeRawData          bool   `json:"includeRawData"`
	IncludeRecommendations  bool   `json:"includeRecommendations"`
	IncludeExecutiveSummary bool   `json:"includeExecutiveSummary"`
	CompanyName             string `json:"companyName,omitempty"`
}

// generateSurveyReport handles POST /api/survey/report?id=xxx.
// Generates a PDF report for the specified survey.
func (s *Server) generateSurveyReport(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	surveyID := r.URL.Query().Get("id")
	if !isValidSurveyID(surveyID) {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.invalidId"),
			"",
		) // fixes #694
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	// Parse options from request body (all optional)
	var req GenerateReportRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.WarnContext(r.Context(), "Invalid request body for report generation", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusBadRequest,
				ErrCodeBadRequest,
				localizer.T("errors.api.invalidRequestBody"),
				"",
			) // fixes #694, #H7
			return
		}
	}

	// Validate company name length
	if len(req.CompanyName) > companyNameMaxLength {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.validationFailed"),
			"company name too long (max 100 characters)",
		) // fixes #694
		return
	}

	// Convert to survey package options
	options := survey.ReportOptions{
		IncludeHeatmaps:         req.IncludeHeatmaps,
		IncludeRawData:          req.IncludeRawData,
		IncludeRecommendations:  req.IncludeRecommendations,
		IncludeExecutiveSummary: req.IncludeExecutiveSummary,
		CompanyName:             req.CompanyName,
	}

	// Use defaults if nothing specified
	if !options.IncludeHeatmaps && !options.IncludeRawData &&
		!options.IncludeRecommendations && !options.IncludeExecutiveSummary {
		options = survey.DefaultReportOptions()
	}

	pdfBytes, err := s.surveyManager().GenerateReport(surveyID, options)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate report",
			"survey_id", surveyID,
			"error", err)
		logger.ErrorContext(r.Context(), "Failed to export survey", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.survey.exportFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	// Return PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().
		Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"survey-report-%s.pdf\"", surveyID[:8]))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	// #nosec G705 -- response body is a server-generated PDF; Content-Type already set to application/pdf above.
	if _, writeErr := w.Write(pdfBytes); writeErr != nil {
		logger.ErrorContext(r.Context(), "Failed to write PDF response", "error", writeErr)
	}
}
