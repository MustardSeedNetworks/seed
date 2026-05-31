package lldp_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/lldp"
)

type fakeClient struct {
	vbs     []snmp.Varbind
	walkErr error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by lldp")
}

func (f *fakeClient) Walk(_ context.Context, _ string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	return f.vbs, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []lldp.Observation
	err error
}

func (p *fakePublisher) PublishLLDP(_ context.Context, obs lldp.Observation) error {
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

const remPrefix = "1.0.8802.1.1.2.1.4.1.1"

// remOID builds a fully-qualified lldpRemTable OID for a column +
// index triple. timeMark=1 throughout the tests since most agents
// reset it on every poll cycle.
func remOID(col string, localPort, remIndex uint32) string {
	return remPrefix + "." + col + ".1." +
		uint32Str(localPort) + "." + uint32Str(remIndex)
}

func uint32Str(v uint32) string {
	switch v {
	case 1:
		return "1"
	case 2:
		return "2"
	case 10:
		return "10"
	case 100:
		return "100"
	}
	// fallback: small test values only
	return "0"
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	c := lldp.New(nil, nil, at)
	if c.Name() != lldp.Name {
		t.Errorf("Name() = %q, want %q", c.Name(), lldp.Name)
	}
}

func TestCollect_BuildsNeighborFromCompleteRow(t *testing.T) {
	t.Parallel()
	mac := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: remOID("4", 1, 1), Value: 4},
		{OID: remOID("5", 1, 1), Value: mac},
		{OID: remOID("6", 1, 1), Value: 5},
		{OID: remOID("7", 1, 1), Value: "Gi0/24"},
		{OID: remOID("8", 1, 1), Value: "uplink"},
		{OID: remOID("9", 1, 1), Value: "core-sw-1"},
		{OID: remOID("10", 1, 1), Value: "JunOS 22.4"},
		{OID: remOID("11", 1, 1), Value: []byte{0x28, 0x00}},
		{OID: remOID("12", 1, 1), Value: []byte{0x28, 0x00}},
	}}
	pub := &fakePublisher{}
	c := lldp.New(factoryFor(fc), pub, at)

	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Neighbors) != 1 {
		t.Fatalf("got %+v, want 1 neighbor", pub.got)
	}
	n := pub.got[0].Neighbors[0]
	if n.LocalPortNum != 1 || n.LldpRemIndex != 1 {
		t.Errorf("index = (port=%d, rem=%d), want (1,1)", n.LocalPortNum, n.LldpRemIndex)
	}
	if n.ChassisID != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("ChassisID = %q, want aa:bb:cc:dd:ee:ff", n.ChassisID)
	}
	if n.ChassisIDSubtype != 4 || n.PortIDSubtype != 5 {
		t.Errorf("subtypes = chassis=%d port=%d, want 4 + 5", n.ChassisIDSubtype, n.PortIDSubtype)
	}
	if n.PortID != "Gi0/24" {
		t.Errorf("PortID = %q", n.PortID)
	}
	if n.SysName != "core-sw-1" {
		t.Errorf("SysName = %q", n.SysName)
	}
	if n.SysCapEnabled == 0 {
		t.Errorf("SysCapEnabled = 0, want decoded BITS")
	}
}

func TestCollect_NeighborsSortedByLocalPortThenRemIndex(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: remOID("9", 10, 2), Value: "z"},
		{OID: remOID("9", 10, 1), Value: "y"},
		{OID: remOID("9", 1, 1), Value: "x"},
	}}
	pub := &fakePublisher{}
	c := lldp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	got := pub.got[0].Neighbors
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	want := []struct {
		port, idx uint32
	}{{1, 1}, {10, 1}, {10, 2}}
	for i, w := range want {
		if got[i].LocalPortNum != w.port || got[i].LldpRemIndex != w.idx {
			t.Errorf("neighbors[%d] = (%d,%d), want (%d,%d)",
				i, got[i].LocalPortNum, got[i].LldpRemIndex, w.port, w.idx)
		}
	}
}

func TestCollect_MalformedOIDsAreSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: remOID("9", 1, 1), Value: "good"},
		{OID: "garbage", Value: "bad"},
		{OID: remPrefix + ".9.1.notanum.1", Value: "bad"},
	}}
	pub := &fakePublisher{}
	c := lldp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	got := pub.got[0].Neighbors
	if len(got) != 1 || got[0].SysName != "good" {
		t.Errorf("malformed OIDs should be skipped, got %+v", got)
	}
}

func TestCollect_EmptyTableYieldsEmptyObservation(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: nil}
	pub := &fakePublisher{}
	c := lldp.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("publisher should be called even with empty table; got %d", len(pub.got))
	}
	if len(pub.got[0].Neighbors) != 0 {
		t.Errorf("neighbors = %d, want 0", len(pub.got[0].Neighbors))
	}
}

func TestCollect_WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := lldp.New(
		factoryFor(&fakeClient{walkErr: errors.New("snmp timeout")}),
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected walk error to propagate")
	}
}

func TestCollect_PublishErrorPropagates(t *testing.T) {
	t.Parallel()
	c := lldp.New(
		factoryFor(&fakeClient{}),
		&fakePublisher{err: errors.New("topology busy")},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected publish error to propagate")
	}
}

func TestCollect_NilDepsReturnError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    *lldp.Collector
	}{
		{"nil factory", lldp.New(nil, &fakePublisher{}, at)},
		{"nil publisher", lldp.New(factoryFor(&fakeClient{}), nil, at)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
				t.Error("expected error from nil dep")
			}
		})
	}
}

func TestCollect_FactoryErrorPropagates(t *testing.T) {
	t.Parallel()
	c := lldp.New(
		func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
			return nil, errors.New("creds resolution failed")
		},
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected factory error to propagate")
	}
}

func TestCollect_NonMACChassisIDFallsBackToString(t *testing.T) {
	t.Parallel()
	// 4-byte octet string is not a MAC subtype — should fall back
	// to raw stringification.
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: remOID("5", 1, 1), Value: []byte("name")},
	}}
	pub := &fakePublisher{}
	c := lldp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if pub.got[0].Neighbors[0].ChassisID != "name" {
		t.Errorf("ChassisID = %q, want %q", pub.got[0].Neighbors[0].ChassisID, "name")
	}
}
