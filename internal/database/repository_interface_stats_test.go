package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

func TestInterfaceStats(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	target := &polling.Target{
		ClientID: database.DefaultClientID, Name: "core-sw", IPAddress: "10.0.0.1",
		SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: true,
		CollectorChain: []string{"if_table"},
	}
	if err := db.PollingTargets().Create(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	seen := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	if _, err := db.Topology().Upsert(ctx, &topology.Node{
		ID: "node-a", ClientID: database.DefaultClientID, IdentityHash: "h-a", DisplayName: "core-sw", LastSeen: seen,
	}); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	if err := db.Topology().UpsertTargetNode(ctx, database.DefaultClientID, target.ID, "node-a", seen); err != nil {
		t.Fatalf("map target: %v", err)
	}
	for _, iface := range []*topology.Interface{
		{NodeID: "node-a", IfIndex: 1, IfName: "Gi0/1", IfAlias: "uplink", IfOperStatus: 1, SpeedBps: 1e9, LastSeen: seen},
		{NodeID: "node-a", IfIndex: 2, IfDescr: "GigabitEthernet0/2", IfOperStatus: 2, LastSeen: seen},
		{NodeID: "node-a", IfIndex: 3, IfName: "Gi0/3", IfOperStatus: 1, LastSeen: seen},
	} {
		if err := db.Topology().UpsertInterface(ctx, iface); err != nil {
			t.Fatalf("seed interface %d: %v", iface.IfIndex, err)
		}
	}

	util := ifrate.Utilization{In: 12.5, Out: 3}
	if err := db.Metrics().RecordInterfaceRates(ctx, []ifrate.Rate{
		// An older poll of ifIndex 1, which the newer one replaces.
		{ClientID: database.DefaultClientID, TargetID: target.ID, IfIndex: 1, At: seen.Add(-time.Minute), InErrors: 99},
		{
			ClientID: database.DefaultClientID, TargetID: target.ID, IfIndex: 1, At: seen,
			Octets:   &ifrate.Octets{In: 1000, Out: 200, Utilization: &util},
			InErrors: 4, OutErrors: 1, InDiscards: 2, OutDiscards: 0.5,
		},
		// Octets not rated (a 32-bit counter that could wrap): errors only.
		{ClientID: database.DefaultClientID, TargetID: target.ID, IfIndex: 2, At: seen, OutErrors: 0.25},
	}); err != nil {
		t.Fatalf("RecordInterfaceRates: %v", err)
	}

	got, err := db.Metrics().InterfaceStats(ctx, database.DefaultClientID)
	if err != nil {
		t.Fatalf("InterfaceStats: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d interfaces, want 3: %+v", len(got), got)
	}

	assertRatedUplink(t, got[0], target.ID, seen)
	assertErrorsOnly(t, got[1])
	if got[2].IfIndex != 3 || got[2].Rates != nil {
		t.Errorf("ifIndex 3 = %+v, want listed with no rates yet", got[2])
	}
}

// assertRatedUplink checks ifIndex 1 carries its identity and every point of
// its latest poll, not the older one's.
func assertRatedUplink(t *testing.T, got ifstats.Interface, targetID string, at time.Time) {
	t.Helper()
	ptr := func(v float64) *float64 { return &v }
	require.Equal(t, ifstats.Interface{
		TargetID: targetID, TargetName: "core-sw", IfIndex: 1, Name: "Gi0/1", Alias: "uplink",
		OperStatus: ifstats.OperUp, SpeedBps: 1e9,
		Rates: &ifstats.Rates{
			At: at, InOctets: ptr(1000), OutOctets: ptr(200), InUtilization: ptr(12.5), OutUtilization: ptr(3),
			InErrors: 4, OutErrors: 1, InDiscards: 2, OutDiscards: 0.5,
		},
	}, got)
}

// assertErrorsOnly checks ifIndex 2 falls back to ifDescr for its name and
// carries errors without the octets its poll could not rate.
func assertErrorsOnly(t *testing.T, got ifstats.Interface) {
	t.Helper()
	if got.Name != "GigabitEthernet0/2" || got.OperStatus != ifstats.OperDown {
		t.Errorf("ifIndex 2 identity = %+v, want ifDescr as the name and down", got)
	}
	if r := got.Rates; r == nil || r.InOctets != nil || r.InUtilization != nil || r.OutErrors != 0.25 {
		t.Errorf("ifIndex 2 rates = %+v, want errors without octets", r)
	}
}

func TestInterfaceStatsEmpty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	got, err := db.Metrics().InterfaceStats(context.Background(), database.DefaultClientID)
	if err != nil {
		t.Fatalf("InterfaceStats: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want no interfaces", got)
	}
}
