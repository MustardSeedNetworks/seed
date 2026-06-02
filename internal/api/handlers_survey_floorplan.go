package api

// handlers_survey_floorplan.go contains the floor-plan upload, survey-settings
// update, and AirMapper (.amp) import handlers for the legacy single-floor
// path. Multi-floor variants live in handlers_survey_floors_data.go.

import (
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
	"github.com/krisarmstrong/seed/internal/wifi/survey"
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
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON,
		logger, localizer) {
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

	var req UpdateSurveySettingsRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
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

// UpdateSurveyImportedDataRequest is the payload for #727's imported-data
// endpoint. Any field left nil is left unchanged on the survey; an empty
// slice clears the corresponding list.
type UpdateSurveyImportedDataRequest struct {
	APLocations      []APLocation        `json:"apLocations"`
	ClientLocations  []ClientLocation    `json:"clientLocations"`
	PassFailCriteria []PassFailCriterion `json:"passFailCriteria"`
}

// APLocation is the flat transport view of survey.APLocation. It and its
// siblings (ClientLocation, PassFailCriterion) mirror the survey domain's
// imported-data value objects so the published request schema does not depend
// on the survey package.
type APLocation struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Label    string `json:"label,omitempty"`
	BSSID    string `json:"bssid,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	Notes    string `json:"notes,omitempty"`
	Imported bool   `json:"imported,omitempty"`
}

// ClientLocation is the flat transport view of survey.ClientLocation.
type ClientLocation struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Label    string `json:"label,omitempty"`
	MAC      string `json:"mac,omitempty"`
	Imported bool   `json:"imported,omitempty"`
}

// PassFailCriterion is the flat transport view of survey.PassFailCriterion.
type PassFailCriterion struct {
	Option   string  `json:"option"`
	Name     string  `json:"name,omitempty"`
	Limit    float64 `json:"limit"`
	Suffix   string  `json:"suffix,omitempty"`
	Enabled  bool    `json:"enabled"`
	Mode     string  `json:"mode,omitempty"`
	APCount  int     `json:"ap,omitempty"`
	Imported bool    `json:"imported,omitempty"`
}

// toSurveyAPLocations maps inbound AP placements onto the survey domain type.
// A nil input stays nil (the field is left unchanged); a non-nil input maps
// element-for-element (an empty slice clears the list) — see the request doc.
func toSurveyAPLocations(in []APLocation) []survey.APLocation {
	if in == nil {
		return nil
	}
	out := make([]survey.APLocation, len(in))
	for i, l := range in {
		out[i] = survey.APLocation{
			ID:       l.ID,
			X:        l.X,
			Y:        l.Y,
			Label:    l.Label,
			BSSID:    l.BSSID,
			Vendor:   l.Vendor,
			Notes:    l.Notes,
			Imported: l.Imported,
		}
	}
	return out
}

// toSurveyClientLocations maps inbound client placements onto the survey type,
// preserving nil-vs-empty semantics.
func toSurveyClientLocations(in []ClientLocation) []survey.ClientLocation {
	if in == nil {
		return nil
	}
	out := make([]survey.ClientLocation, len(in))
	for i, l := range in {
		out[i] = survey.ClientLocation{
			ID:       l.ID,
			X:        l.X,
			Y:        l.Y,
			Label:    l.Label,
			MAC:      l.MAC,
			Imported: l.Imported,
		}
	}
	return out
}

// toSurveyPassFailCriteria maps inbound pass/fail criteria onto the survey
// type, preserving nil-vs-empty semantics.
func toSurveyPassFailCriteria(in []PassFailCriterion) []survey.PassFailCriterion {
	if in == nil {
		return nil
	}
	out := make([]survey.PassFailCriterion, len(in))
	for i, c := range in {
		out[i] = survey.PassFailCriterion{
			Option:   c.Option,
			Name:     c.Name,
			Limit:    c.Limit,
			Suffix:   c.Suffix,
			Enabled:  c.Enabled,
			Mode:     c.Mode,
			APCount:  c.APCount,
			Imported: c.Imported,
		}
	}
	return out
}

// updateSurveyImportedData handles PUT /api/wifi/survey/imported-data?id=xxx.
// Replaces the survey's AP/client placements and pass-fail criteria. This is
// what the AirMapper import flow uses to persist parsed survey data so it
// survives reload (#727).
func (s *Server) updateSurveyImportedData(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateSurveyImportedDataRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	update := survey.ImportedDataUpdate{
		APLocations:      toSurveyAPLocations(req.APLocations),
		ClientLocations:  toSurveyClientLocations(req.ClientLocations),
		PassFailCriteria: toSurveyPassFailCriteria(req.PassFailCriteria),
	}
	if err := s.surveyManager().UpdateImportedData(id, update); err != nil {
		logger.ErrorContext(r.Context(), "Failed to update imported data", "error", err)
		ctx.sendBadRequestError("errors.survey.updateFailed")
		return
	}

	updated, err := s.surveyManager().GetSurvey(id)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get survey after imported-data update", "error", err)
		ctx.sendInternalError("errors.survey.getSurveyFailed")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, updated)
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
