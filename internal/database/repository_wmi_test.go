package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

func saveWinRMCredential(t *testing.T, db *database.DB, clientID string) string {
	t.Helper()
	c := &polling.Credentials{
		ClientID: clientID, Name: "inventory login", WinRMUser: `LAB\inventory`, WinRMPasswordCT: "enc:v1:pw",
	}
	require.NoError(t, db.DeviceCredentials().Upsert(context.Background(), c))
	return c.ID
}

func newWMITarget(credID, host string) *wmi.Target {
	return &wmi.Target{
		ClientID: database.DefaultClientID, Name: "win-" + host, Host: host, Port: wmi.DefaultPort,
		CredentialsID: credID,
	}
}

// Listener certificate fingerprints: 64 uppercase hex digits.
const (
	certA = "ABABABABABABABABABABABABABABABABABABABABABABABABABABABABABABABAB"
	certB = "CDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCDCD"
)

func TestDeviceCredentialsStoresAWinRMLogin(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	id := saveWinRMCredential(t, db, database.DefaultClientID)

	got, err := db.DeviceCredentials().Get(context.Background(), id, database.DefaultClientID)
	require.NoError(t, err)
	require.Equal(t, polling.CredentialKindWinRM, got.Kind)
	require.Equal(t, `LAB\inventory`, got.WinRMUser)
	require.Equal(t, "enc:v1:pw", got.WinRMPasswordCT)
	require.Empty(t, got.SSHUser)
	require.Empty(t, got.SecurityLevel)
}

func TestDeviceCredentialsRefusesMixedOrIncompleteWinRM(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	for name, c := range map[string]*polling.Credentials{
		"winrm and ssh":       {Name: "x", WinRMUser: "u", WinRMPasswordCT: "enc:v1:p", SSHUser: "u", SSHPasswordCT: "enc:v1:p"},
		"winrm and community": {Name: "x", WinRMUser: "u", WinRMPasswordCT: "enc:v1:p", SNMPCommunityCT: "enc:v1:c"},
		"no password":         {Name: "x", WinRMUser: "u"},
		"no user":             {Name: "x", WinRMPasswordCT: "enc:v1:p"},
	} {
		c.ClientID = database.DefaultClientID
		require.ErrorIs(t, db.DeviceCredentials().Upsert(ctx, c), database.ErrCredentialAmbiguous, name)
	}

	for name, stmt := range map[string]string{
		"plaintext password": `INSERT INTO device_credentials
			(id, client_id, name, kind, winrm_user, winrm_password_enc, created_at, updated_at)
			VALUES ('cred-p', 'default', 'x', 'winrm', 'u', CAST('hunter2' AS BLOB), 't', 't')`,
		"winrm columns on an ssh row": `INSERT INTO device_credentials
			(id, client_id, name, kind, ssh_user, ssh_password_enc, winrm_user, created_at, updated_at)
			VALUES ('cred-q', 'default', 'x', 'ssh', 'u', CAST('enc:v1:p' AS BLOB), 'u', 't', 't')`,
	} {
		_, err := db.Exec(ctx, stmt)
		require.ErrorContains(t, err, "CHECK constraint failed", name)
	}
}

func TestWMITargetsAreScopedAndUnique(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.WMI()

	require.NoError(t, db.Clients().Create(ctx, &database.Client{ID: "tenant-b", Name: "B", Slug: "tenant-b"}))
	credA := saveWinRMCredential(t, db, database.DefaultClientID)
	credB := saveWinRMCredential(t, db, "tenant-b")

	tgt := newWMITarget(credA, "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))
	require.Regexp(t, `^wmt-[0-9a-f]{12}$`, tgt.ID)

	got, err := repo.GetTarget(ctx, database.DefaultClientID, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, "10.0.0.1", got.Host)
	require.Equal(t, wmi.DefaultPort, got.Port)
	require.Empty(t, got.LastStatus)
	require.True(t, got.LastAttemptAt.IsZero())

	_, err = repo.GetTarget(ctx, "tenant-b", tgt.ID)
	require.ErrorIs(t, err, wmi.ErrNotFound, "another client's target must not resolve")
	list, err := repo.ListTargets(ctx, "tenant-b")
	require.NoError(t, err)
	require.Empty(t, list)

	require.ErrorIs(t, repo.SaveTarget(ctx, newWMITarget(credA, "10.0.0.1")), database.ErrWMITargetConflict)
	require.Error(t, repo.SaveTarget(ctx, newWMITarget(credB, "10.0.0.2")),
		"another client's credential cannot back this client's target")
	require.ErrorIs(t, db.DeviceCredentials().Delete(ctx, database.DefaultClientID, credA),
		database.ErrCredentialInUse, "a credential an inventory target uses cannot be deleted")
	require.ErrorIs(t, repo.DeleteTarget(ctx, "tenant-b", tgt.ID), wmi.ErrNotFound)
}

func TestWMICertIsPendingUntilConfirmed(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.WMI()
	client := database.DefaultClientID

	tgt := newWMITarget(saveWinRMCredential(t, db, client), "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))

	require.NoError(t, repo.SetPendingCert(ctx, client, tgt.ID, certA))
	require.ErrorIs(t, repo.ConfirmCert(ctx, client, tgt.ID, certB), wmi.ErrNotFound,
		"only the pending certificate can be confirmed")
	require.NoError(t, repo.ConfirmCert(ctx, client, tgt.ID, certA))

	got, err := repo.GetTarget(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, certA, got.CertSHA256)
	require.Empty(t, got.PendingCertSHA256)

	require.NoError(t, repo.SetPendingCert(ctx, client, tgt.ID, certB))
	got, err = repo.GetTarget(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Empty(t, got.PendingCertSHA256, "a pinned target records no pending certificate")

	require.NoError(t, repo.ClearCert(ctx, client, tgt.ID))
	got, err = repo.GetTarget(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Empty(t, got.CertSHA256)

	_, err = db.Exec(ctx, `UPDATE wmi_targets SET cert_sha256 = 'sha256:abc' WHERE id = ?`, tgt.ID)
	require.ErrorContains(t, err, "CHECK constraint failed", "a fingerprint is 64 uppercase hex digits")
}

func TestWMIRecordAttemptReplacesTheInventory(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.WMI()
	client := database.DefaultClientID

	tgt := newWMITarget(saveWinRMCredential(t, db, client), "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))

	_, err := repo.GetInventory(ctx, client, tgt.ID)
	require.ErrorIs(t, err, wmi.ErrNotFound, "no collection has succeeded yet")

	first := time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)
	require.NoError(t, repo.RecordAttempt(ctx, wmi.Attempt{
		ClientID: client, TargetID: tgt.ID, At: first, Status: wmi.StatusOK,
		Inventory: &wmi.Inventory{OS: wmi.OperatingSystem{Caption: "Windows 11 Pro"}},
	}))
	require.NoError(t, repo.RecordAttempt(ctx, wmi.Attempt{
		ClientID: client, TargetID: tgt.ID, At: first.Add(time.Hour), Status: wmi.StatusUnreachable,
		Error: "host unreachable: connection refused",
	}))

	got, err := repo.GetTarget(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, wmi.StatusUnreachable, got.LastStatus)
	require.Equal(t, first.Add(time.Hour), got.LastAttemptAt)
	require.Contains(t, got.LastError, "connection refused")

	inv, err := repo.GetInventory(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, first, inv.CollectedAt, "a failed attempt keeps the last inventory")
	require.Equal(t, "Windows 11 Pro", inv.Inventory.OS.Caption)

	require.NoError(t, repo.RecordAttempt(ctx, wmi.Attempt{
		ClientID: client, TargetID: tgt.ID, At: first.Add(2 * time.Hour), Status: wmi.StatusOK,
		Inventory: &wmi.Inventory{OS: wmi.OperatingSystem{Caption: "Windows Server 2025"}},
	}))
	inv, err = repo.GetInventory(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, "Windows Server 2025", inv.Inventory.OS.Caption)
	got, err = repo.GetTarget(ctx, client, tgt.ID)
	require.NoError(t, err)
	require.Empty(t, got.LastError)

	require.ErrorIs(t, repo.RecordAttempt(ctx, wmi.Attempt{
		ClientID: "tenant-b", TargetID: tgt.ID, At: first, Status: wmi.StatusOK, Inventory: &wmi.Inventory{},
	}), wmi.ErrNotFound, "an attempt cannot be recorded against another client's target")

	require.NoError(t, repo.DeleteTarget(ctx, client, tgt.ID))
	var left int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM wmi_inventories`).Scan(&left))
	require.Zero(t, left, "deleting a target deletes its inventory")
}
