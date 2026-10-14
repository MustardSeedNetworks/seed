package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// A node's freshness is judged against the cadence that should refresh it
// (seed#2525), so the list carries the shortest interval of the enabled
// targets mapped to the node, and none when nothing polls it any more.
func TestTopologyList_PollInterval(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	seen := time.Date(2026, 10, 13, 3, 0, 0, 0, time.UTC)

	target := func(name string, seconds int, enabled bool) string {
		t.Helper()
		tg := &polling.Target{
			ClientID: database.DefaultClientID, Name: name, IPAddress: "10.0.0.1",
			SNMPVersion: "v2c", PollIntervalSec: seconds, Enabled: enabled,
			CollectorChain: []string{"sys_info"},
		}
		if err := db.PollingTargets().Create(ctx, tg); err != nil {
			t.Fatalf("create target %s: %v", name, err)
		}
		return tg.ID
	}
	node := func(id string, targetIDs ...string) {
		t.Helper()
		if _, err := db.Topology().Upsert(ctx, &topology.Node{
			ID: id, ClientID: database.DefaultClientID, IdentityHash: "h-" + id, DisplayName: id, LastSeen: seen,
		}); err != nil {
			t.Fatalf("upsert node %s: %v", id, err)
		}
		for _, tid := range targetIDs {
			if err := db.Topology().UpsertTargetNode(ctx, database.DefaultClientID, tid, id, seen); err != nil {
				t.Fatalf("map %s to %s: %v", tid, id, err)
			}
		}
	}

	// One device reached through two addresses polled at different cadences,
	// plus a faster target that has been switched off.
	node("node-core",
		target("core-mgmt", 300, true),
		target("core-loop", 60, true),
		target("core-old", 10, false),
	)
	node("node-retired", target("retired", 60, false))
	node("node-unmapped")

	nodes, err := db.Topology().List(ctx, topology.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := map[string]time.Duration{}
	for _, n := range nodes {
		got[n.ID] = n.PollInterval
	}
	want := map[string]time.Duration{
		"node-core":     time.Minute,
		"node-retired":  0,
		"node-unmapped": 0,
	}
	for id, w := range want {
		if g, ok := got[id]; !ok || g != w {
			t.Errorf("%s PollInterval = %v (listed %t), want %v", id, g, ok, w)
		}
	}
}
