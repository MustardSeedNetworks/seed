-- 00024_drop_writerless_tables.sql — delete every table nothing writes
-- (P-A6, #3029).
--
-- 00001 created these for features that were never built or were built
-- without the table: Bluetooth, BGP, Wi-Fi clients/associations/roams/deauths,
-- VoIP calls, a second device inventory (discovered_devices and its
-- interfaces, history and ports) and a pipeline run log. No Go code wrote any
-- of them. wifi_networks, wifi_access_points, channel_utilization,
-- network_problems and oui_vendors had a repository (DiscoveryRepository) that
-- no code called; network_problems' Go side did not even match its columns.
-- An empty table that implies a capability is the #2327 defect class, so each
-- one goes. A feature that ships later defines the table it writes, with its
-- producer, in its own migration: VoIP MOS (P-A7), Bluetooth (P-E3) and the
-- Wave 9 Wi-Fi rows. TestEveryTableHasAWriter keeps it that way.
--
-- microburst_events stays (the P-A6 capture listener writes it). Its device_id
-- referenced discovered_devices and was never set: bursts are measured on the
-- probe's own link and named by interface. SQLite cannot drop a foreign-key
-- column in place, so the table is rebuilt without it. Children drop before
-- their parents, so no ON DELETE action fires.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE microburst_events_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				interface_name TEXT NOT NULL,
				direction TEXT NOT NULL,
				peak_utilization_pct REAL NOT NULL,
				duration_ms INTEGER NOT NULL,
				sampling_mode TEXT NOT NULL,
				link_speed_mbps INTEGER,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)
			) STRICT;
INSERT INTO microburst_events_new
	(id, timestamp, interface_name, direction, peak_utilization_pct,
	 duration_ms, sampling_mode, link_speed_mbps, client_id)
SELECT id, timestamp, interface_name, direction, peak_utilization_pct,
	duration_ms, sampling_mode, link_speed_mbps, client_id
FROM microburst_events;
DROP TABLE microburst_events;
ALTER TABLE microburst_events_new RENAME TO microburst_events;
CREATE INDEX idx_microburst_events_client ON microburst_events(client_id);
CREATE INDEX idx_microburst_interface ON microburst_events(interface_name);
CREATE INDEX idx_microburst_timestamp ON microburst_events(timestamp);

DROP TABLE bgp_sessions;
DROP TABLE bluetooth_devices;
DROP TABLE bluetooth_scan_history;
DROP TABLE channel_utilization;
DROP TABLE device_interfaces;
DROP TABLE device_ports;
DROP TABLE discovery_history;
DROP TABLE mib_sources;
DROP TABLE network_problems;
DROP TABLE oui_vendors;
DROP TABLE pipeline_runs;
DROP TABLE voip_calls;
DROP TABLE wifi_access_points;
DROP TABLE wifi_associations;
DROP TABLE wifi_clients;
DROP TABLE wifi_deauths;
DROP TABLE wifi_roams;
DROP TABLE discovery_interfaces;
DROP TABLE wifi_networks;
DROP TABLE discovered_devices;

-- +goose Down
CREATE TABLE discovered_devices (
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
				is_online INTEGER DEFAULT 1 CHECK (is_online IN (0,1)),
				notes TEXT,
				tags TEXT,
				metadata_json TEXT,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_disc_devices_auth ON discovered_devices(authorization_status);
CREATE INDEX idx_disc_devices_last_seen ON discovered_devices(last_seen);
CREATE INDEX idx_disc_devices_mac ON discovered_devices(primary_mac);
CREATE INDEX idx_disc_devices_online ON discovered_devices(is_online);
CREATE INDEX idx_disc_devices_type ON discovered_devices(device_type);
CREATE INDEX idx_disc_devices_vendor ON discovered_devices(vendor);
CREATE INDEX idx_discovered_devices_client ON discovered_devices(client_id);
CREATE TABLE wifi_networks (
				id TEXT PRIMARY KEY,
				ssid TEXT NOT NULL,
				is_hidden INTEGER DEFAULT 0 CHECK (is_hidden IN (0,1)),
				security_type TEXT,
				authorization_status TEXT DEFAULT 'unknown',
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				UNIQUE(ssid, security_type)
			) STRICT;
CREATE INDEX idx_wifi_networks_auth ON wifi_networks(authorization_status);
CREATE INDEX idx_wifi_networks_client ON wifi_networks(client_id);
CREATE INDEX idx_wifi_networks_ssid ON wifi_networks(ssid);
CREATE TABLE discovery_interfaces (
				id TEXT PRIMARY KEY,
				device_id TEXT NOT NULL,
				interface_type TEXT NOT NULL,
				mac_address TEXT NOT NULL,
				ip_addresses TEXT,
				interface_name TEXT,
				is_primary INTEGER DEFAULT 0 CHECK (is_primary IN (0,1)),

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
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE,
				UNIQUE(device_id, mac_address)
			) STRICT;
CREATE INDEX idx_disc_iface_bssid ON discovery_interfaces(bssid);
CREATE INDEX idx_disc_iface_device ON discovery_interfaces(device_id);
CREATE INDEX idx_disc_iface_mac ON discovery_interfaces(mac_address);
CREATE INDEX idx_disc_iface_ssid ON discovery_interfaces(ssid);
CREATE INDEX idx_disc_iface_type ON discovery_interfaces(interface_type);
CREATE INDEX idx_discovery_interfaces_client ON discovery_interfaces(client_id);
CREATE TABLE wifi_roams (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_wifi_roams_client ON wifi_roams(client_mac);
CREATE INDEX idx_wifi_roams_from ON wifi_roams(from_bssid);
CREATE INDEX idx_wifi_roams_started ON wifi_roams(started_at);
CREATE INDEX idx_wifi_roams_to ON wifi_roams(to_bssid);
CREATE TABLE wifi_deauths (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				ap_bssid TEXT NOT NULL,
				client_mac TEXT NOT NULL,
				frame_type TEXT NOT NULL,
				reason_code INTEGER NOT NULL,
				reason_text TEXT,
				originator TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_wifi_deauths_ap ON wifi_deauths(ap_bssid);
CREATE INDEX idx_wifi_deauths_client ON wifi_deauths(client_mac);
CREATE INDEX idx_wifi_deauths_reason ON wifi_deauths(reason_code);
CREATE INDEX idx_wifi_deauths_timestamp ON wifi_deauths(timestamp);
CREATE TABLE wifi_clients (
				id TEXT PRIMARY KEY,
				mac_full TEXT NOT NULL UNIQUE,
				vendor_oui TEXT,
				vendor_name TEXT,
				capabilities_json TEXT,
				pnl_json TEXT,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				anonymized INTEGER NOT NULL DEFAULT 0 CHECK (anonymized IN (0,1))
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_wifi_clients_client ON wifi_clients(client_id);
CREATE INDEX idx_wifi_clients_last_seen ON wifi_clients(last_seen);
CREATE INDEX idx_wifi_clients_mac ON wifi_clients(mac_full);
CREATE INDEX idx_wifi_clients_oui ON wifi_clients(vendor_oui);
CREATE TABLE wifi_associations (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_wifi_assoc_ap ON wifi_associations(ap_bssid);
CREATE INDEX idx_wifi_assoc_client ON wifi_associations(client_mac);
CREATE INDEX idx_wifi_assoc_status ON wifi_associations(status_code);
CREATE INDEX idx_wifi_assoc_timestamp ON wifi_associations(timestamp);
CREATE INDEX idx_wifi_associations_client ON wifi_associations(client_id);
CREATE TABLE wifi_access_points (
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
				is_authorized INTEGER DEFAULT 1 CHECK (is_authorized IN (0,1)),

				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), beacon_interval_tu INTEGER, rsn_cipher TEXT, rsn_akm TEXT, phy_capabilities TEXT, supports_11k INTEGER DEFAULT 0 CHECK (supports_11k IN (0,1)), supports_11v INTEGER DEFAULT 0 CHECK (supports_11v IN (0,1)), supports_11r INTEGER DEFAULT 0 CHECK (supports_11r IN (0,1)), bss_load_json TEXT, vendor_ies_json TEXT,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL,
				FOREIGN KEY (ssid_id) REFERENCES wifi_networks(id) ON DELETE SET NULL
			) STRICT;
CREATE INDEX idx_wifi_access_points_client ON wifi_access_points(client_id);
CREATE INDEX idx_wifi_aps_band ON wifi_access_points(band);
CREATE INDEX idx_wifi_aps_bssid ON wifi_access_points(bssid);
CREATE INDEX idx_wifi_aps_channel ON wifi_access_points(channel);
CREATE INDEX idx_wifi_aps_device ON wifi_access_points(device_id);
CREATE INDEX idx_wifi_aps_ssid ON wifi_access_points(ssid_id);
CREATE TABLE voip_calls (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_voip_calls_call_id ON voip_calls(call_id);
CREATE INDEX idx_voip_calls_client ON voip_calls(client_id);
CREATE INDEX idx_voip_calls_mos ON voip_calls(mos_score);
CREATE INDEX idx_voip_calls_started ON voip_calls(started_at);
CREATE TABLE pipeline_runs (
				id TEXT PRIMARY KEY,
				started_at TEXT NOT NULL,
				completed_at TEXT,
				status TEXT NOT NULL,
				triggered_by TEXT,
				phases_enabled TEXT NOT NULL,
				config_json TEXT,
				summary_json TEXT,
				error_message TEXT
			) STRICT;
CREATE INDEX idx_pipeline_runs_started ON pipeline_runs(started_at);
CREATE INDEX idx_pipeline_runs_status ON pipeline_runs(status);
CREATE TABLE oui_vendors (
				oui TEXT PRIMARY KEY,
				vendor_name TEXT NOT NULL,
				vendor_short TEXT,
				is_private INTEGER DEFAULT 0 CHECK (is_private IN (0,1)),
				device_category TEXT,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			) STRICT;
CREATE INDEX idx_oui_category ON oui_vendors(device_category);
CREATE INDEX idx_oui_vendor_name ON oui_vendors(vendor_name);
CREATE TABLE network_problems (
				id TEXT PRIMARY KEY,
				problem_type TEXT NOT NULL,
				severity TEXT NOT NULL,
				device_id TEXT,
				interface_id TEXT,
				description TEXT NOT NULL,
				details_json TEXT,
				is_resolved INTEGER DEFAULT 0 CHECK (is_resolved IN (0,1)),
				detected_at TEXT NOT NULL,
				resolved_at TEXT,
				acknowledged_at TEXT,
				acknowledged_by TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE,
				FOREIGN KEY (interface_id) REFERENCES discovery_interfaces(id) ON DELETE CASCADE
			) STRICT;
CREATE INDEX idx_net_problems_detected ON network_problems(detected_at);
CREATE INDEX idx_net_problems_device ON network_problems(device_id);
CREATE INDEX idx_net_problems_resolved ON network_problems(is_resolved);
CREATE INDEX idx_net_problems_severity ON network_problems(severity);
CREATE INDEX idx_net_problems_type ON network_problems(problem_type);
CREATE INDEX idx_network_problems_client ON network_problems(client_id);
CREATE TABLE mib_sources (
				mib_name TEXT PRIMARY KEY,
				description TEXT,
				vendor TEXT,
				rfc_reference TEXT,
				loaded_at TEXT DEFAULT (datetime('now'))
			) STRICT;
CREATE TABLE discovery_history (
				id TEXT PRIMARY KEY,
				device_id TEXT NOT NULL,
				event_type TEXT NOT NULL,
				event_data TEXT,
				recorded_at TEXT NOT NULL, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE
			) STRICT;
CREATE INDEX idx_disc_history_device ON discovery_history(device_id);
CREATE INDEX idx_disc_history_time ON discovery_history(recorded_at);
CREATE INDEX idx_disc_history_type ON discovery_history(event_type);
CREATE INDEX idx_discovery_history_client ON discovery_history(client_id);
CREATE TABLE device_ports (
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
			) STRICT;
CREATE INDEX idx_device_ports_device ON device_ports(device_id);
CREATE INDEX idx_device_ports_port ON device_ports(port);
CREATE UNIQUE INDEX idx_device_ports_unique ON device_ports(device_id, port, protocol);
CREATE TABLE device_interfaces (
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
			) STRICT;
CREATE INDEX idx_device_interfaces_device ON device_interfaces(device_id);
CREATE INDEX idx_device_interfaces_mac ON device_interfaces(mac_address);
CREATE UNIQUE INDEX idx_device_interfaces_unique ON device_interfaces(device_id, if_index);
CREATE TABLE channel_utilization (
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

				recorded_at TEXT NOT NULL, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				UNIQUE(channel, band, recorded_at)
			) STRICT;
CREATE INDEX idx_channel_util_channel ON channel_utilization(channel, band);
CREATE INDEX idx_channel_util_time ON channel_utilization(recorded_at);
CREATE INDEX idx_channel_utilization_client ON channel_utilization(client_id);
CREATE TABLE bluetooth_scan_history (
				id TEXT PRIMARY KEY,
				adapter_name TEXT,
				scan_type TEXT NOT NULL,
				devices_found INTEGER NOT NULL,
				classic_count INTEGER DEFAULT 0,
				ble_count INTEGER DEFAULT 0,
				scan_duration_ms INTEGER,
				scan_time TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_bluetooth_scan_history_client ON bluetooth_scan_history(client_id);
CREATE INDEX idx_bt_scan_time ON bluetooth_scan_history(scan_time);
CREATE INDEX idx_bt_scan_type ON bluetooth_scan_history(scan_type);
CREATE TABLE bluetooth_devices (
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
				is_connected INTEGER DEFAULT 0 CHECK (is_connected IN (0,1)),
				is_connectable INTEGER DEFAULT 0 CHECK (is_connectable IN (0,1)),
				is_authorized INTEGER DEFAULT 0 CHECK (is_authorized IN (0,1)),
				is_trusted INTEGER DEFAULT 0 CHECK (is_trusted IN (0,1)),
				is_paired INTEGER DEFAULT 0 CHECK (is_paired IN (0,1)),
				is_blocked INTEGER DEFAULT 0 CHECK (is_blocked IN (0,1)),
				service_uuids_json TEXT,
				manufacturer_id INTEGER,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			) STRICT;
CREATE INDEX idx_bluetooth_devices_client ON bluetooth_devices(client_id);
CREATE INDEX idx_bt_devices_address ON bluetooth_devices(address);
CREATE INDEX idx_bt_devices_authorized ON bluetooth_devices(is_authorized);
CREATE INDEX idx_bt_devices_class ON bluetooth_devices(device_class);
CREATE INDEX idx_bt_devices_connected ON bluetooth_devices(is_connected);
CREATE INDEX idx_bt_devices_last_seen ON bluetooth_devices(last_seen);
CREATE INDEX idx_bt_devices_name ON bluetooth_devices(name);
CREATE INDEX idx_bt_devices_type ON bluetooth_devices(bluetooth_type);
CREATE INDEX idx_bt_devices_vendor ON bluetooth_devices(vendor);
CREATE TABLE bgp_sessions (
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
				last_seen TEXT NOT NULL, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			) STRICT;
CREATE INDEX idx_bgp_sessions_client ON bgp_sessions(client_id);
CREATE INDEX idx_bgp_sessions_device ON bgp_sessions(device_id);
CREATE INDEX idx_bgp_sessions_peer ON bgp_sessions(peer_address);
CREATE INDEX idx_bgp_sessions_state ON bgp_sessions(state);
CREATE TABLE microburst_events_old (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				device_id TEXT,
				interface_name TEXT NOT NULL,
				direction TEXT NOT NULL,
				peak_utilization_pct REAL NOT NULL,
				duration_ms INTEGER NOT NULL,
				sampling_mode TEXT NOT NULL,
				link_speed_mbps INTEGER, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			) STRICT;
INSERT INTO microburst_events_old
	(id, timestamp, interface_name, direction, peak_utilization_pct,
	 duration_ms, sampling_mode, link_speed_mbps, client_id)
SELECT id, timestamp, interface_name, direction, peak_utilization_pct,
	duration_ms, sampling_mode, link_speed_mbps, client_id
FROM microburst_events;
DROP TABLE microburst_events;
ALTER TABLE microburst_events_old RENAME TO microburst_events;
CREATE INDEX idx_microburst_device ON microburst_events(device_id);
CREATE INDEX idx_microburst_events_client ON microburst_events(client_id);
CREATE INDEX idx_microburst_interface ON microburst_events(interface_name);
CREATE INDEX idx_microburst_timestamp ON microburst_events(timestamp);
