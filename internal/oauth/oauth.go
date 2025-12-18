// Package oauth provides OAuth2 authentication for SSO providers.
//
// Supports Google, Microsoft (Azure AD), and GitHub as identity providers.
// Each provider fetches user info (email, name) after successful authentication
// to create or update users in the system.
//
// Usage:
//
//	provider := oauth.NewGoogleProvider(clientID, clientSecret, redirectURL)
//	authURL := provider.GetAuthURL(state)
//	// Redirect user to authURL
//	// After callback:
//	token, _ := provider.Exchange(ctx, code)
//	userInfo, _ := provider.GetUserInfo(ctx, token)
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Common errors returned by OAuth providers.
var (
	ErrInvalidProvider = errors.New("invalid OAuth provider")
	ErrInvalidState    = errors.New("invalid state parameter")
	ErrNoEmail         = errors.New("no email returned from provider")
	ErrTokenExchange   = errors.New("failed to exchange authorization code")
	ErrUserInfo        = errors.New("failed to get user info from provider")
)

// Provider represents an OAuth2 identity provider.
type Provider struct {
	Name        string
	Config      *oauth2.Config
	UserInfoURL string
}

// UserInfo contains user information retrieved from an OAuth provider.
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	Provider      string `json:"provider"`
}

// Manager manages multiple OAuth providers.
type Manager struct {
	providers map[string]*Provider
}

// NewManager creates a new OAuth manager.
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]*Provider),
	}
}

// RegisterProvider adds a provider to the manager.
func (m *Manager) RegisterProvider(name string, provider *Provider) {
	m.providers[strings.ToLower(name)] = provider
}

// GetProvider returns a provider by name.
func (m *Manager) GetProvider(name string) (*Provider, error) {
	provider, ok := m.providers[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, name)
	}
	return provider, nil
}

// ListProviders returns the names of all registered providers.
func (m *Manager) ListProviders() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// GetAuthURL returns the OAuth2 authorization URL for user redirect.
func (p *Provider) GetAuthURL(state string) string {
	return p.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges an authorization code for an access token.
func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := p.Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenExchange, err)
	}
	return token, nil
}

// GetUserInfo fetches user information from the provider's userinfo endpoint.
func (p *Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.Config.Client(ctx, token)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfo, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // Best effort read for error message
		return nil, fmt.Errorf("%w: status %d: %s", ErrUserInfo, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfo, err)
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfo, err)
	}

	userInfo.Provider = p.Name

	if userInfo.Email == "" {
		return nil, ErrNoEmail
	}

	return &userInfo, nil
}

// GenerateState creates a cryptographically secure random state string.
// This is used for CSRF protection in the OAuth flow.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateState checks if the state parameter matches the expected value.
func ValidateState(expected, actual string) error {
	if expected == "" || actual == "" {
		return ErrInvalidState
	}
	if expected != actual {
		return ErrInvalidState
	}
	return nil
}
