// Package api provides the HTTP/WebSocket server.
// handlers_speedtest.go contains speed test handlers.
// Split from handlers_health_checks.go for code organization (Plan F).
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Speedtest Types
// ============================================================================

// SpeedtestResponse represents the speedtest results for the API.
type SpeedtestResponse struct {
	Download     float64 `json:"download"` // Mbps
	Upload       float64 `json:"upload"`   // Mbps
	Latency      float64 `json:"latency"`  // ms
	Server       string  `json:"server"`   // Server name
	Location     string  `json:"location"` // Server location
	Host         string  `json:"host"`     // Server host
	Distance     float64 `json:"distance"` // km
	Timestamp    string  `json:"timestamp"`
	TestDuration float64 `json:"testDuration"` // seconds
}

// SpeedtestStatusResponse represents the current speedtest status.
type SpeedtestStatusResponse struct {
	Running  bool               `json:"running"`
	Phase    string             `json:"phase"`
	Progress float64            `json:"progress"`
	Last     *SpeedtestResponse `json:"last,omitempty"`
}

// ============================================================================
// Speedtest Handlers
// ============================================================================

// handleSpeedtest starts a speedtest in the background and returns immediately.
// Use /api/speedtest/status to poll for results.
func (s *Server) handleSpeedtest(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.health.speedtestPostRequired"), "")
		return
	}

	if s.speedtestTester == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, localizer.T("errors.health.speedtestNotAvailable"), "")
		return
	}

	// Check if already running
	status := s.speedtestTester.GetStatus()
	if status.Running {
		sendErrorResponseWithDetails(w, logger, http.StatusConflict, ErrCodeConflict, localizer.T("errors.health.speedtestInProgress"), "")
		return
	}

	// Run the test in the background (takes 30-60 seconds)
	go func(logger *slog.Logger) {
		// Add timeout protection for speedtest operations
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		_, err := s.speedtestTester.RunTest(ctx)
		if err != nil {
			logger.Error("Speedtest failed", "error", err)
		}
	}(logger)

	// Return immediately with "started" status
	sendJSONResponse(w, logger, http.StatusOK, map[string]interface{}{
		"status":  "started",
		"message": "Speedtest started. Poll /api/speedtest/status for results.",
	})
}

// handleSpeedtestStatus returns the current speedtest status.
func (s *Server) handleSpeedtestStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	if s.speedtestTester == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, localizer.T("errors.health.speedtestNotAvailable"), "")
		return
	}

	status := s.speedtestTester.GetStatus()
	resp := SpeedtestStatusResponse{
		Running:  status.Running,
		Phase:    status.Phase,
		Progress: status.Progress,
	}

	// Include last result if available
	if lastResult := s.speedtestTester.GetLastResult(); lastResult != nil {
		resp.Last = &SpeedtestResponse{
			Download:     lastResult.Download,
			Upload:       lastResult.Upload,
			Latency:      lastResult.Latency,
			Server:       lastResult.Server,
			Location:     lastResult.Location,
			Host:         lastResult.Host,
			Distance:     lastResult.Distance,
			Timestamp:    lastResult.Timestamp.Format(time.RFC3339),
			TestDuration: lastResult.TestDuration,
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}
