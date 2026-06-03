package database

// seed.go holds first-run data seeding that runs after migrations during Open.
// Moved out of the (now goose-based) migration engine; the logic is unchanged.

import (
	"context"
	"fmt"
	"time"
)

// seedDefaultProfile creates a default profile if no profiles exist.
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
