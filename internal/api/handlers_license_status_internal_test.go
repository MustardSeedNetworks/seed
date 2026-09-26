// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/license"
)

// TestLicenseStatusCarriesFeatures pins seed#2688: the UI gates every
// <RequireFeature> surface on status.features, so the endpoint that feeds
// it must send the effective catalogue. Before the fix the field did not
// exist and Pro/Trial users saw the Free gate on Path Analysis and Reports.
func TestLicenseStatusCarriesFeatures(t *testing.T) {
	t.Parallel()

	// The strings the UI actually gates on (ui/src/pages/*.tsx). A
	// populated array with drifted keys ships the same defect.
	uiGated := []string{
		"path_analysis",
		"export_csv_json",
		"wifi_analysis",
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T) *Server
		want    []string
		wantNot []string
	}{
		{
			name: "no manager grants the Pro catalogue",
			setup: func(t *testing.T) *Server {
				t.Helper()
				// Dev / pre-install build. registerEngineIfLicensed already
				// treats a nil manager as Pro; the UI signal must agree.
				return &Server{mux: http.NewServeMux()}
			},
			want: uiGated,
		},
		{
			name: "free grants nothing",
			setup: func(t *testing.T) *Server {
				t.Helper()
				s, _ := apiTokenTestSetup(t)
				return s
			},
			wantNot: uiGated,
		},
		{
			name: "trial grants the Pro catalogue",
			setup: func(t *testing.T) *Server {
				t.Helper()
				s, mgr := apiTokenTestSetup(t)
				if res := mgr.StartTrial(); !res.Success {
					t.Fatalf("StartTrial: %s", res.Message)
				}
				return s
			},
			want: uiGated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := licenseStatusJSON(t, tc.setup(t))
			for _, f := range tc.want {
				if !slices.Contains(resp.Features, f) {
					t.Errorf("features missing %q; got %v", f, resp.Features)
				}
			}
			for _, f := range tc.wantNot {
				if slices.Contains(resp.Features, f) {
					t.Errorf("features unexpectedly grants %q; got %v", f, resp.Features)
				}
			}
		})
	}
}

// licenseStatusJSON calls the handler and decodes its body, asserting on the
// way that the "features" key exists at all: an absent key and an empty list
// both decode to a nil slice, and only one of them is the defect.
func licenseStatusJSON(t *testing.T, s *Server) LicenseStatusResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	s.handleLicenseStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/license", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["features"]; !ok {
		t.Fatalf("response has no \"features\" key: %s", rec.Body.String())
	}

	var resp LicenseStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

// TestLicenseStatusFeaturesMatchCatalogue keeps the wire answer and the
// server-side gate in lock-step: whatever HasFeature admits, the UI is told.
func TestLicenseStatusFeaturesMatchCatalogue(t *testing.T) {
	t.Parallel()

	s, mgr := apiTokenTestSetup(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}

	resp := licenseStatusJSON(t, s)
	for _, f := range license.FeaturesForTier(license.TierPro) {
		if !mgr.HasFeature(f) {
			t.Fatalf("catalogue/gate disagree on %q", f)
		}
		if !slices.Contains(resp.Features, f) {
			t.Errorf("wire answer omits %q that HasFeature admits", f)
		}
	}
}
