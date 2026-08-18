package database

// Profiles returns the profile repository.
func (db *DB) Profiles() *ProfileRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.profiles == nil {
		db.profiles = &ProfileRepository{db: db}
	}
	return db.profiles
}

// Metrics returns the metrics repository.
func (db *DB) Metrics() *MetricsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.metrics == nil {
		db.metrics = &MetricsRepository{db: db}
	}
	return db.metrics
}

// Devices returns the device repository.
func (db *DB) Devices() *DeviceRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.devices == nil {
		db.devices = &DeviceRepository{db: db}
	}
	return db.devices
}

// Alerts returns the alert repository.
func (db *DB) Alerts() *AlertRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.alerts == nil {
		db.alerts = &AlertRepository{db: db}
	}
	return db.alerts
}

// Settings returns the settings repository.
func (db *DB) Settings() *SettingsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.settings == nil {
		db.settings = &SettingsRepository{db: db}
	}
	return db.settings
}

// Logs returns the logs repository.
func (db *DB) Logs() *LogRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.logs == nil {
		db.logs = &LogRepository{db: db}
	}
	return db.logs
}

// Discovery returns the discovery repository for WiFi, problems, and OUI data.
func (db *DB) Discovery() *DiscoveryRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.discovery == nil {
		db.discovery = &DiscoveryRepository{db: db}
	}
	return db.discovery
}

// Clients returns the client repository (Stage A1.1 multi-tenancy
// foundation). The seeded default client is the fallback for
// single-tenant deployments.
func (db *DB) Clients() *ClientRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.clients == nil {
		db.clients = &ClientRepository{db: db}
	}
	return db.clients
}

// Probes returns the probe repository (Stage A1.2 unified probe
// engine foundation). Handles probes config + probe_results;
// future probe_rollups_hourly / probe_rollups_daily land in Stage A2.
func (db *DB) Probes() *ProbeRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.probes == nil {
		db.probes = &ProbeRepository{db: db}
	}
	return db.probes
}

// PollingTargets returns the polling-target repository (Stage A3.1).
// Targets carry the per-target SNMP credentials + collector chain
// that the Poller walks on each tick.
func (db *DB) PollingTargets() *PollingTargetRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.pollingTargets == nil {
		db.pollingTargets = &PollingTargetRepository{db: db}
	}
	return db.pollingTargets
}

// DeviceCredentials returns the device-credential repository. Rows hold
// versioned ciphertext; decryption happens at poll time in
// internal/polling/snmp, which owns the keyring seam.
func (db *DB) DeviceCredentials() *DeviceCredentialRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.deviceCredentials == nil {
		db.deviceCredentials = &DeviceCredentialRepository{db: db}
	}
	return db.deviceCredentials
}

// SNMPObservations returns the unified SNMP observations repository
// (Stage A3.5b). Every collector Publisher writes one row per poll;
// Stage A4 topology + the listener pipeline read kind-filtered rows
// downstream.
func (db *DB) SNMPObservations() *SNMPObservationsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.snmpObservations == nil {
		db.snmpObservations = &SNMPObservationsRepository{db: db}
	}
	return db.snmpObservations
}

// ListenerEvents returns the passive-ingress event repository
// (Stage A3.5e). Every Listener emits Events into this store via
// the listener.Sink seam; Stage A4 alert rules read from here.
func (db *DB) ListenerEvents() *ListenerEventsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.listenerEvents == nil {
		db.listenerEvents = &ListenerEventsRepository{db: db}
	}
	return db.listenerEvents
}

// Topology returns the topology repository (Stage A4). Reconcilers
// in internal/topology own writes; the operator UI + alert rules
// own reads.
func (db *DB) Topology() *TopologyRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.topology == nil {
		db.topology = &TopologyRepository{db: db}
	}
	return db.topology
}

// AlertRules returns the alert-rules repository (Stage A5.10).
// Operators CRUD rules here; the listener pipeline reads them on
// each scan and falls back to the hardcoded DefaultListenerRules
// when the table is empty.
func (db *DB) AlertRules() *AlertRulesRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.alertRules == nil {
		db.alertRules = &AlertRulesRepository{db: db}
	}
	return db.alertRules
}

// AlertSuppressions returns the repo for persisted suppression
// markers used by the alert pipelines (#1380). Persisting means a
// restart mid-incident doesn't re-fire alerts that were already
// emitted.
func (db *DB) AlertSuppressions() *AlertSuppressionsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.alertSuppressions == nil {
		db.alertSuppressions = &AlertSuppressionsRepository{db: db}
	}
	return db.alertSuppressions
}

// Anomalies returns the anomaly repository (ADR-0021, Anomaly Platform). It is
// the SQL system of record the engine's persistence Coordinator writes detected
// anomalies through and reads active instances back from on start.
func (db *DB) Anomalies() *AnomalyRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.anomalies == nil {
		db.anomalies = &AnomalyRepository{db: db}
	}
	return db.anomalies
}
