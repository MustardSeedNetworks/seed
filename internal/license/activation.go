// SPDX-License-Identifier: BUSL-1.1

// Package license provides Seed's offline license validation, device
// binding, 14-day trial, and encrypted on-disk state. Keys are issued
// by the canonical keygen tool; this package only validates them.
package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// License file lives at ~/.config/seed/.license, distinct from stem
// and niac to keep per-product activations isolated on shared hosts.
const licenseFileName = ".license"

// TrialDays is the trial-period length before a paid key is required.
const TrialDays = 14

// OfflineMaxDays is the number of days allowed offline after activation.
const OfflineMaxDays = 90

// CheckInInterval is the number of days between optional check-ins.
const CheckInInterval = 30

// encryptionSalt is product-distinct so an attacker can't reuse a
// stem/niac license file by renaming it.
const encryptionSalt = "MSN-SEED-DIAG-2026-LICENSE"

// Hours per day for time calculations.
const hoursPerDay = 24

// Days in a year for license expiration.
const daysPerYear = 365

// ActivationState represents the current license activation status.
type ActivationState struct {
	LicenseKey      string    `json:"licenseKey"`
	DeviceHash      string    `json:"deviceHash"`
	Tier            Tier      `json:"tier"`
	ActivatedAt     time.Time `json:"activatedAt"`
	LastValidatedAt time.Time `json:"lastValidatedAt"`
	ExpiresAt       time.Time `json:"expiresAt"`
	TrialStartedAt  time.Time `json:"trialStartedAt,omitzero"`
	IsTrialMode     bool      `json:"isTrialMode"`
	Features        []string  `json:"features"`
}

// ActivationResult contains the result of an activation attempt.
type ActivationResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	Tier          Tier   `json:"tier,omitempty"`
	DaysRemaining int    `json:"daysRemaining,omitempty"`
	IsTrialMode   bool   `json:"isTrialMode"`
}

// Manager handles license activation and validation.
//
// Manager is safe for concurrent use. State mutations (Activate,
// Deactivate, StartTrial, CheckIn) take a write lock; reads
// (GetState, IsActivated, HasFeature, etc.) take a read lock.
// Per-feature gates in the HTTP layer call read methods on every
// request, so contention on the write path must stay rare — in
// practice activation happens once at deploy time.
type Manager struct {
	mu          sync.RWMutex
	state       *ActivationState
	fingerprint *DeviceFingerprint
	configDir   string
	verifier    *Verifier // verifies signed license tokens; production key by default
}

// NewManager creates a new license manager rooted at the default
// per-user config directory, verifying tokens against the embedded
// production key.
func NewManager() (*Manager, error) {
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		homeDir = "/tmp"
	}
	return NewManagerWithDir(filepath.Join(homeDir, ".config", "seed"))
}

// NewManagerWithDir creates a license manager that persists state in
// the given directory and verifies tokens against the embedded
// production key. Exposed so tests can use a tmpdir without poking at
// the user's real config.
func NewManagerWithDir(configDir string) (*Manager, error) {
	return newManager(configDir, mustVerifierFromB64(licensePublicKeyB64))
}

// NewManagerWithVerifier creates a license manager rooted at configDir
// that verifies tokens against the supplied Verifier instead of the
// embedded production key. It exists so tests can activate tokens minted
// with an ephemeral key (the production private key never ships);
// production code uses [NewManager] or [NewManagerWithDir].
func NewManagerWithVerifier(configDir string, v *Verifier) (*Manager, error) {
	return newManager(configDir, v)
}

func newManager(configDir string, v *Verifier) (*Manager, error) {
	fp, fpErr := GenerateFingerprint()
	if fpErr != nil {
		return nil, fmt.Errorf("failed to generate fingerprint: %w", fpErr)
	}

	m := &Manager{
		state:       nil,
		fingerprint: fp,
		configDir:   configDir,
		verifier:    v,
	}

	// Load existing state (best-effort, non-fatal).
	_ = m.loadState()
	return m, nil
}

// GetState returns the current activation state. The returned pointer
// must not be mutated by callers; treat it as read-only. (Tests
// occasionally read fields off it, hence the pointer return rather
// than a copy.)
func (m *Manager) GetState() *ActivationState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// GetFingerprint returns the device fingerprint. The fingerprint is
// immutable after construction, so no lock is needed.
func (m *Manager) GetFingerprint() *DeviceFingerprint {
	return m.fingerprint
}

// IsActivated returns true if a valid license is active.
func (m *Manager) IsActivated() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isActivatedLocked()
}

// HasFeature reports whether the active license includes the named
// feature. Returns false when:
//   - no license is loaded
//   - the license has expired or is otherwise invalid
//   - the feature is not in the license's feature set
//
// During an active trial, the license carries Pro features regardless
// of which key (if any) was activated; HasFeature reflects that.
//
// Per-feature gates in HTTP handlers call this on every request, so
// it must be cheap: a single map-equivalent lookup under RLock. The
// canonical feature names are the strings in keygen's productCatalog
// (e.g. "wifi_roam_analysis", "scheduled_reports") — mirrored by
// starterFeatures() and proFeatures() in validator.go.
func (m *Manager) HasFeature(feature string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.isActivatedLocked() || m.state == nil {
		return false
	}
	return slices.Contains(m.state.Features, feature)
}

// isActivatedLocked assumes the caller holds at least an RLock.
func (m *Manager) isActivatedLocked() bool {
	if m.state == nil {
		return false
	}
	if m.state.IsTrialMode {
		return m.isTrialValidLocked()
	}
	if !m.state.ExpiresAt.IsZero() && time.Now().After(m.state.ExpiresAt) {
		return false
	}
	if m.state.DeviceHash != m.fingerprint.Hash() {
		return false
	}
	return true
}

// IsTrialValid returns true if trial period is still active.
func (m *Manager) IsTrialValid() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isTrialValidLocked()
}

func (m *Manager) isTrialValidLocked() bool {
	if m.state == nil || !m.state.IsTrialMode {
		return false
	}
	if m.state.TrialStartedAt.IsZero() {
		return false
	}
	trialEnd := m.state.TrialStartedAt.AddDate(0, 0, TrialDays)
	return time.Now().Before(trialEnd)
}

// TrialDaysRemaining returns days left in trial.
func (m *Manager) TrialDaysRemaining() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.trialDaysRemainingLocked()
}

func (m *Manager) trialDaysRemainingLocked() int {
	if m.state == nil || !m.state.IsTrialMode {
		return 0
	}
	if m.state.TrialStartedAt.IsZero() {
		return TrialDays
	}
	trialEnd := m.state.TrialStartedAt.AddDate(0, 0, TrialDays)
	remaining := int(time.Until(trialEnd).Hours() / hoursPerDay)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// StartTrial begins the trial period.
func (m *Manager) StartTrial() *ActivationResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isActivatedLocked() && !m.state.IsTrialMode {
		return &ActivationResult{
			Success: true,
			Message: "Already activated with full license",
			Tier:    m.state.Tier,
		}
	}

	if m.state != nil && !m.state.TrialStartedAt.IsZero() {
		remaining := m.trialDaysRemainingLocked()
		if remaining <= 0 {
			return &ActivationResult{
				Success:     false,
				Message:     "Trial period has expired. Please enter a license key.",
				Tier:        TierInvalid,
				IsTrialMode: true,
			}
		}
		return &ActivationResult{
			Success:       true,
			Message:       fmt.Sprintf("Trial active: %d days remaining", remaining),
			Tier:          TierPro, // Full Pro features during trial.
			DaysRemaining: remaining,
			IsTrialMode:   true,
		}
	}

	m.state = &ActivationState{
		LicenseKey:     "",
		DeviceHash:     m.fingerprint.Hash(),
		Tier:           TierPro, // Full Pro features during trial.
		TrialStartedAt: time.Now(),
		IsTrialMode:    true,
		Features:       proFeatures(),
	}

	if saveErr := m.saveState(); saveErr != nil {
		return &ActivationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to save trial state: %v", saveErr),
			Tier:    TierInvalid,
		}
	}

	return &ActivationResult{
		Success:       true,
		Message:       fmt.Sprintf("Trial started! %d days of full Pro access.", TrialDays),
		Tier:          TierPro,
		DaysRemaining: TrialDays,
		IsTrialMode:   true,
	}
}

// Activate attempts to activate a license key.
func (m *Manager) Activate(licenseKey string) *ActivationResult {
	// Validate the license token offline against this manager's verifier.
	info := m.verifier.Validate(licenseKey)
	if !info.Valid {
		return &ActivationResult{
			Success: false,
			Message: info.ErrorMsg,
			Tier:    TierInvalid,
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = &ActivationState{
		LicenseKey:      info.Key,
		DeviceHash:      m.fingerprint.Hash(),
		Tier:            info.Tier,
		ActivatedAt:     time.Now(),
		LastValidatedAt: time.Now(),
		ExpiresAt:       time.Now().AddDate(1, 0, 0), // 1 year from activation.
		IsTrialMode:     false,
		Features:        info.Features,
	}

	if saveErr := m.saveState(); saveErr != nil {
		return &ActivationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to save activation: %v", saveErr),
			Tier:    TierInvalid,
		}
	}

	return &ActivationResult{
		Success:       true,
		Message:       fmt.Sprintf("License activated successfully! Tier: %s", info.Tier),
		Tier:          info.Tier,
		DaysRemaining: daysPerYear,
	}
}

// Deactivate removes the current license.
func (m *Manager) Deactivate() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	licensePath := filepath.Join(m.configDir, licenseFileName)
	if removeErr := os.Remove(licensePath); removeErr != nil && !os.IsNotExist(removeErr) {
		return fmt.Errorf("failed to remove license file: %w", removeErr)
	}
	m.state = nil
	return nil
}

// CheckIn updates the last-validated timestamp on a non-trial license.
func (m *Manager) CheckIn() *ActivationResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == nil {
		return &ActivationResult{
			Success: false,
			Message: "No active license to validate",
			Tier:    TierInvalid,
		}
	}
	m.state.LastValidatedAt = time.Now()
	_ = m.saveState() // Best-effort.
	return &ActivationResult{
		Success: true,
		Message: "License validated successfully",
		Tier:    m.state.Tier,
	}
}

// NeedsCheckIn returns true if optional check-in is recommended.
func (m *Manager) NeedsCheckIn() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil || m.state.IsTrialMode {
		return false
	}
	daysSinceCheck := int(time.Since(m.state.LastValidatedAt).Hours() / hoursPerDay)
	return daysSinceCheck >= CheckInInterval
}

// loadState reads and decrypts activation state from disk.
func (m *Manager) loadState() error {
	licensePath := filepath.Clean(filepath.Join(m.configDir, licenseFileName))

	f, openErr := os.Open(licensePath)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			return nil
		}
		return fmt.Errorf("open license file: %w", openErr)
	}
	defer func() { _ = f.Close() }()

	data, readErr := io.ReadAll(f)
	if readErr != nil {
		return fmt.Errorf("read license file: %w", readErr)
	}

	decrypted, decryptErr := m.decrypt(data)
	if decryptErr != nil {
		return fmt.Errorf("failed to decrypt license: %w", decryptErr)
	}

	state := &ActivationState{}
	if unmarshalErr := json.Unmarshal(decrypted, state); unmarshalErr != nil {
		return fmt.Errorf("failed to parse license: %w", unmarshalErr)
	}

	m.state = state
	return nil
}

// saveState encrypts and writes activation state to disk.
func (m *Manager) saveState() error {
	if m.state == nil {
		return nil
	}
	if mkdirErr := os.MkdirAll(m.configDir, 0o700); mkdirErr != nil {
		return fmt.Errorf("failed to create config directory: %w", mkdirErr)
	}

	data, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal license state: %w", marshalErr)
	}

	encrypted, encryptErr := m.encrypt(data)
	if encryptErr != nil {
		return encryptErr
	}

	licensePath := filepath.Join(m.configDir, licenseFileName)
	if writeErr := os.WriteFile(licensePath, encrypted, 0o600); writeErr != nil {
		return fmt.Errorf("failed to write license file: %w", writeErr)
	}
	return nil
}

// EncryptSecret encrypts arbitrary bytes for at-rest storage using
// the same key-derivation chain as the license state file. Returns
// base64-encoded AES-256-GCM ciphertext.
//
// Phase 2a (V1.0 NMS expansion) uses this to seal SNMPv3 auth/priv
// passphrases in the device_credentials table. See
// msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md.
//
// Callers should treat encryption as best-effort tamper resistance
// against on-disk snooping — the derivation depends on the device
// fingerprint, so credentials are bound to this seed install. They
// will not roundtrip after a fingerprint change.
func (m *Manager) EncryptSecret(plaintext []byte) ([]byte, error) {
	return m.encrypt(plaintext)
}

// DecryptSecret reverses EncryptSecret. Returns an error if the
// fingerprint key no longer matches what the ciphertext was sealed
// with.
func (m *Manager) DecryptSecret(ciphertext []byte) ([]byte, error) {
	return m.decrypt(ciphertext)
}

func (m *Manager) encrypt(plaintext []byte) ([]byte, error) {
	key := m.deriveKey()

	block, blockErr := aes.NewCipher(key)
	if blockErr != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", blockErr)
	}
	gcm, gcmErr := cipher.NewGCM(block)
	if gcmErr != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", gcmErr)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, nonceErr := io.ReadFull(rand.Reader, nonce); nonceErr != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", nonceErr)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

func (m *Manager) decrypt(ciphertext []byte) ([]byte, error) {
	data, decodeErr := base64.StdEncoding.DecodeString(string(ciphertext))
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", decodeErr)
	}

	key := m.deriveKey()
	block, blockErr := aes.NewCipher(key)
	if blockErr != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", blockErr)
	}
	gcm, gcmErr := cipher.NewGCM(block)
	if gcmErr != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", gcmErr)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, openErr := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if openErr != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", openErr)
	}
	return plaintext, nil
}

func (m *Manager) deriveKey() []byte {
	data := m.fingerprint.Hash() + encryptionSalt
	hash := sha256.Sum256([]byte(data))
	return hash[:] // 32 bytes for AES-256.
}
