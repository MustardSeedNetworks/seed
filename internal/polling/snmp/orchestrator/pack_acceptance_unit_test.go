package orchestrator_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/lldp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
)

// observationLog is the sink's store: it keeps the kind of every observation
// the recorder hands on.
type observationLog struct{ kinds []string }

func (l *observationLog) Insert(_ context.Context, obs *observation.SNMPObservation) error {
	l.kinds = append(l.kinds, obs.Kind)
	return nil
}

func newRowRecorder(poller string) (*rowRecorder, *observationLog) {
	log := &observationLog{}
	return &rowRecorder{poller: poller, next: sink.New(log, slog.New(slog.DiscardHandler), nil)}, log
}

// NIAC keys its manifest's expectedObservations by these names (its
// internal/scenario Collector* constants). Neither repository imports the
// other, so a rename here without one there silently unpairs them.
func TestCollectorsAreTheTenNamesNIACKeysBy(t *testing.T) {
	t.Parallel()
	var names []string
	for _, collector := range orchestrator.Collectors(nopClientFactory, &rowRecorder{}, at) {
		names = append(names, collector.Name())
	}
	want := []string{
		"sys_info", "if_table", "lldp", "cdp", "fdp", "arp", "fdb", "routing", "host_resources", "bgp4_mib",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("collectors = %v, want %v", names, want)
	}
}

func TestRowRecorderCountsFDBPortsNotMACs(t *testing.T) {
	t.Parallel()
	recorder, _ := newRowRecorder("")
	err := recorder.PublishFDB(context.Background(), fdb.Observation{Entries: []fdb.Entry{
		{MACAddress: "00:00:5e:00:53:01", BridgePort: 1},
		{MACAddress: "00:00:5e:00:53:02", BridgePort: 1},
		{MACAddress: "00:00:5e:00:53:03", BridgePort: 2},
	}})
	if err != nil {
		t.Fatalf("PublishFDB: %v", err)
	}
	if recorder.rows != 2 {
		t.Fatalf("rows = %d, want 2 bridge ports", recorder.rows)
	}
}

func TestRowRecorderLeavesOutThePollersFDBEntry(t *testing.T) {
	t.Parallel()
	recorder, _ := newRowRecorder("00:00:5e:00:53:ff")
	err := recorder.PublishFDB(context.Background(), fdb.Observation{Entries: []fdb.Entry{
		{MACAddress: "00:00:5e:00:53:01", BridgePort: 1},
		{MACAddress: "00:00:5e:00:53:ff", BridgePort: 43},
	}})
	if err != nil {
		t.Fatalf("PublishFDB: %v", err)
	}
	if recorder.rows != 1 {
		t.Fatalf("rows = %d, want 1 (the poller's own port is not the pack's)", recorder.rows)
	}
}

// The topology consumers read what the sink stored, so a count the recorder
// keeps for itself would leave them nothing to reconcile.
func TestRowRecorderHandsEveryObservationToTheSink(t *testing.T) {
	t.Parallel()
	recorder, log := newRowRecorder("")
	ctx := context.Background()
	if err := recorder.PublishLLDP(ctx, lldp.Observation{Neighbors: []lldp.Neighbor{{SysName: "sw2"}}}); err != nil {
		t.Fatalf("PublishLLDP: %v", err)
	}
	if err := recorder.PublishFDB(ctx, fdb.Observation{}); err != nil {
		t.Fatalf("PublishFDB: %v", err)
	}
	if want := []string{sink.KindLLDP, sink.KindFDB}; !slices.Equal(log.kinds, want) {
		t.Fatalf("stored kinds = %q, want %q", log.kinds, want)
	}
}

func TestTallyResults(t *testing.T) {
	t.Parallel()
	tallies := tallyResults([]packResult{
		{Collector: "lldp", Agent: "sw1", Rows: 3},
		{Collector: "lldp", Agent: "sw2", Rows: 2},
		{Collector: "lldp", Agent: "host", Rows: 0},
		{Collector: "routing", Agent: "sw1", Err: errors.New("timeout")},
	})
	if got := tallies["lldp"]; got.Devices != 2 || got.Rows != 5 || len(got.Errors) != 0 {
		t.Errorf("lldp = %+v, want 2 devices, 5 rows, no errors (an empty table is not a device)", got)
	}
	if got := tallies["routing"]; got.Devices != 0 || len(got.Errors) != 1 || got.Errors[0] != "sw1: timeout" {
		t.Errorf("routing = %+v, want one error naming sw1 and no devices", got)
	}
}

func TestPackFindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		expected map[string]packObservation
		tallies  map[string]packTally
		want     []string
	}{
		{
			name:     "matching counts",
			expected: map[string]packObservation{"if_table": {Devices: 2, Rows: 7}, "lldp": {Devices: 2}},
			tallies:  map[string]packTally{"if_table": {Devices: 2, Rows: 7}, "lldp": {Devices: 2, Rows: 9}},
		},
		{
			name:     "device count differs",
			expected: map[string]packObservation{"lldp": {Devices: 3}},
			tallies:  map[string]packTally{"lldp": {Devices: 2, Rows: 4}},
			want:     []string{"lldp: found rows on 2 devices, the manifest promises 3"},
		},
		{
			name:     "row count differs",
			expected: map[string]packObservation{"routing": {Devices: 1, Rows: 2}},
			tallies:  map[string]packTally{"routing": {Devices: 1, Rows: 5}},
			want:     []string{"routing: found 5 rows, the manifest promises 2"},
		},
		{
			name:     "promised collector found nothing",
			expected: map[string]packObservation{"fdb": {Devices: 1, Rows: 3}},
			want: []string{
				"fdb: found rows on 0 devices, the manifest promises 1",
				"fdb: found 0 rows, the manifest promises 3",
			},
		},
		{
			name:    "unpromised collector finding rows is no finding",
			tallies: map[string]packTally{"bgp4_mib": {Devices: 1, Rows: 2}},
		},
		{
			name:    "unpromised collector erroring is a finding",
			tallies: map[string]packTally{"bgp4_mib": {Errors: []string{"r1: timeout"}}},
			want:    []string{"bgp4_mib: collect failed on r1: timeout"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := packFindings(tt.expected, tt.tallies); !slices.Equal(got, tt.want) {
				t.Errorf("findings = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgentRows(t *testing.T) {
	t.Parallel()
	got := agentRows([]packResult{
		{Collector: "sys_info", Agent: "sw2", Rows: 1},
		{Collector: "sys_info", Agent: "sw1", Rows: 1},
		{Collector: "lldp", Agent: "sw1", Err: errors.New("timeout")},
	})
	want := []string{"sw1: sys_info=1 lldp=error", "sw2: sys_info=1"}
	if !slices.Equal(got, want) {
		t.Errorf("agentRows = %q, want %q", got, want)
	}
}

func TestTopologyFindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		authored, drawn []packLink
		want            []string
	}{
		{
			name:     "the same pairs in either order",
			authored: []packLink{{"core", "acc1"}, {"acc1", "pump"}},
			drawn:    []packLink{{"pump", "acc1"}, {"acc1", "core"}},
		},
		{
			name:     "parallel cables are one pair",
			authored: []packLink{{"core", "dist"}, {"dist", "core"}},
			drawn:    []packLink{{"core", "dist"}},
		},
		{
			name:     "a cable left out",
			authored: []packLink{{"core", "acc1"}, {"acc1", "pump"}},
			drawn:    []packLink{{"acc1", "core"}},
			want:     []string{"topology: no link drawn for authored acc1 -- pump"},
		},
		{
			name:     "a cable invented",
			authored: []packLink{{"core", "acc1"}},
			drawn:    []packLink{{"acc1", "core"}, {"core", "pump"}},
			want:     []string{"topology: link drawn for core -- pump, which the pack does not author"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := topologyFindings(tt.authored, tt.drawn); !slices.Equal(got, tt.want) {
				t.Errorf("findings = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAlertFindings(t *testing.T) {
	t.Parallel()
	fault := packFault{Device: "core", Interface: "Gi0/2"}
	down := func(source, iface string) *alerts.Alert {
		return &alerts.Alert{Rule: ifaceDownRule, Source: source, Title: "Interface " + iface + " down on " + source}
	}
	tests := []struct {
		name   string
		raised []*alerts.Alert
		want   []string
	}{
		{
			name:   "the faulted interface alone",
			raised: []*alerts.Alert{down("core", "Gi0/2")},
		},
		{
			name: "nothing raised",
			want: []string{"alerts: no iface.down for core Gi0/2, which NIAC took down"},
		},
		{
			name:   "an interface whose name extends the faulted one",
			raised: []*alerts.Alert{down("core", "Gi0/20")},
			want: []string{
				`alerts: iface.down on core ("Interface Gi0/20 down on core"), which the fault does not explain`,
				"alerts: no iface.down for core Gi0/2, which NIAC took down",
			},
		},
		{
			name:   "the same interface on another device",
			raised: []*alerts.Alert{down("core", "Gi0/2"), down("dist", "Gi0/2")},
			want: []string{
				`alerts: iface.down on dist ("Interface Gi0/2 down on dist"), which the fault does not explain`,
			},
		},
		{
			name: "another rule on the faulted device",
			raised: []*alerts.Alert{
				down("core", "Gi0/2"),
				{Rule: "storage.high", Source: "core", Title: "Filesystem / high on core"},
			},
			want: []string{
				`alerts: storage.high on core ("Filesystem / high on core"), which the fault does not explain`,
			},
		},
		{
			name:   "raised twice",
			raised: []*alerts.Alert{down("core", "Gi0/2"), down("core", "Gi0/2")},
			want:   []string{"alerts: 2 iface.down alerts for core Gi0/2, want one"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := alertFindings(fault, tt.raised); !slices.Equal(got, tt.want) {
				t.Errorf("findings = %q, want %q", got, tt.want)
			}
		})
	}
}
