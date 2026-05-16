package api

// handlers_survey_floorplan.go contains the floor-plan upload, survey-settings
// update, and AirMapper (.amp) import handlers for the legacy single-floor
// path. Multi-floor variants live in handlers_survey_floors_data.go.

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// UpdateFloorPlanRequest contains floor plan image and dimension parameters.
type UpdateFloorPlanRequest struct {
	ImageData string  `json:"imageData"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	ScaleM    float64 `json:"scaleM"`
}

// validateFloorPlanRequest validates floor plan request fields.
func validateFloorPlanRequest(req *UpdateFloorPlanRequest, maxSize int64) error {
	if err := validation.ValidateImageDataURL(req.ImageData, int(maxSize)); err != nil {
		return err
	}
	if err := validation.ValidateIntRange(req.Width, "width", 1, floorPlanMaxDimension); err != nil {
		return err
	}
	if err := validation.ValidateIntRange(req.Height, "height", 1, floorPlanMaxDimension); err != nil {
		return err
	}
	return validation.ValidateFloatRange(req.ScaleM, "scaleM", floorPlanMinScale, floorPlanMaxScale)
}

func (s *Server) updateSurveyFloorPlan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.invalidId"),
			"",
		)
		return
	}

	// Rate limit file uploads (fixes #696)
	if !s.endpointRateLimiter().Allow(s.getClientIP(r)) {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusTooManyRequests,
			ErrCodeRateLimit,
			localizer.T("errors.survey.rateLimitExceeded"),
			"",
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeFloorPlan)
	var req UpdateFloorPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	// Validate floor plan fields (fixes #695)
	if err := validateFloorPlanRequest(&req, MaxBodySizeFloorPlan); err != nil {
		logger.WarnContext(r.Context(), "Survey validation failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.validationFailed"),
			"",
		)
		return
	}

	floorPlan := &survey.FloorPlan{
		ImageData: req.ImageData,
		Width:     req.Width,
		Height:    req.Height,
		ScaleM:    req.ScaleM,
	}

	if err := s.surveyManager().UpdateFloorPlan(id, floorPlan); err != nil {
		logger.WarnContext(r.Context(), "Survey not found", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			localizer.T("errors.survey.notFound"),
			"",
		)
		return
	}

	updatedSurvey, err := s.surveyManager().GetSurvey(id)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get survey", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.survey.getSurveyFailed"),
			"",
		)
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, updatedSurvey)
}

// UpdateSurveySettingsRequest is the request body for updating survey settings.
type UpdateSurveySettingsRequest struct {
	SurveyType   string `json:"surveyType"`
	IperfServer  string `json:"iperfServer,omitempty"`
	TestDuration int    `json:"testDuration,omitempty"`
}

// updateSurveySettings handles PUT /api/survey/settings?id=xxx.
func (s *Server) updateSurveySettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := &surveyHandlerContext{w, logger, localizer}

	if r.Method != http.MethodPut {
		ctx.sendMethodNotAllowed()
		return
	}

	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		ctx.sendValidationError("errors.survey.invalidId")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)
	var req UpdateSurveySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid request body", "error", err)
		ctx.sendBadRequestError("errors.api.invalidRequestBody")
		return
	}

	// Validate survey type
	surveyType := survey.Type(req.SurveyType)
	validTypes := []survey.Type{survey.TypePassive, survey.TypeActive, survey.TypeThroughput}
	if !slices.Contains(validTypes, surveyType) {
		ctx.sendValidationError("errors.survey.invalidType")
		return
	}

	// Validate optional fields
	if req.IperfServer != "" && validation.ValidateServerAddress(req.IperfServer) != nil {
		logger.WarnContext(r.Context(), "Survey validation failed: invalid iperf server")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}
	if req.TestDuration != 0 &&
		validation.ValidateIntRange(req.TestDuration, "testDuration", 1, testDurationMaxSec) != nil {
		logger.WarnContext(r.Context(), "Survey validation failed: invalid test duration")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}

	if err := s.surveyManager().UpdateSurveySettings(id, surveyType, req.IperfServer, req.TestDuration); err != nil {
		logger.ErrorContext(r.Context(), "Failed to update survey", "error", err)
		ctx.sendBadRequestError("errors.survey.updateFailed")
		return
	}

	settingsUpdatedSurvey, err := s.surveyManager().GetSurvey(id)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get survey", "error", err)
		ctx.sendInternalError("errors.survey.getSurveyFailed")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, settingsUpdatedSurvey)
}

// importAirMapper handles POST /api/survey/import/airmapper.
// It accepts a multipart form with an .amp file and returns parsed calibration,
// floor plan, and pass/fail criteria data.
func (s *Server) importAirMapper(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := &surveyHandlerContext{w, logger, localizer}

	if r.Method != http.MethodPost {
		ctx.sendMethodNotAllowed()
		return
	}

	if !s.endpointRateLimiter().Allow(s.getClientIP(r)) {
		ctx.sendRateLimitError()
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeAirMapper)
	// #nosec G120 -- request body bounded above via http.MaxBytesReader(MaxBodySizeAirMapper).
	if err := r.ParseMultipartForm(MaxBodySizeAirMapper); err != nil {
		logger.WarnContext(r.Context(), "Survey file too large", "error", err)
		ctx.sendBadRequestError("errors.survey.fileTooLarge")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.WarnContext(r.Context(), "No file provided for survey", "error", err)
		ctx.sendBadRequestError("errors.survey.noFileProvided")
		return
	}
	defer func() { _ = file.Close() }()

	if validation.ValidateFilename(handler.Filename, "filename") != nil {
		logger.WarnContext(r.Context(), "Survey validation failed: invalid filename")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}

	if !strings.HasSuffix(strings.ToLower(handler.Filename), ".amp") {
		ctx.sendValidationError("errors.survey.invalidFileType")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to read survey file", "error", err)
		ctx.sendInternalError("errors.survey.readFileFailed")
		return
	}

	ampFile, err := survey.ParseAirMapperFile(data)
	if err != nil {
		logger.WarnContext(r.Context(), "Failed to parse survey file", "error", err)
		ctx.sendBadRequestError("errors.survey.parseFileFailed")
		return
	}

	result, err := ampFile.ToImportResult()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to process survey file", "error", err)
		ctx.sendInternalError("errors.survey.processFileFailed")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, result)
}
