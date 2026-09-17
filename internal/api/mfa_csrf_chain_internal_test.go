// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestMFAEnrolmentRequiresCSRFThroughTheRealChain pins the server half of the
// MFA-enrolment defect. The pre-session login routes are CSRF-exempt (#2391),
// but the *enrolment* routes are not and must not be: they run under a live
// session, so a cross-site page could ride the operator's cookie and enrol its
// own second factor. The existing MFA suite never sees this because
// GetAuthenticatedHandler() stops at the JWT middleware and omits CSRF
// entirely; only Handler() is the chain production serves.
func TestMFAEnrolmentRequiresCSRFThroughTheRealChain(t *testing.T) {
	enrolment := []string{
		APIVersionPrefix + "/auth/totp/setup",
		APIVersionPrefix + "/auth/totp/verify",
		APIVersionPrefix + "/auth/totp/disable",
		APIVersionPrefix + "/auth/webauthn/register/begin",
		APIVersionPrefix + "/auth/webauthn/register/finish",
	}

	for _, path := range enrolment {
		t.Run(path, func(t *testing.T) {
			s := csrfEndpointServer(t)
			bearer, err := s.authManager().GenerateAccessToken(t.Context(), "admin")
			if err != nil {
				t.Fatalf("mint access token: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer "+bearer)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("POST %s without a CSRF token = %d, want 403; the enrolment route is not protected",
					path, w.Code)
			}

			// With the token the request clears CSRF and reaches the handler,
			// so whatever it answers it is no longer a CSRF refusal.
			token := fetchCSRFToken(t, s, bearer)
			req = httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer "+bearer)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(auth.CSRFHeaderName, token)
			w = httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)

			if w.Code == http.StatusForbidden {
				t.Fatalf("POST %s with a valid CSRF token = 403: %s", path, w.Body.String())
			}
		})
	}
}
