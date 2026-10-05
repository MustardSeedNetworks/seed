package database_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/capture/capturetest"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener/microburst"
)

// TestMicroburstListenerWritesThrough runs the producer over a replayed
// capture into the real table: 3 ms of line rate inbound on a 1 Gb/s link.
func TestMicroburstListenerWritesThrough(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	start := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	peer := net.HardwareAddr{0x02, 0, 0, 0, 0, 0x02}
	header := make([]byte, 14)
	copy(header[6:12], peer)
	// A 1514-byte frame is 1538 bytes on the wire: 12.304 µs at 1 Gb/s.
	const wire = 12304 * time.Nanosecond
	var frames []capturetest.Frame
	for at := time.Duration(0); at < 3*time.Millisecond; at += wire {
		frames = append(frames, capturetest.Frame{
			Data: header,
			Info: gopacket.CaptureInfo{Timestamp: start.Add(at), CaptureLength: 14, Length: 1514},
		})
	}

	l, err := microburst.New(microburst.Config{
		Interface:     "eth0",
		MAC:           net.HardwareAddr{0x02, 0, 0, 0, 0, 0x01},
		LinkSpeedMbps: 1000,
		Opener:        &capturetest.ReplayOpener{LinkType: layers.LinkTypeEthernet, Frames: frames},
		Store:         db.Microbursts(),
		Now:           func() time.Time { return start.Add(time.Minute) },
	})
	require.NoError(t, err)
	ctx := t.Context()
	require.NoError(t, l.Start(ctx))
	require.Eventually(t, func() bool { return l.Stats().Bursts == 1 }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, l.Stop(ctx))

	var (
		ts, iface, dir, mode string
		peak                 float64
		durationMs, speed    int64
	)
	require.NoError(t, db.QueryRow(ctx, `
		SELECT timestamp, interface_name, direction, peak_utilization_pct,
		       duration_ms, sampling_mode, link_speed_mbps
		FROM microburst_events`).Scan(&ts, &iface, &dir, &peak, &durationMs, &mode, &speed))
	require.Equal(t, []any{"2026-10-04T12:00:00.000Z", "eth0", "in", int64(3), "capture-1ms", int64(1000)},
		[]any{ts, iface, dir, durationMs, mode, speed})
	require.InDelta(t, 100, peak, 1.5)
}

func TestMicroburstsPurge(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Microbursts()

	// RunCleanup cuts off against the wall clock.
	now := time.Now().UTC()
	burst := microburst.Event{
		Start: now, Interface: "eth0", Direction: microburst.DirectionOut,
		Duration: 2 * time.Millisecond, PeakUtilization: 97.5, LinkSpeedMbps: 1000,
	}
	old := burst
	old.Start = now.AddDate(0, 0, -100)
	require.NoError(t, repo.InsertMicrobursts(ctx, []microburst.Event{burst, old}))
	require.NoError(t, repo.InsertMicrobursts(ctx, nil))

	res, err := db.RunCleanup(ctx, database.RetentionPolicy{MicroburstDays: 90})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.MicroburstsDeleted)
	var left int
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM microburst_events`).Scan(&left))
	require.Equal(t, 1, left)
}
