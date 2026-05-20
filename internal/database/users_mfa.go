// users_mfa.go — TOTP and WebAuthn persistence for the users table.
//
// Wave 3 (task #85). The TOTP columns were added by migration #29 and
// the webauthn_credentials table is owned exclusively by these helpers.

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrCredentialNotFound is returned when a WebAuthn credential lookup
// finds no matching row.
var ErrCredentialNotFound = errors.New("webauthn credential not found")

// WebAuthnCredential is the on-disk representation of a single
// registered authenticator. A user may have multiple of these.
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

// SetTOTPSecret stores a candidate TOTP secret for the user but leaves
// totp_enabled = 0. The two-step "setup then verify" enrolment lives in
// the API layer; this just persists the candidate.
func (db *DB) SetTOTPSecret(ctx context.Context, username, secret string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET totp_secret = ?, totp_enabled = 0, updated_at = ?
		WHERE username = ?
	`, secret, now, username)
	if err != nil {
		return fmt.Errorf("set totp secret: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// EnableTOTP flips totp_enabled to 1 for the user. Callers MUST have
// already validated a code derived from the stored secret.
func (db *DB) EnableTOTP(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET totp_enabled = 1, updated_at = ?
		WHERE username = ? AND totp_secret IS NOT NULL AND totp_secret != ''
	`, now, username)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DisableTOTP clears both the secret and the enabled flag.
func (db *DB) DisableTOTP(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET totp_secret = NULL, totp_enabled = 0, updated_at = ?
		WHERE username = ?
	`, now, username)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetTOTP returns the stored secret and whether TOTP is enabled for
// the user. A user with totp_enabled=0 may still have a candidate
// secret from an in-progress setup.
func (db *DB) GetTOTP(ctx context.Context, username string) (string, bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return "", false, errors.New("database is closed")
	}

	var sec sql.NullString
	var en sql.NullInt64
	scanErr := db.conn.QueryRowContext(ctx, `
		SELECT totp_secret, totp_enabled FROM users WHERE username = ?
	`, username).Scan(&sec, &en)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return "", false, ErrUserNotFound
	}
	if scanErr != nil {
		return "", false, fmt.Errorf("get totp: %w", scanErr)
	}
	return sec.String, en.Valid && en.Int64 == 1, nil
}

// AddWebAuthnCredential inserts a freshly registered authenticator for
// the given user. Returns the assigned row ID.
func (db *DB) AddWebAuthnCredential(
	ctx context.Context, userID int64, cred WebAuthnCredential,
) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return 0, errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.conn.ExecContext(ctx, `
		INSERT INTO webauthn_credentials (
			user_id, credential_id, public_key, sign_count,
			attestation_type, transports, aaguid, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		userID, cred.CredentialID, cred.PublicKey, int64(cred.SignCount),
		cred.AttestationType, cred.Transports, cred.AAGUID, now,
	)
	if err != nil {
		return 0, fmt.Errorf("add webauthn credential: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// ListWebAuthnCredentials returns all credentials registered for the
// given user. The order is by creation time ascending.
func (db *DB) ListWebAuthnCredentials(
	ctx context.Context, userID int64,
) ([]WebAuthnCredential, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return nil, errors.New("database is closed")
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, credential_id, public_key, sign_count,
		       attestation_type, transports, aaguid, created_at, last_used_at
		FROM webauthn_credentials
		WHERE user_id = ?
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list webauthn credentials: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []WebAuthnCredential
	for rows.Next() {
		var (
			c          WebAuthnCredential
			signCount  int64
			attest     sql.NullString
			transports sql.NullString
			createdAt  string
			lastUsed   sql.NullString
		)
		if scanErr := rows.Scan(
			&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &signCount,
			&attest, &transports, &c.AAGUID, &createdAt, &lastUsed,
		); scanErr != nil {
			return nil, fmt.Errorf("scan webauthn credential: %w", scanErr)
		}
		// Bounds-check the on-disk sign count before narrowing to uint32.
		// SQLite stores INTEGER as int64; a negative or oversized value
		// would corrupt the WebAuthn replay-protection logic, so reject
		// it as a malformed row rather than silently wrapping.
		if signCount < 0 || signCount > 0xFFFFFFFF {
			return nil, fmt.Errorf("scan webauthn credential: sign_count out of range: %d", signCount)
		}
		c.SignCount = uint32(signCount)
		c.AttestationType = attest.String
		c.Transports = transports.String
		if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
			c.CreatedAt = t
		}
		if lastUsed.Valid {
			if t, parseErr := time.Parse(time.RFC3339, lastUsed.String); parseErr == nil {
				c.LastUsedAt = &t
			}
		}
		out = append(out, c)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate webauthn credentials: %w", iterErr)
	}
	return out, nil
}

// UpdateWebAuthnSignCount persists the new authenticator sign count
// (replay-protection counter) and bumps last_used_at to now.
func (db *DB) UpdateWebAuthnSignCount(
	ctx context.Context, credentialID []byte, signCount uint32,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.conn.ExecContext(ctx, `
		UPDATE webauthn_credentials
		SET sign_count = ?, last_used_at = ?
		WHERE credential_id = ?
	`, int64(signCount), now, credentialID)
	if err != nil {
		return fmt.Errorf("update webauthn sign count: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

// DeleteWebAuthnCredential removes a credential by its database ID,
// scoped to the owning user to prevent cross-user deletion.
func (db *DB) DeleteWebAuthnCredential(
	ctx context.Context, userID, credentialDBID int64,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return errors.New("database is closed")
	}

	res, err := db.conn.ExecContext(ctx, `
		DELETE FROM webauthn_credentials WHERE id = ? AND user_id = ?
	`, credentialDBID, userID)
	if err != nil {
		return fmt.Errorf("delete webauthn credential: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrCredentialNotFound
	}
	return nil
}
