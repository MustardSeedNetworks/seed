// Package auth handles JWT authentication.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

// Claims represents the JWT claims.
type Claims struct {
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"` // For token revocation (fixes #525)
	jwt.RegisteredClaims
}

// Manager handles authentication operations.
type Manager struct {
	mu             sync.RWMutex // Protects passwordHash and username (fixes #520)
	jwtSecret      []byte
	sessionTimeout time.Duration
	passwordHash   string
	username       string
	tokenVersion   int // Token version for revocation support (fixes #525)
}

// NewManager creates a new authentication manager.
func NewManager(jwtSecret string, sessionTimeout time.Duration, username, passwordHash string) *Manager {
	secret := jwtSecret
	if secret == "" {
		// Generate a random secret if not provided
		secret = generateRandomSecret()
	}

	return &Manager{
		jwtSecret:      []byte(secret),
		sessionTimeout: sessionTimeout,
		passwordHash:   passwordHash,
		username:       username,
	}
}

// generateRandomSecret creates a cryptographically secure random JWT secret.
// Note: This generates a new secret on each server restart, which will invalidate
// existing tokens. For persistent sessions across restarts, configure jwt_secret in the config file.
func generateRandomSecret() string {
	bytes := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(bytes); err != nil {
		// If crypto/rand fails, the system is critically insecure (fixes #543)
		// Panic instead of using a weak fallback - this should never happen on modern systems
		panic("crypto/rand failed: " + err.Error() + " - system is insecure, cannot continue")
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// Authenticate validates credentials and returns a JWT token.
// Uses constant-time comparison for username to prevent timing attacks (fixes #513).
func (m *Manager) Authenticate(username, password string) (string, error) {
	// Read credentials with read lock (fixes #520)
	m.mu.RLock()
	storedUsername := m.username
	storedPasswordHash := m.passwordHash
	m.mu.RUnlock()

	// Constant-time username comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare(
		[]byte(username),
		[]byte(storedUsername),
	) == 1

	// Password comparison (bcrypt.CompareHashAndPassword is already constant-time)
	passwordMatch := bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(password)) == nil

	// Both checks must succeed - evaluated in constant time
	if !usernameMatch || !passwordMatch {
		return "", ErrInvalidCredentials
	}

	return m.GenerateToken(username)
}

// GenerateToken creates a new JWT token for the given username.
// This is primarily used for testing. For production use, use Authenticate().
func (m *Manager) GenerateToken(username string) (string, error) {
	// Read token version with lock (fixes #520, #525)
	m.mu.RLock()
	currentVersion := m.tokenVersion
	m.mu.RUnlock()

	now := time.Now()
	claims := &Claims{
		Username:     username,
		TokenVersion: currentVersion, // Include version for revocation
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.sessionTimeout)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "LuminetIQ",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims.
// Checks token version to support revocation (fixes #525).
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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
	m.mu.RUnlock()

	if claims.TokenVersion < currentVersion {
		log.Printf("Token revoked: version %d < current %d", claims.TokenVersion, currentVersion)
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// HashPassword creates a bcrypt hash of a password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
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

// Middleware returns an HTTP middleware that validates JWT tokens.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login and setup endpoints
		if r.URL.Path == "/api/auth/login" ||
			r.URL.Path == "/api/setup/status" ||
			r.URL.Path == "/api/setup/complete" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for static files
		if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		var tokenString string

		// For WebSocket connections, support multiple auth methods
		if strings.HasPrefix(r.URL.Path, "/ws") {
			// Method 1 (Preferred): Check Sec-WebSocket-Protocol header
			// Format: "access_token, <token>" or "bearer, <token>"
			protocols := r.Header.Get("Sec-WebSocket-Protocol")
			if protocols != "" {
				tokenString = extractTokenFromSubprotocol(protocols)
			}

			// Method 2 (Deprecated): Query parameter fallback for backwards compatibility
			if tokenString == "" {
				tokenString = r.URL.Query().Get("token")
				if tokenString != "" {
					log.Println("WARNING: WebSocket authentication via query parameter is deprecated. Use Sec-WebSocket-Protocol header instead.")
				}
			}
		}

		// If no WebSocket token, check Authorization header
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}
			tokenString = parts[1]
		}

		claims, err := m.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				http.Error(w, "Token expired", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		r.Header.Set("X-Username", claims.Username)
		next.ServeHTTP(w, r)
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
func randomChar(chars string) (byte, error) {
	charsLen := byte(len(chars))
	// Calculate the largest multiple of charsLen that fits in a byte
	maxValid := 256 - (256 % int(charsLen))

	for {
		var b [1]byte
		if _, err := rand.Read(b[:]); err != nil {
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
func randomInt(n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}

	// For small n, use a single byte
	if n <= 256 {
		maxValid := 256 - (256 % n)
		for {
			var b [1]byte
			if _, err := rand.Read(b[:]); err != nil {
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
	max := uint64(1) << 32
	maxValid := uint32(max - (max % uint64(n)))

	for {
		if _, err := rand.Read(b[:]); err != nil {
			return 0, err
		}
		val := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
		if val < maxValid {
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
		j, err := randomInt(i + 1)
		if err != nil {
			return "", err
		}
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// GenerateJWTSecret creates a cryptographically secure JWT signing secret.
func GenerateJWTSecret() string {
	return generateRandomSecret()
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
	password, err := GenerateSecurePassword(16)
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
func (m *Manager) UpdatePasswordHash(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.passwordHash = hash
	m.tokenVersion++ // Invalidate all existing tokens
	log.Printf("Password hash updated, all existing tokens invalidated (version %d)", m.tokenVersion)
}

// IsDefaultPasswordHash checks if the given hash matches the default "netscope" password.
// This is used to detect if credentials have been changed from the insecure default.
func IsDefaultPasswordHash(hash string) bool {
	// The default hash for "netscope" - if this matches, credentials are insecure
	defaultHashes := []string{
		"$2y$10$1w5ktZnNS0UxbOvHKH2.hu01jsPh2RjkszVsP.7jR5cOZYa4oAI52",
	}

	for _, defaultHash := range defaultHashes {
		if hash == defaultHash {
			return true
		}
	}

	// Also check by actually comparing against "netscope"
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("netscope")); err == nil {
		return true
	}

	return false
}
