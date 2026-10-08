package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// TestLogWriterPersistsEntries pins what the log broadcaster's database writer
// stores: every field of a single entry and of a batch, metadata as JSON, and an
// empty batch as a no-op.
func TestLogWriterPersistsEntries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	w := app.NewLogWriter(db)
	ts := time.Date(2026, 10, 8, 2, 30, 0, 0, time.UTC)

	require.NoError(t, w.WriteLog(ctx, &logging.LogEntry{
		Timestamp: ts, Level: "warn", Layer: "backend", Message: "single",
		Component: "probe", RequestID: "req-1", SessionID: "sess-1", DurationMs: 42,
		Metadata: map[string]any{"iface": "eth0"}, Stack: "trace",
	}))
	require.NoError(t, w.WriteBatch(ctx, nil))
	require.NoError(t, w.WriteBatch(ctx, []*logging.LogEntry{
		{Timestamp: ts.Add(time.Second), Level: "info", Layer: "backend", Message: "batch-1"},
		{Timestamp: ts.Add(2 * time.Second), Level: "error", Layer: "frontend", Message: "batch-2"},
	}))

	got, err := db.Logs().List(ctx, database.LogListOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, got, 3)
	byMsg := map[string]*database.LogEntry{}
	for _, e := range got {
		byMsg[e.Message] = e
	}
	single := byMsg["single"]
	require.NotNil(t, single)
	require.True(t, single.Timestamp.Equal(ts))
	require.Equal(t, "warn", single.Level)
	require.Equal(t, "backend", single.Layer)
	require.Equal(t, "probe", single.Component)
	require.Equal(t, "req-1", single.RequestID)
	require.Equal(t, "sess-1", single.SessionID)
	require.Equal(t, int64(42), single.DurationMs)
	require.JSONEq(t, `{"iface":"eth0"}`, single.Metadata)
	require.Equal(t, "trace", single.Stack)
	require.Equal(t, "error", byMsg["batch-2"].Level)
	require.Empty(t, byMsg["batch-1"].Metadata)
}

// TestDeviceWriterKeysOnMAC pins what discovery's database writer stores: one
// row per device, its MAC as the id, refreshed in place when the next sweep
// finds it again, including at a new address (seed#3210).
func TestDeviceWriterKeysOnMAC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	w := app.NewDeviceWriter(db)
	seen := time.Date(2026, 10, 8, 2, 30, 0, 0, time.UTC)
	const mac = "02:00:00:00:00:05"

	require.NoError(t, w.PersistDevices(ctx, nil))
	require.NoError(t, w.PersistDevices(ctx, []*discovery.DiscoveredDevice{{
		IP: "10.0.0.5", MAC: mac, Hostname: "printer",
		Vendor: "HP", OSGuess: "embedded", LastSeen: seen,
	}}))
	first, err := db.Devices().GetByMAC(ctx, mac)
	require.NoError(t, err)

	require.NoError(t, w.PersistDevices(ctx, []*discovery.DiscoveredDevice{{
		IP: "10.0.0.5", MAC: mac, Hostname: "printer-2",
		Vendor: "HP", OSGuess: "embedded", LastSeen: seen.Add(time.Minute),
	}}))
	got, err := db.Devices().GetByIP(ctx, "10.0.0.5")
	require.NoError(t, err)
	require.Equal(t, mac, got.ID)
	require.Equal(t, mac, got.MACAddress)
	require.Equal(t, "printer-2", got.Hostname)
	require.Equal(t, "HP", got.Vendor)
	require.Equal(t, "embedded", got.DeviceType)
	require.True(t, got.IsActive)

	require.NoError(t, w.PersistDevices(ctx, []*discovery.DiscoveredDevice{{
		IP: "10.0.0.6", MAC: mac, Hostname: "printer-2",
		Vendor: "HP", OSGuess: "embedded", LastSeen: seen.Add(2 * time.Minute),
	}}))
	moved, err := db.Devices().GetByMAC(ctx, mac)
	require.NoError(t, err)
	require.Equal(t, mac, moved.ID)
	require.Equal(t, "10.0.0.6", moved.IPAddress)
	require.True(t, moved.FirstSeen.Equal(first.FirstSeen))
	n, err := db.Devices().Count(ctx, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}

// TestDeviceWriterFlushSurvivesOneFailure: a device the store rejects is
// reported, and the devices after it in the same flush are still written.
func TestDeviceWriterFlushSurvivesOneFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	w := app.NewDeviceWriter(db)

	// A row whose id is one MAC but whose address column holds another, as
	// the IP-keyed writer left behind. Persisting the first MAC collides on id.
	require.NoError(t, db.Devices().Create(ctx, &database.Device{
		ID: "02:00:00:00:00:07", IPAddress: "10.0.0.7", MACAddress: "02:00:00:00:00:99",
	}))

	err := w.PersistDevices(ctx, []*discovery.DiscoveredDevice{
		{IP: "10.0.0.8", MAC: "02:00:00:00:00:07"},
		{IP: "10.0.0.9", MAC: "02:00:00:00:00:09"},
	})
	require.ErrorContains(t, err, "persisting device 10.0.0.8")
	got, getErr := db.Devices().GetByMAC(ctx, "02:00:00:00:00:09")
	require.NoError(t, getErr)
	require.Equal(t, "10.0.0.9", got.IPAddress)
}
