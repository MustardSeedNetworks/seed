package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Health-monitoring transport (ADR-0020). These handlers decode the request,
// call the internal/health/monitoring use-case via s.healthMonitoring, shape the
// response, and map monitoring.ErrUnavailable to the pre-strangle 503. All
// orchestration (query selection, period/rollup choice, score tally, anomaly
// filtering) lives in the use-case.

// handleHealthCheckResults returns the latest health check results.
func (s *Server) handleHealthCheckResults(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	q := r.URL.Query()

	results, err := s.healthMonitoring.Results(r.Context(), q.Get("endpoint"), q.Get("type"))
	if err != nil {
		if errors.Is(err, monitoring.ErrUnavailable) {
			http.Error(w, "Health check service not available", http.StatusServiceUnavailable)
			return
		}
		logger.ErrorContext(r.Context(), "failed to get health check results", "error", err)
		http.Error(w, "Failed to retrieve results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(results); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode health check results", "error", encErr)
	}
}

// handleHealthCheckHistory returns historical health check data.
func (s *Server) handleHealthCheckHistory(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	q := r.URL.Query()

	h, err := s.healthMonitoring.History(r.Context(), q.Get("endpoint"), q.Get("type"), q.Get("period"))
	if err != nil {
		if errors.Is(err, monitoring.ErrUnavailable) {
			http.Error(w, "Health check service not available", http.StatusServiceUnavailable)
			return
		}
		logger.ErrorContext(r.Context(), "failed to get health check history", "error", err)
		http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"type":   string(h.Kind),
		"period": h.Period,
		"start":  h.Start.Format(time.RFC3339),
		"end":    h.End.Format(time.RFC3339),
	}
	switch h.Kind {
	case monitoring.HistoryDailyRollups:
		response["rollups"] = h.DailyRollups
	case monitoring.HistoryHourlyRollups:
		response["rollups"] = h.HourlyRollups
	case monitoring.HistoryRaw:
		response["results"] = h.Results
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode health check history", "error", encErr)
	}
}

// handleHealthCheckScores returns computed health scores for all endpoints.
func (s *Server) handleHealthCheckScores(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	result, err := s.healthMonitoring.Scores(r.Context())
	if err != nil {
		if errors.Is(err, monitoring.ErrUnavailable) {
			http.Error(w, "Health scoring service not available", http.StatusServiceUnavailable)
			return
		}
		logger.ErrorContext(r.Context(), "failed to get health scores", "error", err)
		http.Error(w, "Failed to retrieve scores", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"scores": result.Scores,
		"summary": map[string]int{
			"totalEndpoints": result.Summary.TotalEndpoints,
			"healthy":        result.Summary.Healthy,
			"degraded":       result.Summary.Degraded,
			"critical":       result.Summary.Critical,
			"unknown":        result.Summary.Unknown,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode health scores", "error", encErr)
	}
}

// handleHealthCheckSLA returns SLA compliance information.
func (s *Server) handleHealthCheckSLA(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	q := r.URL.Query()
	endpoint := q.Get("endpoint")

	var response any
	var err error
	if endpoint != "" {
		response, err = s.healthMonitoring.SLAReport(r.Context(), endpoint)
	} else {
		response, err = s.healthMonitoring.SLASummary(r.Context(), q.Get("period"))
	}
	if err != nil {
		if errors.Is(err, monitoring.ErrUnavailable) {
			http.Error(w, "SLA tracking service not available", http.StatusServiceUnavailable)
			return
		}
		logger.ErrorContext(r.Context(), "failed to get SLA data", "error", err)
		http.Error(w, "Failed to retrieve SLA data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode SLA data", "error", encErr)
	}
}

// handleHealthCheckAlerts returns health check alerts (GET) or acknowledges one (POST).
func (s *Server) handleHealthCheckAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getHealthCheckAlerts(w, r)
	case http.MethodPost:
		s.acknowledgeHealthCheckAlert(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getHealthCheckAlerts(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	result, err := s.healthMonitoring.Alerts()
	if err != nil {
		http.Error(w, "Alert service not available", http.StatusServiceUnavailable)
		return
	}

	response := map[string]any{
		"alerts":    result.Alerts,
		"stats":     result.Stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode alerts", "error", encErr)
	}
}

func (s *Server) acknowledgeHealthCheckAlert(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	var req struct {
		AlertID        string `json:"alertId"`
		AcknowledgedBy string `json:"acknowledgedBy"`
	}

	// Strict decode — unknown fields rejected, body size capped. This endpoint
	// pre-dates the structured envelope and writes plain-text http.Error, so we
	// keep that response shape; the security hardening is the inline
	// DisallowUnknownFields + MaxBytesReader.
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AlertID == "" {
		http.Error(w, "Alert ID is required", http.StatusBadRequest)
		return
	}

	switch err := s.healthMonitoring.AcknowledgeAlert(req.AlertID, req.AcknowledgedBy); {
	case errors.Is(err, monitoring.ErrUnavailable):
		http.Error(w, "Alert service not available", http.StatusServiceUnavailable)
		return
	case errors.Is(err, monitoring.ErrAlertNotFound):
		http.Error(w, "Alert not found or already acknowledged", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "Failed to acknowledge alert", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged"}); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode response", "error", encErr)
	}
}

// handleHealthCheckAnomalies returns the active health-source anomalies from the
// unified store (ADR-0021), optionally filtered to one endpoint. The per-endpoint
// rolling statistics the bespoke detector once exposed are gone with it; the
// unified model carries evidence, count, and lifecycle instead.
func (s *Server) handleHealthCheckAnomalies(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	result, err := s.healthMonitoring.Anomalies(r.Context(), r.URL.Query().Get("endpoint"))
	switch {
	case errors.Is(err, monitoring.ErrUnavailable):
		http.Error(w, "Anomaly detection service not available", http.StatusServiceUnavailable)
		return
	case err != nil:
		http.Error(w, "Failed to read anomalies", http.StatusInternalServerError)
		return
	}

	anomalies := result.Anomalies
	if anomalies == nil {
		anomalies = []anomaly.Anomaly{}
	}
	response := map[string]any{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"anomalies":   anomalies,
		"activeCount": result.ActiveCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode anomalies", "error", encErr)
	}
}
