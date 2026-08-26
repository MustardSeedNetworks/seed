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
			(id, client_id, name, kind, snmp_community_enc, created_at, updated_at)
		VALUES (?, ?, ?, 'v2c', ?, ?, ?)
	`, id, clientID, "cred-"+id, []byte("enc:v1:community"), now, now)
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
	// The fixture used to carry a v3 user alongside the community string. The
	// canonical schema has no name for that combination and rejects it, so the
	// v2c fixture now asserts what a v2c credential actually is.
	require.Equal(t, polling.CredentialKindV2c, got.Kind)
	require.Empty(t, got.SNMPv3User)
	require.Empty(t, got.SecurityLevel)
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

// TestDeviceCredentialsListIsScopedToTheClient is the same tenancy invariant
// Get carries, on the path discovery will use. Discovery sweeps hosts it has
// never seen and tries each stored credential in turn, so a List that leaked
// across clients would try one client's community against another's devices.
func TestDeviceCredentialsListIsScopedToTheClient(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	insertCredential(t, db, "cred-mine-1", database.DefaultClientID)
	insertCredential(t, db, "cred-mine-2", database.DefaultClientID)

	// device_credentials.client_id is a real FK, so the second tenant has to
	// exist before it can own anything.
	now := time.Now().UTC()
	_, err := db.Exec(ctx, `
		INSERT INTO clients (id, name, slug, created_at, updated_at)
		VALUES ('other-client', 'Other', 'other', ?, ?)`, now, now)
	require.NoError(t, err)
	insertCredential(t, db, "cred-theirs", "other-client")

	got, err := db.DeviceCredentials().List(ctx, database.DefaultClientID)
	require.NoError(t, err)
	require.Len(t, got, 2)

	ids := []string{got[0].ID, got[1].ID}
	require.ElementsMatch(t, []string{"cred-mine-1", "cred-mine-2"}, ids)
	for _, c := range got {
		require.Equal(t, database.DefaultClientID, c.ClientID)
	}
}

// TestDeviceCredentialsListCarriesCiphertext confirms the rows arrive with
// their secrets so the resolver can decrypt at point of use. The domain type
// marks every secret field `json:"-"`, which is what keeps a handler from
// serialising one.
func TestDeviceCredentialsListCarriesCiphertext(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	insertCredential(t, db, "cred-1", database.DefaultClientID)

	got, err := db.DeviceCredentials().List(ctx, database.DefaultClientID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "enc:v1:community", got[0].SNMPCommunityCT)
	require.Equal(t, polling.CredentialKindV2c, got[0].Kind)
}

// TestDeviceCredentialsListEmptyForUnknownClient — an empty result is not an
// error. Discovery must be able to tell "this client has no credentials" from
// "the database is down", because the first means skip SNMP and the second
// means stop.
func TestDeviceCredentialsListEmptyForUnknownClient(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	got, err := db.DeviceCredentials().List(context.Background(), "no-such-client")
	require.NoError(t, err)
	require.Empty(t, got)
}
