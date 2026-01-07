package auth

// Password recovery via file-based trigger mechanism.
// Allows headless machines to recover admin passwords via SSH access.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

const (
	// recoveryTokenLength is the length of recovery tokens in bytes before encoding.
	recoveryTokenLength = 32

	// recoveryTokenExpiry is how long a recovery token remains valid (15 minutes).
	recoveryTokenExpiry = 15 * time.Minute

	// recoveryTriggerFile is the filename that triggers recovery mode.
	recoveryTriggerFile = ".recovery"

	// recoveryTokenFile is the filename where the token is written.
	recoveryTokenFile = ".recovery-token"
)

// RecoveryToken represents a one-time recovery token with metadata.
type RecoveryToken struct {
	Token     string    // The actual token string (base64-encoded)
	ExpiresAt time.Time // When the token expires
	Used      bool      // Whether the token has been used
}

// RecoveryTokenManager manages password recovery token generation and validation.
// Implements a file-based trigger mechanism for headless password recovery.
//
// Recovery flow:
//  1. Admin creates .recovery file in data directory (requires SSH/filesystem access)
//  2. CheckRecoveryMode() detects trigger, generates token, writes to .recovery-token
//  3. Admin reads token from .recovery-token via SSH
//  4. Admin enters token + new password on login page
//  5. ValidateAndConsume() validates token and marks it as used
//  6. Cleanup() removes trigger files after successful recovery
type RecoveryTokenManager struct {
	mu      sync.RWMutex
	token   *RecoveryToken
	dataDir string
}

// NewRecoveryTokenManager creates a new recovery token manager.
// dataDir is the application data directory where trigger files are monitored.
func NewRecoveryTokenManager(dataDir string) *RecoveryTokenManager {
	return &RecoveryTokenManager{
		dataDir: dataDir,
	}
}

// TriggerFilePath returns the path to the recovery trigger file.
func (m *RecoveryTokenManager) TriggerFilePath() string {
	return filepath.Join(m.dataDir, recoveryTriggerFile)
}

// TokenFilePath returns the path to the recovery token file.
func (m *RecoveryTokenManager) TokenFilePath() string {
	return filepath.Join(m.dataDir, recoveryTokenFile)
}

// CheckRecoveryMode checks if recovery mode should be activated.
// If the trigger file exists and no valid token exists, generates a new token.
// Returns true if recovery mode is active.
func (m *RecoveryTokenManager) CheckRecoveryMode() bool {
	// Check if trigger file exists
	triggerPath := m.TriggerFilePath()
	if _, err := os.Stat(triggerPath); os.IsNotExist(err) {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// If we already have a valid unexpired token, recovery mode is active
	if m.hasValidTokenLocked() {
		return true
	}

	// Generate new token since trigger exists but no valid token
	if err := m.generateTokenLocked(); err != nil {
		logging.GetLogger().Error("Failed to generate recovery token", "error", err)
		return false
	}

	return true
}

// hasValidTokenLocked checks if there's a valid token (must hold lock).
func (m *RecoveryTokenManager) hasValidTokenLocked() bool {
	if m.token == nil || m.token.Used {
		return false
	}
	return time.Now().Before(m.token.ExpiresAt)
}

// generateTokenLocked generates a new token and writes it to file (must hold lock).
func (m *RecoveryTokenManager) generateTokenLocked() error {
	// Generate cryptographically secure random bytes
	tokenBytes := make([]byte, recoveryTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store the new token
	m.token = &RecoveryToken{
		Token:     token,
		ExpiresAt: time.Now().Add(recoveryTokenExpiry),
		Used:      false,
	}

	// Write token to file so admin can read it via SSH
	tokenPath := m.TokenFilePath()
	tokenContent := token + "\n"
	if err := os.WriteFile(tokenPath, []byte(tokenContent), 0o600); err != nil {
		m.token = nil // Clear token if we can't write it
		return err
	}

	logging.GetLogger().Info("Recovery token generated",
		"token_file", tokenPath,
		"expires_in", recoveryTokenExpiry)

	return nil
}

// ValidateAndConsume checks if the provided token is valid.
// The token is invalidated after successful validation (single-use).
// Uses constant-time comparison to prevent timing attacks.
func (m *RecoveryTokenManager) ValidateAndConsume(providedToken string) bool {
	if providedToken == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if token exists
	if m.token == nil {
		logging.GetLogger().Warn("Recovery validation failed: no token exists")
		return false
	}

	// Check if already used
	if m.token.Used {
		logging.GetLogger().Warn("Recovery validation failed: token already used")
		return false
	}

	// Check expiry
	if time.Now().After(m.token.ExpiresAt) {
		logging.GetLogger().Warn("Recovery validation failed: token expired")
		m.token = nil // Clean up expired token
		return false
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(m.token.Token)) != 1 {
		logging.GetLogger().Warn("Recovery validation failed: token mismatch")
		return false
	}

	// Mark as used (single-use)
	m.token.Used = true
	logging.GetLogger().Info("Recovery token validated successfully")

	return true
}

// Cleanup removes recovery trigger files after successful password reset.
// Should be called after the password has been successfully updated.
func (m *RecoveryTokenManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove trigger file
	triggerPath := m.TriggerFilePath()
	if err := os.Remove(triggerPath); err != nil && !os.IsNotExist(err) {
		logging.GetLogger().Warn("Failed to remove recovery trigger file", "path", triggerPath, "error", err)
	}

	// Remove token file
	tokenPath := m.TokenFilePath()
	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		logging.GetLogger().Warn("Failed to remove recovery token file", "path", tokenPath, "error", err)
	}

	// Clear internal state
	m.token = nil

	logging.GetLogger().Info("Recovery mode cleanup complete")
}

// IsActive returns true if recovery mode is currently active.
// This is a read-only check that doesn't trigger token generation.
func (m *RecoveryTokenManager) IsActive() bool {
	// Check if trigger file exists
	triggerPath := m.TriggerFilePath()
	if _, err := os.Stat(triggerPath); os.IsNotExist(err) {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.hasValidTokenLocked()
}

// Invalidate removes any existing token without cleanup of trigger files.
// Used when token expires or becomes invalid.
func (m *RecoveryTokenManager) Invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = nil
}

// RemainingTime returns the time remaining until the token expires.
// Returns 0 if no valid token exists.
func (m *RecoveryTokenManager) RemainingTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.hasValidTokenLocked() {
		return 0
	}

	remaining := time.Until(m.token.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// TokenExpiryDuration returns the configured token expiry duration.
func (m *RecoveryTokenManager) TokenExpiryDuration() time.Duration {
	return recoveryTokenExpiry
}
