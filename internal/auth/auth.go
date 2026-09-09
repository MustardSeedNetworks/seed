// Package auth handles JWT authentication.
//
// auth.go carries the Manager itself and credential verification. Token
// minting and validation live in token.go, the HTTP middleware in
// middleware.go, and password generation in password_generate.go.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

var (
	// ErrInvalidCredentials is returned when username/password is incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken is returned when the JWT token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when the JWT token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrWeakPassword is returned when a password doesn't meet strength requirements.
	ErrWeakPassword = errors.New("password does not meet strength requirements")
)

// SetupModePlaceholder is a placeholder hash used during initial setup.
// This allows the server to start and show the wizard before a real password is set.
// It is not a valid bcrypt hash and will fail any authentication attempt.
const SetupModePlaceholder = "$setup$pending$"

// jwtSecretBytes is the number of random bytes for JWT signing secrets (256-bit key).
const jwtSecretBytes = 32

// UserStore provides user lookup and management operations.
// Implementations can use database, config file, or other storage backends.
type UserStore interface {
	// GetPasswordHash returns the password hash for a user.
	GetPasswordHash(ctx context.Context, username string) (string, error)
	// GetTokenVersion returns the current token version for a user.
	GetTokenVersion(ctx context.Context, username string) (int, error)
	// GetClientID returns the id of the client that owns a user.
	GetClientID(ctx context.Context, username string) (string, error)
	// UpdatePassword updates a user's password hash.
	UpdatePassword(ctx context.Context, username, hash string) error
	// RecordLoginSuccess records a successful login.
	RecordLoginSuccess(ctx context.Context, username string) error
	// RecordLoginFailure records a failed login attempt.
	RecordLoginFailure(ctx context.Context, username string) error
	// IsLocked checks if a user account is locked.
	IsLocked(ctx context.Context, username string) (bool, error)
}

// Manager handles authentication operations.
type Manager struct {
	mu             sync.RWMutex // Protects passwordHash and username (fixes #520)
	jwtSecret      []byte
	sessionTimeout time.Duration
	passwordHash   string
	username       string
	tokenVersion   int             // Token version for revocation support (fixes #525)
	userStore      UserStore       // Optional database-backed user store
	blacklist      *TokenBlacklist // Token blacklist for immediate revocation
}

// NewManager creates a new authentication manager.
func NewManager(
	jwtSecret string,
	sessionTimeout time.Duration,
	username, passwordHash string,
) *Manager {
	secret := jwtSecret
	if secret == "" {
		// Generate a random secret if not provided
		secret = GenerateJWTSecret()
	}

	return &Manager{
		jwtSecret:      []byte(secret),
		sessionTimeout: sessionTimeout,
		passwordHash:   passwordHash,
		username:       username,
		blacklist:      NewTokenBlacklist(),
	}
}

// SetUserStore sets the database-backed user store for authentication.
// When set, the manager will use the database for user lookups instead of in-memory.
func (m *Manager) SetUserStore(store UserStore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userStore = store
	logging.GetLogger().Info("UserStore set for authentication", "hasStore", store != nil)
}

// cryptoRandRead attempts to read random bytes with retry logic.
// This provides resilience against transient crypto/rand failures (fixes G7).
// Returns error only after exhausting all retry attempts.
func cryptoRandRead(b []byte, operation string) error {
	const (
		maxRetries     = 3
		initialBackoff = 10 * time.Millisecond
		maxBackoff     = 100 * time.Millisecond
	)

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if _, err := rand.Read(b); err != nil {
			lastErr = err
			if attempt < maxRetries {
				logging.GetLogger().Warn("crypto/rand failed, retrying",
					"operation", operation,
					"attempt", attempt+1,
					"max_attempts", maxRetries+1,
					"error", err,
					"retry_in", backoff)
				time.Sleep(backoff)
				// Exponential backoff with cap
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			// All retries exhausted
			logging.GetLogger().
				Error("crypto/rand failed after all retries - system is in insecure state",
					"operation", operation,
					"attempts", maxRetries+1,
					"error", err)
			return fmt.Errorf("crypto/rand read failed for %s: %w", operation, err)
		}
		// Success
		if attempt > 0 {
			logging.GetLogger().Info("crypto/rand recovered after retry",
				"operation", operation,
				"attempts", attempt+1)
		}
		return nil
	}

	return lastErr
}

// GenerateJWTSecret creates a cryptographically secure JWT signing secret.
// Note: This generates a new secret on each server restart, which will invalidate
// existing tokens. For persistent sessions across restarts, configure jwt_secret in the config file.
// Fixes #539: Consolidated JWT secret generation into single function.
// Fixes G7: Added retry logic for crypto/rand failures instead of immediate panic.
func GenerateJWTSecret() string {
	bytes := make([]byte, jwtSecretBytes)
	if err := cryptoRandRead(bytes, "GenerateJWTSecret"); err != nil {
		// If crypto/rand fails after retries, the system is critically insecure
		// Panic to prevent operation in an insecure state - this should never happen on modern systems
		panic(
			"crypto/rand failed after retries: " + err.Error() + " - system is insecure, cannot continue",
		)
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// VerifyPasswordOnly checks the username/password pair without issuing
// a JWT. It is the first half of a two-factor login flow: callers
// check this, then probe TOTP enrolment, then either issue an
// MFA-pending token or a full JWT.
//
// Side-effects are identical to Authenticate: success records a login
// success on the UserStore; failure records a login failure. Wave 3
// (#85).
func (m *Manager) VerifyPasswordOnly(ctx context.Context, username, password string) error {
	m.mu.RLock()
	storedUsername := m.username
	storedPasswordHash := m.passwordHash
	userStore := m.userStore
	m.mu.RUnlock()

	if userStore != nil {
		return m.verifyPasswordOnlyWithStore(ctx, userStore, username, password)
	}

	usernameMatch := subtle.ConstantTimeCompare(
		[]byte(username), []byte(storedUsername),
	) == 1
	passwordMatch, needsRehash, verifyErr := VerifyPassword(storedPasswordHash, password)
	if verifyErr != nil {
		return ErrInvalidCredentials
	}
	if !usernameMatch || !passwordMatch {
		return ErrInvalidCredentials
	}
	if needsRehash {
		if newHash, hashErr := HashPassword(password); hashErr == nil {
			m.mu.Lock()
			m.passwordHash = newHash
			m.mu.Unlock()
		}
	}
	return nil
}

// verifyPasswordOnlyWithStore is the UserStore-backed branch of
// VerifyPasswordOnly. Kept separate so the function-length limits stay
// happy and the in-memory branch remains a single straight line.
func (m *Manager) verifyPasswordOnlyWithStore(
	ctx context.Context, userStore UserStore, username, password string,
) error {
	locked, lockErr := userStore.IsLocked(ctx, username)
	if lockErr == nil && locked {
		return ErrInvalidCredentials
	}

	dbHash, err := userStore.GetPasswordHash(ctx, username)
	if err != nil {
		_ = userStore.RecordLoginFailure(ctx, username)
		return ErrInvalidCredentials
	}
	matched, needsRehash, verifyErr := VerifyPassword(dbHash, password)
	if verifyErr != nil {
		_ = userStore.RecordLoginFailure(ctx, username)
		return ErrInvalidCredentials
	}
	if !matched {
		_ = userStore.RecordLoginFailure(ctx, username)
		return ErrInvalidCredentials
	}
	if needsRehash {
		_ = m.rehashAndPersist(ctx, userStore, username, password)
	}
	if successErr := userStore.RecordLoginSuccess(ctx, username); successErr != nil {
		logging.GetLogger().WarnContext(ctx,
			"Failed to record login success", "username", username, "error", successErr)
	}
	return nil
}

// Authenticate validates credentials and returns a JWT token.
// Uses constant-time comparison for username to prevent timing attacks (fixes #513).
// If a UserStore is set, uses database for authentication; otherwise uses in-memory.
func (m *Manager) Authenticate(ctx context.Context, username, password string) (string, error) {
	// Read credentials and userStore with read lock (fixes #520)
	m.mu.RLock()
	storedUsername := m.username
	storedPasswordHash := m.passwordHash
	userStore := m.userStore
	m.mu.RUnlock()

	// If we have a UserStore, use it for authentication
	if userStore != nil {
		return m.authenticateWithUserStore(ctx, userStore, username, password)
	}

	// Fallback to in-memory authentication (legacy/config-based)
	return m.authenticateInMemory(ctx, username, password, storedUsername, storedPasswordHash)
}

// authenticateWithUserStore handles authentication via the database-backed UserStore.
func (m *Manager) authenticateWithUserStore(
	ctx context.Context,
	userStore UserStore,
	username, password string,
) (string, error) {
	// Check if account is locked
	locked, err := userStore.IsLocked(ctx, username)
	if err != nil {
		logging.GetLogger().
			WarnContext(ctx, "Failed to check user lock status", "username", username, "error", err)
	}
	if locked {
		return "", ErrInvalidCredentials
	}

	// Get password hash from database
	dbHash, err := userStore.GetPasswordHash(ctx, username)
	if err != nil {
		// User not found in database - record failure and return error
		_ = userStore.RecordLoginFailure(ctx, username)
		return "", ErrInvalidCredentials
	}

	// Password comparison handles both Argon2id and legacy bcrypt; bcrypt
	// matches trigger transparent rehash + persist (Wave 2 / task #84).
	matched, needsRehash, verifyErr := VerifyPassword(dbHash, password)
	if verifyErr != nil {
		logging.GetLogger().
			ErrorContext(ctx, "Stored password hash has unsupported format",
				"username", username, "error", verifyErr,
				"event", "auth.password.unsupported_hash")
		_ = userStore.RecordLoginFailure(ctx, username)
		return "", ErrInvalidCredentials
	}
	if !matched {
		_ = userStore.RecordLoginFailure(ctx, username)
		return "", ErrInvalidCredentials
	}

	// Transparent migration: re-hash legacy bcrypt credentials with
	// Argon2id and persist before issuing the token. Failure here is
	// logged but does not block the login (the user has already
	// successfully authenticated).
	if needsRehash {
		if rehashErr := m.rehashAndPersist(ctx, userStore, username, password); rehashErr != nil {
			logging.GetLogger().
				WarnContext(ctx, "Password rehash failed; login still permitted",
					"username", username, "error", rehashErr,
					"event", "auth.password.rehash_failed")
		} else {
			logging.GetLogger().
				InfoContext(ctx, "Password migrated to argon2id",
					"username", username,
					"previous_algorithm", string(HashAlgorithmBcrypt),
					"event", "auth.password.rehashed")
		}
	}

	// Record successful login
	if successErr := userStore.RecordLoginSuccess(ctx, username); successErr != nil {
		logging.GetLogger().
			WarnContext(ctx, "Failed to record login success", "username", username, "error", successErr)
	}

	return m.GenerateToken(ctx, username)
}

// rehashAndPersist generates a new Argon2id hash for the given password
// and writes it via the UserStore. It is invoked transparently when a
// legacy bcrypt hash matches at login time.
func (m *Manager) rehashAndPersist(
	ctx context.Context,
	userStore UserStore,
	username, password string,
) error {
	newHash, hashErr := HashPassword(password)
	if hashErr != nil {
		return fmt.Errorf("hash password: %w", hashErr)
	}
	if updateErr := userStore.UpdatePassword(ctx, username, newHash); updateErr != nil {
		return fmt.Errorf("persist new hash: %w", updateErr)
	}
	// Keep the in-memory mirror in sync if this is the default user.
	m.mu.Lock()
	if m.username == username {
		m.passwordHash = newHash
	}
	m.mu.Unlock()
	return nil
}

// authenticateInMemory handles authentication via in-memory credentials (legacy/config-based).
func (m *Manager) authenticateInMemory(
	ctx context.Context,
	username, password, storedUsername, storedPasswordHash string,
) (string, error) {
	// Constant-time username comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare(
		[]byte(username),
		[]byte(storedUsername),
	) == 1

	// Password comparison handles both Argon2id and legacy bcrypt; bcrypt
	// matches trigger transparent rehash of the in-memory hash.
	passwordMatch, needsRehash, verifyErr := VerifyPassword(storedPasswordHash, password)
	if verifyErr != nil {
		logging.GetLogger().
			ErrorContext(ctx, "In-memory password hash has unsupported format",
				"error", verifyErr,
				"event", "auth.password.unsupported_hash")
		return "", ErrInvalidCredentials
	}

	// Both checks must succeed - evaluated in constant time
	if !usernameMatch || !passwordMatch {
		return "", ErrInvalidCredentials
	}

	if needsRehash {
		if newHash, hashErr := HashPassword(password); hashErr == nil {
			m.mu.Lock()
			m.passwordHash = newHash
			m.mu.Unlock()
			logging.GetLogger().
				InfoContext(ctx, "In-memory password migrated to argon2id",
					"username", username,
					"previous_algorithm", string(HashAlgorithmBcrypt),
					"event", "auth.password.rehashed")
		} else {
			logging.GetLogger().
				WarnContext(ctx, "In-memory password rehash failed; login still permitted",
					"error", hashErr,
					"event", "auth.password.rehash_failed")
		}
	}

	return m.GenerateToken(ctx, username)
}

// UpdatePasswordHash updates the auth manager's password hash at runtime.
// This is used when the password is changed via the setup wizard or settings.
// Also increments token version to invalidate all existing tokens (fixes #520, #525).
// If a UserStore is set, updates the database as well.
func (m *Manager) UpdatePasswordHash(ctx context.Context, hash string) {
	m.mu.Lock()
	m.passwordHash = hash
	m.tokenVersion++ // Invalidate all existing tokens
	userStore := m.userStore
	username := m.username
	m.mu.Unlock()

	// If we have a UserStore, update the database as well
	if userStore != nil && username != "" {
		if err := userStore.UpdatePassword(ctx, username, hash); err != nil {
			logging.GetLogger().
				ErrorContext(ctx, "Failed to update password in database", "username", username, "error", err)
		} else {
			logging.GetLogger().InfoContext(ctx, "Password hash updated in database", "username", username)
		}
	}

	logging.GetLogger().
		InfoContext(ctx, "Password hash updated, all existing tokens invalidated", "version", m.tokenVersion)
}

// UpdatePasswordHashForUser updates the password hash for a specific user.
// This is used when changing password for a user that may differ from the default.
func (m *Manager) UpdatePasswordHashForUser(ctx context.Context, username, hash string) error {
	m.mu.RLock()
	userStore := m.userStore
	m.mu.RUnlock()

	if userStore == nil {
		return errors.New("no UserStore configured")
	}

	if err := userStore.UpdatePassword(ctx, username, hash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	logging.GetLogger().InfoContext(ctx, "Password hash updated for user", "username", username)
	return nil
}

// IsDefaultPasswordHash checks if the given hash matches the default "seed" password.
// This is used to detect if credentials have been changed from the insecure default.
func IsDefaultPasswordHash(hash string) bool {
	// Check empty hash (initial setup needed)
	if hash == "" {
		return true
	}

	// Check setup mode placeholder
	if hash == SetupModePlaceholder {
		return true
	}

	// Check default "seed" password against both Argon2id and bcrypt
	// formats (legacy installs still carry bcrypt hashes until migrated).
	if matched, _, err := VerifyPassword(hash, "seed"); err == nil && matched {
		return true
	}

	return false
}

// Stop gracefully shuts down the auth manager and its cleanup goroutines.
func (m *Manager) Stop() {
	if m.blacklist != nil {
		m.blacklist.Stop()
	}
}
