package fdb_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
)

const (
	basePortPrefix = "1.3.6.1.2.1.17.1.4.1"
	tpFdbPrefix    = "1.3.6.1.2.1.17.7.1.2.2.1"
)

type fakeClient struct {
	basePortVbs []snmp.Varbind
	tpFdbVbs    []snmp.Varbind
	walkErr     error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by fdb")
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	switch {
	case strings.HasPrefix(prefix, basePortPrefix):
		return f.basePortVbs, nil
	case strings.HasPrefix(prefix, tpFdbPrefix):
		return f.tpFdbVbs, nil
	}
	return nil, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []fdb.Observation
	err error
}

func (p *fakePublisher) PublishFDB(_ context.Context, obs fdb.Observation) error {
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

func basePortOID(col string, port uint32) string {
	return fmt.Sprintf("%s.%s.%d", basePortPrefix, col, port)
}

func fdbOID(col string, vlan uint32, mac [6]uint32) string {
	return fmt.Sprintf("%s.%s.%d.%d.%d.%d.%d.%d.%d",
		tpFdbPrefix, col, vlan, mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	if fdb.New(nil, nil, at).Name() != fdb.Name {
		t.Error("Name mismatch")
	}
}

func TestCollect_JoinsBasePortMapWithFdbRows(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		basePortVbs: []snmp.Varbind{
			{OID: basePortOID("2", 24), Value: uint32(10024)},
			{OID: basePortOID("2", 48), Value: uint32(10048)},
		},
		tpFdbVbs: []snmp.Varbind{
			{OID: fdbOID("2", 100, [6]uint32{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}), Value: uint32(24)},
			{OID: fdbOID("3", 100, [6]uint32{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}), Value: 3},
			{OID: fdbOID("2", 200, [6]uint32{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}), Value: uint32(48)},
			{OID: fdbOID("3", 200, [6]uint32{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}), Value: 3},
		},
	}
	pub := &fakePublisher{}
	c := fdb.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Entries) != 2 {
		t.Fatalf("got %+v, want 2 entries", pub.got)
	}
	e0 := pub.got[0].Entries[0]
	if e0.VLANID != 100 || e0.MACAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("e0 = %+v", e0)
	}
	if e0.BridgePort != 24 || e0.IfIndex != 10024 {
		t.Errorf("e0 port/ifindex = %d/%d, want 24/10024", e0.BridgePort, e0.IfIndex)
	}
	if e0.Status != fdb.StatusLearned {
		t.Errorf("e0.Status = %d, want %d", e0.Status, fdb.StatusLearned)
	}
}

func TestCollect_UnknownBridgePortIfIndexZero(t *testing.T) {
	t.Parallel()
	// basePortTable has port 24 → ifindex 10024 but the fdb row
	// references port 99 which has no mapping.
	fc := &fakeClient{
		basePortVbs: []snmp.Varbind{
			{OID: basePortOID("2", 24), Value: uint32(10024)},
		},
		tpFdbVbs: []snmp.Varbind{
			{OID: fdbOID("2", 100, [6]uint32{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}), Value: uint32(99)},
		},
	}
	pub := &fakePublisher{}
	c := fdb.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	e := pub.got[0].Entries[0]
	if e.BridgePort != 99 {
		t.Errorf("BridgePort = %d, want 99", e.BridgePort)
	}
	if e.IfIndex != 0 {
		t.Errorf("IfIndex = %d, want 0 for unmapped port", e.IfIndex)
	}
}

func TestCollect_SortedByVLANThenMAC(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		tpFdbVbs: []snmp.Varbind{
			{OID: fdbOID("2", 200, [6]uint32{0xff, 0, 0, 0, 0, 0}), Value: uint32(1)},
			{OID: fdbOID("2", 100, [6]uint32{0xff, 0, 0, 0, 0, 0}), Value: uint32(1)},
			{OID: fdbOID("2", 100, [6]uint32{0x11, 0, 0, 0, 0, 0}), Value: uint32(1)},
		},
	}
	pub := &fakePublisher{}
	c := fdb.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	got := pub.got[0].Entries
	want := []struct {
		vlan uint32
		mac  string
	}{{100, "11:00:00:00:00:00"}, {100, "ff:00:00:00:00:00"}, {200, "ff:00:00:00:00:00"}}
	for i, w := range want {
		if got[i].VLANID != w.vlan || got[i].MACAddress != w.mac {
			t.Errorf("[%d] = (%d,%s), want (%d,%s)",
				i, got[i].VLANID, got[i].MACAddress, w.vlan, w.mac)
		}
	}
}

func TestCollect_MalformedOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		tpFdbVbs: []snmp.Varbind{
			{OID: fdbOID("2", 100, [6]uint32{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}), Value: uint32(1)},
			{OID: tpFdbPrefix + ".2.100.999.0.0.0.0.0", Value: uint32(1)}, // octet > 255
			{OID: "garbage", Value: nil},
		},
	}
	pub := &fakePublisher{}
	c := fdb.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Entries) != 1 {
		t.Errorf("malformed should be skipped, got %d entries", len(pub.got[0].Entries))
	}
}

func TestCollect_EmptyTablesStillPublish(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := fdb.New(factoryFor(&fakeClient{}), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Entries) != 0 {
		t.Errorf("empty FDB should publish empty obs, got %+v", pub.got)
	}
}

func TestCollect_BasePortWalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := fdb.New(
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
	c := fdb.New(
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
		c    *fdb.Collector
	}{
		{"nil factory", fdb.New(nil, &fakePublisher{}, at)},
		{"nil publisher", fdb.New(factoryFor(&fakeClient{}), nil, at)},
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
