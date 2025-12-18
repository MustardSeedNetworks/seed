// Package oauth provides OAuth2 authentication for SSO providers.
package oauth

import (
	"fmt"

	"golang.org/x/oauth2"
)

// Microsoft OAuth2 configuration constants.
const (
	// MicrosoftUserInfoURL is the Microsoft Graph API endpoint for user info.
	MicrosoftUserInfoURL = "https://graph.microsoft.com/v1.0/me"

	// MicrosoftTenantCommon allows any Microsoft account (personal + work).
	MicrosoftTenantCommon = "common"

	// MicrosoftTenantOrganizations allows only work/school accounts.
	MicrosoftTenantOrganizations = "organizations"

	// MicrosoftTenantConsumers allows only personal Microsoft accounts.
	MicrosoftTenantConsumers = "consumers"
)

// MicrosoftDefaultScopes are the default scopes requested for Microsoft OAuth.
var MicrosoftDefaultScopes = []string{
	"openid",
	"email",
	"profile",
	"User.Read",
}

// MicrosoftEndpoint returns the OAuth2 endpoint for Microsoft/Azure AD.
// The tenantID can be:
// - "common" - Any Microsoft account (personal + work)
// - "organizations" - Work/school accounts only
// - "consumers" - Personal accounts only
// - A specific tenant ID for single-tenant apps.
func MicrosoftEndpoint(tenantID string) oauth2.Endpoint {
	if tenantID == "" {
		tenantID = MicrosoftTenantCommon
	}

	return oauth2.Endpoint{
		AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID),
		TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
	}
}

// NewMicrosoftProvider creates a new Microsoft/Azure AD OAuth2 provider.
//
// To set up Microsoft OAuth:
// 1. Go to https://portal.azure.com/
// 2. Navigate to Azure Active Directory > App registrations
// 3. Click "New registration"
// 4. Set a name and choose the supported account types (tenant)
// 5. Add your redirect URI (e.g., https://your-domain/api/sso/callback)
// 6. Go to "Certificates & secrets" and create a new client secret
// 7. Copy the Application (client) ID and the client secret value
//
// The tenantID parameter determines which accounts can sign in:
// - "common" - Any Microsoft account
// - "organizations" - Work/school accounts only
// - "consumers" - Personal Microsoft accounts only
// - Specific tenant ID - Only accounts from that Azure AD tenant.
func NewMicrosoftProvider(clientID, clientSecret, redirectURL, tenantID string, scopes []string) *Provider {
	if len(scopes) == 0 {
		scopes = MicrosoftDefaultScopes
	}

	return &Provider{
		Name: "microsoft",
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
			Endpoint:     MicrosoftEndpoint(tenantID),
		},
		UserInfoURL: MicrosoftUserInfoURL,
	}
}
