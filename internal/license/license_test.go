// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/license"
)

func TestTierString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tier license.Tier
		want string
	}{
		{license.TierFree, "Free"},
		{license.TierStarter, "Starter"},
		{license.TierPro, "Pro"},
		{license.TierInvalid, "Invalid"},
	}
	for _, c := range cases {
		if got := c.tier.String(); got != c.want {
			t.Errorf("Tier(%d).String() = %q, want %q", c.tier, got, c.want)
		}
	}
}

func TestGenerateAndValidateRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []struct {
		product string
		tier    license.Tier
	}{
		{"4001", license.TierStarter},
		{"4002", license.TierPro},
	}
	for _, c := range cases {
		key, err := license.GenerateLicenseKey(c.product, "ABCDEFG", c.tier)
		if err != nil {
			t.Fatalf("GenerateLicenseKey(%s, %v): %v", c.product, c.tier, err)
		}
		info := license.ValidateLicenseKey(key)
		if !info.Valid {
			t.Errorf("ValidateLicenseKey(%q): not valid (err=%q)", key, info.ErrorMsg)
			continue
		}
		if info.ProductCode != c.product {
			t.Errorf("ProductCode = %q, want %q", info.ProductCode, c.product)
		}
		if info.Tier != c.tier {
			t.Errorf("Tier = %v, want %v", info.Tier, c.tier)
		}
	}
}

func TestValidateRejectsBadInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"too short", "AAAA-BBBB"},
		{"too long", "AAAA-BBBB-CCCC-DDDD-EEEE"},
		{"non-alphanumeric", "AAAA-BBBB-CCCC-D!DD"},
		{"bad checksum", "AAAA-BBBB-CCCC-DDDD"},
		{"unknown product code", mustMakeKey(t, "9999", "TESTKEY", license.TierStarter)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			info := license.ValidateLicenseKey(c.key)
			if info.Valid {
				t.Errorf("expected invalid, got valid (key=%q)", c.key)
			}
			if info.ErrorMsg == "" {
				t.Errorf("expected non-empty ErrorMsg")
			}
		})
	}
}

// mustMakeKey is a test helper that calls GenerateLicenseKey and fails
// the test on error.
func mustMakeKey(t *testing.T, product, serial string, tier license.Tier) string {
	t.Helper()
	k, err := license.GenerateLicenseKey(product, serial, tier)
	if err != nil {
		t.Fatalf("mustMakeKey(%q, %q, %v): %v", product, serial, tier, err)
	}
	return k
}

func TestFormatKey(t *testing.T) {
	t.Parallel()
	got := license.FormatKey("MN0725AGLLVZP9GH")
	want := "MN07-25AG-LLVZ-P9GH"
	if got != want {
		t.Errorf("FormatKey = %q, want %q", got, want)
	}
}

// TestKeygenContract pins the cross-tool cipher contract. Every key in
// this table was produced by the canonical keygen tool
// (msn-internal-tools/keygen) and MUST validate identically in every
// product's license package (stem, seed, niac). If this test fails the
// rotor cipher has drifted from keygen — DO NOT "fix" the assertions;
// fix the cipher, or regenerate keygen + update all three products in
// lockstep.
//
// Anchored to keygen v2.2.0 (2026-05-26) — multi_interface moved
// Starter→Pro; multi_user added on Pro; multi_site renamed to
// multi_client on Pro; sso added on Pro.
func TestKeygenContract(t *testing.T) {
	t.Parallel()
	vectors := []keygenVector{
		{
			name:    "seed-starter / serial SEEDSTR",
			key:     "Q207-20AG-LLZR-C2C8",
			tier:    license.TierStarter,
			product: "4001",
			serial:  "SEEDSTR",
			features: []string{
				"monitoring_scheduled",
				"wifi_visibility_basic",
				"compliance_basic",
				"export_csv_json",
			},
		},
		{
			name:    "seed-pro / serial SEEDPRO",
			key:     "MN07-25AG-LLVZ-P9GH",
			tier:    license.TierPro,
			product: "4002",
			serial:  "SEEDPRO",
			features: []string{
				"monitoring_scheduled", "wifi_visibility_basic",
				"compliance_basic", "export_csv_json",
				"wifi_roam_analysis", "wifi_association_forensics",
				"airmapper_baseline_diff", "anomaly_detection", "path_analysis",
				"live_telemetry", "compliance_advanced", "scheduled_reports",
				"audit_pdf", "multi_interface", "multi_user", "multi_client",
				"sso", "white_label", "rest_api",
			},
		},
	}

	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			assertKeygenVector(t, v)
		})
	}
}

// keygenVector is a (key, expected-validation) pair produced by the
// canonical keygen tool.
type keygenVector struct {
	name     string
	key      string
	tier     license.Tier
	product  string
	serial   string
	features []string
}

func assertKeygenVector(t *testing.T, v keygenVector) {
	t.Helper()
	info := license.ValidateLicenseKey(v.key)
	if !info.Valid {
		t.Fatalf("Valid = false, want true (err=%q)", info.ErrorMsg)
	}
	if info.Tier != v.tier {
		t.Errorf("Tier = %v, want %v", info.Tier, v.tier)
	}
	if info.ProductCode != v.product {
		t.Errorf("ProductCode = %q, want %q", info.ProductCode, v.product)
	}
	if info.Serial != v.serial {
		t.Errorf("Serial = %q, want %q", info.Serial, v.serial)
	}
	if len(info.Features) != len(v.features) {
		t.Errorf("Features count = %d, want %d (got %v)", len(info.Features), len(v.features), info.Features)
	}
	for _, f := range v.features {
		if !info.HasFeature(f) {
			t.Errorf("missing feature %q (got %v)", f, info.Features)
		}
	}
}

func TestActivationLifecycle(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}

	// No state yet.
	if mgr.IsActivated() {
		t.Error("expected !IsActivated on fresh manager")
	}

	// Start trial.
	trial := mgr.StartTrial()
	if !trial.Success || !trial.IsTrialMode || trial.Tier != license.TierPro {
		t.Errorf("StartTrial unexpected: %+v", trial)
	}
	if !mgr.IsActivated() || !mgr.IsTrialValid() {
		t.Error("expected trial to be active")
	}

	// Activate a real key.
	key, _ := license.GenerateLicenseKey("4002", "ABCDEFG", license.TierPro)
	res := mgr.Activate(key)
	if !res.Success || res.Tier != license.TierPro {
		t.Errorf("Activate unexpected: %+v", res)
	}
	state := mgr.GetState()
	if state.IsTrialMode {
		t.Error("expected non-trial state after Activate")
	}

	// Reload from disk and re-check.
	mgr2, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("reload NewManagerWithDir: %v", err)
	}
	if !mgr2.IsActivated() {
		t.Error("expected reloaded state to be activated")
	}
	if mgr2.GetState().Tier != license.TierPro {
		t.Errorf("reloaded tier = %v, want %v", mgr2.GetState().Tier, license.TierPro)
	}

	// Deactivate.
	if deactErr := mgr2.Deactivate(); deactErr != nil {
		t.Fatalf("Deactivate: %v", deactErr)
	}
	if mgr2.IsActivated() {
		t.Error("expected !IsActivated after Deactivate")
	}
}

// TestManagerConcurrentReadsAndWrites exercises the RWMutex so `go test
// -race` fails loudly if the locking ever regresses. Per-feature gates
// in the HTTP layer call read methods on every request; activation
// can land via CLI or future portal pushes mid-flight.
func TestManagerConcurrentReadsAndWrites(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}

	key, _ := license.GenerateLicenseKey("4002", "ABCDEFG", license.TierPro)

	// Spin up a writer goroutine that toggles activation, plus several
	// reader goroutines that hammer the read API.
	done := make(chan struct{})
	go func() {
		for range 50 {
			mgr.Activate(key)
			_ = mgr.Deactivate()
		}
		close(done)
	}()

	for range 8 {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = mgr.IsActivated()
					_ = mgr.GetState()
					_ = mgr.IsTrialValid()
					_ = mgr.TrialDaysRemaining()
					_ = mgr.NeedsCheckIn()
				}
			}
		}()
	}

	<-done
}
