package api

// handlers_problems.go extends the discovery API with network problem detection
// endpoints. The handlers are pure transport (ADR-0020): decode/encode and map
// a problems use-case sentinel to its HTTP status; the detection orchestration
// lives in internal/discovery/problems, with the adapter in internal/app.

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/problems"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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

// writeProblemsUnavailable writes the pre-strangle 503 for an absent detector.
func writeProblemsUnavailable(w http.ResponseWriter, logger *slog.Logger) {
	sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
		ErrCodeServiceUnavail, "Problem detector not available", "")
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

	active, err := s.networkProblems.Active()
	if err != nil {
		writeProblemsUnavailable(w, logger)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, NetworkProblemsResponse{
		Problems: active.Problems,
		Summary:  active.Summary,
		Total:    len(active.Problems),
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

	result, err := s.networkProblems.Scan(r.Context())
	if err != nil {
		if errors.Is(err, problems.ErrUnavailable) {
			writeProblemsUnavailable(w, logger)
			return
		}
		logger.ErrorContext(r.Context(), "Problem scan failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Problem scan failed: "+err.Error(), "")
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
// PUT /api/v1/discovery/problems/thresholds - Update thresholds.
func (s *Server) handleProblemThresholds(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !s.networkProblems.Available() {
		writeProblemsUnavailable(w, logger)
		return
	}

	switch r.Method {
	case http.MethodGet:
		thresholds, err := s.networkProblems.Thresholds()
		if err != nil {
			writeProblemsUnavailable(w, logger)
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, thresholds)

	case http.MethodPut:
		var thresholds discovery.ProblemThresholds
		if !decodeJSONStrictLocalized(w, r, &thresholds, MaxBodySizeJSON, logger, localizer) {
			return
		}
		if err := s.networkProblems.SetThresholds(thresholds); err != nil {
			writeProblemsUnavailable(w, logger)
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"status":  statusSuccess,
			"message": "Thresholds updated",
		})

	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	}
}
