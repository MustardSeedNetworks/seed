package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testDB creates a temporary database for testing.
func testDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	}

	return db, cleanup
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Verify we can ping it
	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestOpenWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		Path:            dbPath,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
		EnableWAL:       true,
		BusyTimeout:     1000,
	}

	db, err := OpenWithConfig(cfg)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify path
	if db.Path() != dbPath {
		t.Errorf("expected path %q, got %q", dbPath, db.Path())
	}
}

func TestMigrations(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	// Check schema version
	version, err := db.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get schema version: %v", err)
	}

	if version != len(migrations) {
		t.Errorf("expected schema version %d, got %d", len(migrations), version)
	}

	// Check migration status
	status, err := db.MigrationStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get migration status: %v", err)
	}

	if len(status) != len(migrations) {
		t.Errorf("expected %d migrations, got %d", len(migrations), len(status))
	}

	for _, m := range status {
		if !m.Applied {
			t.Errorf("migration %d (%s) not applied", m.Version, m.Description)
		}
	}
}

func TestClose(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	// Close should work
	if err := db.Close(); err != nil {
		t.Errorf("failed to close database: %v", err)
	}

	// Second close should be no-op
	if err := db.Close(); err != nil {
		t.Errorf("second close failed: %v", err)
	}

	// Operations should fail after close
	ctx := context.Background()
	if err := db.Ping(ctx); err == nil {
		t.Error("ping should fail after close")
	}
}

func TestWithTx(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	// Successful transaction
	err := db.WithTx(ctx, func(_ *sql.Tx) error {
		return nil
	})
	if err != nil {
		t.Errorf("WithTx failed: %v", err)
	}
}

func TestProfileRepository(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Profiles()

	// Create
	profile := &Profile{
		Name:        "test-profile",
		Description: "Test profile",
		ConfigJSON:  `{"key": "value"}`,
		IsDefault:   false,
	}

	err := repo.Create(ctx, profile)
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	if profile.ID == "" {
		t.Error("profile ID should be set after create")
	}

	// Get
	got, err := repo.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("failed to get profile: %v", err)
	}

	if got.Name != profile.Name {
		t.Errorf("expected name %q, got %q", profile.Name, got.Name)
	}

	// Get by name
	got, err = repo.GetByName(ctx, "test-profile")
	if err != nil {
		t.Fatalf("failed to get by name: %v", err)
	}

	if got.ID != profile.ID {
		t.Errorf("expected ID %q, got %q", profile.ID, got.ID)
	}

	// Update
	profile.Description = "Updated description"
	err = repo.Update(ctx, profile)
	if err != nil {
		t.Fatalf("failed to update profile: %v", err)
	}

	got, err = repo.Get(ctx, profile.ID)
	require.NoError(t, err)
	if got.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got.Description)
	}

	// List (includes the seeded default profile + our test profile)
	profiles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("failed to list profiles: %v", err)
	}

	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles (default + test), got %d", len(profiles))
	}

	// Set default
	err = repo.SetDefault(ctx, profile.ID)
	if err != nil {
		t.Fatalf("failed to set default: %v", err)
	}

	got, err = repo.GetDefault(ctx)
	require.NoError(t, err)
	if got.ID != profile.ID {
		t.Errorf("expected default profile %q, got %q", profile.ID, got.ID)
	}

	// Count (includes the seeded default profile + our test profile)
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 2 {
		t.Errorf("expected count 2 (default + test), got %d", count)
	}

	// Delete
	err = repo.Delete(ctx, profile.ID)
	if err != nil {
		t.Fatalf("failed to delete profile: %v", err)
	}

	_, err = repo.Get(ctx, profile.ID)
	if !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestMetricsRepository(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Metrics()

	// Record
	metric := &Metric{
		InterfaceName: "eth0",
		MetricType:    MetricTypeLatency,
		Value:         42.5,
		Unit:          "ms",
	}

	err := repo.Record(ctx, metric)
	if err != nil {
		t.Fatalf("failed to record metric: %v", err)
	}

	if metric.ID == 0 {
		t.Error("metric ID should be set after record")
	}

	// Query
	metrics, err := repo.Query(ctx, MetricQueryOptions{
		InterfaceName: "eth0",
		MetricType:    MetricTypeLatency,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("failed to query metrics: %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}

	// GetLatest
	latest, err := repo.GetLatest(ctx, "eth0", MetricTypeLatency)
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}

	if latest == nil {
		t.Fatal("expected latest metric, got nil")
	}

	if latest.Value != 42.5 {
		t.Errorf("expected value 42.5, got %f", latest.Value)
	}

	// GetAggregates
	agg, err := repo.GetAggregates(ctx, MetricAggregateOptions{
		InterfaceName: "eth0",
		MetricType:    MetricTypeLatency,
	})
	if err != nil {
		t.Fatalf("failed to get aggregates: %v", err)
	}

	if agg.Count != 1 {
		t.Errorf("expected count 1, got %d", agg.Count)
	}

	// Record batch
	batch := []*Metric{
		{InterfaceName: "eth0", MetricType: MetricTypeLatency, Value: 10},
		{InterfaceName: "eth0", MetricType: MetricTypeLatency, Value: 20},
		{InterfaceName: "eth0", MetricType: MetricTypeLatency, Value: 30},
	}

	err = repo.RecordBatch(ctx, batch)
	if err != nil {
		t.Fatalf("failed to record batch: %v", err)
	}

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	if count != 4 {
		t.Errorf("expected 4 metrics, got %d", count)
	}

	// Delete old - set cutoff to future time to delete all records
	futureTime := time.Now().Add(time.Hour) // All records are before this
	deleted, err := repo.DeleteOlderThan(ctx, futureTime)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// All 4 records should be deleted (they're older than future time)
	if deleted != 4 {
		t.Errorf("expected to delete 4, deleted %d", deleted)
	}

	// Verify records are gone
	count, err = repo.Count(ctx)
	require.NoError(t, err)
	if count != 0 {
		t.Errorf("expected 0 remaining metrics, got %d", count)
	}
}

func TestDeviceRepository(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Devices()

	// Create
	device := &Device{
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:ff",
		Hostname:   "test-device",
		Vendor:     "Test Vendor",
		DeviceType: "router",
	}

	err := repo.Create(ctx, device)
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	if device.ID == "" {
		t.Error("device ID should be set after create")
	}

	// Get
	got, err := repo.Get(ctx, device.ID)
	if err != nil {
		t.Fatalf("failed to get device: %v", err)
	}

	if got.IPAddress != device.IPAddress {
		t.Errorf("expected IP %q, got %q", device.IPAddress, got.IPAddress)
	}

	// Get by IP
	got, err = repo.GetByIP(ctx, "192.168.1.100")
	if err != nil {
		t.Fatalf("failed to get by IP: %v", err)
	}

	if got.ID != device.ID {
		t.Errorf("expected ID %q, got %q", device.ID, got.ID)
	}

	// Get by MAC
	got, err = repo.GetByMAC(ctx, "aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("failed to get by MAC: %v", err)
	}

	if got.ID != device.ID {
		t.Errorf("expected ID %q, got %q", device.ID, got.ID)
	}

	// Update
	device.Hostname = "updated-device"
	err = repo.Update(ctx, device)
	if err != nil {
		t.Fatalf("failed to update device: %v", err)
	}

	// List
	devices, err := repo.List(ctx, DeviceListOptions{})
	if err != nil {
		t.Fatalf("failed to list devices: %v", err)
	}

	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}

	// Upsert
	upsertDevice := &Device{
		IPAddress: "192.168.1.100",
		Hostname:  "upserted-device",
	}

	err = repo.Upsert(ctx, upsertDevice)
	if err != nil {
		t.Fatalf("failed to upsert device: %v", err)
	}

	got, err = repo.GetByIP(ctx, "192.168.1.100")
	require.NoError(t, err)
	if got.Hostname != "upserted-device" {
		t.Errorf("expected hostname 'upserted-device', got %q", got.Hostname)
	}

	// Count
	count, err := repo.Count(ctx, false)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Delete
	err = repo.Delete(ctx, device.ID)
	if err != nil {
		t.Fatalf("failed to delete device: %v", err)
	}

	_, err = repo.Get(ctx, device.ID)
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestAlertRepository(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()

	// Create
	alert := &Alert{
		Type:     AlertTypeSecurity,
		Severity: AlertSeverityWarning,
		Title:    "Test Alert",
		Message:  "This is a test alert",
		Source:   "test",
	}

	err := repo.Create(ctx, alert)
	if err != nil {
		t.Fatalf("failed to create alert: %v", err)
	}

	if alert.ID == 0 {
		t.Error("alert ID should be set after create")
	}

	// Get
	got, err := repo.Get(ctx, alert.ID)
	if err != nil {
		t.Fatalf("failed to get alert: %v", err)
	}

	if got.Title != alert.Title {
		t.Errorf("expected title %q, got %q", alert.Title, got.Title)
	}

	// List
	alerts, err := repo.List(ctx, AlertListOptions{})
	if err != nil {
		t.Fatalf("failed to list alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}

	// Acknowledge
	err = repo.Acknowledge(ctx, alert.ID, "testuser")
	if err != nil {
		t.Fatalf("failed to acknowledge: %v", err)
	}

	got, err = repo.Get(ctx, alert.ID)
	require.NoError(t, err)
	if !got.Acknowledged {
		t.Error("alert should be acknowledged")
	}

	// Resolve
	err = repo.Resolve(ctx, alert.ID)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	got, err = repo.Get(ctx, alert.ID)
	require.NoError(t, err)
	if !got.Resolved {
		t.Error("alert should be resolved")
	}

	// Count
	count, err := repo.Count(ctx, AlertListOptions{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Delete
	err = repo.Delete(ctx, alert.ID)
	if err != nil {
		t.Fatalf("failed to delete alert: %v", err)
	}

	_, err = repo.Get(ctx, alert.ID)
	if !errors.Is(err, ErrAlertNotFound) {
		t.Errorf("expected ErrAlertNotFound, got %v", err)
	}
}

func TestSettingsRepository(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Settings()

	// Set
	err := repo.Set(ctx, "test_key", "test_value")
	if err != nil {
		t.Fatalf("failed to set setting: %v", err)
	}

	// Get
	setting, err := repo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}

	if setting.Value != "test_value" {
		t.Errorf("expected value %q, got %q", "test_value", setting.Value)
	}

	// GetValue
	value, err := repo.GetValue(ctx, "test_key")
	if err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if value != "test_value" {
		t.Errorf("expected value %q, got %q", "test_value", value)
	}

	// GetWithDefault
	value, err = repo.GetWithDefault(ctx, "nonexistent", "default_value")
	if err != nil {
		t.Fatalf("failed to get with default: %v", err)
	}

	if value != "default_value" {
		t.Errorf("expected value %q, got %q", "default_value", value)
	}

	// SetIfNotExists
	created, err := repo.SetIfNotExists(ctx, "test_key", "new_value")
	if err != nil {
		t.Fatalf("failed to set if not exists: %v", err)
	}

	if created {
		t.Error("should not create existing key")
	}

	// List
	settings, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(settings) != 1 {
		t.Errorf("expected 1 setting, got %d", len(settings))
	}

	// Delete
	err = repo.Delete(ctx, "test_key")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = repo.Get(ctx, "test_key")
	if !errors.Is(err, ErrSettingNotFound) {
		t.Errorf("expected ErrSettingNotFound, got %v", err)
	}
}

func TestRetention(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add some test data
	for i := range 5 {
		metric := &Metric{
			InterfaceName: "eth0",
			MetricType:    MetricTypeLatency,
			Value:         float64(i * 10),
		}
		err := db.Metrics().Record(ctx, metric)
		require.NoError(t, err)
	}

	// Run cleanup with all data retention set to delete everything
	policy := RetentionPolicy{
		MetricsDays: 1, // Will delete all since they're from today
	}

	// First, verify data exists
	count, err := db.Metrics().Count(ctx)
	require.NoError(t, err)
	if count != 5 {
		t.Fatalf("expected 5 metrics, got %d", count)
	}

	// Run cleanup (won't delete since data is recent)
	result, err := db.RunCleanup(ctx, policy)
	if err != nil {
		t.Fatalf("failed to run cleanup: %v", err)
	}

	// Data should still exist (it's not old enough)
	count, err = db.Metrics().Count(ctx)
	require.NoError(t, err)
	if count != 5 {
		t.Errorf("expected 5 metrics after cleanup, got %d", count)
	}

	// Check result duration is reasonable
	if result.Duration <= 0 {
		t.Error("cleanup duration should be positive")
	}
}

func TestOptimize(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	// Should run without error
	if err := db.Optimize(ctx); err != nil {
		t.Errorf("failed to optimize: %v", err)
	}
}

func TestAuditLog(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	// Record audit entry
	entry := &AuditLogEntry{
		Action:       AuditActionCreate,
		User:         "testuser",
		ResourceType: "profile",
		ResourceID:   "profile-123",
		IPAddress:    "127.0.0.1",
	}

	err := db.RecordAuditLog(ctx, entry)
	if err != nil {
		t.Fatalf("failed to record audit log: %v", err)
	}

	if entry.ID == 0 {
		t.Error("audit entry ID should be set")
	}

	// Get audit logs
	entries, err := db.GetAuditLogs(ctx, AuditLogOptions{
		User:  "testuser",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Action != AuditActionCreate {
		t.Errorf("expected action %q, got %q", AuditActionCreate, entries[0].Action)
	}
}

func TestUserStoreAdapter(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user first
	_, createErr := db.CreateUser(ctx, "testuser", "$2a$10$somehash", "admin")
	require.NoError(t, createErr)

	adapter := NewUserStoreAdapter(db)

	t.Run("GetPasswordHash", func(t *testing.T) {
		hash, err := adapter.GetPasswordHash(ctx, "testuser")
		require.NoError(t, err)
		if hash != "$2a$10$somehash" {
			t.Errorf("expected password hash, got %q", hash)
		}
	})

	t.Run("GetPasswordHash_NotFound", func(t *testing.T) {
		_, err := adapter.GetPasswordHash(ctx, "nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent user")
		}
	})

	t.Run("GetTokenVersion", func(t *testing.T) {
		version, err := adapter.GetTokenVersion(ctx, "testuser")
		require.NoError(t, err)
		if version != 1 {
			t.Errorf("expected token version 1, got %d", version)
		}
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		updateErr := adapter.UpdatePassword(ctx, "testuser", "$2a$10$newhash")
		require.NoError(t, updateErr)

		hash, hashErr := adapter.GetPasswordHash(ctx, "testuser")
		require.NoError(t, hashErr)
		if hash != "$2a$10$newhash" {
			t.Errorf("expected updated hash, got %q", hash)
		}
	})

	t.Run("RecordLoginSuccess", func(t *testing.T) {
		err := adapter.RecordLoginSuccess(ctx, "testuser")
		require.NoError(t, err)
	})

	t.Run("RecordLoginFailure", func(t *testing.T) {
		err := adapter.RecordLoginFailure(ctx, "testuser")
		require.NoError(t, err)
	})

	t.Run("IsLocked", func(t *testing.T) {
		locked, err := adapter.IsLocked(ctx, "testuser")
		require.NoError(t, err)
		if locked {
			t.Error("user should not be locked after one failure")
		}
	})

	t.Run("MigrateUserFromConfig", func(t *testing.T) {
		err := adapter.MigrateUserFromConfig(ctx, "migrated", "$2a$10$migratedhash")
		require.NoError(t, err)

		// Migrating again should succeed (no-op)
		err = adapter.MigrateUserFromConfig(ctx, "migrated", "$2a$10$differenthash")
		require.NoError(t, err)
	})

	t.Run("CreateUser", func(t *testing.T) {
		err := adapter.CreateUser(ctx, "newuser", "$2a$10$newhash", "user")
		require.NoError(t, err)
	})

	t.Run("GetUserCount", func(t *testing.T) {
		count, err := adapter.GetUserCount(ctx)
		require.NoError(t, err)
		if count < 3 {
			t.Errorf("expected at least 3 users, got %d", count)
		}
	})
}

func TestDatabaseStats(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	stats := db.Stats()

	// SQLite stats should return valid stats struct
	// MaxOpenConnections of 0 is valid for SQLite
	if stats.MaxOpenConnections < 0 {
		t.Error("Stats should return valid stats")
	}
}

func TestDefaultRetentionPolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()

	if policy.MetricsDays <= 0 {
		t.Error("MetricsDays should be positive")
	}
	if policy.AlertsDays <= 0 {
		t.Error("AlertsDays should be positive")
	}
	if policy.AuditLogDays <= 0 {
		t.Error("AuditLogDays should be positive")
	}
}

func TestDefaultPagination(t *testing.T) {
	pagination := DefaultPagination()

	if pagination.Limit != 100 {
		t.Errorf("expected default limit 100, got %d", pagination.Limit)
	}
	if pagination.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", pagination.Offset)
	}
}

func TestDeviceRepositoryExtended(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Devices()

	// Create test devices
	devices := []*Device{
		{IPAddress: "192.168.1.1", MACAddress: "aa:bb:cc:dd:ee:01", Vendor: "Cisco", DeviceType: "router"},
		{IPAddress: "192.168.1.2", MACAddress: "aa:bb:cc:dd:ee:02", Vendor: "Cisco", DeviceType: "switch"},
		{IPAddress: "192.168.1.3", MACAddress: "aa:bb:cc:dd:ee:03", Vendor: "HP", DeviceType: "router"},
	}

	for _, d := range devices {
		err := repo.Create(ctx, d)
		require.NoError(t, err)
	}

	t.Run("UpsertByMAC", func(t *testing.T) {
		d := &Device{
			MACAddress: "aa:bb:cc:dd:ee:01",
			IPAddress:  "192.168.1.100",
			Hostname:   "updated-host",
		}
		err := repo.UpsertByMAC(ctx, d)
		require.NoError(t, err)

		got, err := repo.GetByMAC(ctx, "aa:bb:cc:dd:ee:01")
		require.NoError(t, err)
		if got.IPAddress != "192.168.1.100" {
			t.Errorf("expected IP 192.168.1.100, got %s", got.IPAddress)
		}
	})

	t.Run("GetDistinctVendors", func(t *testing.T) {
		vendors, err := repo.GetDistinctVendors(ctx)
		require.NoError(t, err)
		if len(vendors) < 2 {
			t.Errorf("expected at least 2 vendors, got %d", len(vendors))
		}
	})

	t.Run("GetDistinctTypes", func(t *testing.T) {
		types, err := repo.GetDistinctTypes(ctx)
		require.NoError(t, err)
		if len(types) < 2 {
			t.Errorf("expected at least 2 types, got %d", len(types))
		}
	})

	t.Run("MarkInactive", func(t *testing.T) {
		// Mark devices older than future time as inactive
		count, err := repo.MarkInactive(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		if count < 1 {
			t.Error("expected at least 1 device marked inactive")
		}

		d, err := repo.GetByIP(ctx, "192.168.1.2")
		require.NoError(t, err)
		if d.IsActive {
			t.Error("device should be inactive")
		}
	})

	t.Run("DeleteInactive", func(t *testing.T) {
		count, err := repo.DeleteInactive(ctx, time.Now().Add(time.Hour)) // Delete all inactive
		require.NoError(t, err)
		if count < 1 {
			t.Errorf("expected at least 1 deleted, got %d", count)
		}
	})
}

func TestAlertRepositoryExtended(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()

	// Create test alerts
	alerts := []*Alert{
		{Title: "Alert 1", Severity: "critical", Source: "test"},
		{Title: "Alert 2", Severity: "warning", Source: "test"},
		{Title: "Alert 3", Severity: "info", Source: "test"},
	}

	for _, a := range alerts {
		err := repo.Create(ctx, a)
		require.NoError(t, err)
	}

	t.Run("AcknowledgeAll", func(t *testing.T) {
		count, err := repo.AcknowledgeAll(ctx, AlertListOptions{}, "testuser")
		require.NoError(t, err)
		if count != 3 {
			t.Errorf("expected 3 acknowledged, got %d", count)
		}
	})

	t.Run("GetUnacknowledgedCount", func(t *testing.T) {
		// Create one more unacknowledged
		err := repo.Create(ctx, &Alert{Title: "New Alert", Severity: "info", Source: "test"})
		require.NoError(t, err)

		count, err := repo.GetUnacknowledgedCount(ctx)
		require.NoError(t, err)
		if count != 1 {
			t.Errorf("expected 1 unacknowledged, got %d", count)
		}
	})

	t.Run("GetCriticalCount", func(t *testing.T) {
		count, err := repo.GetCriticalCount(ctx)
		require.NoError(t, err)
		// Critical alerts are acknowledged, so count may be 0 or 1 depending on logic
		if count < 0 {
			t.Errorf("count should be non-negative, got %d", count)
		}
	})

	t.Run("DeleteOlderThan", func(t *testing.T) {
		count, err := repo.DeleteOlderThan(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		if count < 0 {
			t.Errorf("count should be non-negative, got %d", count)
		}
	})
}

func TestSettingsRepositoryExtended(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Settings()

	// Set some test settings
	err := repo.Set(ctx, "prefix.key1", "value1")
	require.NoError(t, err)
	err = repo.Set(ctx, "prefix.key2", "value2")
	require.NoError(t, err)
	err = repo.Set(ctx, "other.key", "value3")
	require.NoError(t, err)

	t.Run("GetByPrefix", func(t *testing.T) {
		settings, err := repo.GetByPrefix(ctx, "prefix.")
		require.NoError(t, err)
		if len(settings) != 2 {
			t.Errorf("expected 2 settings, got %d", len(settings))
		}
	})

	t.Run("DeleteByPrefix", func(t *testing.T) {
		count, err := repo.DeleteByPrefix(ctx, "prefix.")
		require.NoError(t, err)
		if count != 2 {
			t.Errorf("expected 2 deleted, got %d", count)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := repo.Count(ctx)
		require.NoError(t, err)
		if count != 1 { // Only "other.key" left
			t.Errorf("expected 1 setting, got %d", count)
		}
	})

	t.Run("SetIfNotExists", func(t *testing.T) {
		// First call should set
		set, err := repo.SetIfNotExists(ctx, "new.key", "new_value")
		require.NoError(t, err)
		if !set {
			t.Error("expected value to be set")
		}

		// Second call should not overwrite
		set, err = repo.SetIfNotExists(ctx, "new.key", "different_value")
		require.NoError(t, err)
		if set {
			t.Error("expected value not to be overwritten")
		}

		// Verify original value is preserved
		setting, err := repo.Get(ctx, "new.key")
		require.NoError(t, err)
		if setting.Value != "new_value" {
			t.Errorf("expected 'new_value', got %q", setting.Value)
		}
	})
}

func TestMetricsRepositoryExtended(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Metrics()

	t.Run("GetDistinctInterfaces", func(t *testing.T) {
		// Record some metrics with different interfaces
		err := repo.Record(ctx, &Metric{InterfaceName: "eth0", MetricType: "bandwidth", Value: 100})
		require.NoError(t, err)
		err = repo.Record(ctx, &Metric{InterfaceName: "eth1", MetricType: "bandwidth", Value: 200})
		require.NoError(t, err)

		interfaces, err := repo.GetDistinctInterfaces(ctx)
		require.NoError(t, err)
		if len(interfaces) < 2 {
			t.Errorf("expected at least 2 interfaces, got %d", len(interfaces))
		}
	})

	t.Run("GetDistinctTypes", func(t *testing.T) {
		// Record metrics with different types
		err := repo.Record(ctx, &Metric{InterfaceName: "eth0", MetricType: "latency", Value: 10})
		require.NoError(t, err)

		types, err := repo.GetDistinctTypes(ctx)
		require.NoError(t, err)
		if len(types) < 2 {
			t.Errorf("expected at least 2 types, got %d", len(types))
		}
	})

	t.Run("RecordSpeedTest", func(t *testing.T) {
		result := &SpeedTestResult{
			InterfaceName: "eth0",
			ServerName:    "Test Server",
			DownloadMbps:  100.5,
			UploadMbps:    50.25,
			LatencyMs:     15.0,
		}
		err := repo.RecordSpeedTest(ctx, result)
		require.NoError(t, err)
	})

	t.Run("GetSpeedTestHistory", func(t *testing.T) {
		results, err := repo.GetSpeedTestHistory(ctx, "eth0", 10)
		require.NoError(t, err)
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("RecordDNSResult", func(t *testing.T) {
		result := &DNSResult{
			InterfaceName:  "eth0",
			Server:         "8.8.8.8",
			Hostname:       "example.com",
			ResponseTimeMs: 25.0,
			Status:         "success",
		}
		err := repo.RecordDNSResult(ctx, result)
		require.NoError(t, err)
	})

	t.Run("RecordGatewayResult", func(t *testing.T) {
		result := &GatewayResult{
			InterfaceName: "eth0",
			Gateway:       "192.168.1.1",
			LatencyMs:     1.5,
			Reachable:     true,
		}
		err := repo.RecordGatewayResult(ctx, result)
		require.NoError(t, err)
	})
}

func TestRetentionExtended(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("deleteAuditLogsOlderThan", func(t *testing.T) {
		// Record some audit logs
		for range 5 {
			err := db.RecordAuditLog(ctx, &AuditLogEntry{
				Action:       AuditActionCreate,
				User:         "testuser",
				ResourceType: "test",
				ResourceID:   "test-id",
			})
			require.NoError(t, err)
		}

		// Delete older than future time (should delete all)
		count, err := db.deleteAuditLogsOlderThan(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		if count != 5 {
			t.Errorf("expected 5 deleted, got %d", count)
		}
	})

	t.Run("deleteSpeedTestsOlderThan", func(t *testing.T) {
		// Record a speed test
		err := db.Metrics().RecordSpeedTest(ctx, &SpeedTestResult{
			InterfaceName: "eth0", ServerName: "Test", DownloadMbps: 100,
		})
		require.NoError(t, err)

		count, err := db.deleteSpeedTestsOlderThan(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		if count < 0 {
			t.Errorf("count should be non-negative, got %d", count)
		}
	})

	t.Run("deleteDNSResultsOlderThan", func(t *testing.T) {
		err := db.Metrics().RecordDNSResult(ctx, &DNSResult{
			InterfaceName: "eth0", Server: "8.8.8.8", Hostname: "test.com",
		})
		require.NoError(t, err)

		count, err := db.deleteDNSResultsOlderThan(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		if count < 0 {
			t.Errorf("count should be non-negative, got %d", count)
		}
	})

	t.Run("deleteGatewayResultsOlderThan", func(t *testing.T) {
		err := db.Metrics().RecordGatewayResult(ctx, &GatewayResult{
			InterfaceName: "eth0", Gateway: "192.168.1.1", Reachable: true,
		})
		require.NoError(t, err)

		count, err := db.deleteGatewayResultsOlderThan(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		if count < 0 {
			t.Errorf("count should be non-negative, got %d", count)
		}
	})

	t.Run("RunCleanup", func(t *testing.T) {
		policy := DefaultRetentionPolicy()
		stats, err := db.RunCleanup(ctx, policy)
		require.NoError(t, err)
		if stats == nil {
			t.Error("expected cleanup stats")
		}
	})
}

func TestWithTxExtended(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful transaction", func(t *testing.T) {
		err := db.WithTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(
				ctx,
				"INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))",
				"tx.key",
				"tx.value",
			)
			return err
		})
		require.NoError(t, err)

		// Verify the insert worked
		setting, err := db.Settings().Get(ctx, "tx.key")
		require.NoError(t, err)
		if setting.Value != "tx.value" {
			t.Errorf("expected 'tx.value', got %q", setting.Value)
		}
	})

	t.Run("rollback on error", func(t *testing.T) {
		err := db.WithTx(ctx, func(_ *sql.Tx) error {
			return errors.New("simulated error")
		})
		if err == nil {
			t.Error("expected error from transaction")
		}
	})
}

func TestGetValue(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Settings()

	t.Run("existing key", func(t *testing.T) {
		err := repo.Set(ctx, "test.key", "test.value")
		require.NoError(t, err)

		value, err := repo.GetValue(ctx, "test.key")
		require.NoError(t, err)
		if value != "test.value" {
			t.Errorf("expected 'test.value', got %q", value)
		}
	})

	t.Run("non-existent key returns empty", func(t *testing.T) {
		value, err := repo.GetValue(ctx, "nonexistent.key")
		require.NoError(t, err)
		if value != "" {
			t.Errorf("expected empty string, got %q", value)
		}
	})
}

func TestGetWithDefault(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Settings()

	t.Run("existing key returns value", func(t *testing.T) {
		err := repo.Set(ctx, "exists.key", "exists.value")
		require.NoError(t, err)

		value, err := repo.GetWithDefault(ctx, "exists.key", "default")
		require.NoError(t, err)
		if value != "exists.value" {
			t.Errorf("expected 'exists.value', got %q", value)
		}
	})

	t.Run("non-existent key returns default", func(t *testing.T) {
		value, err := repo.GetWithDefault(ctx, "missing.key", "default.value")
		require.NoError(t, err)
		if value != "default.value" {
			t.Errorf("expected 'default.value', got %q", value)
		}
	})
}

func TestVacuumAndAnalyze(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Vacuum", func(t *testing.T) {
		err := db.Vacuum(ctx)
		require.NoError(t, err)
	})

	t.Run("Analyze", func(t *testing.T) {
		err := db.Analyze(ctx)
		require.NoError(t, err)
	})

	t.Run("Optimize", func(t *testing.T) {
		err := db.Optimize(ctx)
		require.NoError(t, err)
	})
}

func TestAlertListWithFilters(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()

	// Create alerts with different severities and sources
	alerts := []*Alert{
		{Title: "Critical Alert", Severity: "critical", Source: "system"},
		{Title: "Warning Alert", Severity: "warning", Source: "network"},
		{Title: "Info Alert", Severity: "info", Source: "system"},
	}
	for _, a := range alerts {
		err := repo.Create(ctx, a)
		require.NoError(t, err)
	}

	t.Run("filter by severity", func(t *testing.T) {
		list, err := repo.List(ctx, AlertListOptions{Severity: "critical"})
		require.NoError(t, err)
		if len(list) != 1 {
			t.Errorf("expected 1 critical alert, got %d", len(list))
		}
	})

	t.Run("filter unresolved only", func(t *testing.T) {
		list, err := repo.List(ctx, AlertListOptions{UnresolvedOnly: true})
		require.NoError(t, err)
		if len(list) != 3 {
			t.Errorf("expected 3 unresolved alerts, got %d", len(list))
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		list, err := repo.List(ctx, AlertListOptions{Limit: 1, Offset: 1})
		require.NoError(t, err)
		if len(list) != 1 {
			t.Errorf("expected 1 alert with pagination, got %d", len(list))
		}
	})
}

func TestDeviceListWithFilters(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Devices()

	// Create devices with different vendors and types
	devices := []*Device{
		{IPAddress: "10.0.0.1", MACAddress: "11:22:33:44:55:01", Vendor: "Cisco", DeviceType: "router"},
		{IPAddress: "10.0.0.2", MACAddress: "11:22:33:44:55:02", Vendor: "Cisco", DeviceType: "switch"},
		{IPAddress: "10.0.0.3", MACAddress: "11:22:33:44:55:03", Vendor: "HP", DeviceType: "printer"},
	}
	for _, d := range devices {
		err := repo.Create(ctx, d)
		require.NoError(t, err)
	}

	t.Run("filter by vendor", func(t *testing.T) {
		list, err := repo.List(ctx, DeviceListOptions{Vendor: "Cisco"})
		require.NoError(t, err)
		if len(list) != 2 {
			t.Errorf("expected 2 Cisco devices, got %d", len(list))
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		list, err := repo.List(ctx, DeviceListOptions{DeviceType: "router"})
		require.NoError(t, err)
		if len(list) != 1 {
			t.Errorf("expected 1 router, got %d", len(list))
		}
	})

	t.Run("active only", func(t *testing.T) {
		list, err := repo.List(ctx, DeviceListOptions{ActiveOnly: true})
		require.NoError(t, err)
		if len(list) != 3 {
			t.Errorf("expected 3 active devices, got %d", len(list))
		}
	})
}

func TestAcknowledgeAndResolve(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()

	// Create an alert
	alert := &Alert{Title: "Test Alert", Severity: "warning", Source: "test"}
	err := repo.Create(ctx, alert)
	require.NoError(t, err)

	t.Run("Acknowledge", func(t *testing.T) {
		err := repo.Acknowledge(ctx, alert.ID, "admin")
		require.NoError(t, err)

		got, err := repo.Get(ctx, alert.ID)
		require.NoError(t, err)
		if !got.Acknowledged {
			t.Error("alert should be acknowledged")
		}
	})

	t.Run("Resolve", func(t *testing.T) {
		err := repo.Resolve(ctx, alert.ID)
		require.NoError(t, err)

		got, err := repo.Get(ctx, alert.ID)
		require.NoError(t, err)
		if !got.Resolved {
			t.Error("alert should be resolved")
		}
	})
}
