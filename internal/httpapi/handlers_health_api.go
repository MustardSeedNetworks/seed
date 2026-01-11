package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Health check API constants.
const (
	healthQueryLimitDefault  = 100
	healthQueryLimitHistory  = 1000
	queryParamValueTrue      = "true"
	period1h                 = "1h"
	period6h                 = "6h"
	period24h                = "24h"
	period7d                 = "7d"
	period30d                = "30d"
)

// handleHealthCheckResults returns the latest health check results.
func (s *Server) handleHealthCheckResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger := logging.FromContext(r.Context())
	ctx := r.Context()
	repo := s.services.Health.Repository
	if repo == nil {
		http.Error(w, "Health check service not available", http.StatusServiceUnavailable)
		return
	}

	// Get optional query parameters
	endpointName := r.URL.Query().Get("endpoint")
	checkType := r.URL.Query().Get("type")

	var results []*database.HealthCheckResult
	var err error

	if endpointName != "" || checkType != "" {
		// Query with filters
		results, err = repo.Query(ctx, database.HealthCheckQueryOptions{
			CheckType:    checkType,
			EndpointName: endpointName,
			Limit:        healthQueryLimitDefault,
		})
	} else {
		// Get latest for all endpoints
		results, err = repo.GetLatestForAllEndpoints(ctx)
	}

	if err != nil {
		logger.Error("failed to get health check results", "error", err)
		http.Error(w, "Failed to retrieve results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(results); encErr != nil {
		logger.Error("failed to encode health check results", "error", encErr)
	}
}

// handleHealthCheckHistory returns historical health check data.
func (s *Server) handleHealthCheckHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger := logging.FromContext(r.Context())
	ctx := r.Context()
	repo := s.services.Health.Repository
	if repo == nil {
		http.Error(w, "Health check service not available", http.StatusServiceUnavailable)
		return
	}

	endpointName := r.URL.Query().Get("endpoint")
	checkType := r.URL.Query().Get("type")
	period := r.URL.Query().Get("period") // 1h, 6h, 24h, 7d, 30d

	// Parse time range
	end := time.Now()
	start := end.Add(-24 * time.Hour) // Default to 24h

	switch period {
	case period1h:
		start = end.Add(-1 * time.Hour)
	case period6h:
		start = end.Add(-6 * time.Hour)
	case period24h:
		start = end.Add(-24 * time.Hour)
	case period7d:
		start = end.Add(-7 * 24 * time.Hour)
	case period30d:
		start = end.Add(-30 * 24 * time.Hour)
	}

	timeRange := database.TimeRange{Start: start, End: end}

	// Decide whether to use raw data or rollups
	var response any

	switch period {
	case period7d, period30d:
		// Use daily rollups for longer periods
		rollups, err := repo.GetDailyRollups(ctx, checkType, endpointName, timeRange)
		if err != nil {
			logger.Error("failed to get daily rollups", "error", err)
			http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
			return
		}
		response = map[string]any{
			"type":    "daily_rollups",
			"period":  period,
			"start":   start.Format(time.RFC3339),
			"end":     end.Format(time.RFC3339),
			"rollups": rollups,
		}
	case period6h, period24h:
		// Use hourly rollups for medium periods
		rollups, err := repo.GetHourlyRollups(ctx, checkType, endpointName, timeRange)
		if err != nil {
			logger.Error("failed to get hourly rollups", "error", err)
			http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
			return
		}
		response = map[string]any{
			"type":    "hourly_rollups",
			"period":  period,
			"start":   start.Format(time.RFC3339),
			"end":     end.Format(time.RFC3339),
			"rollups": rollups,
		}
	default:
		// Use raw data for short periods
		results, err := repo.Query(ctx, database.HealthCheckQueryOptions{
			CheckType:    checkType,
			EndpointName: endpointName,
			TimeRange:    timeRange,
			Limit:        healthQueryLimitHistory,
		})
		if err != nil {
			logger.Error("failed to get health check history", "error", err)
			http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
			return
		}
		response = map[string]any{
			"type":    "raw",
			"period":  period,
			"start":   start.Format(time.RFC3339),
			"end":     end.Format(time.RFC3339),
			"results": results,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.Error("failed to encode health check history", "error", encErr)
	}
}

// handleHealthCheckScores returns computed health scores for all endpoints.
func (s *Server) handleHealthCheckScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger := logging.FromContext(r.Context())
	ctx := r.Context()
	scorer := s.services.Health.Scorer
	if scorer == nil {
		http.Error(w, "Health scoring service not available", http.StatusServiceUnavailable)
		return
	}

	// Get scores from the health scorer
	scores, err := scorer.CalculateAllScores(ctx)
	if err != nil {
		logger.Error("failed to get health scores", "error", err)
		http.Error(w, "Failed to retrieve scores", http.StatusInternalServerError)
		return
	}

	// Calculate summary stats using typed counters
	var healthy, degraded, critical, unknown int
	for _, score := range scores {
		switch score.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "critical":
			critical++
		default:
			unknown++
		}
	}

	response := map[string]any{
		"scores": scores,
		"summary": map[string]int{
			"totalEndpoints": len(scores),
			"healthy":        healthy,
			"degraded":       degraded,
			"critical":       critical,
			"unknown":        unknown,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.Error("failed to encode health scores", "error", encErr)
	}
}

// handleHealthCheckSLA returns SLA compliance information.
func (s *Server) handleHealthCheckSLA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger := logging.FromContext(r.Context())
	ctx := r.Context()
	slaTracker := s.services.Health.SLATracker
	if slaTracker == nil {
		http.Error(w, "SLA tracking service not available", http.StatusServiceUnavailable)
		return
	}

	endpointName := r.URL.Query().Get("endpoint")
	period := r.URL.Query().Get("period") // daily, weekly, monthly

	if period == "" {
		period = "daily"
	}

	var response any
	var err error

	if endpointName != "" {
		// Get SLA report for specific endpoint
		response, err = slaTracker.GenerateCurrentPeriodReport(ctx, endpointName)
	} else {
		// Get SLA summary for all endpoints
		response, err = slaTracker.GenerateSummary(ctx, period)
	}

	if err != nil {
		logger.Error("failed to get SLA data", "error", err)
		http.Error(w, "Failed to retrieve SLA data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.Error("failed to encode SLA data", "error", encErr)
	}
}

// handleHealthCheckAlerts returns health check alerts.
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
	alertMgr := s.services.Health.AlertManager
	if alertMgr == nil {
		http.Error(w, "Alert service not available", http.StatusServiceUnavailable)
		return
	}

	// Get active alerts
	rawAlerts := alertMgr.GetActiveAlerts()
	stats := alertMgr.GetAlertStats()

	response := map[string]any{
		"alerts":    rawAlerts,
		"stats":     stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.Error("failed to encode alerts", "error", encErr)
	}
}

func (s *Server) acknowledgeHealthCheckAlert(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	alertMgr := s.services.Health.AlertManager
	if alertMgr == nil {
		http.Error(w, "Alert service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		AlertID        string `json:"alertId"`
		AcknowledgedBy string `json:"acknowledgedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AlertID == "" {
		http.Error(w, "Alert ID is required", http.StatusBadRequest)
		return
	}

	if !alertMgr.AcknowledgeAlert(req.AlertID, req.AcknowledgedBy) {
		http.Error(w, "Alert not found or already acknowledged", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged"}); encErr != nil {
		logger.Error("failed to encode response", "error", encErr)
	}
}

// handleHealthCheckAnomalies returns detected anomalies.
func (s *Server) handleHealthCheckAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger := logging.FromContext(r.Context())
	detector := s.services.Health.AnomalyDetector
	if detector == nil {
		http.Error(w, "Anomaly detection service not available", http.StatusServiceUnavailable)
		return
	}

	endpointName := r.URL.Query().Get("endpoint")
	includeStats := r.URL.Query().Get("includeStats") == queryParamValueTrue

	response := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Get active anomalies
	anomalies := detector.GetActiveAnomalies()
	if endpointName != "" {
		var filtered []*struct {
			ID           string    `json:"id"`
			EndpointName string    `json:"endpointName"`
			Type         string    `json:"type"`
			Severity     string    `json:"severity"`
			Message      string    `json:"message"`
			Value        float64   `json:"value"`
			Expected     float64   `json:"expected"`
			Deviation    float64   `json:"deviation"`
			DetectedAt   time.Time `json:"detectedAt"`
		}
		for _, a := range anomalies {
			if a.EndpointName == endpointName {
				filtered = append(filtered, &struct {
					ID           string    `json:"id"`
					EndpointName string    `json:"endpointName"`
					Type         string    `json:"type"`
					Severity     string    `json:"severity"`
					Message      string    `json:"message"`
					Value        float64   `json:"value"`
					Expected     float64   `json:"expected"`
					Deviation    float64   `json:"deviation"`
					DetectedAt   time.Time `json:"detectedAt"`
				}{
					ID:           a.ID,
					EndpointName: a.EndpointName,
					Type:         a.Type,
					Severity:     a.Severity,
					Message:      a.Message,
					Value:        a.Value,
					Expected:     a.Expected,
					Deviation:    a.Deviation,
					DetectedAt:   a.DetectedAt,
				})
			}
		}
		response["anomalies"] = filtered
	} else {
		response["anomalies"] = anomalies
	}

	response["activeCount"] = len(anomalies)

	// Include statistics if requested
	if includeStats {
		allStats := detector.GetAllStats()
		var statsOutput []map[string]any
		for name, stats := range allStats {
			if endpointName == "" || name == endpointName {
				statsOutput = append(statsOutput, map[string]any{
					"endpointName": stats.EndpointName,
					"mean":         stats.Mean,
					"stdDev":       stats.StdDev,
					"min":          stats.Min,
					"max":          stats.Max,
					"sampleCount":  stats.SampleCount,
					"lastValue":    stats.LastValue,
					"lastUpdate":   stats.LastUpdate.Format(time.RFC3339),
				})
			}
		}
		response["statistics"] = statsOutput
	}

	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		logger.Error("failed to encode anomalies", "error", encErr)
	}
}
