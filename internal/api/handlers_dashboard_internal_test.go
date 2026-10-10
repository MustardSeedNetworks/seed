// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const myDashboardPath = APIVersionPrefix + "/users/me/dashboard"

func serveDashboard(t *testing.T, s *Server, req *http.Request) (int, DashboardLayout, ErrorResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	var layout DashboardLayout
	var errResp ErrorResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &layout); err != nil {
			t.Fatalf("decode layout: %v; body=%s", err, w.Body.String())
		}
	} else if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	}
	return w.Code, layout, errResp
}

// A layout belongs to the caller: a viewer saves their own through the full
// middleware chain, reads it back in order, and another user still sees no
// saved layout.
func TestMyDashboard_PerUser(t *testing.T) {
	s := ssoGateServer(t)

	code, layout, _ := serveDashboard(t, s, authedRequest(t, s, http.MethodGet, myDashboardPath, "viewer", ""))
	if code != http.StatusOK || layout.Saved || layout.Widgets == nil || len(layout.Widgets) != 0 {
		t.Fatalf("GET before save = %d %+v, want 200 with an unsaved, empty (non-null) list", code, layout)
	}

	code, layout, _ = serveDashboard(t, s, authedRequest(t, s, http.MethodPut, myDashboardPath, "viewer",
		`{"widgets":["gateway","link","dns"]}`))
	if code != http.StatusOK || !layout.Saved {
		t.Fatalf("viewer PUT = %d %+v, want 200 saved", code, layout)
	}

	code, layout, _ = serveDashboard(t, s, authedRequest(t, s, http.MethodGet, myDashboardPath, "viewer", ""))
	if code != http.StatusOK || !layout.Saved || strings.Join(layout.Widgets, ",") != "gateway,link,dns" {
		t.Errorf("viewer GET = %d %+v, want the saved order", code, layout)
	}

	code, layout, _ = serveDashboard(t, s, authedRequest(t, s, http.MethodGet, myDashboardPath, "admin", ""))
	if code != http.StatusOK || layout.Saved || len(layout.Widgets) != 0 {
		t.Errorf("admin GET = %d %+v, want the viewer's save to be invisible", code, layout)
	}
}

func TestMyDashboard_Refusals(t *testing.T) {
	s := ssoGateServer(t)

	noCSRF := authedRequest(t, s, http.MethodPut, myDashboardPath, "viewer", `{"widgets":["link"]}`)
	noCSRF.Header.Del("X-Csrf-Token")

	anonymous := httptest.NewRequest(http.MethodGet, myDashboardPath, http.NoBody)

	tests := []struct {
		name        string
		req         *http.Request
		wantCode    int
		wantDetails string
	}{
		{"PUT without a CSRF token", noCSRF, http.StatusForbidden, ""},
		{"no credentials", anonymous, http.StatusUnauthorized, ""},
		{"undeclared method", authedRequest(t, s, http.MethodPost, myDashboardPath, "viewer",
			`{"widgets":["link"]}`), http.StatusMethodNotAllowed, ""},
		{"widgets left out", authedRequest(t, s, http.MethodPut, myDashboardPath, "viewer",
			`{}`), http.StatusBadRequest, ""},
		{"unknown field", authedRequest(t, s, http.MethodPut, myDashboardPath, "viewer",
			`{"widgets":["link"],"owner":"admin"}`), http.StatusBadRequest, ""},
		{"repeated widget", authedRequest(t, s, http.MethodPut, myDashboardPath, "viewer",
			`{"widgets":["link","link"]}`), http.StatusBadRequest, "widget 2 repeats widget 1"},
		{"malformed id", authedRequest(t, s, http.MethodPut, myDashboardPath, "viewer",
			`{"widgets":["<b>"]}`), http.StatusBadRequest, "is not a widget name"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, _, errResp := serveDashboard(t, s, tc.req)
			if code != tc.wantCode {
				t.Fatalf("status = %d, want %d", code, tc.wantCode)
			}
			if tc.wantDetails != "" && !strings.Contains(errResp.Details, tc.wantDetails) {
				t.Errorf("details = %q, want it to contain %q", errResp.Details, tc.wantDetails)
			}
		})
	}

	// Nothing above was stored.
	_, layout, _ := serveDashboard(t, s, authedRequest(t, s, http.MethodGet, myDashboardPath, "viewer", ""))
	if layout.Saved {
		t.Errorf("a refused PUT saved a layout: %+v", layout)
	}
}
