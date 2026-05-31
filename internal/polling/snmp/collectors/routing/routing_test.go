package routing_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/routing"
)

const tablePrefix = "1.3.6.1.2.1.4.24.4.1"

type fakeClient struct {
	vbs     []snmp.Varbind
	walkErr error
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by routing")
}

func (f *fakeClient) Walk(_ context.Context, _ string) ([]snmp.Varbind, error) {
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	return f.vbs, nil
}

type fakePublisher struct {
	mu  sync.Mutex
	got []routing.Observation
	err error
}

func (p *fakePublisher) PublishRouting(_ context.Context, obs routing.Observation) error {
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

// routeOID composes a routing-table OID for a column + key tuple.
// All current tests use a /24 + tos=0; if non-default tests arrive,
// lift mask/tos into parameters.
func routeOID(col, dest, nextHop string) string {
	return fmt.Sprintf("%s.%s.%s.255.255.255.0.0.%s",
		tablePrefix, col, dest, nextHop)
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	if routing.New(nil, nil, at).Name() != routing.Name {
		t.Error("Name mismatch")
	}
}

func TestCollect_BuildsRouteFromColumns(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: routeOID("5", "10.0.0.0", "10.0.0.254"), Value: uint32(1)},
		{OID: routeOID("6", "10.0.0.0", "10.0.0.254"), Value: routing.TypeLocal},
		{OID: routeOID("7", "10.0.0.0", "10.0.0.254"), Value: routing.ProtoLocal},
		{OID: routeOID("8", "10.0.0.0", "10.0.0.254"), Value: uint32(3600)},
		{OID: routeOID("11", "10.0.0.0", "10.0.0.254"), Value: 1},
	}}
	pub := &fakePublisher{}
	c := routing.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 || len(pub.got[0].Routes) != 1 {
		t.Fatalf("got %+v, want 1 route", pub.got)
	}
	r := pub.got[0].Routes[0]
	if r.Destination != "10.0.0.0" || r.Mask != "255.255.255.0" {
		t.Errorf("dest/mask = %s/%s", r.Destination, r.Mask)
	}
	if r.NextHop != "10.0.0.254" {
		t.Errorf("NextHop = %q", r.NextHop)
	}
	if r.IfIndex != 1 || r.Type != routing.TypeLocal || r.Proto != routing.ProtoLocal {
		t.Errorf("if/type/proto = %d/%d/%d", r.IfIndex, r.Type, r.Proto)
	}
	if r.AgeSeconds != 3600 || r.Metric1 != 1 {
		t.Errorf("age/metric = %d/%d", r.AgeSeconds, r.Metric1)
	}
}

func TestCollect_SortedByDestThenMaskThenNextHop(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: routeOID("5", "192.168.1.0", "10.0.0.1"), Value: uint32(1)},
		{OID: routeOID("5", "10.0.0.0", "10.0.0.2"), Value: uint32(1)},
		{OID: routeOID("5", "10.0.0.0", "10.0.0.1"), Value: uint32(1)},
	}}
	pub := &fakePublisher{}
	c := routing.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	got := pub.got[0].Routes
	want := []struct{ dest, nh string }{
		{"10.0.0.0", "10.0.0.1"},
		{"10.0.0.0", "10.0.0.2"},
		{"192.168.1.0", "10.0.0.1"},
	}
	for i, w := range want {
		if got[i].Destination != w.dest || got[i].NextHop != w.nh {
			t.Errorf("[%d] = (%s,%s), want (%s,%s)",
				i, got[i].Destination, got[i].NextHop, w.dest, w.nh)
		}
	}
}

func TestCollect_MalformedOIDsSkipped(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: routeOID("5", "10.0.0.0", "10.0.0.254"), Value: uint32(1)},
		{OID: tablePrefix + ".5.10.0.0.0.255.255.255.0.0.10.0.0", Value: 1}, // 12 fields (wrong)
		{OID: "garbage", Value: nil},
	}}
	pub := &fakePublisher{}
	c := routing.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Routes) != 1 {
		t.Errorf("malformed should be skipped, got %d routes", len(pub.got[0].Routes))
	}
}

func TestCollect_EmptyTableStillPublishes(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := routing.New(factoryFor(&fakeClient{}), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if len(pub.got) != 1 || len(pub.got[0].Routes) != 0 {
		t.Errorf("empty table should publish empty obs, got %+v", pub.got)
	}
}

func TestCollect_WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := routing.New(
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
	c := routing.New(
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
		c    *routing.Collector
	}{
		{"nil factory", routing.New(nil, &fakePublisher{}, at)},
		{"nil publisher", routing.New(factoryFor(&fakeClient{}), nil, at)},
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

func TestCollect_OctetOverflowRejected(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: []snmp.Varbind{
		// Dest octet > 255 — should not produce a route.
		{OID: tablePrefix + ".5.999.0.0.1.255.255.255.0.0.10.0.0.1", Value: uint32(1)},
	}}
	pub := &fakePublisher{}
	c := routing.New(factoryFor(fc), pub, at)
	_ = c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})

	if len(pub.got[0].Routes) != 0 {
		t.Errorf("overflow should be skipped, got %d routes", len(pub.got[0].Routes))
	}
}
