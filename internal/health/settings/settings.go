// Package settings is the application service for the health-checks settings
// endpoint (ADR-0020 clean-hexagonal, WS-A4). It composes the two stores of
// record: the endpoint target lists live in the probes table (ProbeStore, since
// ADR-0027 P2) while the DNS/performance/speedtest/iperf toggles remain
// config-file backed (ConfigStore). On update it also syncs the live DNS and
// speedtest testers through the Appliers port. All three ports are satisfied by
// adapters in the composition root.
package settings

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// ProbeStore is the endpoint-target store of record (the probes table). Endpoints
// loads the configured health-check targets; SaveEndpoints replaces them (and
// reschedules the running probe engine).
type ProbeStore interface {
	Endpoints(ctx context.Context) (config.HealthChecksConfig, error)
	SaveEndpoints(ctx context.Context, hc config.HealthChecksConfig) error
}

// ConfigStore reads/persists the config-file-backed toggles. Read runs fn under
// the config RLock; Write runs fn under the write lock and saves after releasing
// it (the #783 unlock-before-save pattern).
type ConfigStore interface {
	Read(fn func(*config.Config))
	Write(fn func(*config.Config) error) error
}

// Appliers syncs the live testers with newly-saved settings.
type Appliers interface {
	ApplyDNS(hostname string, servers []config.DNSServer)
	ApplySpeedtestServer(id string)
}

// Service is the health-checks settings application service.
type Service struct {
	probes   ProbeStore
	cfg      ConfigStore
	appliers Appliers
}

// NewService builds the service over its ports.
func NewService(probes ProbeStore, cfg ConfigStore, appliers Appliers) *Service {
	return &Service{probes: probes, cfg: cfg, appliers: appliers}
}

// Settings is the read/write model: the endpoint targets plus the
// config-file-backed toggles.
type Settings struct {
	Endpoints config.HealthChecksConfig

	DNSHostname string
	DNSServers  []config.DNSServer

	RunPerformance bool
	RunSpeedtest   bool
	RunIperf       bool
	RunDiscovery   bool

	SpeedtestServerID      string
	SpeedtestAutoRunOnLink bool
	IperfAutoRunOnLink     bool
}

// Get composes the current settings: endpoint targets from the probes table and
// the toggles from the config.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	eps, err := s.probes.Endpoints(ctx)
	if err != nil {
		return Settings{}, err
	}
	out := Settings{Endpoints: eps}
	s.cfg.Read(func(c *config.Config) {
		out.DNSHostname = c.DNS.TestHostname
		out.DNSServers = c.DNS.Servers
		out.RunPerformance = c.HealthChecks.RunPerformance
		out.RunSpeedtest = c.HealthChecks.RunSpeedtest
		out.RunIperf = c.HealthChecks.RunIperf
		out.RunDiscovery = c.HealthChecks.RunDiscovery
		out.SpeedtestServerID = c.Speedtest.ServerID
		out.SpeedtestAutoRunOnLink = c.Speedtest.AutoRunOnLink
		out.IperfAutoRunOnLink = c.Iperf.AutoRunOnLink
	})
	return out, nil
}

// Update persists the settings: the endpoint targets are saved to the probes
// table first (the store of record), then the toggles are written to the config,
// and finally the live testers are synced. A non-empty DNS hostname is applied;
// the DNS server list always replaces the prior one (the original contract).
func (s *Service) Update(ctx context.Context, in Settings) error {
	if err := s.probes.SaveEndpoints(ctx, in.Endpoints); err != nil {
		return err
	}
	if err := s.cfg.Write(func(c *config.Config) error {
		if in.DNSHostname != "" {
			c.DNS.TestHostname = in.DNSHostname
		}
		c.DNS.Servers = in.DNSServers
		c.HealthChecks.RunPerformance = in.RunPerformance
		c.HealthChecks.RunSpeedtest = in.RunSpeedtest
		c.HealthChecks.RunIperf = in.RunIperf
		c.HealthChecks.RunDiscovery = in.RunDiscovery
		c.Speedtest.ServerID = in.SpeedtestServerID
		c.Speedtest.AutoRunOnLink = in.SpeedtestAutoRunOnLink
		c.Iperf.AutoRunOnLink = in.IperfAutoRunOnLink
		return nil
	}); err != nil {
		return err
	}
	s.appliers.ApplyDNS(in.DNSHostname, in.DNSServers)
	s.appliers.ApplySpeedtestServer(in.SpeedtestServerID)
	return nil
}
