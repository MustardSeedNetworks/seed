// Package orchestrator wires the ten SNMP collectors (sys_info,
// if_table, lldp, cdp, fdp, arp, fdb, routing, host_resources,
// bgp4_mib) into a single [engine.Engine] that the server lifecycle
// registry starts and stops.
//
// Build returns a configured [*snmp.Poller] with every default
// collector registered against a single [sink.Sink] persisting into
// snmp_observations. The orchestrator does not own the SNMP client
// factory — callers inject one ([snmp.ClientFactory]) so production
// can plug in a real gosnmp dialer while tests pass a fake.
package orchestrator

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	Targets       snmp.PollerStorage
	Observations  sink.ObservationsStore
	Scheduler     *scheduler.Scheduler
	ClientFactory snmp.ClientFactory
	Logger        *slog.Logger
	Now           func() time.Time

	// Credentials and Decrypter resolve each target's stored secrets at poll
	// time. Both are required: a poller without them cannot authenticate, and
	// polling unauthenticated is a security failure rather than a degraded mode.
	Credentials snmp.CredentialStore
	Decrypter   snmp.SecretDecrypter
}

// Build returns a *snmp.Poller with all ten default collectors
// registered against a sink that persists into snmp_observations.
// The returned Poller satisfies [engine.Engine] so the server
// registers it directly with the engine registry.
//
// Returns an error if any required Config field is unset.
func Build(cfg Config) (*snmp.Poller, error) {
	if cfg.Targets == nil {
		return nil, errors.New("orchestrator: Targets required")
	}
	if cfg.Observations == nil {
		return nil, errors.New("orchestrator: Observations required")
	}
	if cfg.Scheduler == nil {
		return nil, errors.New("orchestrator: Scheduler required")
	}
	if cfg.ClientFactory == nil {
		return nil, errors.New("orchestrator: ClientFactory required")
	}
	if cfg.Credentials == nil {
		return nil, errors.New("orchestrator: Credentials required")
	}
	if cfg.Decrypter == nil {
		return nil, errors.New("orchestrator: Decrypter required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	persistSink := sink.New(cfg.Observations, logger, now)
	poller := snmp.NewPoller(cfg.Targets, cfg.Scheduler, logger)

	resolver, err := snmp.NewCredentialResolver(cfg.Credentials, cfg.Decrypter)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: %w", err)
	}
	poller.SetCredentialResolver(resolver)

	for _, collector := range Collectors(cfg.ClientFactory, persistSink, now) {
		poller.RegisterCollector(collector)
	}

	return poller, nil
}

// Publisher receives every collector's observations. [sink.Sink] is the
// production implementation; the NIAC acceptance suite counts them instead.
type Publisher interface {
	sysinfo.Publisher
	iftable.Publisher
	lldp.Publisher
	cdp.Publisher
	arp.Publisher
	fdb.Publisher
	routing.Publisher
	hostresources.Publisher
	bgp4.Publisher
}

// Collectors returns the ten collectors a poller runs, in registration
// order. cdp and fdp share a Publisher (CDP), distinguished downstream by
// Observation.TablePrefix.
func Collectors(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) []snmp.Collector {
	return []snmp.Collector{
		sysinfo.New(factory, publisher, now),
		iftable.New(factory, publisher, now),
		lldp.New(factory, publisher, now),
		cdp.New(factory, publisher, now),
		fdp.New(factory, publisher, now),
		arp.New(factory, publisher, now),
		fdb.New(factory, publisher, now),
		routing.New(factory, publisher, now),
		hostresources.New(factory, publisher, now),
		bgp4.New(factory, publisher, now),
	}
}
