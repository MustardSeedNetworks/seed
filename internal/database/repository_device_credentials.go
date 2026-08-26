package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
		SELECT id, client_id, name, kind, security_level,
			snmp_community_enc, snmp_v3_user,
			snmp_v3_auth_enc, snmp_v3_priv_enc, snmp_v3_auth_proto,
			snmp_v3_priv_proto, created_at, updated_at
		FROM device_credentials
		WHERE id = ? AND client_id = ?
	`

	var (
		c                           polling.Credentials
		community, authSec, privSec []byte
		level                       sql.NullString
		user, authProto, privProto  sql.NullString
		createdAt, updatedAt        string
	)
	err := r.db.QueryRow(ctx, query, id, clientID).Scan(
		&c.ID, &c.ClientID, &c.Name, &c.Kind, &level,
		&community, &user,
		&authSec, &privSec, &authProto, &privProto,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", polling.ErrCredentialsNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get device_credentials: %w", err)
	}

	c.SecurityLevel = level.String
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

// List returns every credential belonging to one client, newest first.
//
// Discovery needs this and Get cannot serve it: polling knows which credential
// a target references, but discovery sweeps hosts it has never seen and has to
// try each stored credential in turn — the same shape the plaintext
// config.Communities list used to provide (#1799).
//
// Ciphertext comes back with the rows. The domain type marks every secret
// field `json:"-"`, so a handler that serialises the result cannot leak one,
// and decryption stays at point of use.
func (r *DeviceCredentialRepository) List(
	ctx context.Context,
	clientID string,
) ([]*polling.Credentials, error) {
	const query = `
		SELECT id, client_id, name, kind, security_level,
			snmp_community_enc, snmp_v3_user,
			snmp_v3_auth_enc, snmp_v3_priv_enc, snmp_v3_auth_proto,
			snmp_v3_priv_proto, created_at, updated_at
		FROM device_credentials
		WHERE client_id = ?
		ORDER BY created_at DESC, id
	`

	rows, queryErr := r.db.Query(ctx, query, clientID)
	if queryErr != nil {
		return nil, fmt.Errorf("list device_credentials: %w", queryErr)
	}
	defer func() { _ = rows.Close() }()

	var out []*polling.Credentials
	for rows.Next() {
		var (
			c                           polling.Credentials
			community, authSec, privSec []byte
			level                       sql.NullString
			user, authProto, privProto  sql.NullString
			createdAt, updatedAt        string
		)
		if scanErr := rows.Scan(
			&c.ID, &c.ClientID, &c.Name, &c.Kind, &level,
			&community, &user,
			&authSec, &privSec, &authProto, &privProto,
			&createdAt, &updatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan device_credentials: %w", scanErr)
		}
		c.SecurityLevel = level.String
		c.SNMPCommunityCT = string(community)
		c.SNMPv3AuthCT = string(authSec)
		c.SNMPv3PrivCT = string(privSec)
		c.SNMPv3User = user.String
		c.SNMPv3AuthProto = authProto.String
		c.SNMPv3PrivProto = privProto.String
		c.CreatedAt = parseCredentialTime(createdAt)
		c.UpdatedAt = parseCredentialTime(updatedAt)
		out = append(out, &c)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate device_credentials: %w", iterErr)
	}
	return out, nil
}

// Upsert writes a credentials row. The caller supplies already-encrypted
// secrets; passing plaintext here would persist it, so the parameter names
// carry the CT suffix the domain type uses.
//
// kind and security_level are derived here rather than taken from the caller,
// because they are not independent facts: they are names for which secrets the
// credential holds. Deriving them means the two can never disagree. A
// credential whose contents have no canonical name is refused here, with a
// message saying which combination, rather than left for the schema to reject
// with a constraint number.
func (r *DeviceCredentialRepository) Upsert(ctx context.Context, c *polling.Credentials) error {
	kind, level, err := canonicalizeCredentials(c)
	if err != nil {
		return err
	}

	// A blank id means create, matching PollingTargetRepository.Create. The id
	// is generated here rather than in the handler so it cannot be supplied by
	// a caller: a client that chose its own would be able to guess or collide
	// with another tenant's.
	if c.ID == "" {
		c.ID = "cred-" + randomID()
	}

	const query = `
		INSERT INTO device_credentials (
			id, client_id, name, kind, security_level,
			snmp_community_enc, snmp_v3_user,
			snmp_v3_auth_enc, snmp_v3_priv_enc, snmp_v3_auth_proto,
			snmp_v3_priv_proto, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			client_id = excluded.client_id,
			name = excluded.name,
			kind = excluded.kind,
			security_level = excluded.security_level,
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

	if _, execErr := r.db.Exec(ctx, query,
		c.ID, c.ClientID, c.Name, kind, nullIfEmpty(level),
		blobOrNull(c.SNMPCommunityCT), nullIfEmpty(c.SNMPv3User),
		blobOrNull(c.SNMPv3AuthCT), blobOrNull(c.SNMPv3PrivCT),
		nullIfEmpty(normalizeProto(c.SNMPv3AuthProto)),
		nullIfEmpty(normalizeProto(c.SNMPv3PrivProto)),
		created, now,
	); execErr != nil {
		return fmt.Errorf("upsert device_credentials: %w", execErr)
	}
	return nil
}

// ErrCredentialAmbiguous is returned when a credential holds a combination of
// secrets the canonical form has no name for — both a community string and a
// v3 user, or neither, or privacy without authentication.
var ErrCredentialAmbiguous = errors.New("device_credentials: credential is neither a v2c nor a v3 credential")

// canonicalizeCredentials names what a credential actually holds.
func canonicalizeCredentials(c *polling.Credentials) (string, string, error) {
	hasCommunity := c.SNMPCommunityCT != ""
	hasUser := strings.TrimSpace(c.SNMPv3User) != ""

	switch {
	case hasCommunity && hasUser:
		return "", "", fmt.Errorf("%w: %q has both a community string and a v3 user", ErrCredentialAmbiguous, c.ID)
	case !hasCommunity && !hasUser:
		return "", "", fmt.Errorf("%w: %q has neither a community string nor a v3 user", ErrCredentialAmbiguous, c.ID)
	case hasCommunity:
		return polling.CredentialKindV2c, "", nil
	}

	switch {
	case c.SNMPv3PrivCT != "" && c.SNMPv3AuthCT == "":
		return "", "", fmt.Errorf("%w: %q has privacy without authentication", ErrCredentialAmbiguous, c.ID)
	case c.SNMPv3PrivCT != "":
		return polling.CredentialKindV3, polling.SecurityLevelAuthPriv, nil
	case c.SNMPv3AuthCT != "":
		return polling.CredentialKindV3, polling.SecurityLevelAuthNoPriv, nil
	default:
		return polling.CredentialKindV3, polling.SecurityLevelNoAuthNoPriv, nil
	}
}

// blobOrNull writes NULL rather than a zero-length blob for an absent secret.
// The two are different to the schema: NULL means "this kind of credential
// does not have one", an empty blob is a secret that is not ciphertext.
func blobOrNull(ct string) any {
	if ct == "" {
		return nil
	}
	return []byte(ct)
}

// nullIfEmpty writes NULL rather than "" for an absent column, on the same
// terms as blobOrNull.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// normalizeProto folds a protocol name to the spelling the schema accepts.
// SHA1 is the same protocol as SHA and is stored as SHA; AES128 likewise as
// AES. An unrecognised name is left alone so the CHECK constraint rejects it
// by name rather than being silently coerced into something that works.
func normalizeProto(name string) string {
	switch upper := strings.ToUpper(strings.TrimSpace(name)); upper {
	case "SHA1":
		return "SHA"
	case "AES128":
		return "AES"
	default:
		return upper
	}
}

// ErrCredentialInUse is returned when a credential cannot be deleted because a
// polling target still references it.
var ErrCredentialInUse = errors.New("device_credentials: credential is in use by a polling target")

// Delete removes a credentials row owned by clientID.
//
// The reference from polling_targets is ON DELETE RESTRICT, so deleting a
// credential a target still uses now fails instead of succeeding. The previous
// SET NULL quietly unbound every target that referenced it — an operator
// deleting one unused credential could disarm a dozen live ones and get no
// error. Refusing is the honest answer: unbind the targets first, deliberately.
func (r *DeviceCredentialRepository) Delete(ctx context.Context, clientID, id string) error {
	if clientID == "" {
		return errors.New("device_credentials: client id required for Delete")
	}
	_, err := r.db.Exec(ctx,
		`DELETE FROM device_credentials WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		if isForeignKeyConstraintError(err) {
			return fmt.Errorf("%w: %s", ErrCredentialInUse, id)
		}
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

// isForeignKeyConstraintError reports whether err is SQLite refusing a write
// because a foreign key would be violated — here, the ON DELETE RESTRICT from
// polling_targets.
func isForeignKeyConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}
