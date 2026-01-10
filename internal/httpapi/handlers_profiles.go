package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ProfileRequest represents a profile create/update request.
type ProfileRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsDefault   bool            `json:"is_default"`
}

// ProfileResponse represents a profile in API responses.
type ProfileResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsDefault   bool            `json:"is_default"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// ProfileListResponse represents the list profiles response.
type ProfileListResponse struct {
	Profiles []ProfileResponse `json:"profiles"`
	Total    int               `json:"total"`
}

// ProfileImportRequest represents an import request.
type ProfileImportRequest struct {
	Version   string           `json:"version"`
	Profiles  []ProfileRequest `json:"profiles"`
	Overwrite bool             `json:"overwrite"`
}

// ProfileImportResponse represents an import result.
type ProfileImportResponse struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// ProfileExportResponse represents an export response.
type ProfileExportResponse struct {
	Version    string            `json:"version"`
	ExportedAt string            `json:"exported_at"`
	Profiles   []ProfileResponse `json:"profiles"`
}

// handleProfiles routes profile requests by method.
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// Check if database is available
	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.profile.dbNotAvailable"), "") // fixes #694
		return
	}

	// Extract profile ID from path if present
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" && r.Method == http.MethodGet:
		s.handleListProfiles(w, r)
	case path == "" && r.Method == http.MethodPost:
		s.handleCreateProfile(w, r)
	case strings.HasSuffix(path, "/duplicate") && r.Method == http.MethodPost:
		s.handleDuplicateProfile(w, r)
	case path != "" && r.Method == http.MethodGet:
		s.handleGetProfile(w, r, path)
	case path != "" && r.Method == http.MethodPut:
		s.handleUpdateProfile(w, r, path)
	case path != "" && r.Method == http.MethodDelete:
		s.handleDeleteProfile(w, r, path)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
	}
}

// handleListProfiles returns all profiles.
func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	profiles, err := s.db().Profiles().List(ctx)
	if err != nil {
		logger.Error("Failed to list profiles", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.listFailed"), "") // fixes #694, #H7
		return
	}

	response := ProfileListResponse{
		Profiles: make([]ProfileResponse, 0, len(profiles)),
		Total:    len(profiles),
	}

	for _, p := range profiles {
		response.Profiles = append(response.Profiles, profileToResponse(p))
	}

	sendJSONResponse(w, logger, http.StatusOK, response)
}

// handleCreateProfile creates a new profile.
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "") // fixes #694, #H7
		return
	}

	// Validate required fields
	if req.Name == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.profile.nameRequired"), "") // fixes #694
		return
	}

	// Create profile
	profile := &database.Profile{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		ConfigJSON:  string(req.Config),
		IsDefault:   req.IsDefault,
	}

	if err := s.db().Profiles().Create(ctx, profile); err != nil {
		if errors.Is(err, database.ErrProfileNameExists) {
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, localizer.T("errors.profile.nameExists"), "") // fixes #694
			return
		}
		logger.Error("Failed to create profile", "error", err, "profile_name", req.Name)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.createFailed"), "") // fixes #694, #H7
		return
	}

	// If this profile is set as default, it was already handled by the repository
	sendJSONResponse(w, logger, http.StatusCreated, profileToResponse(profile))
}

// handleGetProfile returns a single profile by ID.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	profile, err := s.db().Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleUpdateProfile updates an existing profile.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "") // fixes #694, #H7
		return
	}

	// Get existing profile
	profile, err := s.db().Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		return
	}

	// Update fields
	if req.Name != "" {
		profile.Name = req.Name
	}
	profile.Description = req.Description
	if req.Config != nil {
		profile.ConfigJSON = string(req.Config)
	}
	profile.IsDefault = req.IsDefault

	if updateErr := s.db().Profiles().Update(ctx, profile); updateErr != nil {
		if errors.Is(updateErr, database.ErrProfileNameExists) {
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, localizer.T("errors.profile.nameExists"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.updateFailed"), "") // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleDeleteProfile deletes a profile.
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	// Check if profile exists
	profile, err := s.db().Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		return
	}

	// Prevent deleting the default profile
	if profile.IsDefault {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.profile.cannotDeleteDefault"), "") // fixes #694
		return
	}

	// Check if this is the active profile
	activeID, _ := s.db().Settings().
		GetValue(ctx, database.SettingKeyActiveProfile)

	if activeID == id {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.profile.cannotDeleteActive"), "") // fixes #694
		return
	}

	if deleteErr := s.db().Profiles().Delete(ctx, id); deleteErr != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.deleteFailed"), "") // fixes #694, #H7
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"message": "Profile deleted successfully",
	})
}

// handleActiveProfile handles getting and setting the active profile.
func (s *Server) handleActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.profile.dbNotAvailable"), "") // fixes #694
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetActiveProfile(w, r)
	case http.MethodPost, http.MethodPut:
		s.handleSetActiveProfile(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
	}
}

// handleGetActiveProfile returns the currently active profile.
func (s *Server) handleGetActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	// Get active profile ID from settings
	activeID, err := s.db().Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getActiveFailed"), "") // fixes #694, #H7
		return
	}

	// If no active profile set, return the default profile
	if activeID == "" {
		profile, defaultErr := s.db().Profiles().GetDefault(ctx)
		if defaultErr != nil {
			if errors.Is(defaultErr, database.ErrProfileNotFound) {
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusNotFound,
					ErrCodeNotFound,
					localizer.T("errors.profile.noActiveOrDefault"),
					"",
				) // fixes #694
				return
			}
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusInternalServerError,
				ErrCodeInternal,
				localizer.T("errors.profile.getDefaultFailed"),
				"",
			) // fixes #694, #H7
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
		return
	}

	// Get the active profile
	profile, err := s.db().Profiles().Get(ctx, activeID)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			// Active profile was deleted, fall back to default
			profile, err = s.db().Profiles().GetDefault(ctx)
			if err != nil {
				sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
					ErrCodeNotFound, localizer.T("errors.profile.activeNotFound"), "") // fixes #694
				return
			}
			// Update setting to use default
			_ = s.db().Settings().
				Set(ctx, database.SettingKeyActiveProfile, profile.ID)
		} else {
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleSetActiveProfile sets the active profile and applies its settings.
func (s *Server) handleSetActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	ctx := r.Context()

	var req struct {
		ProfileID string `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "") // fixes #694, #H7
		return
	}

	if req.ProfileID == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.profile.idRequired"), "") // fixes #694
		return
	}

	// Verify profile exists
	profile, err := s.db().Profiles().Get(ctx, req.ProfileID)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		return
	}

	// Set active profile in settings
	if setErr := s.db().Settings().Set(ctx, database.SettingKeyActiveProfile, req.ProfileID); setErr != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.setActiveFailed"), "") // fixes #694, #H7
		return
	}

	// Apply profile settings to the active config (fixes #781)
	// Uses Config.ApplyProfileJSON() - single source of truth, no ProfileSettings intermediate
	if profile.ConfigJSON != "" {
		if applyErr := s.config.ApplyProfileJSON(profile.ConfigJSON); applyErr != nil {
			logger.Warn(
				"Failed to parse profile settings, using defaults",
				"error",
				applyErr,
				"profile_id",
				profile.ID,
			)
		} else {
			// Save updated config to file (fixes #782 - return error instead of silent warning)
			if saveErr := s.config.Save(s.configPath); saveErr != nil {
				logger.Error("Failed to save config after profile switch", "error", saveErr)
				sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
					ErrCodeInternal, localizer.T("errors.config.failedToSave"), saveErr.Error())
				return
			}
			logger.Info("Applied profile settings", "profile_id", profile.ID, "profile_name", profile.Name)
		}
	}

	// Broadcast profile change via WebSocket
	s.wsHub().Broadcast(Message{
		Type: "profileChanged",
		Payload: map[string]any{
			"profile_id":   profile.ID,
			"profile_name": profile.Name,
		},
	})

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"message": "Active profile updated",
		"profile": profileToResponse(profile),
	})
}

// handleDuplicateProfile creates a copy of an existing profile.
func (s *Server) handleDuplicateProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.profile.dbNotAvailable"), "") // fixes #694
		return
	}

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	ctx := r.Context()

	// Extract profile ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	path = strings.TrimSuffix(path, "/duplicate")
	id := path

	// Get source profile
	source, err := s.db().Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		return
	}

	// Parse optional new name from request body
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	newName := req.Name
	if newName == "" {
		newName = source.Name + " (Copy)"
	}

	// Create duplicate
	duplicate := &database.Profile{
		ID:          uuid.New().String(),
		Name:        newName,
		Description: source.Description,
		ConfigJSON:  source.ConfigJSON,
		IsDefault:   false, // Duplicates are never default
	}

	if createErr := s.db().Profiles().Create(ctx, duplicate); createErr != nil {
		if errors.Is(createErr, database.ErrProfileNameExists) {
			// Try with timestamp suffix
			duplicate.Name = fmt.Sprintf(
				"%s (%s)",
				source.Name,
				time.Now().Format("2006-01-02 15:04"),
			)
			if retryErr := s.db().Profiles().Create(ctx, duplicate); retryErr != nil {
				sendErrorResponseWithDetails(w, logger, http.StatusConflict,
					ErrCodeConflict, localizer.T("errors.profile.nameExists"), "") // fixes #694
				return
			}
		} else {
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.duplicateFailed"), "") // fixes #694, #H7
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusCreated, profileToResponse(duplicate))
}

// handleImportProfiles imports profiles from JSON.
func (s *Server) handleImportProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.profile.dbNotAvailable"), "") // fixes #694
		return
	}

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	ctx := r.Context()

	var req ProfileImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.profile.invalidJson"), "") // fixes #694, #H7
		return
	}

	result := ProfileImportResponse{
		Errors: make([]string, 0),
	}

	for i, p := range req.Profiles {
		if p.Name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Profile %d: name is required", i+1))
			result.Skipped++
			continue
		}

		// Check if profile with this name exists
		existing, _ := s.db().Profiles().GetByName(ctx, p.Name)
		if existing != nil {
			s.handleExistingProfileImport(ctx, existing, &p, req.Overwrite, &result)
			continue
		}

		// Create new profile
		profile := &database.Profile{
			ID:          uuid.New().String(),
			Name:        p.Name,
			Description: p.Description,
			ConfigJSON:  string(p.Config),
			IsDefault:   false, // Never import as default
		}

		if err := s.db().Profiles().Create(ctx, profile); err != nil {
			result.Errors = append(
				result.Errors,
				fmt.Sprintf("Profile '%s': failed to create - %v", p.Name, err),
			)
			result.Skipped++
		} else {
			result.Created++
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// handleExistingProfileImport handles importing a profile that already exists.
func (s *Server) handleExistingProfileImport(
	ctx context.Context,
	existing *database.Profile,
	p *ProfileRequest,
	overwrite bool,
	result *ProfileImportResponse,
) {
	if !overwrite {
		result.Errors = append(
			result.Errors,
			fmt.Sprintf("Profile '%s': already exists (use overwrite=true to update)", p.Name),
		)
		result.Skipped++
		return
	}

	// Update existing profile
	existing.Description = p.Description
	existing.ConfigJSON = string(p.Config)
	if err := s.db().Profiles().Update(ctx, existing); err != nil {
		result.Errors = append(
			result.Errors,
			fmt.Sprintf("Profile '%s': failed to update - %v", p.Name, err),
		)
		result.Skipped++
		return
	}
	result.Updated++
}

// handleExportProfiles exports all profiles to JSON.
func (s *Server) handleExportProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.profile.dbNotAvailable"), "") // fixes #694
		return
	}

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	ctx := r.Context()

	profiles, err := s.db().Profiles().List(ctx)
	if err != nil {
		logger.Error("Failed to list profiles", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.listFailed"), "") // fixes #694, #H7
		return
	}

	response := ProfileExportResponse{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profiles:   make([]ProfileResponse, 0, len(profiles)),
	}

	for _, p := range profiles {
		response.Profiles = append(response.Profiles, profileToResponse(p))
	}

	// Set headers for file download
	w.Header().
		Set("Content-Disposition", fmt.Sprintf("attachment; filename=seed-profiles-%s.json", time.Now().Format("2006-01-02")))

	sendJSONResponse(w, logger, http.StatusOK, response)
}

// profileToResponse converts a database profile to an API response.
func profileToResponse(p *database.Profile) ProfileResponse {
	var configJSON json.RawMessage
	if p.ConfigJSON != "" {
		configJSON = json.RawMessage(p.ConfigJSON)
	} else {
		configJSON = json.RawMessage("{}")
	}

	return ProfileResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Config:      configJSON,
		IsDefault:   p.IsDefault,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}
