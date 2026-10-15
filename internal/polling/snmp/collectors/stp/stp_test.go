package stp_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/stp"
)

const (
	portTable    = "1.3.6.1.2.1.17.2.15.1."
	ifIndexTable = "1.3.6.1.2.1.17.1.4.1.2."
)

type fakeClient struct {
	scalars []snmp.Varbind
	walks   map[string][]snmp.Varbind
	getErr  error
	walkErr error
}

func (f *fakeClient) Get(_ context.Context, oids []string) ([]snmp.Varbind, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	// Answer in request order, nil for anything the agent lacks.
	out := make([]snmp.Varbind, 0, len(oids))
	for _, oid := range oids {
		vb := snmp.Varbind{OID: oid}
		for _, s := range f.scalars {
			if s.OID == oid {
				vb.Value = s.Value
			}
		}
		out = append(out, vb)
	}
	return out, nil
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	return f.walks[prefix], nil
}

func (f *fakeClient) WalkLimit(_ context.Context, _ string, _ int) ([]snmp.Varbind, error) {
	return nil, errors.New("WalkLimit not used by stp")
}

type fakePublisher struct {
	got []stp.Observation
	err error
}

func (p *fakePublisher) PublishSTP(_ context.Context, obs stp.Observation) error {
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

func at() time.Time { return time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC) }

func walks(ports, ifIndexes []snmp.Varbind) map[string][]snmp.Varbind {
	return map[string][]snmp.Varbind{
		"1.3.6.1.2.1.17.2.15.1":  ports,
		"1.3.6.1.2.1.17.1.4.1.2": ifIndexes,
	}
}

func collect(t *testing.T, fc *fakeClient) stp.Observation {
	t.Helper()
	pub := &fakePublisher{}
	if err := stp.New(factoryFor(fc), pub, at).Collect(
		context.Background(), snmp.Target{ID: "sw-1", ClientID: "default"}, snmp.ResolvedCredentials{},
	); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("published %d observations, want 1", len(pub.got))
	}
	return pub.got[0]
}

// The values are copied from niac's cisco-c3750-03 capture walk: a non-root
// switch (32769/00:1c:b1:da:2c:80) whose root port 100 faces the upstream
// designated bridge, every other port designated by itself.
func TestCollect_CaptureFromNonRootSwitch(t *testing.T) {
	t.Parallel()
	self := []byte{0x80, 0x01, 0x00, 0x1c, 0xb1, 0xda, 0x2c, 0x80}
	upstream := []byte{0x80, 0x01, 0x00, 0x11, 0x93, 0x4d, 0x63, 0x00}
	fc := &fakeClient{
		scalars: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.17.1.1.0", Value: []byte{0x00, 0x1c, 0xb1, 0xda, 0x2c, 0x80}},
			{OID: "1.3.6.1.2.1.17.2.1.0", Value: 3},
			{OID: "1.3.6.1.2.1.17.2.2.0", Value: 32769},
			{OID: "1.3.6.1.2.1.17.2.3.0", Value: uint32(5500600)},
			{OID: "1.3.6.1.2.1.17.2.4.0", Value: uint(45)},
			{OID: "1.3.6.1.2.1.17.2.5.0", Value: []byte{0x80, 0x00, 0x00, 0x01, 0xe7, 0x00, 0x9b, 0x80}},
			{OID: "1.3.6.1.2.1.17.2.6.0", Value: 52},
			{OID: "1.3.6.1.2.1.17.2.7.0", Value: 100},
		},
		walks: walks([]snmp.Varbind{
			{OID: "." + portTable + "3.13", Value: 5},
			{OID: "." + portTable + "3.100", Value: 5},
			{OID: "." + portTable + "4.13", Value: 1},
			{OID: "." + portTable + "4.100", Value: 1},
			{OID: "." + portTable + "5.13", Value: 19},
			{OID: "." + portTable + "8.13", Value: self},
			{OID: "." + portTable + "8.100", Value: upstream},
			{OID: "." + portTable + "10.13", Value: uint(1)},
			{OID: "." + portTable + "10.100", Value: uint(1)},
		}, []snmp.Varbind{
			{OID: ifIndexTable + "13", Value: 10113},
			{OID: ifIndexTable + "100", Value: 10646},
		}),
	}

	got := collect(t, fc)
	if got.TargetID != "sw-1" || got.ClientID != "default" || !got.ObservedAt.Equal(at()) {
		t.Errorf("identity = %q/%q/%v", got.TargetID, got.ClientID, got.ObservedAt)
	}
	if want := (stp.BridgeID{Priority: 32769, Address: "00:1c:b1:da:2c:80"}); got.Bridge != want {
		t.Errorf("Bridge = %v, want %v", got.Bridge, want)
	}
	if want := (stp.BridgeID{Priority: 32768, Address: "00:01:e7:00:9b:80"}); got.Root != want {
		t.Errorf("Root = %v, want %v", got.Root, want)
	}
	if got.Root.String() != "32768/00:01:e7:00:9b:80" {
		t.Errorf("Root.String() = %q", got.Root.String())
	}
	if got.Protocol != 3 || got.RootCost != 52 || got.RootPort != 100 ||
		got.TopologyChanges != 45 || got.SecondsSinceTopologyChange != 55006 {
		t.Errorf("scalars = %+v", got)
	}
	want := []stp.Port{
		{
			Port: 13, IfIndex: 10113, State: stp.StateForwarding, Role: stp.RoleDesignated, PathCost: 19,
			DesignatedBridge: stp.BridgeID{Priority: 32769, Address: "00:1c:b1:da:2c:80"}, ForwardTransitions: 1,
		},
		{
			Port: 100, IfIndex: 10646, State: stp.StateForwarding, Role: stp.RoleRoot,
			DesignatedBridge: stp.BridgeID{Priority: 32769, Address: "00:11:93:4d:63:00"}, ForwardTransitions: 1,
		},
	}
	if !slices.Equal(got.Ports, want) {
		t.Errorf("Ports =\n%+v\nwant\n%+v", got.Ports, want)
	}
}

func TestCollect_PortRoles(t *testing.T) {
	t.Parallel()
	selfMAC := []byte{0x02, 0, 0, 0, 0, 0x01}
	self := append([]byte{0x80, 0x00}, selfMAC...)
	other := []byte{0x80, 0x00, 0x02, 0, 0, 0, 0, 0x02}
	tests := []struct {
		name       string
		bridgeMAC  any
		state      int
		enable     int
		designated []byte
		want       string
	}{
		{"root port", selfMAC, stp.StateForwarding, 1, other, stp.RoleRoot},
		{"designated", selfMAC, stp.StateForwarding, 1, self, stp.RoleDesignated},
		{"backup: own segment, blocking", selfMAC, stp.StateBlocking, 1, self, stp.RoleBackup},
		{"alternate: another bridge designated", selfMAC, stp.StateBlocking, 1, other, stp.RoleAlternate},
		{"disabled by state", selfMAC, stp.StateDisabled, 1, self, stp.RoleDisabled},
		{"disabled by dot1dStpPortEnable", selfMAC, stp.StateForwarding, 2, self, stp.RoleDisabled},
		{"own bridge unknown", nil, stp.StateBlocking, 1, other, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Port 1 is always the root port, so the port under test is 1
			// only in the root-port case.
			port := "2"
			if tt.want == stp.RoleRoot {
				port = "1"
			}
			fc := &fakeClient{
				scalars: []snmp.Varbind{
					{OID: "1.3.6.1.2.1.17.1.1.0", Value: tt.bridgeMAC},
					{OID: "1.3.6.1.2.1.17.2.2.0", Value: 32768},
					{OID: "1.3.6.1.2.1.17.2.5.0", Value: other},
					{OID: "1.3.6.1.2.1.17.2.7.0", Value: 1},
				},
				walks: walks([]snmp.Varbind{
					{OID: portTable + "3." + port, Value: tt.state},
					{OID: portTable + "4." + port, Value: tt.enable},
					{OID: portTable + "8." + port, Value: tt.designated},
				}, nil),
			}
			got := collect(t, fc)
			if len(got.Ports) != 1 || got.Ports[0].Role != tt.want {
				t.Errorf("Ports = %+v, want one port with role %q", got.Ports, tt.want)
			}
		})
	}
}

// A router or a switch with spanning tree off answers none of dot1dStp; it
// still publishes, with no root, so the alert pipeline can tell it apart.
func TestCollect_NoSpanningTree(t *testing.T) {
	t.Parallel()
	got := collect(t, &fakeClient{})
	if got.Root != (stp.BridgeID{}) || got.Bridge != (stp.BridgeID{}) || len(got.Ports) != 0 {
		t.Errorf("observation = %+v, want no root, bridge or ports", got)
	}
}

func TestCollect_Errors(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	tests := []struct {
		name string
		c    *stp.Collector
	}{
		{"no factory", stp.New(nil, &fakePublisher{}, at)},
		{"no publisher", stp.New(factoryFor(&fakeClient{}), nil, at)},
		{"dial", stp.New(func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) {
			return nil, boom
		}, &fakePublisher{}, at)},
		{"get", stp.New(factoryFor(&fakeClient{getErr: boom}), &fakePublisher{}, at)},
		{"walk", stp.New(factoryFor(&fakeClient{walkErr: boom}), &fakePublisher{}, at)},
		{"publish", stp.New(factoryFor(&fakeClient{}), &fakePublisher{err: boom}, at)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
				t.Error("Collect succeeded, want an error")
			}
		})
	}
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	if got := stp.New(nil, nil, nil).Name(); got != stp.Name {
		t.Errorf("Name = %q, want %q", got, stp.Name)
	}
}
