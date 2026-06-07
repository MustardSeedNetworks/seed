package arp_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
)

const tablePrefix = "1.3.6.1.2.1.4.22.1"

type fakeClient struct {
	vbs     []snmp.Varbind
	walkErr error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by arp")
}

func (f *fakeClient) Walk(_ context.Context, _ string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	return f.vbs, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []arp.Observation
	err error
}

func (p *fakePublisher) PublishARP(_ context.Context, obs arp.Observation) error {
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

func arpOID(col string, ifIdx uint32, ip string) string {
	return fmt.Sprintf("%s.%s.%d.%s", tablePrefix, col, ifIdx, ip)
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	if arp.New(nil, nil, at).Name() != arp.Name {
		t.Error("Name() did not equal arp.Name")
	}
}

func TestCollect_BuildsEntriesFromTable(t *testing.T) {
	t.Parallel()
	mac := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: arpOID("2", 1, "10.0.0.1"), Value: mac},
		{OID: arpOID("4", 1, "10.0.0.1"), Value: arp.MediaTypeDynamic},
		{OID: arpOID("2", 1, "10.0.0.2"), Value: []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}},
		{OID: arpOID("4", 1, "10.0.0.2"), Value: arp.MediaTypeStatic},
	}}
	pub := &fakePublisher{}
	c := arp.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Entries) != 2 {
		t.Fatalf("got %+v, want 2 entries", pub.got)
	}
	e0 := pub.got[0].Entries[0]
	if e0.IfIndex != 1 || e0.IPAddress != "10.0.0.1" {
		t.Errorf("e0 = %+v", e0)
	}
	if e0.MACAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("e0.MAC = %q", e0.MACAddress)
	}
	if e0.MediaType != arp.MediaTypeDynamic {
		t.Errorf("e0.MediaType = %d, want %d", e0.MediaType, arp.MediaTypeDynamic)
	}
	if pub.got[0].Entries[1].MediaType != arp.MediaTypeStatic {
		t.Errorf("e1.MediaType = %d, want static", pub.got[0].Entries[1].MediaType)
	}
}

func TestCollect_SortedByIfIndexThenIP(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: arpOID("4", 10, "10.0.0.5"), Value: 3},
		{OID: arpOID("4", 1, "10.0.0.250"), Value: 3},
		{OID: arpOID("4", 1, "10.0.0.5"), Value: 3},
	}}
	pub := &fakePublisher{}
	c := arp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	got := pub.got[0].Entries
	want := []struct {
		idx uint32
		ip  string
	}{{1, "10.0.0.250"}, {1, "10.0.0.5"}, {10, "10.0.0.5"}}
	// IP is sorted lexicographically — "10.0.0.250" < "10.0.0.5" in string order.
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].IfIndex != w.idx || got[i].IPAddress != w.ip {
			t.Errorf("[%d] = (%d,%s), want (%d,%s)",
				i, got[i].IfIndex, got[i].IPAddress, w.idx, w.ip)
		}
	}
}

func TestCollect_MalformedOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: arpOID("2", 1, "10.0.0.1"), Value: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}},
		{OID: "garbage", Value: nil},
		{OID: tablePrefix + ".2.notanum.10.0.0.2", Value: nil},
		{OID: tablePrefix + ".2.1.999.0.0.1", Value: nil}, // octet > 255
	}}
	pub := &fakePublisher{}
	c := arp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Entries) != 1 {
		t.Errorf("malformed should be skipped, got %d entries", len(pub.got[0].Entries))
	}
}

func TestCollect_EmptyTableStillPublishes(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := arp.New(factoryFor(&fakeClient{}), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if len(pub.got) != 1 || len(pub.got[0].Entries) != 0 {
		t.Errorf("empty table should publish empty obs, got %+v", pub.got)
	}
}

func TestCollect_WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := arp.New(
		factoryFor(&fakeClient{walkErr: errors.New("timeout")}),
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected walk error")
	}
}

func TestCollect_PublishErrorPropagates(t *testing.T) {
	t.Parallel()
	c := arp.New(
		factoryFor(&fakeClient{}),
		&fakePublisher{err: errors.New("topo busy")},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected publish error")
	}
}

func TestCollect_NilDepsReturnError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    *arp.Collector
	}{
		{"nil factory", arp.New(nil, &fakePublisher{}, at)},
		{"nil publisher", arp.New(factoryFor(&fakeClient{}), nil, at)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
				t.Error("expected nil-dep error")
			}
		})
	}
}

func TestCollect_FactoryErrorPropagates(t *testing.T) {
	t.Parallel()
	c := arp.New(
		func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
			return nil, errors.New("creds failed")
		},
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected factory error")
	}
}
