// Package database provides schema migrations for The Seed database.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migration represents a database schema migration.
type Migration struct {
	Version     int
	Description string
	Up          string
}

// migrations is the list of all database migrations in order.
// IMPORTANT: Never modify existing migrations, only add new ones.
var migrations = []Migration{
	{
		Version:     1,
		Description: "Create schema version table",
		Up: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at TEXT NOT NULL,
				description TEXT
			);
		`,
	},
	{
		Version:     2,
		Description: "Create profiles table",
		Up: `
			CREATE TABLE IF NOT EXISTS profiles (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL UNIQUE,
				description TEXT,
				config_json TEXT NOT NULL,
				is_default INTEGER DEFAULT 0,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_profiles_name ON profiles(name);
			CREATE INDEX IF NOT EXISTS idx_profiles_is_default ON profiles(is_default);
		`,
	},
	{
		Version:     3,
		Description: "Create metrics table for historical data",
		Up: `
			CREATE TABLE IF NOT EXISTS metrics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				metric_type TEXT NOT NULL,
				value REAL NOT NULL,
				unit TEXT,
				timestamp TEXT NOT NULL,
				metadata_json TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_metrics_interface ON metrics(interface_name);
			CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(metric_type);
			CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
			CREATE INDEX IF NOT EXISTS idx_metrics_interface_type_time ON metrics(interface_name, metric_type, timestamp);
		`,
	},
	{
		Version:     4,
		Description: "Create devices table for discovered devices",
		Up: `
			CREATE TABLE IF NOT EXISTS devices (
				id TEXT PRIMARY KEY,
				ip_address TEXT NOT NULL,
				mac_address TEXT,
				hostname TEXT,
				vendor TEXT,
				device_type TEXT,
				os_family TEXT,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				is_active INTEGER DEFAULT 1,
				ports_json TEXT,
				metadata_json TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices(ip_address);
			CREATE INDEX IF NOT EXISTS idx_devices_mac ON devices(mac_address);
			CREATE INDEX IF NOT EXISTS idx_devices_hostname ON devices(hostname);
			CREATE INDEX IF NOT EXISTS idx_devices_active ON devices(is_active);
			CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);
		`,
	},
	{
		Version:     5,
		Description: "Create alerts table",
		Up: `
			CREATE TABLE IF NOT EXISTS alerts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				type TEXT NOT NULL,
				severity TEXT NOT NULL,
				title TEXT NOT NULL,
				message TEXT NOT NULL,
				source TEXT,
				device_id TEXT,
				acknowledged INTEGER DEFAULT 0,
				acknowledged_by TEXT,
				acknowledged_at TEXT,
				resolved INTEGER DEFAULT 0,
				resolved_at TEXT,
				created_at TEXT NOT NULL,
				metadata_json TEXT,
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL
			);

			CREATE INDEX IF NOT EXISTS idx_alerts_type ON alerts(type);
			CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
			CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged ON alerts(acknowledged);
			CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved);
			CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at);
			CREATE INDEX IF NOT EXISTS idx_alerts_device ON alerts(device_id);
		`,
	},
	{
		Version:     6,
		Description: "Create settings table for key-value settings",
		Up: `
			CREATE TABLE IF NOT EXISTS settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);
		`,
	},
	{
		Version:     7,
		Description: "Create speed test results table",
		Up: `
			CREATE TABLE IF NOT EXISTS speedtest_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				server_name TEXT,
				server_location TEXT,
				download_mbps REAL,
				upload_mbps REAL,
				latency_ms REAL,
				jitter_ms REAL,
				packet_loss REAL,
				timestamp TEXT NOT NULL,
				metadata_json TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_speedtest_interface ON speedtest_results(interface_name);
			CREATE INDEX IF NOT EXISTS idx_speedtest_timestamp ON speedtest_results(timestamp);
		`,
	},
	{
		Version:     8,
		Description: "Create wifi survey samples table",
		Up: `
			CREATE TABLE IF NOT EXISTS survey_samples (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				survey_id TEXT NOT NULL,
				x REAL NOT NULL,
				y REAL NOT NULL,
				signal_dbm INTEGER,
				noise_dbm INTEGER,
				snr_db INTEGER,
				channel INTEGER,
				frequency_mhz INTEGER,
				bssid TEXT,
				ssid TEXT,
				timestamp TEXT NOT NULL,
				networks_json TEXT,
				metadata_json TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_survey_samples_survey ON survey_samples(survey_id);
			CREATE INDEX IF NOT EXISTS idx_survey_samples_coords ON survey_samples(x, y);
		`,
	},
	{
		Version:     9,
		Description: "Create dns results table",
		Up: `
			CREATE TABLE IF NOT EXISTS dns_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				server TEXT NOT NULL,
				hostname TEXT NOT NULL,
				response_time_ms REAL,
				resolved_ip TEXT,
				status TEXT NOT NULL,
				error_message TEXT,
				timestamp TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_dns_interface ON dns_results(interface_name);
			CREATE INDEX IF NOT EXISTS idx_dns_server ON dns_results(server);
			CREATE INDEX IF NOT EXISTS idx_dns_timestamp ON dns_results(timestamp);
		`,
	},
	{
		Version:     10,
		Description: "Create gateway ping results table",
		Up: `
			CREATE TABLE IF NOT EXISTS gateway_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				gateway TEXT NOT NULL,
				latency_ms REAL,
				packet_loss REAL,
				reachable INTEGER,
				timestamp TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_gateway_interface ON gateway_results(interface_name);
			CREATE INDEX IF NOT EXISTS idx_gateway_timestamp ON gateway_results(timestamp);
		`,
	},
	{
		Version:     11,
		Description: "Create audit log table",
		Up: `
			CREATE TABLE IF NOT EXISTS audit_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				action TEXT NOT NULL,
				user TEXT,
				resource_type TEXT,
				resource_id TEXT,
				old_value_json TEXT,
				new_value_json TEXT,
				ip_address TEXT,
				user_agent TEXT,
				timestamp TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
			CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user);
			CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
			CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log(resource_type, resource_id);
		`,
	},
}

// migrate runs all pending migrations.
func (db *DB) migrate() error {
	ctx := context.Background()

	// Ensure schema_migrations table exists (migration 1)
	_, err := db.conn.ExecContext(ctx, migrations[0].Up)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := db.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Run pending migrations
	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}

		if err := db.runMigration(ctx, m); err != nil {
			return fmt.Errorf("failed to run migration %d (%s): %w", m.Version, m.Description, err)
		}
	}

	return nil
}

// getCurrentVersion returns the current schema version.
func (db *DB) getCurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM schema_migrations
	`).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// runMigration executes a single migration within a transaction.
func (db *DB) runMigration(ctx context.Context, m Migration) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				// Log rollback error but don't override original error
				_ = rbErr // Original error already being returned
			}
		}
	}()

	// Execute migration SQL
	if _, err = tx.ExecContext(ctx, m.Up); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (version, applied_at, description)
		VALUES (?, ?, ?)
	`, m.Version, now, m.Description)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// MigrationStatus returns the status of all migrations.
func (db *DB) MigrationStatus(ctx context.Context) ([]MigrationInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	// Get applied migrations
	rows, err := db.conn.QueryContext(ctx, `
		SELECT version, applied_at, description FROM schema_migrations ORDER BY version
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]time.Time)
	for rows.Next() {
		var version int
		var appliedAt string
		var desc sql.NullString
		if err := rows.Scan(&version, &appliedAt, &desc); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		t, parseErr := time.Parse(time.RFC3339, appliedAt)
		if parseErr != nil {
			// Fallback to current time if stored timestamp is malformed
			t = time.Now().UTC()
		}
		applied[version] = t
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate migration rows: %w", err)
	}

	// Build status list
	var result []MigrationInfo
	for _, m := range migrations {
		info := MigrationInfo{
			Version:     m.Version,
			Description: m.Description,
			Applied:     false,
		}
		if t, ok := applied[m.Version]; ok {
			info.Applied = true
			info.AppliedAt = t
		}
		result = append(result, info)
	}

	return result, nil
}

// MigrationInfo represents the status of a migration.
type MigrationInfo struct {
	Version     int
	Description string
	Applied     bool
	AppliedAt   time.Time
}

// SchemaVersion returns the current schema version.
func (db *DB) SchemaVersion(ctx context.Context) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, fmt.Errorf("database is closed")
	}

	return db.getCurrentVersion(ctx)
}
