// Package oauth provides OAuth2 authentication for SSO providers.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHub OAuth2 configuration constants.
const (
	GitHubUserInfoURL  = "https://api.github.com/user"
	GitHubUserEmailURL = "https://api.github.com/user/emails"
)

// GitHubDefaultScopes are the default scopes requested for GitHub OAuth.
var GitHubDefaultScopes = []string{
	"read:user",
	"user:email",
}

// NewGitHubProvider creates a new GitHub OAuth2 provider.
//
// To set up GitHub OAuth:
// 1. Go to https://github.com/settings/developers
// 2. Click "New OAuth App"
// 3. Set the Application name and Homepage URL
// 4. Set the Authorization callback URL (e.g., https://your-domain/api/sso/callback)
// 5. Click "Register application"
// 6. Copy the Client ID
// 7. Click "Generate a new client secret" and copy the secret.
func NewGitHubProvider(clientID, clientSecret, redirectURL string, scopes []string) *Provider {
	if len(scopes) == 0 {
		scopes = GitHubDefaultScopes
	}

	provider := &Provider{
		Name: "github",
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
			Endpoint:     github.Endpoint,
		},
		UserInfoURL: GitHubUserInfoURL,
	}

	return provider
}

// GitHubUserResponse represents the response from GitHub's user API.
type GitHubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubEmailResponse represents an email from GitHub's user/emails API.
type GitHubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// GetGitHubUserInfo fetches user information from GitHub.
// GitHub requires separate API calls for user info and emails.
func GetGitHubUserInfo(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*UserInfo, error) {
	client := config.Client(ctx, token)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get user info
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitHubUserInfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserInfo, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserInfo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrUserInfo, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserInfo, err)
	}

	var ghUser GitHubUserResponse
	if unmarshalErr := json.Unmarshal(body, &ghUser); unmarshalErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserInfo, unmarshalErr)
	}

	userInfo := &UserInfo{
		ID:            strconv.FormatInt(ghUser.ID, 10),
		Name:          ghUser.Name,
		Email:         ghUser.Email,
		Picture:       ghUser.AvatarURL,
		Provider:      "github",
		EmailVerified: false, // Will be updated below
	}

	// If email is not public, fetch from emails endpoint
	if userInfo.Email == "" {
		email, verified, emailErr := getGitHubPrimaryEmail(ctx, client)
		if emailErr != nil {
			return nil, emailErr
		}
		userInfo.Email = email
		userInfo.EmailVerified = verified
	}

	if userInfo.Email == "" {
		return nil, ErrNoEmail
	}

	// Use login name if display name is not set
	if userInfo.Name == "" {
		userInfo.Name = ghUser.Login
	}

	return userInfo, nil
}

// getGitHubPrimaryEmail fetches the user's primary email from GitHub.
func getGitHubPrimaryEmail(ctx context.Context, client *http.Client) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitHubUserEmailURL, http.NoBody)
	if err != nil {
		return "", false, fmt.Errorf("%w: %w", ErrUserInfo, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("%w: %w", ErrUserInfo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("%w: status %d: %s", ErrUserInfo, resp.StatusCode, string(body))
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", false, fmt.Errorf("%w: %w", ErrUserInfo, readErr)
	}

	var emails []GitHubEmailResponse
	if unmarshalErr := json.Unmarshal(body, &emails); unmarshalErr != nil {
		return "", false, fmt.Errorf("%w: %w", ErrUserInfo, unmarshalErr)
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, true, nil
		}
	}

	// Fall back to any verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email, true, nil
		}
	}

	// Last resort: any email
	for _, email := range emails {
		return email.Email, email.Verified, nil
	}

	return "", false, nil
}
