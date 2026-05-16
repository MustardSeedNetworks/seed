package api

// handlers_survey_floors.go contains the multi-floor management handlers
// (#654): list / add / get / update / delete a floor, plus the method
// dispatchers that route by HTTP verb.

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// handleSurveyFloors dispatches floor list/add requests based on HTTP method.
func (s *Server) handleSurveyFloors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listFloors(w, r)
	case http.MethodPost:
		s.addFloor(w, r)
	default:
		logger := logging.FromContext(r.Context())
		localizer := i18n.FromRequest(r)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
	}
}

// handleSurveyFloor dispatches single floor requests based on HTTP method.
func (s *Server) handleSurveyFloor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getFloor(w, r)
	case http.MethodPut:
		s.updateFloor(w, r)
	case http.MethodDelete:
		s.deleteFloor(w, r)
	default:
		logger := logging.FromContext(r.Context())
		localizer := i18n.FromRequest(r)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
	}
}

// isValidFloorID validates a floor ID to prevent injection attacks.
func isValidFloorID(id string) bool {
	return validation.ValidateSurveyID(id) == nil // Same rules as survey ID
}

// AddFloorRequest contains parameters for adding a new floor.
type AddFloorRequest struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// listFloors handles GET /api/survey/floors?id=xxx.
// Returns all floors for a survey.
func (s *Server) listFloors(w http.ResponseWriter, r *http.Request) {
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

	floors, err := s.surveyManager().GetFloors(surveyID)
	if err != nil {
		logger.WarnContext(r.Context(), "Survey not found", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			localizer.T("errors.survey.notFound"),
			"",
		) // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floors)
}

// addFloor handles POST /api/survey/floors?id=xxx.
// Adds a new floor to a survey.
func (s *Server) addFloor(w http.ResponseWriter, r *http.Request) {
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

	var req AddFloorRequest
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

	// Validate floor name
	if err := validation.ValidateStringLength(req.Name, "name", 1, floorNameMaxLength); err != nil {
		logger.WarnContext(r.Context(), "Survey validation failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.validationFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	// Validate floor level
	if err := validation.ValidateIntRange(req.Level, "level", floorLevelMin, floorLevelMax); err != nil {
		logger.WarnContext(r.Context(), "Survey validation failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.validationFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	floor, err := s.surveyManager().AddFloor(surveyID, req.Name, req.Level)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to add floor", "survey_id", surveyID, "error", err)
		logger.ErrorContext(r.Context(), "Internal error", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.api.internalError"),
			"",
		) // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// getFloor handles GET /api/survey/floor?id=xxx&floorId=yyy.
// Returns a specific floor.
func (s *Server) getFloor(w http.ResponseWriter, r *http.Request) {
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

	floor, err := s.surveyManager().GetFloor(surveyID, floorID)
	if err != nil {
		logger.WarnContext(r.Context(), "Survey not found", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			localizer.T("errors.survey.notFound"),
			"",
		) // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// UpdateFloorRequest contains parameters for updating floor metadata.
type UpdateFloorRequest struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// updateFloor handles PUT /api/survey/floor?id=xxx&floorId=yyy.
// Updates floor metadata (name, level).
func (s *Server) updateFloor(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := &surveyHandlerContext{w, logger, localizer}

	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) {
		ctx.sendValidationError("errors.survey.invalidId")
		return
	}
	if !isValidFloorID(floorID) {
		ctx.sendValidationError("errors.survey.invalidId")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req UpdateFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid request body", "error", err)
		ctx.sendBadRequestError("errors.api.invalidRequestBody")
		return
	}

	// Validate floor name and level
	if validation.ValidateStringLength(req.Name, "name", 1, floorNameMaxLength) != nil {
		logger.WarnContext(r.Context(), "Survey validation failed: invalid floor name")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}
	if validation.ValidateIntRange(req.Level, "level", floorLevelMin, floorLevelMax) != nil {
		logger.WarnContext(r.Context(), "Survey validation failed: invalid floor level")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}

	if err := s.surveyManager().UpdateFloor(surveyID, floorID, req.Name, req.Level); err != nil {
		logger.ErrorContext(r.Context(),
			"Failed to update floor",
			"survey_id",
			surveyID,
			"floor_id",
			floorID,
			"error",
			err,
		)
		ctx.sendInternalError("errors.survey.updateFailed")
		return
	}

	floor, err := s.surveyManager().GetFloor(surveyID, floorID)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get floor", "error", err)
		ctx.sendInternalError("errors.survey.getSurveyFailed")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// deleteFloor handles DELETE /api/survey/floor?id=xxx&floorId=yyy.
// Removes a floor from a survey.
func (s *Server) deleteFloor(w http.ResponseWriter, r *http.Request) {
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

	if err := s.surveyManager().DeleteFloor(surveyID, floorID); err != nil {
		logger.ErrorContext(r.Context(),
			"Failed to delete floor",
			"survey_id",
			surveyID,
			"floor_id",
			floorID,
			"error",
			err,
		)
		logger.ErrorContext(r.Context(), "Failed to delete survey", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.survey.deleteFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, statusResponse{Status: statusDeleted})
}
