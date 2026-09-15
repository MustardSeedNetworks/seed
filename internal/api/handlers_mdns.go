package api

// handlers_mdns.go serves the Bonjour/DNS-SD browse of #364: what services
// are advertised on the attached segment, and whether mDNS from another
// subnet is reaching it.
//
// The cross-subnet verdict is returned as a state plus its evidence, not as a
// sentence. The sentence a technician reads ("no reflector detected between
// these subnets") is composed in the UI, so it exists in every locale rather
// than only in the one the server was built with.

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/bonjour"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// bonjourWindowParam bounds how long the browse listens. The browser clamps
// it; the handler only rejects what is not a duration at all.
const bonjourWindowParam = "window"

// handleBonjourBrowse serves GET /api/v1/discovery/bonjour.
func (s *Server) handleBonjourBrowse(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	window, err := bonjourWindow(r)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mdns.invalidWindow"), "")
		return
	}

	browser := bonjour.NewBrowser(s.getInterfaceFromRequest(r))
	if window > 0 {
		browser.SetWindow(window)
	}

	result, err := browser.Browse(r.Context())
	if err != nil {
		// Joining the multicast group is the failure a technician will
		// actually hit — a down interface, or a name that does not exist —
		// so it is reported as unavailable rather than as a server fault.
		logger.WarnContext(r.Context(), "bonjour browse failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.mdns.browseUnavailable"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// bonjourWindow reads the optional window parameter. Absent is zero, meaning
// the browser's own default.
func bonjourWindow(r *http.Request) (time.Duration, error) {
	raw := r.URL.Query().Get(bonjourWindowParam)
	if raw == "" {
		return 0, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0, errors.New("window must be a positive number of seconds")
	}
	return time.Duration(seconds) * time.Second, nil
}
