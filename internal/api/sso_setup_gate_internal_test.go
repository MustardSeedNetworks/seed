// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/oauth"
)

// ssoCallback drives login then callback through a cookie jar against a server
// whose only SSO user is the fake IdP's, returning the callback response and
// whether the browser ended up holding an access cookie.
func ssoCallback(t *testing.T, s *Server) (*http.Response, bool) {
	t.Helper()
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
	authURL, err := url.Parse(login.Header.Get("Location"))
	if err != nil {
		t.Fatalf("login Location: %v", err)
	}
	callback := browserGet(t, browser, daemon.URL+"/api/v1/sso/callback?"+
		url.Values{"code": {"auth-code"}, "state": {authURL.Query().Get("state")}}.Encode())

	daemonURL, err := url.Parse(daemon.URL)
	if err != nil {
		t.Fatalf("daemon URL: %v", err)
	}
	signedIn := slices.ContainsFunc(jar.Cookies(daemonURL), func(c *http.Cookie) bool {
		return c.Name == auth.CookieNameAccess
	})
	return callback, signedIn
}

func TestSSOLoginRefusedBeforeSetup(t *testing.T) {
	s := ssoGateServer(t)
	s.config.Lock()
	s.config.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	s.config.Unlock()

	callback, signedIn := ssoCallback(t, s)

	if signedIn {
		t.Error("SSO signed in before setup completed")
	}
	if loc := callback.Header.Get("Location"); !strings.Contains(strings.ToLower(loc), "complete+setup+first") &&
		!strings.Contains(strings.ToLower(loc), "complete%20setup%20first") {
		t.Errorf("callback redirected to %q, want the complete-setup-first error", loc)
	}
	if _, err := s.identityUsers.Get(t.Context(), "google:1001"); err == nil {
		t.Error("SSO login before setup created a user row")
	}
}

func TestSSOLoginAfterSetupIsViewer(t *testing.T) {
	s := ssoGateServer(t)

	callback, signedIn := ssoCallback(t, s)

	if !signedIn || callback.Header.Get("Location") != "/" {
		t.Fatalf("SSO after setup: signedIn=%v location=%q", signedIn, callback.Header.Get("Location"))
	}
	u, err := s.identityUsers.Get(t.Context(), "google:1001")
	if err != nil {
		t.Fatalf("get SSO user: %v", err)
	}
	if u.Role != roles.Viewer {
		t.Errorf("SSO user role = %q, want viewer", u.Role)
	}
}
