// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSSOResaveKeepsStoredSecret pins #3280: the settings page never holds the
// stored secret, so every save after the first sent an empty one and the
// handler stored it, which dropped the provider from the OAuth manager.
func TestSSOResaveKeepsStoredSecret(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		wantSecret string
	}{
		{"empty secret keeps the stored one", "", "stored-secret"},
		{"new secret replaces it", "rotated-secret", "rotated-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ssoGateServer(t)
			s.config.Auth.SSO.Providers[0].ClientSecret = "stored-secret"

			update := SSOUpdateRequest{
				Provider:     "google",
				Enabled:      true,
				ClientID:     "original-client-id",
				ClientSecret: tt.secret,
				RedirectURL:  "https://seed.example/api/v1/sso/callback",
			}
			body, err := json.Marshal(update)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, authedRequest(t, s, http.MethodPut, "/api/v1/sso/update", "admin", string(body)))
			if w.Code != http.StatusOK {
				t.Fatalf("PUT /api/v1/sso/update = %d, want 200: %s", w.Code, w.Body.String())
			}

			if got := s.config.Auth.SSO.Providers[0].ClientSecret; got != tt.wantSecret {
				t.Errorf("stored client secret = %q, want %q", got, tt.wantSecret)
			}
			if !s.config.Auth.SSO.Providers[0].Enabled {
				t.Error("provider not enabled after the save")
			}
		})
	}
}

// TestSSOSettingsReturnsStoredFieldsButNotTheSecret is the read half of #3280:
// the page can only send back what it was given.
func TestSSOSettingsReturnsStoredFieldsButNotTheSecret(t *testing.T) {
	s := ssoGateServer(t)
	p := &s.config.Auth.SSO.Providers[0]
	p.Enabled = true
	p.ClientSecret = "stored-secret"
	p.RedirectURL = "https://seed.example/api/v1/sso/callback"
	p.Scopes = []string{"openid", "email"}

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authedRequest(t, s, http.MethodGet, "/api/v1/sso/settings", "admin", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/sso/settings = %d, want 200: %s", w.Code, w.Body.String())
	}

	want := `{"providers":[{"name":"google","enabled":true,"clientId":"original-client-id",` +
		`"hasClientSecret":true,"redirectUrl":"https://seed.example/api/v1/sso/callback",` +
		`"scopes":["openid","email"]}]}`
	require.JSONEq(t, want, w.Body.String())
}
