package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

func saveSSHCredential(t *testing.T, db *database.DB, clientID string) string {
	t.Helper()
	c := &polling.Credentials{
		ClientID: clientID, Name: "backup login", SSHUser: "backup", SSHPasswordCT: "enc:v1:pw",
	}
	require.NoError(t, db.DeviceCredentials().Upsert(context.Background(), c))
	return c.ID
}

func newDeviceConfigTarget(credID, host string) *deviceconfig.Target {
	return &deviceconfig.Target{
		ClientID: database.DefaultClientID, Name: "core-" + host, Host: host, Port: 22,
		Platform: deviceconfig.PlatformCiscoIOS, CredentialsID: credID,
	}
}

func TestDeviceCredentialsStoresAnSSHLogin(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	id := saveSSHCredential(t, db, database.DefaultClientID)

	got, err := db.DeviceCredentials().Get(context.Background(), id, database.DefaultClientID)
	require.NoError(t, err)
	require.Equal(t, polling.CredentialKindSSH, got.Kind)
	require.Equal(t, "backup", got.SSHUser)
	require.Equal(t, "enc:v1:pw", got.SSHPasswordCT)
	require.Empty(t, got.SNMPCommunityCT)
	require.Empty(t, got.SecurityLevel)
}

func TestDeviceCredentialsRefusesMixedOrIncompleteSSH(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	for name, c := range map[string]*polling.Credentials{
		"ssh and community": {Name: "x", SSHUser: "u", SSHPasswordCT: "enc:v1:p", SNMPCommunityCT: "enc:v1:c"},
		"ssh and v3 user":   {Name: "x", SSHUser: "u", SSHPasswordCT: "enc:v1:p", SNMPv3User: "v3"},
		"no password":       {Name: "x", SSHUser: "u"},
		"no user":           {Name: "x", SSHPasswordCT: "enc:v1:p"},
	} {
		c.ClientID = database.DefaultClientID
		require.ErrorIs(t, db.DeviceCredentials().Upsert(ctx, c), database.ErrCredentialAmbiguous, name)
	}

	// The schema is the backstop: a plaintext password is not ciphertext.
	_, err := db.Exec(ctx, `INSERT INTO device_credentials
		(id, client_id, name, kind, ssh_user, ssh_password_enc, created_at, updated_at)
		VALUES ('cred-p', 'default', 'x', 'ssh', 'u', CAST('hunter2' AS BLOB), 't', 't')`)
	require.ErrorContains(t, err, "CHECK constraint failed")
}

func TestDeviceConfigTargetsAreScopedAndUnique(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.DeviceConfigs()

	require.NoError(t, db.Clients().Create(ctx, &database.Client{ID: "tenant-b", Name: "B", Slug: "tenant-b"}))
	credA := saveSSHCredential(t, db, database.DefaultClientID)
	credB := saveSSHCredential(t, db, "tenant-b")

	tgt := newDeviceConfigTarget(credA, "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))
	require.Regexp(t, `^dct-[0-9a-f]{12}$`, tgt.ID)

	got, err := repo.GetTarget(ctx, database.DefaultClientID, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, "10.0.0.1", got.Host)
	require.Equal(t, deviceconfig.PlatformCiscoIOS, got.Platform)

	_, err = repo.GetTarget(ctx, "tenant-b", tgt.ID)
	require.ErrorIs(t, err, deviceconfig.ErrNotFound, "another client's target must not resolve")
	list, err := repo.ListTargets(ctx, "tenant-b")
	require.NoError(t, err)
	require.Empty(t, list)

	require.ErrorIs(t, repo.SaveTarget(ctx, newDeviceConfigTarget(credA, "10.0.0.1")),
		database.ErrDeviceConfigTargetConflict)

	// The (client, credential) foreign key: tenant-b's credential cannot back
	// a default-client target.
	require.Error(t, repo.SaveTarget(ctx, newDeviceConfigTarget(credB, "10.0.0.2")))

	require.ErrorIs(t, db.DeviceCredentials().Delete(ctx, database.DefaultClientID, credA),
		database.ErrCredentialInUse, "a credential a backup target uses cannot be deleted")
}

func TestDeviceConfigHostKeyPinsOnceAndClears(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.DeviceConfigs()

	tgt := newDeviceConfigTarget(saveSSHCredential(t, db, database.DefaultClientID), "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))

	require.NoError(t, repo.PinHostKey(ctx, database.DefaultClientID, tgt.ID, "SHA256:first"))
	require.NoError(t, repo.PinHostKey(ctx, database.DefaultClientID, tgt.ID, "SHA256:second"))
	got, err := repo.GetTarget(ctx, database.DefaultClientID, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, "SHA256:first", got.HostKeySHA256, "a pin must never be overwritten by a later run")

	require.NoError(t, repo.ClearHostKey(ctx, database.DefaultClientID, tgt.ID))
	got, err = repo.GetTarget(ctx, database.DefaultClientID, tgt.ID)
	require.NoError(t, err)
	require.Empty(t, got.HostKeySHA256)

	require.ErrorIs(t, repo.ClearHostKey(ctx, "tenant-x", tgt.ID), deviceconfig.ErrNotFound)
}

func TestDeviceConfigBackupsRecordAttemptsAndCascade(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.DeviceConfigs()

	tgt := newDeviceConfigTarget(saveSSHCredential(t, db, database.DefaultClientID), "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))

	t0 := time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)
	ok := &deviceconfig.Backup{
		ClientID: database.DefaultClientID, TargetID: tgt.ID, TakenAt: t0,
		Status: deviceconfig.StatusOK, Config: "hostname core\n", SHA256: "abc",
	}
	failed := &deviceconfig.Backup{
		ClientID: database.DefaultClientID, TargetID: tgt.ID, TakenAt: t0.Add(time.Hour),
		Status: deviceconfig.StatusAuthFailed, Error: "authentication failed",
	}
	require.NoError(t, repo.InsertBackup(ctx, ok))
	require.NoError(t, repo.InsertBackup(ctx, failed))

	list, err := repo.ListBackups(ctx, database.DefaultClientID, tgt.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, failed.ID, list[0].ID, "newest first")
	require.Equal(t, deviceconfig.StatusAuthFailed, list[0].Status)
	require.Empty(t, list[1].Config, "lists carry no configuration")
	require.Equal(t, "abc", list[1].SHA256)

	got, err := repo.GetBackup(ctx, database.DefaultClientID, ok.ID)
	require.NoError(t, err)
	require.Equal(t, "hostname core\n", got.Config)
	require.True(t, got.TakenAt.Equal(t0))
	_, err = repo.GetBackup(ctx, "tenant-x", ok.ID)
	require.ErrorIs(t, err, deviceconfig.ErrNotFound)

	// An ok attempt without a configuration is not a backup.
	require.Error(t, repo.InsertBackup(ctx, &deviceconfig.Backup{
		ClientID: database.DefaultClientID, TargetID: tgt.ID, TakenAt: t0, Status: deviceconfig.StatusOK,
	}))

	require.NoError(t, repo.DeleteTarget(ctx, database.DefaultClientID, tgt.ID))
	_, err = repo.GetBackup(ctx, database.DefaultClientID, ok.ID)
	require.ErrorIs(t, err, deviceconfig.ErrNotFound, "history goes with its target")
}
