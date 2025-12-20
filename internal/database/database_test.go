package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testDB creates a temporary database for testing.
func testDB(t *testing.T) (db *DB, cleanup func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var err error
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup = func() {
		db.Close()
		os.Remove(dbPath)
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
	defer db.Close()

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
	defer db.Close()

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

//nolint:gocyclo // Test functions require comprehensive scenario coverage
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

	got, _ = repo.Get(ctx, profile.ID)
	if got.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got.Description)
	}

	// List
	profiles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("failed to list profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}

	// Set default
	err = repo.SetDefault(ctx, profile.ID)
	if err != nil {
		t.Fatalf("failed to set default: %v", err)
	}

	got, _ = repo.GetDefault(ctx)
	if got.ID != profile.ID {
		t.Errorf("expected default profile %q, got %q", profile.ID, got.ID)
	}

	// Count
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Delete
	err = repo.Delete(ctx, profile.ID)
	if err != nil {
		t.Fatalf("failed to delete profile: %v", err)
	}

	_, err = repo.Get(ctx, profile.ID)
	if err != ErrProfileNotFound {
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

	count, _ := repo.Count(ctx)
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
	count, _ = repo.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0 remaining metrics, got %d", count)
	}
}

//nolint:gocyclo // Test functions require comprehensive scenario coverage
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

	got, _ = repo.GetByIP(ctx, "192.168.1.100")
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
	if err != ErrDeviceNotFound {
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

	got, _ = repo.Get(ctx, alert.ID)
	if !got.Acknowledged {
		t.Error("alert should be acknowledged")
	}

	// Resolve
	err = repo.Resolve(ctx, alert.ID)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	got, _ = repo.Get(ctx, alert.ID)
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
	if err != ErrAlertNotFound {
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
	if err != ErrSettingNotFound {
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
		db.Metrics().Record(ctx, metric)
	}

	// Run cleanup with all data retention set to delete everything
	policy := RetentionPolicy{
		MetricsDays: 1, // Will delete all since they're from today
	}

	// First, verify data exists
	count, _ := db.Metrics().Count(ctx)
	if count != 5 {
		t.Fatalf("expected 5 metrics, got %d", count)
	}

	// Run cleanup (won't delete since data is recent)
	result, err := db.RunCleanup(ctx, policy)
	if err != nil {
		t.Fatalf("failed to run cleanup: %v", err)
	}

	// Data should still exist (it's not old enough)
	count, _ = db.Metrics().Count(ctx)
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
