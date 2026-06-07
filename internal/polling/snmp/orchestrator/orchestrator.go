// Package orchestrator wires the eleven Stage A3 SNMP collectors
// (sys_info, if_table, lldp, cdp, fdp, arp, fdb, routing,
// host_resources, bgp4_mib) into a single [engine.Engine] that the
// server lifecycle registry starts and stops.
//
// Build returns a configured [*snmp.Poller] with every default
// collector registered against a single [sink.Sink] persisting into
// snmp_observations. The orchestrator does not own the SNMP client
// factory — callers inject one ([snmp.ClientFactory]) so production
// can plug in a real gosnmp dialer while tests pass a fake.
package orchestrator

import (
	"errors"
	"log/slog"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/bgp4"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/cdp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/lldp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/routing"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/sysinfo"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

// Config holds the dependencies the orchestrator needs to build a
// fully wired Poller. Logger and Now are optional — nil values fall
// back to [slog.Default] and [time.Now].UTC respectively.
type Config struct {
	DB            *database.DB
	Scheduler     *scheduler.Scheduler
	ClientFactory snmp.ClientFactory
	Logger        *slog.Logger
	Now           func() time.Time
}

// Build returns a *snmp.Poller with all eleven default collectors
// registered against a sink that persists into snmp_observations.
// The returned Poller satisfies [engine.Engine] so the server
// registers it directly with the engine registry.
//
// Returns an error if any required Config field is unset.
func Build(cfg Config) (*snmp.Poller, error) {
	if cfg.DB == nil {
		return nil, errors.New("orchestrator: DB required")
	}
	if cfg.Scheduler == nil {
		return nil, errors.New("orchestrator: Scheduler required")
	}
	if cfg.ClientFactory == nil {
		return nil, errors.New("orchestrator: ClientFactory required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	persistSink := sink.New(cfg.DB.SNMPObservations(), logger, now)
	poller := snmp.NewPoller(cfg.DB.PollingTargets(), cfg.Scheduler, logger)

	// Register every collector. cdp + fdp share a Publisher (CDP),
	// distinguished downstream by Observation.TablePrefix.
	poller.RegisterCollector(sysinfo.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(iftable.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(lldp.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(cdp.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(fdp.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(arp.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(fdb.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(routing.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(hostresources.New(cfg.ClientFactory, persistSink, now))
	poller.RegisterCollector(bgp4.New(cfg.ClientFactory, persistSink, now))

	return poller, nil
}
