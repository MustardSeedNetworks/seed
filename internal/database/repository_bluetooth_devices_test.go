package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// TestBluetoothDevicesPersistAcrossScans is P-E3's acceptance at the store: a
// device heard again keeps the id and first-seen time of its first scan and
// the name an earlier scan heard, and takes this scan's signal.
func TestBluetoothDevicesPersistAcrossScans(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.BluetoothDevices()

	first := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	scan := func(id string, at time.Time, name string, rssi int) []discovery.BluetoothDevice {
		return []discovery.BluetoothDevice{{
			ID: id, Address: "74:DA:CF:00:00:01", AddressType: discovery.BluetoothAddressPublic,
			Name: name, Vendor: "Sensor Co", Type: discovery.BluetoothTypeBLE,
			DeviceClass: discovery.BluetoothClassHealth, ManufacturerID: 0x004C,
			RSSI: rssi, TxPower: 8, FirstSeen: at, LastSeen: at,
		}}
	}
	require.NoError(t, repo.RememberBluetoothDevices(ctx, scan("id-1", first, "Sensor 12", -74)))

	later := first.Add(time.Hour)
	again := scan("id-2", later, "", -60)
	require.NoError(t, repo.RememberBluetoothDevices(ctx, again))
	require.Equal(t, "id-1", again[0].ID)
	require.True(t, again[0].FirstSeen.Equal(first), "first seen = %v, want %v", again[0].FirstSeen, first)

	var (
		count            int
		name, lastSeen   string
		rssi, manufactID int
	)
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*), name, rssi, manufacturer_id, last_seen
		FROM bluetooth_devices`).Scan(&count, &name, &rssi, &manufactID, &lastSeen))
	require.Equal(t, []any{1, "Sensor 12", -60, 0x004C, "2026-10-16T10:00:00.000Z"},
		[]any{count, name, rssi, manufactID, lastSeen})
	require.NoError(t, repo.RememberBluetoothDevices(ctx, nil))
}

func TestBluetoothDevicesPurge(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	// RunCleanup cuts off against the wall clock.
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -100)
	require.NoError(t, db.BluetoothDevices().RememberBluetoothDevices(ctx, []discovery.BluetoothDevice{
		{ID: "recent", Address: "74:DA:CF:00:00:01", Type: discovery.BluetoothTypeBLE, FirstSeen: now, LastSeen: now},
		{ID: "gone", Address: "74:DA:CF:00:00:02", Type: discovery.BluetoothTypeClassic, FirstSeen: old, LastSeen: old},
	}))

	res, err := db.RunCleanup(ctx, database.RetentionPolicy{InactiveDeviceDays: 90})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.BluetoothDevicesDeleted)
	var left string
	require.NoError(t, db.QueryRow(ctx, `SELECT group_concat(id) FROM bluetooth_devices`).Scan(&left))
	require.Equal(t, "recent", left)
}
