package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
)

// ProfileRequest represents a profile create/update request.
type ProfileRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsDefault   bool            `json:"isDefault"`
}

// ProfileResponse represents a profile in API responses.
type ProfileResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsDefault   bool            `json:"isDefault"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
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
	ExportedAt string            `json:"exportedAt"`
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

	profiles, err := s.profiles.List(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to list profiles", "error", err)
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

	// multi_client gate: Free/Starter may keep their bootstrap profile,
	// but a SECOND (or later) profile requires Pro. Checked before body
	// decode so we return 402 without scanning the request payload.
	if !s.enforceMultiClientGate(w, r) {
		return
	}

	var req ProfileRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	// Preserve the pre-strangle ordering: empty-name 400 fires before the
	// multi_interface gate so the message is stable for an empty over-cap body.
	if req.Name == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.profile.nameRequired"), "") // fixes #694
		return
	}

	// multi_interface gate (seed#1192): if the saved profile.Config exceeds
	// the Free/Starter 1+1 cap, Pro is required.
	if !s.enforceMultiInterfaceGate(w, r, req.Config) {
		return
	}

	profile, err := s.profiles.Create(r.Context(), catalog.NewProfile{
		Name:        req.Name,
		Description: req.Description,
		ConfigJSON:  string(req.Config),
		IsDefault:   req.IsDefault,
	})
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrNameRequired):
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeValidation, localizer.T("errors.profile.nameRequired"), "")
		case errors.Is(err, catalog.ErrNameExists):
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, localizer.T("errors.profile.nameExists"), "") // fixes #694
		default:
			logger.ErrorContext(r.Context(), "Failed to create profile", "error", err, "profile_name", req.Name)
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.createFailed"), "") // fixes #694, #H7
		}
		return
	}

	sendJSONResponse(w, logger, http.StatusCreated, profileToResponse(profile))
}

// handleGetProfile returns a single profile by ID.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	profile, err := s.profiles.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
			return
		}
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		return
	}

	w.Header().Set("ETag", profileETag(profile))
	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleUpdateProfile updates an existing profile.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req ProfileRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	// multi_interface gate (seed#1192): done before the DB read so a 402
	// short-circuits without a wasted query.
	if req.Config != nil && !s.enforceMultiInterfaceGate(w, r, req.Config) {
		return
	}

	update := catalog.ProfileUpdate{
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   req.IsDefault,
	}
	if req.Config != nil {
		cfg := string(req.Config)
		update.ConfigJSON = &cfg
	}

	// Optimistic concurrency (ADR re-arch Phase 5): an If-Match ETag, when
	// present, makes the write conditional on the profile not having changed
	// since the caller read it. Absent (or "*") => unconditional, as before.
	profile, err := s.profiles.Update(r.Context(), id, update, parseIfMatch(r))
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrNotFound):
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
		case errors.Is(err, catalog.ErrNameExists):
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, localizer.T("errors.profile.nameExists"), "") // fixes #694
		case errors.Is(err, catalog.ErrConflict):
			sendErrorResponseWithDetails(w, logger, http.StatusPreconditionFailed,
				ErrCodeConflict, localizer.T("errors.profile.conflict"), "")
		default:
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.updateFailed"), "") // fixes #694, #H7
		}
		return
	}

	w.Header().Set("ETag", profileETag(profile))
	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// profileETag returns the strong ETag for a profile — its row_version, the
// monotonic optimistic-concurrency token, wrapped in quotes per RFC 9110. Unlike
// the prior updated_at token it is exact, so a sub-second double-write conflicts.
func profileETag(p catalog.Profile) string {
	return `"` + strconv.FormatInt(p.RowVersion, 10) + `"`
}

// parseIfMatch returns the bare ETag value from the request's If-Match header,
// or "" when absent or "*" (i.e. unconditional). Surrounding quotes and an
// optional weak-validator prefix are stripped.
func parseIfMatch(r *http.Request) string {
	v := strings.TrimSpace(r.Header.Get("If-Match"))
	if v == "" || v == "*" {
		return ""
	}
	v = strings.TrimPrefix(v, "W/")
	return strings.Trim(v, `"`)
}

// handleDeleteProfile deletes a profile.
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request, id string) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	err := s.profiles.Delete(r.Context(), id)
	switch {
	case err == nil:
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"message": "Profile deleted successfully",
		})
	case errors.Is(err, catalog.ErrNotFound):
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
	case errors.Is(err, catalog.ErrDeleteDefault):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.profile.cannotDeleteDefault"), "") // fixes #694
	case errors.Is(err, catalog.ErrDeleteActive):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.profile.cannotDeleteActive"), "") // fixes #694
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.profile.deleteFailed"), "") // fixes #694, #H7
	}
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

	profile, err := s.profiles.ActiveProfile(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrNoActiveOrDefault):
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.noActiveOrDefault"), "") // fixes #694
		case errors.Is(err, catalog.ErrActiveNotFound):
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.activeNotFound"), "") // fixes #694
		case errors.Is(err, catalog.ErrDefaultLookup):
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.getDefaultFailed"), "") // fixes #694, #H7
		case errors.Is(err, catalog.ErrActiveLookup):
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.getActiveFailed"), "") // fixes #694, #H7
		default:
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.getFailed"), "") // fixes #694, #H7
		}
		return
	}

	w.Header().Set("ETag", profileETag(profile))
	sendJSONResponse(w, logger, http.StatusOK, profileToResponse(profile))
}

// handleSetActiveProfile sets the active profile and applies its settings.
func (s *Server) handleSetActiveProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req struct {
		ProfileID string `json:"profileId"`
	}
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	profile, err := s.profiles.SetActiveProfile(r.Context(), req.ProfileID)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrIDRequired):
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeValidation, localizer.T("errors.profile.idRequired"), "") // fixes #694
		case errors.Is(err, catalog.ErrNotFound):
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
		default:
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.setActiveFailed"), "") // fixes #694, #H7
		}
		return
	}

	// Apply profile settings to the active config (fixes #781).
	// Uses Config.ApplyProfileJSON() - single source of truth.
	if profile.ConfigJSON != "" {
		if applyErr := s.config.ApplyProfileJSON(profile.ConfigJSON); applyErr != nil {
			logger.WarnContext(r.Context(),
				"Failed to parse profile settings, using defaults",
				"error", applyErr, "profile_id", profile.ID,
			)
		} else if saveErr := s.config.Save(s.configPath); saveErr != nil {
			// fixes #782 - return error instead of silent warning
			logger.ErrorContext(r.Context(), "Failed to save config after profile switch", "error", saveErr)
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.config.failedToSave"), saveErr.Error())
			return
		} else {
			logger.InfoContext(r.Context(),
				"Applied profile settings", "profile_id", profile.ID, "profile_name", profile.Name,
			)
		}
	}

	// Broadcast profile change via SSE
	s.sseHub().Broadcast(Message{
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

	// multi_client gate: duplicating always creates a NEW profile, which by
	// definition pushes the operator over the Free/Starter 1-profile cap.
	if !s.enforceMultiClientGate(w, r) {
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

	// Extract profile ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	id := strings.TrimSuffix(path, "/duplicate")

	// Parse optional new name from request body
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	duplicate, err := s.profiles.Duplicate(r.Context(), id, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrNotFound):
			sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
				ErrCodeNotFound, localizer.T("errors.profile.notFound"), "") // fixes #694
		case errors.Is(err, catalog.ErrNameExists):
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, localizer.T("errors.profile.nameExists"), "") // fixes #694
		default:
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.profile.duplicateFailed"), "") // fixes #694, #H7
		}
		return
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

	// multi_client gate: any import that would land >1 profile total requires
	// Pro. We gate at entry so partial imports don't leave the operator in an
	// inconsistent state.
	if !s.enforceMultiClientGate(w, r) {
		return
	}

	var req ProfileImportRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	items := make([]catalog.ImportItem, 0, len(req.Profiles))
	for _, p := range req.Profiles {
		items = append(items, catalog.ImportItem{
			Name:        p.Name,
			Description: p.Description,
			ConfigJSON:  string(p.Config),
		})
	}

	res := s.profiles.Import(r.Context(), items, req.Overwrite)
	sendJSONResponse(w, logger, http.StatusOK, ProfileImportResponse{
		Created: res.Created,
		Updated: res.Updated,
		Skipped: res.Skipped,
		Errors:  res.Errors,
	})
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

	profiles, err := s.profiles.List(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to list profiles", "error", err)
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

// enforceMultiClientGate returns true if the request may proceed and writes a
// 402 + returns false when the operator has reached the Free/Starter 1-profile
// cap and lacks the `multi_client` feature. The first profile (bootstrap) is
// always allowed so a fresh install is functional on any tier; this gate fires
// only when creating a 2nd or later profile.
//
// MSP positioning: customer-facing copy always calls these "client profiles."
// The internal feature key stays `multi_client`.
func (s *Server) enforceMultiClientGate(w http.ResponseWriter, r *http.Request) bool {
	logger := logging.FromContext(r.Context())

	if s.db() == nil {
		// No DB → no profile count to compare; fall through. The downstream
		// handler returns a service-unavailable error.
		return true
	}

	count, err := s.profiles.Count(r.Context())
	if err != nil {
		logger.WarnContext(r.Context(), "multi_client gate: failed to count profiles", "error", err)
		// Fail open — refusing to gate is safer than blocking the operator when
		// the DB hiccups. CI catches it as a real error.
		return true
	}
	if count < 1 {
		// First-ever profile is always free.
		return true
	}

	// 2nd+ profile. Check the license.
	mgr := s.services.Auth.License
	if mgr == nil {
		// License disabled (dev / test builds) — permit.
		return true
	}
	if mgr.HasFeature("multi_client") {
		return true
	}

	localizer := i18n.FromRequest(r)
	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusPaymentRequired,
		"TIER_TOO_LOW",
		localizer.T("errors.profile.multiClientRequired"),
		"",
	)
	return false
}

// profileToResponse converts a use-case profile to an API response.
func profileToResponse(p catalog.Profile) ProfileResponse {
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

// profileInterfaceShape is the just-enough projection of a profile's ConfigJSON
// we need to count interfaces for the multi_interface gate.
type profileInterfaceShape struct {
	Interface struct {
		Default  string   `json:"default"`
		WiFi     string   `json:"wifi"`
		Ethernet []string `json:"ethernet"`
		WiFiList []string `json:"wifiList"`
	} `json:"interface"`
}

// enforceMultiInterfaceGate returns true if the request may proceed and writes a
// 402 + returns false when the profile's interface block exceeds 1 ethernet + 1
// wifi without the `multi_interface` feature.
func (s *Server) enforceMultiInterfaceGate(w http.ResponseWriter, r *http.Request, rawConfig json.RawMessage) bool {
	logger := logging.FromContext(r.Context())

	if len(rawConfig) == 0 {
		return true
	}

	var shape profileInterfaceShape
	if err := json.Unmarshal(rawConfig, &shape); err != nil {
		// Malformed JSON is the caller's problem to surface; let the downstream
		// validation reject it. We must not gate on parse failure or we'd lock
		// operators out of fixing bad configs.
		logger.DebugContext(r.Context(), "multi_interface gate: skipping due to config parse error", "error", err)
		return true
	}

	ifc := shape.Interface
	ethCount := countNonEmpty(ifc.Default, ifc.Ethernet)
	wifiCount := countNonEmpty(ifc.WiFi, ifc.WiFiList)

	if ethCount <= 1 && wifiCount <= 1 {
		return true
	}

	mgr := s.services.Auth.License
	if mgr == nil {
		return true
	}
	if mgr.HasFeature("multi_interface") {
		return true
	}

	localizer := i18n.FromRequest(r)
	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusPaymentRequired,
		"TIER_TOO_LOW",
		localizer.T("errors.profile.multiInterfaceRequired"),
		"",
	)
	return false
}

// countNonEmpty counts how many distinct non-empty interface names are present
// in (primary, extras). Empty strings and duplicates of the primary are skipped.
func countNonEmpty(primary string, extras []string) int {
	seen := make(map[string]struct{}, len(extras)+1)
	if primary != "" {
		seen[primary] = struct{}{}
	}
	for _, name := range extras {
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	return len(seen)
}
