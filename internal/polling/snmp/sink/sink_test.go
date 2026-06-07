package sink_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/bgp4"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/cdp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/lldp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/routing"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/sysinfo"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
func at() time.Time              { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

type fakeStore struct {
	mu     sync.Mutex
	rows   []*database.SNMPObservation
	insErr error
}

func (f *fakeStore) Insert(_ context.Context, obs *database.SNMPObservation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.insErr != nil {
		return f.insErr
	}
	f.rows = append(f.rows, obs)
	return nil
}

func newSink() (*sink.Sink, *fakeStore) {
	store := &fakeStore{}
	return sink.New(store, silentLogger(), at), store
}

func TestPublishSysInfo_WritesObservationRow(t *testing.T) {
	t.Parallel()
	s, store := newSink()
	obs := sysinfo.Observation{
		ClientID:   "client-a",
		TargetID:   "t-1",
		ObservedAt: time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		SysDescr:   "Cisco IOS XE",
		SysName:    "router-1",
	}
	if err := s.PublishSysInfo(context.Background(), obs); err != nil {
		t.Fatalf("PublishSysInfo: %v", err)
	}
	if len(store.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(store.rows))
	}
	row := store.rows[0]
	if row.Kind != sink.KindSysInfo {
		t.Errorf("kind = %q, want %q", row.Kind, sink.KindSysInfo)
	}
	if row.ClientID != "client-a" || row.TargetID != "t-1" {
		t.Errorf("client/target = %s/%s", row.ClientID, row.TargetID)
	}

	var decoded sysinfo.Observation
	if err := json.Unmarshal([]byte(row.PayloadJSON), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.SysDescr != "Cisco IOS XE" || decoded.SysName != "router-1" {
		t.Errorf("decoded payload = %+v", decoded)
	}
}

func TestPublishCDP_DefaultPrefixWritesCDPKind(t *testing.T) {
	t.Parallel()
	s, store := newSink()
	obs := cdp.Observation{
		ClientID:    "c-a",
		TargetID:    "t-1",
		TablePrefix: cdp.DefaultTablePrefix,
	}
	_ = s.PublishCDP(context.Background(), obs)
	if store.rows[0].Kind != sink.KindCDP {
		t.Errorf("kind = %q, want %q", store.rows[0].Kind, sink.KindCDP)
	}
}

func TestPublishCDP_FoundryPrefixWritesFDPKind(t *testing.T) {
	t.Parallel()
	s, store := newSink()
	obs := cdp.Observation{
		ClientID:    "c-a",
		TargetID:    "t-1",
		TablePrefix: "1.3.6.1.4.1.1991.1.1.3.2.2.1",
	}
	_ = s.PublishCDP(context.Background(), obs)
	if store.rows[0].Kind != sink.KindFDP {
		t.Errorf("kind = %q, want %q (FDP)", store.rows[0].Kind, sink.KindFDP)
	}
}

func TestAllPublishers_WriteCorrectKinds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	target := "t-1"

	type check struct {
		name string
		kind string
		do   func(*sink.Sink) error
	}
	checks := []check{
		{
			"iftable",
			sink.KindIfTable,
			func(s *sink.Sink) error {
				return s.PublishIfTable(ctx, iftable.Observation{TargetID: target})
			},
		},
		{
			"lldp",
			sink.KindLLDP,
			func(s *sink.Sink) error {
				return s.PublishLLDP(ctx, lldp.Observation{TargetID: target})
			},
		},
		{
			"arp",
			sink.KindARP,
			func(s *sink.Sink) error {
				return s.PublishARP(ctx, arp.Observation{TargetID: target})
			},
		},
		{
			"fdb",
			sink.KindFDB,
			func(s *sink.Sink) error {
				return s.PublishFDB(ctx, fdb.Observation{TargetID: target})
			},
		},
		{
			"routing",
			sink.KindRouting,
			func(s *sink.Sink) error {
				return s.PublishRouting(ctx, routing.Observation{TargetID: target})
			},
		},
		{
			"hostresources",
			sink.KindHostResources,
			func(s *sink.Sink) error {
				return s.PublishHostResources(ctx, hostresources.Observation{TargetID: target})
			},
		},
		{
			"bgp4",
			sink.KindBGP4,
			func(s *sink.Sink) error {
				return s.PublishBGP4(ctx, bgp4.Observation{TargetID: target})
			},
		},
	}
	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, store := newSink()
			if err := tt.do(s); err != nil {
				t.Fatalf("Publish: %v", err)
			}
			if len(store.rows) != 1 {
				t.Fatalf("rows = %d, want 1", len(store.rows))
			}
			if store.rows[0].Kind != tt.kind {
				t.Errorf("kind = %q, want %q", store.rows[0].Kind, tt.kind)
			}
		})
	}
}

func TestPersist_PreservesObservedAt(t *testing.T) {
	t.Parallel()
	s, store := newSink()
	expected := time.Date(2026, 1, 15, 8, 30, 45, 0, time.UTC)
	_ = s.PublishSysInfo(context.Background(), sysinfo.Observation{
		TargetID:   "t-1",
		ObservedAt: expected,
	})
	if !store.rows[0].ObservedAt.Equal(expected) {
		t.Errorf("ObservedAt = %v, want %v", store.rows[0].ObservedAt, expected)
	}
}

func TestPersist_StampsIngestedAtFromNowFunc(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := sink.New(store, silentLogger(), func() time.Time { return clock })

	_ = s.PublishSysInfo(context.Background(), sysinfo.Observation{TargetID: "t-1"})
	if !store.rows[0].IngestedAt.Equal(clock) {
		t.Errorf("IngestedAt = %v, want %v (injected clock)",
			store.rows[0].IngestedAt, clock)
	}
}

func TestPersist_StoreErrorPropagates(t *testing.T) {
	t.Parallel()
	store := &fakeStore{insErr: errors.New("disk full")}
	s := sink.New(store, silentLogger(), at)
	if err := s.PublishSysInfo(context.Background(), sysinfo.Observation{TargetID: "t-1"}); err == nil {
		t.Error("expected insert error to propagate")
	}
}

func TestNew_NilLoggerUsesDefault(t *testing.T) {
	t.Parallel()
	s := sink.New(&fakeStore{}, nil, at)
	// Should not panic when used.
	if err := s.PublishSysInfo(context.Background(), sysinfo.Observation{TargetID: "t-1"}); err != nil {
		t.Errorf("Publish with nil logger: %v", err)
	}
}

func TestNew_NilNowUsesTimeNow(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	s := sink.New(store, silentLogger(), nil)
	before := time.Now().UTC().Add(-time.Second)
	if err := s.PublishSysInfo(context.Background(), sysinfo.Observation{TargetID: "t-1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)
	if store.rows[0].IngestedAt.Before(before) || store.rows[0].IngestedAt.After(after) {
		t.Errorf("IngestedAt = %v, want between %v and %v",
			store.rows[0].IngestedAt, before, after)
	}
}
