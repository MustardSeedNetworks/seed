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

// Health-monitoring transport (ADR-0020). The handler decodes the request, calls
// the internal/health/monitoring use-case via s.healthMonitoring, shapes the
// response, and maps monitoring.ErrUnavailable to 503. The legacy
// results/history/scores/sla/alerts read-path over the empty health_check_results
// table was deleted as dead code (ADR-0026); only the probe-backed anomaly read
// (ADR-0021/0025) remains.

// handleHealthCheckAnomalies returns the active anomalies from the unified store
// (ADR-0021), optionally filtered to one endpoint. The active-monitoring probe
// engine is the producer (source=probe, ADR-0025); the anomaly model carries
// evidence, count, and lifecycle.
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
