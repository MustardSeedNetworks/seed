package api

// handlers_driver_stats.go serves the NIC driver's own error counters (#416).
//
// A link can be up, negotiated at the right speed, and still dropping frames.
// These counters say which: CRC errors point at cabling, receive drops at a
// host that cannot keep up, pause frames at congestion downstream. None of it
// is visible from an interface listing.
//
// Linux only — ethtool is a Linux ioctl interface. That is a capability, not a
// silent empty result, so the route refuses with a 501 that names the reason
// (#749, #750).

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/capabilities"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/netif"
)

// DriverStatsResponse is the curated counter set for one interface.
type DriverStatsResponse struct {
	Interface string                 `json:"interface"`
	Counters  []netif.CuratedCounter `json:"counters"`
	// Total is how many counters the driver exposed in all, so an operator can
	// see that the curated view is a selection rather than the whole story.
	Total int `json:"total"`
}

// handleDriverStats serves GET /api/v1/telemetry/interface/driver-stats.
func (s *Server) handleDriverStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !s.requirePlatform(w, r, capabilities.DriverStatistics) {
		return
	}

	name := r.URL.Query().Get("interface")
	if name == "" {
		name = s.defaultInterface()
	}
	if name == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.network.invalidInterface"), "")

		return
	}

	raw, err := netif.DriverStats(name)
	if err != nil {
		logger.WarnContext(r.Context(), "Failed to read driver statistics",
			"interface", name, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.driverStatsFailed"), "")

		return
	}

	sendJSONResponse(w, logger, http.StatusOK, DriverStatsResponse{
		Interface: name,
		Counters:  netif.Curate(raw),
		Total:     len(raw),
	})
}
