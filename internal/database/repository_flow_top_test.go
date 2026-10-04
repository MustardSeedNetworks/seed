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

// The fixture's hosts. D is IPv6 so the pair ordering is exercised across
// families: "192.0.2.3" sorts before "2001:db8::4" as text.
const (
	hostA = "10.0.0.1"
	hostB = "10.0.0.2"
	hostC = "192.0.2.3"
	hostD = "2001:db8::4"
)

// flowDay is the fixture's day.
func flowDay() time.Time { return time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC) }

func fixtureFlow(src, dst string, proto uint8, bytes, pkts uint64, end time.Time) flow.Record {
	return flow.Record{
		Exporter: netip.MustParseAddr("192.0.2.254"), Format: flow.FormatIPFIX,
		Start: end.Add(-time.Second), End: end,
		SrcAddr: netip.MustParseAddr(src), DstAddr: netip.MustParseAddr(dst),
		Protocol: proto, Bytes: bytes, Packets: pkts,
	}
}

// seedTopFixture stores six flows ending 10:05-11:30 on flowDay and one on the
// day before, rolls up every hour and both days they touch, and returns the
// window [10:00, 12:00) the six fall in.
func seedTopFixture(t *testing.T, db *database.DB) (time.Time, time.Time) {
	t.Helper()
	ctx := context.Background()
	at := func(h, m int) time.Time {
		return flowDay().Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}
	require.NoError(t, db.FlowRecords().InsertFlows(ctx, []flow.Record{
		fixtureFlow(hostA, hostB, 6, 1000, 10, at(10, 5)),
		fixtureFlow(hostB, hostA, 6, 500, 5, at(10, 10)),
		fixtureFlow(hostA, hostC, 17, 3000, 3, at(10, 20)),
		fixtureFlow(hostC, hostD, 6, 200, 50, at(11, 15)),
		fixtureFlow(hostD, hostD, 1, 100, 1, at(11, 20)),
		fixtureFlow(hostA, hostB, 17, 40, 1, at(11, 30)),
		// Outside every window below.
		fixtureFlow(hostA, hostC, 17, 9999, 9, at(-1, 59)),
	}))

	src := database.NewFlowRollupSource(db)
	for _, h := range []int{-1, 10, 11} {
		// Twice: a rollup re-run replaces its buckets rather than adding.
		for range 2 {
			_, err := src.RollupHour(ctx, at(h, 0))
			require.NoError(t, err)
		}
	}
	for _, d := range []time.Time{flowDay().AddDate(0, 0, -1), flowDay()} {
		_, err := src.RollupDay(ctx, d)
		require.NoError(t, err)
	}
	return at(10, 0), at(12, 0)
}

// The expected lists are worked by hand from the fixture:
//
//	A: 1000+500+3000+40 = 4540 B, 10+5+3+1 = 19 pkts
//	B: 1000+500+40      = 1540 B, 10+5+1   = 16 pkts
//	C: 3000+200         = 3200 B, 3+50     = 53 pkts
//	D: 200+100          =  300 B, 50+1     = 51 pkts (D->D counted once)
func TestFlowTopTalkers(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	from, to := seedTopFixture(t, db)
	repo := db.FlowRecords()

	byBytes := []database.FlowTalker{
		{Addr: hostA, Bytes: 4540, Packets: 19},
		{Addr: hostC, Bytes: 3200, Packets: 53},
		{Addr: hostB, Bytes: 1540, Packets: 16},
		{Addr: hostD, Bytes: 300, Packets: 51},
	}
	byPackets := []database.FlowTalker{byBytes[1], byBytes[3], byBytes[0], byBytes[2]}

	for _, tc := range []struct {
		name     string
		tier     database.FlowTier
		from, to time.Time
	}{
		{"raw", database.FlowTierRaw, from, to},
		{"hourly", database.FlowTierHourly, from, to},
		// The daily tier answers at day granularity; the fixture's day holds
		// exactly the six flows. The window ends mid-day, as one ending now
		// does: the day it ends in is still read.
		{"daily", database.FlowTierDaily, flowDay(), flowDay().Add(13 * time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.TopTalkers(
				context.Background(),
				"default",
				tc.tier,
				tc.from,
				tc.to,
				database.FlowRankBytes,
				10,
			)
			require.NoError(t, err)
			require.Equal(t, byBytes, got)

			got, err = repo.TopTalkers(
				context.Background(),
				"default",
				tc.tier,
				tc.from,
				tc.to,
				database.FlowRankPackets,
				10,
			)
			require.NoError(t, err)
			require.Equal(t, byPackets, got)

			got, err = repo.TopTalkers(
				context.Background(),
				"default",
				tc.tier,
				tc.from,
				tc.to,
				database.FlowRankBytes,
				2,
			)
			require.NoError(t, err)
			require.Equal(t, byBytes[:2], got)

			got, err = repo.TopTalkers(
				context.Background(),
				"other",
				tc.tier,
				tc.from,
				tc.to,
				database.FlowRankBytes,
				10,
			)
			require.NoError(t, err)
			require.Empty(t, got, "another client's flows are not this client's")
		})
	}
}

// Conversations, both directions summed per protocol:
//
//	A-B tcp  1000+500 = 1500 B, 15 pkts
//	A-C udp           = 3000 B,  3 pkts
//	C-D tcp           =  200 B, 50 pkts
//	D-D icmp          =  100 B,  1 pkt
//	A-B udp           =   40 B,  1 pkt
func TestFlowTopConversations(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	from, to := seedTopFixture(t, db)
	repo := db.FlowRecords()

	abTCP := database.FlowConversation{AddrA: hostA, AddrB: hostB, Protocol: 6, Bytes: 1500, Packets: 15}
	acUDP := database.FlowConversation{AddrA: hostA, AddrB: hostC, Protocol: 17, Bytes: 3000, Packets: 3}
	cdTCP := database.FlowConversation{AddrA: hostC, AddrB: hostD, Protocol: 6, Bytes: 200, Packets: 50}
	ddICMP := database.FlowConversation{AddrA: hostD, AddrB: hostD, Protocol: 1, Bytes: 100, Packets: 1}
	abUDP := database.FlowConversation{AddrA: hostA, AddrB: hostB, Protocol: 17, Bytes: 40, Packets: 1}

	for _, tier := range []database.FlowTier{database.FlowTierRaw, database.FlowTierHourly} {
		got, err := repo.TopConversations(context.Background(), "default", tier, from, to, database.FlowRankBytes, 10)
		require.NoError(t, err)
		require.Equal(t, []database.FlowConversation{acUDP, abTCP, cdTCP, ddICMP, abUDP}, got)

		got, err = repo.TopConversations(context.Background(), "default", tier, from, to, database.FlowRankPackets, 3)
		require.NoError(t, err)
		require.Equal(t, []database.FlowConversation{cdTCP, abTCP, acUDP}, got)
	}
}

func TestFlowRollupSaturatesAndPurges(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	hour := flowDay().Add(10 * time.Hour)

	// Two records whose sum passes int64: SUM would fail the pass for the
	// hour on every run; the rollup pins the bucket instead.
	require.NoError(t, db.FlowRecords().InsertFlows(ctx, []flow.Record{
		fixtureFlow(hostA, hostB, 6, math.MaxUint64, 1, hour.Add(time.Minute)),
		fixtureFlow(hostA, hostB, 6, 1000, 1, hour.Add(2*time.Minute)),
	}))
	src := database.NewFlowRollupSource(db)
	n, err := src.RollupHour(ctx, hour)
	require.NoError(t, err)
	require.Equal(t, 2, n, "one conversation row and one application row")
	_, err = src.RollupDay(ctx, flowDay())
	require.NoError(t, err)

	got, err := db.FlowRecords().TopTalkers(ctx, "default", database.FlowTierHourly,
		hour, hour.Add(time.Hour), database.FlowRankBytes, 1)
	require.NoError(t, err)
	require.Equal(t, []database.FlowTalker{{Addr: hostA, Bytes: math.MaxInt64, Packets: 2}}, got)
	apps, err := db.FlowRecords().TopApplications(ctx, "default", database.FlowTierDaily,
		hour, hour.Add(time.Hour), database.FlowRankBytes, 1)
	require.NoError(t, err)
	require.Equal(t, []database.FlowApplication{{Name: "unknown", Bytes: math.MaxInt64, Packets: 2}}, apps)

	purged, err := src.PurgeHourly(ctx, hour.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(2), purged)
	purged, err = src.PurgeDaily(ctx, flowDay().AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Equal(t, int64(2), purged)
}
