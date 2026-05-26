// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/license"
)

// nopHandler is a handler that records whether it was reached and
// writes a fixed 200 body. Used to verify the middleware short-circuits
// on a feature-gate failure.
func nopHandler(seen *bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		*seen = true
		w.WriteHeader(http.StatusOK)
	}
}

func TestRequireFeature_AllowsWhenLicenseDisabled(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	s.services.Auth.License = nil // simulate dev build

	called := false
	h := s.requireFeature("any_feature", nopHandler(&called))
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if !called {
		t.Error("expected handler to run when license manager is nil")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequireFeature_AllowsWhenFeaturePresent(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}

	called := false
	h := s.requireFeature("rest_api", nopHandler(&called))
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if !called {
		t.Error("expected handler to run when trial license has the feature")
	}
}

func TestRequireFeature_Blocks402WhenFeatureAbsent(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}

	called := false
	h := s.requireFeature("nonexistent_feature_xyz", nopHandler(&called))
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if called {
		t.Error("handler should not run when feature is missing")
	}
	if w.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402", w.Code)
	}

	var body FeatureGateResponse
	if decErr := json.NewDecoder(w.Body).Decode(&body); decErr != nil {
		t.Fatalf("decode response: %v", decErr)
	}
	if body.Code != errCodeTierTooLow {
		t.Errorf("Code = %q, want %q", body.Code, errCodeTierTooLow)
	}
	if body.RequiredFeature != "nonexistent_feature_xyz" {
		t.Errorf("RequiredFeature = %q, want %q", body.RequiredFeature, "nonexistent_feature_xyz")
	}
	if body.UpgradeMessage == "" {
		t.Error("expected non-empty UpgradeMessage")
	}
}

func TestRequireFeature_NoLicenseReturnsFreeTier(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	// No StartTrial / Activate — state stays nil ⇒ Free tier.

	h := s.requireFeature("scheduled_reports", nopHandler(new(bool)))
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/anything", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", w.Code)
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.CurrentTier != license.TierFree.String() {
		t.Errorf("CurrentTier = %q, want %q", body.CurrentTier, license.TierFree.String())
	}
}
