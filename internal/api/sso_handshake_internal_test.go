// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"golang.org/x/oauth2"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/oauth"
)

// fakeIDP answers the token exchange and the userinfo read the way an OAuth
// provider does, for one fixed user.
func fakeIDP(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "idp-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oauth.UserInfo{ID: "1001", Email: "sso@example.com", Name: "SSO User"})
	})
	idp := httptest.NewServer(mux)
	t.Cleanup(idp.Close)
	return idp
}

// TestSSOHandshakeSignsInThroughABrowserCookieJar pins #3253: the handshake
// cookies were scoped to /api/sso while the routes are /api/v1/sso/*, so a
// browser never sent oauth_state to the callback and every SSO sign-in ended
// in "OAuth session expired". The jar here applies cookie Path and Secure the
// way a browser does, which a hand-copied Cookie header would not.
func TestSSOHandshakeSignsInThroughABrowserCookieJar(t *testing.T) {
	s := ssoGateServer(t)
	idp := fakeIDP(t)
	s.oauthManager().RegisterProvider("google", &oauth.Provider{
		Name: "google",
		Config: &oauth2.Config{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "https://seed.example/api/v1/sso/callback",
			Endpoint:     oauth2.Endpoint{AuthURL: idp.URL + "/auth", TokenURL: idp.URL + "/token"},
		},
		UserInfoURL: idp.URL + "/userinfo",
	})

	daemon := httptest.NewTLSServer(s.Handler())
	t.Cleanup(daemon.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	browser := daemon.Client()
	browser.Jar = jar
	browser.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	login := browserGet(t, browser, daemon.URL+"/api/v1/sso/login?provider=google")
	if login.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("GET /sso/login = %d, want %d", login.StatusCode, http.StatusTemporaryRedirect)
	}
	authURL, err := url.Parse(login.Header.Get("Location"))
	if err != nil {
		t.Fatalf("login Location: %v", err)
	}
	state := authURL.Query().Get("state")
	if state == "" {
		t.Fatalf("login redirected to %q with no state", authURL)
	}

	callback := browserGet(t, browser, daemon.URL+"/api/v1/sso/callback?"+
		url.Values{"code": {"auth-code"}, "state": {state}}.Encode())
	if callback.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("GET /sso/callback = %d, want %d", callback.StatusCode, http.StatusTemporaryRedirect)
	}
	if got := callback.Header.Get("Location"); got != "/" {
		t.Fatalf("callback redirected to %q, want / (signed in)", got)
	}
	daemonURL, err := url.Parse(daemon.URL)
	if err != nil {
		t.Fatalf("daemon URL: %v", err)
	}
	signedIn := slices.ContainsFunc(jar.Cookies(daemonURL), func(c *http.Cookie) bool {
		return c.Name == auth.CookieNameAccess
	})
	if !signedIn {
		t.Errorf("callback set no %s cookie: the browser is not signed in", auth.CookieNameAccess)
	}
}

func browserGet(t *testing.T, client *http.Client, target string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)
	if err != nil {
		t.Fatalf("build GET %s: %v", target, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	_ = resp.Body.Close()
	return resp
}
