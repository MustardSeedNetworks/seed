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
		{
			Description: "Create health check results table for historical tracking",
			Up: `
			CREATE TABLE IF NOT EXISTS health_check_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_type TEXT NOT NULL,
				endpoint_name TEXT NOT NULL,
				endpoint_target TEXT NOT NULL,
				success INTEGER NOT NULL,
				latency_ms REAL,
				status_code INTEGER,
				error_message TEXT,
				metadata_json TEXT,
				recorded_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_health_check_type_time ON health_check_results(check_type, recorded_at);
			CREATE INDEX IF NOT EXISTS idx_health_check_endpoint_time ON health_check_results(endpoint_name, recorded_at);
			CREATE INDEX IF NOT EXISTS idx_health_check_recorded ON health_check_results(recorded_at);
		`,
		},
		{
			Description: "Create health check hourly rollups table",
			Up: `
			CREATE TABLE IF NOT EXISTS health_check_rollups_hourly (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_type TEXT NOT NULL,
				endpoint_name TEXT NOT NULL,
				hour_bucket TEXT NOT NULL,
				total_checks INTEGER NOT NULL,
				successful_checks INTEGER NOT NULL,
				avg_latency_ms REAL,
				min_latency_ms REAL,
				max_latency_ms REAL,
				p95_latency_ms REAL
			);

			CREATE UNIQUE INDEX IF NOT EXISTS idx_health_hourly_unique
				ON health_check_rollups_hourly(check_type, endpoint_name, hour_bucket);
			CREATE INDEX IF NOT EXISTS idx_health_hourly_bucket ON health_check_rollups_hourly(hour_bucket);
		`,
		},
		{
			Description: "Create health check daily rollups table",
			Up: `
			CREATE TABLE IF NOT EXISTS health_check_rollups_daily (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_type TEXT NOT NULL,
				endpoint_name TEXT NOT NULL,
				day_bucket TEXT NOT NULL,
				total_checks INTEGER NOT NULL,
				successful_checks INTEGER NOT NULL,
				avg_latency_ms REAL,
				min_latency_ms REAL,
				max_latency_ms REAL,
				p95_latency_ms REAL,
				availability_percent REAL
			);

			CREATE UNIQUE INDEX IF NOT EXISTS idx_health_daily_unique
				ON health_check_rollups_daily(check_type, endpoint_name, day_bucket);
			CREATE INDEX IF NOT EXISTS idx_health_daily_bucket ON health_check_rollups_daily(day_bucket);
		`,
		},
		// ============================================================
		// UNIFIED DISCOVERY ENGINE MIGRATIONS (19-26)
		// ============================================================
		{
			Description: "Create unified discovered_devices table",
			Up: `
			-- Core device table with unified identity across wired/wifi/bluetooth
			CREATE TABLE IF NOT EXISTS discovered_devices (
				id TEXT PRIMARY KEY,
				primary_mac TEXT NOT NULL UNIQUE,
				hostname TEXT,
				vendor TEXT,
				device_type TEXT DEFAULT 'unknown',
				device_model TEXT,
				authorization_status TEXT DEFAULT 'unknown',
				criticality INTEGER DEFAULT 5,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				is_online INTEGER DEFAULT 1,
				notes TEXT,
				tags TEXT,
				metadata_json TEXT,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX IF NOT EXISTS idx_disc_devices_mac ON discovered_devices(primary_mac);
			CREATE INDEX IF NOT EXISTS idx_disc_devices_type ON discovered_devices(device_type);
			CREATE INDEX IF NOT EXISTS idx_disc_devices_vendor ON discovered_devices(vendor);
			CREATE INDEX IF NOT EXISTS idx_disc_devices_last_seen ON discovered_devices(last_seen);
			CREATE INDEX IF NOT EXISTS idx_disc_devices_online ON discovered_devices(is_online);
			CREATE INDEX IF NOT EXISTS idx_disc_devices_auth ON discovered_devices(authorization_status);
		`,
		},
		{
			Description: "Create discovery_interfaces table for multiple interfaces per device",
			Up: `
			-- Device interfaces (wired, wifi, bluetooth) - supports multiple per device
			CREATE TABLE IF NOT EXISTS discovery_interfaces (
				id TEXT PRIMARY KEY,
				device_id TEXT NOT NULL,
				interface_type TEXT NOT NULL,
				mac_address TEXT NOT NULL,
				ip_addresses TEXT,
				interface_name TEXT,
				is_primary INTEGER DEFAULT 0,

				-- Wired-specific
				switch_port TEXT,
				switch_name TEXT,
				vlan_id INTEGER,
				duplex TEXT,
				speed_mbps INTEGER,
				poe_status TEXT,

				-- WiFi-specific
				ssid TEXT,
				bssid TEXT,
				signal_dbm INTEGER,
				noise_dbm INTEGER,
				channel INTEGER,
				channel_width INTEGER,
				frequency_mhz INTEGER,
				wifi_standards TEXT,
				security_type TEXT,

				-- Bluetooth-specific
				bt_class TEXT,
				bt_version TEXT,
				bt_signal INTEGER,

				last_seen TEXT NOT NULL,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE,
				UNIQUE(device_id, mac_address)
			);

			CREATE INDEX IF NOT EXISTS idx_disc_iface_device ON discovery_interfaces(device_id);
			CREATE INDEX IF NOT EXISTS idx_disc_iface_mac ON discovery_interfaces(mac_address);
			CREATE INDEX IF NOT EXISTS idx_disc_iface_type ON discovery_interfaces(interface_type);
			CREATE INDEX IF NOT EXISTS idx_disc_iface_ssid ON discovery_interfaces(ssid);
			CREATE INDEX IF NOT EXISTS idx_disc_iface_bssid ON discovery_interfaces(bssid);
		`,
		},
		{
			Description: "Create wifi_networks table for SSID tracking",
			Up: `
			-- WiFi networks (SSIDs) discovered
			CREATE TABLE IF NOT EXISTS wifi_networks (
				id TEXT PRIMARY KEY,
				ssid TEXT NOT NULL,
				is_hidden INTEGER DEFAULT 0,
				security_type TEXT,
				authorization_status TEXT DEFAULT 'unknown',
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT,
				UNIQUE(ssid, security_type)
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_networks_ssid ON wifi_networks(ssid);
			CREATE INDEX IF NOT EXISTS idx_wifi_networks_auth ON wifi_networks(authorization_status);
		`,
		},
		{
			Description: "Create wifi_access_points table for BSSID tracking",
			Up: `
			-- WiFi access points (BSSIDs)
			CREATE TABLE IF NOT EXISTS wifi_access_points (
				id TEXT PRIMARY KEY,
				device_id TEXT,
				bssid TEXT NOT NULL UNIQUE,
				ssid_id TEXT,
				ap_name TEXT,
				vendor TEXT,

				-- Radio info
				channel INTEGER,
				channel_width INTEGER,
				frequency_mhz INTEGER,
				band TEXT,
				wifi_standards TEXT,

				-- Signal
				signal_dbm INTEGER,
				noise_dbm INTEGER,

				-- Status
				client_count INTEGER DEFAULT 0,
				max_clients INTEGER,
				is_authorized INTEGER DEFAULT 1,

				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL,
				FOREIGN KEY (ssid_id) REFERENCES wifi_networks(id) ON DELETE SET NULL
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_aps_bssid ON wifi_access_points(bssid);
			CREATE INDEX IF NOT EXISTS idx_wifi_aps_ssid ON wifi_access_points(ssid_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_aps_device ON wifi_access_points(device_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_aps_channel ON wifi_access_points(channel);
			CREATE INDEX IF NOT EXISTS idx_wifi_aps_band ON wifi_access_points(band);
		`,
		},
		{
			Description: "Create channel_utilization table for WiFi spectrum analysis",
			Up: `
			-- Channel utilization metrics for spectrum analysis
			CREATE TABLE IF NOT EXISTS channel_utilization (
				id TEXT PRIMARY KEY,
				channel INTEGER NOT NULL,
				band TEXT NOT NULL,
				frequency_mhz INTEGER NOT NULL,

				-- Utilization metrics
				utilization_percent REAL,
				non_wifi_percent REAL,
				retry_percent REAL,
				ap_count INTEGER,
				client_count INTEGER,

				recorded_at TEXT NOT NULL,

				UNIQUE(channel, band, recorded_at)
			);

			CREATE INDEX IF NOT EXISTS idx_channel_util_time ON channel_utilization(recorded_at);
			CREATE INDEX IF NOT EXISTS idx_channel_util_channel ON channel_utilization(channel, band);
		`,
		},
		{
			Description: "Create discovery_history table for device event timeline",
			Up: `
			-- Discovery event history for device timeline
			CREATE TABLE IF NOT EXISTS discovery_history (
				id TEXT PRIMARY KEY,
				device_id TEXT NOT NULL,
				event_type TEXT NOT NULL,
				event_data TEXT,
				recorded_at TEXT NOT NULL,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_disc_history_device ON discovery_history(device_id);
			CREATE INDEX IF NOT EXISTS idx_disc_history_time ON discovery_history(recorded_at);
			CREATE INDEX IF NOT EXISTS idx_disc_history_type ON discovery_history(event_type);
		`,
		},
		{
			Description: "Create oui_vendors table for MAC vendor lookup",
			Up: `
			-- OUI vendor database for MAC address lookup
			CREATE TABLE IF NOT EXISTS oui_vendors (
				oui TEXT PRIMARY KEY,
				vendor_name TEXT NOT NULL,
				vendor_short TEXT,
				is_private INTEGER DEFAULT 0,
				device_category TEXT,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX IF NOT EXISTS idx_oui_vendor_name ON oui_vendors(vendor_name);
			CREATE INDEX IF NOT EXISTS idx_oui_category ON oui_vendors(device_category);
		`,
		},
		{
			Description: "Create network_problems table for problem detection",
			Up: `
			-- Network problems detected by discovery engine
			CREATE TABLE IF NOT EXISTS network_problems (
				id TEXT PRIMARY KEY,
				problem_type TEXT NOT NULL,
				severity TEXT NOT NULL,
				device_id TEXT,
				interface_id TEXT,
				description TEXT NOT NULL,
				details_json TEXT,
				is_resolved INTEGER DEFAULT 0,
				detected_at TEXT NOT NULL,
				resolved_at TEXT,
				acknowledged_at TEXT,
				acknowledged_by TEXT,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE,
				FOREIGN KEY (interface_id) REFERENCES discovery_interfaces(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_net_problems_type ON network_problems(problem_type);
			CREATE INDEX IF NOT EXISTS idx_net_problems_device ON network_problems(device_id);
			CREATE INDEX IF NOT EXISTS idx_net_problems_severity ON network_problems(severity);
			CREATE INDEX IF NOT EXISTS idx_net_problems_resolved ON network_problems(is_resolved);
			CREATE INDEX IF NOT EXISTS idx_net_problems_detected ON network_problems(detected_at);
		`,
		},
		{
			Description: "Create bluetooth_devices table for Bluetooth discovery",
			Up: `
			-- Bluetooth devices discovered via BLE/Classic scanning
			CREATE TABLE IF NOT EXISTS bluetooth_devices (
				id TEXT PRIMARY KEY,
				device_id TEXT,
				address TEXT NOT NULL UNIQUE,
				name TEXT,
				alias TEXT,
				vendor TEXT,
				bluetooth_type TEXT NOT NULL,
				device_class TEXT,
				appearance INTEGER DEFAULT 0,
				class_of_device INTEGER DEFAULT 0,
				rssi INTEGER,
				tx_power INTEGER,
				is_connected INTEGER DEFAULT 0,
				is_connectable INTEGER DEFAULT 0,
				is_authorized INTEGER DEFAULT 0,
				is_trusted INTEGER DEFAULT 0,
				is_paired INTEGER DEFAULT 0,
				is_blocked INTEGER DEFAULT 0,
				service_uuids_json TEXT,
				manufacturer_id INTEGER,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			);

			CREATE INDEX IF NOT EXISTS idx_bt_devices_address ON bluetooth_devices(address);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_name ON bluetooth_devices(name);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_type ON bluetooth_devices(bluetooth_type);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_class ON bluetooth_devices(device_class);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_vendor ON bluetooth_devices(vendor);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_connected ON bluetooth_devices(is_connected);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_authorized ON bluetooth_devices(is_authorized);
			CREATE INDEX IF NOT EXISTS idx_bt_devices_last_seen ON bluetooth_devices(last_seen);
		`,
		},
		{
			Description: "Create bluetooth_scan_history table for scan records",
			Up: `
			-- Historical Bluetooth scan results
			CREATE TABLE IF NOT EXISTS bluetooth_scan_history (
				id TEXT PRIMARY KEY,
				adapter_name TEXT,
				scan_type TEXT NOT NULL,
				devices_found INTEGER NOT NULL,
				classic_count INTEGER DEFAULT 0,
				ble_count INTEGER DEFAULT 0,
				scan_duration_ms INTEGER,
				scan_time TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_bt_scan_time ON bluetooth_scan_history(scan_time);
			CREATE INDEX IF NOT EXISTS idx_bt_scan_type ON bluetooth_scan_history(scan_type);
		`,
		},
		{
			Description: "Create MIB database tables for SNMP OID resolution",
			Up: `
			-- OID name-to-numeric mappings for SNMP operations
			-- Stores 918+ standard OID definitions from RFC MIBs
			CREATE TABLE IF NOT EXISTS mib_oid_names (
				name TEXT PRIMARY KEY,           -- Human-readable name (e.g., "sysDescr")
				oid TEXT NOT NULL,               -- Numeric OID (e.g., "1.3.6.1.2.1.1.1")
				full_path TEXT,                  -- Full descriptive path (optional)
				mib_name TEXT,                   -- Source MIB name (e.g., "SNMPv2-MIB")
				created_at TEXT DEFAULT (datetime('now'))
			);

			-- Index for OID prefix searches and lookups
			CREATE INDEX IF NOT EXISTS idx_mib_oid_names_oid ON mib_oid_names(oid);
			CREATE INDEX IF NOT EXISTS idx_mib_oid_names_mib ON mib_oid_names(mib_name);

			-- MIB source tracking for documentation
			CREATE TABLE IF NOT EXISTS mib_sources (
				mib_name TEXT PRIMARY KEY,
				description TEXT,
				vendor TEXT,
				rfc_reference TEXT,
				loaded_at TEXT DEFAULT (datetime('now'))
			);
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
