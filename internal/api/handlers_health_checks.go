package api

// handlers_health_checks.go holds the small shared helpers the health-check
// surface still needs after the legacy /run stack was deleted (ADR-0027 P3):
// the success/warning/error status vocabulary used across the API package and
// the latency→status threshold helper used by the probe-backed /run mapping
// (healthcheckrun.go). The on-demand run handler itself lives in
// healthcheckrun.go.

// Shared test/result status vocabulary, used across the API handlers.
const (
	statusError   = "error"
	statusWarning = "warning"
	statusSuccess = "success"
)

// Transport protocol labels used by port-style health checks.
const (
	protoTCP = "tcp"
	protoUDP = "udp"
)

// getTestStatus classifies a latency (ms) against warning/critical bounds
// (ms): below warning is success, below critical is warning, otherwise error.
func getTestStatus(latencyMs float64, warningMs, criticalMs int64) string {
	if latencyMs < float64(warningMs) {
		return statusSuccess
	}
	if latencyMs < float64(criticalMs) {
		return statusWarning
	}
	return statusError
}
