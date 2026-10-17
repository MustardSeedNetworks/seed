package database_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

func TestDeviceImages(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.DeviceConfigs()

	at := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	image := func(name string, i int) *deviceconfig.Image {
		return &deviceconfig.Image{
			ClientID: database.DefaultClientID, Name: name, Platform: deviceconfig.PlatformCiscoIOS, Size: 42,
			SHA256: strings.Repeat("a", 64), SHA512: strings.Repeat("b", 128), UploadedBy: "admin",
			UploadedAt: at.Add(time.Duration(i) * time.Minute),
		}
	}
	first, second := image("cat9k-17.9.bin", 0), image("cat9k-17.12.bin", 1)
	require.NoError(t, repo.InsertImage(ctx, first))
	require.NoError(t, repo.InsertImage(ctx, second))
	require.Regexp(t, `^dim-[0-9a-f]{12}$`, first.ID)

	require.ErrorIs(t, repo.InsertImage(ctx, image("cat9k-17.9.bin", 2)), deviceconfig.ErrImageExists)
	nxos := image("n9k.bin", 3)
	nxos.Platform = deviceconfig.PlatformCiscoNXOS
	require.ErrorContains(t, repo.InsertImage(ctx, nxos), "CHECK constraint failed", "NX-OS images are refused")

	list, err := repo.ListImages(ctx, database.DefaultClientID)
	require.NoError(t, err)
	require.Equal(t, []deviceconfig.Image{*second, *first}, list, "newest first, read back whole")
	other, err := repo.ListImages(ctx, "tenant-x")
	require.NoError(t, err)
	require.Empty(t, other)

	got, err := repo.GetImage(ctx, database.DefaultClientID, first.ID)
	require.NoError(t, err)
	require.Equal(t, *first, got)
	_, err = repo.GetImage(ctx, "tenant-x", first.ID)
	require.ErrorIs(t, err, deviceconfig.ErrNotFound, "another client's image is not found")

	require.ErrorIs(t, repo.DeleteImage(ctx, "tenant-x", first.ID), deviceconfig.ErrNotFound)
	require.NoError(t, repo.DeleteImage(ctx, database.DefaultClientID, first.ID))
	require.ErrorIs(t, repo.DeleteImage(ctx, database.DefaultClientID, first.ID), deviceconfig.ErrNotFound)
}
