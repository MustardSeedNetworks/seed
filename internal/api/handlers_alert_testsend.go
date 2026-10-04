package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// ErrCodeDeliveryFailed is a receiver that refused or could not be reached.
const ErrCodeDeliveryFailed = "DELIVERY_FAILED"

// handleAlertTestSend serves POST /api/v1/settings/alerts/test (#2997): send a
// synthetic alert on one channel now, through the receiver that is saved, and
// answer with what the receiver said. A relay that refuses AUTH or a webhook
// that answers 401 is found here, by the operator who just configured it,
// rather than by the first real alert that silently never arrives.
//
// It tests the saved receiver, never one named in the request: the route
// would otherwise make the daemon connect wherever a caller asks.
func (s *Server) handleAlertTestSend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx)

	var req struct {
		Channel alerts.Channel `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "body must be {\"channel\": \"email\" | \"webhook\"}", "")
		return
	}
	if req.Channel != alerts.ChannelEmail && req.Channel != alerts.ChannelWebhook {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "channel must be \"email\" or \"webhook\"", "")
		return
	}
	if s.alertDelivery == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "alert delivery is not running", "")
		return
	}

	err := s.alertDelivery.SendTest(ctx, req.Channel)
	switch {
	case err == nil:
		writeJSON(w, r, map[string]any{"channel": req.Channel, "sent": true})
	case errors.Is(err, alertdelivery.ErrNotConfigured):
		sendErrorResponseWithDetails(w, logger, http.StatusConflict,
			ErrCodeConflict, err.Error(), "")
	default:
		logger.WarnContext(ctx, "alert test send failed", "channel", req.Channel, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadGateway,
			ErrCodeDeliveryFailed, err.Error(), "")
	}
}
