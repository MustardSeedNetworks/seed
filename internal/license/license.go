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

// FormatKey returns a signed token trimmed for display.
func FormatKey(key string) string {
	return fnd.FormatKey(key)
}
