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
	// Resolve to the package constant so nothing the caller sent reaches the
	// log or the receiver lookup (CodeQL go/log-injection cannot see a guard).
	var channel alerts.Channel
	switch req.Channel {
	case alerts.ChannelEmail:
		channel = alerts.ChannelEmail
	case alerts.ChannelWebhook:
		channel = alerts.ChannelWebhook
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "channel must be \"email\" or \"webhook\"", "")
		return
	}
	if s.alertDelivery == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "alert delivery is not running", "")
		return
	}

	err := s.alertDelivery.SendTest(ctx, channel)
	switch {
	case err == nil:
		writeJSON(w, r, map[string]any{"channel": channel, "sent": true})
	case errors.Is(err, alertdelivery.ErrNotConfigured):
		sendErrorResponseWithDetails(w, logger, http.StatusConflict,
			ErrCodeConflict, err.Error(), "")
	default:
		logger.WarnContext(ctx, "alert test send failed", "channel", channel, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadGateway,
			ErrCodeDeliveryFailed, err.Error(), "")
	}
}
