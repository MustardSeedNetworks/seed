package api

// Guest-network isolation audit handlers (#397). Lets the user
// configure a list of sensitive internal IP addresses (EMR, PACS,
// etc.) and trigger an on-demand audit from the UI; the audit probes
// the configured targets and raises a Critical alert in the UI when
// any are reachable from the network the appliance is currently on.
//

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/security/guestaudit"
	secsettings "github.com/MustardSeedNetworks/seed/internal/security/settings"
)

const guestAuditRunTimeout = 60 * time.Second

// writeGuestAuditError maps a guest-audit settings error to its HTTP response: a
// GuestAuditValidationError becomes a 400 with the field-specific i18n message and
// the offending value as detail; anything else is a 500 save failure.
func (s *Server) writeGuestAuditError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var vErr secsettings.GuestAuditValidationError
	if errors.As(err, &vErr) {
		msgKey := "errors.guestAudit.invalidTarget"
		detail := vErr.Value
		if vErr.Kind == "port" {
			msgKey = "errors.guestAudit.invalidPort"
			detail = ""
		}
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeValidation,
			localizer.T(msgKey), detail,
		)
		return
	}

	logger.ErrorContext(r.Context(), "Failed to save guest-audit settings", "error", err)
	sendErrorResponseWithDetails(
		w, logger, http.StatusInternalServerError, ErrCodeInternal,
		localizer.T("errors.config.failedToSave"), "",
	)
}

// handleGuestAuditSettings returns or updates the configured target list.
// Routes: GET and PUT on /api/v1/security/guest-audit/settings.
func (s *Server) handleGuestAuditSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, logger, http.StatusOK, s.securitySettings.GuestAudit())

	case http.MethodPut:
		var settings config.GuestNetworkAuditConfig
		if !decodeJSONStrictLocalized(w, r, &settings, MaxBodySizeJSON, logger, localizer) {
			return
		}

		if err := s.securitySettings.UpdateGuestAudit(settings); err != nil {
			s.writeGuestAuditError(w, r, err)
			return
		}

		sendJSONResponse(
			w,
			logger,
			http.StatusOK,
			map[string]string{"status": "updated"},
		)

	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
	}
}

// handleGuestAuditRun executes the guest-network isolation audit on demand.
// Route: POST /api/v1/security/guest-audit/run.
func (s *Server) handleGuestAuditRun(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	cfg := s.securitySettings.GuestAudit()
	if !cfg.Enabled {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.guestAudit.notEnabled"),
			"",
		)
		return
	}
	if len(cfg.Targets) == 0 {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.guestAudit.noTargets"),
			"",
		)
		return
	}

	targets := make([]guestaudit.Target, 0, len(cfg.Targets))
	for _, t := range cfg.Targets {
		targets = append(targets, guestaudit.Target{IP: t.IP, Label: t.Label})
	}

	ctx, cancel := context.WithTimeout(r.Context(), guestAuditRunTimeout)
	defer cancel()

	report, err := guestaudit.Run(ctx, guestaudit.Options{
		Targets: targets,
		Ports:   cfg.Ports,
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "Guest audit run failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.guestAudit.runFailed"),
			"",
		)
		return
	}

	if report.IsolationFailed {
		logger.WarnContext(r.Context(), "Guest network isolation FAILED",
			"reachable_targets", report.ReachableTargets,
			"total_targets", report.TotalTargets,
		)
	} else {
		logger.InfoContext(r.Context(), "Guest network isolation verified",
			"total_targets", report.TotalTargets,
		)
	}

	sendJSONResponse(w, logger, http.StatusOK, report)
}
