package topology_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// fdbEntry is one dot1qTpFdbTable row as the collector publishes it.
type fdbEntry struct {
	MACAddress string
	BridgePort uint32
	IfIndex    uint32
	Status     int
	VLANID     uint32
}

const fdbLearned = 3

func fdbObs(target string, observed time.Time, entries []fdbEntry) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Entries": entries})
	return &observation.SNMPObservation{
		ClientID: edgeTestClient, TargetID: target, Kind: "fdb",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

// newFDBStore is one access switch (node-SW) whose ifIndex 5 has the
// endpoint node-PC behind it.
func newFDBStore() *fakeEdgeStore {
	store := newFakeEdgeStore()
	store.targetMap["c|t-sw"] = "node-SW"
	store.macMap["aa:bb:cc:00:00:01"] = macNode{nodeID: "node-PC", ifName: "eth0"}
	return store
}

func runFDBReconcile(t *testing.T, store *fakeEdgeStore, rows []*observation.SNMPObservation) {
	t.Helper()
	r, err := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: &fakeObservations{rows: rows}, Store: store,
		Settings: newFakeSettings(), Logger: silentLogger(), Now: at,
	})
	if err != nil {
		t.Fatalf("NewEdgeReconciler: %v", err)
	}
	if err = r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
}

func fdbLinksOf(store *fakeEdgeStore) []*topology.Link {
	out := make([]*topology.Link, 0, len(store.links))
	for _, l := range store.links {
		if l.LinkType == "fdb" {
			out = append(out, l)
		}
	}
	return out
}

func TestFDBReconcile_LearnedMACOnAnAccessPortBecomesALink(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	runFDBReconcile(t, store, []*observation.SNMPObservation{
		fdbObs("t-sw", at(), []fdbEntry{
			{
				MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5,
				Status: fdbLearned, VLANID: 200,
			},
		}),
	})

	links := fdbLinksOf(store)
	if len(links) != 1 {
		t.Fatalf("fdb links = %d, want 1", len(links))
	}
	l := links[0]
	if l.SourceNodeID != "node-SW" || l.TargetNodeID != "node-PC" {
		t.Errorf("endpoints = (%s, %s), want (node-SW, node-PC)", l.SourceNodeID, l.TargetNodeID)
	}
	if l.SourceInterface != "ifIndex-5" {
		t.Errorf("SourceInterface = %q, want ifIndex-5", l.SourceInterface)
	}
	if l.TargetInterface != "eth0" {
		t.Errorf("TargetInterface = %q, want eth0", l.TargetInterface)
	}
	var evidence map[string]string
	if err := json.Unmarshal([]byte(l.EvidenceJSON), &evidence); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	if evidence["remote_mac"] != "aa:bb:cc:00:00:01" || evidence["bridge_port"] != "5" {
		t.Errorf("evidence = %v, want the MAC and bridge port that produced the edge", evidence)
	}
}

// The same MAC tagged in several VLANs is still one MAC on one port
// — counting the raw rows would make every trunk look like an
// access port.
func TestFDBReconcile_SameMACInManyVLANsIsOnePort(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	runFDBReconcile(t, store, []*observation.SNMPObservation{
		fdbObs("t-sw", at(), []fdbEntry{
			{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5, Status: fdbLearned, VLANID: 200},
			{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5, Status: fdbLearned, VLANID: 300},
		}),
	})
	if got := len(fdbLinksOf(store)); got != 1 {
		t.Fatalf("fdb links = %d, want 1", got)
	}
}

func TestFDBReconcile_SkipsPortsThatAreNotAccessPorts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entries []fdbEntry
		reason  string
	}{
		{
			name: "two MACs on one port is a trunk",
			entries: []fdbEntry{
				{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5, Status: fdbLearned, VLANID: 200},
				{MACAddress: "aa:bb:cc:00:00:02", BridgePort: 5, IfIndex: 5, Status: fdbLearned, VLANID: 200},
			},
			reason: "a port with more than one MAC says nothing about what is plugged into it",
		},
		{
			name: "self rows are the switch's own addresses",
			entries: []fdbEntry{
				{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5, Status: 4, VLANID: 200},
			},
			reason: "dot1qTpFdbStatus self is not something attached to the port",
		},
		{
			name: "a bridge port with no ifTable row cannot be named",
			entries: []fdbEntry{
				{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 0, Status: fdbLearned, VLANID: 200},
			},
			reason: "ifIndex 0 would render the port as ifIndex-0",
		},
		{
			name: "a MAC that resolves to no node",
			entries: []fdbEntry{
				{MACAddress: "aa:bb:cc:00:00:99", BridgePort: 6, IfIndex: 6, Status: fdbLearned, VLANID: 200},
			},
			reason: "V1.0 draws edges between known nodes only",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newFDBStore()
			runFDBReconcile(t, store, []*observation.SNMPObservation{
				fdbObs("t-sw", at(), tt.entries),
			})
			if got := len(fdbLinksOf(store)); got != 0 {
				t.Errorf("fdb links = %d, want 0: %s", got, tt.reason)
			}
		})
	}
}

// An uplink quiet enough to have learned exactly one MAC is still an
// uplink, and that one MAC is a device somewhere beyond it — never
// something plugged into this port. LLDP already described the port,
// which is what says so.
func TestFDBReconcile_PortClaimedByLLDPIsSkipped(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	store.sysNameMap["core-sw"] = "node-CORE"

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		lldpObs("t-sw", at(), []map[string]any{
			{"LocalPortNum": 49, "PortID": "Gi0/1", "SysName": "core-sw"},
		}),
		// node-PC is two hops away; its MAC is learned on the uplink.
		fdbObs("t-sw", at(), []fdbEntry{
			{
				MACAddress: "aa:bb:cc:00:00:01", BridgePort: 49, IfIndex: 49,
				Status: fdbLearned, VLANID: 200,
			},
		}),
	})
	if got := len(fdbLinksOf(store)); got != 0 {
		t.Errorf("fdb links = %d, want 0: LLDP already describes ifIndex-49, "+
			"so the MAC behind it is not attached to this port", got)
	}
}

// The same neighbor learned on a *different* port — a second cable,
// a stale entry after a move — must not produce a second, weaker
// edge to a node LLDP already covers.
func TestFDBReconcile_NeighborAlreadyLinkedIsSkipped(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	store.sysNameMap["core-sw"] = "node-CORE"
	store.macMap["aa:bb:cc:00:00:0c"] = macNode{nodeID: "node-CORE", ifName: "Gi0/1"}

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		lldpObs("t-sw", at(), []map[string]any{
			{"LocalPortNum": 49, "PortID": "Gi0/1", "SysName": "core-sw"},
		}),
		fdbObs("t-sw", at(), []fdbEntry{
			{
				MACAddress: "aa:bb:cc:00:00:0c", BridgePort: 48, IfIndex: 48,
				Status: fdbLearned, VLANID: 200,
			},
		}),
	})
	if got := len(fdbLinksOf(store)); got != 0 {
		t.Errorf("fdb links = %d, want 0: node-CORE is already an LLDP neighbor", got)
	}
}

// The switch's own MAC on one of its own ports would otherwise draw
// a link from a node to itself.
func TestFDBReconcile_SelfLinkIsSkipped(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	store.macMap["aa:bb:cc:00:00:aa"] = macNode{nodeID: "node-SW", ifName: "Vlan200"}
	runFDBReconcile(t, store, []*observation.SNMPObservation{
		fdbObs("t-sw", at(), []fdbEntry{
			{
				MACAddress: "aa:bb:cc:00:00:aa", BridgePort: 7, IfIndex: 7,
				Status: fdbLearned, VLANID: 200,
			},
		}),
	})
	if got := len(fdbLinksOf(store)); got != 0 {
		t.Errorf("fdb links = %d, want 0: a node cannot be linked to itself", got)
	}
}

// An observation from a switch that sysinfo has not reconciled yet
// is skipped, not an error — it lands on the next pass.
func TestFDBReconcile_UnknownSourceTargetIsSkipped(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	runFDBReconcile(t, store, []*observation.SNMPObservation{
		fdbObs("t-unmapped", at(), []fdbEntry{
			{
				MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5,
				Status: fdbLearned, VLANID: 200,
			},
		}),
	})
	if got := len(fdbLinksOf(store)); got != 0 {
		t.Errorf("fdb links = %d, want 0", got)
	}
}

// The fdb pass keeps its own high-water mark: the neighbor mark is
// already far ahead on any install that has been polling, and
// sharing it would leave the access layer blank until every switch
// is polled again.
func TestFDBReconcile_HighWaterIsSeparateFromTheNeighborMark(t *testing.T) {
	t.Parallel()
	store := newFDBStore()
	settings := newFakeSettings()
	r, err := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: &fakeObservations{rows: []*observation.SNMPObservation{
			fdbObs("t-sw", at(), []fdbEntry{
				{
					MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5,
					Status: fdbLearned, VLANID: 200,
				},
			}),
		}}, Store: store, Settings: settings, Logger: silentLogger(), Now: at,
	})
	if err != nil {
		t.Fatalf("NewEdgeReconciler: %v", err)
	}
	if err = r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	got, err := settings.GetWithDefault(context.Background(), "topology.edge.fdb.high_water", "")
	if err != nil {
		t.Fatalf("GetWithDefault: %v", err)
	}
	if got == "" {
		t.Fatal("fdb high-water not saved under its own key")
	}
}
