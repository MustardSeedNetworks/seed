package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// VulnStatusResolved marks a finding a later scan pass of its device no longer
// reported. Findings are stored as 'new' (the column default) until then.
const VulnStatusResolved = "resolved"

// VulnerabilityFinding is one CVE a scan pass found on a device.
type VulnerabilityFinding struct {
	CVEID             string
	Severity          string
	CVSSScore         float64
	Description       string
	AffectedComponent string
	AffectedVersion   string
}

// VulnerabilityRepository persists vulnerability scan passes into
// device_vulnerabilities, the table report generation and export read.
type VulnerabilityRepository struct {
	db *DB
}

// RecordScan stores one completed scan pass of a device. Each finding is
// upserted on (device_id, cve_id): a re-reported finding keeps its first
// detection time and its status, and a resolved one reopens. Open findings the
// pass no longer reports are resolved at scannedAt.
func (r *VulnerabilityRepository) RecordScan(
	ctx context.Context, deviceID string, findings []VulnerabilityFinding, scannedAt time.Time,
) error {
	at := scannedAt.UTC().Format(time.RFC3339)
	cveIDs := make([]string, 0, len(findings))
	for i := range findings {
		cveIDs = append(cveIDs, findings[i].CVEID)
	}
	reported, err := json.Marshal(cveIDs)
	if err != nil {
		return fmt.Errorf("encoding reported CVE ids: %w", err)
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		for i := range findings {
			f := &findings[i]
			if _, execErr := tx.ExecContext(ctx, `
				INSERT INTO device_vulnerabilities
					(device_id, cve_id, severity, cvss_score, description,
					 affected_component, affected_version, detected_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(device_id, cve_id) DO UPDATE SET
					severity = excluded.severity,
					cvss_score = excluded.cvss_score,
					description = excluded.description,
					affected_component = excluded.affected_component,
					affected_version = excluded.affected_version,
					status = CASE WHEN status = ? THEN 'new' ELSE status END,
					resolved_at = NULL
			`, deviceID, f.CVEID, f.Severity, f.CVSSScore, toNullString(f.Description),
				toNullString(f.AffectedComponent), toNullString(f.AffectedVersion), at,
				VulnStatusResolved,
			); execErr != nil {
				return fmt.Errorf("recording %s on device %s: %w", f.CVEID, deviceID, execErr)
			}
		}
		if _, execErr := tx.ExecContext(ctx, `
			UPDATE device_vulnerabilities SET status = ?, resolved_at = ?
			WHERE device_id = ? AND status IS NOT ?
			  AND cve_id NOT IN (SELECT value FROM json_each(?))
		`, VulnStatusResolved, at, deviceID, VulnStatusResolved, string(reported),
		); execErr != nil {
			return fmt.Errorf("resolving findings no longer reported on device %s: %w", deviceID, execErr)
		}
		return nil
	})
}
