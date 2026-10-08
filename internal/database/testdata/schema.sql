-- index: idx_alert_deliveries_status
CREATE INDEX idx_alert_deliveries_status ON alert_deliveries(status);

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

-- index: idx_alerts_open_rule
CREATE INDEX idx_alerts_open_rule
    ON alerts(rule) WHERE acknowledged = 0 AND resolved = 0;

-- index: idx_alerts_resolved
CREATE INDEX idx_alerts_resolved ON alerts(resolved);

-- index: idx_alerts_root_cause
CREATE INDEX idx_alerts_root_cause ON alerts(root_cause_id);

-- index: idx_alerts_rule_source
CREATE INDEX idx_alerts_rule_source ON alerts(rule, source, created_at);

-- index: idx_alerts_severity
CREATE INDEX idx_alerts_severity ON alerts(severity);

-- index: idx_alerts_type
CREATE INDEX idx_alerts_type ON alerts(type);

-- index: idx_anomalies_active
CREATE INDEX idx_anomalies_active ON anomalies(id) WHERE is_resolved = 0;

-- index: idx_anomalies_last_seen
CREATE INDEX idx_anomalies_last_seen ON anomalies(last_seen);

-- index: idx_anomalies_severity
CREATE INDEX idx_anomalies_severity ON anomalies(severity);

-- index: idx_anomalies_source
CREATE INDEX idx_anomalies_source ON anomalies(source);

-- index: idx_anomalies_subject
CREATE INDEX idx_anomalies_subject ON anomalies(subject_kind, subject_id);

-- index: idx_anomaly_rollups_daily_bucket
CREATE INDEX idx_anomaly_rollups_daily_bucket ON anomaly_rollups_daily(day_bucket);

-- index: idx_anomaly_rollups_daily_subject
CREATE INDEX idx_anomaly_rollups_daily_subject ON anomaly_rollups_daily(subject_kind, subject_id);

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

-- index: idx_clients_slug
CREATE INDEX idx_clients_slug ON clients(slug);

-- index: idx_device_credentials_client
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);

-- index: idx_device_credentials_name
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

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

-- index: idx_flow_applications_daily_bucket
CREATE INDEX idx_flow_applications_daily_bucket ON flow_applications_daily(day_bucket);

-- index: idx_flow_applications_hourly_bucket
CREATE INDEX idx_flow_applications_hourly_bucket ON flow_applications_hourly(hour_bucket);

-- index: idx_flow_conversations_daily_bucket
CREATE INDEX idx_flow_conversations_daily_bucket ON flow_conversations_daily(day_bucket);

-- index: idx_flow_conversations_hourly_bucket
CREATE INDEX idx_flow_conversations_hourly_bucket ON flow_conversations_hourly(hour_bucket);

-- index: idx_flow_records_end
CREATE INDEX idx_flow_records_end ON flow_records(flow_end);

-- index: idx_flow_records_exporter_end
CREATE INDEX idx_flow_records_exporter_end ON flow_records(exporter, flow_end);

-- index: idx_job_idempotency_job
CREATE INDEX idx_job_idempotency_job ON job_idempotency(job_id);

-- index: idx_jobs_completed
CREATE INDEX idx_jobs_completed ON jobs(completed_at);

-- index: idx_jobs_created
CREATE INDEX idx_jobs_created ON jobs(created_at);

-- index: idx_jobs_kind
CREATE INDEX idx_jobs_kind ON jobs(kind);

-- index: idx_jobs_state
CREATE INDEX idx_jobs_state ON jobs(state);

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

-- index: idx_microburst_events_client
CREATE INDEX idx_microburst_events_client ON microburst_events(client_id);

-- index: idx_microburst_interface
CREATE INDEX idx_microburst_interface ON microburst_events(interface_name);

-- index: idx_microburst_timestamp
CREATE INDEX idx_microburst_timestamp ON microburst_events(timestamp);

-- index: idx_outbox_published
CREATE INDEX idx_outbox_published ON outbox(published_at);

-- index: idx_outbox_unpublished
CREATE INDEX idx_outbox_unpublished ON outbox(id) WHERE published_at IS NULL;

-- index: idx_polling_targets_client
CREATE INDEX idx_polling_targets_client  ON polling_targets(client_id);

-- index: idx_polling_targets_enabled
CREATE INDEX idx_polling_targets_enabled ON polling_targets(enabled);

-- index: idx_polling_targets_ip
CREATE INDEX idx_polling_targets_ip      ON polling_targets(ip_address);

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

-- index: idx_users_client
CREATE INDEX idx_users_client               ON users(client_id);

-- index: idx_users_email
CREATE INDEX idx_users_email                ON users(email);

-- index: idx_users_provider_external_id
CREATE INDEX idx_users_provider_external_id ON users(auth_provider, external_id);

-- index: idx_users_username
CREATE INDEX idx_users_username             ON users(username);

-- index: idx_voip_streams_client
CREATE INDEX idx_voip_streams_client ON voip_streams(client_id);

-- index: idx_voip_streams_mos
CREATE INDEX idx_voip_streams_mos ON voip_streams(mos);

-- index: idx_voip_streams_started
CREATE INDEX idx_voip_streams_started ON voip_streams(started_at);

-- index: idx_vuln_status_history_vuln
CREATE INDEX idx_vuln_status_history_vuln
	ON vulnerability_status_history(vulnerability_id, id);

-- index: idx_webauthn_credential_id
CREATE UNIQUE INDEX idx_webauthn_credential_id
				ON webauthn_credentials(credential_id);

-- index: idx_webauthn_user
CREATE INDEX idx_webauthn_user ON webauthn_credentials(user_id);

-- table: alert_deliveries
CREATE TABLE alert_deliveries (
    alert_id     INTEGER NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    channel      TEXT NOT NULL,
    status       TEXT NOT NULL,
    attempted_at TEXT,
    error        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (alert_id, channel)
) STRICT, WITHOUT ROWID;

-- table: alert_rules
CREATE TABLE alert_rules (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
				match_kind TEXT,
				match_severity TEXT,
				match_payload_contains TEXT,
				alert_type TEXT NOT NULL,
				alert_severity TEXT NOT NULL,
				alert_title TEXT NOT NULL,
				alert_message TEXT NOT NULL,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			, window_seconds INTEGER NOT NULL DEFAULT 0, threshold_count INTEGER NOT NULL DEFAULT 1) STRICT;

-- table: alert_suppressions
CREATE TABLE alert_suppressions (
					fingerprint TEXT PRIMARY KEY,
					rule_id TEXT NOT NULL,
					entity_key TEXT NOT NULL,
					suppress_until TEXT NOT NULL,
					created_at TEXT NOT NULL
				) STRICT;

-- table: alerts
CREATE TABLE alerts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				type TEXT NOT NULL,
				severity TEXT NOT NULL,
				title TEXT NOT NULL,
				message TEXT NOT NULL,
				source TEXT,
				device_id TEXT,
				acknowledged INTEGER DEFAULT 0 CHECK (acknowledged IN (0,1)),
				acknowledged_by TEXT,
				acknowledged_at TEXT,
				resolved INTEGER DEFAULT 0 CHECK (resolved IN (0,1)),
				resolved_at TEXT,
				created_at TEXT NOT NULL,
				metadata_json TEXT, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), rule TEXT NOT NULL DEFAULT '', root_cause_id INTEGER REFERENCES alerts(id) ON DELETE SET NULL, escalation_stage INTEGER NOT NULL DEFAULT 0, escalated_at TEXT,
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL
			) STRICT;

-- table: anomalies
CREATE TABLE anomalies (
	id              TEXT    NOT NULL PRIMARY KEY,          -- defKey|subjectKind|subjectId
	def_key         TEXT    NOT NULL CHECK (def_key <> ''),
	source          TEXT    NOT NULL CHECK (source <> ''), -- wifi|wired|snmp|bluetooth|health|security|autotest
	category        TEXT    NOT NULL CHECK (category <> ''),
	severity        TEXT    NOT NULL CHECK (severity <> ''),
	subject_kind    TEXT    NOT NULL CHECK (subject_kind <> ''),
	subject_id      TEXT    NOT NULL,
	title           TEXT    NOT NULL,
	description     TEXT    NOT NULL,
	recommendation  TEXT    NOT NULL,
	evidence        TEXT,                                  -- JSON object (map[string]string)
	standards       TEXT,                                  -- JSON array (IEEE/RFC cites)
	count           INTEGER NOT NULL CHECK (count >= 0),
	first_seen      TEXT    NOT NULL,                       -- RFC3339
	last_seen       TEXT    NOT NULL,                       -- RFC3339
	resolved_at     TEXT,                                   -- RFC3339, NULL while active
	is_resolved     INTEGER NOT NULL DEFAULT 0 CHECK (is_resolved IN (0, 1)),
	acknowledged_by TEXT,
	acknowledged_at TEXT
) STRICT;

-- table: anomaly_rollups_daily
CREATE TABLE anomaly_rollups_daily (
	day_bucket       TEXT    NOT NULL,                      -- YYYY-MM-DD UTC (retention.dayFormat)
	def_key          TEXT    NOT NULL CHECK (def_key <> ''),
	source           TEXT    NOT NULL CHECK (source <> ''), -- wifi|wired|snmp|bluetooth|health|security|autotest
	category         TEXT    NOT NULL CHECK (category <> ''),
	subject_kind     TEXT    NOT NULL CHECK (subject_kind <> ''),
	subject_id       TEXT    NOT NULL,
	max_severity     TEXT    NOT NULL CHECK (max_severity <> ''), -- highest severity held as of the census
	count_cumulative INTEGER NOT NULL CHECK (count_cumulative >= 0),
	first_seen       TEXT    NOT NULL,                       -- RFC3339, carried from the live row
	last_seen        TEXT    NOT NULL,                       -- RFC3339, carried from the live row
	is_resolved      INTEGER NOT NULL DEFAULT 0 CHECK (is_resolved IN (0, 1)),
	resolved_at      TEXT,                                   -- RFC3339, NULL while active
	PRIMARY KEY (day_bucket, def_key, subject_kind, subject_id) -- idempotent re-census
) STRICT;

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
			) STRICT;

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
			) STRICT;

-- table: clients
CREATE TABLE clients (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				slug TEXT NOT NULL UNIQUE,
				branding_json TEXT,
				default_retention_overrides_json TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			) STRICT;

-- table: device_credentials
CREATE TABLE "device_credentials" (
				id                 TEXT NOT NULL,
				client_id          TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				name               TEXT NOT NULL,
				kind               TEXT NOT NULL CHECK (kind IN ('v2c','v3')),
				security_level     TEXT CHECK (security_level IS NULL OR security_level IN ('noAuthNoPriv','authNoPriv','authPriv')),
				snmp_community_enc BLOB,
				snmp_v3_user       TEXT,
				snmp_v3_auth_enc   BLOB,
				snmp_v3_priv_enc   BLOB,
				snmp_v3_auth_proto TEXT CHECK (snmp_v3_auth_proto IS NULL OR snmp_v3_auth_proto IN ('SHA','SHA224','SHA256','SHA384','SHA512')),
				snmp_v3_priv_proto TEXT CHECK (snmp_v3_priv_proto IS NULL OR snmp_v3_priv_proto IN ('DES','AES','AES192','AES256')),
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				PRIMARY KEY (id),
				UNIQUE (client_id, id),

				-- Every stored secret is versioned ciphertext. The legacy
				-- unversioned "enc:…" fails this too, on purpose: the key that
				-- produced it is unknown, so it cannot be rotated.
				CHECK (snmp_community_enc IS NULL OR CAST(snmp_community_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_auth_enc   IS NULL OR CAST(snmp_v3_auth_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_priv_enc   IS NULL OR CAST(snmp_v3_priv_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),

				-- v2c is a community string and nothing else.
				CHECK (kind <> 'v2c' OR (
					snmp_community_enc IS NOT NULL
					AND security_level IS NULL
					AND snmp_v3_user     IS NULL
					AND snmp_v3_auth_enc IS NULL
					AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL
					AND snmp_v3_priv_proto IS NULL
				)),

				-- v3 is a user plus a security level, and no community.
				CHECK (kind <> 'v3' OR (
					snmp_community_enc IS NULL
					AND snmp_v3_user IS NOT NULL AND snmp_v3_user <> ''
					AND security_level IS NOT NULL
				)),

				-- The security level and the secrets present cannot disagree,
				-- which is what makes "privacy without authentication"
				-- unrepresentable rather than merely discouraged.
				CHECK (security_level <> 'noAuthNoPriv' OR (
					snmp_v3_auth_enc IS NULL AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (security_level <> 'authNoPriv' OR (
					snmp_v3_auth_enc IS NOT NULL AND snmp_v3_auth_proto IS NOT NULL
					AND snmp_v3_priv_enc IS NULL AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (security_level <> 'authPriv' OR (
					snmp_v3_auth_enc IS NOT NULL AND snmp_v3_auth_proto IS NOT NULL
					AND snmp_v3_priv_enc IS NOT NULL AND snmp_v3_priv_proto IS NOT NULL
				))
			) STRICT;

-- table: device_vulnerabilities
CREATE TABLE "device_vulnerabilities" (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				device_id TEXT NOT NULL,
				cve_id TEXT NOT NULL,
				severity TEXT,
				cvss_score REAL,
				cvss_vector TEXT,
				affected_component TEXT,
				affected_version TEXT,
				fix_available INTEGER DEFAULT 0 CHECK (fix_available IN (0,1)),
				status TEXT NOT NULL DEFAULT 'new'
					CHECK (status IN ('new','acknowledged','ignored','resolved')),
				detected_at TEXT NOT NULL,
				resolved_at TEXT,
				notes TEXT,
				description TEXT,
				FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
			) STRICT;

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
				is_active INTEGER DEFAULT 1 CHECK (is_active IN (0,1)),
				ports_json TEXT,
				metadata_json TEXT
			) STRICT;

-- table: flow_applications_daily
CREATE TABLE flow_applications_daily (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	application TEXT NOT NULL,
	day_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, application, day_bucket)
) STRICT;

-- table: flow_applications_hourly
CREATE TABLE flow_applications_hourly (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	application TEXT NOT NULL,
	hour_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, application, hour_bucket)
) STRICT;

-- table: flow_conversations_daily
CREATE TABLE flow_conversations_daily (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	protocol INTEGER NOT NULL,
	day_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, src_addr, dst_addr, protocol, day_bucket)
) STRICT;

-- table: flow_conversations_hourly
CREATE TABLE flow_conversations_hourly (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	protocol INTEGER NOT NULL,
	hour_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, src_addr, dst_addr, protocol, hour_bucket)
) STRICT;

-- table: flow_records
CREATE TABLE "flow_records" (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	exporter TEXT NOT NULL,
	format TEXT NOT NULL CHECK (format IN ('netflow5', 'netflow9', 'ipfix', 'sflow5')),
	observation_domain INTEGER NOT NULL,
	flow_start TEXT NOT NULL,
	flow_end TEXT NOT NULL,
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	src_port INTEGER NOT NULL,
	dst_port INTEGER NOT NULL,
	protocol INTEGER NOT NULL,
	tcp_flags INTEGER NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	input_if INTEGER NOT NULL,
	output_if INTEGER NOT NULL,
	received_at TEXT NOT NULL
, application TEXT NOT NULL DEFAULT 'unknown') STRICT;

-- table: job_idempotency
CREATE TABLE job_idempotency (
	key          TEXT NOT NULL PRIMARY KEY,
	request_hash TEXT NOT NULL,
	job_id       TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
	created_at   TEXT NOT NULL
) STRICT;

-- table: jobs
CREATE TABLE jobs (
	id           TEXT NOT NULL PRIMARY KEY,
	kind         TEXT NOT NULL,
	state        TEXT NOT NULL DEFAULT 'queued'
	             CHECK (state IN ('queued','running','succeeded','failed','cancelled')),
	progress     REAL NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 1),
	result_json  TEXT,
	error        TEXT,
	created_at   TEXT NOT NULL,
	updated_at   TEXT NOT NULL,
	completed_at TEXT
) STRICT;

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
			) STRICT;

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
			) STRICT;

-- table: metrics
CREATE TABLE metrics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				metric_type TEXT NOT NULL,
				value REAL NOT NULL,
				unit TEXT,
				timestamp TEXT NOT NULL,
				metadata_json TEXT
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), target_kind TEXT NOT NULL DEFAULT 'interface', target_id TEXT NOT NULL DEFAULT '') STRICT;

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
			) STRICT;

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
			) STRICT;

-- table: mib_oid_names
CREATE TABLE mib_oid_names (
				name TEXT PRIMARY KEY,           -- Human-readable name (e.g., "sysDescr")
				oid TEXT NOT NULL,               -- Numeric OID (e.g., "1.3.6.1.2.1.1.1")
				full_path TEXT,                  -- Full descriptive path (optional)
				mib_name TEXT,                   -- Source MIB name (e.g., "SNMPv2-MIB")
				created_at TEXT DEFAULT (datetime('now'))
			) STRICT;

-- table: microburst_events
CREATE TABLE "microburst_events" (
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

-- table: outbox
CREATE TABLE outbox (
	-- AUTOINCREMENT so ids are monotonic and never reused after retention
	-- deletes — the dedup key (Message.ID) must stay stable and collision-free.
	id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	topic        TEXT NOT NULL CHECK (topic <> ''),
	payload      BLOB NOT NULL,
	created_at   TEXT NOT NULL,
	published_at TEXT
) STRICT;

-- table: polling_targets
CREATE TABLE "polling_targets" (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				ip_address TEXT NOT NULL,
				snmp_version TEXT NOT NULL DEFAULT 'v2c',
				credentials_id TEXT,
				poll_interval_seconds INTEGER NOT NULL DEFAULT 300,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
				last_polled_at TEXT,
				last_status TEXT,
				last_error TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				collector_chain TEXT NOT NULL DEFAULT '["sys_info","if_table","lldp","arp","fdb"]',
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;

-- table: probe_results
CREATE TABLE probe_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				probe_id TEXT NOT NULL,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				timestamp TEXT NOT NULL,
				success INTEGER NOT NULL CHECK (success IN (0,1)),
				latency_ms REAL,
				error TEXT,
				metadata_json TEXT,
				FOREIGN KEY (probe_id) REFERENCES probes(id) ON DELETE CASCADE
			) STRICT;

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
			) STRICT;

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
			) STRICT;

-- table: probes
CREATE TABLE probes (
				id TEXT PRIMARY KEY,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				kind TEXT NOT NULL,
				display_name TEXT NOT NULL,
				target TEXT NOT NULL,
				params_json TEXT,
				interval_seconds INTEGER NOT NULL DEFAULT 60,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
				warning_json TEXT,
				critical_json TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			) STRICT;

-- table: profiles
CREATE TABLE profiles (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL UNIQUE,
				description TEXT,
				config_json TEXT NOT NULL,
				is_default INTEGER DEFAULT 0 CHECK (is_default IN (0,1)),
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), row_version INTEGER NOT NULL DEFAULT 1) STRICT;

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
			) STRICT;

-- table: scheduled_reports
CREATE TABLE scheduled_reports (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				template TEXT NOT NULL,
				format TEXT NOT NULL,
				schedule_json TEXT NOT NULL,
				parameters_json TEXT,
				recipients_json TEXT,
				enabled INTEGER DEFAULT 1 CHECK (enabled IN (0,1)),
				last_run TEXT,
				next_run TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			) STRICT;

-- table: settings
CREATE TABLE settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at TEXT NOT NULL
			) STRICT;

-- table: snmp_observations
CREATE TABLE snmp_observations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				target_id TEXT NOT NULL,
				kind TEXT NOT NULL,
				observed_at TEXT NOT NULL,
				payload_json TEXT NOT NULL,
				ingested_at TEXT NOT NULL
			) STRICT;

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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;

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
			) STRICT;

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
			) STRICT;

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
			) STRICT;

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
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;

-- table: topology_target_nodes
CREATE TABLE topology_target_nodes (
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				target_id TEXT NOT NULL,
				node_id TEXT NOT NULL REFERENCES topology_nodes(id) ON DELETE CASCADE,
				last_seen TEXT NOT NULL,
				PRIMARY KEY (client_id, target_id)
			) STRICT;

-- table: user_dashboards
CREATE TABLE user_dashboards (
	user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	widgets TEXT NOT NULL CHECK (json_valid(widgets) AND json_type(widgets) = 'array'),
	updated_at TEXT NOT NULL
) STRICT;

-- table: users
CREATE TABLE "users" (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				username        TEXT    NOT NULL UNIQUE CHECK (LENGTH(username) >= 3 AND LENGTH(username) <= 64),
				password_hash   TEXT    NOT NULL,
				role            TEXT    NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
				is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
				last_login      TEXT,
				failed_attempts INTEGER NOT NULL DEFAULT 0,
				locked_until    TEXT,
				token_version   INTEGER NOT NULL DEFAULT 1,
				totp_secret     TEXT,
				totp_enabled    INTEGER NOT NULL DEFAULT 0 CHECK (totp_enabled IN (0,1)),
				auth_provider   TEXT    NOT NULL DEFAULT 'local' CHECK (auth_provider IN ('local','google','microsoft','github')),
				external_id     TEXT,
				email           TEXT,
				display_name    TEXT,
				client_id       TEXT    NOT NULL DEFAULT 'default' REFERENCES clients(id),
				created_at      TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL,
				UNIQUE (auth_provider, external_id)
			) STRICT;

-- table: voip_streams
CREATE TABLE voip_streams (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	interface_name TEXT NOT NULL,
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	ssrc INTEGER NOT NULL,
	codec TEXT NOT NULL,
	started_at TEXT NOT NULL,
	ended_at TEXT NOT NULL,
	packets_expected INTEGER NOT NULL,
	packets_received INTEGER NOT NULL,
	loss_pct REAL NOT NULL,
	jitter_ms REAL NOT NULL,
	max_jitter_ms REAL NOT NULL,
	delay_ms REAL NOT NULL,
	r_factor REAL NOT NULL,
	mos REAL NOT NULL
) STRICT;

-- table: vulnerability_status_history
CREATE TABLE vulnerability_status_history (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				vulnerability_id INTEGER NOT NULL
					REFERENCES device_vulnerabilities(id) ON DELETE CASCADE,
				from_status TEXT NOT NULL
					CHECK (from_status IN ('new','acknowledged','ignored','resolved')),
				to_status TEXT NOT NULL
					CHECK (to_status IN ('new','acknowledged','ignored','resolved')),
				actor TEXT,
				reason TEXT,
				changed_at TEXT NOT NULL
			) STRICT;

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
			) STRICT;

