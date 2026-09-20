package api

import (
	"context"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	snmppoller "github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

// newPromotionTestServer returns a Server with the pieces promotion touches: a
// real database, so the polling-target rows it writes are the rows the poller
// and the topology read.
func newPromotionTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	s := &Server{}
	s.dbConn = db
	s.pollingTargets = app.NewPollingTargets(s.db)
	return s
}

// seedCredential stores a credential promotion can reference. polling_targets
// references device_credentials(client_id, id), so a promoted target that
// names a credential no row backs is refused by the database.
func seedCredential(t *testing.T, db *database.DB, id, kind string) {
	t.Helper()
	cred := &polling.Credentials{
		ID: id, ClientID: database.DefaultClientID, Name: id, Kind: kind,
	}
	switch kind {
	case polling.CredentialKindV3:
		cred.SNMPv3User = "operator"
		cred.SecurityLevel = polling.SecurityLevelAuthPriv
		cred.SNMPv3AuthCT = "enc:v1:auth"
		cred.SNMPv3PrivCT = "enc:v1:priv"
		cred.SNMPv3AuthProto = "SHA256"
		cred.SNMPv3PrivProto = "AES"
	default:
		cred.SNMPCommunityCT = "enc:v1:community"
	}
	if err := db.DeviceCredentials().Upsert(context.Background(), cred); err != nil {
		t.Fatalf("seed credential %s: %v", id, err)
	}
}

func answered(ip, sysName, credID, version string) *discovery.DiscoveredDevice {
	return &discovery.DiscoveredDevice{IP: ip, SNMPData: &discovery.SNMPFullData{
		System:     &snmp.SystemInfo{SysName: sysName},
		Credential: snmp.CredentialRef{ID: credID, Version: version},
	}}
}

// The row's headline: a sweep that found SNMP-answering devices leaves polling
// targets behind, each naming the credential that device actually answered, so
// the topology fills without a hand-built target list (seed#2692).
func TestPromotionWritesATargetPerAnsweringDevice(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	seedCredential(t, s.db(), "cred-v3", polling.CredentialKindV3)

	s.promoteDiscoveredDevices(context.Background(), []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
		answered("10.44.40.2", "core-sw", "cred-v3", snmp.VersionV3),
		{IP: "10.44.40.3"}, // answered nothing
	})

	targets := listTargets(t, s)
	if len(targets) != 2 {
		t.Fatalf("promoted %d targets, want 2: %+v", len(targets), targets)
	}
	byIP := map[string]*polling.Target{}
	for _, target := range targets {
		byIP[target.IPAddress] = target
	}
	first, ok := byIP["10.44.40.1"]
	if !ok {
		t.Fatalf("10.44.40.1 was not promoted: %+v", targets)
	}
	if first.Name != "edge-rtr" || first.CredentialsID != "cred-v2c" ||
		first.SNMPVersion != snmp.VersionV2c || !first.Enabled {
		t.Errorf("promoted target = %+v", first)
	}
	second := byIP["10.44.40.2"]
	if second == nil || second.SNMPVersion != snmp.VersionV3 ||
		second.CredentialsID != "cred-v3" {
		t.Errorf("v3 device promoted as %+v; polling it as v2c fails silently", second)
	}
}

// Promotion only ever adds. A target the operator disabled must survive every
// later sweep exactly as it is: disabling is how an operator says "do not poll
// this", and re-creating or re-enabling it would overrule them once a minute.
func TestPromotionLeavesAnExistingTargetAlone(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	existing := &polling.Target{
		ClientID: database.DefaultClientID, Name: "operator's own",
		IPAddress: "10.44.40.1", SNMPVersion: snmp.VersionV2c, Enabled: false,
	}
	if err := s.db().PollingTargets().Create(context.Background(), existing); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	s.promoteDiscoveredDevices(context.Background(), []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
	})

	targets := listTargets(t, s)
	if len(targets) != 1 {
		t.Fatalf("target count %d, want the one that was already there: %+v",
			len(targets), targets)
	}
	if targets[0].Name != "operator's own" || targets[0].Enabled {
		t.Errorf("existing target was rewritten: %+v", targets[0])
	}
}

// A second sweep of an unchanged network writes nothing: every device it found
// already has the target the first sweep created.
func TestPromotionIsIdempotentAcrossSweeps(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	sweep := []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
		answered("10.44.40.2", "core-sw", "cred-v2c", snmp.VersionV2c),
	}

	s.promoteDiscoveredDevices(context.Background(), sweep)
	first := listTargets(t, s)
	s.promoteDiscoveredDevices(context.Background(), sweep)
	second := listTargets(t, s)

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("target counts %d then %d, want 2 then 2", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID || !first[i].UpdatedAt.Equal(second[i].UpdatedAt) {
			t.Errorf("target %s was rewritten by the second sweep", first[i].IPAddress)
		}
	}
}

func listTargets(t *testing.T, s *Server) []*polling.Target {
	t.Helper()
	targets, err := s.pollingTargets.ListAll(context.Background(), database.DefaultClientID)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	return targets
}

// fakeDiscoveryStore is the settings store the learner persists through.
type fakeDiscoveryStore struct{ cfg config.NetworkDiscoveryConfig }

func (f *fakeDiscoveryStore) Discovery() config.NetworkDiscoveryConfig { return f.cfg }

func (f *fakeDiscoveryStore) SaveDiscovery(cfg config.NetworkDiscoveryConfig) error {
	f.cfg = cfg
	return nil
}

type noopSink struct{}

func (noopSink) SetTargetNetworks([]string) error { return nil }

type noopApplier struct{}

func (noopApplier) ReloadOptions() error { return nil }

// One sweep, two consumers. The discovery service holds a single observer, so
// the composition is where a consumer gets lost: a sweep that learned its
// target networks but promoted nothing (or the reverse) is exactly the defect
// each of seed#2695 and seed#2692 was filed for.
func TestAfterSweepBothLearnsAndPromotes(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	store := &fakeDiscoveryStore{}
	s.discoverySettings = discoverysettings.NewService(store, noopSink{}, noopApplier{})
	s.deviceDisc = enumerate.NewDeviceDiscovery("")

	router := answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c)
	router.SNMPData.Routing = []discovery.SNMPRoute{
		{Destination: "10.44.10.0", Prefix: 24, Type: "remote", Protocol: "ospf"},
	}

	s.afterSweep(context.Background(), []*discovery.DiscoveredDevice{router})

	if got := listTargets(t, s); len(got) != 1 || got[0].IPAddress != "10.44.40.1" {
		t.Errorf("promotion did not run from the sweep observer: %+v", got)
	}
	learned := store.cfg.TargetNetworks
	if len(learned) != 1 || learned[0].CIDR != "10.44.10.0/24" {
		t.Errorf("learning did not run from the sweep observer: %+v", learned)
	}
}

// The rescan ticker and the manual scan button both reach promotion, and
// polling_targets has no unique index on the address: two overlapping runs
// that each read the target list before either wrote would each create a row
// for the same device, and the poller would then poll it twice.
func TestConcurrentPromotionsCreateOneTargetPerDevice(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	sweep := []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
		answered("10.44.40.2", "core-sw", "cred-v2c", snmp.VersionV2c),
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			s.promoteDiscoveredDevices(context.Background(), sweep)
		})
	}
	wg.Wait()

	if got := listTargets(t, s); len(got) != 2 {
		t.Fatalf("promoted %d targets, want 2: %+v", len(got), got)
	}
}

// Promotion acts on the operator's behalf with no request to scope it, so it
// answers "whose deployment is this" exactly the way the credential resolver
// does: one client, or nothing happens. Guessing would put one tenant's
// credential on another tenant's device — the isolation failure the vault
// exists to prevent.
func TestPromotionRefusesWhenTheTenantIsAmbiguous(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	second := &database.Client{Name: "second tenant", Slug: "second"}
	if err := s.db().Clients().Create(context.Background(), second); err != nil {
		t.Fatalf("seed client: %v", err)
	}

	s.promoteDiscoveredDevices(context.Background(), []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
	})

	if got := listTargets(t, s); len(got) != 0 {
		t.Fatalf("promoted %+v; want nothing while the tenant is ambiguous", got)
	}
}

// countingPollerStorage counts the poller's target reads, which is how a
// reload shows up from outside the poller.
type countingPollerStorage struct {
	mu     sync.Mutex
	reads  int
	shared *Server
}

func (c *countingPollerStorage) ListEnabled(ctx context.Context) ([]*polling.Target, error) {
	c.mu.Lock()
	c.reads++
	c.mu.Unlock()
	return c.shared.db().PollingTargets().ListEnabled(ctx)
}

func (c *countingPollerStorage) UpdateLastPoll(context.Context, string, string, string) error {
	return nil
}

func (c *countingPollerStorage) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

// noopScheduler satisfies the poller's scheduler seam without running a job.
type noopScheduler struct{}

func (noopScheduler) Register(scheduler.Job) {}
func (noopScheduler) Unregister(string) bool { return true }
func (noopScheduler) Start(context.Context)  {}
func (noopScheduler) Stop()                  {}

// A promoted target nothing polls is a row in a table. The live poller reads
// its target list once and reloads when the list changes — the create handler
// says so, and promotion writes the same rows by another door.
func TestPromotionReloadsTheLivePoller(t *testing.T) {
	s := newPromotionTestServer(t)
	seedCredential(t, s.db(), "cred-v2c", polling.CredentialKindV2c)
	storage := &countingPollerStorage{shared: s}
	poller := snmppoller.NewPoller(storage, noopScheduler{}, nil)
	if err := poller.Start(context.Background()); err != nil {
		t.Fatalf("start poller: %v", err)
	}
	t.Cleanup(func() { _ = poller.Stop(context.Background()) })
	s.snmpPoller = poller
	before := storage.count()

	s.promoteDiscoveredDevices(context.Background(), []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
	})

	if storage.count() <= before {
		t.Fatal("the poller was not reloaded: promoted targets are not polled until a restart")
	}

	// A sweep that promoted nothing must not churn the live poller.
	steady := storage.count()
	s.promoteDiscoveredDevices(context.Background(), []*discovery.DiscoveredDevice{
		answered("10.44.40.1", "edge-rtr", "cred-v2c", snmp.VersionV2c),
	})
	if storage.count() != steady {
		t.Errorf("poller reloaded although nothing was promoted")
	}
}
