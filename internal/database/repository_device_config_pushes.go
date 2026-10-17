package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

const deviceConfigPlanColumns = `id, client_id, target_id, created_by, created_at, expires_at, mode,
	diff_source, backup_id, running_sha256, diff_sha256, added, removed, applied_at, applied_by, apply_outcome`

func scanDeviceConfigPlan(row interface{ Scan(...any) error }, extra ...any) (deviceconfig.Plan, error) {
	var (
		p                                deviceconfig.Plan
		createdAt, expiresAt             string
		mode, source                     string
		appliedAt, appliedBy, applyState sql.NullString
	)
	dest := append([]any{
		&p.ID, &p.ClientID, &p.TargetID, &p.CreatedBy, &createdAt, &expiresAt, &mode,
		&source, &p.BackupID, &p.RunningSHA256, &p.DiffSHA256, &p.Added, &p.Removed,
		&appliedAt, &appliedBy, &applyState,
	}, extra...)
	if err := row.Scan(dest...); err != nil {
		return deviceconfig.Plan{}, err
	}
	p.CreatedAt = parseBackupTime(createdAt)
	p.ExpiresAt = parseBackupTime(expiresAt)
	p.Mode = deviceconfig.PushMode(mode)
	p.DiffSource = deviceconfig.DiffSource(source)
	if appliedAt.Valid {
		at := parseBackupTime(appliedAt.String)
		p.AppliedAt = &at
	}
	p.AppliedBy = appliedBy.String
	p.ApplyOutcome = deviceconfig.ApplyOutcome(applyState.String)
	return p, nil
}

// InsertPlan stores a push plan, generating its id. The candidate and the
// diff are stored as their ciphertext.
func (r *DeviceConfigRepository) InsertPlan(ctx context.Context, p *deviceconfig.Plan) error {
	p.ID = "dcp-" + randomID()
	_, err := r.db.Exec(ctx, `INSERT INTO device_config_push_plans (id, client_id, target_id, created_by,
			created_at, expires_at, mode, diff_source, backup_id, running_sha256, diff_sha256, added, removed,
			candidate_enc, diff_enc)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.ClientID, p.TargetID, p.CreatedBy, p.CreatedAt.UTC().Format(time.RFC3339),
		p.ExpiresAt.UTC().Format(time.RFC3339), string(p.Mode), string(p.DiffSource), p.BackupID,
		p.RunningSHA256, p.DiffSHA256, p.Added, p.Removed, []byte(p.CandidateCT), []byte(p.DiffCT))
	if err != nil {
		return fmt.Errorf("insert device_config_push_plans: %w", err)
	}
	return nil
}

// GetPlan returns one plan with its candidate's and diff's ciphertext.
func (r *DeviceConfigRepository) GetPlan(ctx context.Context, clientID, id string) (deviceconfig.Plan, error) {
	var candidate, diff []byte
	p, err := scanDeviceConfigPlan(r.db.QueryRow(ctx, `SELECT `+deviceConfigPlanColumns+`, candidate_enc, diff_enc
		FROM device_config_push_plans WHERE id = ? AND client_id = ?`, id, clientID), &candidate, &diff)
	if errors.Is(err, sql.ErrNoRows) {
		return deviceconfig.Plan{}, deviceconfig.ErrNotFound
	}
	if err != nil {
		return deviceconfig.Plan{}, fmt.Errorf("get device_config_push_plans: %w", err)
	}
	p.CandidateCT, p.DiffCT = string(candidate), string(diff)
	return p, nil
}

// ListPlans returns a target's plans newest first, without ciphertext.
func (r *DeviceConfigRepository) ListPlans(
	ctx context.Context, clientID, targetID string, limit int,
) ([]deviceconfig.Plan, error) {
	rows, err := r.db.Query(ctx, `SELECT `+deviceConfigPlanColumns+`
		FROM device_config_push_plans WHERE client_id = ? AND target_id = ?
		ORDER BY created_at DESC, id LIMIT ?`, clientID, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("list device_config_push_plans: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []deviceconfig.Plan
	for rows.Next() {
		p, scanErr := scanDeviceConfigPlan(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan device_config_push_plans: %w", scanErr)
		}
		out = append(out, p)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device_config_push_plans: %w", iterErr)
	}
	return out, nil
}

// PrunePlans deletes all but the target's newest keep plans.
func (r *DeviceConfigRepository) PrunePlans(ctx context.Context, clientID, targetID string, keep int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM device_config_push_plans
		WHERE client_id = ? AND target_id = ? AND id NOT IN (
			SELECT id FROM device_config_push_plans WHERE client_id = ? AND target_id = ?
			ORDER BY created_at DESC, id LIMIT ?)`,
		clientID, targetID, clientID, targetID, keep)
	if err != nil {
		return fmt.Errorf("prune device_config_push_plans: %w", err)
	}
	return nil
}

// ClaimApply marks a plan applied, only if it is unapplied and unexpired at
// now. Timestamps are RFC 3339 UTC, so they compare as text.
func (r *DeviceConfigRepository) ClaimApply(ctx context.Context, clientID, id, user string, now time.Time) error {
	at := now.UTC().Format(time.RFC3339)
	res, err := r.db.Exec(ctx, `UPDATE device_config_push_plans SET applied_at = ?, applied_by = ?
		WHERE id = ? AND client_id = ? AND applied_at IS NULL AND expires_at > ?`, at, user, id, clientID, at)
	if err != nil {
		return fmt.Errorf("claim device_config_push_plans apply: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("claim device_config_push_plans apply: %w", err)
	}
	if n == 0 {
		return deviceconfig.ErrPlanSpent
	}
	return nil
}

// FinishApply records the outcome of a claimed plan's apply.
func (r *DeviceConfigRepository) FinishApply(
	ctx context.Context, clientID, id string, outcome deviceconfig.ApplyOutcome,
) error {
	_, err := r.db.Exec(ctx, `UPDATE device_config_push_plans SET apply_outcome = ?
		WHERE id = ? AND client_id = ? AND applied_at IS NOT NULL`, string(outcome), id, clientID)
	if err != nil {
		return fmt.Errorf("finish device_config_push_plans apply: %w", err)
	}
	return nil
}
