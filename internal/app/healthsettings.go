package app

// healthsettings.go wires the composition root to the health-checks settings
// application service (ADR-0020, WS-A4). The adapters implement the service's
// ports over the probes table (the endpoint store of record, via the shared
// internal/health/probemap mapping), the live config (DNS/perf toggles), and the
// live DNS/speedtest testers. Collaborators are resolved lazily so a later-set
// value (the api test harness) is honored, and nil collaborators degrade to
// no-ops.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/health/probemap"
	healthsettings "github.com/MustardSeedNetworks/seed/internal/health/settings"
)

// NewHealthSettings builds the health-checks settings use-case over the probes
// table, the live config, and the live DNS/speedtest testers. reschedule is the
// probe-engine reschedule (best-effort, nil-safe); dnsTester/speedTester are lazy
// accessors.
func NewHealthSettings(
	probes func() *database.ProbeRepository,
	reschedule func(context.Context) error,
	cfg *config.Config,
	path string,
	dnsTester func() *dns.Tester,
	speedTester func() *speedtest.Tester,
) *healthsettings.Service {
	return healthsettings.NewService(
		healthProbeStore{probes: probes, reschedule: reschedule},
		healthConfigStore{cfg: cfg, path: path},
		healthAppliers{dnsTester: dnsTester, speedTester: speedTester},
	)
}

// healthProbeStore implements healthsettings.ProbeStore over the probes table via
// the shared probemap mapping, rescheduling the engine after a save.
type healthProbeStore struct {
	probes     func() *database.ProbeRepository
	reschedule func(context.Context) error
}

func (a healthProbeStore) Endpoints(ctx context.Context) (config.HealthChecksConfig, error) {
	return probemap.LoadEndpoints(ctx, a.probes())
}

func (a healthProbeStore) SaveEndpoints(ctx context.Context, hc config.HealthChecksConfig) error {
	probes, err := probemap.ProbesFromConfig(&hc)
	if err != nil {
		return err
	}
	if err = a.probes().ReplaceProbesByKinds(
		ctx, database.DefaultClientID, probemap.Kinds(), probes,
	); err != nil {
		return err
	}
	// Best-effort reschedule: persistence already succeeded, so a reschedule
	// failure is non-fatal — the new set takes effect on the next engine start.
	if a.reschedule != nil {
		_ = a.reschedule(ctx)
	}
	return nil
}

// healthConfigStore implements healthsettings.ConfigStore over the live config,
// owning the lock + on-disk save (Write releases the lock before Save — the #783
// pattern).
type healthConfigStore struct {
	cfg  *config.Config
	path string
}

func (s healthConfigStore) Read(fn func(*config.Config)) {
	s.cfg.RLock()
	defer s.cfg.RUnlock()
	fn(s.cfg)
}

func (s healthConfigStore) Write(fn func(*config.Config) error) error {
	s.cfg.Lock()
	err := fn(s.cfg)
	s.cfg.Unlock()
	if err != nil {
		return err
	}
	return s.cfg.Save(s.path)
}

// healthAppliers implements healthsettings.Appliers over the live DNS and
// speedtest testers, resolved lazily; nil testers are no-ops.
type healthAppliers struct {
	dnsTester   func() *dns.Tester
	speedTester func() *speedtest.Tester
}

func (a healthAppliers) ApplyDNS(hostname string, servers []config.DNSServer) {
	t := a.dnsTester()
	if t == nil {
		return
	}
	if hostname != "" {
		t.SetTestHostname(hostname)
	}
	configured := make([]dns.ConfiguredServer, 0, len(servers))
	for _, d := range servers {
		configured = append(configured, dns.ConfiguredServer{Address: d.Address, Enabled: d.Enabled})
	}
	t.SetConfiguredServers(configured)
}

func (a healthAppliers) ApplySpeedtestServer(id string) {
	if t := a.speedTester(); t != nil {
		t.SetServerID(id)
	}
}
