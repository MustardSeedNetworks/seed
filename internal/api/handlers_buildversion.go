package api

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/version"
)

// handleBuildVersion serves GET /__version with build metadata for deployment
// validation. Unauthenticated by design — operators need to verify which
// binary is running without holding a session. Required by the Universal
// Build Contract; see CLAUDE.md.
//
// In addition to version.Info() the response carries `tlsFingerprint`, the
// SHA-256 fingerprint of the active TLS certificate. The field is always
// present so the response shape is stable; it is an empty string when the
// server runs without TLS (HTTP mode or ACME-managed certs).
func (s *Server) handleBuildVersion(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
		return
	}
	resp := version.Info()
	resp["tlsFingerprint"] = s.tlsFingerprintForResponse()
	sendJSONResponse(w, logger, http.StatusOK, resp)
}
