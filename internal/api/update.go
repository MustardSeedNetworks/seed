package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/update"
	"github.com/MustardSeedNetworks/seed/internal/update/lifecycle"
)

// Software-update transport (ADR-0020). These handlers decode the request,
// call the internal/update/lifecycle use-case via s.updateLifecycle, shape the
// response, and map lifecycle.ErrUnavailable to the pre-strangle 503. All
// orchestration (download/apply preconditions, status read-model assembly,
// configuration patching) lives in the use-case.

// sendUpdateError sends a standardized error response for update endpoints.
func sendUpdateError(w http.ResponseWriter, r *http.Request, status int, message string) {
	logger := logging.FromContext(r.Context())
	sendErrorResponseWithDetails(w, logger, status, "update_error", message, "")
}

// sendUpdateJSON sends a JSON response for update endpoints.
//
//nolint:unparam // status is kept for API consistency, currently always StatusOK.
func sendUpdateJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	logger := logging.FromContext(r.Context())
	sendJSONResponse(w, logger, status, data)
}

// UpdateCheckResponse represents the response from an update check.
type UpdateCheckResponse struct {
	*update.UpdateInfo
}

// UpdateStatusResponse represents the current update status.
type UpdateStatusResponse struct {
	*update.UpdateStatus

	LastCheck      string `json:"lastCheck,omitempty"`
	UpdateReady    bool   `json:"updateReady"`
	RequiresAction bool   `json:"requiresAction"`
}

// UpdateConfigResponse represents the update configuration.
type UpdateConfigResponse struct {
	Enabled           bool   `json:"enabled"`
	CheckInterval     string `json:"checkInterval"`
	AutoDownload      bool   `json:"autoDownload"`
	AutoApply         bool   `json:"autoApply"`
	IncludePrerelease bool   `json:"includePrerelease"`
}

// UpdateConfigRequest represents a request to update the update configuration.
type UpdateConfigRequest struct {
	Enabled           *bool   `json:"enabled,omitempty"`
	CheckInterval     *string `json:"checkInterval,omitempty"`
	AutoDownload      *bool   `json:"autoDownload,omitempty"`
	AutoApply         *bool   `json:"autoApply,omitempty"`
	IncludePrerelease *bool   `json:"includePrerelease,omitempty"`
}

// updateConfigResponse shapes a domain config into the transport DTO.
func updateConfigResponse(config update.UpdateConfig) UpdateConfigResponse {
	return UpdateConfigResponse{
		Enabled:           config.Enabled,
		CheckInterval:     config.CheckInterval.String(),
		AutoDownload:      config.AutoDownload,
		AutoApply:         config.AutoApply,
		IncludePrerelease: config.IncludePrerelease,
	}
}

// registerUpdateRoutes registers update-related HTTP routes.
func (s *Server) registerUpdateRoutes() {
	op := database.RoleOperator
	// These use Go 1.22 method-prefixed patterns; the method is part of the path.
	s.registerAll([]route{
		{path: "GET /api/v1/updates/check", handler: s.handleUpdateCheck},
		{path: "GET /api/v1/updates/status", handler: s.handleUpdateStatus},
		{path: "GET /api/v1/updates/info", handler: s.handleUpdateInfo},
		// Update actions mutate system state → operator-or-above only (#1226).
		{path: "POST /api/v1/updates/download", handler: s.handleUpdateDownload, minRole: op},
		{path: "POST /api/v1/updates/apply", handler: s.handleUpdateApply, minRole: op},
		{path: "POST /api/v1/updates/rollback", handler: s.handleUpdateRollback, minRole: op},
		{path: "GET /api/v1/updates/config", handler: s.handleGetUpdateConfig},
		{path: "PATCH /api/v1/updates/config", handler: s.handleUpdateConfig, minRole: op},
	})
}

// handleUpdateCheck checks for available updates.
func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	info, err := s.updateLifecycle.Check(r.Context())
	if err != nil {
		if errors.Is(err, lifecycle.ErrUnavailable) {
			sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
			return
		}
		logger.WarnContext(r.Context(), "Update check failed", "error", err)
		sendUpdateError(w, r, http.StatusInternalServerError, "Failed to check for updates")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, UpdateCheckResponse{UpdateInfo: info})
}

// handleUpdateStatus returns the current update status.
func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.updateLifecycle.Status()
	if err != nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	resp := UpdateStatusResponse{
		UpdateStatus:   &status.Status,
		UpdateReady:    status.Ready,
		RequiresAction: status.RequiresAction,
	}
	if !status.LastCheck.IsZero() {
		resp.LastCheck = status.LastCheck.Format(time.RFC3339)
	}

	sendUpdateJSON(w, r, http.StatusOK, resp)
}

// handleUpdateInfo returns information about available updates.
func (s *Server) handleUpdateInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.updateLifecycle.Info()
	if err != nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, UpdateCheckResponse{UpdateInfo: info})
}

// handleUpdateDownload downloads the available update.
func (s *Server) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	switch err := s.updateLifecycle.Download(r.Context()); {
	case errors.Is(err, lifecycle.ErrUnavailable):
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	case errors.Is(err, lifecycle.ErrNoUpdate):
		sendUpdateError(w, r, http.StatusBadRequest, "No update available")
		return
	case err != nil:
		logger.WarnContext(r.Context(), "Update download failed", "error", err)
		sendUpdateError(w, r, http.StatusInternalServerError, "Failed to download update")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, map[string]any{
		"status":  "downloaded",
		"message": "Update downloaded successfully",
	})
}

// handleUpdateApply applies the downloaded update.
func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	switch err := s.updateLifecycle.Apply(r.Context()); {
	case errors.Is(err, lifecycle.ErrUnavailable):
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	case errors.Is(err, lifecycle.ErrNotDownloaded):
		sendUpdateError(w, r, http.StatusBadRequest, "No update downloaded")
		return
	case err != nil:
		logger.WarnContext(r.Context(), "Update apply failed", "error", err)
		sendUpdateError(w, r, http.StatusInternalServerError, "Failed to apply update")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, map[string]any{
		"status":  "applied",
		"message": "Update applied successfully. Restart required.",
	})
}

// handleUpdateRollback rolls back to the previous version.
func (s *Server) handleUpdateRollback(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	switch err := s.updateLifecycle.Rollback(); {
	case errors.Is(err, lifecycle.ErrUnavailable):
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	case err != nil:
		logger.WarnContext(r.Context(), "Update rollback failed", "error", err)
		sendUpdateError(w, r, http.StatusInternalServerError, "Failed to rollback update")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, map[string]any{
		"status":  "rolled_back",
		"message": "Rolled back to previous version. Restart required.",
	})
}

// handleGetUpdateConfig returns the current update configuration.
func (s *Server) handleGetUpdateConfig(w http.ResponseWriter, r *http.Request) {
	config, err := s.updateLifecycle.Config()
	if err != nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, updateConfigResponse(config))
}

// handleUpdateConfig updates the update configuration.
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Strict decode — unknown fields rejected, body size capped.
	// This endpoint uses sendUpdateError (a localized variant with
	// release-channel context), so we keep that envelope shape and just
	// harden the decode inline.
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req UpdateConfigRequest
	if err := dec.Decode(&req); err != nil {
		sendUpdateError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	config, err := s.updateLifecycle.Configure(lifecycle.ConfigPatch{
		Enabled:           req.Enabled,
		CheckInterval:     req.CheckInterval,
		AutoDownload:      req.AutoDownload,
		AutoApply:         req.AutoApply,
		IncludePrerelease: req.IncludePrerelease,
	})
	switch {
	case errors.Is(err, lifecycle.ErrUnavailable):
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	case errors.Is(err, lifecycle.ErrInvalidInterval):
		sendUpdateError(w, r, http.StatusBadRequest,
			"Invalid checkInterval: must be a duration of at least 1m")
		return
	case err != nil:
		sendUpdateError(w, r, http.StatusInternalServerError, "Failed to update configuration")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, updateConfigResponse(config))
}
