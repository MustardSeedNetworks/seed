package api

// profile_license_gates.go holds the licence gates a profile write passes:
// multiple clients and multiple interfaces per profile.

import (
	"encoding/json"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// enforceMultiClientGate returns true if the request may proceed and writes a
// 402 + returns false when the operator has reached the Free/Starter 1-profile
// cap and lacks the `multi_client` feature. The first profile (bootstrap) is
// always allowed so a fresh install is functional on any tier; this gate fires
// only when creating a 2nd or later profile.
//
// MSP positioning: customer-facing copy always calls these "client profiles."
// The internal feature key stays `multi_client`.
func (s *Server) enforceMultiClientGate(w http.ResponseWriter, r *http.Request) bool {
	logger := logging.FromContext(r.Context())

	if !s.profiles.Available() {
		// No DB → no profile count to compare; fall through. The downstream
		// handler returns a service-unavailable error.
		return true
	}

	count, err := s.profiles.Count(r.Context())
	if err != nil {
		logger.WarnContext(r.Context(), "multi_client gate: failed to count profiles", "error", err)
		// Fail open — refusing to gate is safer than blocking the operator when
		// the DB hiccups. CI catches it as a real error.
		return true
	}
	if count < 1 {
		// First-ever profile is always free.
		return true
	}

	// 2nd+ profile. Check the license.
	mgr := s.licenseManager()
	if mgr == nil {
		// License disabled (dev / test builds) — permit.
		return true
	}
	if mgr.HasFeature("multi_client") {
		return true
	}

	localizer := i18n.FromRequest(r)
	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusPaymentRequired,
		"TIER_TOO_LOW",
		localizer.T("errors.profile.multiClientRequired"),
		"",
	)
	return false
}

// profileInterfaceShape is the just-enough projection of a profile's ConfigJSON
// we need to count interfaces for the multi_interface gate.
type profileInterfaceShape struct {
	Interface struct {
		Default  string   `json:"default"`
		WiFi     string   `json:"wifi"`
		Ethernet []string `json:"ethernet"`
		WiFiList []string `json:"wifiList"`
	} `json:"interface"`
}

// enforceMultiInterfaceGate returns true if the request may proceed and writes a
// 402 + returns false when the profile's interface block exceeds 1 ethernet + 1
// wifi without the `multi_interface` feature.
func (s *Server) enforceMultiInterfaceGate(w http.ResponseWriter, r *http.Request, rawConfig json.RawMessage) bool {
	logger := logging.FromContext(r.Context())

	if len(rawConfig) == 0 {
		return true
	}

	var shape profileInterfaceShape
	if err := json.Unmarshal(rawConfig, &shape); err != nil {
		// Malformed JSON is the caller's problem to surface; let the downstream
		// validation reject it. We must not gate on parse failure or we'd lock
		// operators out of fixing bad configs.
		logger.DebugContext(r.Context(), "multi_interface gate: skipping due to config parse error", "error", err)
		return true
	}

	ifc := shape.Interface
	ethCount := countNonEmpty(ifc.Default, ifc.Ethernet)
	wifiCount := countNonEmpty(ifc.WiFi, ifc.WiFiList)

	if ethCount <= 1 && wifiCount <= 1 {
		return true
	}

	mgr := s.licenseManager()
	if mgr == nil {
		return true
	}
	if mgr.HasFeature("multi_interface") {
		return true
	}

	localizer := i18n.FromRequest(r)
	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusPaymentRequired,
		"TIER_TOO_LOW",
		localizer.T("errors.profile.multiInterfaceRequired"),
		"",
	)
	return false
}

// countNonEmpty counts how many distinct non-empty interface names are present
// in (primary, extras). Empty strings and duplicates of the primary are skipped.
func countNonEmpty(primary string, extras []string) int {
	seen := make(map[string]struct{}, len(extras)+1)
	if primary != "" {
		seen[primary] = struct{}{}
	}
	for _, name := range extras {
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	return len(seen)
}
