// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// MSN signed-license token format (v1), shared by convention across Seed,
// Stem, and NIAC and produced by the canonical keygen tool. See ADR-0019.
//
//	<scheme>.<base64url(payload)>.<base64url(signature)>
//
// where:
//   - scheme is the literal "MSN1" (format + version tag),
//   - payload is the canonical JSON encoding of [Payload],
//   - signature is the 64-byte Ed25519 signature over the *exact* payload
//     bytes (the bytes that were base64url-encoded, not a re-marshaling).
//
// Verification is fully offline: the binary embeds only the Ed25519 PUBLIC
// key; the private key never leaves the keygen tool. A token cannot be forged
// or altered without the private key, which replaces the old rotor cipher +
// self-checksum scheme whose generator shipped inside every binary.
const (
	tokenScheme    = "MSN1"
	tokenSeparator = "."
	tokenParts     = 3

	// payloadVersion is the current [Payload] schema version. A future
	// breaking change bumps this and the verifier rejects unknown versions.
	payloadVersion = 1
)

var (
	errBadToken     = errors.New("malformed license token")
	errBadScheme    = errors.New("unrecognized license token scheme")
	errBadSignature = errors.New("license signature verification failed")
	errBadPayload   = errors.New("license payload is not valid")
)

// Payload is the signed content of a license token. The JSON field names are
// the cross-product wire contract: every MSN product and the keygen tool must
// agree on them. Field *order* is irrelevant — verification is over the raw
// signed bytes, and unmarshaling is order-independent.
type Payload struct {
	Version    int    `json:"v"`                    // schema version (payloadVersion)
	Product    string `json:"product"`              // "stem" | "niac" | "seed"
	Code       string `json:"code"`                 // product code, e.g. "4002"
	Serial     string `json:"serial"`               // unique per license
	Tier       int    `json:"tier"`                 // numeric tier (wire value)
	MaxDevices int    `json:"maxDevices,omitempty"` // 0 ⇒ product default
	IssuedAt   int64  `json:"iat"`                  // unix seconds
	ExpiresAt  int64  `json:"exp,omitempty"`        // unix seconds; 0 ⇒ perpetual
}

// Verifier validates signed license tokens against one Ed25519 public key.
// It holds no private material and is safe for concurrent use.
type Verifier struct {
	pub ed25519.PublicKey
}

// NewVerifier returns a Verifier for the given Ed25519 public key. Exported so
// tests can verify against an ephemeral test key without touching the embedded
// production key.
func NewVerifier(pub ed25519.PublicKey) *Verifier {
	return &Verifier{pub: pub}
}

// mustVerifierFromB64 decodes a standard-base64 Ed25519 public key and returns
// a Verifier. It panics on a malformed or wrong-length key: the only caller is
// the package-level production key constant, so a bad value is a build-time bug.
func mustVerifierFromB64(b64 string) *Verifier {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic("license: invalid embedded public key encoding: " + err.Error())
	}
	if len(raw) != ed25519.PublicKeySize {
		panic("license: embedded public key has wrong length")
	}
	return &Verifier{pub: ed25519.PublicKey(raw)}
}

// parseAndVerify splits a token, verifies its Ed25519 signature against v.pub,
// and returns the decoded payload. It performs the cryptographic check before
// any structural interpretation so an attacker-supplied payload is never
// trusted.
func (v *Verifier) parseAndVerify(token string) (*Payload, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, tokenSeparator)
	if len(parts) != tokenParts {
		return nil, errBadToken
	}
	if parts[0] != tokenScheme {
		return nil, errBadScheme
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errBadToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errBadToken
	}
	if len(sig) != ed25519.SignatureSize {
		return nil, errBadSignature
	}

	if !ed25519.Verify(v.pub, payloadBytes, sig) {
		return nil, errBadSignature
	}

	p := &Payload{}
	if unmarshalErr := json.Unmarshal(payloadBytes, p); unmarshalErr != nil {
		return nil, errBadPayload
	}
	if p.Version != payloadVersion {
		return nil, errBadPayload
	}
	return p, nil
}
