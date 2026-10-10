package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestMiddlewareTrustsOnlyTheResolvedPATMarker pins the shape of the #2450 fix
// at the JWT middleware: a request skips JWT validation only when the PAT
// middleware resolved its token and marked the context. A bearer that merely
// looks like a PAT is not enough — the prefix is caller-controlled.
func TestMiddlewareTrustsOnlyTheResolvedPATMarker(t *testing.T) {
	t.Parallel()
	m := auth.NewManager("", time.Hour, "", "")
	t.Cleanup(m.Stop)

	reached := false
	handler := m.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	request := func(marked bool) *http.Request {
		r := httptest.NewRequest(http.MethodPut, "/api/v1/settings", http.NoBody)
		r.Header.Set("Authorization", "Bearer sd_pat_deadbeef")
		if marked {
			r = r.WithContext(auth.WithAPITokenAuth(r.Context()))
		}
		return r
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(false))
	if reached || rec.Code != http.StatusUnauthorized {
		t.Errorf("unmarked PAT-shaped bearer: reached=%t status=%d, want refused with 401", reached, rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, request(true))
	if !reached {
		t.Errorf("a PAT-authenticated request was refused: %d %s", rec.Code, rec.Body.String())
	}
}

// TestMFAPendingTokenIsSinglePurpose: the pending token names its user, and an
// access token is not accepted in its place.
func TestMFAPendingTokenIsSinglePurpose(t *testing.T) {
	t.Parallel()
	m := auth.NewManager("", time.Hour, "", "")
	t.Cleanup(m.Stop)

	pending, err := m.GenerateMFAPendingToken(t.Context(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if user, validateErr := m.ValidateMFAPendingToken(pending); validateErr != nil || user != "alice" {
		t.Errorf("ValidateMFAPendingToken(pending) = %q, %v; want alice", user, validateErr)
	}

	access, err := m.GenerateToken(t.Context(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{"access token": access, "empty": "", "garbage": "a.b.c"} {
		if _, validateErr := m.ValidateMFAPendingToken(token); !errors.Is(validateErr, auth.ErrInvalidMFAToken) {
			t.Errorf("ValidateMFAPendingToken(%s) error = %v, want ErrInvalidMFAToken", name, validateErr)
		}
	}
	if _, emptyErr := m.GenerateMFAPendingToken(t.Context(), ""); !errors.Is(emptyErr, auth.ErrInvalidMFAToken) {
		t.Errorf("GenerateMFAPendingToken(\"\") error = %v, want ErrInvalidMFAToken", emptyErr)
	}
}
