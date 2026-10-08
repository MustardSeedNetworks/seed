package api

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
	"github.com/MustardSeedNetworks/seed/internal/listener/microburst"
	"github.com/MustardSeedNetworks/seed/internal/listener/snmptrap"
	"github.com/MustardSeedNetworks/seed/internal/listener/syslog"
	"github.com/MustardSeedNetworks/seed/internal/listener/voip"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// initListeners wires the passive-ingress listeners (syslog UDP, SNMP
// traps, NetFlow/IPFIX/sFlow) into the engine registry. Each is opt-in via an
// env variable — operators set SEED_SYSLOG_BIND / SEED_SNMP_TRAP_BIND /
// SEED_FLOW_BIND (e.g. ":514", ":162", ":2055") to enable them. Default
// is off because binding to <1024 requires elevated privileges and we
// don't want the server to crash out of the box when run as a non-root
// user. SEED_MICROBURST_IFACE names the probe's own interface to measure
// microbursts on, and SEED_VOIP_IFACE the interface whose RTP calls are
// scored (the probe's own link or a SPAN port). Both are off by default
// because they capture every frame on that link.
//
// V1.0 NMS expansion — Stage A3.5e-4.
func (s *Server) initListeners(store app.ListenerPersistence) {
	logger := logging.GetLogger()

	if addr := os.Getenv("SEED_SYSLOG_BIND"); addr != "" {
		l, err := syslog.New(syslog.Config{
			BindAddr: addr,
			Sink:     store.Events,
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
			BindAddr:   addr,
			Store:      store.Flows,
			Indicators: store.FlowIndicators,
			Sink:       store.Events,
			Logger:     logger,
		})
		if err != nil {
			logger.Warn("flow collector init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("flow collector registry registration failed", "error", regErr)
		}
	}

	if iface := os.Getenv("SEED_MICROBURST_IFACE"); iface != "" {
		l, err := s.newMicroburstListener(iface, store.Microbursts)
		if err != nil {
			logger.Warn("microburst listener init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("microburst listener registry registration failed", "error", regErr)
		}
	}

	if iface := os.Getenv("SEED_VOIP_IFACE"); iface != "" {
		l, err := voip.New(voip.Config{
			Interface: iface,
			Opener:    defaultCaptureOpener(),
			Store:     store.VoIP,
			Sink:      store.Events,
			Logger:    logger,
		})
		if err != nil {
			logger.Warn("voip listener init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("voip listener registry registration failed", "error", regErr)
		}
	}

	if addr := os.Getenv("SEED_SNMP_TRAP_BIND"); addr != "" {
		l, err := snmptrap.New(snmptrap.Config{
			BindAddr:    addr,
			Sink:        store.Events,
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

// bitsPerMegabit converts the interface manager's bits per second to the
// Mb/s the microburst table stores.
const bitsPerMegabit = 1_000_000

// newMicroburstListener measures against the interface's own address and
// negotiated speed, both read from the interface manager.
func (s *Server) newMicroburstListener(iface string, bursts microburst.Store) (*microburst.Listener, error) {
	if s.netMgr == nil {
		return nil, errors.New("microburst: no interface manager")
	}
	info, err := s.netMgr.GetInterface(iface)
	if err != nil {
		return nil, fmt.Errorf("microburst: %w", err)
	}
	mac, err := net.ParseMAC(info.HardwareAddr)
	if err != nil {
		return nil, fmt.Errorf("microburst: %s address %q: %w", iface, info.HardwareAddr, err)
	}
	return microburst.New(microburst.Config{
		Interface:     iface,
		MAC:           mac,
		LinkSpeedMbps: info.Speed / bitsPerMegabit,
		Opener:        defaultCaptureOpener(),
		Store:         bursts,
		Logger:        logging.GetLogger(),
	})
}
