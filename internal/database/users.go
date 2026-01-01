// Package database provides user management for The Seed.
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
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Common errors for user operations.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserExists        = errors.New("user already exists")
	ErrUserLocked        = errors.New("user account is locked")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrNoUsersConfigured = errors.New("no users configured")
)

// GetUser retrieves a user by username.
func (db *DB) GetUser(ctx context.Context, username string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	var user User
	var lastLogin, lockedUntil sql.NullString
	var createdAt, updatedAt string

	err := db.conn.QueryRowContext(ctx, `
		SELECT id, username, password_hash, role, is_active, last_login,
		       failed_attempts, locked_until, token_version, created_at, updated_at
		FROM users
		WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.IsActive,
		&lastLogin, &user.FailedAttempts, &lockedUntil, &user.TokenVersion,
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

// Note: isUniqueConstraintError is defined in repository_profiles.go
