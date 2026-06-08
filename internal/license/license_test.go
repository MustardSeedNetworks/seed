// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/license"
)

// testSigningSeedB64 is a TEST-ONLY Ed25519 private seed, distinct from the
// production key. Tokens minted with it validate through a Verifier built on its
// public half (testVerifier) and through a Manager created with that verifier,
// but are correctly REJECTED by the embedded production key — which is what
// makes the forgery tests meaningful.
const testSigningSeedB64 = "XCx+b6yDNFoRanJhHeqX3pjXlhjNvXvAzojwaSq8lAs="

// Production-signed contract vectors (serial 1234567), produced by the keygen
// tool against the embedded production key (pre-launch keypair). They MUST
// validate via the default verifier; regenerate together with the embedded key
// if it rotates. See ADR-0019.
const (
	prodSeedStarterVector = "MSN1.eyJjb2RlIjoiNDAwMSIsImlhdCI6MTc4MDg3NjgwMCwibWF4RGV2aWNlcyI6MywicHJvZHVjdCI6InNlZWQiLCJzZXJpYWwiOiIxMjM0NTY3IiwidGllciI6MSwidiI6MX0.KEv70KrphG0Y7ATG_OPJhf4I0YJNcF7KNAVY4GPSj_Mdvxkhi4aEi6_h4Ux2EV-vkiA3lV0l_Bo7yTN9zI29CA"
	prodSeedProVector     = "MSN1.eyJjb2RlIjoiNDAwMiIsImlhdCI6MTc4MDg3NjgwMCwibWF4RGV2aWNlcyI6MywicHJvZHVjdCI6InNlZWQiLCJzZXJpYWwiOiIxMjM0NTY3IiwidGllciI6MiwidiI6MX0.wGtw4OLbVFHE19Zqt7ZK4_10P6sbmvdwa0pjoY_9U0ggR2w_Ix5Sy8KvIB3p4uO62p8tIhMon6hj_T60pK4VDA"
)

func testSigningKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	seed, err := base64.StdEncoding.DecodeString(testSigningSeedB64)
	if err != nil {
		t.Fatalf("decode test seed: %v", err)
	}
	return ed25519.NewKeyFromSeed(seed)
}

// testVerifier returns a Verifier for the test signing key. Tokens from
// signTestKey validate against it.
func testVerifier(t *testing.T) *license.Verifier {
	t.Helper()
	pub := testSigningKey(t).Public().(ed25519.PublicKey)
	return license.NewVerifier(pub)
}

// signLicenseToken mints an MSN1 token signed by priv. It mirrors the keygen /
// verifier wire format so tests can produce arbitrary tokens (including ones
// signed by an attacker key for forgery tests, or for another product) without
// the production private key.
func signLicenseToken(
	t *testing.T,
	priv ed25519.PrivateKey,
	product, code, serial string,
	tier license.Tier,
	exp int64,
) string {
	t.Helper()
	payload := map[string]any{
		"v":          1,
		"product":    product,
		"code":       code,
		"serial":     serial,
		"tier":       int(tier),
		"maxDevices": 3,
		"iat":        time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC).Unix(),
	}
	if exp > 0 {
		payload["exp"] = exp
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	sig := ed25519.Sign(priv, b)
	return "MSN1." + base64.RawURLEncoding.EncodeToString(b) +
		"." + base64.RawURLEncoding.EncodeToString(sig)
}

// signTestKey replaces the removed license.GenerateLicenseKey: it mints a
// production-shaped seed token signed by the TEST key. It validates through
// testVerifier and through a Manager built with
// NewManagerWithVerifier(dir, testVerifier(t)).
func signTestKey(t *testing.T, code, serial string, tier license.Tier) string {
	t.Helper()
	return signLicenseToken(t, testSigningKey(t), "seed", code, serial, tier, 0)
}

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

func TestSignAndValidateRoundTrip(t *testing.T) {
	t.Parallel()
	v := testVerifier(t)
	cases := []struct {
		product string
		tier    license.Tier
	}{
		{"4001", license.TierStarter},
		{"4002", license.TierPro},
	}
	for _, c := range cases {
		key := signTestKey(t, c.product, "ABCDEFG", c.tier)
		info := v.Validate(key)
		if !info.Valid {
			t.Errorf("Validate(%q): not valid (err=%q)", key, info.ErrorMsg)
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
	v := testVerifier(t)
	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"not a token", "AAAA-BBBB-CCCC-DDDD"},
		{"wrong scheme", "MSN9.abc.def"},
		{"too few parts", "MSN1.abc"},
		{"garbage payload", "MSN1.!!!.???"},
		{"unknown product code", mustMakeKey(t, "9999", "TESTKEY", license.TierStarter)},
		{"code/tier mismatch", mustMakeKey(t, "4001", "TESTKEY", license.TierPro)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			info := v.Validate(c.key)
			if info.Valid {
				t.Errorf("expected invalid, got valid (key=%q)", c.key)
			}
			if info.ErrorMsg == "" {
				t.Errorf("expected non-empty ErrorMsg")
			}
		})
	}
}

// mustMakeKey is a test helper that calls signTestKey and fails the test on
// error.
func mustMakeKey(t *testing.T, product, serial string, tier license.Tier) string {
	t.Helper()
	return signTestKey(t, product, serial, tier)
}

func TestFormatKey(t *testing.T) {
	t.Parallel()
	// Tokens are single-line and copy/paste ready; FormatKey only trims
	// surrounding whitespace (it must NOT strip base64url '-'/'_').
	got := license.FormatKey("  " + prodSeedProVector + "\n")
	if got != prodSeedProVector {
		t.Errorf("FormatKey = %q, want %q", got, prodSeedProVector)
	}
}

// TestKeygenContract pins the cross-tool signing contract. Each token in this
// table was produced by the canonical keygen tool (msn-internal-tools/keygen)
// signing with the production private key, and MUST validate against the
// embedded production public key in every product. If this test fails the
// embedded key has drifted from keygen — DO NOT "fix" the assertions; rotate
// the embedded key and regenerate these vectors in lockstep (see ADR-0019).
func TestKeygenContract(t *testing.T) {
	t.Parallel()
	vectors := []keygenVector{
		{
			name:     "seed-starter / serial 1234567",
			key:      prodSeedStarterVector,
			tier:     license.TierStarter,
			product:  "4001",
			serial:   "1234567",
			features: starterFeatureList(),
		},
		{
			name:     "seed-pro / serial 1234567",
			key:      prodSeedProVector,
			tier:     license.TierPro,
			product:  "4002",
			serial:   "1234567",
			features: proFeatureList(),
		},
	}

	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			assertKeygenVector(t, v)
		})
	}
}

// starterFeatureList mirrors validator.go's starterFeatures(); kept here so the
// contract test pins the externally observable feature set.
func starterFeatureList() []string {
	return []string{
		"monitoring_scheduled", "wifi_visibility_basic",
		"compliance_basic", "export_csv_json",
		"topology_local", "dns_monitoring", "ssl_cert_monitoring",
	}
}

// proFeatureList mirrors validator.go's proFeatures() (Starter set + Pro
// additions).
func proFeatureList() []string {
	return append(starterFeatureList(),
		"wifi_roam_analysis", "wifi_association_forensics", "airmapper_baseline_diff",
		"anomaly_detection", "path_analysis", "live_telemetry",
		"compliance_advanced", "scheduled_reports", "audit_pdf",
		"multi_interface", "multi_user", "multi_client",
		"sso", "white_label", "rest_api",
		"topology_estate", "estate_polling", "microburst_detection",
		"voip_mos_scoring", "server_monitoring", "extended_retention",
		"bgp_monitoring", "wifi_management_capture", "wifi_rogue_detection",
		"config_backup_diff", "netflow_collection",
	)
}

// keygenVector is a (token, expected-validation) pair produced by the canonical
// keygen tool against the production key.
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
	// Validates against the EMBEDDED PRODUCTION key — the whole point of the
	// contract test.
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

// TestProductMismatchRejected verifies a correctly signed token for another
// product (stem) is rejected by seed's validator even when signed by the same
// key seed trusts.
func TestProductMismatchRejected(t *testing.T) {
	t.Parallel()
	v := testVerifier(t)
	token := signLicenseToken(t, testSigningKey(t), "stem", "2001", "1234567", license.TierPro, 0)
	info := v.Validate(token)
	if info.Valid {
		t.Error("a token issued for product 'stem' must not validate in seed")
	}
	if info.ErrorMsg != license.ErrLicenseInvalid {
		t.Errorf("ErrorMsg = %q, want %q", info.ErrorMsg, license.ErrLicenseInvalid)
	}
}

// TestForgeryRejected verifies a token signed by an attacker key is rejected by
// the embedded production verifier (ValidateLicenseKey).
func TestForgeryRejected(t *testing.T) {
	t.Parallel()
	_, attackerPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}
	forged := signLicenseToken(t, attackerPriv, "seed", "4002", "1234567", license.TierPro, 0)
	info := license.ValidateLicenseKey(forged)
	if info.Valid {
		t.Error("token signed by a non-production key must be rejected")
	}
	if info.ErrorMsg != license.ErrLicenseInvalid {
		t.Errorf("forged token ErrorMsg = %q, want %q", info.ErrorMsg, license.ErrLicenseInvalid)
	}
}

// TestTamperRejected verifies that altering a signed token's payload (flipping
// Starter→Pro) invalidates the signature.
func TestTamperRejected(t *testing.T) {
	t.Parallel()
	v := testVerifier(t)
	starter := signTestKey(t, "4001", "1234567", license.TierStarter)
	// Swap the payload segment for a Pro payload while keeping the Starter
	// signature: the signature no longer matches.
	pro := signTestKey(t, "4002", "1234567", license.TierPro)
	tampered := spliceProPayload(t, starter, pro)
	info := v.Validate(tampered)
	if info.Valid {
		t.Error("a token with a swapped payload must fail signature verification")
	}
}

// spliceProPayload builds a token from the Pro payload segment and the Starter
// signature segment, producing a deliberately invalid token.
func spliceProPayload(t *testing.T, starter, pro string) string {
	t.Helper()
	sParts := splitToken(t, starter)
	pParts := splitToken(t, pro)
	return pParts[0] + "." + pParts[1] + "." + sParts[2]
}

func splitToken(t *testing.T, token string) [3]string {
	t.Helper()
	var out [3]string
	n := 0
	start := 0
	for i := 0; i < len(token) && n < 3; i++ {
		if token[i] == '.' {
			out[n] = token[start:i]
			n++
			start = i + 1
		}
	}
	if n != 2 {
		t.Fatalf("token does not have 3 parts: %q", token)
	}
	out[2] = token[start:]
	return out
}

// TestExpiredTokenRejected verifies a signed token whose exp is in the past is
// rejected with the expiry message.
func TestExpiredTokenRejected(t *testing.T) {
	t.Parallel()
	v := testVerifier(t)
	expired := signLicenseToken(t, testSigningKey(t), "seed", "4002", "1234567", license.TierPro,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	info := v.Validate(expired)
	if info.Valid {
		t.Error("expired token should not be valid")
	}
	if info.ErrorMsg != "License has expired" {
		t.Errorf("ErrorMsg = %q, want %q", info.ErrorMsg, "License has expired")
	}
}

func TestActivationLifecycle(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithVerifier(tmp, testVerifier(t))
	if err != nil {
		t.Fatalf("NewManagerWithVerifier: %v", err)
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

	// Activate a real (test-signed) key.
	key := signTestKey(t, "4002", "ABCDEFG", license.TierPro)
	res := mgr.Activate(key)
	if !res.Success || res.Tier != license.TierPro {
		t.Errorf("Activate unexpected: %+v", res)
	}
	state := mgr.GetState()
	if state.IsTrialMode {
		t.Error("expected non-trial state after Activate")
	}

	// Reload from disk and re-check.
	mgr2, err := license.NewManagerWithVerifier(tmp, testVerifier(t))
	if err != nil {
		t.Fatalf("reload NewManagerWithVerifier: %v", err)
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

// TestManagerConcurrentReadsAndWrites exercises the RWMutex so `go test -race`
// fails loudly if the locking ever regresses. Per-feature gates in the HTTP
// layer call read methods on every request; activation can land via CLI or
// future portal pushes mid-flight.
func TestManagerConcurrentReadsAndWrites(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithVerifier(tmp, testVerifier(t))
	if err != nil {
		t.Fatalf("NewManagerWithVerifier: %v", err)
	}

	key := signTestKey(t, "4002", "ABCDEFG", license.TierPro)

	// Spin up a writer goroutine that toggles activation, plus several reader
	// goroutines that hammer the read API.
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
