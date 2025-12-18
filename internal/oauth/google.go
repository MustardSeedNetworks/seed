// Package oauth provides OAuth2 authentication for SSO providers.
package oauth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Google OAuth2 configuration constants.
const (
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// GoogleDefaultScopes are the default scopes requested for Google OAuth.
var GoogleDefaultScopes = []string{
	"openid",
	"email",
	"profile",
}

// NewGoogleProvider creates a new Google OAuth2 provider.
//
// To set up Google OAuth:
// 1. Go to https://console.cloud.google.com/
// 2. Create or select a project
// 3. Enable the "Google+ API" or "People API"
// 4. Go to Credentials > Create Credentials > OAuth Client ID
// 5. Set Application type to "Web application"
// 6. Add your redirect URI (e.g., https://your-domain/api/sso/callback)
// 7. Copy the Client ID and Client Secret.
func NewGoogleProvider(clientID, clientSecret, redirectURL string, scopes []string) *Provider {
	if len(scopes) == 0 {
		scopes = GoogleDefaultScopes
	}

	return &Provider{
		Name: "google",
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
		},
		UserInfoURL: GoogleUserInfoURL,
	}
}
