package httpapi

// handlers_problems.go extends the discovery API with network problem detection endpoints.

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Network Problem Detection API Handlers
// ============================================================================

// NetworkProblemsResponse contains detected network problems.
type NetworkProblemsResponse struct {
	Problems []discovery.NetworkProblem `json:"problems"`
	Summary  *discovery.ProblemSummary  `json:"summary"`
	Total    int                        `json:"total"`
}

// ProblemScanResponse contains problem detection scan results.
type ProblemScanResponse struct {
	Problems     []discovery.NetworkProblem      `json:"problems"`
	IPConflicts  []discovery.IPConflict          `json:"ipConflicts"`
	InterfaceErr []discovery.InterfaceErrorStats `json:"interfaceErrors"`
	WiFiProblems []discovery.WiFiProblem         `json:"wifiProblems,omitempty"`
	ScanTime     string                          `json:"scanTime"`
	DurationMS   int64                           `json:"durationMs"`
}

// handleNetworkProblems returns current network problems.
//
// GET /api/v1/discovery/problems
//
// Returns the list of detected network problems from the most recent scan.
//
// Response: 200 OK with NetworkProblemsResponse.
func (s *Server) handleNetworkProblems(w http.ResponseWriter, r *http.Request) {
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
		)
		return
	}

	detector := s.problemDetector()
	if detector == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Problem detector not available",
			"",
		)
		return
	}

	problems := detector.GetActiveProblems()
	summary := detector.GetSummary()

	sendJSONResponse(w, logger, http.StatusOK, NetworkProblemsResponse{
		Problems: problems,
		Summary:  summary,
		Total:    len(problems),
	})
}

// handleProblemScan triggers a network problem detection scan.
//
// POST /api/v1/discovery/problems/scan
//
// Runs problem detection on discovered devices and returns results.
//
// Response: 200 OK with ProblemScanResponse.
func (s *Server) handleProblemScan(w http.ResponseWriter, r *http.Request) {
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
		)
		return
	}

	detector := s.problemDetector()
	if detector == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Problem detector not available",
			"",
		)
		return
	}

	// Get discovered devices from discovery service
	var devices []*discovery.DiscoveredDevice
	if s.discoveryService() != nil {
		devices = s.discoveryService().GetDevices()
	}

	result, err := detector.Scan(r.Context(), devices)
	if err != nil {
		logger.Error("Problem scan failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Problem scan failed: "+err.Error(),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, ProblemScanResponse{
		Problems:     result.Problems,
		IPConflicts:  result.IPConflicts,
		InterfaceErr: result.InterfaceErrors,
		WiFiProblems: result.WiFiProblems,
		ScanTime:     result.ScanTime.Format("2006-01-02T15:04:05Z07:00"),
		DurationMS:   result.ScanDurationMS,
	})
}

// handleProblemThresholds handles GET/PUT for problem detection thresholds.
//
// GET /api/v1/discovery/problems/thresholds - Get current thresholds
// PUT /api/v1/discovery/problems/thresholds - Update thresholds
func (s *Server) handleProblemThresholds(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	detector := s.problemDetector()
	if detector == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Problem detector not available",
			"",
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		thresholds := detector.GetThresholds()
		sendJSONResponse(w, logger, http.StatusOK, thresholds)

	case http.MethodPut:
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

		var thresholds discovery.ProblemThresholds
		if err := json.NewDecoder(r.Body).Decode(&thresholds); err != nil {
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

		detector.SetThresholds(thresholds)
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Thresholds updated",
		})

	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
	}
}

// problemDetector returns the problem detector from the service container.
func (s *Server) problemDetector() *discovery.ProblemDetector {
	if s.services == nil || s.services.Discovery == nil {
		return nil
	}
	return s.services.Discovery.ProblemDetector
}
