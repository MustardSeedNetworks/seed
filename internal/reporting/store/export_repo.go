package store

import (
	"context"
	"fmt"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/reporting"
)

// ExportRepo implements reporting.ExportRepo over the devices and
// device_vulnerabilities tables. The SQL and scanning were lifted verbatim from
// the reporting package (services_export.go) when reporting was made I/O-free —
// Phase 3 slice 1b-v.
type ExportRepo struct {
	db *database.DB
}

// NewExportRepo constructs an ExportRepo backed by db.
func NewExportRepo(db *database.DB) *ExportRepo {
	return &ExportRepo{db: db}
}

// Compile-time assertion that the adapter satisfies reporting's port.
var _ reporting.ExportRepo = (*ExportRepo)(nil)

// ExportDevices returns all device rows, newest-seen first.
func (r *ExportRepo) ExportDevices(ctx context.Context) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ip_address, mac_address, hostname, vendor, device_type, first_seen, last_seen
		FROM devices ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying devices: %w", err)
	}
	defer rows.Close()

	var devices []map[string]any
	for rows.Next() {
		var id, ip, mac, hostname, vendor, deviceType, firstSeen, lastSeen string
		if scanErr := rows.Scan(
			&id,
			&ip,
			&mac,
			&hostname,
			&vendor,
			&deviceType,
			&firstSeen,
			&lastSeen,
		); scanErr != nil {
			continue
		}
		devices = append(devices, map[string]any{
			"id":          id,
			"ip_address":  ip,
			"mac_address": mac,
			"hostname":    hostname,
			"vendor":      vendor,
			"device_type": deviceType,
			"first_seen":  firstSeen,
			"last_seen":   lastSeen,
		})
	}

	return devices, nil
}

// ExportVulnerabilities returns all vulnerability rows joined to device IPs.
func (r *ExportRepo) ExportVulnerabilities(ctx context.Context) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dv.id, dv.device_id, dv.cve_id, dv.severity, dv.description, dv.discovered_at, d.ip_address
		FROM device_vulnerabilities dv
		LEFT JOIN devices d ON dv.device_id = d.id
		ORDER BY dv.severity DESC, dv.discovered_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying vulnerabilities: %w", err)
	}
	defer rows.Close()

	var vulns []map[string]any
	for rows.Next() {
		var id int
		var deviceID, cveID, severity, desc, discoveredAt string
		var ipAddr *string
		if scanErr := rows.Scan(&id, &deviceID, &cveID, &severity, &desc, &discoveredAt, &ipAddr); scanErr != nil {
			continue
		}
		vulns = append(vulns, map[string]any{
			"id":            id,
			"device_id":     deviceID,
			"cve_id":        cveID,
			"severity":      severity,
			"description":   desc,
			"discovered_at": discoveredAt,
			"device_ip":     ipAddr,
		})
	}

	return vulns, nil
}
