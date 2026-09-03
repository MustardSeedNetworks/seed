// SPDX-License-Identifier: BUSL-1.1

package api

// middleware_license.go provides per-route license-tier gating. Wrap any
// handler that exposes a paid feature with Server.requireFeature so the
// route returns 402 Payment Required when the active license doesn't
// cover the feature.

import (
	"encoding/json"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// errCodeTierTooLow is the standard error code for routes blocked by
// the license tier. UIs key off this code to render the upgrade hint
// (vs. a generic auth failure).
const errCodeTierTooLow = "TIER_TOO_LOW"

// FeatureGateResponse is the JSON body returned with a 402 Payment
// Required when a request hits a feature it isn't licensed for. The
// UI's <RequireFeature> primitive matches on this shape to decide
// what upgrade hint to render.
type FeatureGateResponse struct {
	Error           string `json:"error"`
	Code            string `json:"code"`
	RequiredFeature string `json:"requiredFeature"`
	CurrentTier     string `json:"currentTier"`
	UpgradeMessage  string `json:"upgradeMessage"`
}

// requireFeature wraps `next` so it only runs when the active license
// includes `feature`. Returns 402 Payment Required with a structured
// upgrade hint otherwise.
//
// A nil license manager is treated as "license disabled" (developer
// builds, tests with no manager wired) and permits all features — so
// the gating layer never breaks local development workflows.
func (s *Server) requireFeature(feature string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.hasFeature(feature) {
			next(w, r)

			return
		}

		s.sendFeatureGate(w, r, feature)
	}
}

// hasFeature reports whether the active license includes `feature`.
//
// A nil license manager is treated as "license disabled" (developer builds,
// tests with no manager wired) and permits everything, matching requireFeature.
func (s *Server) hasFeature(feature string) bool {
	mgr := s.licenseManager()

	return mgr == nil || mgr.HasFeature(feature)
}

// sendFeatureGate writes the 402 a gated surface answers with.
//
// Split out of requireFeature because not every entitlement boundary is a whole
// route: a report is Pro because of the format it asks for, not because of the
// path it was posted to, so the handler has to be able to answer the same way
// the middleware would.
func (s *Server) sendFeatureGate(w http.ResponseWriter, r *http.Request, feature string) {
	// Resolve current tier for the response body. State may be nil
	// (no license activated) — present that as "Free".
	tierName := license.TierFree.String()
	if mgr := s.licenseManager(); mgr != nil {
		if st := mgr.GetState(); st != nil {
			tierName = license.Tier(st.Tier).String()
		}
	}

	resp := FeatureGateResponse{
		Error:           "Feature requires a higher tier",
		Code:            errCodeTierTooLow,
		RequiredFeature: feature,
		CurrentTier:     tierName,
		UpgradeMessage: "Start a 14-day Pro trial with `seed license trial` " +
			"or activate a Pro key with `seed license activate -k <KEY>`.",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"failed to encode feature-gate response", "error", encErr)
	}
}
