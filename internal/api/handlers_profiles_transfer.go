package api

// handlers_profiles_transfer.go holds the profile duplicate, import and
// export handlers.

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
)

// handleDuplicateProfile creates a copy of an existing profile. The body is
// optional: without one the store names the copy.
func (s *Server) handleDuplicateProfile(w http.ResponseWriter, r *http.Request) {
	if !s.profileStoreAvailable(w, r) {
		return
	}
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// multi_client gate: duplicating always creates a NEW profile, which by
	// definition pushes the operator over the Free/Starter 1-profile cap.
	if !s.enforceMultiClientGate(w, r) {
		return
	}

	var req ProfileDuplicateRequest
	if r.Body != http.NoBody && !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	id := r.PathValue("profileId")
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

	if !s.profiles.Available() {
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

	if !s.profiles.Available() {
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
