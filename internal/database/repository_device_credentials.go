package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// DeviceCredentialRepository owns access to device_credentials.
//
// The secret columns are BLOBs holding versioned ciphertext produced by the
// config keyring. This repository moves those bytes verbatim and never
// decrypts: the keyring seam lives in internal/polling/snmp, so plaintext
// exists only for the duration of a poll.
type DeviceCredentialRepository struct {
	db *DB
}

// Get returns one credentials row, scoped to the client that owns it. A miss
// is polling.ErrCredentialsNotFound so callers can distinguish "no such
// credential" from a storage failure — the difference decides whether a poll is
// misconfigured or the database is down.
//
// The client_id predicate is the invariant polling_targets.credentials_id
// cannot carry: the FK proves the row exists, not that it belongs to the
// target's client. Without it a target in one client would resolve and use
// another client's community string and v3 secrets.
func (r *DeviceCredentialRepository) Get(ctx context.Context, id, clientID string) (*polling.Credentials, error) {
	const query = `
		SELECT id, client_id, name, snmp_community_enc, snmp_v3_user,
			snmp_v3_auth_enc, snmp_v3_priv_enc, snmp_v3_auth_proto,
			snmp_v3_priv_proto, created_at, updated_at
		FROM device_credentials
		WHERE id = ? AND client_id = ?
	`

	var (
		c                           polling.Credentials
		community, authSec, privSec []byte
		user, authProto, privProto  sql.NullString
		createdAt, updatedAt        string
	)
	err := r.db.QueryRow(ctx, query, id, clientID).Scan(
		&c.ID, &c.ClientID, &c.Name, &community, &user,
		&authSec, &privSec, &authProto, &privProto,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", polling.ErrCredentialsNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get device_credentials: %w", err)
	}

	c.SNMPCommunityCT = string(community)
	c.SNMPv3AuthCT = string(authSec)
	c.SNMPv3PrivCT = string(privSec)
	c.SNMPv3User = user.String
	c.SNMPv3AuthProto = authProto.String
	c.SNMPv3PrivProto = privProto.String
	c.CreatedAt = parseCredentialTime(createdAt)
	c.UpdatedAt = parseCredentialTime(updatedAt)

	return &c, nil
}

// Upsert writes a credentials row. The caller supplies already-encrypted
// secrets; passing plaintext here would persist it, so the parameter names
// carry the CT suffix the domain type uses.
func (r *DeviceCredentialRepository) Upsert(ctx context.Context, c *polling.Credentials) error {
	const query = `
		INSERT INTO device_credentials (
			id, client_id, name, snmp_community_enc, snmp_v3_user,
			snmp_v3_auth_enc, snmp_v3_priv_enc, snmp_v3_auth_proto,
			snmp_v3_priv_proto, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			client_id = excluded.client_id,
			name = excluded.name,
			snmp_community_enc = excluded.snmp_community_enc,
			snmp_v3_user = excluded.snmp_v3_user,
			snmp_v3_auth_enc = excluded.snmp_v3_auth_enc,
			snmp_v3_priv_enc = excluded.snmp_v3_priv_enc,
			snmp_v3_auth_proto = excluded.snmp_v3_auth_proto,
			snmp_v3_priv_proto = excluded.snmp_v3_priv_proto,
			updated_at = excluded.updated_at
	`

	now := time.Now().UTC().Format(time.RFC3339)
	created := now
	if !c.CreatedAt.IsZero() {
		created = c.CreatedAt.UTC().Format(time.RFC3339)
	}

	if _, err := r.db.Exec(ctx, query,
		c.ID, c.ClientID, c.Name, []byte(c.SNMPCommunityCT), c.SNMPv3User,
		[]byte(c.SNMPv3AuthCT), []byte(c.SNMPv3PrivCT), c.SNMPv3AuthProto,
		c.SNMPv3PrivProto, created, now,
	); err != nil {
		return fmt.Errorf("upsert device_credentials: %w", err)
	}
	return nil
}

// Delete removes a credentials row. Deleting a credential a target still
// references leaves that target unpollable rather than silently unauthenticated
// — the resolver fails closed on the missing row.
func (r *DeviceCredentialRepository) Delete(ctx context.Context, id string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM device_credentials WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete device_credentials: %w", err)
	}
	return nil
}

func parseCredentialTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
