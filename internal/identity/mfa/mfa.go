// Package mfa owns the second-factor entities of the identity surface
// (ADR-0024): the registered WebAuthn authenticator and its not-found
// sentinel. The database persists and returns these types, so internal/api
// handlers build and read credentials without importing the database
// (seed#2750).
package mfa

import (
	"errors"
	"time"
)

// ErrCredentialNotFound is returned when a WebAuthn credential lookup
// finds no matching row.
var ErrCredentialNotFound = errors.New("webauthn credential not found")

// WebAuthnCredential is a single registered authenticator. A user may have
// several.
type WebAuthnCredential struct {
	ID              int64
	UserID          int64
	CredentialID    []byte // raw credential ID bytes (NOT base64-encoded)
	PublicKey       []byte // COSE-encoded public key from the authenticator
	SignCount       uint32
	AttestationType string
	Transports      string // comma-joined transport hints ("usb,nfc",...)
	AAGUID          []byte
	CreatedAt       time.Time
	LastUsedAt      *time.Time
}
