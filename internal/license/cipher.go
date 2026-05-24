// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"strings"
)

// Rotor constants for cipher operations.
const (
	rotorModulus      = 36 // 10 digits + 26 letters.
	digitModulus      = 10
	letterModulus     = 26
	alphanumericSplit = 10 // Values 0-9 are digits, 10-35 are letters.
	byteMask          = 0xFF
)

// MSN-specific rotor tables (unique to Mustard Seed Networks products).
// These provide the substitution mapping for license key encoding/decoding.

func baseRotor10() [10]byte {
	return [10]byte{7, 2, 9, 0, 5, 8, 1, 6, 3, 4}
}

func baseRotor26() [26]byte {
	return [26]byte{
		19, 3, 24, 7, 12, 0, 21, 15, 8, 25,
		2, 17, 10, 5, 22, 13, 1, 18, 6, 11,
		23, 4, 16, 9, 20, 14,
	}
}

// InitRotors is retained for backwards compatibility but is now a no-op.
func InitRotors() {}

// RotorCipher provides Enigma-style encoding/decoding.
type RotorCipher struct {
	position   int // Current rotor position (advances with each character).
	rotor10    [10]byte
	rotor26    [26]byte
	rotor10Inv [10]byte
	rotor26Inv [26]byte
}

// NewRotorCipher creates a new cipher with the given starting position.
func NewRotorCipher(startPosition int) *RotorCipher {
	rotor10 := baseRotor10()
	rotor26 := baseRotor26()

	var rotor10Inv [10]byte
	var rotor26Inv [26]byte

	for i, v := range rotor10 {
		rotor10Inv[v] = byte(i)
	}
	for i, v := range rotor26 {
		rotor26Inv[v] = byte(i)
	}

	position := startPosition % rotorModulus
	if position < 0 {
		position += rotorModulus
	}

	return &RotorCipher{
		position:   position,
		rotor10:    rotor10,
		rotor26:    rotor26,
		rotor10Inv: rotor10Inv,
		rotor26Inv: rotor26Inv,
	}
}

// Encode encodes a single character through the rotor.
func (rc *RotorCipher) Encode(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		// Digit encoding.
		idx := (int(c-'0') + rc.position) % digitModulus
		encoded := rc.rotor10[idx]
		rc.position = (rc.position + 1) % rotorModulus
		return '0' + encoded
	case c >= 'A' && c <= 'Z':
		// Letter encoding.
		idx := int(c - 'A')
		idx = (idx + rc.position) % letterModulus
		encoded := rc.rotor26[idx]
		rc.position = (rc.position + 1) % rotorModulus
		return 'A' + encoded
	case c >= 'a' && c <= 'z':
		// Lowercase - convert to upper, encode, keep case.
		idx := int(c - 'a')
		idx = (idx + rc.position) % letterModulus
		encoded := rc.rotor26[idx]
		rc.position = (rc.position + 1) % rotorModulus
		return 'a' + encoded
	default:
		// Non-alphanumeric passes through unchanged.
		return c
	}
}

// shiftByMod returns a non-negative value in [0, modulus). idx and modulus
// are int because rc.position is int; the result fits in byte because
// modulus is a small literal constant (10 or 26).
func shiftByMod(idx, position, modulus int) byte {
	v := (idx - position%modulus + modulus) % modulus
	// Mask to byte: modular arithmetic above guarantees v ∈ [0, modulus)
	// where modulus is 10 or 26, both well within byte range. The mask
	// proves the bound to the type checker without a branch.
	return byte(v & byteMask)
}

// Decode decodes a single character through the inverse rotor.
func (rc *RotorCipher) Decode(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		// Digit decoding.
		shift := shiftByMod(int(rc.rotor10Inv[int(c-'0')]), rc.position, digitModulus)
		rc.position = (rc.position + 1) % rotorModulus
		return '0' + shift
	case c >= 'A' && c <= 'Z':
		// Letter decoding.
		shift := shiftByMod(int(rc.rotor26Inv[int(c-'A')]), rc.position, letterModulus)
		rc.position = (rc.position + 1) % rotorModulus
		return 'A' + shift
	case c >= 'a' && c <= 'z':
		// Lowercase decoding.
		shift := shiftByMod(int(rc.rotor26Inv[int(c-'a')]), rc.position, letterModulus)
		rc.position = (rc.position + 1) % rotorModulus
		return 'a' + shift
	default:
		return c
	}
}

// EncodeString encodes an entire string.
func (rc *RotorCipher) EncodeString(s string) string {
	result := make([]byte, len(s))
	for i := range len(s) {
		result[i] = rc.Encode(s[i])
	}
	return string(result)
}

// DecodeString decodes an entire string.
func (rc *RotorCipher) DecodeString(s string) string {
	result := make([]byte, len(s))
	for i := range len(s) {
		result[i] = rc.Decode(s[i])
	}
	return string(result)
}

// CalculateChecksum generates a 2-character checksum for a string.
// Uses a simple polynomial hash that's quick to compute offline.
func CalculateChecksum(s string) string {
	s = strings.ToUpper(s)
	var sum1, sum2 int

	for i, c := range s {
		val := 0
		if c >= '0' && c <= '9' {
			val = int(c - '0')
		} else if c >= 'A' && c <= 'Z' {
			val = int(c-'A') + alphanumericSplit
		}

		// Polynomial accumulation with position weighting.
		sum1 = (sum1 + val*(i+1)) % rotorModulus
		sum2 = (sum2 + val*val + i) % rotorModulus
	}

	// Convert to alphanumeric.
	c1 := toAlphanumeric(sum1)
	c2 := toAlphanumeric(sum2)

	return string([]byte{c1, c2})
}

// ValidateChecksum checks if the last 2 characters are a valid checksum.
func ValidateChecksum(s string) bool {
	const minLength = 3
	if len(s) < minLength {
		return false
	}
	s = strings.ToUpper(s)
	payload := s[:len(s)-2]
	checksum := s[len(s)-2:]
	return CalculateChecksum(payload) == checksum
}

// toAlphanumeric converts a value 0-35 to 0-9 or A-Z.
func toAlphanumeric(val int) byte {
	// Defensive: by construction val is always in [0, alphanumericSplit+25],
	// but an explicit guard keeps gosec G115 happy and prevents panics from
	// hypothetical caller bugs.
	if val < 0 || val >= alphanumericSplit+26 {
		return '?'
	}
	if val < alphanumericSplit {
		return '0' + byte(val)
	}
	return 'A' + byte(val-alphanumericSplit)
}

// fromAlphanumeric converts 0-9 or A-Z to a value 0-35.
// Currently unused but kept for future license parsing.
func fromAlphanumeric(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'A' && c <= 'Z' {
		return int(c-'A') + alphanumericSplit
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + alphanumericSplit
	}
	return 0
}
