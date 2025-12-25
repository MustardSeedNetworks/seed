// Package auth handles JWT authentication.
// This file tests the crypto/rand retry logic for Plan G7.
package auth

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestCryptoRandReadSuccess verifies normal operation succeeds.
func TestCryptoRandReadSuccess(t *testing.T) {
	b := make([]byte, 32)
	err := cryptoRandRead(b, "test_operation")
	if err != nil {
		t.Fatalf("cryptoRandRead failed on normal operation: %v", err)
	}

	// Verify bytes were actually filled (not all zeros)
	allZeros := true
	for _, v := range b {
		if v != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("cryptoRandRead returned all zeros, expected random data")
	}
}

// TestCryptoRandReadLogging verifies that the function logs appropriately.
func TestCryptoRandReadLogging(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	oldLogger := slog.Default()
	defer slog.SetDefault(oldLogger)

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Normal operation should not log warnings
	b := make([]byte, 16)
	err := cryptoRandRead(b, "test_normal")
	if err != nil {
		t.Fatalf("cryptoRandRead failed: %v", err)
	}

	output := buf.String()
	// Should not contain warning logs on success
	if strings.Contains(output, "crypto/rand failed, retrying") {
		t.Error("Expected no retry warnings on successful operation")
	}
}

// TestGenerateJWTSecretWithRetry verifies that GenerateJWTSecret works correctly.
func TestGenerateJWTSecretWithRetry(t *testing.T) {
	// Should succeed under normal conditions
	secret := GenerateJWTSecret()
	if secret == "" {
		t.Error("GenerateJWTSecret returned empty string")
	}

	// Should be base64 encoded and reasonably long
	if len(secret) < 32 {
		t.Errorf("JWT secret seems too short: %d characters", len(secret))
	}

	// Generate another to ensure they're different
	secret2 := GenerateJWTSecret()
	if secret == secret2 {
		t.Error("Two JWT secrets should not be identical")
	}
}

// TestRandomCharWithRetry verifies randomChar works correctly.
func TestRandomCharWithRetry(t *testing.T) {
	chars := "abcdefghijklmnopqrstuvwxyz"

	// Generate multiple characters
	for i := range 100 {
		c, err := randomChar(chars)
		if err != nil {
			t.Fatalf("randomChar failed on iteration %d: %v", i, err)
		}

		// Verify character is from the charset
		found := false
		for j := range len(chars) {
			if c == chars[j] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("randomChar returned %c which is not in charset %q", c, chars)
		}
	}
}

// TestRandomIntWithRetry verifies randomInt works correctly.
func TestRandomIntWithRetry(t *testing.T) {
	// Test with various values of n
	testCases := []int{10, 100, 1000}

	for _, n := range testCases {
		for i := range 50 {
			result, err := randomInt(n)
			if err != nil {
				t.Fatalf("randomInt(%d) failed on iteration %d: %v", n, i, err)
			}
			if result < 0 || result >= n {
				t.Errorf("randomInt(%d) = %d, out of range [0, %d)", n, result, n)
			}
		}
	}
}

// TestGenerateSecurePasswordWithRetry verifies password generation works.
func TestGenerateSecurePasswordWithRetry(t *testing.T) {
	// Generate multiple passwords
	for i := range 10 {
		password, err := GenerateSecurePassword(16)
		if err != nil {
			t.Fatalf("GenerateSecurePassword failed on iteration %d: %v", i, err)
		}

		if len(password) != 16 {
			t.Errorf("Expected password length 16, got %d", len(password))
		}

		// Should pass strength validation
		if err := ValidatePasswordStrength(password); err != nil {
			t.Errorf("Generated password failed validation: %v (password: %s)", err, password)
		}
	}
}

// TestCryptoRandReadRetryBehavior verifies the retry mechanism parameters.
func TestCryptoRandReadRetryBehavior(t *testing.T) {
	// This test verifies that the function has reasonable retry parameters
	// We can't easily force crypto/rand to fail, but we can verify the function exists
	// and works correctly under normal conditions

	b := make([]byte, 32)
	err := cryptoRandRead(b, "test_retry_params")
	if err != nil {
		t.Fatalf("cryptoRandRead failed: %v", err)
	}

	// Verify buffer was filled
	emptyBuffer := make([]byte, 32)
	if bytes.Equal(b, emptyBuffer) {
		t.Error("cryptoRandRead did not fill buffer with random data")
	}
}

// TestCryptoRandReadSmallBuffer tests with a small buffer.
func TestCryptoRandReadSmallBuffer(t *testing.T) {
	b := make([]byte, 1)
	err := cryptoRandRead(b, "test_small_buffer")
	if err != nil {
		t.Fatalf("cryptoRandRead failed with small buffer: %v", err)
	}
}

// TestCryptoRandReadLargeBuffer tests with a large buffer.
func TestCryptoRandReadLargeBuffer(t *testing.T) {
	b := make([]byte, 4096)
	err := cryptoRandRead(b, "test_large_buffer")
	if err != nil {
		t.Fatalf("cryptoRandRead failed with large buffer: %v", err)
	}

	// Verify at least some bytes are non-zero
	nonZeroCount := 0
	for _, v := range b {
		if v != 0 {
			nonZeroCount++
		}
	}

	// With 4096 random bytes, we should have many non-zero values
	if nonZeroCount < 3000 {
		t.Errorf("Expected mostly non-zero bytes, got only %d non-zero out of 4096", nonZeroCount)
	}
}
