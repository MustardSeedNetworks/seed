// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/license"
)

// TestSSEEventsRequiresLiveTelemetry asserts the `/events` SSE route
// returns 402 Payment Required for unlicensed callers and routes
// through to handleSSE for Pro-trial callers. Mirrors the canonical
// requireFeature test pattern from TestRootsPathRequiresPro.
//
// Pre-2026-05-29 this route was ungated: Free / Starter callers
// received the Pro-tier real-time stream. This test guards against
// that regression.
func TestSSEEventsRequiresLiveTelemetry(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)
	s.setupRoutes()

	// 1. No license → 402, requiredFeature reports the right key.
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/events", http.NoBody)
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
	if body.RequiredFeature != "live_telemetry" {
		t.Errorf("requiredFeature = %q, want %q", body.RequiredFeature, "live_telemetry")
	}
	if body.CurrentTier != license.TierFree.String() {
		t.Errorf("currentTier = %q, want %q", body.CurrentTier, license.TierFree.String())
	}

	// 2. Start trial → Pro features → no longer 402. The handler may
	//    fail downstream (no auth cookie set, no flusher in the test
	//    recorder, etc.), but it must NOT 402 — that's the regression
	//    guard.
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	req2 := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/events", http.NoBody)
	req2.Header.Set("X-Username", "alice")
	w2 := httptest.NewRecorder()
	s.mux.ServeHTTP(w2, req2)
	if w2.Code == http.StatusPaymentRequired {
		t.Errorf("trial license: status = 402 (should pass the gate); body=%s", w2.Body.String())
	}
}

// TestDiscoveryEngineEventsStaysOpen guards the explicit policy
// decision in setupSSEAndStatic: `/discovery/engine/events` is a
// discovery-lifecycle stream, available on every tier. If a future
// refactor accidentally extends the `live_telemetry` gate to this
// route, this test fails.
func TestDiscoveryEngineEventsStaysOpen(t *testing.T) {
	t.Parallel()
	s, _ := apiTokenTestSetup(t)
	s.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/discovery/engine/events", http.NoBody)
	req.Header.Set("X-Username", "alice")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code == http.StatusPaymentRequired {
		t.Errorf("discovery events should NOT be license-gated; got 402: body=%s", w.Body.String())
	}
}
