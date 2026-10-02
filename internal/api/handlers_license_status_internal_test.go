// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/license/licensetest"
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

// TestLicenseStatusReportsOnlyTheLiveGrant pins D-SEED-13 and seed#2704 on the
// wire: a spent, foreign or unbacked licence must report the Free it grants and
// why, not the tier it once held, because the UI's token and tier surfaces read
// nothing else.
func TestLicenseStatusReportsOnlyTheLiveGrant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mgr           func(t *testing.T) *license.Manager
		wantTier      string
		wantActivated bool
		wantMint      bool
		wantReason    string
	}{
		{
			name:          "live trial",
			mgr:           withState(func(t *testing.T) license.ActivationState { return trialState(t, 1) }),
			wantTier:      "Trial",
			wantActivated: true,
			wantMint:      true,
		},
		{
			name: "expired trial",
			mgr: withState(
				func(t *testing.T) license.ActivationState { return trialState(t, license.TrialDays+1) },
			),
			wantTier:   license.TierFree.String(),
			wantReason: license.ReasonTrialExpired,
		},
		{
			name: "live Pro licence",
			mgr: func(t *testing.T) *license.Manager {
				t.Helper()
				return licensetest.PaidManager(t, license.TierPro, time.Now().Add(24*time.Hour), thisDevice(t))
			},
			wantTier:      license.TierPro.String(),
			wantActivated: true,
			wantMint:      true,
		},
		{
			name: "expired Pro licence",
			mgr: func(t *testing.T) *license.Manager {
				t.Helper()
				return licensetest.PaidManager(t, license.TierPro, time.Now().Add(-time.Hour), thisDevice(t))
			},
			wantTier:   license.TierFree.String(),
			wantReason: license.ReasonExpired,
		},
		{
			name: "Pro licence activated on another device",
			mgr: func(t *testing.T) *license.Manager {
				t.Helper()
				return licensetest.PaidManager(t, license.TierPro, time.Time{}, "another-device")
			},
			wantTier:   license.TierFree.String(),
			wantReason: license.ReasonOtherDevice,
		},
		{
			name: "unbacked Pro state",
			mgr: withState(func(*testing.T) license.ActivationState {
				return license.ActivationState{
					Tier:     int(license.TierPro),
					Features: license.FeaturesForTier(license.TierPro),
				}
			}),
			wantTier:   license.TierFree.String(),
			wantReason: fnd.StatusUnverified.String(),
		},
		{
			name: "no licence is Free with no reason",
			mgr: func(t *testing.T) *license.Manager {
				t.Helper()
				mgr, err := license.NewManagerWithDir(t.TempDir())
				if err != nil {
					t.Fatalf("license manager: %v", err)
				}
				return mgr
			},
			wantTier: license.TierFree.String(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{mux: http.NewServeMux(), licenseMgr: tc.mgr(t)}
			resp := licenseStatusJSON(t, s)
			if resp.Tier != tc.wantTier || resp.Activated != tc.wantActivated ||
				resp.CanMintTokens != tc.wantMint || resp.Reason != tc.wantReason {
				t.Errorf("tier=%q activated=%t canMint=%t reason=%q, want %q %t %t %q",
					resp.Tier, resp.Activated, resp.CanMintTokens, resp.Reason,
					tc.wantTier, tc.wantActivated, tc.wantMint, tc.wantReason)
			}
			if !tc.wantActivated &&
				(resp.TierValue != int(license.TierFree) || resp.IsTrialMode || len(resp.Features) != 0) {
				t.Errorf("tierValue=%d isTrialMode=%t features=%v, want the Free grant",
					resp.TierValue, resp.IsTrialMode, resp.Features)
			}
		})
	}
}

// withState loads a manager over the state build returns, sealed on disk.
func withState(build func(t *testing.T) license.ActivationState) func(t *testing.T) *license.Manager {
	return func(t *testing.T) *license.Manager {
		t.Helper()
		return managerWithState(t, build(t))
	}
}

func thisDevice(t *testing.T) string {
	t.Helper()
	fp, err := fnd.GenerateFingerprint()
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	return fp.Hash()
}
