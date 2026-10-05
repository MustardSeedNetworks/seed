package database_test

import (
	"context"
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/capture/capturetest"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/listener/voip"
)

type discardSink struct{}

func (discardSink) Publish(context.Context, listener.Event) error { return nil }

// TestVoIPListenerWritesThrough runs the producer over a replayed call into
// the real table: 10 s of clean G.711 from 192.0.2.10 to 192.0.2.20.
func TestVoIPListenerWritesThrough(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	var frames []capturetest.Frame
	for i := range 500 {
		rtp := make([]byte, 172)
		rtp[0] = 0x80
		binary.BigEndian.PutUint16(rtp[2:4], uint16(i))
		binary.BigEndian.PutUint32(rtp[4:8], uint32(160*i))
		binary.BigEndian.PutUint32(rtp[8:12], 0xfeed)
		ip := &layers.IPv4{
			Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
			SrcIP: net.IPv4(192, 0, 2, 10), DstIP: net.IPv4(192, 0, 2, 20),
		}
		udp := &layers.UDP{SrcPort: 40000, DstPort: 30000}
		require.NoError(t, udp.SetNetworkLayerForChecksum(ip))
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
		require.NoError(t, gopacket.SerializeLayers(buf, opts,
			&layers.Ethernet{
				SrcMAC: net.HardwareAddr{2, 0, 0, 0, 0, 1}, DstMAC: net.HardwareAddr{2, 0, 0, 0, 0, 2},
				EthernetType: layers.EthernetTypeIPv4,
			}, ip, udp, gopacket.Payload(rtp)))
		frames = append(frames, capturetest.Frame{
			Data: buf.Bytes(),
			Info: gopacket.CaptureInfo{
				Timestamp:     start.Add(time.Duration(i) * 20 * time.Millisecond),
				CaptureLength: len(buf.Bytes()), Length: len(buf.Bytes()),
			},
		})
	}

	l, err := voip.New(voip.Config{
		Interface: "eth0",
		Opener:    &capturetest.ReplayOpener{LinkType: layers.LinkTypeEthernet, Frames: frames},
		Store:     db.VoIPStreams(),
		Sink:      discardSink{},
		Now:       func() time.Time { return start.Add(time.Hour) },
	})
	require.NoError(t, err)
	ctx := t.Context()
	require.NoError(t, l.Start(ctx))
	require.Eventually(t, func() bool { return l.Stats().Reports == 1 }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, l.Stop(ctx))

	var (
		src, dst, codec, started, ended string
		ssrc, expected, received        int64
		loss, mos                       float64
	)
	require.NoError(t, db.QueryRow(ctx, `
		SELECT src_addr, dst_addr, ssrc, codec, started_at, ended_at,
		       packets_expected, packets_received, loss_pct, mos
		FROM voip_streams`).Scan(&src, &dst, &ssrc, &codec, &started, &ended, &expected, &received, &loss, &mos))
	require.Equal(t,
		[]any{
			"192.0.2.10:40000", "192.0.2.20:30000", int64(0xfeed), "PCMU",
			"2026-10-05T12:00:00.180Z", "2026-10-05T12:00:09.980Z", int64(491), int64(491),
		},
		[]any{src, dst, ssrc, codec, started, ended, expected, received})
	require.Zero(t, loss)
	require.InDelta(t, 4.4, mos, 0.05)
}

func TestVoIPStreamsPurge(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.VoIPStreams()

	// RunCleanup cuts off against the wall clock.
	now := time.Now().UTC()
	report := voip.Report{
		Interface: "eth0", Src: netip.MustParseAddrPort("192.0.2.10:40000"),
		Dst: netip.MustParseAddrPort("192.0.2.20:30000"), SSRC: 1, Codec: "PCMA",
		Start: now, End: now.Add(time.Minute), Expected: 3000, Received: 3000, MOS: 4.4, RFactor: 92,
	}
	old := report
	old.Start = now.AddDate(0, 0, -100)
	require.NoError(t, repo.InsertVoIPReports(ctx, []voip.Report{report, old}))
	require.NoError(t, repo.InsertVoIPReports(ctx, nil))

	res, err := db.RunCleanup(ctx, database.RetentionPolicy{VoIPDays: 90})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.VoIPStreamsDeleted)
	var left int
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM voip_streams`).Scan(&left))
	require.Equal(t, 1, left)
}
