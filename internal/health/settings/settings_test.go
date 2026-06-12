package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/health/settings"
)

type fakeProbeStore struct {
	endpoints config.HealthChecksConfig
	saved     *config.HealthChecksConfig
	loadErr   error
	saveErr   error
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

func TestGetComposesProbesAndConfig(t *testing.T) {
	probes := &fakeProbeStore{endpoints: config.HealthChecksConfig{
		PingTargets: []config.PingTarget{{Name: "g", Host: "8.8.8.8", Enabled: true}},
	}}
	cfg := &config.Config{}
	cfg.DNS.TestHostname = "example.com"
	cfg.HealthChecks.RunSpeedtest = true
	cfg.Speedtest.ServerID = "42"
	svc := settings.NewService(probes, &fakeConfigStore{cfg: cfg}, &fakeAppliers{})

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
		&fakeProbeStore{loadErr: wantErr}, &fakeConfigStore{cfg: &config.Config{}}, &fakeAppliers{})
	if _, err := svc.Get(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("probe load error not surfaced: %v", err)
	}
}

func TestUpdatePersistsProbesConfigAndSyncsTesters(t *testing.T) {
	probes := &fakeProbeStore{}
	cfg := &config.Config{}
	appliers := &fakeAppliers{}
	svc := settings.NewService(probes, &fakeConfigStore{cfg: cfg}, appliers)

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
	svc := settings.NewService(probes, &fakeConfigStore{cfg: cfg}, appliers)

	if err := svc.Update(context.Background(), settings.Settings{DNSHostname: "x"}); err == nil {
		t.Fatal("Update should return the probe save error")
	}
	if cfg.DNS.TestHostname != "" || appliers.dnsHostname != "" {
		t.Error("config/testers must not be touched when probe save fails")
	}
}
