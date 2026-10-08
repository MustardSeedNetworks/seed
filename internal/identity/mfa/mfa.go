// Package mfa owns the second-factor entities of the identity surface
// (ADR-0024): the registered WebAuthn authenticator, its not-found sentinel,
// and the Store port the MFA handlers persist factors through. The database
// implements Store, wired by the composition root, so internal/api handlers
// enrol and verify factors without reaching the database (seed#2750).
package mfa

import (
	"context"
	"errors"
	"time"
)

// Store persists a user's second factors: the TOTP secret and enabled flag on
// the user, and the user's registered WebAuthn credentials. Every method but
// Available assumes Available reported true.
type Store interface {
	Available() bool
	SetTOTPSecret(ctx context.Context, username, secret string) error
	EnableTOTP(ctx context.Context, username string) error
	DisableTOTP(ctx context.Context, username string) error
	GetTOTP(ctx context.Context, username string) (secret string, enabled bool, err error)
	AddWebAuthnCredential(ctx context.Context, userID int64, cred WebAuthnCredential) (int64, error)
	ListWebAuthnCredentials(ctx context.Context, userID int64) ([]WebAuthnCredential, error)
	UpdateWebAuthnSignCount(ctx context.Context, credentialID []byte, signCount uint32) error
}

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
