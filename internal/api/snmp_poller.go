package api

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	snmporchestrator "github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpclient"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

// initSNMPPoller wires the orchestrator-built [*snmp.Poller] into
// the engine registry. Three things have to be true for the poller
// to do useful work:
//
//  1. The orchestrator needs a [snmp.ClientFactory] — we supply
//     the production gosnmp-backed one from internal/polling/snmp/
//     snmpclient.
//  2. There needs to be at least one row in polling_targets —
//     V1.0 operators populate this via the A5.3 CRUD API. With
//     zero rows the poller still starts (idempotent) but does
//     no work.
//  3. The scheduler needs a tick interval; snmpPollerSchedulerTick
//     defaults to 5s — see the doc comment for the rationale.
//
// V1.0 NMS expansion — Stage A5.4.
func (s *Server) initSNMPPoller(db *database.DB) {
	logger := logging.GetLogger()
	sched := scheduler.New(snmpPollerSchedulerTick)
	factory := snmpclient.NewFactory(snmpclient.Options{})
	// Without a config there is no keyring, so no target could be authenticated.
	// Skip the poller rather than registering one that refuses every target.
	if s.config == nil {
		logger.Warn("snmp poller init skipped: no config, so no credential keyring")
		return
	}
	keyring, err := s.config.CredentialKeyring()
	if err != nil {
		logger.Warn("snmp poller init failed: credential keyring unavailable", "error", err)
		return
	}

	poller, err := snmporchestrator.Build(snmporchestrator.Config{
		Targets:       db.PollingTargets(),
		Observations:  db.SNMPObservations(),
		Scheduler:     sched,
		ClientFactory: factory,
		Logger:        logger,
		Credentials:   db.DeviceCredentials(),
		Decrypter:     keyring,
	})
	if err != nil {
		logger.Warn("snmp poller init failed", "error", err)
		return
	}
	if regErr := s.registerEngineIfLicensed(poller); regErr != nil {
		logger.Warn("snmp poller registry registration failed", "error", regErr)
		return
	}
	s.snmpPoller = poller
}

// reloadSNMPPoller asks the live poller to re-read polling_targets after a
// create, update or delete. A server without a poller (Free tier, no
// keyring, tests) has nothing to reload.
func (s *Server) reloadSNMPPoller(ctx context.Context) {
	if s.snmpPoller == nil {
		return
	}
	if err := s.snmpPoller.Reload(ctx); err != nil {
		logging.FromContext(ctx).WarnContext(ctx, "snmp poller reload failed", "error", err)
	}
}
