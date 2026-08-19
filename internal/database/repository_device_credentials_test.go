package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// insertCredential writes one device_credentials row directly. There is no
// write path in the repository yet (credentials are provisioned out of band),
// so the test owns the INSERT — and encrypts the secrets the same way the
// future writer must: Config.EncryptCredentialValue into the BLOB columns.
func insertCredential(t *testing.T, db *database.DB, cfg *config.Config, id, clientID, community string) {
	t.Helper()
	enc, err := cfg.EncryptCredentialValue(community)
	require.NoError(t, err)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(context.Background(), `
		INSERT INTO device_credentials
			(id, client_id, name, snmp_community_enc, snmp_v3_user,
			 snmp_v3_auth_proto, snmp_v3_priv_proto, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, clientID, "cred-"+id, []byte(enc), "snmpuser", "SHA", "AES", now, now)
	require.NoError(t, err)
}

func TestDeviceCredentials_GetRoundTripsCiphertext(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	cfg := &config.Config{}

	insertCredential(t, db, cfg, "cred-1", "default", "s3cr3t-community")

	got, err := db.DeviceCredentials().Get(ctx, "cred-1", "default")
	require.NoError(t, err)
	require.Equal(t, "cred-1", got.ID)
	require.Equal(t, "default", got.ClientID)
	require.Equal(t, "snmpuser", got.V3User)
	require.Equal(t, "SHA", got.V3AuthProto)
	require.Equal(t, "AES", got.V3PrivProto)
	require.NotContains(t, got.CommunityEnc, "s3cr3t-community",
		"the repository must hand back ciphertext, never plaintext")

	plain, err := cfg.DecryptSNMPPassword(got.CommunityEnc)
	require.NoError(t, err, "the stored BLOB must decrypt with the credential DEK")
	require.Equal(t, "s3cr3t-community", plain)

	// Unset v3 secrets stay empty rather than becoming garbage.
	require.Empty(t, got.V3AuthEnc)
	require.Empty(t, got.V3PrivEnc)
}

func TestDeviceCredentials_GetMissesAreNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	cfg := &config.Config{}

	err := db.Clients().Create(ctx, &database.Client{ID: "tenant-b", Name: "Tenant B", Slug: "tenant-b"})
	require.NoError(t, err)
	insertCredential(t, db, cfg, "cred-b", "tenant-b", "tenant-b-community")

	_, err = db.DeviceCredentials().Get(ctx, "no-such-cred", "default")
	require.ErrorIs(t, err, polling.ErrCredentialNotFound)

	// Present, but owned by another client: the FK alone would let a
	// polling target reach it, so the client_id predicate must reject it.
	_, err = db.DeviceCredentials().Get(ctx, "cred-b", "default")
	require.ErrorIs(t, err, polling.ErrCredentialNotFound)
}
