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
	sendJSONResponse(w, logger, http.StatusOK, version.Info())
}
