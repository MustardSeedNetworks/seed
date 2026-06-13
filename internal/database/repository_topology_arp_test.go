package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

func newTopoARPDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(t.TempDir() + "/seed.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedARPBindings(t *testing.T, db *database.DB) {
	t.Helper()
	ctx := context.Background()
	// FK requires topology_nodes rows for both source nodes.
	for _, n := range []*topology.Node{
		{
			ID: "node-a", ClientID: "default", IdentityHash: "h-a",
			DisplayName: "router-a", LastSeen: time.Now().UTC(),
		},
		{
			ID: "node-b", ClientID: "default", IdentityHash: "h-b",
			DisplayName: "router-b", LastSeen: time.Now().UTC(),
		},
	} {
		if _, err := db.Topology().Upsert(ctx, n); err != nil {
			t.Fatalf("seed node %s: %v", n.ID, err)
		}
	}
	rows := []*topology.ARPBinding{
		{
			ClientID: "default", SourceNodeID: "node-a", IfIndex: 1,
			IPAddress: "10.0.0.1", MACAddress: "aa:bb:cc:00:00:01",
			LastSeen: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC),
		},
		{
			ClientID: "default", SourceNodeID: "node-a", IfIndex: 1,
			IPAddress: "10.0.0.2", MACAddress: "aa:bb:cc:00:00:02",
			LastSeen: time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC),
		},
		{
			ClientID: "default", SourceNodeID: "node-b", IfIndex: 2,
			IPAddress: "10.0.1.1", MACAddress: "aa:bb:cc:00:01:01",
			LastSeen: time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC),
		},
	}
	for _, r := range rows {
		if err := db.Topology().UpsertARPBinding(ctx, r); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
	}
}

func TestListARPBindings_EmptyReturnsEmpty(t *testing.T) {
	t.Parallel()
	db := newTopoARPDB(t)
	rows, err := db.Topology().ListARPBindings(context.Background(),
		topology.ARPListOptions{})
	if err != nil {
		t.Fatalf("ListARPBindings: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("empty table -> %d rows, want 0", len(rows))
	}
}

func TestListARPBindings_ReturnsAllRowsOrderedByLastSeenDesc(t *testing.T) {
	t.Parallel()
	db := newTopoARPDB(t)
	seedARPBindings(t, db)

	rows, err := db.Topology().ListARPBindings(context.Background(),
		topology.ARPListOptions{})
	if err != nil {
		t.Fatalf("ListARPBindings: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	if !rows[0].LastSeen.After(rows[1].LastSeen) && !rows[0].LastSeen.Equal(rows[1].LastSeen) {
		t.Errorf("rows not sorted by last_seen desc: %v then %v",
			rows[0].LastSeen, rows[1].LastSeen)
	}
}

func TestListARPBindings_FilterBySourceNode(t *testing.T) {
	t.Parallel()
	db := newTopoARPDB(t)
	seedARPBindings(t, db)

	rows, err := db.Topology().ListARPBindings(context.Background(),
		topology.ARPListOptions{SourceNodeID: "node-b"})
	if err != nil {
		t.Fatalf("ListARPBindings: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("source_node=node-b -> %d rows, want 1", len(rows))
	}
	if rows[0].SourceNodeID != "node-b" {
		t.Errorf("got source_node = %q, want node-b", rows[0].SourceNodeID)
	}
}

func TestListARPBindings_FilterBySince(t *testing.T) {
	t.Parallel()
	db := newTopoARPDB(t)
	seedARPBindings(t, db)

	cutoff := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	rows, err := db.Topology().ListARPBindings(context.Background(),
		topology.ARPListOptions{Since: cutoff})
	if err != nil {
		t.Fatalf("ListARPBindings: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("since=2026-05-31 -> %d rows, want 2 (the two May-31 entries)", len(rows))
	}
}

func TestListARPBindings_LimitClamps(t *testing.T) {
	t.Parallel()
	db := newTopoARPDB(t)
	seedARPBindings(t, db)

	rows, err := db.Topology().ListARPBindings(context.Background(),
		topology.ARPListOptions{Limit: 1})
	if err != nil {
		t.Fatalf("ListARPBindings: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("limit=1 -> %d rows, want 1", len(rows))
	}
}
