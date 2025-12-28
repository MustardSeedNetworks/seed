// Package database provides user store adapter for authentication.
package database

import (
	"context"
	"time"
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
		maxLoginAttempts: 5,                // Lock after 5 failed attempts
		lockDuration:     15 * time.Minute, // Lock for 15 minutes
	}
}

// GetPasswordHash returns the password hash for a user.
func (a *UserStoreAdapter) GetPasswordHash(username string) (string, error) {
	ctx := context.Background()
	user, err := a.db.GetUser(ctx, username)
	if err != nil {
		return "", err
	}
	return user.PasswordHash, nil
}

// GetTokenVersion returns the current token version for a user.
func (a *UserStoreAdapter) GetTokenVersion(username string) (int, error) {
	ctx := context.Background()
	return a.db.GetTokenVersion(ctx, username)
}

// UpdatePassword updates a user's password hash.
func (a *UserStoreAdapter) UpdatePassword(username, hash string) error {
	ctx := context.Background()
	return a.db.UpdateUserPassword(ctx, username, hash)
}

// RecordLoginSuccess records a successful login.
func (a *UserStoreAdapter) RecordLoginSuccess(username string) error {
	ctx := context.Background()
	return a.db.RecordLoginSuccess(ctx, username)
}

// RecordLoginFailure records a failed login attempt.
func (a *UserStoreAdapter) RecordLoginFailure(username string) error {
	ctx := context.Background()
	_, err := a.db.RecordLoginFailure(ctx, username, a.maxLoginAttempts, a.lockDuration)
	return err
}

// IsLocked checks if a user account is locked.
func (a *UserStoreAdapter) IsLocked(username string) (bool, error) {
	ctx := context.Background()
	return a.db.IsUserLocked(ctx, username)
}

// MigrateUserFromConfig migrates a user from config to the database.
// This is called on startup to ensure config users exist in the database.
func (a *UserStoreAdapter) MigrateUserFromConfig(username, passwordHash string) error {
	ctx := context.Background()
	return a.db.MigrateUserFromConfig(ctx, username, passwordHash)
}

// CreateUser creates a new user in the database.
func (a *UserStoreAdapter) CreateUser(username, passwordHash, role string) error {
	ctx := context.Background()
	_, err := a.db.CreateUser(ctx, username, passwordHash, role)
	return err
}

// GetUserCount returns the number of users in the database.
func (a *UserStoreAdapter) GetUserCount() (int, error) {
	ctx := context.Background()
	return a.db.GetUserCount(ctx)
}
