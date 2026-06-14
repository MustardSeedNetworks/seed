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
	"fmt"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/health/probemap"
	healthsettings "github.com/MustardSeedNetworks/seed/internal/health/settings"
	"github.com/MustardSeedNetworks/seed/internal/probe"
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
	settings func() *database.SettingsRepository,
) *healthsettings.Service {
	return healthsettings.NewService(
		healthProbeStore{probes: probes, reschedule: reschedule},
		healthConfigStore{cfg: cfg, path: path},
		healthAppliers{dnsTester: dnsTester, speedTester: speedTester},
		healthSeedMarker{settings: settings},
	)
}

// healthProbeStore implements healthsettings.ProbeStore over the probes table via
// the shared probemap mapping, rescheduling the engine after a save.
type healthProbeStore struct {
	probes     func() *database.ProbeRepository
	reschedule func(context.Context) error
}

func (a healthProbeStore) Endpoints(ctx context.Context) (config.HealthChecksConfig, error) {
	probes, err := a.list(ctx)
	if err != nil {
		return config.HealthChecksConfig{}, err
	}
	return probemap.EndpointsFromProbes(probes)
}

// list loads the health-check probes and translates rows to domain probes.
func (a healthProbeStore) list(ctx context.Context) ([]probe.Probe, error) {
	rows, err := a.probes().ListProbes(ctx, database.DefaultClientID, "")
	if err != nil {
		return nil, err
	}
	kinds := probemap.Kinds()
	out := make([]probe.Probe, 0, len(rows))
	for _, p := range rows {
		if slices.Contains(kinds, p.Kind) {
			out = append(out, dbProbeToModel(p))
		}
	}
	return out, nil
}

func (a healthProbeStore) List(ctx context.Context) ([]probe.Probe, error) { return a.list(ctx) }

func (a healthProbeStore) Count(ctx context.Context) (int, error) {
	total := 0
	for _, kind := range probemap.Kinds() {
		n, err := a.probes().CountProbes(ctx, database.DefaultClientID, kind)
		if err != nil {
			return 0, fmt.Errorf("count %s probes: %w", kind, err)
		}
		total += n
	}
	return total, nil
}

func (a healthProbeStore) SaveEndpoints(ctx context.Context, hc config.HealthChecksConfig) error {
	domain, err := probemap.ProbesFromConfig(&hc)
	if err != nil {
		return err
	}
	rows := make([]*database.Probe, 0, len(domain))
	for _, p := range domain {
		rows = append(rows, modelToDBProbe(p))
	}
	if err = a.probes().ReplaceProbesByKinds(
		ctx, database.DefaultClientID, probemap.Kinds(), rows,
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

// settingKeyHealthChecksSeeded marks that the factory health-check probes have
// been seeded. Its presence (not the probe count) is the gate.
const settingKeyHealthChecksSeeded = "health_checks.seeded"

// healthSeedMarker implements healthsettings.SeedMarker over the settings KV.
type healthSeedMarker struct {
	settings func() *database.SettingsRepository
}

func (m healthSeedMarker) Seeded(ctx context.Context) (bool, error) {
	v, err := m.settings().GetValue(ctx, settingKeyHealthChecksSeeded)
	if err != nil {
		return false, fmt.Errorf("read health-check seed marker: %w", err)
	}
	return v != "", nil
}

func (m healthSeedMarker) MarkSeeded(ctx context.Context) error {
	if err := m.settings().Set(ctx, settingKeyHealthChecksSeeded, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set health-check seed marker: %w", err)
	}
	return nil
}
