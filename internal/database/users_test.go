package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestUserCRUD(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "seed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	db, err := Open(tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("CreateUser", func(t *testing.T) {
		user, err := db.CreateUser(ctx, "admin", "$2a$10$hashedpassword", "admin")
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.Username != "admin" {
			t.Errorf("Expected username 'admin', got %s", user.Username)
		}
		if user.Role != "admin" {
			t.Errorf("Expected role 'admin', got %s", user.Role)
		}
		if !user.IsActive {
			t.Error("Expected user to be active")
		}
		if user.TokenVersion != 1 {
			t.Errorf("Expected token version 1, got %d", user.TokenVersion)
		}
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		_, err := db.CreateUser(ctx, "admin", "$2a$10$anotherpassword", "admin")
		if err != ErrUserExists {
			t.Errorf("Expected ErrUserExists, got %v", err)
		}
	})

	t.Run("GetUser", func(t *testing.T) {
		user, err := db.GetUser(ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if user.Username != "admin" {
			t.Errorf("Expected username 'admin', got %s", user.Username)
		}
	})

	t.Run("GetNonexistentUser", func(t *testing.T) {
		_, err := db.GetUser(ctx, "nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("UpdateUserPassword", func(t *testing.T) {
		newHash := "$2a$10$newhash"
		err := db.UpdateUserPassword(ctx, "admin", newHash)
		if err != nil {
			t.Fatalf("Failed to update password: %v", err)
		}

		user, err := db.GetUser(ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if user.PasswordHash != newHash {
			t.Errorf("Expected new password hash, got %s", user.PasswordHash)
		}
		if user.TokenVersion != 2 {
			t.Errorf("Expected token version 2 after password change, got %d", user.TokenVersion)
		}
	})

	t.Run("UpdateNonexistentUserPassword", func(t *testing.T) {
		err := db.UpdateUserPassword(ctx, "nonexistent", "$2a$10$hash")
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("GetUserCount", func(t *testing.T) {
		count, err := db.GetUserCount(ctx)
		if err != nil {
			t.Fatalf("Failed to get user count: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 user, got %d", count)
		}
	})

	t.Run("GetTokenVersion", func(t *testing.T) {
		version, err := db.GetTokenVersion(ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to get token version: %v", err)
		}
		if version != 2 {
			t.Errorf("Expected version 2, got %d", version)
		}
	})

	t.Run("IncrementTokenVersion", func(t *testing.T) {
		err := db.IncrementTokenVersion(ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to increment token version: %v", err)
		}

		version, err := db.GetTokenVersion(ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to get token version: %v", err)
		}
		if version != 3 {
			t.Errorf("Expected version 3, got %d", version)
		}
	})
}

func TestLoginTracking(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "seed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	db, err := Open(tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create test user
	_, err = db.CreateUser(ctx, "testuser", "$2a$10$hash", "admin")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("RecordLoginSuccess", func(t *testing.T) {
		err := db.RecordLoginSuccess(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to record login success: %v", err)
		}

		user, err := db.GetUser(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if user.LastLogin == nil {
			t.Error("Expected LastLogin to be set")
		}
		if user.FailedAttempts != 0 {
			t.Errorf("Expected 0 failed attempts, got %d", user.FailedAttempts)
		}
	})

	t.Run("RecordLoginFailure", func(t *testing.T) {
		// Record 2 failures (below threshold)
		for i := 0; i < 2; i++ {
			locked, err := db.RecordLoginFailure(ctx, "testuser", 5, 15*time.Minute)
			if err != nil {
				t.Fatalf("Failed to record login failure: %v", err)
			}
			if locked {
				t.Error("Should not be locked after 2 attempts")
			}
		}

		user, err := db.GetUser(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		if user.FailedAttempts != 2 {
			t.Errorf("Expected 2 failed attempts, got %d", user.FailedAttempts)
		}
	})

	t.Run("AccountLockAfterMaxAttempts", func(t *testing.T) {
		// Record more failures to reach threshold
		for i := 0; i < 3; i++ {
			locked, err := db.RecordLoginFailure(ctx, "testuser", 5, 15*time.Minute)
			if err != nil {
				t.Fatalf("Failed to record login failure: %v", err)
			}
			if i == 2 && !locked {
				t.Error("Should be locked after 5 total attempts")
			}
		}

		isLocked, err := db.IsUserLocked(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to check lock status: %v", err)
		}
		if !isLocked {
			t.Error("Expected user to be locked")
		}
	})

	t.Run("SuccessfulLoginClearsLock", func(t *testing.T) {
		err := db.RecordLoginSuccess(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to record login success: %v", err)
		}

		isLocked, err := db.IsUserLocked(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to check lock status: %v", err)
		}
		if isLocked {
			t.Error("Expected user to be unlocked after successful login")
		}

		user, err := db.GetUser(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		if user.FailedAttempts != 0 {
			t.Errorf("Expected 0 failed attempts after success, got %d", user.FailedAttempts)
		}
	})
}

func TestMigrateUserFromConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "seed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	db, err := Open(tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("MigrateNewUser", func(t *testing.T) {
		err := db.MigrateUserFromConfig(ctx, "migrated", "$2a$10$migratedhash")
		if err != nil {
			t.Fatalf("Failed to migrate user: %v", err)
		}

		user, err := db.GetUser(ctx, "migrated")
		if err != nil {
			t.Fatalf("Failed to get migrated user: %v", err)
		}
		if user.Username != "migrated" {
			t.Errorf("Expected username 'migrated', got %s", user.Username)
		}
	})

	t.Run("MigrateExistingUser", func(t *testing.T) {
		// Should not error when user already exists
		err := db.MigrateUserFromConfig(ctx, "migrated", "$2a$10$differenthash")
		if err != nil {
			t.Fatalf("Failed to migrate existing user: %v", err)
		}

		// Password should not have changed
		user, err := db.GetUser(ctx, "migrated")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		if user.PasswordHash != "$2a$10$migratedhash" {
			t.Error("Password hash should not have changed for existing user")
		}
	})
}

func TestUserDatabaseClosed(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "seed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	db, err := Open(tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Close the database
	db.Close()

	ctx := context.Background()

	// All operations should fail on closed database
	_, err = db.GetUser(ctx, "admin")
	if err == nil {
		t.Error("Expected error on closed database")
	}

	_, err = db.CreateUser(ctx, "admin", "hash", "admin")
	if err == nil {
		t.Error("Expected error on closed database")
	}

	err = db.UpdateUserPassword(ctx, "admin", "hash")
	if err == nil {
		t.Error("Expected error on closed database")
	}

	_, err = db.GetUserCount(ctx)
	if err == nil {
		t.Error("Expected error on closed database")
	}
}
