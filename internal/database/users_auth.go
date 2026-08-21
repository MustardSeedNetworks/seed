package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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

// GetClientID returns the id of the client that owns a user. It is the
// source of truth the session's client claim is minted from, so it never
// substitutes a default: an unknown user is ErrUserNotFound, not tenant zero.
func (db *DB) GetClientID(ctx context.Context, username string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return "", errors.New("database is closed")
	}

	var clientID string
	err := db.conn.QueryRowContext(ctx, `
		SELECT client_id FROM users WHERE username = ?
	`, username).Scan(&clientID)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get client id: %w", err)
	}
	return clientID, nil
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
