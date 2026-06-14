package api

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
)

// ============================================================================
// Settings Handlers (fixes #544 - split from handlers.go)
// ============================================================================

// handleSettingsDefaults returns all default settings as the single source of truth.
// This eliminates the need for duplicated DEFAULT_* constants in the frontend.
// The defaults are served from the backend's DefaultConfig() function.
func (s *Server) handleSettingsDefaults(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	defaults := config.GetDefaultSettings()
	sendJSONResponse(w, logger, http.StatusOK, defaults)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPut:
		s.updateSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
	}
}

// getSettings serves the mutable application settings. Thin transport (ADR-0020):
// the read model + ETag are assembled by the settings-management service.
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	settings, etag := s.settingsManagement.Get()
	w.Header().Set("ETag", etag)
	sendJSONResponse(w, logger, http.StatusOK, settings)
}

// updateSettings applies a settings update. Thin transport: decode, delegate the
// optimistic-ETag check + apply + persist to the management service, then persist
// to the active profile and emit the fresh token. Typed service errors map to
// HTTP (conflict → 412, validation → 400, store → 500).
func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()
	localizer := i18n.FromRequest(r)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var updates map[string]any
	if !decodeJSONStrict(w, r, &updates, MaxBodySizeJSON) {
		return
	}

	switch err := s.settingsManagement.Update(updates, parseIfMatch(r)); {
	case errors.Is(err, management.ErrConflict):
		sendErrorResponseWithDetails(w, logger, http.StatusPreconditionFailed,
			ErrCodeConflict, localizer.T("errors.settings.conflict"), "")
		return
	case errors.Is(err, management.ErrValidation):
		logger.WarnContext(ctx, "Invalid settings format", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "Invalid settings format. Check server logs for details.", "")
		return
	case err != nil:
		logger.ErrorContext(ctx, "Failed to save config", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to save config", "")
		return
	}

	// Also persist settings to the active profile (fixes #781). The use-case is a
	// no-op when no database/profile is wired (ADR-0016 phase 3).
	if err := s.settingsStore.SaveToActiveProfile(ctx); err != nil {
		logger.ErrorContext(ctx, "Failed to save settings to profile", "error", err)
		sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
			"error": "Failed to save settings",
		})
		return
	}

	// Emit the fresh token so the client can chain a subsequent conditional write.
	w.Header().Set("ETag", s.settingsManagement.ETag())
	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getLinkSettings(w, r)
	case http.MethodPut:
		s.updateLinkSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
	}
}

// getLinkSettings returns current link settings via the settings use-case.
func (s *Server) getLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	sendJSONResponse(w, logger, http.StatusOK, s.settingsManagement.Link())
}

// updateLinkSettings updates link settings in config and saves to active profile.
func (s *Server) updateLinkSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var updates config.LinkConfig
	if !decodeJSONStrict(w, r, &updates, MaxBodySizeJSON) {
		return
	}

	// Validate + persist via the settings use-case; an invalid mode is a 400.
	if err := s.settingsManagement.UpdateLink(updates); err != nil {
		if errors.Is(err, management.ErrValidation) {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeValidation, "Invalid mode value", "")
			return
		}
		logger.ErrorContext(ctx, "Failed to save link settings", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to save link settings", "")
		return
	}

	// Persist to the active profile (ADR-0016 phase 3; no-op without a db).
	if err := s.settingsStore.SaveToActiveProfile(ctx); err != nil {
		logger.ErrorContext(ctx, "Failed to save link settings to profile", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to save link settings", "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}

// ============================================================================
// Cable Test Settings Handlers (fixes #740)
// ============================================================================

// handleCableTestSettings handles GET/PUT for /api/settings/cable.
// Cable test settings control TDR cable diagnostics behavior.
func (s *Server) handleCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getCableTestSettings(w, r)
	case http.MethodPut:
		s.updateCableTestSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
	}
}

// getCableTestSettings returns current cable test settings via the settings use-case.
func (s *Server) getCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	sendJSONResponse(w, logger, http.StatusOK, s.settingsManagement.CableTest())
}

// updateCableTestSettings updates cable test settings in config and saves to active profile.
func (s *Server) updateCableTestSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var updates config.CableTestConfig
	if !decodeJSONStrict(w, r, &updates, MaxBodySizeJSON) {
		return
	}

	// Persist via the settings use-case, then to the active profile.
	if err := s.settingsManagement.UpdateCableTest(updates); err != nil {
		logger.ErrorContext(ctx, "Failed to save cable test settings", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to save cable test settings", "")
		return
	}

	// Persist to the active profile (ADR-0016 phase 3; no-op without a db).
	if err := s.settingsStore.SaveToActiveProfile(ctx); err != nil {
		logger.ErrorContext(ctx, "Failed to save cable test settings to profile", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to save cable test settings", "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}
