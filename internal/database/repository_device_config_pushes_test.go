package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

func TestDeviceConfigPushCredentials(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.DeviceConfigs()

	backup := saveSSHCredential(t, db, database.DefaultClientID)
	push := saveSSHCredential(t, db, database.DefaultClientID)
	tgt := newDeviceConfigTarget(backup, "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))

	require.NoError(t, repo.SetPushCredentials(ctx, database.DefaultClientID, tgt.ID, push))
	got, err := repo.GetTarget(ctx, database.DefaultClientID, tgt.ID)
	require.NoError(t, err)
	require.Equal(t, push, got.PushCredentialsID)

	require.Error(t, db.DeviceCredentials().Delete(ctx, database.DefaultClientID, push),
		"a push credential in use cannot be deleted")
	require.ErrorContains(t, repo.SetPushCredentials(ctx, database.DefaultClientID, tgt.ID, backup),
		"CHECK constraint failed", "the backup login is never the push login")
	require.ErrorIs(t, repo.SetPushCredentials(ctx, "tenant-x", tgt.ID, push), deviceconfig.ErrNotFound)

	require.NoError(t, repo.SetPushCredentials(ctx, database.DefaultClientID, tgt.ID, ""))
	got, err = repo.GetTarget(ctx, database.DefaultClientID, tgt.ID)
	require.NoError(t, err)
	require.Empty(t, got.PushCredentialsID)
}

func TestDeviceConfigPushPlans(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.DeviceConfigs()

	tgt := newDeviceConfigTarget(saveSSHCredential(t, db, database.DefaultClientID), "10.0.0.1")
	require.NoError(t, repo.SaveTarget(ctx, tgt))

	created := time.Date(2026, 10, 17, 3, 0, 0, 0, time.UTC)
	plan := func(i int) *deviceconfig.Plan {
		at := created.Add(time.Duration(i) * time.Minute)
		return &deviceconfig.Plan{
			ClientID: database.DefaultClientID, TargetID: tgt.ID, CreatedBy: "admin",
			CreatedAt: at, ExpiresAt: at.Add(deviceconfig.PlanLifetime), Mode: deviceconfig.PushMerge,
			DiffSource: deviceconfig.DiffPredicted, BackupID: "dcb-000000000001", RunningSHA256: "aa",
			DiffSHA256: "bb", Added: 2, Removed: 1, CandidateCT: "enc:v1:cand", DiffCT: "enc:v1:diff",
		}
	}
	first := plan(0)
	require.NoError(t, repo.InsertPlan(ctx, first))
	require.Regexp(t, `^dcp-[0-9a-f]{12}$`, first.ID)

	got, err := repo.GetPlan(ctx, database.DefaultClientID, first.ID)
	require.NoError(t, err)
	require.Equal(t, *first, got)
	_, err = repo.GetPlan(ctx, "tenant-x", first.ID)
	require.ErrorIs(t, err, deviceconfig.ErrNotFound)

	for i := 1; i <= 3; i++ {
		require.NoError(t, repo.InsertPlan(ctx, plan(i)))
	}
	require.NoError(t, repo.PrunePlans(ctx, database.DefaultClientID, tgt.ID, 2))
	list, err := repo.ListPlans(ctx, database.DefaultClientID, tgt.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, created.Add(3*time.Minute), list[0].CreatedAt, "newest first")
	require.Empty(t, list[0].CandidateCT, "a list carries no ciphertext")
	require.Empty(t, list[0].DiffCT)

	plain := plan(9)
	plain.DiffCT = "the diff in the clear"
	require.ErrorContains(t, repo.InsertPlan(ctx, plain), "CHECK constraint failed", "plaintext is refused")
	empty := plan(9)
	empty.Added, empty.Removed = 0, 0
	require.ErrorContains(t, repo.InsertPlan(ctx, empty), "CHECK constraint failed", "a plan changes something")

	require.NoError(t, repo.DeleteTarget(ctx, database.DefaultClientID, tgt.ID))
	list, err = repo.ListPlans(ctx, database.DefaultClientID, tgt.ID, 10)
	require.NoError(t, err)
	require.Empty(t, list, "plans go with their target")
}
