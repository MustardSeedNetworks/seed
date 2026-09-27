package orchestrator_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
)

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
	recorder := &rowRecorder{}
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
