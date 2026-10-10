package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

// DeviceConfigRepository owns device_config_targets and device_config_backups (P-D1).
// Every query is scoped to the client, so another client's target or backup is
// indistinguishable from one that does not exist.
type DeviceConfigRepository struct {
	db *DB
}

// ErrDeviceConfigTargetConflict is returned when a client already has a target
// on the same host and port.
var ErrDeviceConfigTargetConflict = errors.New("device_config_targets: a target already uses this host and port")

const deviceConfigTargetColumns = `id, client_id, name, host, port, platform,
	credentials_id, host_key_sha256, created_at, updated_at`

func scanDeviceConfigTarget(row interface{ Scan(...any) error }) (deviceconfig.Target, error) {
	var (
		t                    deviceconfig.Target
		platform             string
		hostKey              sql.NullString
		createdAt, updatedAt string
	)
	if err := row.Scan(&t.ID, &t.ClientID, &t.Name, &t.Host, &t.Port, &platform,
		&t.CredentialsID, &hostKey, &createdAt, &updatedAt); err != nil {
		return deviceconfig.Target{}, err
	}
	t.Platform = deviceconfig.Platform(platform)
	t.HostKeySHA256 = hostKey.String
	t.CreatedAt = parseCredentialTime(createdAt)
	t.UpdatedAt = parseCredentialTime(updatedAt)
	return t, nil
}

// ListTargets returns the client's targets ordered by name.
func (r *DeviceConfigRepository) ListTargets(ctx context.Context, clientID string) ([]deviceconfig.Target, error) {
	rows, err := r.db.Query(ctx, `SELECT `+deviceConfigTargetColumns+`
		FROM device_config_targets WHERE client_id = ? ORDER BY name, id`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list device_config_targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []deviceconfig.Target
	for rows.Next() {
		t, scanErr := scanDeviceConfigTarget(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan device_config_targets: %w", scanErr)
		}
		out = append(out, t)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device_config_targets: %w", iterErr)
	}
	return out, nil
}

// GetTarget returns one target; a miss is deviceconfig.ErrNotFound.
func (r *DeviceConfigRepository) GetTarget(ctx context.Context, clientID, id string) (deviceconfig.Target, error) {
	t, err := scanDeviceConfigTarget(r.db.QueryRow(ctx, `SELECT `+deviceConfigTargetColumns+`
		FROM device_config_targets WHERE id = ? AND client_id = ?`, id, clientID))
	if errors.Is(err, sql.ErrNoRows) {
		return deviceconfig.Target{}, deviceconfig.ErrNotFound
	}
	if err != nil {
		return deviceconfig.Target{}, fmt.Errorf("get device_config_targets: %w", err)
	}
	return t, nil
}

// SaveTarget creates a target when t.ID is blank, generating the id, and
// otherwise replaces the client's target with that id.
func (r *DeviceConfigRepository) SaveTarget(ctx context.Context, t *deviceconfig.Target) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var (
		res sql.Result
		err error
	)
	if t.ID == "" {
		t.ID = "dct-" + randomID()
		res, err = r.db.Exec(ctx, `INSERT INTO device_config_targets (`+deviceConfigTargetColumns+`)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.ClientID, t.Name, t.Host, t.Port, string(t.Platform),
			t.CredentialsID, nullIfEmpty(t.HostKeySHA256), now, now)
	} else {
		res, err = r.db.Exec(ctx, `UPDATE device_config_targets SET
				name = ?, host = ?, port = ?, platform = ?, credentials_id = ?,
				host_key_sha256 = ?, updated_at = ?
			WHERE id = ? AND client_id = ?`,
			t.Name, t.Host, t.Port, string(t.Platform), t.CredentialsID,
			nullIfEmpty(t.HostKeySHA256), now, t.ID, t.ClientID)
	}
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDeviceConfigTargetConflict
		}
		return fmt.Errorf("save device_config_targets: %w", err)
	}
	return requireOneRow(res)
}

// DeleteTarget removes a target; its backups go with it (ON DELETE CASCADE).
func (r *DeviceConfigRepository) DeleteTarget(ctx context.Context, clientID, id string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM device_config_targets WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return fmt.Errorf("delete device_config_targets: %w", err)
	}
	return requireOneRow(res)
}

// PinHostKey records the fingerprint only where none is pinned yet.
func (r *DeviceConfigRepository) PinHostKey(ctx context.Context, clientID, id, fingerprint string) error {
	_, err := r.db.Exec(ctx, `UPDATE device_config_targets SET host_key_sha256 = ?
		WHERE id = ? AND client_id = ? AND host_key_sha256 IS NULL`, fingerprint, id, clientID)
	if err != nil {
		return fmt.Errorf("pin device_config_targets host key: %w", err)
	}
	return nil
}

// ClearHostKey forgets a target's pinned key.
func (r *DeviceConfigRepository) ClearHostKey(ctx context.Context, clientID, id string) error {
	res, err := r.db.Exec(ctx, `UPDATE device_config_targets SET host_key_sha256 = NULL, updated_at = ?
		WHERE id = ? AND client_id = ?`, time.Now().UTC().Format(time.RFC3339), id, clientID)
	if err != nil {
		return fmt.Errorf("clear device_config_targets host key: %w", err)
	}
	return requireOneRow(res)
}

// InsertBackup records one attempt, generating its id.
func (r *DeviceConfigRepository) InsertBackup(ctx context.Context, b *deviceconfig.Backup) error {
	b.ID = "dcb-" + randomID()
	_, err := r.db.Exec(ctx, `INSERT INTO device_config_backups
			(id, client_id, target_id, taken_at, status, error, config, sha256)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.ClientID, b.TargetID, b.TakenAt.UTC().Format(time.RFC3339Nano), string(b.Status),
		nullIfEmpty(b.Error), nullIfEmpty(b.Config), nullIfEmpty(b.SHA256))
	if err != nil {
		return fmt.Errorf("insert device_config_backups: %w", err)
	}
	return nil
}

// ListBackups returns a target's attempts newest first, without configs.
func (r *DeviceConfigRepository) ListBackups(
	ctx context.Context, clientID, targetID string, limit int,
) ([]deviceconfig.Backup, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, target_id, taken_at, status, error, sha256
		FROM device_config_backups WHERE client_id = ? AND target_id = ?
		ORDER BY taken_at DESC, id LIMIT ?`, clientID, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("list device_config_backups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []deviceconfig.Backup
	for rows.Next() {
		var (
			b           deviceconfig.Backup
			takenAt     string
			status      string
			msg, digest sql.NullString
		)
		if scanErr := rows.Scan(&b.ID, &b.ClientID, &b.TargetID, &takenAt, &status, &msg, &digest); scanErr != nil {
			return nil, fmt.Errorf("scan device_config_backups: %w", scanErr)
		}
		b.TakenAt = parseBackupTime(takenAt)
		b.Status = deviceconfig.Status(status)
		b.Error = msg.String
		b.SHA256 = digest.String
		out = append(out, b)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device_config_backups: %w", iterErr)
	}
	return out, nil
}

// GetBackup returns one attempt with its configuration.
func (r *DeviceConfigRepository) GetBackup(ctx context.Context, clientID, id string) (deviceconfig.Backup, error) {
	var (
		b                   deviceconfig.Backup
		takenAt, status     string
		msg, config, digest sql.NullString
	)
	err := r.db.QueryRow(ctx, `SELECT id, client_id, target_id, taken_at, status, error, config, sha256
		FROM device_config_backups WHERE id = ? AND client_id = ?`, id, clientID).
		Scan(&b.ID, &b.ClientID, &b.TargetID, &takenAt, &status, &msg, &config, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return deviceconfig.Backup{}, deviceconfig.ErrNotFound
	}
	if err != nil {
		return deviceconfig.Backup{}, fmt.Errorf("get device_config_backups: %w", err)
	}
	b.TakenAt = parseBackupTime(takenAt)
	b.Status = deviceconfig.Status(status)
	b.Error = msg.String
	b.Config = config.String
	b.SHA256 = digest.String
	return b, nil
}

func parseBackupTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// requireOneRow turns "matched nothing" into deviceconfig.ErrNotFound: the id
// is absent or belongs to another client, and the caller cannot tell which.
func requireOneRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return deviceconfig.ErrNotFound
	}
	return nil
}
