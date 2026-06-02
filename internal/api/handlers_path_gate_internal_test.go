// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/license"
)

// TestRootsPathRequiresPro asserts the path-analysis route returns
// 402 Payment Required for unlicensed (Free) callers and routes
// through to the handler for Pro-trial callers. Subsumes the manual
// proof that PR-B5's requireFeature wiring works end-to-end.
func TestRootsPathRequiresPro(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	// setupRoutes is normally called by NewServer; the test helper
	// skips it (the helper builds a Server directly). Invoke it here
	// so the /path/path mux entry exists.
	s.setupRoutes()

	// 1. No license → 402.
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/path/path", http.NoBody)
	req.Header.Set("X-Username", "alice")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("free tier: status = %d, want 402; body=%s", w.Code, w.Body.String())
	}
	var body FeatureGateResponse
	if decErr := json.NewDecoder(w.Body).Decode(&body); decErr != nil {
		t.Fatalf("decode 402 body: %v", decErr)
	}
	if body.RequiredFeature != "path_analysis" {
		t.Errorf("requiredFeature = %q, want %q", body.RequiredFeature, "path_analysis")
	}
	if body.CurrentTier != license.TierFree.String() {
		t.Errorf("currentTier = %q, want %q", body.CurrentTier, license.TierFree.String())
	}

	// 2. Start trial → Pro features → no longer 402. The handler may
	//    400/500 because it expects query params we don't supply, but
	//    crucially it must NOT 402. That's what we assert.
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	req2 := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/path/path", http.NoBody)
	req2.Header.Set("X-Username", "alice")
	w2 := httptest.NewRecorder()
	s.mux.ServeHTTP(w2, req2)
	if w2.Code == http.StatusPaymentRequired {
		t.Errorf("trial license: status = 402 (should pass the gate); body=%s", w2.Body.String())
	}
}
