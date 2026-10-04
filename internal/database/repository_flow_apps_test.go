package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/appid"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

func portFlow(src, dst string, proto uint8, sport, dport uint16, bytes, pkts uint64, end time.Time) flow.Record {
	r := fixtureFlow(src, dst, proto, bytes, pkts, end)
	r.SrcPort, r.DstPort = sport, dport
	return r
}

// The expected totals are worked by hand from the fixture: the request and
// response of one HTTPS session are both https (1000+500 B, 10+5 pkts);
// TCP 9999 is unknown under the builtin table and historian once the
// operator names it, and the flow stored before that keeps its name.
func TestFlowTopApplications(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.FlowRecords()
	at := func(h, m int) time.Time {
		return flowDay().Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}

	require.NoError(t, repo.InsertFlows(ctx, []flow.Record{
		portFlow(hostA, hostB, 6, 51514, 443, 1000, 10, at(10, 5)),
		portFlow(hostB, hostA, 6, 443, 51514, 500, 5, at(10, 10)),
		portFlow(hostA, hostC, 17, 40000, 53, 300, 3, at(10, 20)),
		portFlow(hostC, hostD, 6, 50001, 9999, 200, 2, at(11, 15)),
		portFlow(hostD, hostA, 1, 0, 0, 100, 1, at(11, 20)),
	}))

	custom, err := appid.New(append(appid.Builtin().Signatures(), appid.Signature{
		Name: "historian", Description: "Plant historian", Protocol: 6, Ports: []string{"9999"},
	}))
	require.NoError(t, err)
	require.NoError(t, repo.SetAppSignatures(ctx, custom))
	require.NoError(t, repo.InsertFlows(ctx, []flow.Record{
		portFlow(hostC, hostD, 6, 50002, 9999, 400, 4, at(11, 40)),
	}))

	src := database.NewFlowRollupSource(db)
	for _, h := range []int{10, 11} {
		_, rollupErr := src.RollupHour(ctx, at(h, 0))
		require.NoError(t, rollupErr)
	}
	_, err = src.RollupDay(ctx, flowDay())
	require.NoError(t, err)

	byBytes := []database.FlowApplication{
		{Name: "https", Bytes: 1500, Packets: 15},
		{Name: "historian", Bytes: 400, Packets: 4},
		{Name: "dns", Bytes: 300, Packets: 3},
		{Name: appid.Unknown, Bytes: 200, Packets: 2},
		{Name: "icmp", Bytes: 100, Packets: 1},
	}

	for _, tier := range []database.FlowTier{database.FlowTierRaw, database.FlowTierHourly, database.FlowTierDaily} {
		got, readErr := repo.TopApplications(ctx, database.DefaultClientID, tier, at(10, 0), at(12, 0),
			database.FlowRankBytes, 10)
		require.NoError(t, readErr)
		require.Equal(t, byBytes, got, "tier %d by bytes", tier)

		got, readErr = repo.TopApplications(ctx, database.DefaultClientID, tier, at(10, 0), at(12, 0),
			database.FlowRankPackets, 2)
		require.NoError(t, readErr)
		// Packets rank the fixture in the same order; the limit is what this checks.
		require.Equal(t, byBytes[:2], got, "tier %d by packets, limit 2", tier)
	}
}

func TestFlowAppSignaturesPersistAndReset(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	sigs, err := db.FlowRecords().AppSignatures(ctx)
	require.NoError(t, err)
	require.False(t, sigs.Custom)
	require.Equal(t, "https", sigs.Table.Classify(6, 50000, 443))

	custom, err := appid.New([]appid.Signature{{Name: "web", Protocol: 6, Ports: []string{"443"}}})
	require.NoError(t, err)
	require.NoError(t, db.FlowRecords().SetAppSignatures(ctx, custom))

	stored, err := db.Settings().GetValue(ctx, database.SettingKeyFlowAppSignatures)
	require.NoError(t, err)
	reread, err := appid.Parse([]byte(stored))
	require.NoError(t, err)
	require.Equal(t, "web", reread.Classify(6, 50000, 443))

	require.NoError(t, db.FlowRecords().ResetAppSignatures(ctx))
	sigs, err = db.FlowRecords().AppSignatures(ctx)
	require.NoError(t, err)
	require.False(t, sigs.Custom)
	stored, err = db.Settings().GetValue(ctx, database.SettingKeyFlowAppSignatures)
	require.NoError(t, err)
	require.Empty(t, stored)
}

// A stored table that no longer parses fails the insert, which the
// collector logs and counts, rather than quietly naming flows by the builtin.
func TestFlowInsertRefusesUnreadableSignatures(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	require.NoError(t, db.Settings().Set(ctx, database.SettingKeyFlowAppSignatures, `{"signatures":[]}`))

	err := db.FlowRecords().InsertFlows(ctx, []flow.Record{
		portFlow(hostA, hostB, 6, 50000, 443, 1, 1, flowDay()),
	})
	require.ErrorContains(t, err, "stored application signatures")
}
