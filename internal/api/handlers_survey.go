// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
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
		logger.Warn("Invalid request body", "error", err)
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
	if err := validation.ValidateStringLength(req.Name, "name", 1, 100); err != nil {
		logger.Warn("Survey validation failed", "error", err)
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
	if len(req.Description) > 500 {
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
			logger.Warn("Survey validation failed", "error", err)
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
		if s.netManager != nil {
			req.Interface = s.netManager.GetCurrentInterface()
		}
	}

	newSurvey, err := s.surveyManager.CreateSurvey(
		req.Name,
		req.Description,
		req.Interface,
		survey.Type(req.SurveyType),
	)
	if err != nil {
		logger.Error("Failed to create survey", "error", err)
		logger.Error("Internal error", "error", err)
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

	logger.Info("Survey created",
		"survey_id", newSurvey.ID,
		"name", newSurvey.Name,
		"type", newSurvey.SurveyType,
		"interface", newSurvey.Interface)

	sendJSONResponse(w, logger, http.StatusOK, newSurvey)
}

func (s *Server) listSurveys(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveys := s.surveyManager.ListSurveys()
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

	surveyData, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		logger.Warn("Survey not found", "error", err)
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

	if err := s.surveyManager.DeleteSurvey(id); err != nil {
		logger.Warn("Survey not found", "error", err)
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

	logger.Info("Survey deleted", "survey_id", id)

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
		logger.Error("Failed to start survey", "error", err)
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
	updatedSurvey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		logger.Error("Failed to get survey", "error", err)
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

	logger.Info("Survey state changed",
		"survey_id", id,
		"action", actionName,
		"status", updatedSurvey.Status)

	sendJSONResponse(w, logger, http.StatusOK, updatedSurvey)
}

func (s *Server) startSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.handleSurveyStateChange(w, r, logger, "start", s.surveyManager.StartSurvey)
}

func (s *Server) pauseSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.handleSurveyStateChange(w, r, logger, "pause", s.surveyManager.PauseSurvey)
}

func (s *Server) completeSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	s.handleSurveyStateChange(w, r, logger, "complete", s.surveyManager.CompleteSurvey)
}

// AddSampleRequest contains a WiFi signal sample measurement for a survey location.
type AddSampleRequest struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	SampleData any `json:"sampleData"`
}

func (s *Server) addSurveySample(w http.ResponseWriter, r *http.Request) {
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

	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req AddSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
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

	// Process the sample data to calculate aggregations for PassiveSample
	sampleData := req.SampleData
	if dataMap, ok := req.SampleData.(map[string]any); ok {
		// Check if this is a PassiveSample by looking for the "networks" field
		if _, hasNetworks := dataMap["networks"]; hasNetworks {
			// Re-marshal and unmarshal to get the typed PassiveSample
			dataBytes, marshalErr := json.Marshal(dataMap)
			if marshalErr != nil {
				logger.Warn("Failed to marshal sample data for type detection",
					"survey_id", id,
					"error", marshalErr)
			} else {
				var passiveSample survey.PassiveSample
				if unmarshalErr := json.Unmarshal(dataBytes, &passiveSample); unmarshalErr != nil {
					logger.Debug("Sample data is not PassiveSample type, using raw data",
						"survey_id", id,
						"error", unmarshalErr)
				} else {
					// Calculate aggregations
					passiveSample.CalculateAggregations()
					sampleData = passiveSample
				}
			}
		} else if _, hasBSSID := dataMap["bssid"]; hasBSSID {
			// Check if this is an ActiveSample
			activeBytes, activeMarshalErr := json.Marshal(dataMap)
			if activeMarshalErr != nil {
				logger.Warn("Failed to marshal active sample data",
					"survey_id", id,
					"error", activeMarshalErr)
			} else {
				var activeSample survey.ActiveSample
				if activeUnmarshalErr := json.Unmarshal(activeBytes, &activeSample); activeUnmarshalErr != nil {
					logger.Debug("Sample data is not ActiveSample type, using raw data",
						"survey_id", id,
						"error", activeUnmarshalErr)
				} else {
					sampleData = activeSample
				}
			}
		}
	}

	if err := s.surveyManager.AddSample(id, req.X, req.Y, sampleData); err != nil {
		logger.Error("Failed to add sample to survey", "error", err, "survey_id", id)
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

	logger.Info("Survey sample added",
		"survey_id", id,
		"x", req.X,
		"y", req.Y)

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "sample added"})
}

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
	if err := validation.ValidateIntRange(req.Width, "width", 1, 10000); err != nil {
		return err
	}
	if err := validation.ValidateIntRange(req.Height, "height", 1, 10000); err != nil {
		return err
	}
	return validation.ValidateFloatRange(req.ScaleM, "scaleM", 0.01, 1000.0)
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
	if !s.endpointRateLimiter.Allow(s.getClientIP(r)) {
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
		logger.Warn("Invalid request body", "error", err)
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
		logger.Warn("Survey validation failed", "error", err)
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

	if err := s.surveyManager.UpdateFloorPlan(id, floorPlan); err != nil {
		logger.Warn("Survey not found", "error", err)
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

	updatedSurvey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		logger.Error("Failed to get survey", "error", err)
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
		logger.Warn("Invalid request body", "error", err)
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
		logger.Warn("Survey validation failed: invalid iperf server")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}
	if req.TestDuration != 0 &&
		validation.ValidateIntRange(req.TestDuration, "testDuration", 1, 300) != nil {
		logger.Warn("Survey validation failed: invalid test duration")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}

	if err := s.surveyManager.UpdateSurveySettings(id, surveyType, req.IperfServer, req.TestDuration); err != nil {
		logger.Error("Failed to update survey", "error", err)
		ctx.sendBadRequestError("errors.survey.updateFailed")
		return
	}

	settingsUpdatedSurvey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		logger.Error("Failed to get survey", "error", err)
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

	if !s.endpointRateLimiter.Allow(s.getClientIP(r)) {
		ctx.sendRateLimitError()
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeAirMapper)
	if err := r.ParseMultipartForm(MaxBodySizeAirMapper); err != nil {
		logger.Warn("Survey file too large", "error", err)
		ctx.sendBadRequestError("errors.survey.fileTooLarge")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.Warn("No file provided for survey", "error", err)
		ctx.sendBadRequestError("errors.survey.noFileProvided")
		return
	}
	defer func() { _ = file.Close() }()

	if validation.ValidateFilename(handler.Filename, "filename") != nil {
		logger.Warn("Survey validation failed: invalid filename")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}

	if !strings.HasSuffix(strings.ToLower(handler.Filename), ".amp") {
		ctx.sendValidationError("errors.survey.invalidFileType")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		logger.Error("Failed to read survey file", "error", err)
		ctx.sendInternalError("errors.survey.readFileFailed")
		return
	}

	ampFile, err := survey.ParseAirMapperFile(data)
	if err != nil {
		logger.Warn("Failed to parse survey file", "error", err)
		ctx.sendBadRequestError("errors.survey.parseFileFailed")
		return
	}

	result, err := ampFile.ToImportResult()
	if err != nil {
		logger.Error("Failed to process survey file", "error", err)
		ctx.sendInternalError("errors.survey.processFileFailed")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, result)
}

// getSurveyDeadZones handles GET /api/survey/dead-zones?id=xxx&threshold=-75.
// Analyzes the survey and detects areas with poor WiFi coverage.
func (s *Server) getSurveyDeadZones(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
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

	// Parse optional threshold parameter (default: -75 dBm)
	threshold := survey.DefaultThreshold
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		var t int
		if _, err := fmt.Sscanf(thresholdStr, "%d", &t); err == nil {
			// Validate threshold range (-100 to -30 dBm is reasonable)
			if t >= -100 && t <= -30 {
				threshold = t
			} else {
				sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, localizer.T("errors.survey.invalidThreshold"), "") // fixes #694
				return
			}
		} else {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, localizer.T("errors.survey.invalidThresholdValue"), "") // fixes #694
			return
		}
	}

	analysis, err := s.surveyManager.DetectDeadZones(id, threshold)
	if err != nil {
		logger.Error("Failed to detect dead zones",
			"survey_id", id,
			"threshold", threshold,
			"error", err)
		logger.Error("Failed to calculate dead zones", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.survey.deadZonesFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, analysis)
}

// parseHeatmapConfig parses heatmap configuration from query parameters.
func parseHeatmapConfig(r *http.Request) survey.HeatmapConfig {
	config := survey.DefaultHeatmapConfig()

	// Parse heatmap type
	heatmapType := r.URL.Query().Get("type")
	if heatmapType == "" {
		heatmapType = "rssi"
	}
	config.Type = survey.ParseHeatmapType(heatmapType)

	// Optional: cell size (1-50)
	if cellSize := r.URL.Query().Get("cell_size"); cellSize != "" {
		var size int
		if _, err := fmt.Sscanf(cellSize, "%d", &size); err == nil && size > 0 && size <= 50 {
			config.CellSize = size
		}
	}

	// Optional: opacity (0-255)
	if opacity := r.URL.Query().Get("opacity"); opacity != "" {
		var op int
		if _, err := fmt.Sscanf(opacity, "%d", &op); err == nil && op >= 0 && op <= 255 {
			config.Opacity = uint8(op) //#nosec G115 -- bounds checked above
		}
	}

	// Optional: show samples
	if r.URL.Query().Get("show_samples") == "false" {
		config.ShowSamples = false
	}

	return config
}

// getSurveyHeatmap handles GET /api/survey/heatmap?id=xxx&type=rssi.
// Generates a heatmap visualization from survey sample data.
func (s *Server) getSurveyHeatmap(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
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

	config := parseHeatmapConfig(r)

	result, err := s.surveyManager.GenerateHeatmap(id, config)
	if err != nil {
		logger.Error("Failed to generate heatmap",
			"survey_id", id,
			"type", config.Type,
			"error", err)
		logger.Error("Failed to generate heatmap", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.survey.heatmapFailed"),
			"",
		) // fixes #694, #H7
		return
	}

	// Check if client wants raw PNG or JSON with base64
	if r.URL.Query().Get("format") == "png" {
		w.Header().Set("Content-Type", "image/png")
		w.Header().
			Set("Content-Disposition", fmt.Sprintf("inline; filename=\"heatmap-%s-%s.png\"", id[:8], config.Type))
		w.WriteHeader(http.StatusOK)
		if _, writeErr := w.Write(result.Image); writeErr != nil {
			logger.Error("Failed to write heatmap image", "error", writeErr)
		}
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// ============================================================================
// Multi-floor Management Handlers (#654)
// ============================================================================

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
//

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

	floors, err := s.surveyManager.GetFloors(surveyID)
	if err != nil {
		logger.Warn("Survey not found", "error", err)
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
		logger.Warn("Invalid request body", "error", err)
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
	if err := validation.ValidateStringLength(req.Name, "name", 1, 50); err != nil {
		logger.Warn("Survey validation failed", "error", err)
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
	if err := validation.ValidateIntRange(req.Level, "level", -10, 200); err != nil {
		logger.Warn("Survey validation failed", "error", err)
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

	floor, err := s.surveyManager.AddFloor(surveyID, req.Name, req.Level)
	if err != nil {
		logger.Error("Failed to add floor", "survey_id", surveyID, "error", err)
		logger.Error("Internal error", "error", err)
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

	floor, err := s.surveyManager.GetFloor(surveyID, floorID)
	if err != nil {
		logger.Warn("Survey not found", "error", err)
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
		logger.Warn("Invalid request body", "error", err)
		ctx.sendBadRequestError("errors.api.invalidRequestBody")
		return
	}

	// Validate floor name and level
	if validation.ValidateStringLength(req.Name, "name", 1, 50) != nil {
		logger.Warn("Survey validation failed: invalid floor name")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}
	if validation.ValidateIntRange(req.Level, "level", -10, 200) != nil {
		logger.Warn("Survey validation failed: invalid floor level")
		ctx.sendValidationError("errors.survey.validationFailed")
		return
	}

	if err := s.surveyManager.UpdateFloor(surveyID, floorID, req.Name, req.Level); err != nil {
		logger.Error(
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

	floor, err := s.surveyManager.GetFloor(surveyID, floorID)
	if err != nil {
		logger.Error("Failed to get floor", "error", err)
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

	if err := s.surveyManager.DeleteFloor(surveyID, floorID); err != nil {
		logger.Error(
			"Failed to delete floor",
			"survey_id",
			surveyID,
			"floor_id",
			floorID,
			"error",
			err,
		)
		logger.Error("Failed to delete survey", "error", err)
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

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "deleted"})
}

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
		logger.Warn("Invalid request body", "error", err)
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

	if err := s.surveyManager.SetActiveFloor(surveyID, req.FloorID); err != nil {
		logger.Error(
			"Failed to set active floor",
			"survey_id",
			surveyID,
			"floor_id",
			req.FloorID,
			"error",
			err,
		)
		logger.Error("Failed to update survey", "error", err)
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
	updatedSurvey, err := s.surveyManager.GetSurvey(surveyID)
	if err != nil {
		logger.Error("Failed to get survey", "error", err)
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

	if !s.endpointRateLimiter.Allow(s.getClientIP(r)) {
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
		logger.Warn("Invalid request body", "error", err)
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
		logger.Warn("Survey validation failed", "error", err)
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
	if err := s.surveyManager.UpdateFloorPlanByFloorID(surveyID, floorID, floorPlan); err != nil {
		logger.Warn("Survey not found", "error", err)
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

	floor, err := s.surveyManager.GetFloor(surveyID, floorID)
	if err != nil {
		logger.Error("Failed to get survey", "error", err)
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
		logger.Warn("Invalid request body", "error", err)
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

	// Process the sample data (same as addSurveySample)
	sampleData := req.SampleData
	if dataMap, ok := req.SampleData.(map[string]any); ok {
		if _, hasNetworks := dataMap["networks"]; hasNetworks {
			dataBytes, marshalErr := json.Marshal(dataMap)
			if marshalErr == nil {
				var passiveSample survey.PassiveSample
				if unmarshalErr := json.Unmarshal(dataBytes, &passiveSample); unmarshalErr == nil {
					passiveSample.CalculateAggregations()
					sampleData = passiveSample
				}
			}
		} else if _, hasBSSID := dataMap["bssid"]; hasBSSID {
			activeBytes, activeMarshalErr := json.Marshal(dataMap)
			if activeMarshalErr == nil {
				var activeSample survey.ActiveSample
				if activeUnmarshalErr := json.Unmarshal(activeBytes, &activeSample); activeUnmarshalErr == nil {
					sampleData = activeSample
				}
			}
		}
	}

	if err := s.surveyManager.AddSampleToFloor(surveyID, floorID, req.X, req.Y, sampleData); err != nil {
		logger.Error(
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

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "sample added"})
}

// ============================================================================
// Report Generation Handlers (#653)
// ============================================================================

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
			logger.Warn("Invalid request body for report generation", "error", err)
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
	if len(req.CompanyName) > 100 {
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

	pdfBytes, err := s.surveyManager.GenerateReport(surveyID, options)
	if err != nil {
		logger.Error("Failed to generate report",
			"survey_id", surveyID,
			"error", err)
		logger.Error("Failed to export survey", "error", err)
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
	if _, writeErr := w.Write(pdfBytes); writeErr != nil {
		logger.Error("Failed to write PDF response", "error", writeErr)
	}
}
