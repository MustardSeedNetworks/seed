package topology_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/sysinfo"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpwalkfile"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// The role tests elsewhere in this package feed the classifier values
// a test author chose. These replay the recorded walk corpus through
// the real sysinfo collector, so the scalars are the ones eight
// devices actually emitted — including the three whose sysObjectID is
// a lie (MustardSeedNetworks/niac-go#2154).

type rolePublisher struct{ obs []sysinfo.Observation }

func (p *rolePublisher) PublishSysInfo(_ context.Context, o sysinfo.Observation) error {
	p.obs = append(p.obs, o)
	return nil
}

// nodeFromWalk runs the collector against one recorded walk and then
// the sysinfo reconciler over the observation the collector produced,
// marshalled the way the sink marshals it.
func nodeFromWalk(t *testing.T, path string) *topology.Node {
	t.Helper()
	walk, err := snmpwalkfile.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	pub := &rolePublisher{}
	collector := sysinfo.New(
		func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) {
			return walk.Client(), nil
		},
		pub,
		at,
	)
	target := snmp.Target{ID: "t1", ClientID: "c", IPAddress: "192.0.2.1", SNMPVersion: "2c"}
	if collectErr := collector.Collect(context.Background(), target, snmp.ResolvedCredentials{}); collectErr != nil {
		t.Fatalf("collect %s: %v", path, collectErr)
	}
	if len(pub.obs) != 1 {
		t.Fatalf("observations = %d, want 1", len(pub.obs))
	}
	raw, _ := json.Marshal(pub.obs[0])

	nodes := &fakeNodes{}
	r, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: &fakeObservations{rows: []*observation.SNMPObservation{{
			ClientID: "c", TargetID: "t1", Kind: "sys_info",
			ObservedAt: at(), PayloadJSON: string(raw),
		}}},
		Nodes: nodes, Settings: newFakeSettings(), Logger: silentLogger(), Now: at,
	})
	if err != nil {
		t.Fatalf("reconciler: %v", err)
	}
	if err = r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("reconcile %s: %v", path, err)
	}
	if len(nodes.upserts) != 1 {
		t.Fatalf("upserts = %d, want 1", len(nodes.upserts))
	}
	return nodes.upserts[0]
}

// Every device in the corpus is a switch, and every one of them was
// reported as a vendor name or `unknown` before seed#2456. Three of
// the eight — arista, aruba, dell — carry a Cisco sysObjectID, so
// they are also the proof that the role does not come from the
// vendor: under the old code all three read "cisco".
func TestDeviceRole_RecordedWalkCorpus(t *testing.T) {
	t.Parallel()
	paths, err := filepath.Glob(filepath.Join(
		"..", "polling", "snmp", "snmpwalkfile", "testdata", "*.walk"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) != 8 {
		t.Fatalf("walk fixtures = %d, want the 8 recorded vendors; "+
			"a changed corpus needs this expectation re-read, not re-fitted", len(paths))
	}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".walk")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			node := nodeFromWalk(t, path)
			if node.DeviceType != topology.RoleSwitch {
				t.Errorf("DeviceType = %q, want %q", node.DeviceType, topology.RoleSwitch)
			}
			var meta map[string]string
			if unmarshalErr := json.Unmarshal([]byte(node.MetadataJSON), &meta); unmarshalErr != nil {
				t.Fatalf("metadata: %v", unmarshalErr)
			}
			if meta["sys_descr"] == "" {
				t.Error("sys_descr is empty; the walk was not actually read")
			}
		})
	}
}

// The vendor did not disappear when it stopped being the device type
// — it moved to the node's metadata, where the UI can still show it.
func TestDeviceRole_VendorMovesToMetadata(t *testing.T) {
	t.Parallel()
	node := nodeFromWalk(t, filepath.Join(
		"..", "polling", "snmp", "snmpwalkfile", "testdata", "cisco-wc17-02-r01.walk"))
	var meta map[string]string
	if err := json.Unmarshal([]byte(node.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if meta["vendor"] != "cisco" {
		t.Errorf("metadata vendor = %q, want cisco", meta["vendor"])
	}
}
