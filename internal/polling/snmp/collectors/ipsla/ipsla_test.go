package ipsla_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
)

const (
	admin  = "1.3.6.1.4.1.9.9.42.1.2.1.1."
	echo   = "1.3.6.1.4.1.9.9.42.1.2.2.1."
	latest = "1.3.6.1.4.1.9.9.42.1.2.10.1."
	jitter = "1.3.6.1.4.1.9.9.42.1.5.2.1."
)

// fakeClient serves each walked column from one flat varbind list, the way
// an agent would.
type fakeClient struct {
	vbs     []snmp.Varbind
	walkErr error
	walked  []string
}

func (*fakeClient) Get(context.Context, []string) ([]snmp.Varbind, error) {
	return nil, errors.New("Get not used by ip_sla")
}

func (f *fakeClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	f.walked = append(f.walked, prefix)
	if f.walkErr != nil {
		return nil, f.walkErr
	}
	var out []snmp.Varbind
	for _, vb := range f.vbs {
		if strings.HasPrefix(strings.TrimPrefix(vb.OID, "."), prefix+".") {
			out = append(out, vb)
		}
	}
	return out, nil
}

func (*fakeClient) WalkLimit(context.Context, string, int) ([]snmp.Varbind, error) {
	return nil, errors.New("WalkLimit not used by ip_sla")
}

type fakePublisher struct {
	got []ipsla.Observation
	err error
}

func (p *fakePublisher) PublishIPSLA(_ context.Context, obs ipsla.Observation) error {
	if p.err != nil {
		return p.err
	}
	p.got = append(p.got, obs)
	return nil
}

func factoryFor(c *fakeClient) snmp.ClientFactory {
	return func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) { return c, nil }
}

func at() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }

func target() snmp.Target { return snmp.Target{ID: "rtr-1", ClientID: "default"} }

func collect(t *testing.T, fc *fakeClient) ipsla.Observation {
	t.Helper()
	pub := &fakePublisher{}
	err := ipsla.New(factoryFor(fc), pub, at).Collect(context.Background(), target(), snmp.ResolvedCredentials{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("published %d observations, want 1", len(pub.got))
	}
	return pub.got[0]
}

// Operation 10 is an ICMP echo to 192.0.2.1 that completed in 12 ms;
// operation 20 a udp-jitter to 2001:db8::1 over its 300 ms threshold, with
// 95 of 100 packets back and a G.711 MOS of 4.34; operation 30 timed out;
// operation 40 is configured but has never run; and index 50 has a result
// but no operation.
func TestCollect_Operations(t *testing.T) {
	t.Parallel()
	v6 := []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: "." + admin + "3.10", Value: []byte("wan-gw")},
		{OID: "." + admin + "4.10", Value: 1},
		{OID: "." + admin + "5.10", Value: 100},
		{OID: "." + admin + "6.10", Value: 60},
		{OID: "." + admin + "9.10", Value: ipsla.RowStatusActive},
		{OID: "." + admin + "3.20", Value: []byte("voice-dc")},
		{OID: "." + admin + "4.20", Value: 9},
		{OID: "." + admin + "5.20", Value: 300},
		{OID: "." + admin + "6.20", Value: 30},
		{OID: "." + admin + "9.20", Value: ipsla.RowStatusActive},
		{OID: "." + admin + "4.30", Value: 1},
		{OID: "." + admin + "9.30", Value: ipsla.RowStatusActive},
		{OID: "." + admin + "4.40", Value: 6},
		{OID: "." + admin + "9.40", Value: 2},
		{OID: "." + echo + "2.10", Value: []byte{192, 0, 2, 1}},
		{OID: "." + echo + "2.20", Value: v6},
		{OID: "." + echo + "2.40", Value: []byte("db.example.net")},
		{OID: "." + latest + "1.10", Value: uint(12)},
		{OID: "." + latest + "2.10", Value: ipsla.SenseOK},
		{OID: "." + latest + "5.10", Value: uint32(360_000)},
		{OID: "." + latest + "1.20", Value: uint(310)},
		{OID: "." + latest + "2.20", Value: ipsla.SenseOverThreshold},
		{OID: "." + latest + "5.20", Value: uint32(361_000)},
		{OID: "." + latest + "1.30", Value: uint(0)},
		{OID: "." + latest + "2.30", Value: ipsla.SenseTimeout},
		{OID: "." + latest + "5.30", Value: uint32(362_000)},
		{OID: "." + latest + "1.40", Value: uint(0)},
		{OID: "." + latest + "2.40", Value: ipsla.SenseOther},
		{OID: "." + latest + "5.40", Value: uint32(0)},
		{OID: "." + latest + "2.50", Value: ipsla.SenseOK},
		{OID: "." + jitter + "1.20", Value: uint(95)},
		{OID: "." + jitter + "26.20", Value: uint(2)},
		{OID: "." + jitter + "27.20", Value: uint(1)},
		{OID: "." + jitter + "29.20", Value: uint(2)},
		{OID: "." + jitter + "42.20", Value: uint(434)},
		{OID: "." + jitter + "46.20", Value: uint(7)},
	}}

	got := collect(t, fc)
	if got.TargetID != "rtr-1" || got.ClientID != "default" || !got.ObservedAt.Equal(at()) {
		t.Errorf("identity = %q/%q/%v", got.TargetID, got.ClientID, got.ObservedAt)
	}
	want := []ipsla.Operation{
		{
			Index: 10, Tag: "wan-gw", RttType: 1, Target: "192.0.2.1",
			ThresholdMs: 100, FrequencyS: 60, Status: ipsla.RowStatusActive,
			Latest: &ipsla.Result{Sense: ipsla.SenseOK, RTTMs: 12, RunUptime: 360_000},
		},
		{
			Index: 20, Tag: "voice-dc", RttType: 9, Target: "2001:db8::1",
			ThresholdMs: 300, FrequencyS: 30, Status: ipsla.RowStatusActive,
			Latest: &ipsla.Result{
				Sense: ipsla.SenseOverThreshold, RTTMs: 310, RunUptime: 361_000,
				Jitter: &ipsla.Jitter{AvgJitterMs: 7, Received: 95, Lost: 5, MOS: 4.34},
			},
		},
		{
			Index: 30, RttType: 1, Status: ipsla.RowStatusActive,
			Latest: &ipsla.Result{Sense: ipsla.SenseTimeout, RunUptime: 362_000},
		},
		{Index: 40, RttType: 6, Target: "db.example.net", Status: 2},
	}
	if !reflect.DeepEqual(got.Operations, want) {
		t.Errorf("Operations =\n%+v\nwant\n%+v", got.Operations, want)
	}
}

func TestResult_Completed(t *testing.T) {
	t.Parallel()
	for sense, want := range map[int]bool{
		ipsla.SenseOther:         false,
		ipsla.SenseOK:            true,
		2:                        false, // disconnected
		ipsla.SenseOverThreshold: true,
		ipsla.SenseTimeout:       false,
		16:                       false, // error
	} {
		if got := (ipsla.Result{Sense: sense}).Completed(); got != want {
			t.Errorf("sense %d: Completed() = %v, want %v", sense, got, want)
		}
	}
}

// A device without CISCO-RTTMON-MIB answers every walk empty; the poll still
// publishes, so the observation records that it has no operations.
func TestCollect_NoMIB(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{}
	got := collect(t, fc)
	if len(got.Operations) != 0 {
		t.Errorf("Operations = %+v, want none", got.Operations)
	}
	if len(fc.walked) != 15 {
		t.Errorf("walked %d columns, want 15", len(fc.walked))
	}
}

func TestCollect_Errors(t *testing.T) {
	t.Parallel()
	errBoom := errors.New("boom")
	tests := []struct {
		name    string
		factory snmp.ClientFactory
		pub     ipsla.Publisher
		want    string
	}{
		{"no factory", nil, &fakePublisher{}, "client factory not configured"},
		{"no publisher", factoryFor(&fakeClient{}), nil, "publisher not configured"},
		{
			"dial",
			func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) { return nil, errBoom },
			&fakePublisher{},
			"dial",
		},
		{"walk", factoryFor(&fakeClient{walkErr: errBoom}), &fakePublisher{}, "walk 1.3.6.1.4.1.9.9.42.1.2.1.1.3"},
		{"publish", factoryFor(&fakeClient{}), &fakePublisher{err: errBoom}, "publish"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ipsla.New(tt.factory, tt.pub, at).Collect(context.Background(), target(), snmp.ResolvedCredentials{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Collect error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Parallel()
	if got := ipsla.New(nil, nil, nil).Name(); got != "ip_sla" {
		t.Errorf("Name() = %q", got)
	}
}
