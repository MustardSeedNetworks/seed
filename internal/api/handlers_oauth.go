package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/oauth"
)

// OAuth constants.
const (
	// oauthStateCookie is the name of the cookie used to store CSRF state.
	oauthStateCookie = "oauth_state"

	// oauthStateCookieExpiry is the expiration time for the OAuth state cookie.
	oauthStateCookieExpiry = 10 * time.Minute

	// oauthExchangeTimeoutSec is the timeout in seconds for OAuth token exchange operations.
	oauthExchangeTimeoutSec = 30

	// urlLogPrefixLen bounds the URL prefix length included in error logs
	// when an OAuth redirect URL fails the allowlist check.
	urlLogPrefixLen = 64
)

// SSOProvidersResponse lists enabled SSO providers.
type SSOProvidersResponse struct {
	Providers []string `json:"providers"`
}

// initOAuthManager creates and configures the OAuth manager from config.
func (s *Server) initOAuthManager() {
	s.services.Auth.OAuth = oauth.NewManager()

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

		s.oauthManager().RegisterProvider(providerConfig.Name, provider)
	}
}

// handleSSOProviders returns the list of enabled SSO providers.
func (s *Server) handleSSOProviders(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
		return
	}

	providers := s.oauthManager().ListProviders()
	sendJSONResponse(w, logger, http.StatusOK, SSOProvidersResponse{
		Providers: providers,
	})
}

// handleSSOLogin initiates OAuth flow by redirecting to the provider.
func (s *Server) handleSSOLogin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
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

	// Get provider from query parameter
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Missing provider parameter",
			"",
		)
		return
	}

	// Get the OAuth provider
	provider, err := s.oauthManager().GetProvider(providerName)
	if err != nil {
		logger.WarnContext(r.Context(), "Invalid SSO provider requested",
			"provider", providerName,
			"client_ip", s.getClientIP(r),
			"error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			fmt.Sprintf("Invalid provider: %s", providerName),
			"",
		)
		return
	}

	// Generate CSRF state token
	state, err := oauth.GenerateState()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate OAuth state", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Failed to initiate OAuth",
			"",
		)
		return
	}

	// Store state in secure cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/api/sso",
		MaxAge:   int(oauthStateCookieExpiry.Seconds()),
		HttpOnly: true,
		Secure:   true, // HTTPS required at daemon level
		SameSite: http.SameSiteLaxMode,
	})

	// Also store provider in a cookie so callback knows which provider to use
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/api/sso",
		MaxAge:   int(oauthStateCookieExpiry.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Security audit log
	logger.InfoContext(r.Context(), "SSO login initiated",
		"provider", providerName,
		"client_ip", s.getClientIP(r),
		"event", "auth.sso.initiated")

	// Redirect to OAuth provider. provider.GetAuthURL() builds the URL using
	// oauth2.Config.AuthCodeURL — the base AuthURL is the provider's hard-
	// configured endpoint (Google, GitHub, etc.) set at provider construction
	// from internal config, NOT from the request. The state query param is
	// CSRF nonce material. Defense-in-depth: validate the produced URL still
	// starts with the configured provider AuthURL before redirecting.
	authURL := provider.GetAuthURL(state)
	if !strings.HasPrefix(authURL, provider.Config.Endpoint.AuthURL) {
		logger.ErrorContext(
			r.Context(),
			"OAuth URL host mismatch",
			"url_prefix",
			authURL[:min(urlLogPrefixLen, len(authURL))],
		)
		http.Error(w, "invalid OAuth redirect", http.StatusInternalServerError)
		return
	}
	//nolint:gosec // G710: authURL is verified to start with provider.Config.Endpoint.AuthURL (hardcoded at provider construction) above; gosec taint analysis can't follow the prefix check
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// oauthCallbackParams holds validated OAuth callback parameters.
type oauthCallbackParams struct {
	provider     *oauth.Provider
	providerName string
	code         string
}

// validateOAuthCallback validates the OAuth callback request and returns the provider and code.
func (s *Server) validateOAuthCallback(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	clientIP string,
) (*oauthCallbackParams, bool) {
	// Check for OAuth error response from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		logger.WarnContext(r.Context(), "OAuth provider returned error",
			"error", errParam,
			"description", errDesc,
			"client_ip", clientIP,
			"event", "auth.sso.provider_error")
		s.redirectWithError(w, r, fmt.Sprintf("OAuth error: %s", errDesc))
		return nil, false
	}

	// Get and validate state from cookie
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil {
		logger.WarnContext(r.Context(),
			"Missing OAuth state cookie",
			"client_ip",
			clientIP,
			"event",
			"auth.sso.missing_state",
		)
		s.redirectWithError(w, r, "OAuth session expired. Please try again.")
		return nil, false
	}

	stateParam := r.URL.Query().Get("state")
	if validateErr := oauth.ValidateState(stateCookie.Value, stateParam); validateErr != nil {
		logger.WarnContext(r.Context(), "Invalid OAuth state", "client_ip", clientIP, "event", "auth.sso.invalid_state")
		s.redirectWithError(w, r, "Invalid OAuth state. Please try again.")
		return nil, false
	}

	// Get provider from cookie
	providerCookie, err := r.Cookie("oauth_provider")
	if err != nil {
		logger.WarnContext(r.Context(),
			"Missing OAuth provider cookie",
			"client_ip",
			clientIP,
			"event",
			"auth.sso.missing_provider",
		)
		s.redirectWithError(w, r, "OAuth session expired. Please try again.")
		return nil, false
	}

	providerName := providerCookie.Value
	provider, err := s.oauthManager().GetProvider(providerName)
	if err != nil {
		logger.ErrorContext(r.Context(), "Invalid provider in callback", "provider", providerName, "error", err)
		s.redirectWithError(w, r, "Invalid OAuth provider.")
		return nil, false
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		logger.WarnContext(r.Context(), "Missing authorization code",
			"provider", providerName,
			"client_ip", clientIP,
			"event", "auth.sso.missing_code")
		s.redirectWithError(w, r, "Missing authorization code.")
		return nil, false
	}

	return &oauthCallbackParams{provider: provider, providerName: providerName, code: code}, true
}

// exchangeCodeForUserInfo exchanges the OAuth code for a token and retrieves user info.
func (s *Server) exchangeCodeForUserInfo(
	ctx context.Context,
	params *oauthCallbackParams,
	logger *slog.Logger,
	clientIP string,
) (*oauth.UserInfo, error) {
	token, err := params.provider.Exchange(ctx, params.code)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to exchange OAuth code",
			"provider", params.providerName,
			"client_ip", clientIP,
			"event", "auth.sso.exchange_failed",
			"error", err)
		return nil, errors.New("failed to authenticate")
	}

	var userInfo *oauth.UserInfo
	if params.providerName == "github" {
		userInfo, err = oauth.GetGitHubUserInfo(ctx, params.provider.Config, token)
	} else {
		userInfo, err = params.provider.GetUserInfo(ctx, token)
	}

	if err != nil {
		logger.ErrorContext(ctx, "Failed to get user info",
			"provider", params.providerName,
			"client_ip", clientIP,
			"event", "auth.sso.userinfo_failed",
			"error", err)
		return nil, errors.New("failed to get user information")
	}

	return userInfo, nil
}

// completeOAuthLogin generates tokens, sets cookies, and redirects.
//
// Per seed#1198: upserts the IdP user into the `users` table BEFORE
// issuing session tokens so multi_user CRUD (#1191) can see and manage
// SSO-authenticated identities. Tokens are issued against the canonical
// `username` (synthetic, "<provider>:<external_id>") rather than the
// raw email so a local-auth account and an SSO account sharing an
// email never collide.
func (s *Server) completeOAuthLogin(
	w http.ResponseWriter,
	r *http.Request,
	userInfo *oauth.UserInfo,
	providerName string,
	logger *slog.Logger,
) bool {
	db := s.services.Database.DB
	if db == nil {
		logger.ErrorContext(r.Context(), "SSO callback: database unavailable for user upsert",
			"provider", providerName, "email", userInfo.Email)
		s.redirectWithError(w, r, "User store unavailable.")
		return false
	}

	user, err := db.UpsertSSOUser(r.Context(), database.SSOUserInput{
		Provider:    providerName,
		ExternalID:  userInfo.ID,
		Email:       userInfo.Email,
		DisplayName: userInfo.Name,
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "SSO callback: UpsertSSOUser failed",
			"provider", providerName, "email", userInfo.Email, "error", err)
		s.redirectWithError(w, r, "Could not create your user record. Contact your administrator.")
		return false
	}

	accessToken, err := s.authManager().GenerateAccessToken(r.Context(), user.Username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate access token",
			"provider", providerName,
			"username", user.Username,
			"error", err)
		s.redirectWithError(w, r, "Failed to create session.")
		return false
	}

	refreshToken, err := s.authManager().GenerateRefreshToken(r.Context(), user.Username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate refresh token",
			"provider", providerName,
			"username", user.Username,
			"error", err)
		s.redirectWithError(w, r, "Failed to create session.")
		return false
	}

	s.clearOAuthCookies(w)

	cookieConfig := auth.DefaultCookieConfig()
	auth.SetAccessTokenCookie(w, accessToken, cookieConfig)
	auth.SetRefreshTokenCookie(w, refreshToken, cookieConfig)

	logger.InfoContext(r.Context(), "SSO user record synced",
		"provider", providerName,
		"username", user.Username,
		"role", user.Role,
		"event", "auth.sso.user_synced")

	return true
}

// handleSSOCallback handles the OAuth callback from the provider.
func (s *Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	clientIP := s.getClientIP(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "",
		)
		return
	}

	// Validate callback and get OAuth parameters
	params, ok := s.validateOAuthCallback(w, r, logger, clientIP)
	if !ok {
		return
	}

	// Exchange code for token and get user info
	ctx, cancel := context.WithTimeout(r.Context(), oauthExchangeTimeoutSec*time.Second)
	defer cancel()

	userInfo, err := s.exchangeCodeForUserInfo(ctx, params, logger, clientIP)
	if err != nil {
		s.redirectWithError(w, r, err.Error())
		return
	}

	// Security audit log: successful SSO authentication
	logger.InfoContext(r.Context(), "SSO authentication successful",
		"provider", params.providerName,
		"email", userInfo.Email,
		"client_ip", clientIP,
		"event", "auth.sso.success")

	// Generate tokens and set cookies
	if !s.completeOAuthLogin(w, r, userInfo, params.providerName, logger) {
		return
	}

	// Redirect to frontend
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// redirectWithError redirects to the frontend with an error message.
func (s *Server) redirectWithError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	s.clearOAuthCookies(w)
	// Always redirect to the root of OUR app — never an attacker-controlled
	// URL. errorMsg is properly URL-encoded as a query param value.
	target := (&url.URL{
		Path:     "/",
		RawQuery: url.Values{"sso_error": []string{errorMsg}}.Encode(),
	}).String()
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

// clearOAuthCookies removes the OAuth state cookies.
func (s *Server) clearOAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/api/sso",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    "",
		Path:     "/api/sso",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
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
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
		return
	}

	// Security: Require authentication (fixes #757)
	token, _ := auth.GetTokenFromRequest(r)
	if token == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			"Authentication required",
			"",
		)
		return
	}
	if _, err := s.authManager().ValidateToken(r.Context(), token); err != nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			"Invalid or expired token",
			"",
		)
		return
	}

	providers := make([]SSOProviderInfo, 0, len(s.config.Auth.SSO.Providers))
	for _, p := range s.config.Auth.SSO.Providers {
		providers = append(providers, SSOProviderInfo{
			Name:    p.Name,
			Enabled: p.Enabled && p.ClientID != "",
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"providers": providers,
	})
}

// ssoUpdateRequest represents the SSO provider update request.
type ssoUpdateRequest struct {
	Provider     string   `json:"provider"`
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	TenantID     string   `json:"tenant_id,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// requireSSOAuth validates authentication for SSO update operations.
// Returns true if authentication is valid, false otherwise (response already sent).
func (s *Server) requireSSOAuth(w http.ResponseWriter, r *http.Request, logger *slog.Logger) bool {
	token, _ := auth.GetTokenFromRequest(r)
	if token == "" {
		clientIP := s.getClientIP(r)
		logger.WarnContext(r.Context(),
			"Unauthenticated SSO update attempt",
			"client_ip",
			clientIP,
			"event",
			"auth.sso.blocked",
		)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			"Authentication required",
			"",
		)
		return false
	}
	if _, err := s.authManager().ValidateToken(r.Context(), token); err != nil {
		clientIP := s.getClientIP(r)
		logger.WarnContext(r.Context(),
			"Invalid token SSO update attempt",
			"client_ip",
			clientIP,
			"event",
			"auth.sso.blocked",
		)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			"Invalid or expired token",
			"",
		)
		return false
	}
	return true
}

// updateProviderConfig updates the provider configuration in the config.
// Returns true if the provider was found and updated.
func (s *Server) updateProviderConfig(req *ssoUpdateRequest) bool {
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
		return true
	}
	return false
}

// handleSSOUpdate updates SSO provider configuration.
// Security fix #757, #760: Require authentication and add body limit + config locking.
func (s *Server) handleSSOUpdate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method != http.MethodPut {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		)
		return
	}

	// Security: Require authentication (fixes #757)
	if !s.requireSSOAuth(w, r, logger) {
		return
	}

	// Existing handler was English-only — use the non-localized helper.
	var req ssoUpdateRequest
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
	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "updated"})
}
