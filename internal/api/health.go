package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Health-monitoring transport (ADR-0020). The handler decodes the request, calls
// the internal/health/monitoring use-case via s.healthMonitoring, shapes the
// response, and maps monitoring.ErrUnavailable to 503. The legacy
// results/history/scores/sla/alerts read-path over the empty health_check_results
// table was deleted as dead code (ADR-0026); only the probe-backed anomaly read
// (ADR-0021/0025) remains.

// ProbeAnomaliesResponse is the active probe anomalies, optionally filtered to
// one endpoint, with the unfiltered active count.
type ProbeAnomaliesResponse struct {
	Timestamp   time.Time         `json:"timestamp"`
	Anomalies   []anomaly.Anomaly `json:"anomalies"`
	ActiveCount int               `json:"activeCount"`
}

// handleHealthCheckAnomalies returns the active anomalies from the unified store
// (ADR-0021), optionally filtered to one endpoint. The active-monitoring probe
// engine is the producer (source=probe, ADR-0025); the anomaly model carries
// evidence, count, and lifecycle.
func (s *Server) handleHealthCheckAnomalies(w http.ResponseWriter, r *http.Request) {
	result, err := s.healthMonitoring.Anomalies(r.Context(), r.URL.Query().Get("endpoint"))
	switch {
	case errors.Is(err, monitoring.ErrUnavailable):
		writeError(
			w,
			r,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Anomaly detection service not available",
		)
		return
	case err != nil:
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to read anomalies")
		return
	}

	anomalies := result.Anomalies
	if anomalies == nil {
		anomalies = []anomaly.Anomaly{}
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, ProbeAnomaliesResponse{
		Timestamp:   time.Now().UTC().Truncate(time.Second),
		Anomalies:   anomalies,
		ActiveCount: result.ActiveCount,
	})
}
