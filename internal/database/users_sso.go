package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SSOUserInput is the payload for UpsertSSOUser. The auth_provider +
// external_id pair is the unique key — matching purely on email would
// allow a compromised IdP to take over an existing local-account user.
type SSOUserInput struct {
	Provider    string // "google" | "microsoft" | "github"
	ExternalID  string // the IdP's stable subject claim
	Email       string // for display and cross-provider matching
	DisplayName string // optional human name from the IdP
}

// UpsertSSOUser returns the user matching (provider, external_id), or
// creates a new row if none exists. On first-ever user creation across
// any channel, the new user becomes 'admin'; subsequent SSO-created
// users default to 'viewer' and an existing admin can promote them.
//
// The synthetic username is "<provider>:<external_id>"; we don't try to
// reuse the email as the username because emails can change at the IdP
// and aren't guaranteed unique across providers (and the local-auth
// users.username UNIQUE constraint applies to the entire users table).
// SSO users never have a usable password_hash — we store a sentinel
// value that bcrypt cannot match against any input.
func (db *DB) UpsertSSOUser(ctx context.Context, in SSOUserInput) (*User, error) {
	if in.Provider == "" || in.ExternalID == "" {
		return nil, errors.New("provider and external_id are required")
	}
	switch in.Provider {
	case AuthProviderGoogle, AuthProviderMicrosoft, AuthProviderGitHub:
		// ok
	default:
		return nil, fmt.Errorf("unsupported SSO provider: %s", in.Provider)
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	// Fast path: look up by (provider, external_id).
	existing, lookupErr := db.lookupSSOUserLocked(ctx, in.Provider, in.ExternalID)
	if lookupErr != nil && !errors.Is(lookupErr, ErrUserNotFound) {
		return nil, lookupErr
	}
	if existing != nil {
		// Refresh email + display name from latest IdP response.
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := db.conn.ExecContext(ctx, `
			UPDATE users SET email = ?, display_name = ?, updated_at = ? WHERE id = ?
		`, in.Email, in.DisplayName, now, existing.ID); err != nil {
			return nil, fmt.Errorf("failed to refresh SSO user: %w", err)
		}
		existing.Email = in.Email
		existing.DisplayName = in.DisplayName
		return existing, nil
	}

	// New user. Decide initial role.
	var totalUsers int
	if cntErr := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalUsers); cntErr != nil {
		return nil, fmt.Errorf("failed to count users for SSO bootstrap: %w", cntErr)
	}
	role := RoleViewer
	if totalUsers == 0 {
		role = RoleAdmin
	}

	username := in.Provider + ":" + in.ExternalID
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// "!" prefix is not in bcrypt's alphabet so this value never matches.
	const ssoSentinelHash = "!sso-no-password"

	res, err := db.conn.ExecContext(ctx, `
		INSERT INTO users
			(username, password_hash, role, is_active, token_version,
			 auth_provider, external_id, email, display_name,
			 created_at, updated_at)
		VALUES (?, ?, ?, 1, 1, ?, ?, ?, ?, ?, ?)
	`, username, ssoSentinelHash, role, in.Provider, in.ExternalID, in.Email, in.DisplayName, nowStr, nowStr)
	if err != nil {
		if isUniqueConstraintError(err) {
			// Race: another request created the same user. Retry the lookup.
			return db.lookupSSOUserLocked(ctx, in.Provider, in.ExternalID)
		}
		return nil, fmt.Errorf("failed to insert SSO user: %w", err)
	}
	id, _ := res.LastInsertId()

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: ssoSentinelHash,
		Role:         role,
		IsActive:     true,
		TokenVersion: 1,
		AuthProvider: in.Provider,
		ExternalID:   in.ExternalID,
		Email:        in.Email,
		DisplayName:  in.DisplayName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// lookupSSOUserLocked finds a user by (provider, external_id). MUST be
// called with db.mu held.
func (db *DB) lookupSSOUserLocked(ctx context.Context, provider, externalID string) (*User, error) {
	var u User
	var lastLogin, lockedUntil, email, displayName sql.NullString
	var createdAt, updatedAt string

	err := db.conn.QueryRowContext(ctx, `
		SELECT id, username, password_hash, role, is_active, last_login,
		       failed_attempts, locked_until, token_version,
		       auth_provider, external_id, email, display_name,
		       created_at, updated_at
		FROM users
		WHERE auth_provider = ? AND external_id = ?
	`, provider, externalID).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive,
		&lastLogin, &u.FailedAttempts, &lockedUntil, &u.TokenVersion,
		&u.AuthProvider, &u.ExternalID, &email, &displayName,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to look up SSO user: %w", err)
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if lastLogin.Valid {
		t, _ := time.Parse(time.RFC3339, lastLogin.String)
		u.LastLogin = &t
	}
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		u.LockedUntil = &t
	}
	u.Email = email.String
	u.DisplayName = displayName.String
	return &u, nil
}
