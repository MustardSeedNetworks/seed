package api

// handlers_survey_analysis.go contains the dead-zone analysis and heatmap
// generation handlers, plus the query-string parsers they share.

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// getSurveyDeadZones handles GET /api/survey/dead-zones?id=xxx&threshold=-75.
// Analyzes the survey and detects areas with poor WiFi coverage.
func (s *Server) getSurveyDeadZones(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

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
	threshold, thresholdErr := parseThresholdParam(r.URL.Query().Get("threshold"))
	if thresholdErr != nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T(thresholdErr.Error()),
			"",
		) // fixes #694
		return
	}

	analysis, err := s.surveyManager().DetectDeadZones(id, threshold)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to detect dead zones",
			"survey_id", id,
			"threshold", threshold,
			"error", err)
		logger.ErrorContext(ctx, "Failed to calculate dead zones", "error", err)
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

// parseThresholdParam parses and validates the threshold parameter.
// Returns the threshold value and an error with the i18n key if invalid.
func parseThresholdParam(thresholdStr string) (int, error) {
	if thresholdStr == "" {
		return survey.DefaultThreshold, nil
	}

	var t int
	if _, err := fmt.Sscanf(thresholdStr, "%d", &t); err != nil {
		return 0, errors.New("errors.survey.invalidThresholdValue")
	}

	// Validate threshold range (-100 to -30 dBm is reasonable)
	if t < -100 || t > -30 {
		return 0, errors.New("errors.survey.invalidThreshold")
	}

	return t, nil
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

	result, err := s.surveyManager().GenerateHeatmap(id, config)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate heatmap",
			"survey_id", id,
			"type", config.Type,
			"error", err)
		logger.ErrorContext(r.Context(), "Failed to generate heatmap", "error", err)
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
		// #nosec G705 -- response body is a server-generated PNG; Content-Type already set to image/png above.
		if _, writeErr := w.Write(result.Image); writeErr != nil {
			logger.ErrorContext(r.Context(), "Failed to write heatmap image", "error", writeErr)
		}
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}
