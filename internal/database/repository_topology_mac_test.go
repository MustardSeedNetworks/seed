package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// No producer writes topology_nodes.primary_mac, so an interface's
// ifPhysAddress is the only MAC seed knows for most devices. Both
// have to resolve, or every MAC-keyed consumer (the ARP reconciler's
// primary_ip backfill, the edge reconciler's fdb pass) is dead.
func TestNodeForMAC_ResolvesThroughInterfacePhysAddress(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := database.Open(t.TempDir() + "/seed.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	if _, err = db.Topology().Upsert(ctx, &topology.Node{
		ID: "node-pc", ClientID: "default", IdentityHash: "h-pc",
		DisplayName: "pc-1", LastSeen: now,
	}); err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	if err = db.Topology().UpsertInterface(ctx, &topology.Interface{
		NodeID: "node-pc", IfIndex: 1, IfName: "eth0",
		IfPhysAddr: "aa:bb:cc:00:11:22", LastSeen: now,
	}); err != nil {
		t.Fatalf("upsert interface: %v", err)
	}

	nodeID, ifName, err := db.Topology().NodeForMAC(ctx, "default", "aa:bb:cc:00:11:22")
	if err != nil {
		t.Fatalf("NodeForMAC: %v", err)
	}
	if nodeID != "node-pc" {
		t.Errorf("nodeID = %q, want node-pc", nodeID)
	}
	if ifName != "eth0" {
		t.Errorf("ifName = %q, want eth0", ifName)
	}
}

func TestNodeForMAC_UnknownMACIsNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := database.Open(t.TempDir() + "/seed.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err = db.Topology().Upsert(ctx, &topology.Node{
		ID: "node-pc", ClientID: "default", IdentityHash: "h-pc",
		DisplayName: "pc-1", LastSeen: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	for _, mac := range []string{"aa:bb:cc:00:11:22", ""} {
		_, _, err = db.Topology().NodeForMAC(ctx, "default", mac)
		if !errors.Is(err, topology.ErrTopologyNodeNotFound) {
			t.Errorf("NodeForMAC(%q) error = %v, want ErrTopologyNodeNotFound", mac, err)
		}
	}
}

// An interface MAC belonging to another client's node must not
// resolve — the fdb pass would otherwise cross-link two installs.
func TestNodeForMAC_IsScopedToTheClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := database.Open(t.TempDir() + "/seed.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	if _, err = db.Topology().Upsert(ctx, &topology.Node{
		ID: "node-pc", ClientID: "default", IdentityHash: "h-pc",
		DisplayName: "pc-1", LastSeen: now,
	}); err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	if err = db.Topology().UpsertInterface(ctx, &topology.Interface{
		NodeID: "node-pc", IfIndex: 1, IfName: "eth0",
		IfPhysAddr: "aa:bb:cc:00:11:22", LastSeen: now,
	}); err != nil {
		t.Fatalf("upsert interface: %v", err)
	}
	_, _, err = db.Topology().NodeForMAC(ctx, "other-client", "aa:bb:cc:00:11:22")
	if !errors.Is(err, topology.ErrTopologyNodeNotFound) {
		t.Errorf("cross-client NodeForMAC error = %v, want ErrTopologyNodeNotFound", err)
	}
}
