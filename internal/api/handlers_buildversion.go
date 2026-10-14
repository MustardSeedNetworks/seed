package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/version"
)

// BuildVersionResponse is GET /__version: the build metadata plus
// tlsFingerprint, the SHA-256 fingerprint of the active TLS certificate. The
// field is always present so the response shape is stable; it is empty if the
// active certificate is missing or unreadable.
type BuildVersionResponse struct {
	version.BuildInfo

	TLSFingerprint string `json:"tlsFingerprint"`
}

// handleBuildVersion serves GET /__version with build metadata for deployment
// validation. Unauthenticated by design — operators need to verify which
// binary is running without holding a session. Required by the Universal
// Build Contract; see the repository development policy.
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
	sendJSONResponse(w, logger, http.StatusOK, BuildVersionResponse{
		BuildInfo:      version.Info(),
		TLSFingerprint: s.tlsFingerprintForResponse(),
	})
}
