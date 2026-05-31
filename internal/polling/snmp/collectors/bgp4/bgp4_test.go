package bgp4_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/bgp4"
)

const tablePrefix = "1.3.6.1.2.1.15.3.1"

type fakeClient struct {
	vbs     []snmp.Varbind
	walkErr error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by bgp4")
}

func (f *fakeClient) Walk(_ context.Context, _ string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	return f.vbs, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []bgp4.Observation
	err error
}

func (p *fakePublisher) PublishBGP4(_ context.Context, obs bgp4.Observation) error {
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

func peerOID(col, remoteAddr string) string {
	return fmt.Sprintf("%s.%s.%s", tablePrefix, col, remoteAddr)
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	if bgp4.New(nil, nil, at).Name() != bgp4.Name {
		t.Error("Name mismatch")
	}
}

func TestCollect_BuildsPeerFromColumns(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: peerOID("1", "10.0.0.1"), Value: []byte{192, 0, 2, 1}},
		{OID: peerOID("2", "10.0.0.1"), Value: bgp4.StateEstablished},
		{OID: peerOID("3", "10.0.0.1"), Value: bgp4.AdminStart},
		{OID: peerOID("4", "10.0.0.1"), Value: 4},
		{OID: peerOID("5", "10.0.0.1"), Value: []byte{10, 0, 0, 100}},
		{OID: peerOID("6", "10.0.0.1"), Value: 179},
		{OID: peerOID("8", "10.0.0.1"), Value: 12345},
		{OID: peerOID("9", "10.0.0.1"), Value: uint32(65001)},
		{OID: peerOID("10", "10.0.0.1"), Value: uint32(987654)},
		{OID: peerOID("11", "10.0.0.1"), Value: uint32(123456)},
		{OID: peerOID("12", "10.0.0.1"), Value: uint32(2000000)},
		{OID: peerOID("13", "10.0.0.1"), Value: uint32(500000)},
		{OID: peerOID("15", "10.0.0.1"), Value: uint32(3)},
		{OID: peerOID("16", "10.0.0.1"), Value: uint32(86400)},
	}}
	pub := &fakePublisher{}
	c := bgp4.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Peers) != 1 {
		t.Fatalf("got %+v, want 1 peer", pub.got)
	}
	p := pub.got[0].Peers[0]
	if p.RemoteAddr != "10.0.0.1" {
		t.Errorf("RemoteAddr = %q", p.RemoteAddr)
	}
	if p.Identifier != "192.0.2.1" {
		t.Errorf("Identifier = %q, want 192.0.2.1", p.Identifier)
	}
	if p.LocalAddr != "10.0.0.100" {
		t.Errorf("LocalAddr = %q", p.LocalAddr)
	}
	if p.State != bgp4.StateEstablished || p.AdminStatus != bgp4.AdminStart {
		t.Errorf("state/admin = %d/%d", p.State, p.AdminStatus)
	}
	if p.RemoteAS != 65001 {
		t.Errorf("RemoteAS = %d", p.RemoteAS)
	}
	if p.InUpdates != 987654 || p.OutUpdates != 123456 {
		t.Errorf("updates in/out = %d/%d", p.InUpdates, p.OutUpdates)
	}
	if p.InTotalMessages != 2000000 || p.OutTotalMessages != 500000 {
		t.Errorf("total in/out = %d/%d", p.InTotalMessages, p.OutTotalMessages)
	}
	if p.EstablishedTransitions != 3 {
		t.Errorf("EstablishedTransitions = %d", p.EstablishedTransitions)
	}
	if p.EstablishedTimeSeconds != 86400 {
		t.Errorf("EstablishedTimeSeconds = %d", p.EstablishedTimeSeconds)
	}
}

func TestCollect_PeersSortedByRemoteAddr(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: peerOID("2", "192.168.1.1"), Value: bgp4.StateIdle},
		{OID: peerOID("2", "10.0.0.1"), Value: bgp4.StateEstablished},
		{OID: peerOID("2", "10.0.0.10"), Value: bgp4.StateActive},
	}}
	pub := &fakePublisher{}
	c := bgp4.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	want := []string{"10.0.0.1", "10.0.0.10", "192.168.1.1"}
	for i, w := range want {
		if pub.got[0].Peers[i].RemoteAddr != w {
			t.Errorf("[%d] = %s, want %s", i, pub.got[0].Peers[i].RemoteAddr, w)
		}
	}
}

func TestCollect_MalformedOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: peerOID("2", "10.0.0.1"), Value: bgp4.StateEstablished},
		{OID: tablePrefix + ".2.999.0.0.1", Value: bgp4.StateIdle}, // octet > 255
		{OID: "garbage", Value: nil},
	}}
	pub := &fakePublisher{}
	c := bgp4.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Peers) != 1 {
		t.Errorf("malformed should be skipped, got %d peers", len(pub.got[0].Peers))
	}
}

func TestCollect_EmptyTableStillPublishes(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := bgp4.New(factoryFor(&fakeClient{}), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if len(pub.got) != 1 || len(pub.got[0].Peers) != 0 {
		t.Errorf("empty table should publish empty obs, got %+v", pub.got)
	}
}

func TestCollect_WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := bgp4.New(
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
	c := bgp4.New(
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
		c    *bgp4.Collector
	}{
		{"nil factory", bgp4.New(nil, &fakePublisher{}, at)},
		{"nil publisher", bgp4.New(factoryFor(&fakeClient{}), nil, at)},
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
