package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// A device has a MAC per interface and primary_mac names only one of
// them, so an interface's ifPhysAddress has to resolve too, or every
// MAC-keyed consumer (the ARP reconciler's primary_ip backfill, the
// edge reconciler's fdb pass) misses a node by any other port.
func TestNodeForMAC_ResolvesThroughInterfacePhysAddress(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := database.Open(dbtest.Path(t))
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
	db, err := database.Open(dbtest.Path(t))
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
	db, err := database.Open(dbtest.Path(t))
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

// The ARP reconciler's address overwrites; the ifTable reconciler's
// fallback only fills a node nothing has addressed yet, in either order.
func TestPrimaryMAC_ARPWinsOverTheIfTableFallback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	topo := db.Topology()

	const (
		arpMAC   = "aa:bb:cc:00:00:02"
		ifMAC    = "aa:bb:cc:00:00:01"
		arpIP    = "192.0.2.10"
		fillOnly = "node-fill-only"
		fillARP  = "node-fill-then-arp"
		arpFill  = "node-arp-then-fill"
	)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{fillOnly, fillARP, arpFill} {
		if _, err = topo.Upsert(ctx, &topology.Node{
			ID: id, ClientID: "default", IdentityHash: "h-" + id,
			DisplayName: id, LastSeen: now,
		}); err != nil {
			t.Fatalf("upsert %s: %v", id, err)
		}
	}
	steps := []func() error{
		func() error { return topo.FillNodePrimaryMAC(ctx, fillOnly, ifMAC) },
		func() error { return topo.FillNodePrimaryMAC(ctx, fillOnly, arpMAC) },
		func() error { return topo.FillNodePrimaryMAC(ctx, fillARP, ifMAC) },
		func() error { return topo.SetNodePrimaryAddress(ctx, fillARP, arpIP, arpMAC) },
		func() error { return topo.SetNodePrimaryAddress(ctx, arpFill, arpIP, arpMAC) },
		func() error { return topo.FillNodePrimaryMAC(ctx, arpFill, ifMAC) },
	}
	for i, step := range steps {
		if err = step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}

	nodes, err := topo.List(ctx, topology.ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := map[string]string{fillOnly: ifMAC, fillARP: arpMAC, arpFill: arpMAC}
	for _, n := range nodes {
		if n.PrimaryMAC != want[n.ID] {
			t.Errorf("%s primary_mac = %q, want %q", n.ID, n.PrimaryMAC, want[n.ID])
		}
	}
}
