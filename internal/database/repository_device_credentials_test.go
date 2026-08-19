package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// insertCredential writes one device_credentials row directly. The repository
// has no write path — credentials are provisioned out of band — so the test
// owns the INSERT.
func insertCredential(t *testing.T, db *database.DB, id, clientID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(context.Background(), `
		INSERT INTO device_credentials
			(id, client_id, name, snmp_community_enc, snmp_v3_user,
			 snmp_v3_auth_proto, snmp_v3_priv_proto, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, clientID, "cred-"+id, []byte("enc:v1:community"), "operator", "SHA", "AES", now, now)
	require.NoError(t, err)
}

func TestDeviceCredentialsGetReturnsCiphertextForTheOwningClient(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	insertCredential(t, db, "cred-1", database.DefaultClientID)

	got, err := db.DeviceCredentials().Get(ctx, "cred-1", database.DefaultClientID)
	require.NoError(t, err)
	require.Equal(t, "cred-1", got.ID)
	require.Equal(t, database.DefaultClientID, got.ClientID)
	require.Equal(t, "enc:v1:community", got.SNMPCommunityCT)
	require.Equal(t, "operator", got.SNMPv3User)
}

// TestDeviceCredentialsGetRefusesAnotherClientsRow is the invariant
// polling_targets.credentials_id cannot carry: the FK proves the row exists,
// not that it belongs to the target's client.
func TestDeviceCredentialsGetRefusesAnotherClientsRow(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, db.Clients().Create(ctx, &database.Client{
		ID: "tenant-b", Name: "Tenant B", Slug: "tenant-b",
	}))
	insertCredential(t, db, "cred-b", "tenant-b")

	_, err := db.DeviceCredentials().Get(ctx, "cred-b", database.DefaultClientID)
	require.ErrorIs(t, err, polling.ErrCredentialsNotFound)

	_, err = db.DeviceCredentials().Get(ctx, "no-such-cred", database.DefaultClientID)
	require.ErrorIs(t, err, polling.ErrCredentialsNotFound)
}
