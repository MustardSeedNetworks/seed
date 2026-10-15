package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// DeviceGroupRepository owns device_groups and device_group_members (P-D10).
// Every management query is scoped to the client; MemberAddresses is the alert
// pipeline's cross-client read.
type DeviceGroupRepository struct {
	db *DB
}

// List returns the client's groups ordered by name, each with its members.
func (r *DeviceGroupRepository) List(ctx context.Context, clientID string) ([]devicegroups.Group, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, name, description, created_at, updated_at
		FROM device_groups WHERE client_id = ? ORDER BY name, id`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list device_groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []devicegroups.Group
	index := map[string]int{}
	for rows.Next() {
		g, scanErr := scanDeviceGroup(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan device_groups: %w", scanErr)
		}
		index[g.ID] = len(out)
		out = append(out, g)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device_groups: %w", iterErr)
	}

	members, err := r.members(ctx, `SELECT m.group_id, m.target_id FROM device_group_members m
		JOIN device_groups g ON g.id = m.group_id
		WHERE g.client_id = ? ORDER BY m.group_id, m.target_id`, clientID)
	if err != nil {
		return nil, err
	}
	for groupID, ids := range members {
		if i, ok := index[groupID]; ok {
			out[i].Members = ids
		}
	}
	return out, nil
}

// Get returns one group with its members; a miss is devicegroups.ErrNotFound.
func (r *DeviceGroupRepository) Get(ctx context.Context, clientID, id string) (devicegroups.Group, error) {
	g, err := scanDeviceGroup(r.db.QueryRow(ctx, `SELECT id, client_id, name, description, created_at, updated_at
		FROM device_groups WHERE id = ? AND client_id = ?`, id, clientID))
	if errors.Is(err, sql.ErrNoRows) {
		return devicegroups.Group{}, devicegroups.ErrNotFound
	}
	if err != nil {
		return devicegroups.Group{}, fmt.Errorf("get device_groups: %w", err)
	}
	members, err := r.members(ctx, `SELECT group_id, target_id FROM device_group_members
		WHERE group_id = ? ORDER BY target_id`, id)
	if err != nil {
		return devicegroups.Group{}, err
	}
	g.Members = members[id]
	return g, nil
}

// Save creates g when g.ID is blank, generating the id, and otherwise replaces
// the client's group with that id. The member list is replaced in the same
// transaction; a member that is not one of the client's polling targets fails
// the whole save with devicegroups.ErrUnknownMember.
func (r *DeviceGroupRepository) Save(ctx context.Context, g *devicegroups.Group) error {
	now := time.Now().UTC().Format(time.RFC3339)
	creating := g.ID == ""
	if creating {
		g.ID = "grp-" + randomID()
	}
	err := r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var (
			res     sql.Result
			execErr error
		)
		if creating {
			res, execErr = tx.ExecContext(ctx, `INSERT INTO device_groups
				(id, client_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				g.ID, g.ClientID, g.Name, g.Description, now, now)
		} else {
			res, execErr = tx.ExecContext(ctx, `UPDATE device_groups
				SET name = ?, description = ?, updated_at = ? WHERE id = ? AND client_id = ?`,
				g.Name, g.Description, now, g.ID, g.ClientID)
		}
		if execErr != nil {
			return execErr
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return devicegroups.ErrNotFound
		}
		if _, execErr = tx.ExecContext(ctx,
			`DELETE FROM device_group_members WHERE group_id = ?`, g.ID); execErr != nil {
			return execErr
		}
		for _, targetID := range g.Members {
			// The SELECT admits only the client's own targets, so a foreign
			// or unknown id inserts nothing.
			res, execErr = tx.ExecContext(ctx, `INSERT INTO device_group_members (group_id, target_id)
				SELECT ?, id FROM polling_targets WHERE id = ? AND client_id = ?`,
				g.ID, targetID, g.ClientID)
			if execErr != nil {
				return execErr
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return devicegroups.ErrUnknownMember
			}
		}
		return nil
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, devicegroups.ErrNotFound), errors.Is(err, devicegroups.ErrUnknownMember):
		return err
	case isUniqueConstraintError(err):
		return devicegroups.ErrNameTaken
	default:
		return fmt.Errorf("save device_groups: %w", err)
	}
}

// Delete removes a group and its memberships. A group an alert rule names is
// devicegroups.ErrInUse (alert_rules.group_id is ON DELETE RESTRICT).
func (r *DeviceGroupRepository) Delete(ctx context.Context, clientID, id string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM device_groups WHERE id = ? AND client_id = ?`, id, clientID)
	if isForeignKeyConstraintError(err) {
		return devicegroups.ErrInUse
	}
	if err != nil {
		return fmt.Errorf("delete device_groups: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return devicegroups.ErrNotFound
	}
	return nil
}

// MemberTargets returns the group's member polling targets ordered by name.
// The caller has already resolved the group within its client.
func (r *DeviceGroupRepository) MemberTargets(ctx context.Context, groupID string) ([]polling.Target, error) {
	rows, err := r.db.Query(ctx, selectPollingTarget+`
		WHERE id IN (SELECT target_id FROM device_group_members WHERE group_id = ?)
		ORDER BY name, id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list device group members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []polling.Target
	for rows.Next() {
		t, scanErr := scanPollingTarget(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *t)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device group members: %w", iterErr)
	}
	return out, nil
}

// MemberAddresses returns every group's member addresses keyed by group id,
// across all clients. A group with no members is absent. The alert pipeline
// reads it to scope rules; it is not a management read.
func (r *DeviceGroupRepository) MemberAddresses(ctx context.Context) (map[string][]string, error) {
	return r.members(ctx, `SELECT m.group_id, t.ip_address FROM device_group_members m
		JOIN polling_targets t ON t.id = m.target_id ORDER BY m.group_id, t.ip_address`)
}

// members runs a (group id, value) query and collects the values per group.
func (r *DeviceGroupRepository) members(ctx context.Context, query string, args ...any) (map[string][]string, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device_group_members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string][]string{}
	for rows.Next() {
		var groupID, value string
		if scanErr := rows.Scan(&groupID, &value); scanErr != nil {
			return nil, fmt.Errorf("scan device_group_members: %w", scanErr)
		}
		out[groupID] = append(out[groupID], value)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device_group_members: %w", iterErr)
	}
	return out, nil
}

func scanDeviceGroup(row interface{ Scan(...any) error }) (devicegroups.Group, error) {
	var (
		g                    devicegroups.Group
		createdAt, updatedAt string
	)
	if err := row.Scan(&g.ID, &g.ClientID, &g.Name, &g.Description, &createdAt, &updatedAt); err != nil {
		return devicegroups.Group{}, err
	}
	g.CreatedAt = parseCredentialTime(createdAt)
	g.UpdatedAt = parseCredentialTime(updatedAt)
	return g, nil
}
