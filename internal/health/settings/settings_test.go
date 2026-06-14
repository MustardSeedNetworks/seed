package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/health/settings"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

type fakeProbeStore struct {
	endpoints config.HealthChecksConfig
	saved     *config.HealthChecksConfig
	probes    []probe.Probe
	count     int
	loadErr   error
	saveErr   error
	countErr  error
}

func (f *fakeProbeStore) Endpoints(context.Context) (config.HealthChecksConfig, error) {
	return f.endpoints, f.loadErr
}

func (f *fakeProbeStore) SaveEndpoints(_ context.Context, hc config.HealthChecksConfig) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = &hc
	return nil
}

func (f *fakeProbeStore) Count(context.Context) (int, error) {
	return f.count, f.countErr
}

func (f *fakeProbeStore) List(context.Context) ([]probe.Probe, error) {
	return f.probes, f.loadErr
}

type fakeConfigStore struct{ cfg *config.Config }

func (f *fakeConfigStore) Read(fn func(*config.Config))              { fn(f.cfg) }
func (f *fakeConfigStore) Write(fn func(*config.Config) error) error { return fn(f.cfg) }

type fakeAppliers struct {
	dnsHostname string
	dnsServers  []config.DNSServer
	speedID     string
}

func (f *fakeAppliers) ApplyDNS(hostname string, servers []config.DNSServer) {
	f.dnsHostname = hostname
	f.dnsServers = servers
}
func (f *fakeAppliers) ApplySpeedtestServer(id string) { f.speedID = id }

type fakeSeedMarker struct {
	seeded  bool
	markErr error
}

func (f *fakeSeedMarker) Seeded(context.Context) (bool, error) { return f.seeded, nil }
func (f *fakeSeedMarker) MarkSeeded(context.Context) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.seeded = true
	return nil
}

func TestGetComposesProbesAndConfig(t *testing.T) {
	probes := &fakeProbeStore{endpoints: config.HealthChecksConfig{
		PingTargets: []config.PingTarget{{Name: "g", Host: "8.8.8.8", Enabled: true}},
	}}
	cfg := &config.Config{}
	cfg.DNS.TestHostname = "example.com"
	cfg.HealthChecks.RunSpeedtest = true
	cfg.Speedtest.ServerID = "42"
	svc := settings.NewService(probes, &fakeConfigStore{cfg: cfg}, &fakeAppliers{}, &fakeSeedMarker{})

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Endpoints.PingTargets) != 1 || got.Endpoints.PingTargets[0].Host != "8.8.8.8" {
		t.Errorf("endpoints not sourced from probe store: %+v", got.Endpoints)
	}
	if got.DNSHostname != "example.com" || !got.RunSpeedtest || got.SpeedtestServerID != "42" {
		t.Errorf("config toggles not composed: %+v", got)
	}
}

func TestGetSurfacesProbeError(t *testing.T) {
	wantErr := errors.New("db down")
	svc := settings.NewService(
		&fakeProbeStore{loadErr: wantErr}, &fakeConfigStore{cfg: &config.Config{}}, &fakeAppliers{}, &fakeSeedMarker{})
	if _, err := svc.Get(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("probe load error not surfaced: %v", err)
	}
}

func TestUpdatePersistsProbesConfigAndSyncsTesters(t *testing.T) {
	probes := &fakeProbeStore{}
	cfg := &config.Config{}
	appliers := &fakeAppliers{}
	svc := settings.NewService(probes, &fakeConfigStore{cfg: cfg}, appliers, &fakeSeedMarker{})

	in := settings.Settings{
		Endpoints:         config.HealthChecksConfig{PingTargets: []config.PingTarget{{Host: "1.1.1.1"}}},
		DNSHostname:       "host.test",
		DNSServers:        []config.DNSServer{{Address: "9.9.9.9", Enabled: true}},
		RunSpeedtest:      true,
		SpeedtestServerID: "7",
	}
	if err := svc.Update(context.Background(), in); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if probes.saved == nil || len(probes.saved.PingTargets) != 1 {
		t.Errorf("endpoints not persisted to the probe store: %+v", probes.saved)
	}
	if cfg.DNS.TestHostname != "host.test" || !cfg.HealthChecks.RunSpeedtest || cfg.Speedtest.ServerID != "7" {
		t.Errorf("config not written: %+v", cfg)
	}
	if appliers.dnsHostname != "host.test" || appliers.speedID != "7" || len(appliers.dnsServers) != 1 {
		t.Errorf("testers not synced: %+v", appliers)
	}
}

// TestUpdateSkipsConfigOnProbeFailure asserts probe persistence is the gate: if it
// fails, the config is not written and the testers are not synced.
func TestUpdateSkipsConfigOnProbeFailure(t *testing.T) {
	probes := &fakeProbeStore{saveErr: errors.New("probe write failed")}
	cfg := &config.Config{}
	appliers := &fakeAppliers{}
	svc := settings.NewService(probes, &fakeConfigStore{cfg: cfg}, appliers, &fakeSeedMarker{})

	if err := svc.Update(context.Background(), settings.Settings{DNSHostname: "x"}); err == nil {
		t.Fatal("Update should return the probe save error")
	}
	if cfg.DNS.TestHostname != "" || appliers.dnsHostname != "" {
		t.Error("config/testers must not be touched when probe save fails")
	}
}

// TestSeedDefaults_FreshInstall verifies that on a genuine first run the
// factory defaults are saved and the marker is set.
func TestSeedDefaults_FreshInstall(t *testing.T) {
	probes := &fakeProbeStore{count: 0}
	marker := &fakeSeedMarker{}
	svc := settings.NewService(probes, &fakeConfigStore{cfg: &config.Config{}}, &fakeAppliers{}, marker)

	defaults := config.HealthChecksConfig{
		PingTargets: []config.PingTarget{{Name: "gw", Host: "8.8.8.8", Enabled: true}},
	}
	if err := svc.SeedDefaults(context.Background(), defaults); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	if probes.saved == nil {
		t.Error("defaults must be saved to the probe store on first run")
	}
	if !marker.seeded {
		t.Error("marker must be set after seeding")
	}
}

// TestSeedDefaults_Idempotent verifies the marker short-circuits a second seed.
func TestSeedDefaults_Idempotent(t *testing.T) {
	probes := &fakeProbeStore{count: 0}
	marker := &fakeSeedMarker{seeded: true}
	svc := settings.NewService(probes, &fakeConfigStore{cfg: &config.Config{}}, &fakeAppliers{}, marker)

	if err := svc.SeedDefaults(context.Background(), config.HealthChecksConfig{}); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	if probes.saved != nil {
		t.Error("already-seeded install must not overwrite probes")
	}
}

// TestSeedDefaults_UpgradeGuard verifies that an install with existing probes
// but no marker is marked seeded without overwriting the operator's set.
func TestSeedDefaults_UpgradeGuard(t *testing.T) {
	probes := &fakeProbeStore{count: 3}
	marker := &fakeSeedMarker{}
	svc := settings.NewService(probes, &fakeConfigStore{cfg: &config.Config{}}, &fakeAppliers{}, marker)

	if err := svc.SeedDefaults(context.Background(), config.HealthChecksConfig{}); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	if probes.saved != nil {
		t.Error("upgrade path must not overwrite existing probes")
	}
	if !marker.seeded {
		t.Error("marker must be set on the upgrade path")
	}
}
