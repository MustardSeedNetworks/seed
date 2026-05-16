package api

// handlers_survey_floors_data.go contains the per-floor data-modification
// handlers: setting the active floor, replacing a floor's floor-plan image,
// and adding a sample to a specific floor.

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// SetActiveFloorRequest contains parameters for setting the active floor.
type SetActiveFloorRequest struct {
	FloorID string `json:"floorId"`
}

// setActiveFloor handles PUT /api/survey/active-floor?id=xxx.
// Sets the active floor for data collection.
func (s *Server) setActiveFloor(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

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

	var req SetActiveFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid request body", "error", err)
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

	if !isValidFloorID(req.FloorID) {
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

	if err := s.surveyManager().SetActiveFloor(surveyID, req.FloorID); err != nil {
		logger.ErrorContext(r.Context(),
			"Failed to set active floor",
			"survey_id",
			surveyID,
			"floor_id",
			req.FloorID,
			"error",
			err,
		)
		logger.ErrorContext(r.Context(), "Failed to update survey", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.survey.updateFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	// Return the updated survey
	updatedSurvey, err := s.surveyManager().GetSurvey(surveyID)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get survey", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.survey.getSurveyFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, updatedSurvey)
}

// updateFloorFloorPlan handles PUT /api/survey/floor/floorplan?id=xxx&floorId=yyy.
// Updates the floor plan for a specific floor.
func (s *Server) updateFloorFloorPlan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) || !isValidFloorID(floorID) {
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
	if err := s.surveyManager().UpdateFloorPlanByFloorID(surveyID, floorID, floorPlan); err != nil {
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

	floor, err := s.surveyManager().GetFloor(surveyID, floorID)
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
	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// AddFloorSampleRequest contains parameters for adding a sample to a specific floor.
type AddFloorSampleRequest struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	SampleData any `json:"sampleData"`
}

// addFloorSample handles POST /api/survey/floor/sample?id=xxx&floorId=yyy.
// Adds a sample to a specific floor.
func (s *Server) addFloorSample(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

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
	if !isValidFloorID(floorID) {
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

	var req AddFloorSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid request body", "error", err)
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

	// Process the sample data to convert to typed sample
	sampleData := processSampleData(r.Context(), req.SampleData, surveyID, logger)

	if err := s.surveyManager().AddSampleToFloor(surveyID, floorID, req.X, req.Y, sampleData); err != nil {
		logger.ErrorContext(r.Context(),
			"Failed to add sample to floor",
			"error",
			err,
			"survey_id",
			surveyID,
			"floor_id",
			floorID,
		)
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

	sendJSONResponse(w, logger, http.StatusOK, statusResponse{Status: statusSampleAdded})
}
