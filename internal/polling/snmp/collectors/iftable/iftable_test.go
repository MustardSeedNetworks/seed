package iftable_test

import (
	"context"
	"errors"
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
	walkErr     error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by iftable")
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
