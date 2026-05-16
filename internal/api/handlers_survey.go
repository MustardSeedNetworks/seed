package api

// handlers_survey.go contains the WiFi-survey CRUD and state-machine handlers
// (create, list, get, delete, start, pause, complete) plus the validation
// constants, shared types, and the small surveyHandlerContext helper that
// every survey-related file in the api package uses. Sample ingest, floor
// plans, multi-floor management, dead-zone analysis, heatmaps, and report
// generation each live in their own handlers_survey_*.go file.

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
)

// Survey validation limits.
const (
	// surveyNameMaxLength is the maximum length for survey names.
	surveyNameMaxLength = 100

	// floorPlanMaxDimension is the maximum width/height for floor plan images in pixels.
	floorPlanMaxDimension = 10000

	// floorPlanMinScale is the minimum scale value in meters per pixel.
	floorPlanMinScale = 0.01

	// floorPlanMaxScale is the maximum scale value in meters per pixel.
	floorPlanMaxScale = 1000.0

	// testDurationMaxSec is the maximum survey test duration in seconds.
	testDurationMaxSec = 300

	// floorNameMaxLength is the maximum length for floor names.
	floorNameMaxLength = 50

	// floorLevelMax is the maximum building floor level (typical skyscraper height).
	floorLevelMax = 200

	// floorLevelMin is the minimum building floor level (deep basements).
	floorLevelMin = -10

	// surveyDescriptionMaxLength is the maximum length for survey descriptions.
	surveyDescriptionMaxLength = 500

	// companyNameMaxLength is the maximum length for company names in reports.
	companyNameMaxLength = 100
)

// ============================================================================
// WiFi Survey API Handlers (fixes #544 - split from handlers_discovery.go)
// ============================================================================

// CreateSurveyRequest contains parameters for creating a new WiFi survey.
type CreateSurveyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SurveyType  string `json:"surveyType"`
	Interface   string `json:"interface"`
}

// isValidSurveyID validates a survey ID to prevent injection attacks (fixes #695).
// Valid survey IDs must be:
// - Non-empty
// - At most 64 characters long
// - Contain only alphanumeric characters, dashes, and underscores.
func isValidSurveyID(id string) bool {
	// Use the centralized validation function (fixes #695)
	return validation.ValidateSurveyID(id) == nil
}

// surveyHandlerContext bundles common handler dependencies for cleaner code.
type surveyHandlerContext struct {
	w         http.ResponseWriter
	logger    *slog.Logger
	localizer *i18n.Localizer
}

// sendValidationError sends a validation error response.
func (c *surveyHandlerContext) sendValidationError(msgKey string) {
	sendErrorResponseWithDetails(
		c.w,
		c.logger,
		http.StatusBadRequest,
		ErrCodeValidation,
		c.localizer.T(msgKey),
		"",
	)
}

// sendBadRequestError sends a bad request error response.
func (c *surveyHandlerContext) sendBadRequestError(msgKey string) {
	sendErrorResponseWithDetails(
		c.w,
		c.logger,
		http.StatusBadRequest,
		ErrCodeBadRequest,
		c.localizer.T(msgKey),
		"",
	)
}

// sendInternalError sends an internal server error response.
func (c *surveyHandlerContext) sendInternalError(msgKey string) {
	sendErrorResponseWithDetails(
		c.w,
		c.logger,
		http.StatusInternalServerError,
		ErrCodeInternal,
		c.localizer.T(msgKey),
		"",
	)
}

// sendMethodNotAllowed sends a method not allowed error response.
func (c *surveyHandlerContext) sendMethodNotAllowed() {
	sendErrorResponseWithDetails(
		c.w,
		c.logger,
		http.StatusMethodNotAllowed,
		ErrCodeMethodNotAllowed,
		c.localizer.T("errors.api.methodNotAllowed"),
		"",
	)
}

// sendRateLimitError sends a rate limit error response and returns true.
func (c *surveyHandlerContext) sendRateLimitError() bool {
	sendErrorResponseWithDetails(
		c.w,
		c.logger,
		http.StatusTooManyRequests,
		ErrCodeRateLimit,
		c.localizer.T("errors.survey.rateLimitExceeded"),
		"",
	)
	return true
}

func (s *Server) createSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req CreateSurveyRequest
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

	// Validate survey name (fixes #695)
	if err := validation.ValidateStringLength(req.Name, "name", 1, surveyNameMaxLength); err != nil {
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

	// Validate description (fixes #695)
	if len(req.Description) > surveyDescriptionMaxLength {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.descriptionTooLong"),
			"",
		) // fixes #694
		return
	}

	// Validate survey type (fixes #695)
	validTypes := []survey.Type{survey.TypePassive, survey.TypeActive, survey.TypeThroughput}
	validType := slices.Contains(validTypes, survey.Type(req.SurveyType))
	if !validType {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.survey.invalidType"),
			"",
		) // fixes #694
		return
	}

	// Validate interface name if provided (fixes #695)
	if req.Interface != "" {
		if err := validation.ValidateInterface(req.Interface); err != nil {
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
	} else {
		if s.netManager() != nil {
			req.Interface = s.netManager().GetCurrentInterface()
		}
	}

	newSurvey, err := s.surveyManager().CreateSurvey(
		req.Name,
		req.Description,
		req.Interface,
		survey.Type(req.SurveyType),
	)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to create survey", "error", err)
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

	logger.InfoContext(r.Context(), "Survey created",
		"survey_id", newSurvey.ID,
		"name", newSurvey.Name,
		"type", newSurvey.SurveyType,
		"interface", newSurvey.Interface)

	sendJSONResponse(w, logger, http.StatusOK, newSurvey)
}

func (s *Server) listSurveys(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveys := s.surveyManager().ListSurveys()
	sendJSONResponse(w, logger, http.StatusOK, surveys)
}

func (s *Server) getSurvey(w http.ResponseWriter, r *http.Request) {
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
		) // fixes #694
		return
	}

	surveyData, err := s.surveyManager().GetSurvey(id)
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

	sendJSONResponse(w, logger, http.StatusOK, surveyData)
}

func (s *Server) deleteSurvey(w http.ResponseWriter, r *http.Request) {
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
		) // fixes #694
		return
	}

	if err := s.surveyManager().DeleteSurvey(id); err != nil {
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

	logger.InfoContext(r.Context(), "Survey deleted", "survey_id", id)

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "deleted"})
}

// surveyStateAction is a function type for survey state transitions.
type surveyStateAction func(id string) error

// handleSurveyStateChange is a helper that handles survey state transitions (start/pause/complete).
// It extracts common logic to avoid code duplication.
func (s *Server) handleSurveyStateChange(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	actionName string,
	action surveyStateAction,
) {
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
		) // fixes #694
		return
	}

	if err := action(id); err != nil {
		logger.ErrorContext(r.Context(), "Failed to start survey", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.survey.startFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	// Return the updated survey so frontend can update its state
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
		) // fixes #694, #H7
		return
	}

	logger.InfoContext(r.Context(), "Survey state changed",
		"survey_id", id,
		"action", actionName,
		"status", updatedSurvey.Status)

	sendJSONResponse(w, logger, http.StatusOK, updatedSurvey)
}

func (s *Server) startSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.handleSurveyStateChange(w, r, logger, "start", s.surveyManager().StartSurvey)
}

func (s *Server) pauseSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.handleSurveyStateChange(w, r, logger, "pause", s.surveyManager().PauseSurvey)
}

func (s *Server) completeSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.handleSurveyStateChange(w, r, logger, "complete", s.surveyManager().CompleteSurvey)
}
