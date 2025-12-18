// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/survey"
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

func (s *Server) createSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req CreateSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate survey name (fixes #695)
	if err := validation.ValidateStringLength(req.Name, "name", 1, 100); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate description (fixes #695)
	if len(req.Description) > 500 {
		http.Error(w, "description too long (max 500 characters)", http.StatusBadRequest)
		return
	}

	// Validate survey type (fixes #695)
	validTypes := []survey.Type{survey.TypePassive, survey.TypeActive, survey.TypeThroughput}
	validType := false
	for _, t := range validTypes {
		if survey.Type(req.SurveyType) == t {
			validType = true
			break
		}
	}
	if !validType {
		http.Error(w, "invalid survey type (must be passive, active, or throughput)", http.StatusBadRequest)
		return
	}

	// Validate interface name if provided (fixes #695)
	if req.Interface != "" {
		if err := validation.ValidateInterface(req.Interface); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		if s.netManager != nil {
			req.Interface = s.netManager.GetCurrentInterface()
		}
	}

	newSurvey, err := s.surveyManager.CreateSurvey(req.Name, req.Description, req.Interface, survey.Type(req.SurveyType))
	if err != nil {
		logger.Error("Failed to create survey", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	surveyData, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, surveyData)
}

func (s *Server) deleteSurvey(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.DeleteSurvey(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	logger.Info("Survey deleted", "survey_id", id)

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "deleted"})
}

// surveyStateAction is a function type for survey state transitions.
type surveyStateAction func(id string) error

// handleSurveyStateChange is a helper that handles survey state transitions (start/pause/complete).
// It extracts common logic to avoid code duplication.
func (s *Server) handleSurveyStateChange(w http.ResponseWriter, r *http.Request, logger *slog.Logger, actionName string, action surveyStateAction) {
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	if err := action(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the updated survey so frontend can update its state
	updatedSurvey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	X          int         `json:"x"`
	Y          int         `json:"y"`
	SampleData interface{} `json:"sampleData"`
}

func (s *Server) addSurveySample(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req AddSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process the sample data to calculate aggregations for PassiveSample
	sampleData := req.SampleData
	if dataMap, ok := req.SampleData.(map[string]interface{}); ok {
		// Check if this is a PassiveSample by looking for the "networks" field
		if _, hasNetworks := dataMap["networks"]; hasNetworks {
			// Re-marshal and unmarshal to get the typed PassiveSample
			dataBytes, err := json.Marshal(dataMap)
			if err != nil {
				logger.Warn("Failed to marshal sample data for type detection",
					"survey_id", id,
					"error", err)
			} else {
				var passiveSample survey.PassiveSample
				if err := json.Unmarshal(dataBytes, &passiveSample); err != nil {
					logger.Debug("Sample data is not PassiveSample type, using raw data",
						"survey_id", id,
						"error", err)
				} else {
					// Calculate aggregations
					passiveSample.CalculateAggregations()
					sampleData = passiveSample
				}
			}
		} else if _, hasBSSID := dataMap["bssid"]; hasBSSID {
			// Check if this is an ActiveSample
			dataBytes, err := json.Marshal(dataMap)
			if err != nil {
				logger.Warn("Failed to marshal active sample data",
					"survey_id", id,
					"error", err)
			} else {
				var activeSample survey.ActiveSample
				if err := json.Unmarshal(dataBytes, &activeSample); err != nil {
					logger.Debug("Sample data is not ActiveSample type, using raw data",
						"survey_id", id,
						"error", err)
				} else {
					sampleData = activeSample
				}
			}
		}
	}

	if err := s.surveyManager.AddSample(id, req.X, req.Y, sampleData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (s *Server) updateSurveyFloorPlan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Rate limit file uploads (fixes #696)
	clientIP := GetClientIP(r)
	if !s.endpointRateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Limit request body size for floor plan uploads (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeFloorPlan)

	var req UpdateFloorPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate image data URL (fixes #695)
	if err := validation.ValidateImageDataURL(req.ImageData, int(MaxBodySizeFloorPlan)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate dimensions (fixes #695)
	if err := validation.ValidateIntRange(req.Width, "width", 1, 10000); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateIntRange(req.Height, "height", 1, 10000); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate scale (fixes #695)
	if err := validation.ValidateFloatRange(req.ScaleM, "scaleM", 0.01, 1000.0); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	floorPlan := &survey.FloorPlan{
		ImageData: req.ImageData,
		Width:     req.Width,
		Height:    req.Height,
		ScaleM:    req.ScaleM,
	}

	if err := s.surveyManager.UpdateFloorPlan(id, floorPlan); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return the updated survey so the frontend can update its state
	updatedSurvey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req UpdateSurveySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate survey type (fixes #695)
	validTypes := []survey.Type{survey.TypePassive, survey.TypeActive, survey.TypeThroughput}
	surveyType := survey.Type(req.SurveyType)
	validType := false
	for _, t := range validTypes {
		if surveyType == t {
			validType = true
			break
		}
	}
	if !validType {
		http.Error(w, "invalid survey type (must be passive, active, or throughput)", http.StatusBadRequest)
		return
	}

	// Validate iPerf server if provided (fixes #695)
	if req.IperfServer != "" {
		if err := validation.ValidateServerAddress(req.IperfServer); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Validate test duration (fixes #695)
	if req.TestDuration != 0 {
		if err := validation.ValidateIntRange(req.TestDuration, "testDuration", 1, 300); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := s.surveyManager.UpdateSurveySettings(id, surveyType, req.IperfServer, req.TestDuration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the updated survey so the frontend can update its state
	settingsUpdatedSurvey, err := s.surveyManager.GetSurvey(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, settingsUpdatedSurvey)
}

// importAirMapper handles POST /api/survey/import/airmapper.
// It accepts a multipart form with an .amp file and returns parsed calibration,
// floor plan, and pass/fail criteria data.
func (s *Server) importAirMapper(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit file uploads (fixes #696)
	clientIP := GetClientIP(r)
	if !s.endpointRateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Limit file size to 50MB (fixes #701 - reviewed and kept at 50MB for large surveys)
	const maxFileSize = 50 << 20 // 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate filename (fixes #695)
	if err := validation.ValidateFilename(handler.Filename, "filename"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(handler.Filename), ".amp") {
		http.Error(w, "Invalid file type. Please upload an .amp file", http.StatusBadRequest)
		return
	}

	// Read the file contents
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Parse the AirMapper file
	ampFile, err := survey.ParseAirMapperFile(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse AirMapper file: %v", err), http.StatusBadRequest)
		return
	}

	// Convert to import result
	result, err := ampFile.ToImportResult()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process AirMapper data: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// getSurveyDeadZones handles GET /api/survey/dead-zones?id=xxx&threshold=-75.
// Analyzes the survey and detects areas with poor WiFi coverage.
func (s *Server) getSurveyDeadZones(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
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
				http.Error(w, "Threshold must be between -100 and -30 dBm", http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "Invalid threshold value", http.StatusBadRequest)
			return
		}
	}

	analysis, err := s.surveyManager.DetectDeadZones(id, threshold)
	if err != nil {
		logger.Error("Failed to detect dead zones",
			"survey_id", id,
			"threshold", threshold,
			"error", err)
		http.Error(w, fmt.Sprintf("Failed to detect dead zones: %v", err), http.StatusInternalServerError)
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	config := parseHeatmapConfig(r)

	result, err := s.surveyManager.GenerateHeatmap(id, config)
	if err != nil {
		logger.Error("Failed to generate heatmap",
			"survey_id", id,
			"type", config.Type,
			"error", err)
		http.Error(w, fmt.Sprintf("Failed to generate heatmap: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if client wants raw PNG or JSON with base64
	if r.URL.Query().Get("format") == "png" {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"heatmap-%s-%s.png\"", id[:8], config.Type))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(result.Image); err != nil {
			logger.Error("Failed to write heatmap image", "error", err)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	surveyID := r.URL.Query().Get("id")
	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	floors, err := s.surveyManager.GetFloors(surveyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floors)
}

// addFloor handles POST /api/survey/floors?id=xxx.
// Adds a new floor to a survey.
func (s *Server) addFloor(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveyID := r.URL.Query().Get("id")
	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req AddFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate floor name
	if err := validation.ValidateStringLength(req.Name, "name", 1, 50); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate floor level
	if err := validation.ValidateIntRange(req.Level, "level", -10, 200); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	floor, err := s.surveyManager.AddFloor(surveyID, req.Name, req.Level)
	if err != nil {
		logger.Error("Failed to add floor", "survey_id", surveyID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// getFloor handles GET /api/survey/floor?id=xxx&floorId=yyy.
// Returns a specific floor.
func (s *Server) getFloor(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}
	if !isValidFloorID(floorID) {
		http.Error(w, "Invalid floor ID", http.StatusBadRequest)
		return
	}

	floor, err := s.surveyManager.GetFloor(surveyID, floorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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
	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}
	if !isValidFloorID(floorID) {
		http.Error(w, "Invalid floor ID", http.StatusBadRequest)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req UpdateFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate floor name
	if err := validation.ValidateStringLength(req.Name, "name", 1, 50); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate floor level
	if err := validation.ValidateIntRange(req.Level, "level", -10, 200); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.UpdateFloor(surveyID, floorID, req.Name, req.Level); err != nil {
		logger.Error("Failed to update floor", "survey_id", surveyID, "floor_id", floorID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated floor
	floor, err := s.surveyManager.GetFloor(surveyID, floorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// deleteFloor handles DELETE /api/survey/floor?id=xxx&floorId=yyy.
// Removes a floor from a survey.
func (s *Server) deleteFloor(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}
	if !isValidFloorID(floorID) {
		http.Error(w, "Invalid floor ID", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.DeleteFloor(surveyID, floorID); err != nil {
		logger.Error("Failed to delete floor", "survey_id", surveyID, "floor_id", floorID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	surveyID := r.URL.Query().Get("id")

	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SetActiveFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidFloorID(req.FloorID) {
		http.Error(w, "Invalid floor ID", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.SetActiveFloor(surveyID, req.FloorID); err != nil {
		logger.Error("Failed to set active floor", "survey_id", surveyID, "floor_id", req.FloorID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the updated survey
	updatedSurvey, err := s.surveyManager.GetSurvey(surveyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, updatedSurvey)
}

// updateFloorFloorPlan handles PUT /api/survey/floor/floorplan?id=xxx&floorId=yyy.
// Updates the floor plan for a specific floor.
func (s *Server) updateFloorFloorPlan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}
	if !isValidFloorID(floorID) {
		http.Error(w, "Invalid floor ID", http.StatusBadRequest)
		return
	}

	// Rate limit file uploads
	clientIP := GetClientIP(r)
	if !s.endpointRateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Limit request body size for floor plan uploads
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeFloorPlan)

	var req UpdateFloorPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate image data URL
	if err := validation.ValidateImageDataURL(req.ImageData, int(MaxBodySizeFloorPlan)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate dimensions
	if err := validation.ValidateIntRange(req.Width, "width", 1, 10000); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validation.ValidateIntRange(req.Height, "height", 1, 10000); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate scale
	if err := validation.ValidateFloatRange(req.ScaleM, "scaleM", 0.01, 1000.0); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	floorPlan := &survey.FloorPlan{
		ImageData: req.ImageData,
		Width:     req.Width,
		Height:    req.Height,
		ScaleM:    req.ScaleM,
	}

	if err := s.surveyManager.UpdateFloorPlanByFloorID(surveyID, floorID, floorPlan); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return updated floor
	floor, err := s.surveyManager.GetFloor(surveyID, floorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, floor)
}

// AddFloorSampleRequest contains parameters for adding a sample to a specific floor.
type AddFloorSampleRequest struct {
	X          int         `json:"x"`
	Y          int         `json:"y"`
	SampleData interface{} `json:"sampleData"`
}

// addFloorSample handles POST /api/survey/floor/sample?id=xxx&floorId=yyy.
// Adds a sample to a specific floor.
func (s *Server) addFloorSample(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	surveyID := r.URL.Query().Get("id")
	floorID := r.URL.Query().Get("floorId")

	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}
	if !isValidFloorID(floorID) {
		http.Error(w, "Invalid floor ID", http.StatusBadRequest)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req AddFloorSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process the sample data (same as addSurveySample)
	sampleData := req.SampleData
	if dataMap, ok := req.SampleData.(map[string]interface{}); ok {
		if _, hasNetworks := dataMap["networks"]; hasNetworks {
			dataBytes, err := json.Marshal(dataMap)
			if err == nil {
				var passiveSample survey.PassiveSample
				if err := json.Unmarshal(dataBytes, &passiveSample); err == nil {
					passiveSample.CalculateAggregations()
					sampleData = passiveSample
				}
			}
		} else if _, hasBSSID := dataMap["bssid"]; hasBSSID {
			dataBytes, err := json.Marshal(dataMap)
			if err == nil {
				var activeSample survey.ActiveSample
				if err := json.Unmarshal(dataBytes, &activeSample); err == nil {
					sampleData = activeSample
				}
			}
		}
	}

	if err := s.surveyManager.AddSampleToFloor(surveyID, floorID, req.X, req.Y, sampleData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	surveyID := r.URL.Query().Get("id")
	if !isValidSurveyID(surveyID) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	// Parse options from request body (all optional)
	var req GenerateReportRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	// Validate company name length
	if len(req.CompanyName) > 100 {
		http.Error(w, "Company name too long (max 100 characters)", http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	// Return PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"survey-report-%s.pdf\"", surveyID[:8]))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(pdfBytes); err != nil {
		logger.Error("Failed to write PDF response", "error", err)
	}
}
