package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// DeviceCredentialRepository reads device_credentials. There is no write path
// yet: credentials are provisioned out of band, and the SNMP poller only ever
// reads them. When a write path lands it must store the three *_enc columns as
// the ASCII "enc:v<N>:..." string produced by Config.EncryptCredentialValue —
// that is the only form the poll-time resolver accepts.
type DeviceCredentialRepository struct {
	db *DB
}

// Get returns one credential scoped to a client. The client_id predicate is
// load-bearing: polling_targets.credentials_id is a bare FK, so without it a
// target could resolve another client's credential. A miss (unknown id, or an
// id owned by another client) returns [polling.ErrCredentialNotFound].
//
// The returned Credential carries ciphertext only — decryption happens in the
// SNMP credential resolver, not here.
func (r *DeviceCredentialRepository) Get(ctx context.Context, id, clientID string) (*polling.Credential, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, client_id, name,
			snmp_community_enc, snmp_v3_user, snmp_v3_auth_enc, snmp_v3_priv_enc,
			snmp_v3_auth_proto, snmp_v3_priv_proto, created_at, updated_at
		FROM device_credentials WHERE id = ? AND client_id = ?
	`, id, clientID)

	var (
		c                            polling.Credential
		community, auth, priv        []byte
		v3User, authProto, privProto sql.NullString
		createdAt, updatedAt         string
	)
	err := row.Scan(
		&c.ID, &c.ClientID, &c.Name,
		&community, &v3User, &auth, &priv,
		&authProto, &privProto, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, polling.ErrCredentialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get device_credentials: %w", err)
	}

	c.CommunityEnc = string(community)
	c.V3AuthEnc = string(auth)
	c.V3PrivEnc = string(priv)
	c.V3User = v3User.String
	c.V3AuthProto = authProto.String
	c.V3PrivProto = privProto.String
	c.CreatedAt = parseTime(createdAt)
	c.UpdatedAt = parseTime(updatedAt)
	return &c, nil
}
