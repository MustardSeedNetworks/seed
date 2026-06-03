-- index: idx_alert_rules_enabled
CREATE INDEX idx_alert_rules_enabled ON alert_rules(enabled);

-- index: idx_alert_suppressions_until
CREATE INDEX idx_alert_suppressions_until
				  ON alert_suppressions(suppress_until);

-- index: idx_alerts_acknowledged
CREATE INDEX idx_alerts_acknowledged ON alerts(acknowledged);

-- index: idx_alerts_client
CREATE INDEX idx_alerts_client ON alerts(client_id);

-- index: idx_alerts_created
CREATE INDEX idx_alerts_created ON alerts(created_at);

-- index: idx_alerts_device
CREATE INDEX idx_alerts_device ON alerts(device_id);

-- index: idx_alerts_resolved
CREATE INDEX idx_alerts_resolved ON alerts(resolved);

-- index: idx_alerts_severity
CREATE INDEX idx_alerts_severity ON alerts(severity);

-- index: idx_alerts_type
CREATE INDEX idx_alerts_type ON alerts(type);

-- index: idx_api_tokens_active
CREATE INDEX idx_api_tokens_active ON api_tokens(revoked_at);

-- index: idx_api_tokens_hash
CREATE INDEX idx_api_tokens_hash   ON api_tokens(token_hash);

-- index: idx_api_tokens_owner
CREATE INDEX idx_api_tokens_owner  ON api_tokens(owner_username);

-- index: idx_arp_bindings_ip
CREATE INDEX idx_arp_bindings_ip ON topology_arp_bindings(ip_address);

-- index: idx_arp_bindings_last_seen
CREATE INDEX idx_arp_bindings_last_seen ON topology_arp_bindings(last_seen);

-- index: idx_arp_bindings_mac
CREATE INDEX idx_arp_bindings_mac ON topology_arp_bindings(mac_address);

-- index: idx_audit_action
CREATE INDEX idx_audit_action ON audit_log(action);

-- index: idx_audit_resource
CREATE INDEX idx_audit_resource ON audit_log(resource_type, resource_id);

-- index: idx_audit_timestamp
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);

-- index: idx_audit_user
CREATE INDEX idx_audit_user ON audit_log(user);

-- index: idx_bgp_sessions_client
CREATE INDEX idx_bgp_sessions_client ON bgp_sessions(client_id);

-- index: idx_bgp_sessions_device
CREATE INDEX idx_bgp_sessions_device ON bgp_sessions(device_id);

-- index: idx_bgp_sessions_peer
CREATE INDEX idx_bgp_sessions_peer ON bgp_sessions(peer_address);

-- index: idx_bgp_sessions_state
CREATE INDEX idx_bgp_sessions_state ON bgp_sessions(state);

-- index: idx_bluetooth_devices_client
CREATE INDEX idx_bluetooth_devices_client ON bluetooth_devices(client_id);

-- index: idx_bluetooth_scan_history_client
CREATE INDEX idx_bluetooth_scan_history_client ON bluetooth_scan_history(client_id);

-- index: idx_bt_devices_address
CREATE INDEX idx_bt_devices_address ON bluetooth_devices(address);

-- index: idx_bt_devices_authorized
CREATE INDEX idx_bt_devices_authorized ON bluetooth_devices(is_authorized);

-- index: idx_bt_devices_class
CREATE INDEX idx_bt_devices_class ON bluetooth_devices(device_class);

-- index: idx_bt_devices_connected
CREATE INDEX idx_bt_devices_connected ON bluetooth_devices(is_connected);

-- index: idx_bt_devices_last_seen
CREATE INDEX idx_bt_devices_last_seen ON bluetooth_devices(last_seen);

-- index: idx_bt_devices_name
CREATE INDEX idx_bt_devices_name ON bluetooth_devices(name);

-- index: idx_bt_devices_type
CREATE INDEX idx_bt_devices_type ON bluetooth_devices(bluetooth_type);

-- index: idx_bt_devices_vendor
CREATE INDEX idx_bt_devices_vendor ON bluetooth_devices(vendor);

-- index: idx_bt_scan_time
CREATE INDEX idx_bt_scan_time ON bluetooth_scan_history(scan_time);

-- index: idx_bt_scan_type
CREATE INDEX idx_bt_scan_type ON bluetooth_scan_history(scan_type);

-- index: idx_channel_util_channel
CREATE INDEX idx_channel_util_channel ON channel_utilization(channel, band);

-- index: idx_channel_util_time
CREATE INDEX idx_channel_util_time ON channel_utilization(recorded_at);

-- index: idx_channel_utilization_client
CREATE INDEX idx_channel_utilization_client ON channel_utilization(client_id);

-- index: idx_clients_slug
CREATE INDEX idx_clients_slug ON clients(slug);

-- index: idx_device_credentials_client
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);

-- index: idx_device_credentials_name
CREATE INDEX idx_device_credentials_name ON device_credentials(name);

-- index: idx_device_interfaces_device
CREATE INDEX idx_device_interfaces_device ON device_interfaces(device_id);

-- index: idx_device_interfaces_mac
CREATE INDEX idx_device_interfaces_mac ON device_interfaces(mac_address);

-- index: idx_device_interfaces_unique
CREATE UNIQUE INDEX idx_device_interfaces_unique ON device_interfaces(device_id, if_index);

-- index: idx_device_ports_device
CREATE INDEX idx_device_ports_device ON device_ports(device_id);

-- index: idx_device_ports_port
CREATE INDEX idx_device_ports_port ON device_ports(port);

-- index: idx_device_ports_unique
CREATE UNIQUE INDEX idx_device_ports_unique ON device_ports(device_id, port, protocol);

-- index: idx_device_vulns_cve
CREATE INDEX idx_device_vulns_cve ON device_vulnerabilities(cve_id);

-- index: idx_device_vulns_device
CREATE INDEX idx_device_vulns_device ON device_vulnerabilities(device_id);

-- index: idx_device_vulns_severity
CREATE INDEX idx_device_vulns_severity ON device_vulnerabilities(severity);

-- index: idx_device_vulns_status
CREATE INDEX idx_device_vulns_status ON device_vulnerabilities(status);

-- index: idx_device_vulns_unique
CREATE UNIQUE INDEX idx_device_vulns_unique ON device_vulnerabilities(device_id, cve_id);

-- index: idx_devices_active
CREATE INDEX idx_devices_active ON devices(is_active);

-- index: idx_devices_hostname
CREATE INDEX idx_devices_hostname ON devices(hostname);

-- index: idx_devices_ip
CREATE INDEX idx_devices_ip ON devices(ip_address);

-- index: idx_devices_last_seen
CREATE INDEX idx_devices_last_seen ON devices(last_seen);

-- index: idx_devices_mac
CREATE INDEX idx_devices_mac ON devices(mac_address);

-- index: idx_disc_devices_auth
CREATE INDEX idx_disc_devices_auth ON discovered_devices(authorization_status);

-- index: idx_disc_devices_last_seen
CREATE INDEX idx_disc_devices_last_seen ON discovered_devices(last_seen);

-- index: idx_disc_devices_mac
CREATE INDEX idx_disc_devices_mac ON discovered_devices(primary_mac);

-- index: idx_disc_devices_online
CREATE INDEX idx_disc_devices_online ON discovered_devices(is_online);

-- index: idx_disc_devices_type
CREATE INDEX idx_disc_devices_type ON discovered_devices(device_type);

-- index: idx_disc_devices_vendor
CREATE INDEX idx_disc_devices_vendor ON discovered_devices(vendor);

-- index: idx_disc_history_device
CREATE INDEX idx_disc_history_device ON discovery_history(device_id);

-- index: idx_disc_history_time
CREATE INDEX idx_disc_history_time ON discovery_history(recorded_at);

-- index: idx_disc_history_type
CREATE INDEX idx_disc_history_type ON discovery_history(event_type);

-- index: idx_disc_iface_bssid
CREATE INDEX idx_disc_iface_bssid ON discovery_interfaces(bssid);

-- index: idx_disc_iface_device
CREATE INDEX idx_disc_iface_device ON discovery_interfaces(device_id);

-- index: idx_disc_iface_mac
CREATE INDEX idx_disc_iface_mac ON discovery_interfaces(mac_address);

-- index: idx_disc_iface_ssid
CREATE INDEX idx_disc_iface_ssid ON discovery_interfaces(ssid);

-- index: idx_disc_iface_type
CREATE INDEX idx_disc_iface_type ON discovery_interfaces(interface_type);

-- index: idx_discovered_devices_client
CREATE INDEX idx_discovered_devices_client ON discovered_devices(client_id);

-- index: idx_discovery_history_client
CREATE INDEX idx_discovery_history_client ON discovery_history(client_id);

-- index: idx_discovery_interfaces_client
CREATE INDEX idx_discovery_interfaces_client ON discovery_interfaces(client_id);

-- index: idx_dns_interface
CREATE INDEX idx_dns_interface ON dns_results(interface_name);

-- index: idx_dns_results_client
CREATE INDEX idx_dns_results_client ON dns_results(client_id);

-- index: idx_dns_server
CREATE INDEX idx_dns_server ON dns_results(server);

-- index: idx_dns_timestamp
CREATE INDEX idx_dns_timestamp ON dns_results(timestamp);

-- index: idx_gateway_interface
CREATE INDEX idx_gateway_interface ON gateway_results(interface_name);

-- index: idx_gateway_results_client
CREATE INDEX idx_gateway_results_client ON gateway_results(client_id);

-- index: idx_gateway_timestamp
CREATE INDEX idx_gateway_timestamp ON gateway_results(timestamp);

-- index: idx_health_check_endpoint_time
CREATE INDEX idx_health_check_endpoint_time ON health_check_results(endpoint_name, recorded_at);

-- index: idx_health_check_recorded
CREATE INDEX idx_health_check_recorded ON health_check_results(recorded_at);

-- index: idx_health_check_type_time
CREATE INDEX idx_health_check_type_time ON health_check_results(check_type, recorded_at);

-- index: idx_health_daily_bucket
CREATE INDEX idx_health_daily_bucket ON health_check_rollups_daily(day_bucket);

-- index: idx_health_daily_unique
CREATE UNIQUE INDEX idx_health_daily_unique
				ON health_check_rollups_daily(check_type, endpoint_name, day_bucket);

-- index: idx_health_hourly_bucket
CREATE INDEX idx_health_hourly_bucket ON health_check_rollups_hourly(hour_bucket);

-- index: idx_health_hourly_unique
CREATE UNIQUE INDEX idx_health_hourly_unique
				ON health_check_rollups_hourly(check_type, endpoint_name, hour_bucket);

-- index: idx_listener_events_client_kind
CREATE INDEX idx_listener_events_client_kind ON listener_events(client_id, kind, observed_at);

-- index: idx_listener_events_observed_at
CREATE INDEX idx_listener_events_observed_at ON listener_events(observed_at);

-- index: idx_listener_events_source
CREATE INDEX idx_listener_events_source ON listener_events(source_addr, observed_at);

-- index: idx_logs_component
CREATE INDEX idx_logs_component ON logs(component);

-- index: idx_logs_layer
CREATE INDEX idx_logs_layer ON logs(layer);

-- index: idx_logs_level
CREATE INDEX idx_logs_level ON logs(level);

-- index: idx_logs_request_id
CREATE INDEX idx_logs_request_id ON logs(request_id);

-- index: idx_logs_timestamp
CREATE INDEX idx_logs_timestamp ON logs(timestamp);

-- index: idx_metrics_client
CREATE INDEX idx_metrics_client ON metrics(client_id);

-- index: idx_metrics_daily_bucket
CREATE INDEX idx_metrics_daily_bucket ON metrics_daily(day_bucket);

-- index: idx_metrics_daily_client
CREATE INDEX idx_metrics_daily_client ON metrics_daily(client_id);

-- index: idx_metrics_daily_target
CREATE INDEX idx_metrics_daily_target ON metrics_daily(target_kind, target_id, day_bucket);

-- index: idx_metrics_daily_type
CREATE INDEX idx_metrics_daily_type ON metrics_daily(metric_type, day_bucket);

-- index: idx_metrics_hourly_bucket
CREATE INDEX idx_metrics_hourly_bucket ON metrics_hourly(hour_bucket);

-- index: idx_metrics_hourly_client
CREATE INDEX idx_metrics_hourly_client ON metrics_hourly(client_id);

-- index: idx_metrics_hourly_target
CREATE INDEX idx_metrics_hourly_target ON metrics_hourly(target_kind, target_id, hour_bucket);

-- index: idx_metrics_hourly_type
CREATE INDEX idx_metrics_hourly_type ON metrics_hourly(metric_type, hour_bucket);

-- index: idx_metrics_interface
CREATE INDEX idx_metrics_interface ON metrics(interface_name);

-- index: idx_metrics_interface_type_time
CREATE INDEX idx_metrics_interface_type_time ON metrics(interface_name, metric_type, timestamp);

-- index: idx_metrics_target
CREATE INDEX idx_metrics_target ON metrics(target_kind, target_id);

-- index: idx_metrics_timestamp
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);

-- index: idx_metrics_type
CREATE INDEX idx_metrics_type ON metrics(metric_type);

-- index: idx_mib_oid_names_mib
CREATE INDEX idx_mib_oid_names_mib ON mib_oid_names(mib_name);

-- index: idx_mib_oid_names_oid
CREATE INDEX idx_mib_oid_names_oid ON mib_oid_names(oid);

-- index: idx_microburst_device
CREATE INDEX idx_microburst_device ON microburst_events(device_id);

-- index: idx_microburst_events_client
CREATE INDEX idx_microburst_events_client ON microburst_events(client_id);

-- index: idx_microburst_interface
CREATE INDEX idx_microburst_interface ON microburst_events(interface_name);

-- index: idx_microburst_timestamp
CREATE INDEX idx_microburst_timestamp ON microburst_events(timestamp);

-- index: idx_net_problems_detected
CREATE INDEX idx_net_problems_detected ON network_problems(detected_at);

-- index: idx_net_problems_device
CREATE INDEX idx_net_problems_device ON network_problems(device_id);

-- index: idx_net_problems_resolved
CREATE INDEX idx_net_problems_resolved ON network_problems(is_resolved);

-- index: idx_net_problems_severity
CREATE INDEX idx_net_problems_severity ON network_problems(severity);

-- index: idx_net_problems_type
CREATE INDEX idx_net_problems_type ON network_problems(problem_type);

-- index: idx_network_problems_client
CREATE INDEX idx_network_problems_client ON network_problems(client_id);

-- index: idx_oui_category
CREATE INDEX idx_oui_category ON oui_vendors(device_category);

-- index: idx_oui_vendor_name
CREATE INDEX idx_oui_vendor_name ON oui_vendors(vendor_name);

-- index: idx_pipeline_runs_started
CREATE INDEX idx_pipeline_runs_started ON pipeline_runs(started_at);

-- index: idx_pipeline_runs_status
CREATE INDEX idx_pipeline_runs_status ON pipeline_runs(status);

-- index: idx_polling_targets_client
CREATE INDEX idx_polling_targets_client ON polling_targets(client_id);

-- index: idx_polling_targets_enabled
CREATE INDEX idx_polling_targets_enabled ON polling_targets(enabled);

-- index: idx_polling_targets_ip
CREATE INDEX idx_polling_targets_ip ON polling_targets(ip_address);

-- index: idx_probe_results_client
CREATE INDEX idx_probe_results_client ON probe_results(client_id);

-- index: idx_probe_results_client_kind_ts
CREATE INDEX idx_probe_results_client_kind_ts ON probe_results(client_id, kind, timestamp);

-- index: idx_probe_results_kind
CREATE INDEX idx_probe_results_kind ON probe_results(kind);

-- index: idx_probe_results_probe
CREATE INDEX idx_probe_results_probe ON probe_results(probe_id);

-- index: idx_probe_results_timestamp
CREATE INDEX idx_probe_results_timestamp ON probe_results(timestamp);

-- index: idx_probe_rollups_daily_bucket
CREATE INDEX idx_probe_rollups_daily_bucket ON probe_rollups_daily(day_bucket);

-- index: idx_probe_rollups_daily_probe
CREATE INDEX idx_probe_rollups_daily_probe ON probe_rollups_daily(probe_id, day_bucket);

-- index: idx_probe_rollups_hourly_bucket
CREATE INDEX idx_probe_rollups_hourly_bucket ON probe_rollups_hourly(hour_bucket);

-- index: idx_probe_rollups_hourly_probe
CREATE INDEX idx_probe_rollups_hourly_probe ON probe_rollups_hourly(probe_id, hour_bucket);

-- index: idx_probes_client
CREATE INDEX idx_probes_client ON probes(client_id);

-- index: idx_probes_client_kind
CREATE INDEX idx_probes_client_kind ON probes(client_id, kind);

-- index: idx_probes_enabled
CREATE INDEX idx_probes_enabled ON probes(enabled);

-- index: idx_probes_kind
CREATE INDEX idx_probes_kind ON probes(kind);

-- index: idx_profiles_client
CREATE INDEX idx_profiles_client ON profiles(client_id);

-- index: idx_profiles_is_default
CREATE INDEX idx_profiles_is_default ON profiles(is_default);

-- index: idx_profiles_name
CREATE INDEX idx_profiles_name ON profiles(name);

-- index: idx_reports_created_at
CREATE INDEX idx_reports_created_at ON reports(created_at);

-- index: idx_reports_status
CREATE INDEX idx_reports_status ON reports(status);

-- index: idx_reports_type
CREATE INDEX idx_reports_type ON reports(type);

-- index: idx_scheduled_reports_enabled
CREATE INDEX idx_scheduled_reports_enabled ON scheduled_reports(enabled);

-- index: idx_scheduled_reports_next_run
CREATE INDEX idx_scheduled_reports_next_run ON scheduled_reports(next_run);

-- index: idx_snmp_observations_client_kind
CREATE INDEX idx_snmp_observations_client_kind ON snmp_observations(client_id, kind, observed_at);

-- index: idx_snmp_observations_observed_at
CREATE INDEX idx_snmp_observations_observed_at ON snmp_observations(observed_at);

-- index: idx_snmp_observations_target
CREATE INDEX idx_snmp_observations_target ON snmp_observations(target_id, observed_at);

-- index: idx_speedtest_interface
CREATE INDEX idx_speedtest_interface ON speedtest_results(interface_name);

-- index: idx_speedtest_results_client
CREATE INDEX idx_speedtest_results_client ON speedtest_results(client_id);

-- index: idx_speedtest_timestamp
CREATE INDEX idx_speedtest_timestamp ON speedtest_results(timestamp);

-- index: idx_survey_samples_client
CREATE INDEX idx_survey_samples_client ON survey_samples(client_id);

-- index: idx_survey_samples_coords
CREATE INDEX idx_survey_samples_coords ON survey_samples(x, y);

-- index: idx_survey_samples_survey
CREATE INDEX idx_survey_samples_survey ON survey_samples(survey_id);

-- index: idx_topology_interfaces_last_seen
CREATE INDEX idx_topology_interfaces_last_seen ON topology_interfaces(last_seen);

-- index: idx_topology_interfaces_node
CREATE INDEX idx_topology_interfaces_node ON topology_interfaces(node_id);

-- index: idx_topology_interfaces_oper
CREATE INDEX idx_topology_interfaces_oper ON topology_interfaces(if_oper_status);

-- index: idx_topology_links_client
CREATE INDEX idx_topology_links_client ON topology_links(client_id);

-- index: idx_topology_links_last_seen
CREATE INDEX idx_topology_links_last_seen ON topology_links(last_seen);

-- index: idx_topology_links_source
CREATE INDEX idx_topology_links_source ON topology_links(source_node_id);

-- index: idx_topology_links_target
CREATE INDEX idx_topology_links_target ON topology_links(target_node_id);

-- index: idx_topology_nodes_client
CREATE INDEX idx_topology_nodes_client ON topology_nodes(client_id);

-- index: idx_topology_nodes_identity
CREATE INDEX idx_topology_nodes_identity ON topology_nodes(identity_hash);

-- index: idx_topology_nodes_last_seen
CREATE INDEX idx_topology_nodes_last_seen ON topology_nodes(last_seen);

-- index: idx_topology_nodes_type
CREATE INDEX idx_topology_nodes_type ON topology_nodes(device_type);

-- index: idx_topology_target_nodes_node
CREATE INDEX idx_topology_target_nodes_node ON topology_target_nodes(node_id);

-- index: idx_users_active
CREATE INDEX idx_users_active               ON users(is_active);

-- index: idx_users_email
CREATE INDEX idx_users_email                ON users(email);

-- index: idx_users_provider_external_id
CREATE INDEX idx_users_provider_external_id ON users(auth_provider, external_id);

-- index: idx_users_username
CREATE INDEX idx_users_username             ON users(username);

-- index: idx_voip_calls_call_id
CREATE INDEX idx_voip_calls_call_id ON voip_calls(call_id);

-- index: idx_voip_calls_client
CREATE INDEX idx_voip_calls_client ON voip_calls(client_id);

-- index: idx_voip_calls_mos
CREATE INDEX idx_voip_calls_mos ON voip_calls(mos_score);

-- index: idx_voip_calls_started
CREATE INDEX idx_voip_calls_started ON voip_calls(started_at);

-- index: idx_webauthn_credential_id
CREATE UNIQUE INDEX idx_webauthn_credential_id
				ON webauthn_credentials(credential_id);

-- index: idx_webauthn_user
CREATE INDEX idx_webauthn_user ON webauthn_credentials(user_id);

-- index: idx_wifi_access_points_client
CREATE INDEX idx_wifi_access_points_client ON wifi_access_points(client_id);

-- index: idx_wifi_aps_band
CREATE INDEX idx_wifi_aps_band ON wifi_access_points(band);

-- index: idx_wifi_aps_bssid
CREATE INDEX idx_wifi_aps_bssid ON wifi_access_points(bssid);

-- index: idx_wifi_aps_channel
CREATE INDEX idx_wifi_aps_channel ON wifi_access_points(channel);

-- index: idx_wifi_aps_device
CREATE INDEX idx_wifi_aps_device ON wifi_access_points(device_id);

-- index: idx_wifi_aps_ssid
CREATE INDEX idx_wifi_aps_ssid ON wifi_access_points(ssid_id);

-- index: idx_wifi_assoc_ap
CREATE INDEX idx_wifi_assoc_ap ON wifi_associations(ap_bssid);

-- index: idx_wifi_assoc_client
CREATE INDEX idx_wifi_assoc_client ON wifi_associations(client_mac);

-- index: idx_wifi_assoc_status
CREATE INDEX idx_wifi_assoc_status ON wifi_associations(status_code);

-- index: idx_wifi_assoc_timestamp
CREATE INDEX idx_wifi_assoc_timestamp ON wifi_associations(timestamp);

-- index: idx_wifi_associations_client
CREATE INDEX idx_wifi_associations_client ON wifi_associations(client_id);

-- index: idx_wifi_clients_client
CREATE INDEX idx_wifi_clients_client ON wifi_clients(client_id);

-- index: idx_wifi_clients_last_seen
CREATE INDEX idx_wifi_clients_last_seen ON wifi_clients(last_seen);

-- index: idx_wifi_clients_mac
CREATE INDEX idx_wifi_clients_mac ON wifi_clients(mac_full);

-- index: idx_wifi_clients_oui
CREATE INDEX idx_wifi_clients_oui ON wifi_clients(vendor_oui);

-- index: idx_wifi_deauths_ap
CREATE INDEX idx_wifi_deauths_ap ON wifi_deauths(ap_bssid);

-- index: idx_wifi_deauths_client
CREATE INDEX idx_wifi_deauths_client ON wifi_deauths(client_mac);

-- index: idx_wifi_deauths_reason
CREATE INDEX idx_wifi_deauths_reason ON wifi_deauths(reason_code);

-- index: idx_wifi_deauths_timestamp
CREATE INDEX idx_wifi_deauths_timestamp ON wifi_deauths(timestamp);

-- index: idx_wifi_networks_auth
CREATE INDEX idx_wifi_networks_auth ON wifi_networks(authorization_status);

-- index: idx_wifi_networks_client
CREATE INDEX idx_wifi_networks_client ON wifi_networks(client_id);

-- index: idx_wifi_networks_ssid
CREATE INDEX idx_wifi_networks_ssid ON wifi_networks(ssid);

-- index: idx_wifi_roams_client
CREATE INDEX idx_wifi_roams_client ON wifi_roams(client_mac);

-- index: idx_wifi_roams_from
CREATE INDEX idx_wifi_roams_from ON wifi_roams(from_bssid);

-- index: idx_wifi_roams_started
CREATE INDEX idx_wifi_roams_started ON wifi_roams(started_at);

-- index: idx_wifi_roams_to
CREATE INDEX idx_wifi_roams_to ON wifi_roams(to_bssid);

-- index: idx_wifi_rogues_bssid
CREATE INDEX idx_wifi_rogues_bssid ON wifi_rogues(ap_bssid);

-- index: idx_wifi_rogues_client
CREATE INDEX idx_wifi_rogues_client ON wifi_rogues(client_id);

-- index: idx_wifi_rogues_detected
CREATE INDEX idx_wifi_rogues_detected ON wifi_rogues(detected_at);

-- index: idx_wifi_rogues_severity
CREATE INDEX idx_wifi_rogues_severity ON wifi_rogues(severity);

-- index: idx_wifi_rogues_status
CREATE INDEX idx_wifi_rogues_status ON wifi_rogues(status);

-- table: alert_rules
CREATE TABLE alert_rules (
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
			, window_seconds INTEGER NOT NULL DEFAULT 0, threshold_count INTEGER NOT NULL DEFAULT 1);

-- table: alert_suppressions
CREATE TABLE alert_suppressions (
					fingerprint TEXT PRIMARY KEY,
					rule_id TEXT NOT NULL,
					entity_key TEXT NOT NULL,
					suppress_until TEXT NOT NULL,
					created_at TEXT NOT NULL
				);

-- table: alerts
CREATE TABLE alerts (
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
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL
			);

-- table: api_tokens
CREATE TABLE "api_tokens" (
				id              TEXT PRIMARY KEY,
				owner_username  TEXT NOT NULL,
				name            TEXT NOT NULL,
				token_hash      TEXT NOT NULL UNIQUE,
				prefix          TEXT NOT NULL,
				created_at      TEXT NOT NULL,
				last_used_at    TEXT,
				revoked_at      TEXT, scope TEXT
			    CHECK (scope IS NULL OR scope IN ('admin','operator','viewer')),
				FOREIGN KEY (owner_username) REFERENCES users(username) ON DELETE CASCADE ON UPDATE CASCADE
			);

-- table: audit_log
CREATE TABLE audit_log (
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

-- table: bgp_sessions
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
			);

-- table: bluetooth_devices
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
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL
			);

-- table: bluetooth_scan_history
CREATE TABLE bluetooth_scan_history (
				id TEXT PRIMARY KEY,
				adapter_name TEXT,
				scan_type TEXT NOT NULL,
				devices_found INTEGER NOT NULL,
				classic_count INTEGER DEFAULT 0,
				ble_count INTEGER DEFAULT 0,
				scan_duration_ms INTEGER,
				scan_time TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: channel_utilization
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
			);

-- table: clients
CREATE TABLE clients (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				slug TEXT NOT NULL UNIQUE,
				branding_json TEXT,
				default_retention_overrides_json TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

-- table: device_credentials
CREATE TABLE device_credentials (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: device_interfaces
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
			);

-- table: device_ports
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
			);

-- table: device_vulnerabilities
CREATE TABLE device_vulnerabilities (
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

-- table: devices
CREATE TABLE devices (
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

-- table: discovered_devices
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
				is_online INTEGER DEFAULT 1,
				notes TEXT,
				tags TEXT,
				metadata_json TEXT,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: discovery_history
CREATE TABLE discovery_history (
				id TEXT PRIMARY KEY,
				device_id TEXT NOT NULL,
				event_type TEXT NOT NULL,
				event_data TEXT,
				recorded_at TEXT NOT NULL, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE
			);

-- table: discovery_interfaces
CREATE TABLE discovery_interfaces (
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
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE,
				UNIQUE(device_id, mac_address)
			);

-- table: dns_results
CREATE TABLE dns_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				server TEXT NOT NULL,
				hostname TEXT NOT NULL,
				response_time_ms REAL,
				resolved_ip TEXT,
				status TEXT NOT NULL,
				error_message TEXT,
				timestamp TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: gateway_results
CREATE TABLE gateway_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				gateway TEXT NOT NULL,
				latency_ms REAL,
				packet_loss REAL,
				reachable INTEGER,
				timestamp TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: health_check_results
CREATE TABLE health_check_results (
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

-- table: health_check_rollups_daily
CREATE TABLE health_check_rollups_daily (
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

-- table: health_check_rollups_hourly
CREATE TABLE health_check_rollups_hourly (
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

-- table: listener_events
CREATE TABLE listener_events (
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

-- table: logs
CREATE TABLE logs (
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

-- table: metrics
CREATE TABLE metrics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				metric_type TEXT NOT NULL,
				value REAL NOT NULL,
				unit TEXT,
				timestamp TEXT NOT NULL,
				metadata_json TEXT
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), target_kind TEXT NOT NULL DEFAULT 'interface', target_id TEXT NOT NULL DEFAULT '');

-- table: metrics_daily
CREATE TABLE metrics_daily (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				metric_type TEXT NOT NULL,
				interface_name TEXT NOT NULL,
				day_bucket TEXT NOT NULL,
				sample_count INTEGER NOT NULL,
				avg_value REAL,
				min_value REAL,
				max_value REAL,
				p95_value REAL, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), target_kind TEXT NOT NULL DEFAULT 'interface', target_id TEXT NOT NULL DEFAULT '',
				UNIQUE(metric_type, interface_name, day_bucket)
			);

-- table: metrics_hourly
CREATE TABLE metrics_hourly (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				metric_type TEXT NOT NULL,
				interface_name TEXT NOT NULL,
				hour_bucket TEXT NOT NULL,
				sample_count INTEGER NOT NULL,
				avg_value REAL,
				min_value REAL,
				max_value REAL,
				p95_value REAL, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), target_kind TEXT NOT NULL DEFAULT 'interface', target_id TEXT NOT NULL DEFAULT '',
				UNIQUE(metric_type, interface_name, hour_bucket)
			);

-- table: mib_oid_names
CREATE TABLE mib_oid_names (
				name TEXT PRIMARY KEY,           -- Human-readable name (e.g., "sysDescr")
				oid TEXT NOT NULL,               -- Numeric OID (e.g., "1.3.6.1.2.1.1.1")
				full_path TEXT,                  -- Full descriptive path (optional)
				mib_name TEXT,                   -- Source MIB name (e.g., "SNMPv2-MIB")
				created_at TEXT DEFAULT (datetime('now'))
			);

-- table: mib_sources
CREATE TABLE mib_sources (
				mib_name TEXT PRIMARY KEY,
				description TEXT,
				vendor TEXT,
				rfc_reference TEXT,
				loaded_at TEXT DEFAULT (datetime('now'))
			);

-- table: microburst_events
CREATE TABLE microburst_events (
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
			);

-- table: network_problems
CREATE TABLE network_problems (
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
				acknowledged_by TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE CASCADE,
				FOREIGN KEY (interface_id) REFERENCES discovery_interfaces(id) ON DELETE CASCADE
			);

-- table: oui_vendors
CREATE TABLE oui_vendors (
				oui TEXT PRIMARY KEY,
				vendor_name TEXT NOT NULL,
				vendor_short TEXT,
				is_private INTEGER DEFAULT 0,
				device_category TEXT,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			);

-- table: pipeline_runs
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
			);

-- table: polling_targets
CREATE TABLE polling_targets (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), collector_chain TEXT NOT NULL DEFAULT '["sys_info","if_table","lldp","arp","fdb"]');

-- table: probe_results
CREATE TABLE probe_results (
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

-- table: probe_rollups_daily
CREATE TABLE probe_rollups_daily (
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

-- table: probe_rollups_hourly
CREATE TABLE probe_rollups_hourly (
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

-- table: probes
CREATE TABLE probes (
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

-- table: profiles
CREATE TABLE profiles (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL UNIQUE,
				description TEXT,
				config_json TEXT NOT NULL,
				is_default INTEGER DEFAULT 0,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: reports
CREATE TABLE reports (
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

-- table: scheduled_reports
CREATE TABLE scheduled_reports (
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

-- table: settings
CREATE TABLE settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);

-- table: snmp_observations
CREATE TABLE snmp_observations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				target_id TEXT NOT NULL,
				kind TEXT NOT NULL,
				observed_at TEXT NOT NULL,
				payload_json TEXT NOT NULL,
				ingested_at TEXT NOT NULL
			);

-- table: speedtest_results
CREATE TABLE speedtest_results (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: survey_samples
CREATE TABLE survey_samples (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: topology_arp_bindings
CREATE TABLE topology_arp_bindings (
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

-- table: topology_interfaces
CREATE TABLE topology_interfaces (
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

-- table: topology_links
CREATE TABLE topology_links (
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
				evidence_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				FOREIGN KEY (source_node_id) REFERENCES topology_nodes(id) ON DELETE CASCADE,
				FOREIGN KEY (target_node_id) REFERENCES topology_nodes(id) ON DELETE CASCADE
			);

-- table: topology_nodes
CREATE TABLE topology_nodes (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: topology_target_nodes
CREATE TABLE topology_target_nodes (
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				target_id TEXT NOT NULL,
				node_id TEXT NOT NULL REFERENCES topology_nodes(id) ON DELETE CASCADE,
				last_seen TEXT NOT NULL,
				PRIMARY KEY (client_id, target_id)
			);

-- table: users
CREATE TABLE "users" (
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

-- table: voip_calls
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: webauthn_credentials
CREATE TABLE webauthn_credentials (
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

-- table: wifi_access_points
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
				is_authorized INTEGER DEFAULT 1,

				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), beacon_interval_tu INTEGER, rsn_cipher TEXT, rsn_akm TEXT, phy_capabilities TEXT, supports_11k INTEGER DEFAULT 0, supports_11v INTEGER DEFAULT 0, supports_11r INTEGER DEFAULT 0, bss_load_json TEXT, vendor_ies_json TEXT,

				FOREIGN KEY (device_id) REFERENCES discovered_devices(id) ON DELETE SET NULL,
				FOREIGN KEY (ssid_id) REFERENCES wifi_networks(id) ON DELETE SET NULL
			);

-- table: wifi_associations
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: wifi_clients
CREATE TABLE wifi_clients (
				id TEXT PRIMARY KEY,
				mac_full TEXT NOT NULL UNIQUE,
				vendor_oui TEXT,
				vendor_name TEXT,
				capabilities_json TEXT,
				pnl_json TEXT,
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				anonymized INTEGER NOT NULL DEFAULT 0
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: wifi_deauths
CREATE TABLE wifi_deauths (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				ap_bssid TEXT NOT NULL,
				client_mac TEXT NOT NULL,
				frame_type TEXT NOT NULL,
				reason_code INTEGER NOT NULL,
				reason_text TEXT,
				originator TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: wifi_networks
CREATE TABLE wifi_networks (
				id TEXT PRIMARY KEY,
				ssid TEXT NOT NULL,
				is_hidden INTEGER DEFAULT 0,
				security_type TEXT,
				authorization_status TEXT DEFAULT 'unknown',
				first_seen TEXT NOT NULL,
				last_seen TEXT NOT NULL,
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				UNIQUE(ssid, security_type)
			);

-- table: wifi_roams
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

-- table: wifi_rogues
CREATE TABLE wifi_rogues (
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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id));

