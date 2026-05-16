package api

// handlers_survey_samples.go contains the sample-ingest handler plus the
// helpers that detect whether a raw sample is a PassiveSample (has
// "networks") or an ActiveSample (has "bssid") and convert it accordingly.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// AddSampleRequest contains a WiFi signal sample measurement for a survey location.
type AddSampleRequest struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	SampleData any `json:"sampleData"`
}

// processSampleData attempts to convert raw sample data to typed survey samples.
// It detects PassiveSample (with "networks" field) or ActiveSample (with "bssid" field)
// and returns the appropriately typed sample, or the original data if conversion fails.
func processSampleData(ctx context.Context, rawData any, surveyID string, logger *slog.Logger) any {
	dataMap, ok := rawData.(map[string]any)
	if !ok {
		return rawData
	}

	if _, hasNetworks := dataMap["networks"]; hasNetworks {
		return processPassiveSample(ctx, dataMap, surveyID, logger)
	}

	if _, hasBSSID := dataMap["bssid"]; hasBSSID {
		return processActiveSample(ctx, dataMap, surveyID, logger)
	}

	return rawData
}

// processPassiveSample converts a data map to a PassiveSample and calculates aggregations.
func processPassiveSample(ctx context.Context, dataMap map[string]any, surveyID string, logger *slog.Logger) any {
	dataBytes, marshalErr := json.Marshal(dataMap)
	if marshalErr != nil {
		logger.WarnContext(ctx, "Failed to marshal sample data for type detection",
			"survey_id", surveyID,
			"error", marshalErr)
		return dataMap
	}

	var passiveSample survey.PassiveSample
	if unmarshalErr := json.Unmarshal(dataBytes, &passiveSample); unmarshalErr != nil {
		logger.DebugContext(ctx, "Sample data is not PassiveSample type, using raw data",
			"survey_id", surveyID,
			"error", unmarshalErr)
		return dataMap
	}

	passiveSample.CalculateAggregations()
	return passiveSample
}

// processActiveSample converts a data map to an ActiveSample.
func processActiveSample(ctx context.Context, dataMap map[string]any, surveyID string, logger *slog.Logger) any {
	activeBytes, activeMarshalErr := json.Marshal(dataMap)
	if activeMarshalErr != nil {
		logger.WarnContext(ctx, "Failed to marshal active sample data",
			"survey_id", surveyID,
			"error", activeMarshalErr)
		return dataMap
	}

	var activeSample survey.ActiveSample
	if activeUnmarshalErr := json.Unmarshal(activeBytes, &activeSample); activeUnmarshalErr != nil {
		logger.DebugContext(ctx, "Sample data is not ActiveSample type, using raw data",
			"survey_id", surveyID,
			"error", activeUnmarshalErr)
		return dataMap
	}

	return activeSample
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

	// Process the sample data to calculate aggregations for PassiveSample
	sampleData := processSampleData(r.Context(), req.SampleData, id, logger)

	if err := s.surveyManager().AddSample(id, req.X, req.Y, sampleData); err != nil {
		logger.ErrorContext(r.Context(), "Failed to add sample to survey", "error", err, "survey_id", id)
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

	logger.InfoContext(r.Context(), "Survey sample added",
		"survey_id", id,
		"x", req.X,
		"y", req.Y)

	sendJSONResponse(w, logger, http.StatusOK, statusResponse{Status: statusSampleAdded})
}
