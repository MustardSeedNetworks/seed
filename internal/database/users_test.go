package database_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

// userCRUDTestDB holds test dependencies for user CRUD tests.
type userCRUDTestDB struct {
	db  *database.DB
	ctx context.Context
}

// setupUserCRUDTest creates a test database for user CRUD tests.
func setupUserCRUDTest(t *testing.T) *userCRUDTestDB {
	t.Helper()

	tmpFile, tmpErr := os.CreateTemp(t.TempDir(), "seed-test-*.db")
	if tmpErr != nil {
		t.Fatalf("Failed to create temp file: %v", tmpErr)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	if openErr != nil {
		t.Fatalf("Failed to open database: %v", openErr)
	}

	t.Cleanup(func() { _ = db.Close() })

	return &userCRUDTestDB{
		db:  db,
		ctx: context.Background(),
	}
}

// createTestUser creates a user and fails the test if an error occurs.
func (uct *userCRUDTestDB) createTestUser(t *testing.T, username, passwordHash, role string) *database.User {
	t.Helper()

	user, err := uct.db.CreateUser(uct.ctx, username, passwordHash, role)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	return user
}

// getTestUser retrieves a user and fails the test if an error occurs.
func (uct *userCRUDTestDB) getTestUser(t *testing.T, username string) *database.User {
	t.Helper()

	user, err := uct.db.GetUser(uct.ctx, username)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	return user
}

// assertUserFields validates common user field expectations.
func assertUserFields(t *testing.T, user *database.User, expectedUsername, expectedRole string) {
	t.Helper()

	if user.Username != expectedUsername {
		t.Errorf("Expected username %q, got %q", expectedUsername, user.Username)
	}
	if user.Role != expectedRole {
		t.Errorf("Expected role %q, got %q", expectedRole, user.Role)
	}
}

// assertUserActive checks that a user is active with expected token version.
func assertUserActive(t *testing.T, user *database.User, expectedTokenVersion int) {
	t.Helper()

	if !user.IsActive {
		t.Error("Expected user to be active")
	}
	if user.TokenVersion != expectedTokenVersion {
		t.Errorf("Expected token version %d, got %d", expectedTokenVersion, user.TokenVersion)
	}
}

// assertExpectedError checks that an error matches the expected error.
func assertExpectedError(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Errorf("Expected error %v, got %v", want, got)
	}
}

// assertTokenVersion checks that the token version matches expected.
func assertTokenVersion(t *testing.T, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("Expected token version %d, got %d", want, got)
	}
}

// assertPasswordHash checks that the password hash matches expected.
func assertPasswordHash(t *testing.T, user *database.User, expected string) {
	t.Helper()

	if user.PasswordHash != expected {
		t.Errorf("Expected password hash %q, got %q", expected, user.PasswordHash)
	}
}

// assertUserCount checks that the user count matches expected.
func assertUserCount(t *testing.T, uct *userCRUDTestDB, expected int) {
	t.Helper()

	count, err := uct.db.GetUserCount(uct.ctx)
	if err != nil {
		t.Fatalf("Failed to get user count: %v", err)
	}
	if count != expected {
		t.Errorf("Expected %d user(s), got %d", expected, count)
	}
}

func TestUserCRUD(t *testing.T) {
	uct := setupUserCRUDTest(t)

	t.Run("CreateUser", func(t *testing.T) {
		user := uct.createTestUser(t, "admin", "$2a$10$hashedpassword", "admin")
		assertUserFields(t, user, "admin", "admin")
		assertUserActive(t, user, 1)
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		_, err := uct.db.CreateUser(uct.ctx, "admin", "$2a$10$anotherpassword", "admin")
		assertExpectedError(t, err, database.ErrUserExists)
	})

	t.Run("GetUser", func(t *testing.T) {
		user := uct.getTestUser(t, "admin")
		assertUserFields(t, user, "admin", "admin")
	})

	t.Run("GetNonexistentUser", func(t *testing.T) {
		_, err := uct.db.GetUser(uct.ctx, "nonexistent")
		assertExpectedError(t, err, database.ErrUserNotFound)
	})

	t.Run("UpdateUserPassword", func(t *testing.T) {
		newHash := "$2a$10$newhash"
		err := uct.db.UpdateUserPassword(uct.ctx, "admin", newHash)
		if err != nil {
			t.Fatalf("Failed to update password: %v", err)
		}

		user := uct.getTestUser(t, "admin")
		assertPasswordHash(t, user, newHash)
		assertTokenVersion(t, user.TokenVersion, 2)
	})

	t.Run("UpdateNonexistentUserPassword", func(t *testing.T) {
		err := uct.db.UpdateUserPassword(uct.ctx, "nonexistent", "$2a$10$hash")
		assertExpectedError(t, err, database.ErrUserNotFound)
	})

	t.Run("GetUserCount", func(t *testing.T) {
		assertUserCount(t, uct, 1)
	})

	t.Run("GetTokenVersion", func(t *testing.T) {
		version, err := uct.db.GetTokenVersion(uct.ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to get token version: %v", err)
		}
		assertTokenVersion(t, version, 2)
	})

	t.Run("IncrementTokenVersion", func(t *testing.T) {
		err := uct.db.IncrementTokenVersion(uct.ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to increment token version: %v", err)
		}

		version, err := uct.db.GetTokenVersion(uct.ctx, "admin")
		if err != nil {
			t.Fatalf("Failed to get token version: %v", err)
		}
		assertTokenVersion(t, version, 3)
	})
}

// loginTrackingTestDB holds test dependencies for login tracking tests.
type loginTrackingTestDB struct {
	db       *database.DB
	ctx      context.Context
	username string
}

// setupLoginTrackingTest creates a test database and user for login tracking tests.
func setupLoginTrackingTest(t *testing.T) *loginTrackingTestDB {
	t.Helper()

	tmpFile, tmpErr := os.CreateTemp(t.TempDir(), "seed-test-*.db")
	if tmpErr != nil {
		t.Fatalf("Failed to create temp file: %v", tmpErr)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	if openErr != nil {
		t.Fatalf("Failed to open database: %v", openErr)
	}

	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	username := "testuser"

	_, createErr := db.CreateUser(ctx, username, "$2a$10$hash", "admin")
	if createErr != nil {
		t.Fatalf("Failed to create user: %v", createErr)
	}

	return &loginTrackingTestDB{
		db:       db,
		ctx:      ctx,
		username: username,
	}
}

// recordFailures records n login failures and returns whether the last attempt caused a lock.
func (ltd *loginTrackingTestDB) recordFailures(t *testing.T, count int) bool {
	t.Helper()

	var lastLocked bool
	for range count {
		locked, err := ltd.db.RecordLoginFailure(ltd.ctx, ltd.username, 5, 15*time.Minute)
		if err != nil {
			t.Fatalf("Failed to record login failure: %v", err)
		}
		lastLocked = locked
	}
	return lastLocked
}

// assertFailedAttempts verifies the user has the expected number of failed attempts.
func (ltd *loginTrackingTestDB) assertFailedAttempts(t *testing.T, expected int) {
	t.Helper()

	user, err := ltd.db.GetUser(ltd.ctx, ltd.username)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user.FailedAttempts != expected {
		t.Errorf("Expected %d failed attempts, got %d", expected, user.FailedAttempts)
	}
}

// assertLockStatus verifies the user's lock status matches the expected value.
func (ltd *loginTrackingTestDB) assertLockStatus(t *testing.T, expectedLocked bool) {
	t.Helper()

	isLocked, err := ltd.db.IsUserLocked(ltd.ctx, ltd.username)
	if err != nil {
		t.Fatalf("Failed to check lock status: %v", err)
	}
	if isLocked != expectedLocked {
		if expectedLocked {
			t.Error("Expected user to be locked")
		} else {
			t.Error("Expected user to be unlocked")
		}
	}
}

// recordLoginSuccess records a successful login for the test user.
func (ltd *loginTrackingTestDB) recordLoginSuccess(t *testing.T) {
	t.Helper()

	err := ltd.db.RecordLoginSuccess(ltd.ctx, ltd.username)
	if err != nil {
		t.Fatalf("Failed to record login success: %v", err)
	}
}

func TestLoginTracking(t *testing.T) {
	t.Run("RecordLoginSuccess", func(t *testing.T) {
		ltd := setupLoginTrackingTest(t)
		ltd.recordLoginSuccess(t)

		user, err := ltd.db.GetUser(ltd.ctx, ltd.username)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if user.LastLogin == nil {
			t.Error("Expected LastLogin to be set")
		}
		ltd.assertFailedAttempts(t, 0)
	})

	t.Run("RecordLoginFailure", func(t *testing.T) {
		ltd := setupLoginTrackingTest(t)
		locked := ltd.recordFailures(t, 2)

		if locked {
			t.Error("Should not be locked after 2 attempts")
		}
		ltd.assertFailedAttempts(t, 2)
	})

	t.Run("AccountLockAfterMaxAttempts", func(t *testing.T) {
		ltd := setupLoginTrackingTest(t)

		// Record 5 failures to trigger lock (threshold is 5)
		locked := ltd.recordFailures(t, 5)
		if !locked {
			t.Error("Should be locked after 5 total attempts")
		}

		ltd.assertLockStatus(t, true)
	})

	t.Run("SuccessfulLoginClearsLock", func(t *testing.T) {
		ltd := setupLoginTrackingTest(t)

		// Lock the account first
		ltd.recordFailures(t, 5)
		ltd.assertLockStatus(t, true)

		// Successful login should clear lock
		ltd.recordLoginSuccess(t)
		ltd.assertLockStatus(t, false)
		ltd.assertFailedAttempts(t, 0)
	})
}

// migrateUserTestDB holds test dependencies for migrate user tests.
type migrateUserTestDB struct {
	db  *database.DB
	ctx context.Context
}

// setupMigrateUserTest creates a test database for migrate user tests.
func setupMigrateUserTest(t *testing.T) *migrateUserTestDB {
	t.Helper()

	tmpFile, tmpErr := os.CreateTemp(t.TempDir(), "seed-test-*.db")
	if tmpErr != nil {
		t.Fatalf("Failed to create temp file: %v", tmpErr)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	if openErr != nil {
		t.Fatalf("Failed to open database: %v", openErr)
	}

	t.Cleanup(func() { _ = db.Close() })

	return &migrateUserTestDB{
		db:  db,
		ctx: context.Background(),
	}
}

// migrateUser migrates a user and fails the test if an error occurs.
func (mut *migrateUserTestDB) migrateUser(t *testing.T, username, passwordHash string) {
	t.Helper()

	err := mut.db.MigrateUserFromConfig(mut.ctx, username, passwordHash)
	if err != nil {
		t.Fatalf("Failed to migrate user: %v", err)
	}
}

// getUser retrieves a user and fails the test if an error occurs.
func (mut *migrateUserTestDB) getUser(t *testing.T, username string) *database.User {
	t.Helper()

	user, err := mut.db.GetUser(mut.ctx, username)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	return user
}

func TestMigrateUserFromConfig(t *testing.T) {
	mut := setupMigrateUserTest(t)

	t.Run("MigrateNewUser", func(t *testing.T) {
		mut.migrateUser(t, "migrated", "$2a$10$migratedhash")

		user := mut.getUser(t, "migrated")
		assertUserFields(t, user, "migrated", "admin")
	})

	t.Run("MigrateExistingUser", func(t *testing.T) {
		// Should not error when user already exists
		mut.migrateUser(t, "migrated", "$2a$10$differenthash")

		// Password should not have changed
		user := mut.getUser(t, "migrated")
		assertPasswordHash(t, user, "$2a$10$migratedhash")
	})
}

// closedDBTestDB holds test dependencies for closed database tests.
type closedDBTestDB struct {
	db  *database.DB
	ctx context.Context
}

// setupClosedDBTest creates and closes a test database for closed database tests.
func setupClosedDBTest(t *testing.T) *closedDBTestDB {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, err := database.Open(tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Close the database immediately
	_ = db.Close()

	return &closedDBTestDB{
		db:  db,
		ctx: context.Background(),
	}
}

// assertOperationFails checks that an operation returns an error.
func assertOperationFails(t *testing.T, err error, operation string) {
	t.Helper()

	if err == nil {
		t.Errorf("Expected error on closed database for %s", operation)
	}
}

func TestUserDatabaseClosed(t *testing.T) {
	cdt := setupClosedDBTest(t)

	// All operations should fail on closed database
	_, err := cdt.db.GetUser(cdt.ctx, "admin")
	assertOperationFails(t, err, "GetUser")

	_, err = cdt.db.CreateUser(cdt.ctx, "admin", "hash", "admin")
	assertOperationFails(t, err, "CreateUser")

	err = cdt.db.UpdateUserPassword(cdt.ctx, "admin", "hash")
	assertOperationFails(t, err, "UpdateUserPassword")

	_, err = cdt.db.GetUserCount(cdt.ctx)
	assertOperationFails(t, err, "GetUserCount")
}
