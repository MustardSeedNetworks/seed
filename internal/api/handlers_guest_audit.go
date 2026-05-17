package api

// Guest-network isolation audit handlers (#397). Lets the user
// configure a list of sensitive internal IP addresses (EMR, PACS,
// etc.) and trigger an on-demand audit from the UI; the audit probes
// the configured targets and raises a Critical alert in the UI when
// any are reachable from the network the appliance is currently on.
//

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/services/shell/guestaudit"
	"github.com/krisarmstrong/seed/internal/validation"
)

const guestAuditRunTimeout = 60 * time.Second

// handleGuestAuditSettings returns or updates the configured target list.
// Routes: GET and PUT on /api/v1/shell/guest-audit/settings.
func (s *Server) handleGuestAuditSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, logger, http.StatusOK, s.config.Security.GuestNetworkAudit)

	case http.MethodPut:
		var settings config.GuestNetworkAuditConfig
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			logger.WarnContext(r.Context(), "Invalid guest-audit settings body", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusBadRequest,
				ErrCodeBadRequest,
				localizer.T("errors.api.invalidRequestBody"),
				"",
			)
			return
		}

		// Validate each target before persisting.
		for _, t := range settings.Targets {
			if !validation.IsValidIP(t.IP) {
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusBadRequest,
					ErrCodeValidation,
					localizer.T("errors.guestAudit.invalidTarget"),
					t.IP,
				)
				return
			}
		}
		for _, p := range settings.Ports {
			if p < 1 || p > 65535 {
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusBadRequest,
					ErrCodeValidation,
					localizer.T("errors.guestAudit.invalidPort"),
					"",
				)
				return
			}
		}

		s.config.Lock()
		s.config.Security.GuestNetworkAudit = settings
		s.config.Unlock()

		if err := s.config.Save(s.configPath); err != nil {
			logger.ErrorContext(r.Context(), "Failed to save guest-audit settings", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				localizer.T("errors.config.failedToSave"),
				"",
			)
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
// Route: POST /api/v1/shell/guest-audit/run.
func (s *Server) handleGuestAuditRun(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	cfg := s.config.Security.GuestNetworkAudit
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
