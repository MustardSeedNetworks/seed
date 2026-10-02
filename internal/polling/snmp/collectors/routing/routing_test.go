package routing_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/routing"
)

const (
	tablePrefix       = "1.3.6.1.2.1.4.24.4.1"
	legacyTablePrefix = "1.3.6.1.2.1.4.21.1"
)

// fakeClient answers a Walk with the varbinds under the walked prefix, the
// way an agent does, and records the prefixes it was asked for.
type fakeClient struct {
	vbs       []snmp.Varbind
	walkErr   error
	legacyErr error
	walked    []string
}

func (f *fakeClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by routing")
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	f.walked = append(f.walked, prefix)
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	if prefix == legacyTablePrefix && f.legacyErr != nil {
		return nil, f.legacyErr
	}
	var out []snmp.Varbind
	for _, vb := range f.vbs {
		if strings.HasPrefix(vb.OID, prefix+".") {
			out = append(out, vb)
		}
	}
	return out, nil
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

// legacyRouteOID composes an ipRouteTable OID, indexed by destination alone.
func legacyRouteOID(col, dest string) string {
	return fmt.Sprintf("%s.%s.%s", legacyTablePrefix, col, dest)
}

// legacyRow is one ipRouteTable row's columns as an agent serves them, with
// mask and next hop as IpAddress strings.
func legacyRow(dest, mask, nextHop string, routeType int) []snmp.Varbind {
	return []snmp.Varbind{
		{OID: legacyRouteOID("1", dest), Value: dest},
		{OID: legacyRouteOID("2", dest), Value: 3},
		{OID: legacyRouteOID("3", dest), Value: 20},
		{OID: legacyRouteOID("7", dest), Value: nextHop},
		{OID: legacyRouteOID("8", dest), Value: routeType},
		{OID: legacyRouteOID("9", dest), Value: routing.ProtoOSPF},
		{OID: legacyRouteOID("10", dest), Value: 600},
		{OID: legacyRouteOID("11", dest), Value: mask},
	}
}

func collectRoutes(t *testing.T, fc *fakeClient) []routing.Route {
	t.Helper()
	pub := &fakePublisher{}
	if err := routing.New(factoryFor(fc), pub, at).
		Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("published %d observations, want 1", len(pub.got))
	}
	return pub.got[0].Routes
}

func TestCollect_ReadsIPRouteTableWhenTheCIDRTableIsEmpty(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: slices.Concat(
		legacyRow("10.20.0.0", "255.255.0.0", "10.0.0.1", routing.TypeRemote),
		legacyRow("10.0.0.0", "255.255.255.0", "0.0.0.0", routing.TypeLocal),
	)}
	got := collectRoutes(t, fc)

	want := []routing.Route{
		{
			Destination: "10.0.0.0", Mask: "255.255.255.0", NextHop: "0.0.0.0",
			IfIndex: 3, Type: routing.TypeLocal, Proto: routing.ProtoOSPF, AgeSeconds: 600, Metric1: 20,
		},
		{
			Destination: "10.20.0.0", Mask: "255.255.0.0", NextHop: "10.0.0.1",
			IfIndex: 3, Type: routing.TypeRemote, Proto: routing.ProtoOSPF, AgeSeconds: 600, Metric1: 20,
		},
	}
	if !slices.Equal(got, want) {
		t.Errorf("routes = %+v\nwant %+v", got, want)
	}
	if !slices.Equal(fc.walked, []string{tablePrefix, legacyTablePrefix}) {
		t.Errorf("walked %v, want the CIDR table then ipRouteTable", fc.walked)
	}
}

func TestCollect_IPRouteTableRowsWithoutAUsableRoute(t *testing.T) {
	t.Parallel()
	noMask := legacyRow("10.3.0.0", "255.255.0.0", "10.0.0.1", routing.TypeRemote)
	noMask = slices.DeleteFunc(noMask, func(vb snmp.Varbind) bool {
		return vb.OID == legacyRouteOID("11", "10.3.0.0")
	})
	fc := &fakeClient{vbs: slices.Concat(
		legacyRow("10.1.0.0", "255.255.0.0", "10.0.0.1", 2), // invalid(2): deleted by the agent
		legacyRow("10.2.0.0", "255.0.255.0", "10.0.0.1", routing.TypeRemote),
		noMask,
		legacyRow("10.4.0.0", "255.255.0.0", "", routing.TypeRemote),
		legacyRow("10.5.0.0", "255.255.0.0", "10.0.0.1", routing.TypeRemote),
	)}
	got := collectRoutes(t, fc)

	if len(got) != 1 || got[0].Destination != "10.5.0.0" {
		t.Errorf("routes = %+v, want only 10.5.0.0 (invalid, non-contiguous mask, "+
			"missing mask and missing next hop dropped)", got)
	}
}

func TestCollect_IPRouteTableNotWalkedWhenTheCIDRTableHoldsRoutes(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{vbs: slices.Concat(
		[]snmp.Varbind{{OID: routeOID("5", "10.0.0.0", "10.0.0.254"), Value: uint32(1)}},
		legacyRow("10.20.0.0", "255.255.0.0", "10.0.0.1", routing.TypeRemote),
	)}
	got := collectRoutes(t, fc)

	if len(got) != 1 || got[0].Destination != "10.0.0.0" {
		t.Errorf("routes = %+v, want the CIDR table's row only", got)
	}
	if !slices.Equal(fc.walked, []string{tablePrefix}) {
		t.Errorf("walked %v, want the CIDR table only", fc.walked)
	}
}

func TestCollect_IPRouteTableWalkErrorPropagates(t *testing.T) {
	t.Parallel()
	c := routing.New(
		factoryFor(&fakeClient{legacyErr: errors.New("timeout")}),
		&fakePublisher{},
		at,
	)
	err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{})
	if err == nil || !strings.Contains(err.Error(), "ipRouteTable") {
		t.Errorf("err = %v, want the ipRouteTable walk failure", err)
	}
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
