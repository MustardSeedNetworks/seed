package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

// WMIRepository owns wmi_targets and wmi_inventories (P-D11). Every query is
// scoped to the client, so another client's target is indistinguishable from
// one that does not exist.
type WMIRepository struct {
	db *DB
}

// ErrWMITargetConflict is returned when a client already has an inventory
// target on the same host and port.
var ErrWMITargetConflict = errors.New("wmi_targets: a target already uses this host and port")

const wmiTargetColumns = `id, client_id, name, host, port, credentials_id, cert_sha256,
	pending_cert_sha256, last_attempt_at, last_status, last_error, created_at, updated_at`

func scanWMITarget(row interface{ Scan(...any) error }) (wmi.Target, error) {
	var (
		t                        wmi.Target
		cert, pending, attemptAt sql.NullString
		status, lastError        sql.NullString
		createdAt, updatedAt     string
	)
	if err := row.Scan(&t.ID, &t.ClientID, &t.Name, &t.Host, &t.Port, &t.CredentialsID, &cert,
		&pending, &attemptAt, &status, &lastError, &createdAt, &updatedAt); err != nil {
		return wmi.Target{}, err
	}
	t.CertSHA256 = cert.String
	t.PendingCertSHA256 = pending.String
	if attemptAt.Valid {
		t.LastAttemptAt = parseBackupTime(attemptAt.String)
	}
	t.LastStatus = wmi.Status(status.String)
	t.LastError = lastError.String
	t.CreatedAt = parseCredentialTime(createdAt)
	t.UpdatedAt = parseCredentialTime(updatedAt)
	return t, nil
}

// ListTargets returns the client's targets ordered by name.
func (r *WMIRepository) ListTargets(ctx context.Context, clientID string) ([]wmi.Target, error) {
	rows, err := r.db.Query(ctx, `SELECT `+wmiTargetColumns+`
		FROM wmi_targets WHERE client_id = ? ORDER BY name, id`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list wmi_targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []wmi.Target
	for rows.Next() {
		t, scanErr := scanWMITarget(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan wmi_targets: %w", scanErr)
		}
		out = append(out, t)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate wmi_targets: %w", iterErr)
	}
	return out, nil
}

// GetTarget returns one target; a miss is wmi.ErrNotFound.
func (r *WMIRepository) GetTarget(ctx context.Context, clientID, id string) (wmi.Target, error) {
	t, err := scanWMITarget(r.db.QueryRow(ctx, `SELECT `+wmiTargetColumns+`
		FROM wmi_targets WHERE id = ? AND client_id = ?`, id, clientID))
	if errors.Is(err, sql.ErrNoRows) {
		return wmi.Target{}, wmi.ErrNotFound
	}
	if err != nil {
		return wmi.Target{}, fmt.Errorf("get wmi_targets: %w", err)
	}
	return t, nil
}

// SaveTarget creates a target when t.ID is blank, generating the id, and
// otherwise replaces the client's target with that id. The last attempt is
// not part of the definition and is left as it is.
func (r *WMIRepository) SaveTarget(ctx context.Context, t *wmi.Target) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var (
		res sql.Result
		err error
	)
	if t.ID == "" {
		t.ID = "wmt-" + randomID()
		res, err = r.db.Exec(ctx, `INSERT INTO wmi_targets (id, client_id, name, host, port, credentials_id,
				cert_sha256, pending_cert_sha256, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.ClientID, t.Name, t.Host, t.Port, t.CredentialsID,
			nullIfEmpty(t.CertSHA256), nullIfEmpty(t.PendingCertSHA256), now, now)
	} else {
		res, err = r.db.Exec(ctx, `UPDATE wmi_targets SET
				name = ?, host = ?, port = ?, credentials_id = ?,
				cert_sha256 = ?, pending_cert_sha256 = ?, updated_at = ?
			WHERE id = ? AND client_id = ?`,
			t.Name, t.Host, t.Port, t.CredentialsID,
			nullIfEmpty(t.CertSHA256), nullIfEmpty(t.PendingCertSHA256), now, t.ID, t.ClientID)
	}
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrWMITargetConflict
		}
		return fmt.Errorf("save wmi_targets: %w", err)
	}
	return requireOneWMIRow(res)
}

// DeleteTarget removes a target; its inventory goes with it (ON DELETE
// CASCADE).
func (r *WMIRepository) DeleteTarget(ctx context.Context, clientID, id string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM wmi_targets WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return fmt.Errorf("delete wmi_targets: %w", err)
	}
	return requireOneWMIRow(res)
}

// SetPendingCert records the certificate an unpinned target presented; a
// target whose certificate was pinned in the meantime is left alone.
func (r *WMIRepository) SetPendingCert(ctx context.Context, clientID, id, fingerprint string) error {
	_, err := r.db.Exec(ctx, `UPDATE wmi_targets SET pending_cert_sha256 = ?
		WHERE id = ? AND client_id = ? AND cert_sha256 IS NULL`, fingerprint, id, clientID)
	if err != nil {
		return fmt.Errorf("record wmi_targets pending certificate: %w", err)
	}
	return nil
}

// ConfirmCert pins the pending certificate when it is still fingerprint;
// otherwise it matches nothing and is wmi.ErrNotFound.
func (r *WMIRepository) ConfirmCert(ctx context.Context, clientID, id, fingerprint string) error {
	res, err := r.db.Exec(ctx, `UPDATE wmi_targets
		SET cert_sha256 = pending_cert_sha256, pending_cert_sha256 = NULL, updated_at = ?
		WHERE id = ? AND client_id = ? AND pending_cert_sha256 = ?`,
		time.Now().UTC().Format(time.RFC3339), id, clientID, fingerprint)
	if err != nil {
		return fmt.Errorf("confirm wmi_targets certificate: %w", err)
	}
	return requireOneWMIRow(res)
}

// ClearCert forgets a target's pinned and pending certificates.
func (r *WMIRepository) ClearCert(ctx context.Context, clientID, id string) error {
	res, err := r.db.Exec(ctx, `UPDATE wmi_targets
		SET cert_sha256 = NULL, pending_cert_sha256 = NULL, updated_at = ?
		WHERE id = ? AND client_id = ?`, time.Now().UTC().Format(time.RFC3339), id, clientID)
	if err != nil {
		return fmt.Errorf("clear wmi_targets certificate: %w", err)
	}
	return requireOneWMIRow(res)
}

// RecordAttempt stores the attempt on its target and, when it carries an
// inventory, replaces the target's stored one, in one transaction: a reader
// never sees a successful attempt next to the previous inventory.
func (r *WMIRepository) RecordAttempt(ctx context.Context, a wmi.Attempt) error {
	at := a.At.UTC().Format(time.RFC3339Nano)
	var doc []byte
	if a.Inventory != nil {
		var err error
		if doc, err = json.Marshal(a.Inventory); err != nil {
			return fmt.Errorf("encode wmi inventory: %w", err)
		}
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE wmi_targets
			SET last_attempt_at = ?, last_status = ?, last_error = ?
			WHERE id = ? AND client_id = ?`,
			at, string(a.Status), nullIfEmpty(a.Error), a.TargetID, a.ClientID)
		if err != nil {
			return fmt.Errorf("record wmi_targets attempt: %w", err)
		}
		if err = requireOneWMIRow(res); err != nil || doc == nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wmi_inventories (client_id, target_id, collected_at, inventory)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (client_id, target_id) DO UPDATE SET
				collected_at = excluded.collected_at, inventory = excluded.inventory`,
			a.ClientID, a.TargetID, at, string(doc)); err != nil {
			return fmt.Errorf("store wmi_inventories: %w", err)
		}
		return nil
	})
}

// GetInventory returns the target's stored inventory, or wmi.ErrNotFound when
// no collection has succeeded.
func (r *WMIRepository) GetInventory(ctx context.Context, clientID, targetID string) (wmi.StoredInventory, error) {
	var collectedAt, doc string
	err := r.db.QueryRow(ctx, `SELECT collected_at, inventory FROM wmi_inventories
		WHERE client_id = ? AND target_id = ?`, clientID, targetID).Scan(&collectedAt, &doc)
	if errors.Is(err, sql.ErrNoRows) {
		return wmi.StoredInventory{}, wmi.ErrNotFound
	}
	if err != nil {
		return wmi.StoredInventory{}, fmt.Errorf("get wmi_inventories: %w", err)
	}
	out := wmi.StoredInventory{TargetID: targetID, CollectedAt: parseBackupTime(collectedAt)}
	if err = json.Unmarshal([]byte(doc), &out.Inventory); err != nil {
		return wmi.StoredInventory{}, fmt.Errorf("decode wmi_inventories: %w", err)
	}
	return out, nil
}

// requireOneWMIRow turns "matched nothing" into wmi.ErrNotFound: the id is
// absent or belongs to another client, and the caller cannot tell which.
func requireOneWMIRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return wmi.ErrNotFound
	}
	return nil
}
