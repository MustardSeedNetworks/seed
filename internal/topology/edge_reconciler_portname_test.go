package topology_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// linkOfKind returns the single link of a kind, failing when the
// pass wrote none or more than one.
func linkOfKind(t *testing.T, store *fakeEdgeStore, kind string) *topology.Link {
	t.Helper()
	var out []*topology.Link
	for _, l := range store.links {
		if l.LinkType == kind {
			out = append(out, l)
		}
	}
	if len(out) != 1 {
		t.Fatalf("%s links = %d, want 1", kind, len(out))
	}
	return out[0]
}

// seed#2455: the near end of an LLDP link was rendered "ifIndex-2"
// while the far end carried the neighbor's own port name, so no
// comparator could match seed's view of a cable against anyone
// else's.
func TestEdgeReconcile_LLDPLocalPortIsTheInterfaceName(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore().
		withInterface("node-A", 2, "GigabitEthernet1/0/11", "Gi1/0/11")
	store.targetMap["c|t-source"] = "node-A"
	store.sysNameMap["core-sw"] = "node-B"

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		lldpObs("t-source", at(), []map[string]any{
			{
				"LocalPortNum": 2, "PortID": "Hu0/0/1",
				"PortDescription": "HundredGigabitEthernet0/0/1", "SysName": "core-sw",
			},
		}),
	})

	l := linkOfKind(t, store, "lldp")
	if l.SourceInterface != "GigabitEthernet1/0/11" {
		t.Errorf("SourceInterface = %q, want GigabitEthernet1/0/11", l.SourceInterface)
	}
	// Both ends of one cable in one vocabulary: the long form the
	// remote reports in lldpRemPortDesc.
	if l.TargetInterface != "HundredGigabitEthernet0/0/1" {
		t.Errorf("TargetInterface = %q, want HundredGigabitEthernet0/0/1", l.TargetInterface)
	}
}

// cdp/fdp index by cdpInterfaceIfIndex, a real ifIndex, so they
// resolve through the same table — relabelling LLDP alone would
// leave the other two passes speaking the old vocabulary.
func TestEdgeReconcile_CDPLocalPortIsTheInterfaceName(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore().
		withInterface("node-A", 5, "GigabitEthernet1/0/24", "Gi1/0/24")
	store.targetMap["c|t-source"] = "node-A"
	store.sysNameMap["fqdn-core-sw"] = "node-B"

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		cdpObs("t-source", "cdp", at(), []map[string]any{
			{"LocalIfIndex": 5, "DeviceID": "fqdn-core-sw", "DevicePort": "Gi1/0/24"},
		}),
	})

	if got := linkOfKind(t, store, "cdp").SourceInterface; got != "GigabitEthernet1/0/24" {
		t.Errorf("SourceInterface = %q, want GigabitEthernet1/0/24", got)
	}
}

// ifDescr is the long form; ifName is the abbreviation. A node that
// answered ifXTable but not ifDescr still gets a real name.
func TestEdgeReconcile_LocalPortFallsBackToIfNameThenIfIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		descr string
		ifNm  string
		want  string
	}{
		{"ifDescr wins", "GigabitEthernet1/0/11", "Gi1/0/11", "GigabitEthernet1/0/11"},
		{"ifName when ifDescr is empty", "", "Gi1/0/11", "Gi1/0/11"},
		{"ifIndex label when the row carries neither", "", "", "ifIndex-2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newFakeEdgeStore().withInterface("node-A", 2, tt.descr, tt.ifNm)
			store.targetMap["c|t-source"] = "node-A"
			store.sysNameMap["core-sw"] = "node-B"
			runFDBReconcile(t, store, []*observation.SNMPObservation{
				lldpObs("t-source", at(), []map[string]any{
					{"LocalPortNum": 2, "SysName": "core-sw"},
				}),
			})
			if got := linkOfKind(t, store, "lldp").SourceInterface; got != tt.want {
				t.Errorf("SourceInterface = %q, want %q", got, tt.want)
			}
		})
	}
}

// A switch seed has not run the iftable collector against yet, and a
// vendor whose lldpLocPortNum is a bridge port rather than an
// ifIndex, both land here: the label is what it always was.
func TestEdgeReconcile_LocalPortFallsBackWhenTheIfTableHasNoSuchIndex(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore().withInterface("node-A", 2, "GigabitEthernet1/0/11", "Gi1/0/11")
	store.targetMap["c|t-source"] = "node-A"
	store.sysNameMap["core-sw"] = "node-B"

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		lldpObs("t-source", at(), []map[string]any{
			{"LocalPortNum": 97, "SysName": "core-sw"},
		}),
	})

	if got := linkOfKind(t, store, "lldp").SourceInterface; got != "ifIndex-97" {
		t.Errorf("SourceInterface = %q, want ifIndex-97", got)
	}
}

// The fdb pass names its port through the same table, so an access
// port reads the same as an uplink the neighbor pass described.
func TestFDBReconcile_SourceInterfaceIsTheInterfaceName(t *testing.T) {
	t.Parallel()
	store := newFDBStore().withInterface("node-SW", 5, "GigabitEthernet0/5", "Gi0/5")
	runFDBReconcile(t, store, []*observation.SNMPObservation{
		fdbObs("t-sw", at(), []fdbEntry{
			{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 5, IfIndex: 5, Status: fdbLearned, VLANID: 200},
		}),
	})
	if got := linkOfKind(t, store, "fdb").SourceInterface; got != "GigabitEthernet0/5" {
		t.Errorf("SourceInterface = %q, want GigabitEthernet0/5", got)
	}
}

// The uplink test of seed#2454 compares the fdb pass's own label
// against the labels the neighbor pass wrote. Both sides must move
// to names together: resolve one and not the other and the
// comparison silently stops matching, and the MAC of a device two
// hops away becomes a false edge on the uplink.
func TestFDBReconcile_PortClaimedByLLDPIsSkippedWhenBothResolveToNames(t *testing.T) {
	t.Parallel()
	store := newFDBStore().withInterface("node-SW", 49, "GigabitEthernet0/49", "Gi0/49")
	store.sysNameMap["core-sw"] = "node-CORE"

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		lldpObs("t-sw", at(), []map[string]any{
			{"LocalPortNum": 49, "PortID": "Gi0/1", "SysName": "core-sw"},
		}),
		fdbObs("t-sw", at(), []fdbEntry{
			{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 49, IfIndex: 49, Status: fdbLearned, VLANID: 200},
		}),
	})

	if got := linkOfKind(t, store, "lldp").SourceInterface; got != "GigabitEthernet0/49" {
		t.Fatalf("the LLDP claim reads %q, so this test would pass on the label alone", got)
	}
	if got := len(fdbLinksOf(store)); got != 0 {
		t.Errorf("fdb links = %d, want 0: LLDP already describes GigabitEthernet0/49", got)
	}
}

// The far end's report of our port was compared against "ifIndex-N"
// and so had never matched anything. Once the local end is named the
// way the remote names it, a port only the neighbor switch described
// is claimed too.
func TestFDBReconcile_PortClaimedByTheRemoteEndIsSkipped(t *testing.T) {
	t.Parallel()
	store := newFDBStore().withInterface("node-SW", 49, "GigabitEthernet0/49", "Gi0/49")
	store.targetMap["c|t-core"] = "node-CORE"
	store.sysNameMap["sw"] = "node-SW"

	runFDBReconcile(t, store, []*observation.SNMPObservation{
		// Only the core switch runs LLDP; it names our port.
		lldpObs("t-core", at(), []map[string]any{
			{"LocalPortNum": 1, "PortDescription": "GigabitEthernet0/49", "SysName": "sw"},
		}),
		fdbObs("t-sw", at(), []fdbEntry{
			{MACAddress: "aa:bb:cc:00:00:01", BridgePort: 49, IfIndex: 49, Status: fdbLearned, VLANID: 200},
		}),
	})

	if got := len(fdbLinksOf(store)); got != 0 {
		t.Errorf("fdb links = %d, want 0: the core switch already describes GigabitEthernet0/49", got)
	}
}
