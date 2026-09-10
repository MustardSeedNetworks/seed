package database_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/lldp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/sysinfo"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// The access-layer shape the niac hospital pack has and seed#2454
// says seed cannot draw: one access switch, one uplink to the core
// switch, and endpoints that speak no discovery protocol at all —
// they are only ever visible as a MAC in the switch's forwarding
// database. Every payload below travels the production path: the
// collector Observation structs, marshalled by the real sink into
// snmp_observations, read back by the real reconcilers against the
// real repositories.
const (
	fdbSwitchTarget   = "t-access-sw"
	fdbCoreTarget     = "t-core-sw"
	fdbPumpTarget     = "t-infusion-pump"
	fdbPumpMAC        = "aa:bb:cc:00:11:22"
	fdbPumpIP         = "10.51.200.31"
	fdbCoreUplinkMAC  = "aa:bb:cc:00:99:01"
	fdbForeignMAC     = "aa:bb:cc:00:99:02"
	fdbAccessPortIdx  = 5
	fdbUplinkPortIdx  = 49
	fdbAccessPortNum  = 5
	fdbUplinkPortNum  = 49
	fdbEndpointIfName = "eth0"
)

func fdbTestTime() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }

// buildHospitalAccessLayer publishes the pack's observations through
// the real sink and runs the sysinfo + iftable + edge reconcilers in
// the order the daemon does. Returns the topology repository.
func buildHospitalAccessLayer(t *testing.T) (*database.DB, string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(t.TempDir() + "/seed.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	at := fdbTestTime()
	s := sink.New(db.SNMPObservations(), slog.New(slog.DiscardHandler), func() time.Time { return at })

	for _, d := range []struct{ target, name, oid string }{
		{fdbSwitchTarget, "hosp-access-sw-1", "1.3.6.1.4.1.9.1.100"},
		{fdbCoreTarget, "hosp-core-sw-1", "1.3.6.1.4.1.9.1.101"},
		{fdbPumpTarget, "hosp-infusion-pump-1", "1.3.6.1.4.1.4444.1.1"},
	} {
		if pubErr := s.PublishSysInfo(ctx, sysinfo.Observation{
			ClientID: "default", TargetID: d.target, ObservedAt: at,
			SysName: d.name, SysObjectID: d.oid,
		}); pubErr != nil {
			t.Fatalf("publish sysinfo %s: %v", d.target, pubErr)
		}
	}

	// The pump's own ifTable is where its MAC becomes known to seed.
	if pubErr := s.PublishIfTable(ctx, iftable.Observation{
		ClientID: "default", TargetID: fdbPumpTarget, ObservedAt: at,
		Rows: []iftable.Row{{
			IfIndex: 1, IfName: fdbEndpointIfName, IfDescr: "Ethernet",
			IfPhysAddr: fdbPumpMAC, IfAdmin: 1, IfOper: 1,
		}},
	}); pubErr != nil {
		t.Fatalf("publish iftable: %v", pubErr)
	}

	// The access switch's uplink to the core is an LLDP edge; the
	// core's MAC is therefore also in the FDB, on the uplink port.
	if pubErr := s.PublishLLDP(ctx, lldp.Observation{
		ClientID: "default", TargetID: fdbSwitchTarget, ObservedAt: at,
		Neighbors: []lldp.Neighbor{{
			LocalPortNum: fdbUplinkPortIdx, PortID: "Gi0/1",
			SysName: "hosp-core-sw-1",
		}},
	}); pubErr != nil {
		t.Fatalf("publish lldp: %v", pubErr)
	}

	// The access switch's ARP table is where the pump's IP is
	// learned; it reaches the pump's node through the same MAC.
	if pubErr := s.PublishARP(ctx, arp.Observation{
		ClientID: "default", TargetID: fdbSwitchTarget, ObservedAt: at,
		Entries: []arp.Entry{{
			IfIndex: fdbAccessPortIdx, IPAddress: fdbPumpIP,
			MACAddress: fdbPumpMAC, MediaType: 1,
		}},
	}); pubErr != nil {
		t.Fatalf("publish arp: %v", pubErr)
	}

	if pubErr := s.PublishFDB(ctx, fdb.Observation{
		ClientID: "default", TargetID: fdbSwitchTarget, ObservedAt: at,
		Entries: []fdb.Entry{
			// The pump: one learned MAC on one access port.
			{
				VLANID: 200, MACAddress: fdbPumpMAC, BridgePort: fdbAccessPortNum,
				IfIndex: fdbAccessPortIdx, Status: fdb.StatusLearned,
			},
			// The uplink: the core switch plus an unrelated MAC
			// behind it. Two MACs make the port a trunk.
			{
				VLANID: 200, MACAddress: fdbCoreUplinkMAC, BridgePort: fdbUplinkPortNum,
				IfIndex: fdbUplinkPortIdx, Status: fdb.StatusLearned,
			},
			{
				VLANID: 200, MACAddress: fdbForeignMAC, BridgePort: fdbUplinkPortNum,
				IfIndex: fdbUplinkPortIdx, Status: fdb.StatusLearned,
			},
		},
	}); pubErr != nil {
		t.Fatalf("publish fdb: %v", pubErr)
	}

	runTopologyReconcilers(t, db)

	switchNodeID, err := db.Topology().NodeIDForTarget(ctx, "default", fdbSwitchTarget)
	if err != nil {
		t.Fatalf("switch node not reconciled: %v", err)
	}
	return db, switchNodeID
}

func runTopologyReconcilers(t *testing.T, db *database.DB) {
	t.Helper()
	ctx := context.Background()
	obs, topo, settings := db.SNMPObservations(), db.Topology(), db.Settings()
	logger := slog.New(slog.DiscardHandler)

	si, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: obs, Nodes: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("sysinfo reconciler: %v", err)
	}
	if err = si.ReconcileOnce(ctx); err != nil {
		t.Fatalf("sysinfo reconcile: %v", err)
	}

	iface, err := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("iftable reconciler: %v", err)
	}
	if err = iface.ReconcileOnce(ctx); err != nil {
		t.Fatalf("iftable reconcile: %v", err)
	}

	arpRec, err := topology.NewARPReconciler(topology.ARPConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("arp reconciler: %v", err)
	}
	if err = arpRec.ReconcileOnce(ctx); err != nil {
		t.Fatalf("arp reconcile: %v", err)
	}

	edge, err := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("edge reconciler: %v", err)
	}
	if err = edge.ReconcileOnce(ctx); err != nil {
		t.Fatalf("edge reconcile: %v", err)
	}
}

// TestFDBEdges_EndpointBehindAccessPortBecomesALink is seed#2454's
// acceptance: the endpoint speaks no discovery protocol, so the only
// evidence it is on that port is the forwarding database.
func TestFDBEdges_EndpointBehindAccessPortBecomesALink(t *testing.T) {
	t.Parallel()
	db, switchNodeID := buildHospitalAccessLayer(t)

	links, err := db.Topology().ListLinks(context.Background(), switchNodeID)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	var fdbLinks, lldpLinks int
	for _, l := range links {
		switch l.LinkType {
		case "fdb":
			fdbLinks++
			if l.SourceInterface != "ifIndex-5" {
				t.Errorf("fdb link source interface = %q, want ifIndex-5", l.SourceInterface)
			}
			if l.TargetInterface != fdbEndpointIfName {
				t.Errorf("fdb link target interface = %q, want %q",
					l.TargetInterface, fdbEndpointIfName)
			}
		case "lldp":
			lldpLinks++
		}
	}
	if lldpLinks != 1 {
		t.Errorf("lldp links = %d, want 1 (the uplink)", lldpLinks)
	}
	if fdbLinks != 1 {
		t.Fatalf("fdb links = %d, want 1 (the pump behind the access port); "+
			"seed#2454: the fdb observations are never reconciled into edges", fdbLinks)
	}
}

// The ARP reconciler's primary_ip backfill has never fired in
// production: it resolves the binding's MAC through NodeIDForMAC,
// which matched only topology_nodes.primary_mac — a column no
// producer writes. Resolving through the interface's ifPhysAddress
// wakes it up, so the pump's node finally carries an address.
func TestFDBEdges_ARPBackfillGivesTheEndpointItsAddress(t *testing.T) {
	t.Parallel()
	db, _ := buildHospitalAccessLayer(t)

	nodes, err := db.Topology().List(context.Background(), topology.ListOptions{})
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	for _, n := range nodes {
		if n.SysName != "hosp-infusion-pump-1" {
			continue
		}
		if n.PrimaryIP != fdbPumpIP {
			t.Fatalf("pump primary_ip = %q, want %q", n.PrimaryIP, fdbPumpIP)
		}
		return
	}
	t.Fatal("pump node not found")
}
