// SPDX-License-Identifier: BUSL-1.1

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// APITokenRecord is the persisted form of a personal-access token.
// The plaintext token is never stored — only the SHA-256 hex digest
// in TokenHash. Prefix is the first 12 chars of the plaintext, kept
// so the UI can identify a token without revealing it.
type APITokenRecord struct {
	ID            string
	OwnerUsername string
	Name          string
	TokenHash     string
	Prefix        string
	CreatedAt     time.Time
	LastUsedAt    time.Time
	RevokedAt     time.Time
	// Scope caps the effective role of requests made with this token.
	// Empty means inherit the owner's role (the default for tokens minted
	// before #1255). When set, the effective role is min(owner.role,
	// scope) at auth time so a less-privileged automation token can be
	// minted from an admin owner.
	Scope string
}

// IsActive reports whether the token is currently usable (not revoked).
func (r APITokenRecord) IsActive() bool {
	return r.RevokedAt.IsZero()
}

// APITokenRepository persists api_tokens rows. Methods are safe to call
// concurrently because they delegate to *database/sql which handles
// connection pooling.
type APITokenRepository struct {
	db *DB
}

// NewAPITokenRepository constructs a repository bound to the given DB.
func NewAPITokenRepository(db *DB) *APITokenRepository {
	return &APITokenRepository{db: db}
}

// Insert persists a newly minted token record. An empty Scope is stored
// as NULL so the column reads back as "" via the COALESCE in scan, which
// the auth layer interprets as "inherit owner's role" (#1255).
func (r *APITokenRepository) Insert(ctx context.Context, t APITokenRecord) error {
	var scope any
	if t.Scope != "" {
		scope = t.Scope
	}
	_, err := r.db.conn.ExecContext(ctx, `
		INSERT INTO api_tokens (id, owner_username, name, token_hash, prefix, created_at, scope)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.OwnerUsername, t.Name, t.TokenHash, t.Prefix, t.CreatedAt.UTC().Format(time.RFC3339Nano), scope)
	if err != nil {
		return fmt.Errorf("insert api token: %w", err)
	}
	return nil
}

// FindActiveByHash returns the active (non-revoked) token row matching
// the given hash, or [sql.ErrNoRows] if no match.
func (r *APITokenRepository) FindActiveByHash(ctx context.Context, hash string) (APITokenRecord, error) {
	row := r.db.conn.QueryRowContext(ctx, `
		SELECT id, owner_username, name, token_hash, prefix, created_at,
		       COALESCE(last_used_at, ''), COALESCE(revoked_at, ''),
		       COALESCE(scope, '')
		FROM api_tokens
		WHERE token_hash = ? AND revoked_at IS NULL
	`, hash)
	return scanAPIToken(row)
}

// ListByOwner returns all tokens owned by the given user, ordered by
// creation time descending. Revoked tokens are included so the UI can
// show "revoked" entries that haven't been deleted; callers filter as
// needed.
func (r *APITokenRepository) ListByOwner(ctx context.Context, owner string) ([]APITokenRecord, error) {
	rows, err := r.db.conn.QueryContext(ctx, `
		SELECT id, owner_username, name, token_hash, prefix, created_at,
		       COALESCE(last_used_at, ''), COALESCE(revoked_at, ''),
		       COALESCE(scope, '')
		FROM api_tokens
		WHERE owner_username = ?
		ORDER BY created_at DESC
	`, owner)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []APITokenRecord
	for rows.Next() {
		rec, scanErr := scanAPIToken(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, rec)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate api tokens: %w", rowsErr)
	}
	return out, nil
}

// TouchLastUsed marks the token's last_used_at to now. Failures are
// non-fatal for the caller's auth check but are returned so callers
// that care can log them.
func (r *APITokenRepository) TouchLastUsed(ctx context.Context, id string) error {
	_, err := r.db.conn.ExecContext(ctx, `
		UPDATE api_tokens SET last_used_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("touch api token: %w", err)
	}
	return nil
}

// Revoke marks the token as revoked. Returns [sql.ErrNoRows] if no token
// with that ID exists for the given owner.
func (r *APITokenRepository) Revoke(ctx context.Context, id, owner string) error {
	res, err := r.db.conn.ExecContext(ctx, `
		UPDATE api_tokens
		SET revoked_at = ?
		WHERE id = ? AND owner_username = ? AND revoked_at IS NULL
	`, time.Now().UTC().Format(time.RFC3339Nano), id, owner)
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// rowScanner abstracts *[sql.Row] and *[sql.Rows] for scanAPIToken's reuse.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAPIToken(r rowScanner) (APITokenRecord, error) {
	var rec APITokenRecord
	var createdStr, lastUsedStr, revokedStr string

	scanErr := r.Scan(
		&rec.ID, &rec.OwnerUsername, &rec.Name, &rec.TokenHash, &rec.Prefix,
		&createdStr, &lastUsedStr, &revokedStr, &rec.Scope,
	)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return rec, sql.ErrNoRows
		}
		return rec, fmt.Errorf("scan api token: %w", scanErr)
	}

	rec.CreatedAt = parseTime(createdStr)
	rec.LastUsedAt = parseTime(lastUsedStr)
	rec.RevokedAt = parseTime(revokedStr)
	return rec, nil
}

// parseTime is a forgiving parser — returns zero time on empty/invalid
// inputs because legacy rows may use slightly different formats.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
