// hashing.go — password hashing primitives for package auth.
//
// This file implements Argon2id-based password hashing per RFC 9106 and
// provides a transparent verification path that also accepts legacy
// bcrypt hashes (cost 12) so existing users can be migrated on first
// successful login.
//
// The encoded hash string follows the PHC string convention so the
// algorithm + parameters are self-describing:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<base64salt>$<base64hash>
//
// Bcrypt hashes ($2a$, $2y$, $2b$) are also recognised by VerifyPassword;
// callers should re-hash with Argon2id when needsRehash is reported.

package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Argon2id parameters per RFC 9106 recommendations.
const (
	argon2Memory  uint32 = 64 * 1024 // 64 MiB
	argon2Time    uint32 = 3
	argon2Threads uint8  = 4
	argon2SaltLen uint32 = 16
	argon2KeyLen  uint32 = 32

	argon2IDPrefix  = "$argon2id$"
	argon2Version   = argon2.Version
	bcryptPrefix2a  = "$2a$"
	bcryptPrefix2b  = "$2b$"
	bcryptPrefix2y  = "$2y$"
	phcExpectParts  = 6 // "", "argon2id", "v=19", "m=...,t=...,p=...", "<b64salt>", "<b64hash>"
	phcParamSegment = 3
	phcSaltSegment  = 4
	phcHashSegment  = 5
)

// HashAlgorithm names a stored hash's underlying algorithm.
type HashAlgorithm string

const (
	// HashAlgorithmArgon2id indicates a PHC-encoded Argon2id hash.
	HashAlgorithmArgon2id HashAlgorithm = "argon2id"
	// HashAlgorithmBcrypt indicates a legacy bcrypt hash (2a/2b/2y).
	HashAlgorithmBcrypt HashAlgorithm = "bcrypt"
	// HashAlgorithmUnknown indicates a hash whose prefix is not recognised.
	HashAlgorithmUnknown HashAlgorithm = "unknown"
)

var (
	// ErrUnsupportedHash is returned when a stored hash cannot be parsed
	// or does not match a known prefix.
	ErrUnsupportedHash = errors.New("unsupported password hash format")

	// ErrInvalidArgon2Params is returned when an Argon2id encoded string
	// has malformed parameters.
	ErrInvalidArgon2Params = errors.New("invalid argon2id parameters")
)

// HashPassword hashes the given password using Argon2id with the
// project's standard parameters and returns a PHC-encoded string.
//
// The function name and signature are unchanged from the previous
// bcrypt implementation; only the underlying algorithm has changed.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if err := cryptoRandRead(salt, "HashPassword.salt"); err != nil {
		return "", fmt.Errorf("argon2id salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, nil
}

// DetectHashAlgorithm returns the algorithm a stored hash uses based on
// its prefix. Unknown/empty hashes yield HashAlgorithmUnknown.
func DetectHashAlgorithm(hash string) HashAlgorithm {
	switch {
	case strings.HasPrefix(hash, argon2IDPrefix):
		return HashAlgorithmArgon2id
	case strings.HasPrefix(hash, bcryptPrefix2a),
		strings.HasPrefix(hash, bcryptPrefix2b),
		strings.HasPrefix(hash, bcryptPrefix2y):
		return HashAlgorithmBcrypt
	default:
		return HashAlgorithmUnknown
	}
}

// VerifyPassword checks the given password against the stored hash.
//
// Returns:
//   - matched: true when the password matches the hash.
//   - needsRehash: true when the hash is a legacy bcrypt hash and the
//     caller should re-hash + persist with Argon2id.
//   - err: non-nil only when the hash format is not recognised
//     (ErrUnsupportedHash) or the Argon2id parameters are malformed.
//
// Wrong-password results return (false, false, nil); callers should
// treat that as ErrInvalidCredentials.
func VerifyPassword(hash, password string) (bool, bool, error) {
	switch DetectHashAlgorithm(hash) {
	case HashAlgorithmArgon2id:
		ok, vErr := verifyArgon2id(hash, password)
		if vErr != nil {
			return false, false, vErr
		}
		return ok, false, nil

	case HashAlgorithmBcrypt:
		cmpErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if errors.Is(cmpErr, bcrypt.ErrMismatchedHashAndPassword) {
			// Wrong-password is not an error condition; callers treat
			// (false, false, nil) as ErrInvalidCredentials upstream.
			return false, false, nil
		}
		if cmpErr != nil {
			return false, false, fmt.Errorf("bcrypt compare: %w", cmpErr)
		}
		return true, true, nil

	case HashAlgorithmUnknown:
		return false, false, ErrUnsupportedHash

	default:
		return false, false, ErrUnsupportedHash
	}
}

// verifyArgon2id parses a PHC-encoded Argon2id string and verifies the
// given password in constant time.
func verifyArgon2id(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != phcExpectParts {
		return false, ErrInvalidArgon2Params
	}

	var version int
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil {
		return false, ErrInvalidArgon2Params
	}
	if version != argon2Version {
		return false, ErrInvalidArgon2Params
	}

	var memory, time uint32
	var threads uint8
	if _, scanErr := fmt.Sscanf(
		parts[phcParamSegment],
		"m=%d,t=%d,p=%d",
		&memory, &time, &threads,
	); scanErr != nil {
		return false, ErrInvalidArgon2Params
	}

	salt, decodeErr := base64.RawStdEncoding.DecodeString(parts[phcSaltSegment])
	if decodeErr != nil {
		return false, ErrInvalidArgon2Params
	}
	expected, decodeErr := base64.RawStdEncoding.DecodeString(parts[phcHashSegment])
	if decodeErr != nil {
		return false, ErrInvalidArgon2Params
	}

	// #nosec G115 -- len(expected) is bounded by argon2KeyLen=32 in practice; safe for uint32.
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
