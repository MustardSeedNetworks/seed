// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/license"
)

// Production-signed Seed tokens (serial 1234567), produced by the canonical
// keygen tool against the embedded production public key. They MUST activate
// through Seed's product policy — this pins the cross-tool signing contract
// (product name "seed", codes 4001/4002, tier feature sets) the way stem/niac
// pin theirs. If the production key rotates, regenerate these vectors and the
// embedded key together. Generic crypto properties (forgery, tampering,
// wrong-product, expiry, bad input) are covered in foundation's pkg/license
// tests; this file only exercises Seed's product-specific wiring.
const (
	prodSeedStarterVector = "MSN1.eyJjb2RlIjoiNDAwMSIsImlhdCI6MTc4MDg3NjgwMCwibWF4RGV2aWNlcyI6MywicHJvZHVjdCI6InNlZWQiLCJzZXJpYWwiOiIxMjM0NTY3IiwidGllciI6MSwidiI6MX0.KEv70KrphG0Y7ATG_OPJhf4I0YJNcF7KNAVY4GPSj_Mdvxkhi4aEi6_h4Ux2EV-vkiA3lV0l_Bo7yTN9zI29CA"
	prodSeedProVector     = "MSN1.eyJjb2RlIjoiNDAwMiIsImlhdCI6MTc4MDg3NjgwMCwibWF4RGV2aWNlcyI6MywicHJvZHVjdCI6InNlZWQiLCJzZXJpYWwiOiIxMjM0NTY3IiwidGllciI6MiwidiI6MX0.wGtw4OLbVFHE19Zqt7ZK4_10P6sbmvdwa0pjoY_9U0ggR2w_Ix5Sy8KvIB3p4uO62p8tIhMon6hj_T60pK4VDA"
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

// TestKeygenContract pins the cross-tool signing contract end-to-end: each
// production-signed vector activates through Seed's policy and yields the
// expected tier. This catches a wrong product code, salt, or embedded key in
// Seed's policy wiring.
func TestKeygenContract(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		vector   string
		wantTier license.Tier
	}{
		{"starter", prodSeedStarterVector, license.TierStarter},
		{"pro", prodSeedProVector, license.TierPro},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			mgr, err := license.NewManagerWithDir(t.TempDir())
			if err != nil {
				t.Fatalf("NewManagerWithDir: %v", err)
			}
			res := mgr.Activate(c.vector)
			if !res.Success {
				t.Fatalf("production vector did not activate: %s", res.Message)
			}
			if res.Tier != int(c.wantTier) {
				t.Errorf("Tier = %d, want %s (%d)", res.Tier, c.wantTier, c.wantTier)
			}
		})
	}
}

// TestFeaturesForTier verifies Seed's product policy directly: Starter and Pro
// map to their catalogs and codes (Pro is a superset of Starter); Free and any
// unrecognized tier are rejected so a signed token can't grant more than this
// build knows about.
func TestFeaturesForTier(t *testing.T) {
	t.Parallel()
	p := license.Policy()

	starter, starterCode, ok := p.FeaturesForTier(int(license.TierStarter))
	if !ok {
		t.Fatal("Starter tier not recognized")
	}
	if starterCode != "4001" {
		t.Errorf("Starter code = %q, want 4001", starterCode)
	}

	pro, proCode, ok := p.FeaturesForTier(int(license.TierPro))
	if !ok {
		t.Fatal("Pro tier not recognized")
	}
	if proCode != "4002" {
		t.Errorf("Pro code = %q, want 4002", proCode)
	}
	// Pro is a strict superset of Starter.
	for _, f := range starter {
		if !slices.Contains(pro, f) {
			t.Errorf("Pro missing Starter feature %q", f)
		}
	}
	if len(pro) <= len(starter) {
		t.Errorf("Pro (%d features) should exceed Starter (%d)", len(pro), len(starter))
	}

	for _, tier := range []int{int(license.TierFree), int(license.TierInvalid), 99} {
		if _, _, recognized := p.FeaturesForTier(tier); recognized {
			t.Errorf("tier %d unexpectedly recognized", tier)
		}
	}
}
