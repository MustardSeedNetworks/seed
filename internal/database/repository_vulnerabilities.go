package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/security/vulntriage"
)

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
// device_vulnerabilities, the table report generation and export read. It
// implements vulntriage.Store.
type VulnerabilityRepository struct {
	db *DB
}

// RecordScan stores one completed scan pass of a device. Each finding is
// upserted on (device_id, cve_id): a re-reported finding keeps its first
// detection time and its status, and a resolved one reopens. Findings the pass
// no longer reports are resolved at scannedAt, whatever their triage state.
// Each reopen and resolve is written to the history with no actor.
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
		if _, execErr := tx.ExecContext(ctx, `
			INSERT INTO vulnerability_status_history
				(vulnerability_id, from_status, to_status, changed_at)
			SELECT id, status, ?, ? FROM device_vulnerabilities
			WHERE device_id = ? AND status = ?
			  AND cve_id IN (SELECT value FROM json_each(?))
		`, vulntriage.StatusNew, at, deviceID, vulntriage.StatusResolved, string(reported),
		); execErr != nil {
			return fmt.Errorf("recording reopened findings on device %s: %w", deviceID, execErr)
		}
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
					status = CASE WHEN status = ? THEN ? ELSE status END,
					resolved_at = NULL
			`, deviceID, f.CVEID, f.Severity, f.CVSSScore, toNullString(f.Description),
				toNullString(f.AffectedComponent), toNullString(f.AffectedVersion), at,
				vulntriage.StatusResolved, vulntriage.StatusNew,
			); execErr != nil {
				return fmt.Errorf("recording %s on device %s: %w", f.CVEID, deviceID, execErr)
			}
		}
		if _, execErr := tx.ExecContext(ctx, `
			INSERT INTO vulnerability_status_history
				(vulnerability_id, from_status, to_status, changed_at)
			SELECT id, status, ?, ? FROM device_vulnerabilities
			WHERE device_id = ? AND status != ?
			  AND cve_id NOT IN (SELECT value FROM json_each(?))
		`, vulntriage.StatusResolved, at, deviceID, vulntriage.StatusResolved, string(reported),
		); execErr != nil {
			return fmt.Errorf("recording resolved findings on device %s: %w", deviceID, execErr)
		}
		if _, execErr := tx.ExecContext(ctx, `
			UPDATE device_vulnerabilities SET status = ?, resolved_at = ?
			WHERE device_id = ? AND status != ?
			  AND cve_id NOT IN (SELECT value FROM json_each(?))
		`, vulntriage.StatusResolved, at, deviceID, vulntriage.StatusResolved, string(reported),
		); execErr != nil {
			return fmt.Errorf("resolving findings no longer reported on device %s: %w", deviceID, execErr)
		}
		return nil
	})
}

// ListFindings returns persisted findings, most severe first.
func (r *VulnerabilityRepository) ListFindings(
	ctx context.Context, opts vulntriage.ListOptions,
) ([]vulntriage.Finding, error) {
	query := `
		SELECT v.id, v.device_id, d.ip_address, COALESCE(d.hostname, ''), v.cve_id,
		       COALESCE(v.severity, ''), COALESCE(v.cvss_score, 0),
		       COALESCE(v.description, ''), COALESCE(v.affected_component, ''),
		       COALESCE(v.affected_version, ''), v.status, v.detected_at, v.resolved_at
		FROM device_vulnerabilities v
		JOIN devices d ON d.id = v.device_id
		WHERE 1=1`
	var args []any
	if opts.Status != "" {
		query += " AND v.status = ?"
		args = append(args, opts.Status)
	}
	if opts.DeviceID != "" {
		query += " AND v.device_id = ?"
		args = append(args, opts.DeviceID)
	}
	query += " ORDER BY v.cvss_score DESC, v.id"
	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += sqlOffset
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing vulnerability findings: %w", err)
	}
	defer rows.Close()

	var out []vulntriage.Finding
	for rows.Next() {
		var (
			v        vulntriage.Finding
			detected string
			resolved sql.NullString
		)
		if err = rows.Scan(&v.ID, &v.DeviceID, &v.DeviceIP, &v.Hostname, &v.CVEID,
			&v.Severity, &v.CVSSScore, &v.Description, &v.AffectedComponent,
			&v.AffectedVersion, &v.Status, &detected, &resolved); err != nil {
			return nil, fmt.Errorf("scanning vulnerability finding: %w", err)
		}
		if v.DetectedAt, err = time.Parse(time.RFC3339, detected); err != nil {
			return nil, fmt.Errorf("parsing detected_at of finding %d: %w", v.ID, err)
		}
		if resolved.Valid {
			t, parseErr := time.Parse(time.RFC3339, resolved.String)
			if parseErr != nil {
				return nil, fmt.Errorf("parsing resolved_at of finding %d: %w", v.ID, parseErr)
			}
			v.ResolvedAt = &t
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating vulnerability findings: %w", err)
	}
	return out, nil
}

// SetStatus applies an operator's triage decision and records it in the
// finding's history. The operator may acknowledge, ignore (with a reason) or
// reopen an unresolved finding; see vulntriage.ErrTransition for what is refused.
func (r *VulnerabilityRepository) SetStatus(
	ctx context.Context, id int64, to vulntriage.Status, actor, reason string, at time.Time,
) error {
	switch to {
	case vulntriage.StatusNew, vulntriage.StatusAcknowledged:
	case vulntriage.StatusIgnored:
		if reason == "" {
			return vulntriage.ErrReasonRequired
		}
	case vulntriage.StatusResolved:
		return vulntriage.ErrTransition
	default:
		return fmt.Errorf("%w: unknown status %q", vulntriage.ErrTransition, to)
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var from vulntriage.Status
		err := tx.QueryRowContext(ctx,
			`SELECT status FROM device_vulnerabilities WHERE id = ?`, id).Scan(&from)
		if errors.Is(err, sql.ErrNoRows) {
			return vulntriage.ErrFindingNotFound
		}
		if err != nil {
			return fmt.Errorf("reading finding %d: %w", id, err)
		}
		if from == vulntriage.StatusResolved || from == to {
			return fmt.Errorf("%w: finding %d is %s", vulntriage.ErrTransition, id, from)
		}
		if _, err = tx.ExecContext(ctx,
			`UPDATE device_vulnerabilities SET status = ? WHERE id = ?`, to, id); err != nil {
			return fmt.Errorf("updating finding %d: %w", id, err)
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO vulnerability_status_history
				(vulnerability_id, from_status, to_status, actor, reason, changed_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, id, from, to, actor, toNullString(reason), at.UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("recording status change of finding %d: %w", id, err)
		}
		return nil
	})
}

// History returns a finding's status changes, oldest first.
func (r *VulnerabilityRepository) History(ctx context.Context, id int64) ([]vulntriage.StatusChange, error) {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM device_vulnerabilities WHERE id = ?)`, id).Scan(&exists); err != nil {
		return nil, fmt.Errorf("reading finding %d: %w", id, err)
	}
	if !exists {
		return nil, vulntriage.ErrFindingNotFound
	}

	rows, err := r.db.Query(ctx, `
		SELECT from_status, to_status, COALESCE(actor, ''), COALESCE(reason, ''), changed_at
		FROM vulnerability_status_history
		WHERE vulnerability_id = ?
		ORDER BY id
	`, id)
	if err != nil {
		return nil, fmt.Errorf("listing history of finding %d: %w", id, err)
	}
	defer rows.Close()

	out := []vulntriage.StatusChange{}
	for rows.Next() {
		var (
			c       vulntriage.StatusChange
			changed string
		)
		if err = rows.Scan(&c.From, &c.To, &c.Actor, &c.Reason, &changed); err != nil {
			return nil, fmt.Errorf("scanning history of finding %d: %w", id, err)
		}
		if c.ChangedAt, err = time.Parse(time.RFC3339, changed); err != nil {
			return nil, fmt.Errorf("parsing changed_at of finding %d: %w", id, err)
		}
		out = append(out, c)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating history of finding %d: %w", id, err)
	}
	return out, nil
}
