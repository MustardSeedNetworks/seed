package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User represents a user in the database.
type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	Role           string
	IsActive       bool
	LastLogin      *time.Time
	FailedAttempts int
	LockedUntil    *time.Time
	TokenVersion   int
	AuthProvider   string // 'local' | 'google' | 'microsoft' | 'github'
	ExternalID     string // IdP subject claim (OIDC 'sub' / MS Graph 'id'); empty for local users
	Email          string // display + cross-provider matching
	DisplayName    string // optional human name returned by the IdP
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Valid role identifiers (enforced by the DB CHECK constraint added in the
// hardening migration). The constant set lives in code as well so handlers
// can validate input before hitting the DB.
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// IsValidRole returns true if r is one of admin, operator, or viewer.
func IsValidRole(r string) bool {
	return r == RoleAdmin || r == RoleOperator || r == RoleViewer
}

// Valid auth providers (enforced by the DB CHECK constraint).
const (
	AuthProviderLocal     = "local"
	AuthProviderGoogle    = "google"
	AuthProviderMicrosoft = "microsoft"
	AuthProviderGitHub    = "github"
)

// Common errors for user operations.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserExists        = errors.New("user already exists")
	ErrUserLocked        = errors.New("user account is locked")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrNoUsersConfigured = errors.New("no users configured")
	ErrInvalidRole       = errors.New("invalid role")
	ErrLastAdmin         = errors.New("cannot demote or delete the last admin")
)

// GetUser retrieves a user by username.
func (db *DB) GetUser(ctx context.Context, username string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	var user User
	var lastLogin, lockedUntil, externalID, email, displayName sql.NullString
	var createdAt, updatedAt string

	err := db.conn.QueryRowContext(ctx, `
		SELECT id, username, password_hash, role, is_active, last_login,
		       failed_attempts, locked_until, token_version,
		       auth_provider, external_id, email, display_name,
		       created_at, updated_at
		FROM users
		WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.IsActive,
		&lastLogin, &user.FailedAttempts, &lockedUntil, &user.TokenVersion,
		&user.AuthProvider, &externalID, &email, &displayName,
		&createdAt, &updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Parse timestamps
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if lastLogin.Valid {
		t, _ := time.Parse(time.RFC3339, lastLogin.String)
		user.LastLogin = &t
	}
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		user.LockedUntil = &t
	}
	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if displayName.Valid {
		user.DisplayName = displayName.String
	}

	return &user, nil
}

// CreateUser creates a new user in the database.
func (db *DB) CreateUser(ctx context.Context, username, passwordHash, role string) (*User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, role, is_active, token_version, created_at, updated_at)
		VALUES (?, ?, ?, 1, 1, ?, ?)
	`, username, passwordHash, role, nowStr, nowStr)
	if err != nil {
		// Check for unique constraint violation
		if isUniqueConstraintError(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, _ := result.LastInsertId()

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		IsActive:     true,
		TokenVersion: 1,
		AuthProvider: AuthProviderLocal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// UpdateUserPassword updates a user's password hash and increments token version.
func (db *DB) UpdateUserPassword(ctx context.Context, username, passwordHash string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	result, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, token_version = token_version + 1, updated_at = ?
		WHERE username = ?
	`, passwordHash, now, username)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// RecordLoginSuccess records a successful login.
func (db *DB) RecordLoginSuccess(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET last_login = ?, failed_attempts = 0, locked_until = NULL, updated_at = ?
		WHERE username = ?
	`, now, now, username)

	return err
}

// RecordLoginFailure records a failed login attempt.
// Returns true if the account is now locked.
func (db *DB) RecordLoginFailure(
	ctx context.Context,
	username string,
	maxAttempts int,
	lockDuration time.Duration,
) (bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return false, errors.New("database is closed")
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// Get current failed attempts
	var failedAttempts int
	err := db.conn.QueryRowContext(ctx, `
		SELECT failed_attempts FROM users WHERE username = ?
	`, username).Scan(&failedAttempts)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil // User doesn't exist, don't reveal this
	}
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	newAttempts := failedAttempts + 1
	var lockedUntil *string

	if newAttempts >= maxAttempts {
		lockTime := now.Add(lockDuration).Format(time.RFC3339)
		lockedUntil = &lockTime
	}

	_, err = db.conn.ExecContext(ctx, `
		UPDATE users
		SET failed_attempts = ?, locked_until = ?, updated_at = ?
		WHERE username = ?
	`, newAttempts, lockedUntil, nowStr, username)
	if err != nil {
		return false, fmt.Errorf("failed to record login failure: %w", err)
	}

	return lockedUntil != nil, nil
}

// IsUserLocked checks if a user account is locked.
func (db *DB) IsUserLocked(ctx context.Context, username string) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return false, errors.New("database is closed")
	}

	var lockedUntil sql.NullString
	err := db.conn.QueryRowContext(ctx, `
		SELECT locked_until FROM users WHERE username = ?
	`, username).Scan(&lockedUntil)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check lock status: %w", err)
	}

	if !lockedUntil.Valid {
		return false, nil
	}

	lockTime, err := time.Parse(time.RFC3339, lockedUntil.String)
	if err != nil {
		return false, nil
	}

	return time.Now().Before(lockTime), nil
}

// GetUserCount returns the number of users in the database.
func (db *DB) GetUserCount(ctx context.Context) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, errors.New("database is closed")
	}

	var count int
	err := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// GetTokenVersion returns the current token version for a user.
func (db *DB) GetTokenVersion(ctx context.Context, username string) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, errors.New("database is closed")
	}

	var version int
	err := db.conn.QueryRowContext(ctx, `
		SELECT token_version FROM users WHERE username = ?
	`, username).Scan(&version)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get token version: %w", err)
	}

	return version, nil
}

// IncrementTokenVersion invalidates all existing tokens for a user.
func (db *DB) IncrementTokenVersion(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET token_version = token_version + 1, updated_at = ?
		WHERE username = ?
	`, now, username)

	return err
}

// MigrateUserFromConfig migrates a user from config to database if not already present.
// This provides backward compatibility during the transition.
func (db *DB) MigrateUserFromConfig(ctx context.Context, username, passwordHash string) error {
	// Check if user already exists
	_, err := db.GetUser(ctx, username)
	if err == nil {
		// User exists, no migration needed
		return nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	// Create the user
	_, err = db.CreateUser(ctx, username, passwordHash, "admin")
	if err != nil && !errors.Is(err, ErrUserExists) {
		return fmt.Errorf("failed to migrate user: %w", err)
	}

	return nil
}

// ListUsers returns every user, ordered by username.
func (db *DB) ListUsers(ctx context.Context) ([]*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, username, password_hash, role, is_active, last_login,
		       failed_attempts, locked_until, token_version,
		       auth_provider, external_id, email, display_name,
		       created_at, updated_at
		FROM users
		ORDER BY username
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*User
	for rows.Next() {
		var u User
		var lastLogin, lockedUntil, externalID, email, displayName sql.NullString
		var createdAt, updatedAt string
		if scanErr := rows.Scan(
			&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive,
			&lastLogin, &u.FailedAttempts, &lockedUntil, &u.TokenVersion,
			&u.AuthProvider, &externalID, &email, &displayName,
			&createdAt, &updatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", scanErr)
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
		u.ExternalID = externalID.String
		u.Email = email.String
		u.DisplayName = displayName.String
		out = append(out, &u)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", rowsErr)
	}
	return out, nil
}

// UpdateUserRole sets a user's role. Refuses to demote the last admin
// (returns ErrLastAdmin). The CHECK constraint on the role column would
// reject an invalid role at the DB layer too, but we surface a clean
// Go-level error.
func (db *DB) UpdateUserRole(ctx context.Context, username, role string) error {
	if !IsValidRole(role) {
		return ErrInvalidRole
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Fetch current role to detect a demote-from-admin path.
	var currentRole string
	scanErr := tx.QueryRowContext(ctx,
		`SELECT role FROM users WHERE username = ?`, username).Scan(&currentRole)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to read current role: %w", scanErr)
	}

	if currentRole == RoleAdmin && role != RoleAdmin {
		var adminCount int
		cntErr := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users WHERE role = 'admin' AND is_active = 1`).Scan(&adminCount)
		if cntErr != nil {
			return fmt.Errorf("failed to count admins: %w", cntErr)
		}
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		UPDATE users SET role = ?, updated_at = ? WHERE username = ?
	`, role, now, username)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit role change: %w", commitErr)
	}
	return nil
}

// DeleteUser hard-deletes a user. Refuses to delete the last admin
// (returns ErrLastAdmin). The FK cascade on api_tokens automatically
// purges the user's tokens; the user's authenticated sessions are
// already revoked because the FK delete also removes the token_version
// row they verify against (incrementing TokenVersion is unnecessary
// once the row is gone).
//
// Soft-delete via is_active=0 is also supported (use DeactivateUser);
// hard-delete is reserved for "remove this person entirely" actions.
func (db *DB) DeleteUser(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var role string
	scanErr := tx.QueryRowContext(ctx,
		`SELECT role FROM users WHERE username = ?`, username).Scan(&role)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to read role: %w", scanErr)
	}

	if role == RoleAdmin {
		var adminCount int
		cntErr := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users WHERE role = 'admin' AND is_active = 1`).Scan(&adminCount)
		if cntErr != nil {
			return fmt.Errorf("failed to count admins: %w", cntErr)
		}
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit delete: %w", commitErr)
	}
	return nil
}

// DeactivateUser sets is_active = 0 and increments token_version so any
// outstanding sessions/tokens are immediately invalidated. Use this
// instead of DeleteUser when the user record needs to remain for audit
// trails or reactivation.
func (db *DB) DeactivateUser(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET is_active = 0, token_version = token_version + 1, updated_at = ?
		WHERE username = ?
	`, now, username)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

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

// Note: isUniqueConstraintError is defined in repository_profiles.go
