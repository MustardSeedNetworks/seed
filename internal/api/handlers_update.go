package api

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/update"
)

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

	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	info, err := updateService.CheckForUpdate(r.Context())
	if err != nil {
		logger.WarnContext(r.Context(), "Update check failed", "error", err)
		sendUpdateError(w, r, http.StatusInternalServerError, "Failed to check for updates")
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, UpdateCheckResponse{UpdateInfo: info})
}

// handleUpdateStatus returns the current update status.
func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	status := updateService.GetStatus()
	lastCheck := updateService.GetLastCheckTime()
	info := updateService.GetUpdateInfo()

	resp := UpdateStatusResponse{
		UpdateStatus: &status,
		UpdateReady:  updateService.IsUpdateDownloaded(),
	}

	if !lastCheck.IsZero() {
		resp.LastCheck = lastCheck.Format("2006-01-02T15:04:05Z07:00")
	}

	// Determine if user action is needed
	if info != nil && info.Available && !updateService.IsUpdateDownloaded() {
		resp.RequiresAction = true
	}

	sendUpdateJSON(w, r, http.StatusOK, resp)
}

// handleUpdateInfo returns information about available updates.
func (s *Server) handleUpdateInfo(w http.ResponseWriter, r *http.Request) {
	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	info := updateService.GetUpdateInfo()
	if info == nil {
		// No update check has been performed yet
		sendUpdateJSON(w, r, http.StatusOK, update.UpdateInfo{
			Available:      false,
			CurrentVersion: "",
			LatestVersion:  "",
		})
		return
	}

	sendUpdateJSON(w, r, http.StatusOK, UpdateCheckResponse{UpdateInfo: info})
}

// handleUpdateDownload downloads the available update.
func (s *Server) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	info := updateService.GetUpdateInfo()
	if info == nil || !info.Available {
		sendUpdateError(w, r, http.StatusBadRequest, "No update available")
		return
	}

	// Start download (this is blocking)
	_, err := updateService.DownloadUpdate(r.Context(), nil)
	if err != nil {
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

	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	if !updateService.IsUpdateDownloaded() {
		sendUpdateError(w, r, http.StatusBadRequest, "No update downloaded")
		return
	}

	// Apply the update
	if err := updateService.ApplyUpdate(r.Context()); err != nil {
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

	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	if err := updateService.Rollback(); err != nil {
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
	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	config := updateService.GetConfig()

	sendUpdateJSON(w, r, http.StatusOK, UpdateConfigResponse{
		Enabled:           config.Enabled,
		CheckInterval:     config.CheckInterval.String(),
		AutoDownload:      config.AutoDownload,
		AutoApply:         config.AutoApply,
		IncludePrerelease: config.IncludePrerelease,
	})
}

// handleUpdateConfig updates the update configuration.
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	updateService := s.services.GetUpdateService()
	if updateService == nil {
		sendUpdateError(w, r, http.StatusServiceUnavailable, "Update service not available")
		return
	}

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

	config := updateService.GetConfig()

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.AutoDownload != nil {
		config.AutoDownload = *req.AutoDownload
	}
	if req.AutoApply != nil {
		config.AutoApply = *req.AutoApply
	}
	if req.IncludePrerelease != nil {
		config.IncludePrerelease = *req.IncludePrerelease
	}

	updateService.SetConfig(config)

	sendUpdateJSON(w, r, http.StatusOK, UpdateConfigResponse{
		Enabled:           config.Enabled,
		CheckInterval:     config.CheckInterval.String(),
		AutoDownload:      config.AutoDownload,
		AutoApply:         config.AutoApply,
		IncludePrerelease: config.IncludePrerelease,
	})
}
