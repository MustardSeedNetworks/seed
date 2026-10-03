package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/license/licensetest"
)

// TestLicenseStatusNamesOnlyTheLiveTier pins seed#2704 on the CLI: a paid
// licence that has expired or belongs to another device grants Free, and
// `seed license status` must say so instead of printing the tier it held.
func TestLicenseStatusNamesOnlyTheLiveTier(t *testing.T) {
	t.Parallel()

	fp, err := fnd.GenerateFingerprint()
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	tests := []struct {
		name       string
		exp        time.Time
		deviceHash string
		want       []string
	}{
		{
			name:       "live Pro licence",
			exp:        time.Now().Add(24 * time.Hour),
			deviceHash: fp.Hash(),
			want:       []string{"Tier:        Pro\n", "Features:"},
		},
		{
			name:       "expired Pro licence",
			exp:        time.Now().Add(-time.Hour),
			deviceHash: fp.Hash(),
			want:       []string{"Tier:        Free (the Pro license is not in force)\n", "Reason:      expired "},
		},
		{
			name:       "Pro licence for another device",
			deviceHash: "another-device",
			want:       []string{"Tier:        Free (the Pro license is not in force)\n", "another device"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			writeLicenseStatus(&out, licensetest.PaidManager(t, license.TierPro, tc.exp, tc.deviceHash))
			for _, w := range tc.want {
				if !strings.Contains(out.String(), w) {
					t.Errorf("output lacks %q:\n%s", w, out.String())
				}
			}
		})
	}
}
