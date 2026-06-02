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
			Description: "Create reports and scheduled_reports tables for reporting",
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
		{
			// Wave 3 (#85): TOTP MFA + WebAuthn passkeys.
			//
			// totp_secret holds the shared secret as base32 text. It is
			// stored at rest in the SQLite database with file-system
			// permissions; we deliberately do not encrypt it with the
			// JWT-secret-derived key because the JWT secret in seed is
			// regenerated on every restart unless explicitly configured
			// (see auth.GenerateJWTSecret docstring), which would lock
			// users out of their MFA on each restart. Operators who need
			// at-rest encryption should rely on full-disk encryption or
			// configure jwt_secret persistently and revisit this in a
			// future hardening pass.
			//
			// totp_enabled gates whether login should require a TOTP
			// code as a second factor. It is set to 1 only after the
			// user has confirmed possession of the secret by submitting
			// a valid code via /api/v1/auth/totp/verify.
			Description: "Add TOTP MFA columns and webauthn_credentials table (#85)",
			Up: `
			ALTER TABLE users ADD COLUMN totp_secret TEXT;
			ALTER TABLE users ADD COLUMN totp_enabled INTEGER DEFAULT 0;

			CREATE TABLE IF NOT EXISTS webauthn_credentials (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				credential_id BLOB NOT NULL UNIQUE,
				public_key BLOB NOT NULL,
				sign_count INTEGER NOT NULL DEFAULT 0,
				attestation_type TEXT,
				transports TEXT,
				aaguid BLOB,
				created_at TEXT NOT NULL,
				last_used_at TEXT,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_webauthn_user ON webauthn_credentials(user_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_webauthn_credential_id
				ON webauthn_credentials(credential_id);
		`,
		},
		{
			// Phase D-2 (LICENSE_STRATEGY.md): API personal-access tokens
			// for programmatic access. Pro-tier capability; minting is
			// gated server-side at the handler. Token plaintext is shown
			// once at creation; only its SHA-256 hash is stored.
			Description: "Add api_tokens table for personal-access tokens",
			Up: `
			CREATE TABLE IF NOT EXISTS api_tokens (
				id              TEXT PRIMARY KEY,
				owner_username  TEXT NOT NULL,
				name            TEXT NOT NULL,
				token_hash      TEXT NOT NULL UNIQUE,
				prefix          TEXT NOT NULL,
				created_at      TEXT NOT NULL,
				last_used_at    TEXT,
				revoked_at      TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_api_tokens_owner    ON api_tokens(owner_username);
			CREATE INDEX IF NOT EXISTS idx_api_tokens_hash     ON api_tokens(token_hash);
			CREATE INDEX IF NOT EXISTS idx_api_tokens_active   ON api_tokens(revoked_at);
		`,
		},
		{
			// Multi-feature hardening 2026-05-26 (msn-docs PR #14 §2):
			// adds CHECK constraints + SSO columns to users, and an
			// ON DELETE CASCADE FK from api_tokens.owner_username so
			// revoking a user automatically invalidates their tokens.
			// Pre-alpha: empty production DBs, so the table-swap pattern
			// SQLite forces on us (no ALTER ADD CHECK / ADD FK) is safe.
			Description: "Harden users + api_tokens schemas; add SSO columns",
			Up: `
			-- Disable FK enforcement during the swap so api_tokens can
			-- briefly reference a renamed users table without erroring.
			PRAGMA foreign_keys = OFF;

			-- 1. Recreate users with CHECK constraints + SSO columns.
			--    auth_provider = which channel created the row ('local'
			--    for password users; provider name for SSO). external_id
			--    is the IdP's stable subject claim ('sub' for OIDC, 'id'
			--    for Microsoft Graph). The (auth_provider, external_id)
			--    pair is the SSO lookup key so the same email arriving
			--    via two providers is two separate user rows on purpose.
			CREATE TABLE users_new (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				username        TEXT    NOT NULL UNIQUE CHECK (LENGTH(username) >= 3 AND LENGTH(username) <= 64),
				password_hash   TEXT    NOT NULL,
				role            TEXT    NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
				is_active       INTEGER NOT NULL DEFAULT 1,
				last_login      TEXT,
				failed_attempts INTEGER NOT NULL DEFAULT 0,
				locked_until    TEXT,
				token_version   INTEGER NOT NULL DEFAULT 1,
				totp_secret     TEXT,
				totp_enabled    INTEGER NOT NULL DEFAULT 0,
				auth_provider   TEXT    NOT NULL DEFAULT 'local' CHECK (auth_provider IN ('local','google','microsoft','github')),
				external_id     TEXT,
				email           TEXT,
				display_name    TEXT,
				created_at      TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL,
				UNIQUE (auth_provider, external_id)
			);

			-- Copy existing rows. Pre-existing data is all local-auth
			-- (no SSO callback could have been invoked yet), so we leave
			-- auth_provider at its default of 'local' and external_id/
			-- email/display_name NULL. Role mapping: existing 'admin' is
			-- preserved; anything unrecognized gets demoted to 'viewer'
			-- (defensive — should not occur on a fresh dev DB). TOTP
			-- columns from the Wave-3 MFA migration are preserved
			-- verbatim so existing enrolments survive the table swap.
			INSERT INTO users_new
				(id, username, password_hash, role, is_active, last_login,
				 failed_attempts, locked_until, token_version,
				 totp_secret, totp_enabled,
				 created_at, updated_at)
			SELECT
				id, username, password_hash,
				CASE WHEN role IN ('admin','operator','viewer') THEN role ELSE 'viewer' END,
				COALESCE(is_active, 1), last_login,
				COALESCE(failed_attempts, 0), locked_until,
				COALESCE(token_version, 1),
				totp_secret, COALESCE(totp_enabled, 0),
				created_at, updated_at
			FROM users;

			DROP TABLE users;
			ALTER TABLE users_new RENAME TO users;

			CREATE INDEX idx_users_username             ON users(username);
			CREATE INDEX idx_users_active               ON users(is_active);
			CREATE INDEX idx_users_provider_external_id ON users(auth_provider, external_id);
			CREATE INDEX idx_users_email                ON users(email);

			-- 2. Recreate api_tokens with FK to users.username (CASCADE).
			--    Deleting a user automatically revokes every token they
			--    own; this matches the IncrementTokenVersion behaviour
			--    that's already wired for JWT session revocation.
			CREATE TABLE api_tokens_new (
				id              TEXT PRIMARY KEY,
				owner_username  TEXT NOT NULL,
				name            TEXT NOT NULL,
				token_hash      TEXT NOT NULL UNIQUE,
				prefix          TEXT NOT NULL,
				created_at      TEXT NOT NULL,
				last_used_at    TEXT,
				revoked_at      TEXT,
				FOREIGN KEY (owner_username) REFERENCES users(username) ON DELETE CASCADE ON UPDATE CASCADE
			);

			INSERT INTO api_tokens_new
				(id, owner_username, name, token_hash, prefix, created_at, last_used_at, revoked_at)
			SELECT
				id, owner_username, name, token_hash, prefix, created_at, last_used_at, revoked_at
			FROM api_tokens
			-- Defensive: drop any orphan rows whose owner no longer
			-- exists in users. Pre-alpha, should be empty.
			WHERE owner_username IN (SELECT username FROM users);

			DROP TABLE api_tokens;
			ALTER TABLE api_tokens_new RENAME TO api_tokens;

			CREATE INDEX idx_api_tokens_owner  ON api_tokens(owner_username);
			CREATE INDEX idx_api_tokens_hash   ON api_tokens(token_hash);
			CREATE INDEX idx_api_tokens_active ON api_tokens(revoked_at);

			PRAGMA foreign_keys = ON;
		`,
		},
		{
			// Wave 5 (#1255): per-token scope for personal-access tokens.
			// NULL means the token inherits the owner's role (existing
			// behavior); a non-NULL value caps the effective role at
			// min(owner.role, token.scope). Validated by the CHECK
			// constraint so writes through the application layer or via
			// the SQLite shell both stay bounded to the legal set.
			Description: "Add scope column to api_tokens for per-token role capping (#1255)",
			Up: `
			ALTER TABLE api_tokens ADD COLUMN scope TEXT
			    CHECK (scope IS NULL OR scope IN ('admin','operator','viewer'));
		`,
		},
		{
			// Stage A1.1 (2026-05-30) — multi-tenancy foundation. Creates
			// the clients table and seeds a single 'default' client. All
			// observation tables get client_id added in subsequent
			// migrations; existing rows backfill to 'default'.
			// MSP-first per SEED_ARCHITECTURE.md section 3.0.
			Description: "Create clients table and seed default client (multi-tenancy foundation)",
			Up: `
			CREATE TABLE IF NOT EXISTS clients (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				slug TEXT NOT NULL UNIQUE,
				branding_json TEXT,
				default_retention_overrides_json TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_clients_slug ON clients(slug);

			INSERT OR IGNORE INTO clients (id, name, slug, created_at, updated_at)
			VALUES ('default', 'Default', 'default', datetime('now'), datetime('now'));
		`,
		},
		{
			// Stage A1.1 — add client_id to legacy observation + result
			// tables. Backfills existing rows to the default client via
			// the column DEFAULT. AirMapper / Wi-Fi survey
			// (survey_samples) included; wifi/ code continues working
			// unchanged because writes default to 'default' client. A
			// follow-up A1 step will update wifi/ to set client_id
			// explicitly.
			Description: "Add client_id to legacy observation and result tables",
			Up: `
			ALTER TABLE profiles ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE alerts ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE metrics ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE speedtest_results ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE dns_results ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE gateway_results ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE survey_samples ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);

			CREATE INDEX IF NOT EXISTS idx_profiles_client ON profiles(client_id);
			CREATE INDEX IF NOT EXISTS idx_alerts_client ON alerts(client_id);
			CREATE INDEX IF NOT EXISTS idx_metrics_client ON metrics(client_id);
			CREATE INDEX IF NOT EXISTS idx_speedtest_results_client ON speedtest_results(client_id);
			CREATE INDEX IF NOT EXISTS idx_dns_results_client ON dns_results(client_id);
			CREATE INDEX IF NOT EXISTS idx_gateway_results_client ON gateway_results(client_id);
			CREATE INDEX IF NOT EXISTS idx_survey_samples_client ON survey_samples(client_id);
		`,
		},
		{
			// Stage A1.1 — add client_id to discovery + inventory
			// tables. wifi_networks / wifi_access_points /
			// channel_utilization are AirMapper / Wi-Fi visibility
			// surfaces; same DEFAULT 'default' semantics — existing
			// code unchanged, wifi/ updated in a follow-up step.
			Description: "Add client_id to discovery and inventory tables",
			Up: `
			ALTER TABLE discovered_devices ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE discovery_interfaces ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE wifi_networks ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE wifi_access_points ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE channel_utilization ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE discovery_history ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE bluetooth_devices ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE bluetooth_scan_history ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE network_problems ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);

			CREATE INDEX IF NOT EXISTS idx_discovered_devices_client ON discovered_devices(client_id);
			CREATE INDEX IF NOT EXISTS idx_discovery_interfaces_client ON discovery_interfaces(client_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_networks_client ON wifi_networks(client_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_access_points_client ON wifi_access_points(client_id);
			CREATE INDEX IF NOT EXISTS idx_channel_utilization_client ON channel_utilization(client_id);
			CREATE INDEX IF NOT EXISTS idx_discovery_history_client ON discovery_history(client_id);
			CREATE INDEX IF NOT EXISTS idx_bluetooth_devices_client ON bluetooth_devices(client_id);
			CREATE INDEX IF NOT EXISTS idx_bluetooth_scan_history_client ON bluetooth_scan_history(client_id);
			CREATE INDEX IF NOT EXISTS idx_network_problems_client ON network_problems(client_id);
		`,
		},
		{
			// Stage A1.2 (2026-05-30) — unified probe config table. One
			// row per configured probe; kind selects the registered
			// Checker implementation; params_json carries kind-specific
			// configuration. See SEED_ARCHITECTURE.md section 3.1.
			//
			// V1.0 kinds: dns, tls, ping, tcp, udp, http, https, rtsp,
			// dicom, hl7, fhir, lti, ldap, opcua, modbus, ntp, sip,
			// dot1x, cable, transaction. Adding a new kind is a new
			// Checker implementation in internal/probe/checkers/ — no
			// schema change required.
			Description: "Create probes config table (unified probe engine)",
			Up: `
			CREATE TABLE IF NOT EXISTS probes (
				id TEXT PRIMARY KEY,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				display_name TEXT NOT NULL,
				target TEXT NOT NULL,
				params_json TEXT,
				interval_seconds INTEGER NOT NULL DEFAULT 60,
				enabled INTEGER NOT NULL DEFAULT 1,
				warning_json TEXT,
				critical_json TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_probes_client ON probes(client_id);
			CREATE INDEX IF NOT EXISTS idx_probes_kind ON probes(kind);
			CREATE INDEX IF NOT EXISTS idx_probes_enabled ON probes(enabled);
			CREATE INDEX IF NOT EXISTS idx_probes_client_kind ON probes(client_id, kind);
		`,
		},
		{
			// Stage A1.2 — unified probe results table. Receives every
			// probe.Result emitted by the engine. Coexists with
			// health_check_results during the A1.3 checker port; once
			// all checkers are ported and HealthCheckRepository is
			// retired, A1.5 drops health_check_results.
			Description: "Create probe_results table (unified probe results)",
			Up: `
			CREATE TABLE IF NOT EXISTS probe_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				probe_id TEXT NOT NULL,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				timestamp TEXT NOT NULL,
				success INTEGER NOT NULL,
				latency_ms REAL,
				error TEXT,
				metadata_json TEXT,
				FOREIGN KEY (probe_id) REFERENCES probes(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_probe_results_probe ON probe_results(probe_id);
			CREATE INDEX IF NOT EXISTS idx_probe_results_client ON probe_results(client_id);
			CREATE INDEX IF NOT EXISTS idx_probe_results_kind ON probe_results(kind);
			CREATE INDEX IF NOT EXISTS idx_probe_results_timestamp ON probe_results(timestamp);
			CREATE INDEX IF NOT EXISTS idx_probe_results_client_kind_ts ON probe_results(client_id, kind, timestamp);
		`,
		},
		// ---------------------------------------------------------------
		// V1.0 NMS expansion — Phase 0 schema (2026-05-30).
		// See msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md.
		// Tables are added now (empty) so Phases 1–3 can write to them
		// without further infrastructure work.
		// ---------------------------------------------------------------
		{
			// Phase 2.5 (Wi-Fi Management Frame Analysis) extends the
			// existing wifi_access_points table with the per-beacon
			// Information Element decode fields. ALTER ADD COLUMN is
			// the only safe SQLite ALTER; new columns nullable so
			// existing rows remain valid.
			Description: "Extend wifi_access_points with 802.11 management-frame decode columns",
			Up: `
			ALTER TABLE wifi_access_points ADD COLUMN beacon_interval_tu INTEGER;
			ALTER TABLE wifi_access_points ADD COLUMN rsn_cipher TEXT;
			ALTER TABLE wifi_access_points ADD COLUMN rsn_akm TEXT;
			ALTER TABLE wifi_access_points ADD COLUMN phy_capabilities TEXT;
			ALTER TABLE wifi_access_points ADD COLUMN supports_11k INTEGER DEFAULT 0;
			ALTER TABLE wifi_access_points ADD COLUMN supports_11v INTEGER DEFAULT 0;
			ALTER TABLE wifi_access_points ADD COLUMN supports_11r INTEGER DEFAULT 0;
			ALTER TABLE wifi_access_points ADD COLUMN bss_load_json TEXT;
			ALTER TABLE wifi_access_points ADD COLUMN vendor_ies_json TEXT;
		`,
		},
		{
			// Phase 1c — tiered retention. Mirrors health_check_rollups_hourly.
			// hour_bucket is RFC3339 truncated to the hour. UNIQUE
			// constraint enables UPSERT semantics during rollup runs.
			Description: "Create metrics_hourly for tiered retention rollups",
			Up: `
			CREATE TABLE IF NOT EXISTS metrics_hourly (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				metric_type TEXT NOT NULL,
				interface_name TEXT NOT NULL,
				hour_bucket TEXT NOT NULL,
				sample_count INTEGER NOT NULL,
				avg_value REAL,
				min_value REAL,
				max_value REAL,
				p95_value REAL,
				UNIQUE(metric_type, interface_name, hour_bucket)
			);

			CREATE INDEX IF NOT EXISTS idx_metrics_hourly_bucket ON metrics_hourly(hour_bucket);
			CREATE INDEX IF NOT EXISTS idx_metrics_hourly_type ON metrics_hourly(metric_type, hour_bucket);
		`,
		},
		{
			// Phase 1c — daily aggregates roll up from metrics_hourly.
			Description: "Create metrics_daily for long-term retention rollups",
			Up: `
			CREATE TABLE IF NOT EXISTS metrics_daily (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				metric_type TEXT NOT NULL,
				interface_name TEXT NOT NULL,
				day_bucket TEXT NOT NULL,
				sample_count INTEGER NOT NULL,
				avg_value REAL,
				min_value REAL,
				max_value REAL,
				p95_value REAL,
				UNIQUE(metric_type, interface_name, day_bucket)
			);

			CREATE INDEX IF NOT EXISTS idx_metrics_daily_bucket ON metrics_daily(day_bucket);
			CREATE INDEX IF NOT EXISTS idx_metrics_daily_type ON metrics_daily(metric_type, day_bucket);
		`,
		},
		{
			// Phase 1a — DNS endpoint monitoring. Starter capped at 5
			// monitors via in-handler check; Pro unlimited via same flag.
			Description: "Create dns_monitors for scheduled DNS endpoint monitoring",
			Up: `
			CREATE TABLE IF NOT EXISTS dns_monitors (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				hostname TEXT NOT NULL,
				server TEXT,
				record_type TEXT NOT NULL DEFAULT 'A',
				interval_seconds INTEGER NOT NULL DEFAULT 60,
				enabled INTEGER NOT NULL DEFAULT 1,
				warning_ms INTEGER DEFAULT 100,
				critical_ms INTEGER DEFAULT 500,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_dns_monitors_enabled ON dns_monitors(enabled);
			CREATE INDEX IF NOT EXISTS idx_dns_monitors_hostname ON dns_monitors(hostname);
		`,
		},
		{
			// Phase 1b — SSL/TLS cert expiry monitoring. Starter capped
			// at 5 monitors via in-handler check.
			Description: "Create ssl_monitors for cert expiry tracking",
			Up: `
			CREATE TABLE IF NOT EXISTS ssl_monitors (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				host TEXT NOT NULL,
				port INTEGER NOT NULL DEFAULT 443,
				server_name TEXT,
				check_interval_seconds INTEGER NOT NULL DEFAULT 86400,
				enabled INTEGER NOT NULL DEFAULT 1,
				warning_days INTEGER NOT NULL DEFAULT 30,
				critical_days INTEGER NOT NULL DEFAULT 7,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_ssl_monitors_enabled ON ssl_monitors(enabled);
			CREATE INDEX IF NOT EXISTS idx_ssl_monitors_host ON ssl_monitors(host);
		`,
		},
		{
			// Phase 1b — per-observation history for SSL cert sweeps.
			// CASCADE delete: removing a monitor drops its history.
			Description: "Create cert_observations for SSL/TLS cert sweep history",
			Up: `
			CREATE TABLE IF NOT EXISTS cert_observations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				monitor_id TEXT NOT NULL,
				timestamp TEXT NOT NULL,
				subject TEXT,
				issuer TEXT,
				not_before TEXT,
				not_after TEXT,
				sha256_fingerprint TEXT,
				days_remaining INTEGER,
				status TEXT NOT NULL,
				error_message TEXT,
				FOREIGN KEY (monitor_id) REFERENCES ssl_monitors(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_cert_obs_monitor ON cert_observations(monitor_id);
			CREATE INDEX IF NOT EXISTS idx_cert_obs_timestamp ON cert_observations(timestamp);
			CREATE INDEX IF NOT EXISTS idx_cert_obs_status ON cert_observations(status);
		`,
		},
		{
			// Phase 1e — microburst events. INTEGER PK because high churn.
			// sampling_mode records whether the event was caught at
			// 100ms baseline or in 10ms burst mode (auto-throttle).
			Description: "Create microburst_events for sub-poll-interval burst tracking",
			Up: `
			CREATE TABLE IF NOT EXISTS microburst_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				device_id TEXT,
				interface_name TEXT NOT NULL,
				direction TEXT NOT NULL,
				peak_utilization_pct REAL NOT NULL,
				duration_ms INTEGER NOT NULL,
				sampling_mode TEXT NOT NULL,
				link_speed_mbps INTEGER,
				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			);

			CREATE INDEX IF NOT EXISTS idx_microburst_timestamp ON microburst_events(timestamp);
			CREATE INDEX IF NOT EXISTS idx_microburst_device ON microburst_events(device_id);
			CREATE INDEX IF NOT EXISTS idx_microburst_interface ON microburst_events(interface_name);
		`,
		},
		{
			// Phase 2a — estate-wide SNMP poller target list. credentials_id
			// is nullable so a target can be defined before its credentials.
			Description: "Create polling_targets for estate-wide SNMP poller",
			Up: `
			CREATE TABLE IF NOT EXISTS polling_targets (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				ip_address TEXT NOT NULL,
				snmp_version TEXT NOT NULL DEFAULT 'v2c',
				credentials_id TEXT,
				poll_interval_seconds INTEGER NOT NULL DEFAULT 300,
				enabled INTEGER NOT NULL DEFAULT 1,
				last_polled_at TEXT,
				last_status TEXT,
				last_error TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_polling_targets_enabled ON polling_targets(enabled);
			CREATE INDEX IF NOT EXISTS idx_polling_targets_ip ON polling_targets(ip_address);
		`,
		},
		{
			// Phase 2a — encrypted SNMP credential vault. Secret fields
			// are BLOBs holding ciphertext produced by the same key-derivation
			// chain as internal/auth/ session state (coordinate with auth
			// workstream before Phase 2 implementation).
			Description: "Create device_credentials (encrypted vault) for SNMP auth",
			Up: `
			CREATE TABLE IF NOT EXISTS device_credentials (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				snmp_community_enc BLOB,
				snmp_v3_user TEXT,
				snmp_v3_auth_enc BLOB,
				snmp_v3_priv_enc BLOB,
				snmp_v3_auth_proto TEXT,
				snmp_v3_priv_proto TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_device_credentials_name ON device_credentials(name);
		`,
		},
		{
			// Phase 2b — stable topology node graph after identity merge.
			// identity_hash dedupes the same physical device seen via
			// multiple sources (LLDP/CDP/SNMP/discovery). expires_at
			// drives stale-node archival (default 24h after last_seen).
			Description: "Create topology_nodes for stable identity-merged graph",
			Up: `
			CREATE TABLE IF NOT EXISTS topology_nodes (
				id TEXT PRIMARY KEY,
				identity_hash TEXT NOT NULL UNIQUE,
				display_name TEXT NOT NULL,
				device_type TEXT,
				chassis_id TEXT,
				sys_name TEXT,
				primary_mac TEXT,
				primary_ip TEXT,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				expires_at TEXT,
				metadata_json TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_topology_nodes_identity ON topology_nodes(identity_hash);
			CREATE INDEX IF NOT EXISTS idx_topology_nodes_last_seen ON topology_nodes(last_seen);
			CREATE INDEX IF NOT EXISTS idx_topology_nodes_type ON topology_nodes(device_type);
		`,
		},
		{
			// Phase 2b — stable topology edges. evidence_json records the
			// sources backing the link (LLDP|CDP|FDB|SNMP); used by the
			// reconciliation layer to compute confidence.
			Description: "Create topology_links for stable identity-merged edges",
			Up: `
			CREATE TABLE IF NOT EXISTS topology_links (
				id TEXT PRIMARY KEY,
				source_node_id TEXT NOT NULL,
				target_node_id TEXT NOT NULL,
				source_interface TEXT,
				target_interface TEXT,
				link_type TEXT NOT NULL DEFAULT 'unknown',
				status TEXT NOT NULL DEFAULT 'up',
				speed_mbps INTEGER,
				utilization_pct REAL,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				evidence_json TEXT,
				FOREIGN KEY (source_node_id) REFERENCES topology_nodes(id) ON DELETE CASCADE,
				FOREIGN KEY (target_node_id) REFERENCES topology_nodes(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_topology_links_source ON topology_links(source_node_id);
			CREATE INDEX IF NOT EXISTS idx_topology_links_target ON topology_links(target_node_id);
			CREATE INDEX IF NOT EXISTS idx_topology_links_last_seen ON topology_links(last_seen);
		`,
		},
		{
			// Phase 2.5c — clients seen via probe-request frames.
			// Full MAC retained by default per EtherScope-parity decision
			// (2026-05-29). Anonymize toggle replaces mac_full with the
			// OUI prefix only; pnl_json is hashed when anonymized.
			Description: "Create wifi_clients for 802.11 client tracking",
			Up: `
			CREATE TABLE IF NOT EXISTS wifi_clients (
				id TEXT PRIMARY KEY,
				mac_full TEXT NOT NULL UNIQUE,
				vendor_oui TEXT,
				vendor_name TEXT,
				capabilities_json TEXT,
				pnl_json TEXT,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				anonymized INTEGER NOT NULL DEFAULT 0
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_clients_mac ON wifi_clients(mac_full);
			CREATE INDEX IF NOT EXISTS idx_wifi_clients_oui ON wifi_clients(vendor_oui);
			CREATE INDEX IF NOT EXISTS idx_wifi_clients_last_seen ON wifi_clients(last_seen);
		`,
		},
		{
			// Phase 2.5d — full association handshake forensics. Each row
			// is one observed assoc attempt with its terminal status code
			// (802.11-2020 §9.4.1.9). client_mac/ap_bssid are NOT foreign
			// keys — events are observational facts independent of
			// wifi_clients / wifi_access_points lifecycle.
			Description: "Create wifi_associations for association forensics",
			Up: `
			CREATE TABLE IF NOT EXISTS wifi_associations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				client_mac TEXT NOT NULL,
				ap_bssid TEXT NOT NULL,
				ssid TEXT,
				attempt_type TEXT NOT NULL,
				status_code INTEGER,
				status_text TEXT,
				failure_stage TEXT,
				duration_ms INTEGER,
				rsn_negotiation_json TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_assoc_timestamp ON wifi_associations(timestamp);
			CREATE INDEX IF NOT EXISTS idx_wifi_assoc_client ON wifi_associations(client_mac);
			CREATE INDEX IF NOT EXISTS idx_wifi_assoc_ap ON wifi_associations(ap_bssid);
			CREATE INDEX IF NOT EXISTS idx_wifi_assoc_status ON wifi_associations(status_code);
		`,
		},
		{
			// Phase 2.5e — roam events correlate disassoc on AP-A with
			// (re)assoc on AP-B for same client. roam_type distinguishes
			// 802.11r FT exchange from full reassoc.
			Description: "Create wifi_roams for client roam tracking",
			Up: `
			CREATE TABLE IF NOT EXISTS wifi_roams (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_mac TEXT NOT NULL,
				from_bssid TEXT NOT NULL,
				to_bssid TEXT NOT NULL,
				ssid TEXT,
				started_at TEXT NOT NULL,
				completed_at TEXT,
				duration_ms INTEGER,
				roam_type TEXT,
				rssi_before INTEGER,
				rssi_after INTEGER
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_roams_started ON wifi_roams(started_at);
			CREATE INDEX IF NOT EXISTS idx_wifi_roams_client ON wifi_roams(client_mac);
			CREATE INDEX IF NOT EXISTS idx_wifi_roams_from ON wifi_roams(from_bssid);
			CREATE INDEX IF NOT EXISTS idx_wifi_roams_to ON wifi_roams(to_bssid);
		`,
		},
		{
			// Phase 2.5f — deauth/disassoc reason codes per 802.11-2020.
			// originator: 'ap' if the AP sent the frame, 'client' if the
			// STA sent it. reason_code covers std 1-66 + vendor-specific.
			Description: "Create wifi_deauths for deauth/disassoc reason-code events",
			Up: `
			CREATE TABLE IF NOT EXISTS wifi_deauths (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				ap_bssid TEXT NOT NULL,
				client_mac TEXT NOT NULL,
				frame_type TEXT NOT NULL,
				reason_code INTEGER NOT NULL,
				reason_text TEXT,
				originator TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_deauths_timestamp ON wifi_deauths(timestamp);
			CREATE INDEX IF NOT EXISTS idx_wifi_deauths_ap ON wifi_deauths(ap_bssid);
			CREATE INDEX IF NOT EXISTS idx_wifi_deauths_client ON wifi_deauths(client_mac);
			CREATE INDEX IF NOT EXISTS idx_wifi_deauths_reason ON wifi_deauths(reason_code);
		`,
		},
		{
			// Phase 2.5g — rogue / evil-twin detection events.
			// status moves active → acknowledged → resolved through the UI.
			Description: "Create wifi_rogues for evil-twin and rogue-AP detection",
			Up: `
			CREATE TABLE IF NOT EXISTS wifi_rogues (
				id TEXT PRIMARY KEY,
				detected_at TEXT NOT NULL,
				ap_bssid TEXT NOT NULL,
				ssid TEXT,
				rogue_type TEXT NOT NULL,
				severity TEXT NOT NULL,
				status TEXT NOT NULL DEFAULT 'active',
				evidence_json TEXT,
				acknowledged_at TEXT,
				resolved_at TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_wifi_rogues_detected ON wifi_rogues(detected_at);
			CREATE INDEX IF NOT EXISTS idx_wifi_rogues_bssid ON wifi_rogues(ap_bssid);
			CREATE INDEX IF NOT EXISTS idx_wifi_rogues_status ON wifi_rogues(status);
			CREATE INDEX IF NOT EXISTS idx_wifi_rogues_severity ON wifi_rogues(severity);
		`,
		},
		{
			// Phase 3a — VoIP call records with MOS scoring via E-model.
			// call_id is the RTP-derived synthetic identifier (src+dst+ssrc).
			Description: "Create voip_calls for RTP MOS scoring",
			Up: `
			CREATE TABLE IF NOT EXISTS voip_calls (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				call_id TEXT NOT NULL,
				src_ip TEXT NOT NULL,
				dst_ip TEXT NOT NULL,
				src_port INTEGER,
				dst_port INTEGER,
				codec TEXT,
				started_at TEXT NOT NULL,
				ended_at TEXT,
				duration_seconds INTEGER,
				mos_score REAL,
				avg_jitter_ms REAL,
				packet_loss_pct REAL,
				avg_latency_ms REAL,
				direction TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_voip_calls_call_id ON voip_calls(call_id);
			CREATE INDEX IF NOT EXISTS idx_voip_calls_started ON voip_calls(started_at);
			CREATE INDEX IF NOT EXISTS idx_voip_calls_mos ON voip_calls(mos_score);
		`,
		},
		{
			// Phase 3b — BGP peer session monitoring via BGP4-MIB.
			// device_id FK soft-references the polled router. state and
			// last_state_change drive transition alerts.
			Description: "Create bgp_sessions for BGP4-MIB neighbor monitoring",
			Up: `
			CREATE TABLE IF NOT EXISTS bgp_sessions (
				id TEXT PRIMARY KEY,
				device_id TEXT,
				peer_address TEXT NOT NULL,
				peer_as INTEGER,
				local_as INTEGER,
				state TEXT NOT NULL,
				established_at TEXT,
				last_state_change TEXT NOT NULL,
				prefixes_received INTEGER DEFAULT 0,
				prefixes_sent INTEGER DEFAULT 0,
				last_error TEXT,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			);

			CREATE INDEX IF NOT EXISTS idx_bgp_sessions_device ON bgp_sessions(device_id);
			CREATE INDEX IF NOT EXISTS idx_bgp_sessions_peer ON bgp_sessions(peer_address);
			CREATE INDEX IF NOT EXISTS idx_bgp_sessions_state ON bgp_sessions(state);
		`,
		},
		{
			// Stage A1.1 — add client_id to V1.0 NMS observation tables
			// (metrics rollups, microburst, voip, bgp).
			Description: "Add client_id to V1.0 NMS observation tables",
			Up: `
			ALTER TABLE metrics_hourly ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE metrics_daily ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE microburst_events ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE voip_calls ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE bgp_sessions ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);

			CREATE INDEX IF NOT EXISTS idx_metrics_hourly_client ON metrics_hourly(client_id);
			CREATE INDEX IF NOT EXISTS idx_metrics_daily_client ON metrics_daily(client_id);
			CREATE INDEX IF NOT EXISTS idx_microburst_events_client ON microburst_events(client_id);
			CREATE INDEX IF NOT EXISTS idx_voip_calls_client ON voip_calls(client_id);
			CREATE INDEX IF NOT EXISTS idx_bgp_sessions_client ON bgp_sessions(client_id);
		`,
		},
		{
			// Stage A1.1 — add client_id to V1.0 NMS event tables
			// (wifi_*: clients, associations, roams, deauths, rogues).
			Description: "Add client_id to V1.0 NMS wifi event tables",
			Up: `
			ALTER TABLE wifi_clients ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE wifi_associations ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE wifi_roams ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE wifi_deauths ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE wifi_rogues ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);

			CREATE INDEX IF NOT EXISTS idx_wifi_clients_client ON wifi_clients(client_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_associations_client ON wifi_associations(client_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_roams_client ON wifi_roams(client_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_deauths_client ON wifi_deauths(client_id);
			CREATE INDEX IF NOT EXISTS idx_wifi_rogues_client ON wifi_rogues(client_id);
		`,
		},
		{
			// Stage A1.1 — add client_id to V1.0 NMS topology and
			// polling tables. dns_monitors/ssl_monitors/cert_observations
			// are NOT migrated because Stage A1.2+ drops them as part
			// of the probe-engine unification.
			Description: "Add client_id to V1.0 NMS topology and polling tables",
			Up: `
			ALTER TABLE topology_nodes ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE topology_links ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE polling_targets ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);
			ALTER TABLE device_credentials ADD COLUMN client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id);

			CREATE INDEX IF NOT EXISTS idx_topology_nodes_client ON topology_nodes(client_id);
			CREATE INDEX IF NOT EXISTS idx_topology_links_client ON topology_links(client_id);
			CREATE INDEX IF NOT EXISTS idx_polling_targets_client ON polling_targets(client_id);
			CREATE INDEX IF NOT EXISTS idx_device_credentials_client ON device_credentials(client_id);
		`,
		},
		{
			// Stage A1.9 (2026-05-31) — drop dns_monitors. The DNS
			// checker in internal/probe/checkers/dns.go writes
			// directly into the unified probe_results table; there is
			// no remaining consumer of the dns_monitors config table.
			// The probes config table replaces it (one row per
			// configured probe with kind='dns').
			Description: "Drop dns_monitors (superseded by probes + probe_results)",
			Up: `
			DROP TABLE IF EXISTS dns_monitors;
		`,
		},
		{
			// Stage A1.9 — drop ssl_monitors + cert_observations. The
			// TLS checker in internal/probe/checkers/tls.go writes
			// directly into probe_results with cert metadata in
			// Metadata JSON. No remaining consumers.
			Description: "Drop ssl_monitors (superseded by probes + probe_results)",
			Up: `
			DROP TABLE IF EXISTS ssl_monitors;
		`,
		},
		{
			// Stage A1.9 — drop cert_observations. Cert expiry data
			// lives in probe_results.metadata_json (kind='tls') from
			// the TLS checker.
			Description: "Drop cert_observations (superseded by probe_results.metadata_json)",
			Up: `
			DROP TABLE IF EXISTS cert_observations;
		`,
		},
		{
			// Stage A2.1 (2026-05-31) — add target_kind + target_id to
			// the metrics tables. interface_name is preserved as a
			// backwards-compat column; new writes set both. The
			// retention engine and tier-aware history queries use
			// (target_kind, target_id) going forward; legacy callers
			// still see interface_name.
			//
			// See SEED_ARCHITECTURE.md section 3.2 — Metrics.
			Description: "Add target_kind + target_id to metrics tables",
			Up: `
			ALTER TABLE metrics ADD COLUMN target_kind TEXT NOT NULL DEFAULT 'interface';
			ALTER TABLE metrics ADD COLUMN target_id TEXT NOT NULL DEFAULT '';
			ALTER TABLE metrics_hourly ADD COLUMN target_kind TEXT NOT NULL DEFAULT 'interface';
			ALTER TABLE metrics_hourly ADD COLUMN target_id TEXT NOT NULL DEFAULT '';
			ALTER TABLE metrics_daily ADD COLUMN target_kind TEXT NOT NULL DEFAULT 'interface';
			ALTER TABLE metrics_daily ADD COLUMN target_id TEXT NOT NULL DEFAULT '';

			UPDATE metrics SET target_id = interface_name WHERE target_id = '';
			UPDATE metrics_hourly SET target_id = interface_name WHERE target_id = '';
			UPDATE metrics_daily SET target_id = interface_name WHERE target_id = '';

			CREATE INDEX IF NOT EXISTS idx_metrics_target ON metrics(target_kind, target_id);
			CREATE INDEX IF NOT EXISTS idx_metrics_hourly_target ON metrics_hourly(target_kind, target_id, hour_bucket);
			CREATE INDEX IF NOT EXISTS idx_metrics_daily_target ON metrics_daily(target_kind, target_id, day_bucket);
		`,
		},
		{
			// Stage A2.1 — probe-result rollup tables. Parallel to
			// metrics_hourly/daily; aggregated by (client_id, kind,
			// probe_id) over hour/day buckets. The retention engine
			// rolls probe_results into these tables and tier-purges
			// raw probe_results past 7 days.
			Description: "Create probe_rollups_hourly + probe_rollups_daily",
			Up: `
			CREATE TABLE IF NOT EXISTS probe_rollups_hourly (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				probe_id TEXT NOT NULL,
				hour_bucket TEXT NOT NULL,
				sample_count INTEGER NOT NULL,
				success_count INTEGER NOT NULL,
				avg_latency_ms REAL,
				min_latency_ms REAL,
				max_latency_ms REAL,
				p95_latency_ms REAL,
				UNIQUE(client_id, kind, probe_id, hour_bucket)
			);
			CREATE INDEX IF NOT EXISTS idx_probe_rollups_hourly_bucket ON probe_rollups_hourly(hour_bucket);
			CREATE INDEX IF NOT EXISTS idx_probe_rollups_hourly_probe ON probe_rollups_hourly(probe_id, hour_bucket);

			CREATE TABLE IF NOT EXISTS probe_rollups_daily (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				probe_id TEXT NOT NULL,
				day_bucket TEXT NOT NULL,
				sample_count INTEGER NOT NULL,
				success_count INTEGER NOT NULL,
				avg_latency_ms REAL,
				min_latency_ms REAL,
				max_latency_ms REAL,
				p95_latency_ms REAL,
				UNIQUE(client_id, kind, probe_id, day_bucket)
			);
			CREATE INDEX IF NOT EXISTS idx_probe_rollups_daily_bucket ON probe_rollups_daily(day_bucket);
			CREATE INDEX IF NOT EXISTS idx_probe_rollups_daily_probe ON probe_rollups_daily(probe_id, day_bucket);
		`,
		},
		{
			// Stage A3.1 — collector_chain on polling_targets.
			// The estate-wide SNMP poller dispatches a target's chain
			// in order; each chain entry names a Collector registered
			// at startup (sys_info, if_table, lldp, arp, fdb, ...).
			// Default chain covers the topology-relevant collectors so
			// brand-new targets immediately feed Stage A4 topology.
			Description: "Add collector_chain to polling_targets",
			Up: `
			ALTER TABLE polling_targets ADD COLUMN collector_chain TEXT NOT NULL DEFAULT '["sys_info","if_table","lldp","arp","fdb"]';
		`,
		},
		{
			// Stage A3.5b — unified persistence for every SNMP
			// collector observation. One row per (client, target,
			// kind, observed_at); payload_json carries the typed
			// observation struct. Stage A4 reconciler reads kind-
			// filtered rows to build topology; the listener pipeline
			// reads them to compute deltas and emit events.
			//
			// Single table beats per-kind tables for V1.0 because:
			// 1. Schema evolution is JSON-only — adding a new
			//    collector kind doesn't touch the migration list.
			// 2. Retention is one DELETE WHERE observed_at < cutoff
			//    instead of N tables.
			// 3. The listener pipeline iterates kinds at SQL time,
			//    not at schema definition time.
			Description: "Create snmp_observations for unified collector output",
			Up: `
			CREATE TABLE IF NOT EXISTS snmp_observations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				target_id TEXT NOT NULL,
				kind TEXT NOT NULL,
				observed_at TEXT NOT NULL,
				payload_json TEXT NOT NULL,
				ingested_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_snmp_observations_client_kind ON snmp_observations(client_id, kind, observed_at);
			CREATE INDEX IF NOT EXISTS idx_snmp_observations_target ON snmp_observations(target_id, observed_at);
			CREATE INDEX IF NOT EXISTS idx_snmp_observations_observed_at ON snmp_observations(observed_at);
		`,
		},
		{
			// Stage A3.5e — unified persistence for passive-ingress
			// events (syslog, snmp traps, future netflow). Mirrors
			// snmp_observations: per-kind payload_json + client /
			// source / timestamp indices for retention and for the
			// listener-driven alert rules in Stage A4.
			//
			// source_addr is indexed because the enrichment step
			// resolves it -> (client_id, target_id); O(log n) joins
			// are mandatory during trap storms.
			Description: "Create listener_events for syslog + snmp trap ingress",
			Up: `
			CREATE TABLE IF NOT EXISTS listener_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				source_addr TEXT NOT NULL,
				target_kind TEXT,
				target_id TEXT,
				severity TEXT,
				observed_at TEXT NOT NULL,
				payload_json TEXT NOT NULL,
				ingested_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_listener_events_client_kind ON listener_events(client_id, kind, observed_at);
			CREATE INDEX IF NOT EXISTS idx_listener_events_source ON listener_events(source_addr, observed_at);
			CREATE INDEX IF NOT EXISTS idx_listener_events_observed_at ON listener_events(observed_at);
		`,
		},
		{
			// Stage A4.2 — target -> node mapping. Most A4 reconcilers
			// only see (client_id, target_id) on their observations
			// (if_table, lldp, arp, fdb, routing, bgp4) and need to
			// look up the topology_node that sysinfo identified. This
			// table is the join: sysinfo reconciler writes
			// (client_id, target_id, node_id) on every upsert; later
			// reconcilers read it to attach their slice of state.
			Description: "Create topology_target_nodes mapping",
			Up: `
			CREATE TABLE IF NOT EXISTS topology_target_nodes (
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				target_id TEXT NOT NULL,
				node_id TEXT NOT NULL REFERENCES topology_nodes(id) ON DELETE CASCADE,
				last_seen TEXT NOT NULL,
				PRIMARY KEY (client_id, target_id)
			);

			CREATE INDEX IF NOT EXISTS idx_topology_target_nodes_node ON topology_target_nodes(node_id);
		`,
		},
		{
			// Stage A4.2 — per-node interface state. One row per
			// (node_id, if_index); reconciled from if_table
			// observations. Status + speed live as columns so alert
			// rules can index them cheaply ("WHERE if_oper_status = 2"
			// for down interfaces, "WHERE speed_bps < 1e9" for sub-
			// gigabit links).
			Description: "Create topology_interfaces for per-node if_table state",
			Up: `
			CREATE TABLE IF NOT EXISTS topology_interfaces (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				node_id TEXT NOT NULL REFERENCES topology_nodes(id) ON DELETE CASCADE,
				if_index INTEGER NOT NULL,
				if_name TEXT,
				if_descr TEXT,
				if_alias TEXT,
				if_type INTEGER,
				if_admin_status INTEGER,
				if_oper_status INTEGER,
				if_phys_addr TEXT,
				speed_bps INTEGER,
				last_seen TEXT NOT NULL,
				UNIQUE(node_id, if_index)
			);

			CREATE INDEX IF NOT EXISTS idx_topology_interfaces_node ON topology_interfaces(node_id);
			CREATE INDEX IF NOT EXISTS idx_topology_interfaces_oper ON topology_interfaces(if_oper_status);
			CREATE INDEX IF NOT EXISTS idx_topology_interfaces_last_seen ON topology_interfaces(last_seen);
		`,
		},
		{
			// Stage A5.10 — operator-defined alert rules. The
			// hardcoded rule set in
			// internal/alerts/pipeline/listener_pipeline.go is the
			// fallback; rules in this table are applied first.
			//
			// Match fields are intentionally simple for V1.0:
			//   match_kind             listener event kind (or "" = any)
			//   match_severity         listener severity (or "" = any)
			//   match_payload_contains substring check on payload_json
			//
			// Alert fields drive the row written into the alerts
			// table; title/message support {{.SourceAddr}} etc.
			// via text/template.
			Description: "Create alert_rules for operator-defined rules",
			Up: `
			CREATE TABLE IF NOT EXISTS alert_rules (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				enabled INTEGER NOT NULL DEFAULT 1,
				match_kind TEXT,
				match_severity TEXT,
				match_payload_contains TEXT,
				alert_type TEXT NOT NULL,
				alert_severity TEXT NOT NULL,
				alert_title TEXT NOT NULL,
				alert_message TEXT NOT NULL,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

			CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);
		`,
		},
		{
			// Stage A4.4 — ARP binding store. Reconciled from arp
			// observations; one row per (source_node, ifIndex, IP).
			// mac_address is indexed because the reconciler joins
			// against topology_nodes.primary_mac to backfill node
			// identity (IP-to-MAC -> MAC matches a node's chassis ->
			// IP becomes node.primary_ip).
			Description: "Create topology_arp_bindings",
			Up: `
			CREATE TABLE IF NOT EXISTS topology_arp_bindings (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				source_node_id TEXT NOT NULL REFERENCES topology_nodes(id) ON DELETE CASCADE,
				if_index INTEGER NOT NULL,
				ip_address TEXT NOT NULL,
				mac_address TEXT NOT NULL,
				media_type INTEGER,
				last_seen TEXT NOT NULL,
				UNIQUE(source_node_id, if_index, ip_address)
			);

			CREATE INDEX IF NOT EXISTS idx_arp_bindings_mac ON topology_arp_bindings(mac_address);
			CREATE INDEX IF NOT EXISTS idx_arp_bindings_ip ON topology_arp_bindings(ip_address);
			CREATE INDEX IF NOT EXISTS idx_arp_bindings_last_seen ON topology_arp_bindings(last_seen);
		`,
		},
		{
			// Stage A5/#1380 — persistent alert suppression. Replaces
			// the in-memory map both alert pipelines used to keep so
			// a restart mid-incident doesn't re-fire alerts that were
			// already emitted. fingerprint is the sha256 hash from
			// the original suppression key (rule_id + source + kind).
			Description: "Create alert_suppressions",
			Up: `
				CREATE TABLE IF NOT EXISTS alert_suppressions (
					fingerprint TEXT PRIMARY KEY,
					rule_id TEXT NOT NULL,
					entity_key TEXT NOT NULL,
					suppress_until TEXT NOT NULL,
					created_at TEXT NOT NULL
				);

				CREATE INDEX IF NOT EXISTS idx_alert_suppressions_until
				  ON alert_suppressions(suppress_until);
			`,
		},
		{
			// #1379 — time-windowed alert rules. window_seconds=0
			// preserves fire-on-first-match (legacy default).
			// threshold_count is the count of matching events that
			// must accrue inside window_seconds before the rule fires.
			Description: "Add window_seconds + threshold_count to alert_rules",
			Up: `
				ALTER TABLE alert_rules
				  ADD COLUMN window_seconds INTEGER NOT NULL DEFAULT 0;
				ALTER TABLE alert_rules
				  ADD COLUMN threshold_count INTEGER NOT NULL DEFAULT 1;
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
