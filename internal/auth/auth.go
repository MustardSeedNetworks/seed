// Package auth handles JWT authentication.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"

	"github.com/krisarmstrong/seed/internal/logging"
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

// PasswordRequirements describes minimum password requirements (fixes #535).
const (
	MinPasswordLength = 12 // Increased from 8 for better security
)

// Cryptographic constants for secure random generation.
const (
	// jwtSecretBytes is the number of random bytes for JWT signing secrets (256-bit key).
	jwtSecretBytes = 32

	// maxByteValue is the maximum value of a single byte, used for unbiased random selection.
	maxByteValue = 256

	// bitsPerUint32 is the number of bits in a uint32, used for random integer generation.
	bitsPerUint32 = 32

	// byteShift8 is the bit shift amount for the second byte in uint32 construction.
	byteShift8 = 8

	// byteShift16 is the bit shift amount for the third byte in uint32 construction.
	byteShift16 = 16

	// byteShift24 is the bit shift amount for the fourth byte in uint32 construction.
	byteShift24 = 24

	// initialCredentialsPasswordLength is the password length for auto-generated initial credentials.
	initialCredentialsPasswordLength = 16
)

// SetupModePlaceholder is a placeholder hash used during initial setup.
// This allows the server to start and show the wizard before a real password is set.
// It is not a valid bcrypt hash and will fail any authentication attempt.
const SetupModePlaceholder = "$setup$pending$"

// Common error codes for auth middleware JSON responses (matches api.ErrorResponse).
const (
	errCodeUnauthorized = "UNAUTHORIZED"
	errCodeForbidden    = "FORBIDDEN"
)

// authErrorResponse represents a standardized error response for auth middleware.
type authErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// sendAuthError sends a JSON error response from auth/CSRF middleware.
// This ensures consistent error formats matching the API error schema.
func sendAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := authErrorResponse{
		Error: message,
		Code:  code,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Error("Failed to encode auth error response", "error", err)
	}
}

// Claims represents the JWT claims.
type Claims struct {
	jwt.RegisteredClaims

	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"` // For token revocation (fixes #525)
	TokenType    string `json:"token_type"`    // "access" or "refresh"
}

// UserStore provides user lookup and management operations.
// Implementations can use database, config file, or other storage backends.
type UserStore interface {
	// GetPasswordHash returns the password hash for a user.
	GetPasswordHash(ctx context.Context, username string) (string, error)
	// GetTokenVersion returns the current token version for a user.
	GetTokenVersion(ctx context.Context, username string) (int, error)
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

// GenerateToken creates a new JWT token for the given username.
// This is primarily used for testing. For production use, use Authenticate().
func (m *Manager) GenerateToken(ctx context.Context, username string) (string, error) {
	return m.generateTokenWithType(ctx, username, "access", m.sessionTimeout)
}

// GenerateAccessToken creates a short-lived access token (fixes #478).
func (m *Manager) GenerateAccessToken(ctx context.Context, username string) (string, error) {
	return m.generateTokenWithType(ctx, username, "access", AccessTokenDuration)
}

// GenerateRefreshToken creates a long-lived refresh token (fixes #478).
func (m *Manager) GenerateRefreshToken(ctx context.Context, username string) (string, error) {
	return m.generateTokenWithType(ctx, username, "refresh", RefreshTokenDuration)
}

// generateTokenWithType creates a JWT token with specified type and duration.
func (m *Manager) generateTokenWithType(
	ctx context.Context,
	username, tokenType string,
	duration time.Duration,
) (string, error) {
	// Read token version with lock (fixes #520, #525)
	m.mu.RLock()
	currentVersion := m.tokenVersion
	userStore := m.userStore
	m.mu.RUnlock()

	// If we have a UserStore, get the token version from the database
	// This ensures tokens are generated with the correct version (fixes #927)
	if userStore != nil && username != "" {
		if dbVersion, err := userStore.GetTokenVersion(ctx, username); err == nil {
			currentVersion = dbVersion
		}
		// On error, fall back to in-memory version
	}

	now := time.Now()
	claims := &Claims{
		Username:     username,
		TokenVersion: currentVersion, // Include version for revocation
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "The Seed",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims.
// Checks token version to support revocation (fixes #525).
// Also checks the token blacklist for immediate revocation (ported from Stem).
// If UserStore is set, queries database for current token version.
func (m *Manager) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Check blacklist first (fast path for revoked tokens)
	tokenID := tokenFingerprint(tokenString)
	if m.blacklist != nil && m.blacklist.IsBlacklisted(tokenID) {
		logging.GetLogger().InfoContext(ctx, "Token is blacklisted", "token_id", tokenID[:8]+"...")
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check token version for revocation (fixes #525)
	m.mu.RLock()
	currentVersion := m.tokenVersion
	userStore := m.userStore
	m.mu.RUnlock()

	// If we have a UserStore, get current token version from database
	if userStore != nil && claims.Username != "" {
		dbVersion, versionErr := userStore.GetTokenVersion(ctx, claims.Username)
		if versionErr == nil {
			currentVersion = dbVersion
		}
		// On error, fall back to in-memory version
	}

	if claims.TokenVersion < currentVersion {
		logging.GetLogger().
			InfoContext(ctx, "Token revoked", "version", claims.TokenVersion, "current", currentVersion)
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the claims (fixes #478).
// Ensures the token is actually a refresh token, not an access token.
func (m *Manager) ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Ensure it's a refresh token
	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token (fixes #478).
// This allows short-lived access tokens with long-lived refresh tokens.
// Enforces maximum session lifetime to prevent indefinite sessions (fixes #717).
func (m *Manager) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := m.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	// Check if session has exceeded maximum lifetime (fixes #717)
	// The IssuedAt claim represents when the refresh token (and thus the session) was created
	if claims.IssuedAt != nil {
		sessionAge := time.Since(claims.IssuedAt.Time)
		if sessionAge > MaxSessionLifetime {
			logging.GetLogger().InfoContext(ctx, "Session exceeded maximum lifetime",
				"age", sessionAge,
				"max", MaxSessionLifetime,
				"username", claims.Username)
			return "", ErrTokenExpired
		}
	}

	// Generate new access token with same username
	return m.GenerateAccessToken(ctx, claims.Username)
}

// extractTokenFromSubprotocol extracts the JWT token from WebSocket subprotocol header.
// Supports formats:
//   - "access_token, <token>"
//   - "bearer, <token>"
//   - Just "<token>" (fallback)
func extractTokenFromSubprotocol(protocols string) string {
	// Split by comma to handle multiple protocols
	parts := strings.Split(protocols, ",")

	for i, part := range parts {
		part = strings.TrimSpace(part)

		// Check if this part is the auth protocol indicator
		if part == "access_token" || part == "bearer" {
			// Next part should be the token
			if i+1 < len(parts) {
				return strings.TrimSpace(parts[i+1])
			}
		}
	}

	// Fallback: if no recognized protocol, treat the whole string as token
	// This handles cases where client sends just the token
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0])
	}

	return ""
}

// shouldBypassAuth checks if the given path should skip authentication.
// Returns true for login, refresh, setup, recovery, SSO endpoints and static files.
func shouldBypassAuth(path string) bool {
	// Skip auth for login, refresh, setup, recovery, and SSO endpoints (fixes #478)
	// Note: API routes use /api/v1/ prefix
	switch path {
	case "/api/v1/auth/login", "/api/v1/auth/refresh",
		"/api/v1/setup/status", "/api/v1/setup/complete",
		"/api/v1/recovery/status", "/api/v1/recovery/complete", "/api/v1/recovery/instructions":
		return true
	}
	if strings.HasPrefix(path, "/api/v1/sso/") {
		return true
	}
	// Skip auth for static files (non-API, non-WebSocket paths)
	if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws") {
		return true
	}
	return false
}

// extractTokenFromRequest extracts the JWT token from the request.
// For WebSocket connections, checks Sec-WebSocket-Protocol header first.
// Falls back to cookie/Authorization header via GetTokenFromRequest.
func extractTokenFromRequest(r *http.Request) string {
	// For WebSocket connections, check Sec-WebSocket-Protocol first (fixes #478)
	if strings.HasPrefix(r.URL.Path, "/ws") {
		// Method 1 (Preferred): Check Sec-WebSocket-Protocol header
		// Format: "access_token, <token>" or "bearer, <token>"
		if protocols := r.Header.Get("Sec-WebSocket-Protocol"); protocols != "" {
			if token := extractTokenFromSubprotocol(protocols); token != "" {
				return token
			}
		}
	}

	// Use unified token extraction with cookie priority (fixes #478)
	token, _ := GetTokenFromRequest(r)
	return token
}

// handleTokenValidationError sends the appropriate error response for token validation failures.
func handleTokenValidationError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrTokenExpired) {
		sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Token expired")
		return
	}
	sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Invalid token")
}

// Middleware returns an HTTP middleware that validates JWT tokens.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for bypassed paths (login, refresh, setup, SSO, static files)
		if shouldBypassAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from request (WebSocket subprotocol or cookie/header)
		tokenString := extractTokenFromRequest(r)
		if tokenString == "" {
			sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Unauthorized")
			return
		}

		// Validate the token
		claims, err := m.ValidateToken(r.Context(), tokenString)
		if err != nil {
			handleTokenValidationError(w, err)
			return
		}

		// Validate username claim exists and is not empty (fixes #711)
		if claims.Username == "" {
			sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Invalid token: missing username claim")
			return
		}

		// Add claims to request context
		ctx := logging.WithUserID(r.Context(), claims.Username)
		r.Header.Set("X-Username", claims.Username) // Keep this for other potential uses
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidatePasswordStrength checks if a password meets minimum requirements (fixes #535).
// Requirements: at least 12 characters, contains uppercase, lowercase, digit, and special character.
func ValidatePasswordStrength(password string) error {
	if len(password) < MinPasswordLength {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return ErrWeakPassword
	}

	return nil
}

// randomChar selects an unbiased random character from the given charset.
// Uses rejection sampling to avoid modulo bias (fixes #517).
// Fixes G7: Uses cryptoRandRead for retry logic on crypto/rand failures.
func randomChar(chars string) (byte, error) {
	charsLen := byte(len(chars))
	// Calculate the largest multiple of charsLen that fits in a byte
	maxValid := maxByteValue - (maxByteValue % int(charsLen))

	for {
		var b [1]byte
		if err := cryptoRandRead(b[:], "randomChar"); err != nil {
			return 0, err
		}
		// Accept only if the random byte is in the unbiased range
		if int(b[0]) < maxValid {
			return chars[b[0]%charsLen], nil
		}
		// Reject and retry if in the biased range
	}
}

// randomInt returns an unbiased random integer in the range [0, n).
// Uses rejection sampling to avoid modulo bias.
// Fixes G7: Uses cryptoRandRead for retry logic on crypto/rand failures.
func randomInt(n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}

	// For small n, use a single byte
	if n <= maxByteValue {
		maxValid := maxByteValue - (maxByteValue % n)
		for {
			var b [1]byte
			if err := cryptoRandRead(b[:], "randomInt"); err != nil {
				return 0, err
			}
			if int(b[0]) < maxValid {
				return int(b[0]) % n, nil
			}
		}
	}

	// For larger n, use multiple bytes
	var b [4]byte
	// Use uint64 for calculation to avoid overflow
	maxUint := uint64(1) << bitsPerUint32
	// #nosec G115 -- Result is always < 2^32 (maxUint - remainder), safe for uint32
	maxValid := uint32(maxUint - (maxUint % uint64(n)))

	for {
		if err := cryptoRandRead(b[:], "randomInt"); err != nil {
			return 0, err
		}
		val := uint32(b[0]) | uint32(b[1])<<byteShift8 | uint32(b[2])<<byteShift16 | uint32(b[3])<<byteShift24
		if val < maxValid {
			// #nosec G115 -- val % uint32(n) is always < n, safe for int conversion
			return int(val % uint32(n)), nil
		}
	}
}

// GenerateSecurePassword creates a cryptographically secure random password.
// The password will contain uppercase, lowercase, digits, and special characters (fixes #535).
// Uses rejection sampling to avoid modulo bias (fixes #517).
func GenerateSecurePassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}

	// Character sets for password generation
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
		allChars     = lowerChars + upperChars + digitChars + specialChars
	)

	// Ensure at least one of each required type
	password := make([]byte, length)

	// Ensure we have at least one of each required type using unbiased selection
	var err error
	password[0], err = randomChar(lowerChars)
	if err != nil {
		return "", err
	}
	password[1], err = randomChar(upperChars)
	if err != nil {
		return "", err
	}
	password[2], err = randomChar(digitChars)
	if err != nil {
		return "", err
	}
	password[3], err = randomChar(specialChars)
	if err != nil {
		return "", err
	}

	// Fill the rest randomly from all characters using unbiased selection
	for i := 4; i < length; i++ {
		password[i], err = randomChar(allChars)
		if err != nil {
			return "", err
		}
	}

	// Shuffle the password to randomize positions of required characters
	// Use Fisher-Yates shuffle with unbiased random selection
	for i := len(password) - 1; i > 0; i-- {
		var j int
		j, err = randomInt(i + 1)
		if err != nil {
			return "", err
		}
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// InitialCredentials holds the generated initial credentials for display.
type InitialCredentials struct {
	Username     string
	Password     string
	PasswordHash string
	JWTSecret    string
}

// GenerateInitialCredentials creates new secure credentials for first-boot setup.
func GenerateInitialCredentials(username string) (*InitialCredentials, error) {
	password, err := GenerateSecurePassword(initialCredentialsPasswordLength)
	if err != nil {
		return nil, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	return &InitialCredentials{
		Username:     username,
		Password:     password,
		PasswordHash: hash,
		JWTSecret:    GenerateJWTSecret(),
	}, nil
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

// tokenFingerprint generates a short, unique identifier for a JWT token.
// Uses SHA-256 hash of the token string for efficient blacklist storage.
func tokenFingerprint(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// RevokeToken adds a token to the blacklist for immediate revocation.
// The token remains blacklisted until its natural expiration time.
// This is used on logout to ensure the token cannot be reused.
func (m *Manager) RevokeToken(tokenString string) {
	if m.blacklist == nil || tokenString == "" {
		return
	}

	// Parse the token to get its expiration time
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (any, error) {
		return m.jwtSecret, nil
	})
	if err != nil {
		// Even if parsing fails, blacklist with a default expiration
		m.blacklist.Add(tokenFingerprint(tokenString), time.Now().Add(AccessTokenDuration))
		return
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.ExpiresAt == nil {
		m.blacklist.Add(tokenFingerprint(tokenString), time.Now().Add(AccessTokenDuration))
		return
	}

	// Add to blacklist with the token's actual expiration time
	m.blacklist.Add(tokenFingerprint(tokenString), claims.ExpiresAt.Time)
}

// Stop gracefully shuts down the auth manager and its cleanup goroutines.
func (m *Manager) Stop() {
	if m.blacklist != nil {
		m.blacklist.Stop()
	}
}
