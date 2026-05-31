package cdp_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/cdp"
)

type fakeClient struct {
	prefixSeen string
	vbs        []snmp.Varbind
	walkErr    error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by cdp")
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	f.prefixSeen = prefix
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	return f.vbs, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []cdp.Observation
	err error
}

func (p *fakePublisher) PublishCDP(_ context.Context, obs cdp.Observation) error {
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

func cdpOID(col string, ifIdx, devIdx uint32) string {
	return fmt.Sprintf("%s.%s.%d.%d", cdp.DefaultTablePrefix, col, ifIdx, devIdx)
}

func TestCollector_Name_DefaultsToCDP(t *testing.T) {
	t.Parallel()
	c := cdp.New(nil, nil, at)
	if c.Name() != cdp.Name {
		t.Errorf("Name() = %q, want %q", c.Name(), cdp.Name)
	}
}

func TestCollector_WithNameOverridesForFDP(t *testing.T) {
	t.Parallel()
	c := cdp.New(nil, nil, at, cdp.WithName("fdp"))
	if c.Name() != "fdp" {
		t.Errorf("Name() = %q, want fdp", c.Name())
	}
}

func TestCollect_BuildsNeighborFromCompleteRow(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: cdpOID("3", 1, 1), Value: 1},
		{OID: cdpOID("4", 1, 1), Value: []byte{10, 0, 0, 1}},
		{OID: cdpOID("5", 1, 1), Value: "Cisco IOS XE 17.6"},
		{OID: cdpOID("6", 1, 1), Value: "core-sw-1.example.com"},
		{OID: cdpOID("7", 1, 1), Value: "GigabitEthernet1/0/24"},
		{OID: cdpOID("8", 1, 1), Value: "cisco WS-C9300-24P"},
		{OID: cdpOID("9", 1, 1), Value: uint32(0x28)},
	}}
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Neighbors) != 1 {
		t.Fatalf("got %+v, want 1 neighbor", pub.got)
	}
	n := pub.got[0].Neighbors[0]
	if n.LocalIfIndex != 1 || n.DeviceIndex != 1 {
		t.Errorf("index = (%d,%d), want (1,1)", n.LocalIfIndex, n.DeviceIndex)
	}
	if n.DeviceID != "core-sw-1.example.com" {
		t.Errorf("DeviceID = %q", n.DeviceID)
	}
	if n.DevicePort != "GigabitEthernet1/0/24" {
		t.Errorf("DevicePort = %q", n.DevicePort)
	}
	if n.Platform != "cisco WS-C9300-24P" {
		t.Errorf("Platform = %q", n.Platform)
	}
	if n.Address != "10.0.0.1" {
		t.Errorf("Address = %q, want 10.0.0.1", n.Address)
	}
	if n.Capabilities != 0x28 {
		t.Errorf("Capabilities = 0x%x, want 0x28", n.Capabilities)
	}
}

func TestCollect_IPv6AddressDecodes(t *testing.T) {
	t.Parallel()
	v6 := []byte{
		0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0x01,
	}
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: cdpOID("3", 1, 1), Value: 20},
		{OID: cdpOID("4", 1, 1), Value: v6},
	}}
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if pub.got[0].Neighbors[0].Address != "2001:db8::1" {
		t.Errorf("Address = %q, want 2001:db8::1", pub.got[0].Neighbors[0].Address)
	}
}

func TestCollect_NeighborsSortedByIfIndexThenDeviceIndex(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: cdpOID("6", 10, 2), Value: "c"},
		{OID: cdpOID("6", 10, 1), Value: "b"},
		{OID: cdpOID("6", 1, 1), Value: "a"},
	}}
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	got := pub.got[0].Neighbors
	wantOrder := []struct{ ifIdx, devIdx uint32 }{{1, 1}, {10, 1}, {10, 2}}
	for i, w := range wantOrder {
		if got[i].LocalIfIndex != w.ifIdx || got[i].DeviceIndex != w.devIdx {
			t.Errorf("[%d] = (%d,%d), want (%d,%d)",
				i, got[i].LocalIfIndex, got[i].DeviceIndex, w.ifIdx, w.devIdx)
		}
	}
}

func TestCollect_WithTablePrefixSwitchesToFDP(t *testing.T) {
	t.Parallel()
	const foundryPrefix = "1.3.6.1.4.1.1991.1.1.3.2.2.1"
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: foundryPrefix + ".6.1.1", Value: "foundry-edge-1"},
	}}
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(fc), pub, at,
		cdp.WithName("fdp"),
		cdp.WithTablePrefix(foundryPrefix),
	)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if fc.prefixSeen != foundryPrefix {
		t.Errorf("walked %q, want %q", fc.prefixSeen, foundryPrefix)
	}
	if len(pub.got[0].Neighbors) != 1 || pub.got[0].Neighbors[0].DeviceID != "foundry-edge-1" {
		t.Errorf("foundry row not parsed; got %+v", pub.got[0].Neighbors)
	}
	if pub.got[0].TablePrefix != foundryPrefix {
		t.Errorf("Observation.TablePrefix = %q, want %q", pub.got[0].TablePrefix, foundryPrefix)
	}
}

func TestCollect_MalformedOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: cdpOID("6", 1, 1), Value: "good"},
		{OID: "garbage", Value: "bad"},
		{OID: cdp.DefaultTablePrefix + ".6.notanint.1", Value: "bad"},
	}}
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Neighbors) != 1 || pub.got[0].Neighbors[0].DeviceID != "good" {
		t.Errorf("malformed should be skipped, got %+v", pub.got[0].Neighbors)
	}
}

func TestCollect_EmptyCacheStillPublishes(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(&fakeClient{}), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if len(pub.got) != 1 || len(pub.got[0].Neighbors) != 0 {
		t.Errorf("empty cache should publish empty observation, got %+v", pub.got)
	}
}

func TestCollect_WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := cdp.New(
		factoryFor(&fakeClient{walkErr: errors.New("timeout")}),
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected walk error to propagate")
	}
}

func TestCollect_PublishErrorPropagates(t *testing.T) {
	t.Parallel()
	c := cdp.New(
		factoryFor(&fakeClient{}),
		&fakePublisher{err: errors.New("topo busy")},
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
		c    *cdp.Collector
	}{
		{"nil factory", cdp.New(nil, &fakePublisher{}, at)},
		{"nil publisher", cdp.New(factoryFor(&fakeClient{}), nil, at)},
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
	c := cdp.New(
		func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
			return nil, errors.New("creds failed")
		},
		&fakePublisher{},
		at,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected factory error to propagate")
	}
}

func TestCollect_AddressFallsBackToMacWhenSixBytesAndTypeUnknown(t *testing.T) {
	t.Parallel()
	// No address-type row, but a 6-byte value should fall back to MAC.
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: cdpOID("4", 1, 1), Value: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}},
	}}
	pub := &fakePublisher{}
	c := cdp.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if pub.got[0].Neighbors[0].Address != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("Address fallback = %q", pub.got[0].Neighbors[0].Address)
	}
}
