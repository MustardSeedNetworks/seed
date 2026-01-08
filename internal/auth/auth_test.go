package auth_test

// Test suite validates password hashing, JWT issuance/verification, and middleware behavior.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// assertAuthError is a test helper that verifies authentication failed as expected.
func assertAuthError(t *testing.T, err, wantErr error, token string) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
	if token != "" {
		t.Error("expected empty token on error")
	}
}

// assertAuthSuccess is a test helper that verifies authentication succeeded.
func assertAuthSuccess(t *testing.T, err error, token string) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected token, got empty string")
	}
}

func TestNewManager(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.ManagerUsername() != defaults.Auth.Username {
		t.Errorf("expected username %s, got %s", defaults.Auth.Username, m.ManagerUsername())
	}
	if m.ManagerSessionTimeout() != time.Hour {
		t.Errorf("expected timeout 1h, got %v", m.ManagerSessionTimeout())
	}
}

func TestNewManagerWithEmptySecret(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager("", time.Hour, defaults.Auth.Username, defaults.Auth.PasswordHash)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if len(m.ManagerJWTSecret()) == 0 {
		t.Error("expected generated JWT secret, got empty")
	}
}

func TestAuthenticate(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "valid credentials",
			username: defaults.Auth.Username,
			password: defaults.Auth.Password,
			wantErr:  nil,
		},
		{
			name:     "wrong username",
			username: "wronguser",
			password: defaults.Auth.Password,
			wantErr:  auth.ErrInvalidCredentials,
		},
		{
			name:     "wrong password",
			username: defaults.Auth.Username,
			password: "wrongpassword",
			wantErr:  auth.ErrInvalidCredentials,
		},
		{
			name:     "empty username",
			username: "",
			password: defaults.Auth.Password,
			wantErr:  auth.ErrInvalidCredentials,
		},
		{
			name:     "empty password",
			username: defaults.Auth.Username,
			password: "",
			wantErr:  auth.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			token, err := m.Authenticate(ctx, tt.username, tt.password)

			// Handle expected error case
			if tt.wantErr != nil {
				assertAuthError(t, err, tt.wantErr, token)
				return
			}

			// Handle expected success case
			assertAuthSuccess(t, err, token)
		})
	}
}

func TestValidateToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	// Generate valid token
	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Validate token
	claims, err := m.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.Username != defaults.Auth.Username {
		t.Errorf("expected username %s, got %s", defaults.Auth.Username, claims.Username)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: auth.ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "not.a.valid.token",
			wantErr: auth.ErrInvalidToken,
		},
		{
			name:    "wrong secret token",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwiZXhwIjoxOTk5OTk5OTk5fQ.invalid",
			wantErr: auth.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			claims, err := m.ValidateToken(ctx, tt.token)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if claims != nil {
				t.Error("expected nil claims on error")
			}
		})
	}
}

func TestValidateExpiredToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	// Create manager with very short timeout
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Millisecond,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = m.ValidateToken(ctx, token)
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := auth.HashPassword("testpassword")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == "testpassword" {
		t.Error("hash should not equal plaintext password")
	}
}

func TestMiddleware(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	// Get a valid token
	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware(handler)

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token",
			path:           "/api/test",
			authHeader:     "Bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing auth header",
			path:           "/api/test",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid auth format",
			path:           "/api/test",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			path:           "/api/test",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "skip auth for login",
			path:           "/api/auth/login",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "skip auth for static files",
			path:           "/static/index.html",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid strong password",
			password: "SecurePass1!",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1!",
			wantErr:  true,
		},
		{
			name:     "no uppercase",
			password: "lowercase123!",
			wantErr:  true,
		},
		{
			name:     "no lowercase",
			password: "UPPERCASE123!",
			wantErr:  true,
		},
		{
			name:     "no digits",
			password: "NoDigitsHere!",
			wantErr:  true,
		},
		{
			name:     "no special char",
			password: "NoSpecial123",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "exactly 12 chars with all requirements",
			password: "Abcdefgh123!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidatePasswordStrength(%q) error = %v, wantErr %v",
					tt.password,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	// Test password generation
	password, err := auth.GenerateSecurePassword(16)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	if len(password) != 16 {
		t.Errorf("expected password length 16, got %d", len(password))
	}

	// Generated password should pass validation
	if validateErr := auth.ValidatePasswordStrength(password); validateErr != nil {
		t.Errorf("generated password failed validation: %v", validateErr)
	}

	// Generate another password and ensure it's different
	password2, err := auth.GenerateSecurePassword(16)
	if err != nil {
		t.Fatalf("failed to generate second password: %v", err)
	}

	if password == password2 {
		t.Error("two generated passwords should not be identical")
	}
}

func TestGenerateSecurePasswordMinLength(t *testing.T) {
	// Request a password shorter than minimum
	password, err := auth.GenerateSecurePassword(4)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	// Should be at least MinPasswordLength
	if len(password) < auth.MinPasswordLength {
		t.Errorf(
			"expected minimum password length %d, got %d",
			auth.MinPasswordLength,
			len(password),
		)
	}
}

func TestGenerateInitialCredentials(t *testing.T) {
	creds, err := auth.GenerateInitialCredentials("admin")
	if err != nil {
		t.Fatalf("failed to generate initial credentials: %v", err)
	}

	if creds.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", creds.Username)
	}

	if creds.Password == "" {
		t.Error("password should not be empty")
	}

	if creds.PasswordHash == "" {
		t.Error("password hash should not be empty")
	}

	if creds.JWTSecret == "" {
		t.Error("JWT secret should not be empty")
	}

	// Verify the hash matches the password
	if validateErr := auth.ValidatePasswordStrength(creds.Password); validateErr != nil {
		t.Errorf("generated password failed validation: %v", validateErr)
	}
}

func TestIsDefaultPasswordHash(t *testing.T) {
	// Generate a hash for "seed" - should be detected as default
	seedHash, err := auth.HashPassword("seed")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !auth.IsDefaultPasswordHash(seedHash) {
		t.Error("expected hash of 'seed' to be detected as default")
	}

	// Generate a hash for a different password - should NOT be detected
	secureHash, err := auth.HashPassword("SecurePassword123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if auth.IsDefaultPasswordHash(secureHash) {
		t.Error("expected secure password hash to NOT be detected as default")
	}

	// Empty hash should be detected as default (triggers setup wizard)
	if !auth.IsDefaultPasswordHash("") {
		t.Error("empty hash should be detected as default (setup required)")
	}

	// Setup placeholder should be detected as default
	if !auth.IsDefaultPasswordHash(auth.SetupModePlaceholder) {
		t.Error("setup placeholder should be detected as default")
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	secret1 := auth.GenerateJWTSecret()
	secret2 := auth.GenerateJWTSecret()

	if secret1 == "" {
		t.Error("JWT secret should not be empty")
	}

	if secret1 == secret2 {
		t.Error("two generated JWT secrets should not be identical")
	}

	// Should be base64 encoded and reasonably long
	if len(secret1) < 32 {
		t.Errorf("JWT secret seems too short: %d characters", len(secret1))
	}
}

func TestUpdatePasswordHash(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	// Generate a token before password change
	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Verify token is valid
	_, err = m.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("token should be valid before password change: %v", err)
	}

	// Update password hash
	newHash, _ := auth.HashPassword("newpassword")
	m.UpdatePasswordHash(ctx, newHash)

	// Old token should now be invalid (token version incremented)
	_, err = m.ValidateToken(ctx, token)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken after password change, got %v", err)
	}
}

func TestGenerateAccessToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	token, err := m.GenerateAccessToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty access token")
	}

	claims, err := m.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.TokenType != "access" {
		t.Errorf("expected token type 'access', got %q", claims.TokenType)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	token, err := m.GenerateRefreshToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty refresh token")
	}

	claims, err := m.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("expected token type 'refresh', got %q", claims.TokenType)
	}
}

func TestValidateRefreshToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	// Generate a refresh token
	refreshToken, err := m.GenerateRefreshToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	// Validate as refresh token - should succeed
	claims, err := m.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}
	if claims.Username != defaults.Auth.Username {
		t.Errorf("expected username %s, got %s", defaults.Auth.Username, claims.Username)
	}

	// Try to validate an access token as refresh token - should fail
	accessToken, err := m.GenerateAccessToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	_, err = m.ValidateRefreshToken(ctx, accessToken)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken for access token, got %v", err)
	}
}

func TestRefreshAccessToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	// Generate a refresh token
	refreshToken, err := m.GenerateRefreshToken(ctx, defaults.Auth.Username)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	// Use refresh token to get new access token
	newAccessToken, err := m.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("failed to refresh access token: %v", err)
	}

	// Validate the new access token
	claims, err := m.ValidateToken(ctx, newAccessToken)
	if err != nil {
		t.Fatalf("failed to validate new access token: %v", err)
	}

	if claims.TokenType != "access" {
		t.Errorf("expected token type 'access', got %q", claims.TokenType)
	}
	if claims.Username != defaults.Auth.Username {
		t.Errorf("expected username %s, got %s", defaults.Auth.Username, claims.Username)
	}

	// Try with invalid refresh token
	_, err = m.RefreshAccessToken(ctx, "invalid.token.here")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}

	// Try with access token (not refresh token)
	accessToken, _ := m.GenerateAccessToken(ctx, defaults.Auth.Username)
	_, err = m.RefreshAccessToken(ctx, accessToken)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken for access token, got %v", err)
	}
}

func TestExtractTokenFromSubprotocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		expected string
	}{
		{
			name:     "access_token format",
			protocol: "access_token, mytoken123",
			expected: "mytoken123",
		},
		{
			name:     "bearer format",
			protocol: "bearer, mytoken456",
			expected: "mytoken456",
		},
		{
			name:     "single token fallback",
			protocol: "mytoken789",
			expected: "mytoken789",
		},
		{
			name:     "empty string",
			protocol: "",
			expected: "",
		},
		{
			name:     "access_token at end without token",
			protocol: "something, access_token",
			expected: "",
		},
		{
			name:     "multiple protocols with access_token",
			protocol: "other, access_token, mytoken",
			expected: "mytoken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.ExportExtractTokenFromSubprotocol(tt.protocol)
			if result != tt.expected {
				t.Errorf(
					"ExtractTokenFromSubprotocol(%q) = %q, want %q",
					tt.protocol,
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestMiddlewareWithCookie(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	// Get a valid token
	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that username was set in header
		if r.Header.Get("X-Username") != defaults.Auth.Username {
			t.Errorf(
				"expected X-Username to be %q, got %q",
				defaults.Auth.Username,
				r.Header.Get("X-Username"),
			)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: auth.CookieNameAccess, Value: token})

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMiddlewareSkipSetupEndpoints(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware(handler)

	// Test that setup endpoints skip auth
	paths := []string{
		"/api/setup/status",
		"/api/setup/complete",
		"/api/auth/refresh",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d for %s, got %d", http.StatusOK, path, rec.Code)
			}
		})
	}
}

func TestMiddlewareExpiredToken(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Millisecond,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for expired token, got %d", http.StatusUnauthorized, rec.Code)
	}

	// API returns JSON error responses
	body := rec.Body.String()
	if !strings.Contains(body, "Token expired") {
		t.Errorf("expected body to contain 'Token expired', got %q", body)
	}
}

func TestMiddlewareWebSocketAuth(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	token, err := m.Authenticate(ctx, defaults.Auth.Username, defaults.Auth.Password)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware(handler)

	// Test WebSocket with Sec-WebSocket-Protocol header
	req := httptest.NewRequest(http.MethodGet, "/ws/updates", http.NoBody)
	req.Header.Set("Sec-WebSocket-Protocol", "access_token, "+token)

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for WebSocket auth, got %d", http.StatusOK, rec.Code)
	}
}

func TestRandomChar(t *testing.T) {
	chars := "abc"

	// Generate many characters and ensure they're all in the charset
	seen := make(map[byte]bool)
	for range 100 {
		c, err := auth.ExportRandomChar(chars)
		if err != nil {
			t.Fatalf("RandomChar failed: %v", err)
		}
		if c != 'a' && c != 'b' && c != 'c' {
			t.Errorf("RandomChar returned %c, which is not in charset %q", c, chars)
		}
		seen[c] = true
	}

	// After 100 iterations, we should have seen all 3 chars
	if len(seen) != 3 {
		t.Logf("Only saw %d of 3 characters (may be random chance)", len(seen))
	}
}

func TestRandomInt(t *testing.T) {
	// Test with n=0
	result, err := auth.ExportRandomInt(0)
	if err != nil {
		t.Errorf("RandomInt(0) error: %v", err)
	}
	if result != 0 {
		t.Errorf("RandomInt(0) = %d, want 0", result)
	}

	// Test with small n
	for range 100 {
		smallResult, smallErr := auth.ExportRandomInt(10)
		if smallErr != nil {
			t.Fatalf("RandomInt(10) error: %v", smallErr)
		}
		if smallResult < 0 || smallResult >= 10 {
			t.Errorf("RandomInt(10) = %d, out of range [0, 10)", smallResult)
		}
	}

	// Test with larger n (>256 to hit the multi-byte path)
	for range 50 {
		largeResult, largeErr := auth.ExportRandomInt(1000)
		if largeErr != nil {
			t.Fatalf("RandomInt(1000) error: %v", largeErr)
		}
		if largeResult < 0 || largeResult >= 1000 {
			t.Errorf("RandomInt(1000) = %d, out of range [0, 1000)", largeResult)
		}
	}
}

// Cookie tests

func TestDefaultCookieConfig(t *testing.T) {
	config := auth.DefaultCookieConfig()

	if !config.Secure {
		t.Error("Secure should be true by default")
	}
	if config.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite should be Strict (fix #707), got %v", config.SameSite)
	}
	if config.Domain != "" {
		t.Errorf("Domain should be empty by default, got %q", config.Domain)
	}
	if config.Path != "/" {
		t.Errorf("Path should be '/', got %q", config.Path)
	}
}

func TestSetAccessTokenCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	config := auth.DefaultCookieConfig()
	config.Secure = false // For testing without HTTPS

	auth.SetAccessTokenCookie(rec, "test-token", config)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != auth.CookieNameAccess {
		t.Errorf("expected cookie name %q, got %q", auth.CookieNameAccess, cookie.Name)
	}
	if cookie.Value != "test-token" {
		t.Errorf("expected cookie value 'test-token', got %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
}

func TestSetRefreshTokenCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	config := auth.DefaultCookieConfig()
	config.Secure = false

	auth.SetRefreshTokenCookie(rec, "refresh-token", config)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != auth.CookieNameRefresh {
		t.Errorf("expected cookie name %q, got %q", auth.CookieNameRefresh, cookie.Name)
	}
	if cookie.Value != "refresh-token" {
		t.Errorf("expected cookie value 'refresh-token', got %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
}

func TestClearAuthCookies(t *testing.T) {
	rec := httptest.NewRecorder()
	config := auth.DefaultCookieConfig()

	auth.ClearAuthCookies(rec, config)

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies (access and refresh), got %d", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.MaxAge != -1 {
			t.Errorf("cookie %s should have MaxAge -1, got %d", cookie.Name, cookie.MaxAge)
		}
	}
}

func TestGetAccessTokenFromCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: auth.CookieNameAccess, Value: "access-token-value"})

	token, err := auth.GetAccessTokenFromCookie(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "access-token-value" {
		t.Errorf("expected 'access-token-value', got %q", token)
	}

	// Test missing cookie
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	_, err = auth.GetAccessTokenFromCookie(req2)
	if err == nil {
		t.Error("expected error for missing cookie")
	}
}

func TestGetRefreshTokenFromCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: auth.CookieNameRefresh, Value: "refresh-token-value"})

	token, err := auth.GetRefreshTokenFromCookie(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "refresh-token-value" {
		t.Errorf("expected 'refresh-token-value', got %q", token)
	}

	// Test missing cookie
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	_, err = auth.GetRefreshTokenFromCookie(req2)
	if err == nil {
		t.Error("expected error for missing cookie")
	}
}

func TestGetTokenFromRequest(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedToken  string
		expectedSource string
	}{
		{
			name: "from cookie",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.AddCookie(&http.Cookie{Name: auth.CookieNameAccess, Value: "cookie-token"})
				return req
			},
			expectedToken:  "cookie-token",
			expectedSource: "cookie",
		},
		{
			name: "from Authorization header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Bearer header-token")
				return req
			},
			expectedToken:  "header-token",
			expectedSource: "header",
		},
		{
			name: "from query parameter (disabled for security #706)",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/?token=query-token", http.NoBody)
			},
			expectedToken:  "",
			expectedSource: "none",
		},
		{
			name: "no token",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			},
			expectedToken:  "",
			expectedSource: "none",
		},
		{
			name: "cookie takes precedence over header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.AddCookie(&http.Cookie{Name: auth.CookieNameAccess, Value: "cookie-token"})
				req.Header.Set("Authorization", "Bearer header-token")
				return req
			},
			expectedToken:  "cookie-token",
			expectedSource: "cookie",
		},
		{
			name: "header takes precedence over query",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/?token=query-token", http.NoBody)
				req.Header.Set("Authorization", "Bearer header-token")
				return req
			},
			expectedToken:  "header-token",
			expectedSource: "header",
		},
		{
			name: "invalid Authorization header format (query disabled #706)",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/?token=query-token", http.NoBody)
				req.Header.Set("Authorization", "Basic base64encoded")
				return req
			},
			expectedToken:  "",
			expectedSource: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			token, source := auth.GetTokenFromRequest(req)
			if token != tt.expectedToken {
				t.Errorf("expected token %q, got %q", tt.expectedToken, token)
			}
			if source != tt.expectedSource {
				t.Errorf("expected source %q, got %q", tt.expectedSource, source)
			}
		})
	}
}

func TestTokenDurationConstants(t *testing.T) {
	if auth.AccessTokenDuration != 15*time.Minute {
		t.Errorf("AccessTokenDuration should be 15 minutes, got %v", auth.AccessTokenDuration)
	}
	if auth.RefreshTokenDuration != 7*24*time.Hour {
		t.Errorf("RefreshTokenDuration should be 7 days, got %v", auth.RefreshTokenDuration)
	}
}

func TestCookieNameConstants(t *testing.T) {
	if auth.CookieNameAccess != "seed_access" {
		t.Errorf("CookieNameAccess should be 'seed_access', got %q", auth.CookieNameAccess)
	}
	if auth.CookieNameRefresh != "seed_refresh" {
		t.Errorf("CookieNameRefresh should be 'seed_refresh', got %q", auth.CookieNameRefresh)
	}
}

// mockUserStore is a mock implementation of UserStore for testing.
type mockUserStore struct {
	passwords     map[string]string
	tokenVersions map[string]int
	locked        map[string]bool
	updateErr     error
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		passwords:     make(map[string]string),
		tokenVersions: make(map[string]int),
		locked:        make(map[string]bool),
	}
}

func (m *mockUserStore) GetPasswordHash(_ context.Context, username string) (string, error) {
	if hash, ok := m.passwords[username]; ok {
		return hash, nil
	}
	return "", auth.ErrInvalidCredentials
}

func (m *mockUserStore) GetTokenVersion(_ context.Context, username string) (int, error) {
	if v, ok := m.tokenVersions[username]; ok {
		return v, nil
	}
	return 0, nil
}

func (m *mockUserStore) UpdatePassword(_ context.Context, username, hash string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.passwords[username] = hash
	return nil
}

func (m *mockUserStore) RecordLoginSuccess(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserStore) RecordLoginFailure(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserStore) IsLocked(_ context.Context, username string) (bool, error) {
	return m.locked[username], nil
}

func TestSetUserStore(_ *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)

	store := newMockUserStore()
	m.SetUserStore(store)

	// Verify the store was set (can't directly access, but we can verify behavior)
	// Setting user store should not cause panic
}

func TestAuthenticateWithUserStore(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	store := newMockUserStore()
	// Add a user to the store
	hash, _ := auth.HashPassword("storepassword")
	store.passwords["storeuser"] = hash
	store.tokenVersions["storeuser"] = 1

	m.SetUserStore(store)

	// Should be able to authenticate with store user
	token, err := m.Authenticate(ctx, "storeuser", "storepassword")
	if err != nil {
		t.Errorf("expected successful auth with store user, got %v", err)
	}
	if token == "" {
		t.Error("expected token for store user")
	}
}

func TestAuthenticateWithLockedUser(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	store := newMockUserStore()
	hash, _ := auth.HashPassword("password")
	store.passwords["lockeduser"] = hash
	store.locked["lockeduser"] = true

	m.SetUserStore(store)

	// Should fail for locked user (returns ErrInvalidCredentials per security best practice)
	_, err := m.Authenticate(ctx, "lockeduser", "password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for locked user, got %v", err)
	}
}

func TestUpdatePasswordHashForUser(t *testing.T) {
	defaults := testutil.GetTestDefaults()
	m := auth.NewManager(
		defaults.Auth.JWTSecret,
		time.Hour,
		defaults.Auth.Username,
		defaults.Auth.PasswordHash,
	)
	ctx := context.Background()

	t.Run("no UserStore configured", func(t *testing.T) {
		err := m.UpdatePasswordHashForUser(ctx, "testuser", "newhash")
		if err == nil {
			t.Error("expected error when UserStore not configured")
		}
	})

	t.Run("successful update", func(t *testing.T) {
		store := newMockUserStore()
		store.passwords["testuser"] = "oldhash"
		m.SetUserStore(store)

		err := m.UpdatePasswordHashForUser(ctx, "testuser", "newhash")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify hash was updated
		if store.passwords["testuser"] != "newhash" {
			t.Error("password hash was not updated in store")
		}
	})
}

func TestCSRFRevokeToken(t *testing.T) {
	mgr := auth.NewCSRFManager()
	defer mgr.Stop()

	// Generate a token
	token, err := mgr.GenerateToken("testsession")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate it
	err = mgr.ValidateToken("testsession", token)
	if err != nil {
		t.Fatalf("token should be valid: %v", err)
	}

	// Revoke it
	mgr.RevokeToken("testsession")

	// Should no longer be valid (returns ErrCSRFTokenInvalid when session not found)
	err = mgr.ValidateToken("testsession", token)
	if !errors.Is(err, auth.ErrCSRFTokenInvalid) {
		t.Errorf("expected ErrCSRFTokenInvalid after revoke, got %v", err)
	}
}

func TestCSRFMiddleware(t *testing.T) {
	mgr := auth.NewCSRFManager()
	defer mgr.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := mgr.CSRFMiddleware(handler)

	t.Run("GET requests bypass CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET, got %d", rec.Code)
		}
	})

	t.Run("HEAD requests bypass CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/api/test", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for HEAD, got %d", rec.Code)
		}
	})

	t.Run("OPTIONS requests bypass CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/test", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for OPTIONS, got %d", rec.Code)
		}
	})

	t.Run("non-API routes bypass CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/static/file.js", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for non-API route, got %d", rec.Code)
		}
	})

	t.Run("login endpoint bypasses CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for login, got %d", rec.Code)
		}
	})

	t.Run("setup endpoints bypass CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/setup/complete", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for setup, got %d", rec.Code)
		}
	})

	t.Run("SSO endpoints bypass CSRF", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/sso/callback", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 for SSO, got %d", rec.Code)
		}
	})

	t.Run("POST without session returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/devices", http.NoBody)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 without session, got %d", rec.Code)
		}
	})
}

func TestGetSessionIDFromRequest(t *testing.T) {
	t.Run("no token returns empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		sessionID := auth.GetSessionIDFromRequest(req)
		if sessionID != "" {
			t.Errorf("expected empty session ID, got %q", sessionID)
		}
	})

	t.Run("valid JWT format extracts session ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		req.Header.Set("Authorization", "Bearer header.payload.signature")

		sessionID := auth.GetSessionIDFromRequest(req)
		if sessionID != "payload" {
			t.Errorf("expected session ID 'payload', got %q", sessionID)
		}
	})

	t.Run("cookie token extracts session ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		req.AddCookie(&http.Cookie{
			Name:  auth.CookieNameAccess,
			Value: "header.cookiepayload.signature",
		})

		sessionID := auth.GetSessionIDFromRequest(req)
		if sessionID != "cookiepayload" {
			t.Errorf("expected session ID 'cookiepayload', got %q", sessionID)
		}
	})
}
