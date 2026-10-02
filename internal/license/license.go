// SPDX-License-Identifier: BUSL-1.1

package license

import fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

// The manager, activation state and parsed-license types are the foundation
// core's, aliased so Seed callers keep referring to them as license.Manager
// etc. Their Tier fields are plain ints on the wire; convert with license.Tier
// when a Seed tier name is needed (see Tier.String).
type (
	// Manager owns Seed's activation state, trial and device binding.
	Manager = fnd.Manager
	// ActivationState is the persisted activation snapshot GetState returns.
	ActivationState = fnd.ActivationState
	// ActivationResult is returned by Activate/StartTrial/CheckIn.
	ActivationResult = fnd.ActivationResult
	// Info is a parsed, validated license token.
	Info = fnd.Info
	// DeviceFingerprint identifies the host a license is bound to.
	DeviceFingerprint = fnd.DeviceFingerprint
)

// NewManager creates a license manager rooted at Seed's default config
// directory, verifying tokens against the embedded production key.
func NewManager() (*Manager, error) {
	return fnd.NewManager(fnd.NewProductionVerifier(Policy()), Policy())
}

// NewManagerWithDir creates a license manager that persists state in configDir.
// Used by tests to isolate activation state in a temp directory.
func NewManagerWithDir(configDir string) (*Manager, error) {
	return fnd.NewManagerWithDir(fnd.NewProductionVerifier(Policy()), Policy(), configDir)
}

// EffectiveTier is the tier m grants now. Anything short of a live grant is
// Free: no state, an expired licence or trial, another device's state, or state
// that did not load. The persisted tier records what was once granted, and an
// unverified state carries TierInvalid, which would gate out even Free.
func EffectiveTier(m *Manager) Tier {
	st := m.GetState()
	if st == nil || !m.IsActivated() {
		return TierFree
	}
	return Tier(st.Tier)
}

// FormatKey returns a signed token trimmed for display.
func FormatKey(key string) string {
	return fnd.FormatKey(key)
}
