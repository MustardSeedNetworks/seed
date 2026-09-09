// password_generate.go carries secure password and initial-credential
// generation for package auth. What a password must satisfy to be accepted is
// password_policy.go's business, not this file's.

package auth

import "errors"

// Constants for unbiased random selection in password generation.
const (
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

// randomChar selects an unbiased random character from the given charset.
// Uses rejection sampling to avoid modulo bias (fixes #517).
// Fixes G7: Uses cryptoRandRead for retry logic on crypto/rand failures.
func randomChar(chars string) (byte, error) {
	if len(chars) == 0 || len(chars) > maxByteValue {
		return 0, errors.New("chars must be 1-255 bytes")
	}
	//nolint:gosec // G115: bounded by the len check above; gosec can't follow the guard
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
