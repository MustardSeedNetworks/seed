// Package api provides HTTP handlers for profile management.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/database"
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

	// Check if database is available
	if s.db == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Database not available", "")
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListProfiles returns all profiles.
func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	profiles, err := s.db.Profiles().List(ctx)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to list profiles", err.Error())
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
	ctx := r.Context()

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.Name == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "Profile name is required", "")
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

	if err := s.db.Profiles().Create(ctx, profile); err != nil {
		if errors.Is(err, database.ErrProfileNameExists) {
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, "Profile with this name already exists", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to create profile", err.Error())
		return
	}

	// If this profile is set as default, it was already handled by the repository
	sendJSONResponse(w, logger, http.StatusCreated, profileToResponse(profile))
}

// handleGetProfile returns a single profile by ID.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	profile, err := s.db.Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, "Profile not found", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to get profile", err.Error())
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleUpdateProfile updates an existing profile.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "Invalid request body", err.Error())
		return
	}

	// Get existing profile
	profile, err := s.db.Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, "Profile not found", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to get profile", err.Error())
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

	if err := s.db.Profiles().Update(ctx, profile); err != nil {
		if errors.Is(err, database.ErrProfileNameExists) {
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, "Profile with this name already exists", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to update profile", err.Error())
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleDeleteProfile deletes a profile.
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Check if profile exists
	profile, err := s.db.Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, "Profile not found", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to get profile", err.Error())
		return
	}

	// Prevent deleting the default profile
	if profile.IsDefault {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "Cannot delete the default profile", "")
		return
	}

	// Check if this is the active profile
	activeID, _ := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile) //nolint:errcheck // empty string is fine if setting not found
	if activeID == id {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "Cannot delete the currently active profile", "")
		return
	}

	if err := s.db.Profiles().Delete(ctx, id); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to delete profile", err.Error())
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"message": "Profile deleted successfully",
	})
}

// handleActiveProfile handles getting and setting the active profile.
func (s *Server) handleActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.db == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Database not available", "")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetActiveProfile(w, r)
	case http.MethodPost, http.MethodPut:
		s.handleSetActiveProfile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetActiveProfile returns the currently active profile.
func (s *Server) handleGetActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	// Get active profile ID from settings
	activeID, err := s.db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to get active profile setting", err.Error())
		return
	}

	// If no active profile set, return the default profile
	if activeID == "" {
		profile, err := s.db.Profiles().GetDefault(ctx)
		if err != nil {
			if errors.Is(err, database.ErrProfileNotFound) {
				sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
					ErrCodeNotFound, "No active or default profile found", "Please create a profile first")
				return
			}
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, "Failed to get default profile", err.Error())
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
		return
	}

	// Get the active profile
	profile, err := s.db.Profiles().Get(ctx, activeID)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			// Active profile was deleted, fall back to default
			profile, err = s.db.Profiles().GetDefault(ctx)
			if err != nil {
				sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
					ErrCodeNotFound, "Active profile not found and no default profile exists", "")
				return
			}
			// Update setting to use default
			_ = s.db.Settings().Set(ctx, database.SettingKeyActiveProfile, profile.ID) //nolint:errcheck // best effort, non-critical
		} else {
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, "Failed to get active profile", err.Error())
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleSetActiveProfile sets the active profile.
func (s *Server) handleSetActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	ctx := r.Context()

	var req struct {
		ProfileID string `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.ProfileID == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, "profile_id is required", "")
		return
	}

	// Verify profile exists
	profile, err := s.db.Profiles().Get(ctx, req.ProfileID)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, "Profile not found", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to get profile", err.Error())
		return
	}

	// Set active profile in settings
	if err := s.db.Settings().Set(ctx, database.SettingKeyActiveProfile, req.ProfileID); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to set active profile", err.Error())
		return
	}

	// Broadcast profile change via WebSocket
	s.wsHub.Broadcast(Message{
		Type: "profileChanged",
		Payload: map[string]interface{}{
			"profile_id":   profile.ID,
			"profile_name": profile.Name,
		},
	})

	sendJSONResponse(w, logger, http.StatusOK, map[string]interface{}{
		"message": "Active profile updated",
		"profile": profileToResponse(profile),
	})
}

// handleDuplicateProfile creates a copy of an existing profile.
func (s *Server) handleDuplicateProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.db == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Database not available", "")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract profile ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	path = strings.TrimSuffix(path, "/duplicate")
	id := path

	// Get source profile
	source, err := s.db.Profiles().Get(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, "Profile not found", "")
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to get profile", err.Error())
		return
	}

	// Parse optional new name from request body
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck // empty body is valid, name defaults to copy

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

	if err := s.db.Profiles().Create(ctx, duplicate); err != nil {
		if errors.Is(err, database.ErrProfileNameExists) {
			// Try with timestamp suffix
			duplicate.Name = fmt.Sprintf("%s (%s)", source.Name, time.Now().Format("2006-01-02 15:04"))
			if err := s.db.Profiles().Create(ctx, duplicate); err != nil {
				sendErrorResponseWithDetails(w, logger, http.StatusConflict,
					ErrCodeConflict, "Profile with this name already exists", "")
				return
			}
		} else {
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, "Failed to create duplicate", err.Error())
			return
		}
	}

	sendJSONResponse(w, logger, http.StatusCreated, profileToResponse(duplicate))
}

// handleImportProfiles imports profiles from JSON.
func (s *Server) handleImportProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.db == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Database not available", "")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var req ProfileImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, "Invalid JSON format", err.Error())
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
		existing, _ := s.db.Profiles().GetByName(ctx, p.Name) //nolint:errcheck // not found returns nil
		if existing != nil {
			if req.Overwrite {
				// Update existing profile
				existing.Description = p.Description
				existing.ConfigJSON = string(p.Config)
				if err := s.db.Profiles().Update(ctx, existing); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Profile '%s': failed to update - %v", p.Name, err))
					result.Skipped++
				} else {
					result.Updated++
				}
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Profile '%s': already exists (use overwrite=true to update)", p.Name))
				result.Skipped++
			}
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

		if err := s.db.Profiles().Create(ctx, profile); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Profile '%s': failed to create - %v", p.Name, err))
			result.Skipped++
		} else {
			result.Created++
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// handleExportProfiles exports all profiles to JSON.
func (s *Server) handleExportProfiles(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.db == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, "Database not available", "")
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	profiles, err := s.db.Profiles().List(ctx)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Failed to list profiles", err.Error())
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
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=seed-profiles-%s.json", time.Now().Format("2006-01-02")))

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
