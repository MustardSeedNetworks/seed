package sysinfo_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
	"github.com/krisarmstrong/seed/internal/polling/snmp/collectors/sysinfo"
)

// fakeClient records the OIDs requested and replays a fixed varbind
// set. Wrong-OID tests can override the response per OID.
type fakeClient struct {
	requestedOIDs []string
	getResponse   []snmp.Varbind
	getErr        error
}

func (f *fakeClient) Get(_ context.Context, oids []string) ([]snmp.Varbind, error) {
	f.requestedOIDs = append([]string{}, oids...)
	if f.getErr != nil {
		return nil, f.getErr
	}
	// echo back only the responses we have, indexed by OID match
	out := make([]snmp.Varbind, 0, len(f.getResponse))
	for _, want := range oids {
		for _, vb := range f.getResponse {
			if vb.OID == want {
				out = append(out, vb)
				break
			}
		}
	}
	return out, nil
}

func (f *fakeClient) Walk(_ context.Context, _ string) ([]snmp.Varbind, error) {
	return nil, errors.New("walk not used by sysinfo")
}

// fakePublisher records every PublishSysInfo invocation.
type fakePublisher struct {
	mu   sync.Mutex
	got  []sysinfo.Observation
	err  error
	hits int
}

func (p *fakePublisher) PublishSysInfo(_ context.Context, obs sysinfo.Observation) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hits++
	if p.err != nil {
		return p.err
	}
	p.got = append(p.got, obs)
	return nil
}

func staticFactory(c *fakeClient) snmp.ClientFactory {
	return func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
		return c, nil
	}
}

func fixedClock() time.Time {
	return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
}

func TestCollector_Name(t *testing.T) {
	t.Parallel()
	c := sysinfo.New(nil, nil, fixedClock)
	if got := c.Name(); got != sysinfo.Name {
		t.Errorf("Name() = %q, want %q", got, sysinfo.Name)
	}
}

func TestCollect_FetchesAllSixSystemOIDs(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		getResponse: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.1.1.0", Value: "Cisco IOS XE"},
			{OID: "1.3.6.1.2.1.1.5.0", Value: "router-1"},
		},
	}
	pub := &fakePublisher{}
	c := sysinfo.New(staticFactory(fc), pub, fixedClock)

	target := snmp.Target{ID: "t-1", ClientID: "client-a"}
	if err := c.Collect(context.Background(), target, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	wantOIDs := []string{
		"1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.2.0", "1.3.6.1.2.1.1.3.0",
		"1.3.6.1.2.1.1.4.0", "1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.6.0",
	}
	if len(fc.requestedOIDs) != len(wantOIDs) {
		t.Fatalf("requested %d OIDs, want %d", len(fc.requestedOIDs), len(wantOIDs))
	}
	for i, oid := range wantOIDs {
		if fc.requestedOIDs[i] != oid {
			t.Errorf("OID[%d] = %q, want %q", i, fc.requestedOIDs[i], oid)
		}
	}
}

func TestCollect_PublishesObservationWithTargetMetadata(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		getResponse: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.1.1.0", Value: []byte("Linux 6.5")},
			{OID: "1.3.6.1.2.1.1.2.0", Value: "1.3.6.1.4.1.8072.3.2.10"},
			{OID: "1.3.6.1.2.1.1.3.0", Value: uint32(123456)},
			{OID: "1.3.6.1.2.1.1.4.0", Value: "noc@example.com"},
			{OID: "1.3.6.1.2.1.1.5.0", Value: "switch-1"},
			{OID: "1.3.6.1.2.1.1.6.0", Value: "MDF-Rack-3"},
		},
	}
	pub := &fakePublisher{}
	c := sysinfo.New(staticFactory(fc), pub, fixedClock)

	target := snmp.Target{ID: "t-99", ClientID: "client-x"}
	if err := c.Collect(context.Background(), target, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if len(pub.got) != 1 {
		t.Fatalf("publisher hits = %d, want 1", len(pub.got))
	}
	obs := pub.got[0]
	if obs.ClientID != "client-x" {
		t.Errorf("ClientID = %q, want client-x", obs.ClientID)
	}
	if obs.TargetID != "t-99" {
		t.Errorf("TargetID = %q, want t-99", obs.TargetID)
	}
	if !obs.ObservedAt.Equal(fixedClock()) {
		t.Errorf("ObservedAt = %v, want %v", obs.ObservedAt, fixedClock())
	}
	if obs.SysDescr != "Linux 6.5" {
		t.Errorf("SysDescr = %q, want Linux 6.5", obs.SysDescr)
	}
	if obs.SysObjectID != "1.3.6.1.4.1.8072.3.2.10" {
		t.Errorf("SysObjectID = %q", obs.SysObjectID)
	}
	if obs.SysUpTimeTicks != 123456 {
		t.Errorf("SysUpTimeTicks = %d, want 123456", obs.SysUpTimeTicks)
	}
	if obs.SysName != "switch-1" {
		t.Errorf("SysName = %q", obs.SysName)
	}
	if obs.SysLocation != "MDF-Rack-3" {
		t.Errorf("SysLocation = %q", obs.SysLocation)
	}
	if obs.SysContact != "noc@example.com" {
		t.Errorf("SysContact = %q", obs.SysContact)
	}
}

func TestCollect_PartialResponseStillPublishes(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		// Only 2 of 6 OIDs return a value (agent doesn't expose
		// sysContact/sysLocation, etc.).
		getResponse: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.1.1.0", Value: "Junos"},
			{OID: "1.3.6.1.2.1.1.5.0", Value: "edge-1"},
		},
	}
	pub := &fakePublisher{}
	c := sysinfo.New(staticFactory(fc), pub, fixedClock)

	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("expected exactly one publish, got %d", len(pub.got))
	}
	obs := pub.got[0]
	if obs.SysDescr != "Junos" {
		t.Errorf("SysDescr = %q, want Junos", obs.SysDescr)
	}
	if obs.SysName != "edge-1" {
		t.Errorf("SysName = %q, want edge-1", obs.SysName)
	}
	if obs.SysObjectID != "" || obs.SysContact != "" || obs.SysLocation != "" {
		t.Errorf("missing OIDs should be empty strings, got %+v", obs)
	}
	if obs.SysUpTimeTicks != 0 {
		t.Errorf("SysUpTimeTicks = %d, want 0 for missing", obs.SysUpTimeTicks)
	}
}

func TestCollect_GetErrorPropagates(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{getErr: errors.New("snmp v2c timeout")}
	pub := &fakePublisher{}
	c := sysinfo.New(staticFactory(fc), pub, fixedClock)

	err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{})
	if err == nil {
		t.Fatal("expected Collect to surface Get error")
	}
	if pub.hits != 0 {
		t.Errorf("publisher called %d times, want 0 on Get error", pub.hits)
	}
}

func TestCollect_PublishErrorPropagates(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		getResponse: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.1.5.0", Value: "host"},
		},
	}
	pub := &fakePublisher{err: errors.New("event log full")}
	c := sysinfo.New(staticFactory(fc), pub, fixedClock)

	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected Collect to surface publisher error")
	}
}

func TestCollect_FactoryNilReturnsError(t *testing.T) {
	t.Parallel()
	c := sysinfo.New(nil, &fakePublisher{}, fixedClock)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected error when client factory is nil")
	}
}

func TestCollect_PublisherNilReturnsError(t *testing.T) {
	t.Parallel()
	c := sysinfo.New(staticFactory(&fakeClient{}), nil, fixedClock)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected error when publisher is nil")
	}
}

func TestCollect_FactoryErrorPropagates(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	c := sysinfo.New(
		func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
			return nil, errors.New("no creds resolved")
		},
		pub,
		fixedClock,
	)
	if err := c.Collect(context.Background(), snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected Collect to surface factory error")
	}
	if pub.hits != 0 {
		t.Errorf("publisher called %d times on factory error, want 0", pub.hits)
	}
}

func TestCollect_UnknownValueTypeClampsToZero(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		getResponse: []snmp.Varbind{
			{OID: "1.3.6.1.2.1.1.3.0", Value: -42}, // negative int → 0
		},
	}
	pub := &fakePublisher{}
	c := sysinfo.New(staticFactory(fc), pub, fixedClock)

	if err := c.Collect(context.Background(), snmp.Target{ID: "t-1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if pub.got[0].SysUpTimeTicks != 0 {
		t.Errorf("negative SysUpTime should clamp to 0, got %d", pub.got[0].SysUpTimeTicks)
	}
}
