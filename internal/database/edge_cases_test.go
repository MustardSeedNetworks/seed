package database_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/database"
)

func TestOpenWithConfigEdgeCases(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		cfg := database.Config{
			Path: "",
		}
		_, err := database.OpenWithConfig(cfg)
		if err == nil {
			t.Error("expected error for empty path")
		}
	})

	t.Run("with WAL disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test_no_wal.db")

		cfg := database.Config{
			Path:            dbPath,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: time.Minute,
			EnableWAL:       false,
			BusyTimeout:     1000,
		}

		db, err := database.OpenWithConfig(cfg)
		require.NoError(t, err)
		defer func() { _ = db.Close() }()
	})

	t.Run("with zero busy timeout", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test_no_timeout.db")

		cfg := database.Config{
			Path:            dbPath,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: time.Minute,
			EnableWAL:       true,
			BusyTimeout:     0,
		}

		db, err := database.OpenWithConfig(cfg)
		require.NoError(t, err)
		defer func() { _ = db.Close() }()
	})
}

func TestProfileRepositoryEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Profiles()

	t.Run("Create duplicate name", func(t *testing.T) {
		// Default profile already exists
		profile := &database.Profile{
			Name:        "Default",
			Description: "Duplicate",
			ConfigJSON:  `{}`,
		}

		err := repo.Create(ctx, profile)
		if !errors.Is(err, database.ErrProfileNameExists) {
			t.Errorf("expected ErrProfileNameExists, got %v", err)
		}
	})

	t.Run("Update with duplicate name", func(t *testing.T) {
		// Create a second profile
		profile := &database.Profile{
			Name:        "Second Profile",
			Description: "Second",
			ConfigJSON:  `{}`,
		}
		err := repo.Create(ctx, profile)
		require.NoError(t, err)

		// Try to update it with the name of the default profile
		profile.Name = "Default"
		err = repo.Update(ctx, profile)
		if !errors.Is(err, database.ErrProfileNameExists) {
			t.Errorf("expected ErrProfileNameExists, got %v", err)
		}
	})

	t.Run("Update non-existent profile", func(t *testing.T) {
		profile := &database.Profile{
			ID:          "non-existent-id",
			Name:        "Non-existent",
			Description: "Non-existent",
			ConfigJSON:  `{}`,
		}
		err := repo.Update(ctx, profile)
		if !errors.Is(err, database.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got %v", err)
		}
	})

	t.Run("Delete non-existent profile", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-id")
		if !errors.Is(err, database.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got %v", err)
		}
	})

	t.Run("SetDefault non-existent profile", func(t *testing.T) {
		err := repo.SetDefault(ctx, "non-existent-id")
		if !errors.Is(err, database.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got %v", err)
		}
	})

	t.Run("GetByName non-existent", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "non-existent-name")
		if !errors.Is(err, database.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got %v", err)
		}
	})
}

// testAlertNotFoundError tests that the given function returns ErrAlertNotFound.
func testAlertNotFoundError(t *testing.T, name string, fn func() error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		if err := fn(); !errors.Is(err, database.ErrAlertNotFound) {
			t.Errorf("expected ErrAlertNotFound, got %v", err)
		}
	})
}

func TestAlertRepositoryNotFoundErrors(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()

	testAlertNotFoundError(t, "Acknowledge", func() error { return repo.Acknowledge(ctx, 999999, "testuser") })
	testAlertNotFoundError(t, "Resolve", func() error { return repo.Resolve(ctx, 999999) })
	testAlertNotFoundError(t, "Delete", func() error { return repo.Delete(ctx, 999999) })
	testAlertNotFoundError(t, "Get", func() error { _, err := repo.Get(ctx, 999999); return err })
}

func TestAlertRepositoryFilters(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()
	deviceRepo := db.Devices()

	testDevice := &database.Device{IPAddress: "192.168.1.100", Hostname: "test-device"}
	require.NoError(t, deviceRepo.Create(ctx, testDevice))

	// Create test alert with device
	alert := &database.Alert{
		Type:     database.AlertTypeSecurity,
		Severity: database.AlertSeverityCritical,
		Title:    "Alert with device",
		Message:  "Test",
		Source:   "test",
		DeviceID: &testDevice.ID,
	}
	require.NoError(t, repo.Create(ctx, alert))

	t.Run("List with type filter", func(t *testing.T) {
		alerts, err := repo.List(ctx, database.AlertListOptions{Type: database.AlertTypeSecurity})
		require.NoError(t, err)
		for _, a := range alerts {
			require.Equal(t, database.AlertTypeSecurity, a.Type)
		}
	})

	t.Run("List with device ID filter", func(t *testing.T) {
		alerts, err := repo.List(ctx, database.AlertListOptions{DeviceID: testDevice.ID})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(alerts), 1)
	})

	t.Run("List with unacknowledged only", func(t *testing.T) {
		alerts, err := repo.List(ctx, database.AlertListOptions{UnacknowledgedOnly: true})
		require.NoError(t, err)
		for _, a := range alerts {
			require.False(t, a.Acknowledged)
		}
	})
}

func TestAlertRepositoryEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Alerts()
	deviceRepo := db.Devices()

	testDevice := &database.Device{IPAddress: "192.168.1.100", Hostname: "test-device"}
	require.NoError(t, deviceRepo.Create(ctx, testDevice))

	t.Run("Alert with device ID and metadata", func(t *testing.T) {
		alert := &database.Alert{
			Type:     database.AlertTypeSecurity,
			Severity: database.AlertSeverityCritical,
			Title:    "Alert with device",
			Message:  "Test message",
			Source:   "test",
			DeviceID: &testDevice.ID,
			Metadata: `{"extra": "data"}`,
		}
		require.NoError(t, repo.Create(ctx, alert))

		got, err := repo.Get(ctx, alert.ID)
		require.NoError(t, err)
		require.NotNil(t, got.DeviceID)
		require.Equal(t, testDevice.ID, *got.DeviceID)
		require.JSONEq(t, `{"extra": "data"}`, got.Metadata)
	})

	t.Run("AcknowledgeAll operations", func(t *testing.T) {
		// Create unacknowledged alert
		require.NoError(t, repo.Create(ctx, &database.Alert{
			Type:     database.AlertTypeSystem,
			Severity: database.AlertSeverityWarning,
			Title:    "Warning alert",
			Message:  "Test",
			Source:   "test",
		}))

		count, err := repo.AcknowledgeAll(ctx, database.AlertListOptions{
			Severity: database.AlertSeverityWarning,
		}, "admin")
		require.NoError(t, err)
		require.GreaterOrEqual(t, count, int64(1))
	})

	t.Run("Count with filters", func(t *testing.T) {
		count, err := repo.Count(ctx, database.AlertListOptions{
			Type:     database.AlertTypeSecurity,
			Severity: database.AlertSeverityCritical,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, count, int64(0))
	})
}

func TestDeviceRepositoryEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Devices()

	t.Run("Get non-existent device", func(t *testing.T) {
		_, err := repo.Get(ctx, "non-existent-id")
		if !errors.Is(err, database.ErrDeviceNotFound) {
			t.Errorf("expected ErrDeviceNotFound, got %v", err)
		}
	})

	t.Run("GetByIP non-existent", func(t *testing.T) {
		_, err := repo.GetByIP(ctx, "192.168.99.99")
		if !errors.Is(err, database.ErrDeviceNotFound) {
			t.Errorf("expected ErrDeviceNotFound, got %v", err)
		}
	})

	t.Run("GetByMAC non-existent", func(t *testing.T) {
		_, err := repo.GetByMAC(ctx, "ff:ff:ff:ff:ff:ff")
		if !errors.Is(err, database.ErrDeviceNotFound) {
			t.Errorf("expected ErrDeviceNotFound, got %v", err)
		}
	})

	t.Run("Update non-existent device", func(t *testing.T) {
		device := &database.Device{
			ID:        "non-existent-id",
			IPAddress: "192.168.1.1",
		}
		err := repo.Update(ctx, device)
		if !errors.Is(err, database.ErrDeviceNotFound) {
			t.Errorf("expected ErrDeviceNotFound, got %v", err)
		}
	})

	t.Run("Delete non-existent device", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-id")
		if !errors.Is(err, database.ErrDeviceNotFound) {
			t.Errorf("expected ErrDeviceNotFound, got %v", err)
		}
	})

	t.Run("UpsertByMAC with empty MAC", func(t *testing.T) {
		device := &database.Device{
			IPAddress: "10.0.0.99",
			Hostname:  "no-mac-device",
		}
		err := repo.UpsertByMAC(ctx, device)
		require.NoError(t, err) // Should fall back to Upsert
	})

	t.Run("List with SeenAfter filter", func(t *testing.T) {
		// Create a device first
		device := &database.Device{
			IPAddress:  "10.0.0.100",
			MACAddress: "aa:bb:cc:00:00:01",
			Hostname:   "seen-device",
		}
		err := repo.Create(ctx, device)
		require.NoError(t, err)

		seenAfter := time.Now().Add(-time.Hour)
		devices, err := repo.List(ctx, database.DeviceListOptions{
			SeenAfter: seenAfter,
		})
		require.NoError(t, err)
		if len(devices) < 1 {
			t.Error("expected at least 1 device seen after time")
		}
	})

	t.Run("List with limit and offset", func(t *testing.T) {
		// Create a few more devices
		for i := range 3 {
			device := &database.Device{
				IPAddress:  "10.0.0." + string(rune('1'+i)),
				MACAddress: "aa:bb:cc:00:00:0" + string(rune('1'+i)),
			}
			_ = repo.Create(ctx, device)
		}

		devices, err := repo.List(ctx, database.DeviceListOptions{
			Limit:  2,
			Offset: 1,
		})
		require.NoError(t, err)
		if len(devices) > 2 {
			t.Errorf("expected at most 2 devices, got %d", len(devices))
		}
	})
}

func TestSettingsRepositoryEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Settings()

	t.Run("Delete non-existent setting", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-key")
		if !errors.Is(err, database.ErrSettingNotFound) {
			t.Errorf("expected ErrSettingNotFound, got %v", err)
		}
	})
}

func TestMetricsRepositoryEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Metrics()

	t.Run("GetLatest for non-existent interface", func(t *testing.T) {
		result, err := repo.GetLatest(ctx, "non-existent-interface", "latency")
		require.NoError(t, err)
		if result != nil {
			t.Error("expected nil result for non-existent interface")
		}
	})

	t.Run("Query with time range", func(t *testing.T) {
		// First create some metrics
		err := repo.Record(ctx, &database.Metric{
			InterfaceName: "eth0",
			MetricType:    "bandwidth",
			Value:         100,
		})
		require.NoError(t, err)

		start := time.Now().Add(-time.Hour)
		end := time.Now().Add(time.Hour)

		metrics, err := repo.Query(ctx, database.MetricQueryOptions{
			TimeRange: database.TimeRange{
				Start: start,
				End:   end,
			},
		})
		require.NoError(t, err)
		if len(metrics) < 1 {
			t.Error("expected at least 1 metric in time range")
		}
	})

	t.Run("Query with offset", func(t *testing.T) {
		metrics, err := repo.Query(ctx, database.MetricQueryOptions{
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)
		if len(metrics) < 1 {
			t.Error("expected at least 1 metric")
		}
	})

	t.Run("GetAggregates with time range", func(t *testing.T) {
		start := time.Now().Add(-time.Hour)
		end := time.Now().Add(time.Hour)

		agg, err := repo.GetAggregates(ctx, database.MetricAggregateOptions{
			InterfaceName: "eth0",
			MetricType:    "bandwidth",
			TimeRange: database.TimeRange{
				Start: start,
				End:   end,
			},
		})
		require.NoError(t, err)
		if agg == nil {
			t.Error("expected non-nil aggregate")
		}
	})

	t.Run("GetSpeedTestHistory with zero limit", func(t *testing.T) {
		// Record a speed test first
		err := repo.RecordSpeedTest(ctx, &database.SpeedTestResult{
			InterfaceName: "eth1",
			ServerName:    "Test Server",
			DownloadMbps:  100,
			UploadMbps:    50,
			LatencyMs:     10,
		})
		require.NoError(t, err)

		// Zero limit should use default
		results, err := repo.GetSpeedTestHistory(ctx, "eth1", 0)
		require.NoError(t, err)
		if len(results) < 1 {
			t.Error("expected at least 1 speed test result")
		}
	})
}

func TestAuditLogEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("RecordAuditLog with zero timestamp", func(t *testing.T) {
		entry := &database.AuditLogEntry{
			Action:       database.AuditActionUpdate,
			User:         "testuser",
			ResourceType: "profile",
			ResourceID:   "profile-1",
		}

		err := db.RecordAuditLog(ctx, entry)
		require.NoError(t, err)

		if entry.Timestamp.IsZero() {
			t.Error("expected timestamp to be set")
		}
	})

	t.Run("GetAuditLogs with all filters", func(t *testing.T) {
		since := time.Now().Add(-time.Hour)
		logs, err := db.GetAuditLogs(ctx, database.AuditLogOptions{
			Action:       database.AuditActionUpdate,
			User:         "testuser",
			ResourceType: "profile",
			ResourceID:   "profile-1",
			Since:        since,
			Limit:        10,
			Offset:       0,
		})
		require.NoError(t, err)
		if len(logs) < 1 {
			t.Error("expected at least 1 audit log")
		}
	})
}

func TestRetentionEdgeCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("RunCleanup with zero retention days skips cleanup", func(t *testing.T) {
		policy := database.RetentionPolicy{
			MetricsDays:        0, // Should skip
			AlertsDays:         0,
			InactiveDeviceDays: 0,
			AuditLogDays:       0,
			SpeedTestDays:      0,
			DNSResultDays:      0,
			GatewayResultDays:  0,
		}

		result, err := db.RunCleanup(ctx, policy)
		require.NoError(t, err)
		require.NotNil(t, result, "expected non-nil result")

		// All counts should be 0 when retention days is 0
		if result.MetricsDeleted != 0 || result.AlertsDeleted != 0 {
			t.Error("expected 0 deletions when retention days is 0")
		}
	})
}

// closedDBTestCase defines a test case for operations on a closed database.
type closedDBTestCase struct {
	name string
	fn   func(ctx context.Context, db *database.DB) error
}

// getClosedDBTestCases returns all test cases for closed database operations.
func getClosedDBTestCases() []closedDBTestCase {
	return []closedDBTestCase{
		{"Exec", func(ctx context.Context, db *database.DB) error {
			_, err := db.Exec(ctx, "SELECT 1")
			return err
		}},
		{"Query", func(ctx context.Context, db *database.DB) error {
			rows, err := db.Query(ctx, "SELECT 1")
			if err != nil {
				return err
			}
			defer func() { _ = rows.Close() }()
			return rows.Err()
		}},
		{"BeginTx", func(ctx context.Context, db *database.DB) error {
			_, err := db.BeginTx(ctx, nil)
			return err
		}},
		{"SchemaVersion", func(ctx context.Context, db *database.DB) error {
			_, err := db.SchemaVersion(ctx)
			return err
		}},
		{"MigrationStatus", func(ctx context.Context, db *database.DB) error {
			_, err := db.MigrationStatus(ctx)
			return err
		}},
		{"Vacuum", func(ctx context.Context, db *database.DB) error {
			return db.Vacuum(ctx)
		}},
		{"Analyze", func(ctx context.Context, db *database.DB) error {
			return db.Analyze(ctx)
		}},
		{"GetTokenVersion", func(ctx context.Context, db *database.DB) error {
			_, err := db.GetTokenVersion(ctx, "testuser")
			return err
		}},
		{"IsUserLocked", func(ctx context.Context, db *database.DB) error {
			_, err := db.IsUserLocked(ctx, "testuser")
			return err
		}},
		{"RecordLoginSuccess", func(ctx context.Context, db *database.DB) error {
			return db.RecordLoginSuccess(ctx, "testuser")
		}},
		{"RecordLoginFailure", func(ctx context.Context, db *database.DB) error {
			_, err := db.RecordLoginFailure(ctx, "testuser", 5, time.Minute)
			return err
		}},
		{"IncrementTokenVersion", func(ctx context.Context, db *database.DB) error {
			return db.IncrementTokenVersion(ctx, "testuser")
		}},
	}
}

func TestClosedDatabaseOperations(t *testing.T) {
	db, cleanup := testDB(t)
	_ = db.Close()
	defer cleanup()

	ctx := context.Background()
	for _, tc := range getClosedDBTestCases() {
		t.Run(tc.name+" on closed database", func(t *testing.T) {
			if err := tc.fn(ctx, db); err == nil {
				t.Errorf("expected error for %s on closed database", tc.name)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := database.DefaultConfig("/path/to/db.db")

	if cfg.Path != "/path/to/db.db" {
		t.Errorf("expected path /path/to/db.db, got %s", cfg.Path)
	}
	if cfg.MaxOpenConns <= 0 {
		t.Error("expected positive MaxOpenConns")
	}
	if cfg.MaxIdleConns <= 0 {
		t.Error("expected positive MaxIdleConns")
	}
	if cfg.ConnMaxLifetime <= 0 {
		t.Error("expected positive ConnMaxLifetime")
	}
	if cfg.RetentionDays <= 0 {
		t.Error("expected positive RetentionDays")
	}
	if !cfg.EnableWAL {
		t.Error("expected EnableWAL to be true")
	}
	if cfg.BusyTimeout <= 0 {
		t.Error("expected positive BusyTimeout")
	}
}
