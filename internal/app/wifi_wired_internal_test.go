package app

import (
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
)

// TestWiFiWiredSourceReadsDiscoveryAndPolledTables drives the wired set through
// what production writes: the discovery registry and the SNMP sink's ARP and
// FDB observations. Only each target's newest observation counts, and invalid
// FDB rows and MAC-less devices are skipped.
func TestWiFiWiredSourceReadsDiscoveryAndPolledTables(t *testing.T) {
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := t.Context()
	pub := sink.New(db.SNMPObservations(), nil, nil)
	old := time.Now().Add(-time.Hour)
	now := time.Now()
	for _, obs := range []fdb.Observation{
		{ClientID: "default", TargetID: "sw1", ObservedAt: old, Entries: []fdb.Entry{
			{MACAddress: "00:00:5e:00:53:a0", Status: fdb.StatusLearned},
		}},
		{ClientID: "default", TargetID: "sw1", ObservedAt: now, Entries: []fdb.Entry{
			{MACAddress: "00:00:5e:00:53:a1", Status: fdb.StatusLearned},
			{MACAddress: "00:00:5e:00:53:a2", Status: fdb.StatusInvalid},
		}},
		{ClientID: "default", TargetID: "sw2", ObservedAt: old, Entries: []fdb.Entry{
			{MACAddress: "00:00:5e:00:53:b1", Status: fdb.StatusLearned},
		}},
	} {
		if pubErr := pub.PublishFDB(ctx, obs); pubErr != nil {
			t.Fatalf("publish fdb: %v", pubErr)
		}
	}
	if pubErr := pub.PublishARP(ctx, arp.Observation{
		ClientID: "default", TargetID: "rtr1", ObservedAt: now,
		Entries: []arp.Entry{{IPAddress: "192.0.2.9", MACAddress: "00:00:5e:00:53:c1"}},
	}); pubErr != nil {
		t.Fatalf("publish arp: %v", pubErr)
	}

	devices := []*discovery.DiscoveredDevice{{MAC: "00:00:5e:00:53:d1"}, {IP: "192.0.2.10"}}
	src := WiFiWiredSource(
		func() []*discovery.DiscoveredDevice { return devices },
		func() *Persistence { return NewPersistence(db) },
	)
	got, err := src(ctx)
	if err != nil {
		t.Fatalf("wired source: %v", err)
	}
	slices.Sort(got)
	want := []string{"00:00:5e:00:53:a1", "00:00:5e:00:53:b1", "00:00:5e:00:53:c1", "00:00:5e:00:53:d1"}
	if !slices.Equal(got, want) {
		t.Errorf("wired MACs = %v, want %v", got, want)
	}
}

func TestWiFiWiredSourceWithoutDatabaseReadsDiscoveryOnly(t *testing.T) {
	src := WiFiWiredSource(
		func() []*discovery.DiscoveredDevice { return []*discovery.DiscoveredDevice{{MAC: "00:00:5e:00:53:d1"}} },
		func() *Persistence { return nil },
	)
	got, err := src(t.Context())
	if err != nil {
		t.Fatalf("wired source: %v", err)
	}
	if !slices.Equal(got, []string{"00:00:5e:00:53:d1"}) {
		t.Errorf("wired MACs = %v, want only the discovered device", got)
	}
}
