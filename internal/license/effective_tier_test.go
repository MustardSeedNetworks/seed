// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/license/licensetest"
)

// TestStateOnDiskFailsClosed pins D-SEED-13: only a fresh install may start a
// trial, and a licence file that is damaged, forged or spent grants Free and
// survives the trial attempt untouched.
func TestStateOnDiskFailsClosed(t *testing.T) {
	t.Parallel()

	fp, fpErr := fnd.GenerateFingerprint()
	if fpErr != nil {
		t.Fatalf("fingerprint: %v", fpErr)
	}
	trialStartedDaysAgo := func(days int) license.ActivationState {
		return license.ActivationState{
			DeviceHash:     fp.Hash(),
			Tier:           int(license.TierPro),
			TrialStartedAt: time.Now().AddDate(0, 0, -days),
			IsTrialMode:    true,
		}
	}

	tests := []struct {
		name       string
		write      func(t *testing.T, dir string)
		wantStatus fnd.LoadStatus
		wantTier   license.Tier
		wantTrial  bool
	}{
		{
			name:       "absent state is a fresh install and may start the trial",
			write:      func(*testing.T, string) {},
			wantStatus: fnd.StatusMissing,
			wantTier:   license.TierFree,
			wantTrial:  true,
		},
		{
			name: "unreadable state is Free and keeps its file",
			write: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(licensetest.Path(dir), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: fnd.StatusUnreadable,
			wantTier:   license.TierFree,
		},
		{
			name: "malformed state is Free and keeps its file",
			write: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(licensetest.Path(dir), []byte("not a licence"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: fnd.StatusMalformed,
			wantTier:   license.TierFree,
		},
		{
			name: "a Pro state no signature backs is Free, not below it",
			write: func(t *testing.T, dir string) {
				t.Helper()
				licensetest.WriteState(t, dir, license.ActivationState{
					DeviceHash: fp.Hash(),
					Tier:       int(license.TierPro),
					Features:   license.FeaturesForTier(license.TierPro),
				})
			},
			wantStatus: fnd.StatusUnverified,
			wantTier:   license.TierFree,
		},
		{
			name: "an expired trial is Free and cannot start another",
			write: func(t *testing.T, dir string) {
				t.Helper()
				licensetest.WriteState(t, dir, trialStartedDaysAgo(license.TrialDays+1))
			},
			wantStatus: fnd.StatusLoaded,
			wantTier:   license.TierFree,
		},
		{
			name: "a live trial grants Pro",
			write: func(t *testing.T, dir string) {
				t.Helper()
				licensetest.WriteState(t, dir, trialStartedDaysAgo(1))
			},
			wantStatus: fnd.StatusLoaded,
			wantTier:   license.TierPro,
			wantTrial:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tc.write(t, dir)
			before := snapshot(t, licensetest.Path(dir))

			mgr, err := license.NewManagerWithDir(dir)
			if err != nil {
				t.Fatalf("NewManagerWithDir: %v", err)
			}
			if got := mgr.LoadStatus(); got != tc.wantStatus {
				t.Fatalf("LoadStatus = %s, want %s (%v)", got, tc.wantStatus, mgr.LoadError())
			}
			if got := license.EffectiveTier(mgr); got != tc.wantTier {
				t.Errorf("EffectiveTier = %s, want %s", got, tc.wantTier)
			}

			res := mgr.StartTrial()
			if res.Success != tc.wantTrial {
				t.Fatalf("StartTrial success = %t, want %t (%s)", res.Success, tc.wantTrial, res.Message)
			}
			if tc.wantTrial {
				if got := license.EffectiveTier(mgr); got != license.TierPro {
					t.Errorf("EffectiveTier after trial = %s, want Pro", got)
				}
				return
			}
			if after := snapshot(t, licensetest.Path(dir)); !bytes.Equal(before, after) {
				t.Error("a refused trial rewrote the licence file")
			}
			if got := license.EffectiveTier(mgr); got != license.TierFree {
				t.Errorf("EffectiveTier after refused trial = %s, want Free", got)
			}
		})
	}
}

// snapshot returns the licence file's bytes, a marker for a directory, or nil
// when nothing is there.
func snapshot(t *testing.T, path string) []byte {
	t.Helper()
	info, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		return nil
	}
	if statErr != nil {
		t.Fatalf("stat %s: %v", path, statErr)
	}
	if info.IsDir() {
		return []byte("<dir>")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read %s: %v", path, readErr)
	}
	return data
}
