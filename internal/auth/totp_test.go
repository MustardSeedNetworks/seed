// totp_test.go — unit tests for the TOTP MFA helpers.
package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/auth"
)

// codeAt computes the TOTP code that pquerna/otp would derive for the
// given secret at the given time. We use this in the tests rather than
// hard-coding codes so the tests stay deterministic across time zones
// and clock skews on CI runners.
func codeAt(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)
	return code
}

// TestGenerateTOTPSecret_RoundTrip verifies that a freshly generated
// secret can be used to derive a code that VerifyTOTP accepts.
func TestGenerateTOTPSecret_RoundTrip(t *testing.T) {
	setup, err := auth.GenerateTOTPSecret("alice@example.com", "")
	require.NoError(t, err)
	require.NotNil(t, setup)
	assert.NotEmpty(t, setup.Secret, "secret should be non-empty")
	assert.True(
		t,
		strings.HasPrefix(setup.ProvisioningURI, "otpauth://"),
		"expected otpauth:// URI, got %q", setup.ProvisioningURI,
	)
	// Issuer defaults to "Seed" — case-sensitive substring check is
	// enough since the URI is URL-encoded.
	assert.Contains(t, setup.ProvisioningURI, "Seed")
	assert.NotEmpty(t, setup.QRCodePNG, "QR code PNG must be non-empty")
	// PNG magic bytes — sanity check we didn't return a stub.
	assert.Equal(t,
		[]byte{0x89, 0x50, 0x4e, 0x47}, setup.QRCodePNG[:4],
		"QR code must start with PNG magic bytes")

	code := codeAt(t, setup.Secret, time.Now())
	ok, verifyErr := auth.VerifyTOTP(setup.Secret, code)
	require.NoError(t, verifyErr)
	assert.True(t, ok, "freshly derived code must verify")
}

// TestVerifyTOTP_WrongCodeReturnsFalse confirms that a syntactically
// valid but incorrect code returns (false, nil) — not an error.
func TestVerifyTOTP_WrongCodeReturnsFalse(t *testing.T) {
	setup, err := auth.GenerateTOTPSecret("bob@example.com", "Seed")
	require.NoError(t, err)

	ok, err := auth.VerifyTOTP(setup.Secret, "000000")
	require.NoError(t, err)
	assert.False(t, ok, "all-zero code should not verify")
}

// TestVerifyTOTP_EmptyInputs ensures that empty secret/code return the
// sentinel ErrInvalidTOTPInput error so callers can distinguish input
// validation failures from auth failures.
func TestVerifyTOTP_EmptyInputs(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		code   string
	}{
		{"empty secret", "", "123456"},
		{"empty code", "JBSWY3DPEHPK3PXP", ""},
		{"both empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := auth.VerifyTOTP(tt.secret, tt.code)
			assert.False(t, ok)
			assert.ErrorIs(t, err, auth.ErrInvalidTOTPInput)
		})
	}
}

// TestVerifyTOTP_PastCodeOutsideSkew confirms that a code computed for
// a time more than one period in the past does NOT verify. The skew
// window is ±1 period (30s), so a code from 5 minutes ago is well
// outside it. We use VerifyTOTPAt so the test pins both the code's
// timestamp and the verifier's "now" deterministically.
func TestVerifyTOTP_PastCodeOutsideSkew(t *testing.T) {
	setup, err := auth.GenerateTOTPSecret("carol@example.com", "Seed")
	require.NoError(t, err)

	now := time.Date(2026, time.May, 19, 12, 0, 0, 0, time.UTC)
	oldCode := codeAt(t, setup.Secret, now.Add(-5*time.Minute))
	ok, err := auth.VerifyTOTPAt(setup.Secret, oldCode, now)
	require.NoError(t, err)
	assert.False(t, ok, "code from 5 minutes ago must not verify")
}

// TestVerifyTOTP_AdjacentPeriodVerifies sanity-checks that the skew
// window genuinely accepts the immediately preceding period (clock
// drift tolerance).
func TestVerifyTOTP_AdjacentPeriodVerifies(t *testing.T) {
	setup, err := auth.GenerateTOTPSecret("dave@example.com", "Seed")
	require.NoError(t, err)

	now := time.Date(2026, time.May, 19, 12, 0, 0, 0, time.UTC)
	// Code computed 25 seconds ago is still inside the previous period
	// window (±30s skew).
	driftCode := codeAt(t, setup.Secret, now.Add(-25*time.Second))
	ok, err := auth.VerifyTOTPAt(setup.Secret, driftCode, now)
	require.NoError(t, err)
	assert.True(t, ok, "code within ±30s skew should verify")
}

// TestVerifyTOTP_MalformedSecret asserts that a syntactically invalid
// base32 secret returns ErrInvalidTOTPInput so the caller can log/audit
// it as a corrupt configuration rather than a wrong code.
func TestVerifyTOTP_MalformedSecret(t *testing.T) {
	ok, err := auth.VerifyTOTP("not!valid!base32!!", "123456")
	assert.False(t, ok)
	assert.ErrorIs(t, err, auth.ErrInvalidTOTPInput)
}

// TestVerifyTOTP_WrongLengthCode confirms that a code whose length is
// not six digits is silently rejected (returns false, nil) — we don't
// want to leak "code length wrong" vs "code value wrong".
func TestVerifyTOTP_WrongLengthCode(t *testing.T) {
	setup, err := auth.GenerateTOTPSecret("erin@example.com", "Seed")
	require.NoError(t, err)

	ok, err := auth.VerifyTOTP(setup.Secret, "12345")
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = auth.VerifyTOTP(setup.Secret, "1234567")
	require.NoError(t, err)
	assert.False(t, ok)
}
