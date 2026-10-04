package iftable_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
)

type fakeClient struct {
	ifTableVbs  []snmp.Varbind
	ifXTableVbs []snmp.Varbind
	dot3Vbs     []snmp.Varbind
	dot3Err     error
	upTime      any
	getErr      error
	walkErr     error
}

func (f *fakeClient) Get(_ context.Context, oids []string) ([]snmp.Varbind, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if len(oids) != 1 || oids[0] != "1.3.6.1.2.1.1.3.0" {
		return nil, errors.New("iftable reads only sysUpTime.0")
	}
	return []snmp.Varbind{{OID: oids[0], Value: f.upTime}}, nil
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	switch {
	case strings.HasPrefix(prefix, "1.3.6.1.2.1.2.2.1"):
		return f.ifTableVbs, nil
	case strings.HasPrefix(prefix, "1.3.6.1.2.1.31.1.1.1"):
		return f.ifXTableVbs, nil
	case strings.HasPrefix(prefix, "1.3.6.1.2.1.10.7.2.1"):
		return f.dot3Vbs, f.dot3Err
	}
	return nil, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []iftable.Observation
	err error
}

func (p *fakePublisher) PublishIfTable(_ context.Context, obs iftable.Observation) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	p.got = append(p.got, obs)
	return nil
}

func factoryFor(c *fakeClient) snmp.ClientFactory {
	return func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
		return c, nil
	}
}

func at() time.Time { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	c := iftable.New(nil, nil, at)
	if c.Name() != iftable.Name {
		t.Errorf("Name() = %q, want %q", c.Name(), iftable.Name)
	}
}

func TestCollect_BuildsRowsFromIfTableAndIfXTable(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		ifTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.2.2.1.2.1", Value: "GigabitEthernet0/0"},
			{OID: "1.3.6.1.2.1.2.2.1.3.1", Value: uint32(6)},
			{OID: "1.3.6.1.2.1.2.2.1.5.1", Value: uint32(1_000_000_000)},
			{OID: "1.3.6.1.2.1.2.2.1.6.1", Value: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}},
			{OID: "1.3.6.1.2.1.2.2.1.7.1", Value: 1},
			{OID: "1.3.6.1.2.1.2.2.1.8.1", Value: 1},
			{OID: "1.3.6.1.2.1.2.2.1.2.2", Value: "GigabitEthernet0/1"},
			{OID: "1.3.6.1.2.1.2.2.1.7.2", Value: 2},
			{OID: "1.3.6.1.2.1.2.2.1.8.2", Value: 2},
		},
		ifXTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.31.1.1.1.1.1", Value: "Gi0/0"},
			{OID: "1.3.6.1.2.1.31.1.1.1.18.1", Value: "uplink-to-core"},
			{OID: "1.3.6.1.2.1.31.1.1.1.1.2", Value: "Gi0/1"},
		},
	}
	pub := &fakePublisher{}
	c := iftable.New(factoryFor(fc), pub, at)

	target := snmp.Target{ID: "t-1", ClientID: "client-a"}
	if err := c.Collect(context.Background(), target, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("publisher hits = %d, want 1", len(pub.got))
	}
	obs := pub.got[0]
	if len(obs.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(obs.Rows))
	}
	row0 := obs.Rows[0]
	if row0.IfIndex != 1 || row0.IfDescr != "GigabitEthernet0/0" || row0.IfName != "Gi0/0" {
		t.Errorf("row0 = %+v", row0)
	}
	if row0.IfAlias != "uplink-to-core" {
		t.Errorf("row0.IfAlias = %q, want uplink-to-core", row0.IfAlias)
	}
	if row0.IfPhysAddr != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("row0.IfPhysAddr = %q, want aa:bb:cc:dd:ee:ff", row0.IfPhysAddr)
	}
	if row0.IfAdmin != iftable.StatusUp || row0.IfOper != iftable.StatusUp {
		t.Errorf("row0 status admin/oper = %d/%d, want 1/1", row0.IfAdmin, row0.IfOper)
	}
	if row0.SpeedBps != 1_000_000_000 {
		t.Errorf("row0.SpeedBps = %d, want 1Gbps", row0.SpeedBps)
	}
}

func TestCollect_PrefersIfHighSpeedOverIfSpeed(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		ifTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.2.2.1.5.1", Value: uint32(4_294_967_295)}, // ifSpeed maxed out
		},
		ifXTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.31.1.1.1.15.1", Value: uint32(10_000)}, // ifHighSpeed: 10 Gbps
		},
	}
	pub := &fakePublisher{}
	c := iftable.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if pub.got[0].Rows[0].SpeedBps != 10_000_000_000 {
		t.Errorf("SpeedBps = %d, want 10Gbps (from ifHighSpeed)", pub.got[0].Rows[0].SpeedBps)
	}
}

func TestCollect_RowsSortedAscendingByIfIndex(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		ifTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.2.2.1.2.10", Value: "if-10"},
			{OID: "1.3.6.1.2.1.2.2.1.2.2", Value: "if-2"},
			{OID: "1.3.6.1.2.1.2.2.1.2.1", Value: "if-1"},
		},
	}
	pub := &fakePublisher{}
	c := iftable.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	rows := pub.got[0].Rows
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	wantOrder := []uint32{1, 2, 10}
	for i, want := range wantOrder {
		if rows[i].IfIndex != want {
			t.Errorf("rows[%d].IfIndex = %d, want %d", i, rows[i].IfIndex, want)
		}
	}
}

func TestCollect_IfTableWalkErrorPropagates(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{walkErr: errors.New("snmp v3 auth failed")}
	pub := &fakePublisher{}
	c := iftable.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected walk error to propagate")
	}
}

func TestCollect_PublishErrorPropagates(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{}
	pub := &fakePublisher{err: errors.New("topology busy")}
	c := iftable.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected publish error to propagate")
	}
}

func TestCollect_NilDepsReturnError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    *iftable.Collector
	}{
		{"nil factory", iftable.New(nil, &fakePublisher{}, at)},
		{"nil publisher", iftable.New(factoryFor(&fakeClient{}), nil, at)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
				t.Errorf("%s: expected error", tt.name)
			}
		})
	}
}

func TestCollect_FactoryErrorPropagates(t *testing.T) {
	t.Parallel()
	c := iftable.New(
		func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
			return nil, errors.New("no credentials")
		},
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected factory error to propagate")
	}
}

func TestCollect_NonSixByteMACFallsBackToString(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		ifTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.2.2.1.6.1", Value: []byte{0xaa, 0xbb}}, // 2 bytes only
		},
	}
	pub := &fakePublisher{}
	c := iftable.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	row := pub.got[0].Rows[0]
	// Should NOT be the canonical 6-byte mac format
	if strings.Count(row.IfPhysAddr, ":") == 5 {
		t.Errorf("malformed MAC should NOT format as 6-byte canonical, got %q", row.IfPhysAddr)
	}
}

func TestCollect_MalformedOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		ifTableVbs: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.2.2.1.2.1", Value: "good"},
			{OID: "1.3.6.1.2.1.2.2.1.2.notanindex", Value: "bad"},
			{OID: "garbage-oid", Value: "discarded"},
		},
	}
	pub := &fakePublisher{}
	c := iftable.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Rows) != 1 || pub.got[0].Rows[0].IfDescr != "good" {
		t.Errorf("malformed OIDs should be skipped; got rows = %+v", pub.got[0].Rows)
	}
}

func TestCollect_Counters(t *testing.T) {
	t.Parallel()
	ifTable := []snmp.Varbind{
		{OID: "1.3.6.1.2.1.2.2.1.10.1", Value: uint(1000)},
		{OID: "1.3.6.1.2.1.2.2.1.13.1", Value: uint(3)},
		{OID: "1.3.6.1.2.1.2.2.1.14.1", Value: uint(4)},
		{OID: "1.3.6.1.2.1.2.2.1.16.1", Value: uint(2000)},
		{OID: "1.3.6.1.2.1.2.2.1.19.1", Value: uint(5)},
		{OID: "1.3.6.1.2.1.2.2.1.20.1", Value: uint(6)},
	}
	hcIn := snmp.Varbind{OID: "1.3.6.1.2.1.31.1.1.1.6.1", Value: uint64(1 << 40)}
	hcOut := snmp.Varbind{OID: "1.3.6.1.2.1.31.1.1.1.10.1", Value: uint64(1<<40 + 1)}
	discontinuity := snmp.Varbind{OID: "1.3.6.1.2.1.31.1.1.1.19.1", Value: uint32(777)}

	tests := []struct {
		name string
		ifX  []snmp.Varbind
		want iftable.Counters
	}{
		{
			name: "32-bit octets without ifXTable",
			want: iftable.Counters{
				InOctets: 1000, OutOctets: 2000,
				InErrors: 4, OutErrors: 6, InDiscards: 3, OutDiscards: 5,
			},
		},
		{
			name: "64-bit octets when both HC columns are served",
			ifX:  []snmp.Varbind{hcIn, hcOut, discontinuity},
			want: iftable.Counters{
				InOctets: 1 << 40, OutOctets: 1<<40 + 1, HCOctets: true,
				InErrors: 4, OutErrors: 6, InDiscards: 3, OutDiscards: 5,
				Discontinuity: 777,
			},
		},
		{
			// Mixing a 64-bit in with a 32-bit out would rate the two at
			// different wrap widths; one width per row keeps them comparable.
			name: "one HC column alone falls back to 32-bit for both",
			ifX:  []snmp.Varbind{hcIn},
			want: iftable.Counters{
				InOctets: 1000, OutOctets: 2000,
				InErrors: 4, OutErrors: 6, InDiscards: 3, OutDiscards: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fc := &fakeClient{ifTableVbs: ifTable, ifXTableVbs: tt.ifX, upTime: uint32(4200)}
			pub := &fakePublisher{}
			if err := iftable.New(factoryFor(fc), pub, at).
				Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err != nil {
				t.Fatalf("Collect: %v", err)
			}
			if got := pub.got[0].Rows[0].Counters; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Counters = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCollect_EtherLikeCounters(t *testing.T) {
	t.Parallel()
	ifTable := []snmp.Varbind{
		{OID: "1.3.6.1.2.1.2.2.1.3.1", Value: uint32(53)}, // propVirtual: a VLAN interface
		{OID: "1.3.6.1.2.1.2.2.1.3.2", Value: uint32(6)},  // ethernetCsmacd
	}
	tests := []struct {
		name string
		dot3 []snmp.Varbind
		want map[string]uint64
	}{
		{
			name: "agent without the EtherLike-MIB",
		},
		{
			name: "every counter column, and the columns that are not counters",
			dot3: []snmp.Varbind{
				{OID: "1.3.6.1.2.1.10.7.2.1.1.2", Value: 2}, // dot3StatsIndex
				{OID: "1.3.6.1.2.1.10.7.2.1.2.2", Value: uint(1)},
				{OID: "1.3.6.1.2.1.10.7.2.1.3.2", Value: uint(2)},
				{OID: "1.3.6.1.2.1.10.7.2.1.4.2", Value: uint(3)},
				{OID: "1.3.6.1.2.1.10.7.2.1.5.2", Value: uint(4)},
				{OID: "1.3.6.1.2.1.10.7.2.1.6.2", Value: uint(5)},
				{OID: "1.3.6.1.2.1.10.7.2.1.7.2", Value: uint(6)},
				{OID: "1.3.6.1.2.1.10.7.2.1.8.2", Value: uint(7)},
				{OID: "1.3.6.1.2.1.10.7.2.1.9.2", Value: uint(8)},
				{OID: "1.3.6.1.2.1.10.7.2.1.10.2", Value: uint(9)},
				{OID: "1.3.6.1.2.1.10.7.2.1.11.2", Value: uint(10)},
				{OID: "1.3.6.1.2.1.10.7.2.1.13.2", Value: uint(11)},
				{OID: "1.3.6.1.2.1.10.7.2.1.16.2", Value: uint(12)},
				{OID: "1.3.6.1.2.1.10.7.2.1.17.2", Value: ".0.0"}, // dot3StatsEtherChipSet
				{OID: "1.3.6.1.2.1.10.7.2.1.18.2", Value: uint(13)},
				{OID: "1.3.6.1.2.1.10.7.2.1.19.2", Value: 3}, // dot3StatsDuplexStatus
			},
			want: map[string]uint64{
				iftable.Dot3AlignmentErrors:           1,
				iftable.Dot3FCSErrors:                 2,
				iftable.Dot3SingleCollisionFrames:     3,
				iftable.Dot3MultipleCollisionFrames:   4,
				iftable.Dot3SQETestErrors:             5,
				iftable.Dot3DeferredTransmissions:     6,
				iftable.Dot3LateCollisions:            7,
				iftable.Dot3ExcessiveCollisions:       8,
				iftable.Dot3InternalMacTransmitErrors: 9,
				iftable.Dot3CarrierSenseErrors:        10,
				iftable.Dot3FrameTooLongs:             11,
				iftable.Dot3InternalMacReceiveErrors:  12,
				iftable.Dot3SymbolErrors:              13,
			},
		},
		{
			// net-snmp on Linux serves only the columns its driver reports.
			// An unserved column must not read as a zero count.
			name: "sparse columns and a value that is not a counter",
			dot3: []snmp.Varbind{
				{OID: "1.3.6.1.2.1.10.7.2.1.3.2", Value: uint(0)},
				{OID: "1.3.6.1.2.1.10.7.2.1.8.2", Value: nil},
				{OID: "1.3.6.1.2.1.10.7.2.1.18.2", Value: "garbage"},
			},
			want: map[string]uint64{iftable.Dot3FCSErrors: 0},
		},
		{
			name: "a row for an index the ifTable does not list",
			dot3: []snmp.Varbind{{OID: "1.3.6.1.2.1.10.7.2.1.3.99", Value: uint(4)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fc := &fakeClient{ifTableVbs: ifTable, dot3Vbs: tt.dot3, upTime: uint32(4200)}
			pub := &fakePublisher{}
			if err := iftable.New(factoryFor(fc), pub, at).
				Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err != nil {
				t.Fatalf("Collect: %v", err)
			}
			rows := pub.got[0].Rows
			if len(rows) != 2 {
				t.Fatalf("rows = %+v, want ifIndex 1 and 2 only", rows)
			}
			if vlan := rows[0].Counters.EtherLike; vlan != nil {
				t.Errorf("VLAN interface EtherLike = %v, want nil", vlan)
			}
			if got := rows[1].Counters.EtherLike; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EtherLike = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollect_Dot3WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{upTime: uint32(1), dot3Err: errors.New("timeout")}
	err := iftable.New(factoryFor(fc), &fakePublisher{}, at).
		Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if err == nil || !strings.Contains(err.Error(), "dot3StatsTable") {
		t.Errorf("Collect error = %v, want the dot3StatsTable walk failure", err)
	}
}

func TestCollect_SysUpTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		upTime any
		want   *uint32
	}{
		{"served", uint32(4200), new(uint32(4200))},
		{"not served", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := &fakePublisher{}
			fc := &fakeClient{upTime: tt.upTime}
			if err := iftable.New(factoryFor(fc), pub, at).
				Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err != nil {
				t.Fatalf("Collect: %v", err)
			}
			got := pub.got[0].SysUpTime
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Errorf("SysUpTime = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollect_SysUpTimeGetErrorPropagates(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{getErr: errors.New("timeout")}
	if err := iftable.New(factoryFor(fc), &fakePublisher{}, at).
		Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected Get error to propagate")
	}
}
