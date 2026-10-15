package api

// oauth_settings.go holds the SSO provider settings endpoints.

import (
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// GetSSOProviderConfig returns the configuration for a specific SSO provider.
func GetSSOProviderConfig(cfg *config.Config, name string) *config.SSOProviderConfig {
	for i := range cfg.Auth.SSO.Providers {
		if strings.EqualFold(cfg.Auth.SSO.Providers[i].Name, name) {
			return &cfg.Auth.SSO.Providers[i]
		}
	}
	return nil
}

// SSOProviderInfo is one provider's stored settings as the settings page
// edits them. The secret itself never leaves the server; HasClientSecret
// tells the page whether one is stored, and an update that leaves the secret
// empty keeps it.
type SSOProviderInfo struct {
	Name            string   `json:"name"`
	Enabled         bool     `json:"enabled"`
	ClientID        string   `json:"clientId,omitempty"`
	HasClientSecret bool     `json:"hasClientSecret,omitempty"`
	RedirectURL     string   `json:"redirectUrl,omitempty"`
	TenantID        string   `json:"tenantId,omitempty"`
	Scopes          []string `json:"scopes,omitempty"`
}

// SSOSettingsResponse lists each configured SSO provider and whether it can
// be used, for the settings UI.
type SSOSettingsResponse struct {
	Providers []SSOProviderInfo `json:"providers"`
}

// handleSSOSettings returns SSO configuration status for the settings UI.
//
// The route is registered with minRole: op, and since #2632 it no longer sits
// under an auth-bypass prefix, so the JWT middleware and the role gate have
// both run by the time this is called. The hand-rolled token check that used
// to stand in for them (#757) is gone with the bypass that made it necessary.
func (s *Server) handleSSOSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	providers := make([]SSOProviderInfo, 0, len(s.config.Auth.SSO.Providers))
	for _, p := range s.config.Auth.SSO.Providers {
		providers = append(providers, SSOProviderInfo{
			Name:            p.Name,
			Enabled:         p.Enabled && p.ClientID != "",
			ClientID:        p.ClientID,
			HasClientSecret: p.ClientSecret != "",
			RedirectURL:     p.RedirectURL,
			TenantID:        p.TenantID,
			Scopes:          p.Scopes,
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, SSOSettingsResponse{Providers: providers})
}

// SSOUpdateRequest represents the SSO provider update request.
type SSOUpdateRequest struct {
	Provider     string   `json:"provider"`
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURL  string   `json:"redirectUrl"`
	TenantID     string   `json:"tenantId,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// updateProviderConfig updates the provider configuration in the config.
// An empty ClientSecret keeps the stored one: the settings page is never sent
// the secret, so it cannot send it back (#3280).
// Returns true if the provider was found and updated.
func (s *Server) updateProviderConfig(req *SSOUpdateRequest) bool {
	for i := range s.config.Auth.SSO.Providers {
		if !strings.EqualFold(s.config.Auth.SSO.Providers[i].Name, req.Provider) {
			continue
		}
		s.config.Auth.SSO.Providers[i].Enabled = req.Enabled
		s.config.Auth.SSO.Providers[i].ClientID = req.ClientID
		if req.ClientSecret != "" {
			s.config.Auth.SSO.Providers[i].ClientSecret = req.ClientSecret
		}
		s.config.Auth.SSO.Providers[i].RedirectURL = req.RedirectURL
		s.config.Auth.SSO.Providers[i].TenantID = req.TenantID
		s.config.Auth.SSO.Providers[i].Scopes = req.Scopes
		return true
	}
	return false
}

// handleSSOUpdate updates SSO provider configuration.
//
// Registered with minRole: op and feature "sso"; since #2632 the route passes
// through the JWT, CSRF and role middlewares like every other operator write,
// which is what authenticates it (#757). Body limit and config locking (#760)
// stay here.
func (s *Server) handleSSOUpdate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	// Existing handler was English-only — use the non-localized helper.
	var req SSOUpdateRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	// Lock config during update, unlock before Save() to avoid deadlock (fixes #760, #783)
	s.config.Lock()
	found := s.updateProviderConfig(&req)
	s.config.Unlock()

	if !found {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			"Provider not found",
			"",
		)
		return
	}

	if err := s.config.Save(s.configPath); err != nil {
		logger.ErrorContext(r.Context(), "Failed to save SSO config", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Failed to save configuration",
			"",
		)
		return
	}

	s.initOAuthManager()
	logger.InfoContext(r.Context(),
		"SSO provider updated",
		"provider",
		req.Provider,
		"enabled",
		req.Enabled,
		"event",
		"config.sso.updated",
	)
	sendJSONResponse(w, logger, http.StatusOK, StatusUpdatedResponse{Status: "updated"})
}
