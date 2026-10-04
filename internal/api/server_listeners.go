package api

import (
	"os"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
	listenersink "github.com/MustardSeedNetworks/seed/internal/listener/sink"
	"github.com/MustardSeedNetworks/seed/internal/listener/snmptrap"
	"github.com/MustardSeedNetworks/seed/internal/listener/syslog"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// initListeners wires the passive-ingress listeners (syslog UDP, SNMP
// traps, NetFlow/IPFIX) into the engine registry. Each is opt-in via an
// env variable — operators set SEED_SYSLOG_BIND / SEED_SNMP_TRAP_BIND /
// SEED_FLOW_BIND (e.g. ":514", ":162", ":2055") to enable them. Default
// is off because binding to <1024 requires elevated privileges and we
// don't want the server to crash out of the box when run as a non-root
// user.
//
// V1.0 NMS expansion — Stage A3.5e-4.
func (s *Server) initListeners(db *database.DB) {
	persistSink := listenersink.New(db.ListenerEvents(), logging.GetLogger(), nil)
	logger := logging.GetLogger()

	if addr := os.Getenv("SEED_SYSLOG_BIND"); addr != "" {
		l, err := syslog.New(syslog.Config{
			BindAddr: addr,
			Sink:     persistSink,
			Logger:   logger,
		})
		if err != nil {
			logger.Warn("syslog listener init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("syslog listener registry registration failed", "error", regErr)
		}
	}

	if addr := os.Getenv("SEED_FLOW_BIND"); addr != "" {
		l, err := flow.New(flow.Config{
			BindAddr: addr,
			Store:    db.FlowRecords(),
			Logger:   logger,
		})
		if err != nil {
			logger.Warn("flow collector init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("flow collector registry registration failed", "error", regErr)
		}
	}

	if addr := os.Getenv("SEED_SNMP_TRAP_BIND"); addr != "" {
		l, err := snmptrap.New(snmptrap.Config{
			BindAddr:    addr,
			Sink:        persistSink,
			Credentials: s.snmpCreds,
			Logger:      logger,
		})
		if err != nil {
			logger.Warn("snmp trap listener init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("snmp trap listener registry registration failed", "error", regErr)
		}
	}
}
