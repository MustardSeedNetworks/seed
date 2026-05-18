// SPDX-License-Identifier: BUSL-1.1

package database

import (
	"context"
	"time"
)

// User authentication security defaults.
const (
	// defaultMaxLoginAttempts is the number of failed login attempts before account lockout.
	defaultMaxLoginAttempts = 5

	// defaultLockDurationMinutes is the account lockout duration in minutes.
	defaultLockDurationMinutes = 15
)

// UserStoreAdapter adapts the database to the auth.UserStore interface.
// This allows the auth package to use the database for user management
// without creating a circular dependency.
type UserStoreAdapter struct {
	db               *DB
	maxLoginAttempts int
	lockDuration     time.Duration
}

// NewUserStoreAdapter creates a new adapter that implements auth.UserStore.
func NewUserStoreAdapter(db *DB) *UserStoreAdapter {
	return &UserStoreAdapter{
		db:               db,
		maxLoginAttempts: defaultMaxLoginAttempts,
		lockDuration:     defaultLockDurationMinutes * time.Minute,
	}
}

// GetPasswordHash returns the password hash for a user.
func (a *UserStoreAdapter) GetPasswordHash(ctx context.Context, username string) (string, error) {
	user, err := a.db.GetUser(ctx, username)
	if err != nil {
		return "", err
	}
	return user.PasswordHash, nil
}

// GetTokenVersion returns the current token version for a user.
func (a *UserStoreAdapter) GetTokenVersion(ctx context.Context, username string) (int, error) {
	return a.db.GetTokenVersion(ctx, username)
}

// UpdatePassword updates a user's password hash.
func (a *UserStoreAdapter) UpdatePassword(ctx context.Context, username, hash string) error {
	return a.db.UpdateUserPassword(ctx, username, hash)
}

// RecordLoginSuccess records a successful login.
func (a *UserStoreAdapter) RecordLoginSuccess(ctx context.Context, username string) error {
	return a.db.RecordLoginSuccess(ctx, username)
}

// RecordLoginFailure records a failed login attempt.
func (a *UserStoreAdapter) RecordLoginFailure(ctx context.Context, username string) error {
	_, err := a.db.RecordLoginFailure(ctx, username, a.maxLoginAttempts, a.lockDuration)
	return err
}

// IsLocked checks if a user account is locked.
func (a *UserStoreAdapter) IsLocked(ctx context.Context, username string) (bool, error) {
	return a.db.IsUserLocked(ctx, username)
}

// MigrateUserFromConfig migrates a user from config to the database.
// This is called on startup to ensure config users exist in the database.
func (a *UserStoreAdapter) MigrateUserFromConfig(
	ctx context.Context,
	username, passwordHash string,
) error {
	return a.db.MigrateUserFromConfig(ctx, username, passwordHash)
}

// CreateUser creates a new user in the database.
func (a *UserStoreAdapter) CreateUser(
	ctx context.Context,
	username, passwordHash, role string,
) error {
	_, err := a.db.CreateUser(ctx, username, passwordHash, role)
	return err
}

// GetUserCount returns the number of users in the database.
func (a *UserStoreAdapter) GetUserCount(ctx context.Context) (int, error) {
	return a.db.GetUserCount(ctx)
}
