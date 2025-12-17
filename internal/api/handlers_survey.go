// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/krisarmstrong/seed/internal/survey"
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

// isValidSurveyID validates a survey ID to prevent injection attacks.
// Valid survey IDs must be:
// - Non-empty
// - At most 64 characters long
// - Contain only alphanumeric characters, dashes, and underscores.
func isValidSurveyID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func (s *Server) createSurvey(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req CreateSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Survey name is required", http.StatusBadRequest)
		return
	}

	if req.Interface == "" {
		if s.netManager != nil {
			req.Interface = s.netManager.GetCurrentInterface()
		}
	}

	newSurvey, err := s.surveyManager.CreateSurvey(req.Name, req.Description, req.Interface, survey.Type(req.SurveyType))
	if err != nil {
		slog.Error("Failed to create survey", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, newSurvey)
}

func (s *Server) listSurveys(w http.ResponseWriter, _ *http.Request) {
	surveys := s.surveyManager.ListSurveys()
	sendJSONResponse(w, http.StatusOK, surveys)
}

func (s *Server) getSurvey(w http.ResponseWriter, r *http.Request) {
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

	sendJSONResponse(w, http.StatusOK, surveyData)
}

func (s *Server) deleteSurvey(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	if err := s.surveyManager.DeleteSurvey(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// surveyStateAction is a function type for survey state transitions.
type surveyStateAction func(id string) error

// handleSurveyStateChange is a helper that handles survey state transitions (start/pause/complete).
// It extracts common logic to avoid code duplication.
func (s *Server) handleSurveyStateChange(w http.ResponseWriter, r *http.Request, action surveyStateAction) {
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
	sendJSONResponse(w, http.StatusOK, updatedSurvey)
}

func (s *Server) startSurvey(w http.ResponseWriter, r *http.Request) {
	s.handleSurveyStateChange(w, r, s.surveyManager.StartSurvey)
}

func (s *Server) pauseSurvey(w http.ResponseWriter, r *http.Request) {
	s.handleSurveyStateChange(w, r, s.surveyManager.PauseSurvey)
}

func (s *Server) completeSurvey(w http.ResponseWriter, r *http.Request) {
	s.handleSurveyStateChange(w, r, s.surveyManager.CompleteSurvey)
}

// AddSampleRequest contains a WiFi signal sample measurement for a survey location.
type AddSampleRequest struct {
	X          int         `json:"x"`
	Y          int         `json:"y"`
	SampleData interface{} `json:"sampleData"`
}

func (s *Server) addSurveySample(w http.ResponseWriter, r *http.Request) {
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
				slog.Warn("Failed to marshal sample data for type detection",
					"survey_id", id,
					"error", err)
			} else {
				var passiveSample survey.PassiveSample
				if err := json.Unmarshal(dataBytes, &passiveSample); err != nil {
					slog.Debug("Sample data is not PassiveSample type, using raw data",
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
				slog.Warn("Failed to marshal active sample data",
					"survey_id", id,
					"error", err)
			} else {
				var activeSample survey.ActiveSample
				if err := json.Unmarshal(dataBytes, &activeSample); err != nil {
					slog.Debug("Sample data is not ActiveSample type, using raw data",
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

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "sample added"})
}

// UpdateFloorPlanRequest contains floor plan image and dimension parameters.
type UpdateFloorPlanRequest struct {
	ImageData string  `json:"imageData"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	ScaleM    float64 `json:"scaleM"`
}

func (s *Server) updateSurveyFloorPlan(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if !isValidSurveyID(id) {
		http.Error(w, "Invalid survey ID", http.StatusBadRequest)
		return
	}

	// Limit request body size for floor plan uploads (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeFloorPlan)

	var req UpdateFloorPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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

	sendJSONResponse(w, http.StatusOK, updatedSurvey)
}

// UpdateSurveySettingsRequest is the request body for updating survey settings.
type UpdateSurveySettingsRequest struct {
	SurveyType   string `json:"surveyType"`
	IperfServer  string `json:"iperfServer,omitempty"`
	TestDuration int    `json:"testDuration,omitempty"`
}

// updateSurveySettings handles PUT /api/survey/settings?id=xxx.
func (s *Server) updateSurveySettings(w http.ResponseWriter, r *http.Request) {
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

	// Convert string to survey.Type
	surveyType := survey.Type(req.SurveyType)

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

	sendJSONResponse(w, http.StatusOK, settingsUpdatedSurvey)
}

// importAirMapper handles POST /api/survey/import/airmapper.
// It accepts a multipart form with an .amp file and returns parsed calibration,
// floor plan, and pass/fail criteria data.
func (s *Server) importAirMapper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit file size to 50MB
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

	sendJSONResponse(w, http.StatusOK, result)
}

// getSurveyDeadZones handles GET /api/survey/dead-zones?id=xxx&threshold=-75.
// Analyzes the survey and detects areas with poor WiFi coverage.
func (s *Server) getSurveyDeadZones(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("Failed to detect dead zones",
			"survey_id", id,
			"threshold", threshold,
			"error", err)
		http.Error(w, fmt.Sprintf("Failed to detect dead zones: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, analysis)
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
		slog.Error("Failed to generate heatmap",
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
			slog.Error("Failed to write heatmap image", "error", err)
		}
		return
	}

	sendJSONResponse(w, http.StatusOK, result)
}
