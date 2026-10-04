package database_test

import (
	"context"
	"math"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

func TestFlowRecordsInsertAndPurge(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.FlowRecords()

	end := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	rec := flow.Record{
		Exporter: netip.MustParseAddr("192.0.2.1"), Format: flow.FormatIPFIX, ObservationDomain: 42,
		Start: end.Add(-1500 * time.Millisecond), End: end,
		SrcAddr: netip.MustParseAddr("2001:db8::1"), DstAddr: netip.MustParseAddr("2001:db8::2"),
		SrcPort: 51000, DstPort: 443, Protocol: 6, TCPFlags: 0x18,
		Bytes: math.MaxUint64, Packets: 12, InputIf: 3, OutputIf: 4,
	}
	old := rec
	old.End = end.Add(-8 * 24 * time.Hour)
	require.NoError(t, repo.InsertFlows(ctx, []flow.Record{rec, old}))
	require.NoError(t, repo.InsertFlows(ctx, nil))

	var (
		exporter, format, start, flowEnd, src, dst string
		domain, sport, dport                       int64
		proto, flags, bytes, pkts, in, out         int64
	)
	require.NoError(t, db.QueryRow(ctx, `
		SELECT exporter, format, observation_domain, flow_start, flow_end,
		       src_addr, dst_addr, src_port, dst_port, protocol, tcp_flags,
		       bytes, packets, input_if, output_if
		FROM flow_records ORDER BY flow_end DESC LIMIT 1`).Scan(
		&exporter, &format, &domain, &start, &flowEnd, &src, &dst, &sport, &dport,
		&proto, &flags, &bytes, &pkts, &in, &out))
	require.Equal(t, []any{"192.0.2.1", "ipfix", int64(42), "2026-10-04T11:59:58.500Z", "2026-10-04T12:00:00.000Z"},
		[]any{exporter, format, domain, start, flowEnd})
	require.Equal(t, []any{"2001:db8::1", "2001:db8::2", int64(51000), int64(443), int64(6), int64(0x18)},
		[]any{src, dst, sport, dport, proto, flags})
	// A counter past int64 is pinned, not stored negative.
	require.Equal(t, []any{int64(math.MaxInt64), int64(12), int64(3), int64(4)}, []any{bytes, pkts, in, out})

	purged, err := database.NewFlowRollupSource(db).PurgeRaw(ctx, end.Add(-7*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), purged)
	var left int
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM flow_records`).Scan(&left))
	require.Equal(t, 1, left)
}
