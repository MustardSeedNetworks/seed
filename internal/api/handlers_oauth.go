// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/oauth"
)

// State cookie name and expiration for CSRF protection.
const (
	oauthStateCookie       = "oauth_state"
	oauthStateCookieExpiry = 10 * time.Minute
)

// SSOProvidersResponse lists enabled SSO providers.
type SSOProvidersResponse struct {
	Providers []string `json:"providers"`
}

// initOAuthManager creates and configures the OAuth manager from config.
func (s *Server) initOAuthManager() {
	s.oauthManager = oauth.NewManager()

	for _, providerConfig := range s.config.Auth.SSO.Providers {
		if !providerConfig.Enabled || providerConfig.ClientID == "" {
			continue
		}

		var provider *oauth.Provider
		switch strings.ToLower(providerConfig.Name) {
		case "google":
			provider = oauth.NewGoogleProvider(
				providerConfig.ClientID,
				providerConfig.ClientSecret,
				providerConfig.RedirectURL,
				providerConfig.Scopes,
			)
		case "microsoft":
			provider = oauth.NewMicrosoftProvider(
				providerConfig.ClientID,
				providerConfig.ClientSecret,
				providerConfig.RedirectURL,
				providerConfig.TenantID,
				providerConfig.Scopes,
			)
		case "github":
			provider = oauth.NewGitHubProvider(
				providerConfig.ClientID,
				providerConfig.ClientSecret,
				providerConfig.RedirectURL,
				providerConfig.Scopes,
			)
		default:
			continue
		}

		s.oauthManager.RegisterProvider(providerConfig.Name, provider)
	}
}

// handleSSOProviders returns the list of enabled SSO providers.
func (s *Server) handleSSOProviders(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
		return
	}

	providers := s.oauthManager.ListProviders()
	sendJSONResponse(w, logger, http.StatusOK, SSOProvidersResponse{
		Providers: providers,
	})
}

// handleSSOLogin initiates OAuth flow by redirecting to the provider.
func (s *Server) handleSSOLogin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	// Get provider from query parameter
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Missing provider parameter", "")
		return
	}

	// Get the OAuth provider
	provider, err := s.oauthManager.GetProvider(providerName)
	if err != nil {
		logger.Warn("Invalid SSO provider requested",
			"provider", providerName,
			"client_ip", GetClientIP(r),
			"error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, fmt.Sprintf("Invalid provider: %s", providerName), "")
		return
	}

	// Generate CSRF state token
	state, err := oauth.GenerateState()
	if err != nil {
		logger.Error("Failed to generate OAuth state", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to initiate OAuth", "")
		return
	}

	// Store state in secure cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/api/sso",
		MaxAge:   int(oauthStateCookieExpiry.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil, // Set Secure flag if HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Also store provider in a cookie so callback knows which provider to use
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/api/sso",
		MaxAge:   int(oauthStateCookieExpiry.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Security audit log
	logger.Info("SSO login initiated",
		"provider", providerName,
		"client_ip", GetClientIP(r),
		"event", "auth.sso.initiated")

	// Redirect to OAuth provider
	authURL := provider.GetAuthURL(state)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// handleSSOCallback handles the OAuth callback from the provider.
func (s *Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	clientIP := GetClientIP(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	// Check for OAuth error response from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		logger.Warn("OAuth provider returned error",
			"error", errParam,
			"description", errDesc,
			"client_ip", clientIP,
			"event", "auth.sso.provider_error")
		s.redirectWithError(w, r, fmt.Sprintf("OAuth error: %s", errDesc))
		return
	}

	// Get state from cookie
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil {
		logger.Warn("Missing OAuth state cookie",
			"client_ip", clientIP,
			"event", "auth.sso.missing_state")
		s.redirectWithError(w, r, "OAuth session expired. Please try again.")
		return
	}

	// Validate state parameter (CSRF protection)
	stateParam := r.URL.Query().Get("state")
	if err := oauth.ValidateState(stateCookie.Value, stateParam); err != nil {
		logger.Warn("Invalid OAuth state",
			"client_ip", clientIP,
			"event", "auth.sso.invalid_state")
		s.redirectWithError(w, r, "Invalid OAuth state. Please try again.")
		return
	}

	// Get provider from cookie
	providerCookie, err := r.Cookie("oauth_provider")
	if err != nil {
		logger.Warn("Missing OAuth provider cookie",
			"client_ip", clientIP,
			"event", "auth.sso.missing_provider")
		s.redirectWithError(w, r, "OAuth session expired. Please try again.")
		return
	}

	providerName := providerCookie.Value
	provider, err := s.oauthManager.GetProvider(providerName)
	if err != nil {
		logger.Error("Invalid provider in callback", "provider", providerName, "error", err)
		s.redirectWithError(w, r, "Invalid OAuth provider.")
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		logger.Warn("Missing authorization code",
			"provider", providerName,
			"client_ip", clientIP,
			"event", "auth.sso.missing_code")
		s.redirectWithError(w, r, "Missing authorization code.")
		return
	}

	// Exchange code for token
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := provider.Exchange(ctx, code)
	if err != nil {
		logger.Error("Failed to exchange OAuth code",
			"provider", providerName,
			"client_ip", clientIP,
			"event", "auth.sso.exchange_failed",
			"error", err)
		s.redirectWithError(w, r, "Failed to authenticate. Please try again.")
		return
	}

	// Get user info from provider
	var userInfo *oauth.UserInfo
	if providerName == "github" {
		// GitHub requires special handling for emails
		userInfo, err = oauth.GetGitHubUserInfo(ctx, provider.Config, token)
	} else {
		userInfo, err = provider.GetUserInfo(ctx, token)
	}

	if err != nil {
		logger.Error("Failed to get user info",
			"provider", providerName,
			"client_ip", clientIP,
			"event", "auth.sso.userinfo_failed",
			"error", err)
		s.redirectWithError(w, r, "Failed to get user information.")
		return
	}

	// Security audit log: successful SSO authentication
	logger.Info("SSO authentication successful",
		"provider", providerName,
		"email", userInfo.Email,
		"client_ip", clientIP,
		"event", "auth.sso.success")

	// Generate access and refresh tokens (like login handler does)
	accessToken, err := s.authManager.GenerateAccessToken(userInfo.Email)
	if err != nil {
		logger.Error("Failed to generate access token",
			"provider", providerName,
			"email", userInfo.Email,
			"error", err)
		s.redirectWithError(w, r, "Failed to create session.")
		return
	}

	refreshToken, err := s.authManager.GenerateRefreshToken(userInfo.Email)
	if err != nil {
		logger.Error("Failed to generate refresh token",
			"provider", providerName,
			"email", userInfo.Email,
			"error", err)
		s.redirectWithError(w, r, "Failed to create session.")
		return
	}

	// Clear OAuth state cookies
	s.clearOAuthCookies(w, r)

	// Set httpOnly auth cookies (same as login handler)
	cookieConfig := auth.DefaultCookieConfig()
	if !s.config.Server.HTTPS {
		cookieConfig.Secure = false
	}
	auth.SetAccessTokenCookie(w, accessToken, cookieConfig)
	auth.SetRefreshTokenCookie(w, refreshToken, cookieConfig)

	// Redirect to frontend - auth cookies are set, frontend will detect authenticated state
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// redirectWithError redirects to the frontend with an error message.
func (s *Server) redirectWithError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	s.clearOAuthCookies(w, r)
	// URL-encode the error message
	encoded := strings.ReplaceAll(errorMsg, " ", "%20")
	http.Redirect(w, r, "/?sso_error="+encoded, http.StatusTemporaryRedirect)
}

// clearOAuthCookies removes the OAuth state cookies.
func (s *Server) clearOAuthCookies(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/api/sso",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    "",
		Path:     "/api/sso",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetSSOProviderConfig returns the configuration for a specific SSO provider.
func GetSSOProviderConfig(cfg *config.Config, name string) *config.SSOProviderConfig {
	for i := range cfg.Auth.SSO.Providers {
		if strings.EqualFold(cfg.Auth.SSO.Providers[i].Name, name) {
			return &cfg.Auth.SSO.Providers[i]
		}
	}
	return nil
}

// SSOProviderInfo provides public info about an SSO provider.
type SSOProviderInfo struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// handleSSOSettings returns SSO configuration status for the settings UI.
// Security fix #757: Require authentication to view SSO settings.
func (s *Server) handleSSOSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
		return
	}

	// Security: Require authentication (fixes #757)
	token, _ := auth.GetTokenFromRequest(r)
	if token == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required", "")
		return
	}
	if _, err := s.authManager.ValidateToken(token); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized, "Invalid or expired token", "")
		return
	}

	providers := make([]SSOProviderInfo, 0, len(s.config.Auth.SSO.Providers))
	for _, p := range s.config.Auth.SSO.Providers {
		providers = append(providers, SSOProviderInfo{
			Name:    p.Name,
			Enabled: p.Enabled && p.ClientID != "",
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

// handleSSOUpdate updates SSO provider configuration.
// Security fix #757, #760: Require authentication and add body limit + config locking.
func (s *Server) handleSSOUpdate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method != http.MethodPut {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
		return
	}

	// Security: Require authentication (fixes #757)
	token, _ := auth.GetTokenFromRequest(r)
	if token == "" {
		clientIP := GetClientIP(r)
		logger.Warn("Unauthenticated SSO update attempt",
			"client_ip", clientIP,
			"event", "auth.sso.blocked")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized, "Authentication required", "")
		return
	}
	if _, err := s.authManager.ValidateToken(token); err != nil {
		clientIP := GetClientIP(r)
		logger.Warn("Invalid token SSO update attempt",
			"client_ip", clientIP,
			"event", "auth.sso.blocked")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized, "Invalid or expired token", "")
		return
	}

	// Limit request body size (fixes #760)
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var req struct {
		Provider     string   `json:"provider"`
		Enabled      bool     `json:"enabled"`
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		RedirectURL  string   `json:"redirect_url"`
		TenantID     string   `json:"tenant_id,omitempty"`
		Scopes       []string `json:"scopes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
		return
	}

	// Lock config during update (fixes #760)
	s.config.Lock()
	defer s.config.Unlock()

	// Find and update the provider config
	found := false
	for i := range s.config.Auth.SSO.Providers {
		if !strings.EqualFold(s.config.Auth.SSO.Providers[i].Name, req.Provider) {
			continue
		}
		s.config.Auth.SSO.Providers[i].Enabled = req.Enabled
		s.config.Auth.SSO.Providers[i].ClientID = req.ClientID
		s.config.Auth.SSO.Providers[i].ClientSecret = req.ClientSecret
		s.config.Auth.SSO.Providers[i].RedirectURL = req.RedirectURL
		s.config.Auth.SSO.Providers[i].TenantID = req.TenantID
		s.config.Auth.SSO.Providers[i].Scopes = req.Scopes
		found = true
		break
	}

	if !found {
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound, ErrCodeNotFound, "Provider not found", "")
		return
	}

	// Save config
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save SSO config", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, "Failed to save configuration", "")
		return
	}

	// Reinitialize OAuth manager with new config
	s.initOAuthManager()

	logger.Info("SSO provider updated",
		"provider", req.Provider,
		"enabled", req.Enabled,
		"event", "config.sso.updated")

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status": "updated",
	})
}
