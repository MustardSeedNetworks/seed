package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Migration represents a database schema migration.
type Migration struct {
	Version     int
	Description string
	Up          string
}

// migrationDef is the definition without version (computed from index).
type migrationDef struct {
	Description string
	Up          string
}

// getMigrationDefs returns migration definitions without versions.
// IMPORTANT: Never modify existing migrations, only add new ones.
// The version is computed as index + 1.
//
//nolint:funlen // Migration definitions are intentionally in one place
func getMigrationDefs() []migrationDef {
	return []migrationDef{
		{
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
		{
			Description: "Create pipeline tables for discovery pipeline",
			Up: `
			-- Pipeline run history
			CREATE TABLE IF NOT EXISTS pipeline_runs (
				id TEXT PRIMARY KEY,
				started_at TEXT NOT NULL,
				completed_at TEXT,
				status TEXT NOT NULL,
				triggered_by TEXT,
				phases_enabled TEXT NOT NULL,
				config_json TEXT,
				summary_json TEXT,
				error_message TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);
			CREATE INDEX IF NOT EXISTS idx_pipeline_runs_started ON pipeline_runs(started_at);

			-- Device interfaces from SNMP
			CREATE TABLE IF NOT EXISTS device_interfaces (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				device_id TEXT NOT NULL,
				if_index INTEGER NOT NULL,
				name TEXT,
				description TEXT,
				alias TEXT,
				type INTEGER,
				mtu INTEGER,
				speed_mbps INTEGER,
				mac_address TEXT,
				admin_status TEXT,
				oper_status TEXT,
				collected_at TEXT NOT NULL,
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_device_interfaces_device ON device_interfaces(device_id);
			CREATE INDEX IF NOT EXISTS idx_device_interfaces_mac ON device_interfaces(mac_address);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_device_interfaces_unique ON device_interfaces(device_id, if_index);

			-- Device open ports from port scanning
			CREATE TABLE IF NOT EXISTS device_ports (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				device_id TEXT NOT NULL,
				port INTEGER NOT NULL,
				protocol TEXT NOT NULL DEFAULT 'tcp',
				state TEXT NOT NULL DEFAULT 'open',
				service_name TEXT,
				banner TEXT,
				version TEXT,
				scanned_at TEXT NOT NULL,
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_device_ports_device ON device_ports(device_id);
			CREATE INDEX IF NOT EXISTS idx_device_ports_port ON device_ports(port);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_device_ports_unique ON device_ports(device_id, port, protocol);

			-- Device vulnerabilities from assessment phase
			CREATE TABLE IF NOT EXISTS device_vulnerabilities (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				device_id TEXT NOT NULL,
				cve_id TEXT NOT NULL,
				severity TEXT,
				cvss_score REAL,
				cvss_vector TEXT,
				affected_component TEXT,
				affected_version TEXT,
				fix_available INTEGER DEFAULT 0,
				status TEXT DEFAULT 'new',
				detected_at TEXT NOT NULL,
				resolved_at TEXT,
				notes TEXT,
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_device_vulns_device ON device_vulnerabilities(device_id);
			CREATE INDEX IF NOT EXISTS idx_device_vulns_cve ON device_vulnerabilities(cve_id);
			CREATE INDEX IF NOT EXISTS idx_device_vulns_severity ON device_vulnerabilities(severity);
			CREATE INDEX IF NOT EXISTS idx_device_vulns_status ON device_vulnerabilities(status);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_device_vulns_unique ON device_vulnerabilities(device_id, cve_id);
		`,
		},
		{
			Description: "Create users table for authentication",
			Up: `
			-- Users table for authentication
			-- Moves password hashes from config.yaml to database for better security
			CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'admin',
				is_active INTEGER DEFAULT 1,
				last_login TEXT,
				failed_attempts INTEGER DEFAULT 0,
				locked_until TEXT,
				token_version INTEGER DEFAULT 1,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
			CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);
		`,
		},
		{
			Description: "Create logs table for persistent log storage",
			Up: `
			CREATE TABLE IF NOT EXISTS logs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				level TEXT NOT NULL,
				layer TEXT NOT NULL,
				message TEXT NOT NULL,
				component TEXT,
				request_id TEXT,
				session_id TEXT,
				duration_ms INTEGER,
				metadata_json TEXT,
				stack TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
			CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
			CREATE INDEX IF NOT EXISTS idx_logs_layer ON logs(layer);
			CREATE INDEX IF NOT EXISTS idx_logs_component ON logs(component);
			CREATE INDEX IF NOT EXISTS idx_logs_request_id ON logs(request_id);
		`,
		},
		{
			Description: "Create reports and scheduled_reports tables for Harvest module",
			Up: `
			CREATE TABLE IF NOT EXISTS reports (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				type TEXT NOT NULL,
				format TEXT NOT NULL,
				template TEXT,
				status TEXT NOT NULL DEFAULT 'pending',
				file_path TEXT,
				file_size INTEGER DEFAULT 0,
				parameters_json TEXT,
				error TEXT,
				created_at TEXT NOT NULL,
				completed_at TEXT,
				expires_at TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
			CREATE INDEX IF NOT EXISTS idx_reports_type ON reports(type);
			CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);

			CREATE TABLE IF NOT EXISTS scheduled_reports (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				template TEXT NOT NULL,
				format TEXT NOT NULL,
				schedule_json TEXT NOT NULL,
				parameters_json TEXT,
				recipients_json TEXT,
				enabled INTEGER DEFAULT 1,
				last_run TEXT,
				next_run TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_scheduled_reports_enabled ON scheduled_reports(enabled);
			CREATE INDEX IF NOT EXISTS idx_scheduled_reports_next_run ON scheduled_reports(next_run);
		`,
		},
	}
}

// getMigrations returns migrations with computed version numbers.
// Version = index + 1 (starting from 1).
func getMigrations() []Migration {
	defs := getMigrationDefs()
	migrations := make([]Migration, len(defs))
	for i, d := range defs {
		migrations[i] = Migration{
			Version:     i + 1,
			Description: d.Description,
			Up:          d.Up,
		}
	}
	return migrations
}

// migrate runs all pending migrations.
func (db *DB) migrate() error {
	ctx := context.Background()

	// Ensure schema_migrations table exists (migration 1)
	_, err := db.conn.ExecContext(ctx, getMigrations()[0].Up)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := db.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Run pending migrations
	for _, m := range getMigrations() {
		if m.Version <= currentVersion {
			continue
		}

		if runErr := db.runMigration(ctx, m); runErr != nil {
			return fmt.Errorf(
				"failed to run migration %d (%s): %w",
				m.Version,
				m.Description,
				runErr,
			)
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
		return nil, errors.New("database is closed")
	}

	// Get applied migrations
	rows, err := db.conn.QueryContext(ctx, `
		SELECT version, applied_at, description FROM schema_migrations ORDER BY version
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[int]time.Time)
	for rows.Next() {
		var version int
		var appliedAt string
		var desc sql.NullString
		if scanErr := rows.Scan(&version, &appliedAt, &desc); scanErr != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", scanErr)
		}
		t, parseErr := time.Parse(time.RFC3339, appliedAt)
		if parseErr != nil {
			// Fallback to current time if stored timestamp is malformed
			t = time.Now().UTC()
		}
		applied[version] = t
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed to iterate migration rows: %w", rowsErr)
	}

	// Build status list
	migrations := getMigrations()
	result := make([]MigrationInfo, 0, len(migrations))
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
		return 0, errors.New("database is closed")
	}

	return db.getCurrentVersion(ctx)
}

// seedDefaultProfile creates a default profile if no profiles exist.
// This ensures the app is immediately functional on fresh installs.
// The default profile uses sensible defaults from DefaultConfig().
func (db *DB) seedDefaultProfile() error {
	ctx := context.Background()

	// Check if any profiles exist
	var count int
	err := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count profiles: %w", err)
	}

	// Only seed if no profiles exist
	if count > 0 {
		return nil
	}

	// Create the default profile with settings from DefaultConfig()
	now := time.Now().UTC().Format(time.RFC3339)
	defaultConfigJSON := `{
		"version": 1,
		"thresholds": {
			"dns": {"warning": 50, "critical": 100},
			"gateway": {"warning": 20, "critical": 50},
			"wifi": {"warning": -50, "critical": -70},
			"custom_ping": {"warning": 50, "critical": 100},
			"custom_tcp": {"warning": 100, "critical": 200},
			"custom_http": {"warning": 500, "critical": 1000},
			"http_timings": {
				"dns": {"warning": 50, "critical": 100},
				"tcp": {"warning": 50, "critical": 100},
				"tls": {"warning": 100, "critical": 200},
				"ttfb": {"warning": 200, "critical": 500}
			}
		},
		"health_checks": {
			"ping_targets": [
				{"name": "Google DNS", "host": "8.8.8.8", "enabled": true},
				{"name": "Cloudflare", "host": "1.1.1.1", "enabled": true}
			],
			"http_endpoints": [
				{"name": "Google", "url": "https://www.google.com", "expected_status": 200, "enabled": true}
			],
			"rtsp_endpoints": [
				{"name": "Wowza Demo", "url": "rtsp://wowzaec2demo.streamlock.net/vod/mp4:BigBuckBunny_115k.mp4", "enabled": true}
			],
			"dicom_endpoints": [
				{"name": "Public DICOM", "host": "dicomserver.co.uk", "port": 104, "called_ae": "ANY-SCP", "calling_ae": "SEED-SCU", "enabled": true}
			],
			"run_performance": false,
			"run_speedtest": false,
			"run_iperf": false,
			"run_discovery": false
		},
		"speedtest": {"server_id": "", "auto_run_on_link": true},
		"iperf": {"auto_run_on_link": false, "server": "", "port": 5201, "protocol": "tcp", "direction": "download", "duration": 10, "server_port": 5201, "enable_server": true},
		"fab_options": {
			"run_link": true,
			"run_switch": true,
			"run_vlan": true,
			"run_ip_config": true,
			"run_gateway": true,
			"run_dns": true,
			"run_health_checks": true,
			"run_network_discovery": true,
			"run_speedtest": false,
			"run_iperf": false,
			"run_performance": true,
			"auto_scan_on_link": true
		},
		"display_options": {"show_public_ip": true, "unit_system": "sae"},
		"dns": {"test_hostname": "google.com", "timeout_ms": 5000},
		"snmp": {"communities": ["public"], "timeout_ms": 5000, "retries": 2, "port": 161},
		"network_discovery": {"enabled": true, "auto_scan": true, "scan_interval_secs": 600, "ipv6_enabled": true, "fingerprinting": {"enabled": false, "os_detection": false, "service_probes": false}},
		"link": {"mode": "auto"},
		"cable_test": {"enabled": true}
	}`

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO profiles (id, name, description, config_json, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`, "default", "Default", "Default profile created on first run", defaultConfigJSON, now, now)
	if err != nil {
		return fmt.Errorf("failed to create default profile: %w", err)
	}

	return nil
}
