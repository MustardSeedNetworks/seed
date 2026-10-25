package api

// server_wiring.go contains the NewServer helpers that build the auth surface,
// the capture and diagnostic tools, the licence manager, the probe engine, the
// alert pipelines and the ADR-0020 use-cases. The database and discovery
// helpers live in server_init.go.

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"
	"github.com/MustardSeedNetworks/foundation/pkg/passkey"

	alertpipeline "github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/paths"
	"github.com/MustardSeedNetworks/seed/internal/pipeline/publicip"
	"github.com/MustardSeedNetworks/seed/internal/probe"
	probeanomaly "github.com/MustardSeedNetworks/seed/internal/probe/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// initAuthSecurity wires the auth manager, CSRF manager, setup-token and
// recovery-token managers, the optional WebAuthn manager, and the
// login/endpoint rate limiters. Split out of NewServer to keep it under the
// funlen limit; kept as one helper since these all share the "auth surface"
// concern and construction order does not matter across them.
func (s *Server) initAuthSecurity(cfg *config.Config, trustedProxies *TrustedProxies) {
	s.authMgr = auth.NewManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.SessionTimeout,
		cfg.Auth.DefaultUsername,
		cfg.Auth.DefaultPasswordHash,
	)
	s.csrf = csrf.NewManager()
	dataDir := paths.Resolve(paths.ModeAuto).DataDir
	s.setupToken = NewSetupTokenManager(dataDir)
	s.recovery = auth.NewRecoveryTokenManager(dataDir)
	s.proxies = trustedProxies

	s.loginLimiter = NewRateLimiter(DefaultRateLimitConfig())
	s.endpointLimiter = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())
}

// initTelemetryAndWiFiServices constructs the diagnostic test tools (DNS,
// gateway, VLAN, speedtest, iperf, cable, public-IP) and the Wi-Fi
// manager/scanner. Grouped together because none of these depend on the
// database or on each other; split out of NewServer to keep it under the
// funlen limit.
func (s *Server) initTelemetryAndWiFiServices(cfg *config.Config) {
	s.dnsTest = dns.NewTesterForInterface("", cfg.DNS.TestHostname, dns.DefaultThresholds(), cfg.Interface.Default)
	s.gatewayTest = gateway.NewTesterForInterface(gateway.DefaultThresholds(), cfg.Interface.Default)
	s.vlanMgr = vlan.NewManager(cfg.Interface.Default)
	s.speedtestTest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	s.iperf = iperfDeps{mgr: iperf.NewManager(), version: iperf.GetVersion}
	s.cableTest = cable.NewTester(cfg.Interface.Default)
	s.publicIP = publicip.NewChecker()

	s.wifiMgr = wifi.NewManager(cfg.Interface.ResolvedWiFi())
	s.wifiScan = wifi.NewScanner(cfg.Interface.ResolvedWiFi())
	s.startWiFiHelper()
}

// initCaptureServices constructs the services that perform live packet capture
// (device discovery via LLDP/CDP/EDP, DHCP monitoring, rogue-DHCP detection, and
// VLAN traffic) and injects the capture port adapter into each. The adapter is
// build-tagged (libpcap under CGO/Windows, a CGO-free no-op otherwise) so the
// domain packages stay CGO-free. See docs/architecture/CGO_BUILD_STRATEGY.md.
func (s *Server) initCaptureServices(cfg *config.Config) {
	captureOpener := defaultCaptureOpener()

	// s.discoverySvc is initialized later, after the shared profiler.
	s.deviceDisc = enumerate.NewDeviceDiscoveryWithOUI(
		cfg.Interface.Default,
		cfg.NetworkDiscovery.OUIFilePath,
		enumerate.WithCapture(captureOpener),
	)
	if cfg.NetworkDiscovery.OUIMaxAge > 0 {
		go s.deviceDisc.RefreshOUI(cfg.NetworkDiscovery.OUIFilePath, cfg.NetworkDiscovery.OUIMaxAge)
	}
	s.dhcpMon = dhcp.NewMonitor(cfg.Interface.Default, dhcp.WithCapture(captureOpener))
	s.rogueDet = dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
		Interface:        cfg.Interface.Default,
		KnownServers:     cfg.DHCP.RogueDetection.KnownServers,
		AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
	}, dhcp.WithCapture(captureOpener))
}

// initLicense wires the Phase D-2 license manager onto the server. It is
// best-effort: failure to load isn't fatal. It leaves s.licenseMgr nil,
// which every gate reads as the pre-license dev state and treats as Pro
// (effectiveTier, the mint flag, and the feature catalogue on
// GET /api/v1/license) — so the warning has to say "unenforced", not
// "disabled", which is what it claimed while permitting minting.
func (s *Server) initLicense() {
	lm, lmErr := s.newLicenseManager()
	if lmErr != nil {
		logging.GetLogger().Warn("license manager init failed; licensing is unenforced (Pro)",
			"error", lmErr)
		return
	}
	if status := lm.LoadStatus(); !status.Usable() {
		logging.GetLogger().Warn("license state is unusable; running as Free until the key is re-entered",
			"status", status.String(), "error", lm.LoadError())
	}
	s.licenseMgr = lm
}

// newLicenseManager builds the manager against s.licenseDir, falling back to
// the real user config directory when it is unset.
func (s *Server) newLicenseManager() (*license.Manager, error) {
	if s.licenseDir != "" {
		return license.NewManagerWithDir(s.licenseDir)
	}

	return license.NewManager()
}

// initProbeEngine constructs the unified probe.Engine, wires it to
// the probes table and a fresh scheduler, registers V1.0 baseline
// Checkers (DNS + TLS), and parks it on s.probeEngine for the
// lifecycle to Start. The engine is *not* started here — that
// happens in Server.Start so probes don't run during partial
// server construction.
//
// V1.0 NMS expansion — Stage A1.8.
func (s *Server) initProbeEngine(storage *app.ProbeStorage) {
	sched := scheduler.New(probeSchedulerTick)

	probeEngine := probe.NewEngine(logging.GetLogger()).
		WithStorage(storage, sched)

	// Register V1.0 baseline checkers plus the health-check vertical
	// checkers absorbed onto the probe engine (ADR-0027 P1).
	probeEngine.RegisterChecker(checkers.NewDNSChecker())
	probeEngine.RegisterChecker(checkers.NewTLSChecker())
	probeEngine.RegisterChecker(checkers.NewPingChecker())
	probeEngine.RegisterChecker(checkers.NewTCPChecker())
	probeEngine.RegisterChecker(checkers.NewUDPChecker())
	probeEngine.RegisterChecker(checkers.NewHTTPChecker())
	probeEngine.RegisterChecker(checkers.NewHTTPSChecker())
	probeEngine.RegisterChecker(checkers.NewRTSPChecker())
	probeEngine.RegisterChecker(checkers.NewDICOMChecker())
	probeEngine.RegisterChecker(checkers.NewHL7Checker())
	probeEngine.RegisterChecker(checkers.NewFHIRChecker())
	probeEngine.RegisterChecker(checkers.NewSQLChecker())
	probeEngine.RegisterChecker(checkers.NewFileShareChecker())
	probeEngine.RegisterChecker(checkers.NewLDAPChecker())
	probeEngine.RegisterChecker(checkers.NewLTIChecker())
	probeEngine.RegisterChecker(checkers.NewOPCUAChecker())
	probeEngine.RegisterChecker(checkers.NewModbusChecker())

	s.probeEngine = probeEngine
	s.probeScheduler = sched
	if regErr := s.registerEngineIfLicensed(probeEngine); regErr != nil {
		logging.GetLogger().Warn("probe engine registry registration failed", "error", regErr)
	}

	// Wire the active-monitoring anomaly producer (ADR-0025): it subscribes to
	// the probe engine's ResultEvent channel here (so it does not miss events
	// before its own Start) and persists threshold breaches as anomalies under
	// source=probe through the shared, server-owned Coordinator (ADR-0029).
	// Subscription happens now; the consume/maintenance loops start with the
	// lifecycle. A nil Coordinator (malformed merged catalog) skips the producer
	// rather than aborting probe startup.
	if s.anomalyCoord == nil {
		logging.GetLogger().Warn("probe anomaly producer not wired: no anomaly coordinator")
		return
	}
	producer, prodErr := probeanomaly.New(
		probeEngine.Subscribe(), s.anomalyCoord,
		probeanomaly.WithLogger(logging.GetLogger()),
	)
	if prodErr != nil {
		logging.GetLogger().Warn("probe anomaly producer not wired", "error", prodErr)
		return
	}
	s.probeAnomaly = producer
	if regErr := s.registerEngineIfLicensed(producer); regErr != nil {
		logging.GetLogger().Warn("probe anomaly producer registry registration failed", "error", regErr)
	}
}

// probeSchedulerTick is the scheduler's tick interval — how often
// it checks whether any registered Job is due. Production default
// 5s; tests can run faster via direct scheduler.New construction.
const probeSchedulerTick = 5 * time.Second

// snmpPollerSchedulerTick is the cadence the snmp.Poller's
// scheduler wakes up at to dispatch due target jobs. The actual
// per-target cadence comes from polling_targets.poll_interval_seconds
// (default 300s); a 5s tick gives 1% scheduling-grain overhead on
// the default-cadence targets and is plenty fast for the 60s
// cadence operators typically use on high-priority devices.
const snmpPollerSchedulerTick = 5 * time.Second

// initAlertPipelines wires the two Stage A4.5 / A4.6 alert
// pipelines into the engine registry. The listener pipeline scans
// listener_events for severe syslog + traps; the observation
// pipeline scans snmp_observations for state transitions
// (iface down, BGP flap, storage thresholds). Both write into the
// existing alerts table via the same Alert repository.
//
// V1.0 NMS expansion — Stage A4 wire-up.
func (s *Server) initAlertPipelines(
	listener alertpipeline.ListenerConfig,
	observation alertpipeline.ObservationConfig,
) {
	logger := logging.GetLogger()
	alerts := s.useAlertDelivery(app.NewAlertDelivery(s.persist, s.config, logger))
	listener.Alerts, listener.Logger = alerts, logger
	observation.Alerts, observation.Logger = alerts, logger

	if p, err := alertpipeline.NewListenerPipeline(listener); err != nil {
		logger.Warn("listener alert pipeline init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(p); regErr != nil {
		logger.Warn("listener alert pipeline registry registration failed", "error", regErr)
	}

	if p, err := alertpipeline.NewObservationPipeline(observation); err != nil {
		logger.Warn("observation alert pipeline init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(p); regErr != nil {
		logger.Warn("observation alert pipeline registry registration failed", "error", regErr)
	}

	s.registerAlertEscalator(app.NewAlertEscalator(s.persist, s.alertDelivery, logger))
}

// initWiFiUseCases wires the Wi-Fi troubleshooting use-cases (ADR-0020) from the
// composition root: the visibility-read, management, and discovery use-cases over
// the server's lazy accessors + live config. Called after initDiscovery so the
// discovery bridge the discovery use-case captures already exists.
func (s *Server) initWiFiUseCases() {
	s.wifiQueries = app.NewWiFiQueries(s.wifiVisibility, s.persistence, s.anomalyEngine)
	s.wifiManagement = app.NewWiFiManagement(s.wifiManager, s.wifiScanner, s.netManager, s.config, s.configPath)
	s.wifiDiscovery = app.NewWiFiDiscovery(s.wifiBridge)
	if v := s.wifiVisibility(); v != nil {
		v.SetScanSource(app.WiFiScanSource(s.wifiManagement))
		v.SetWiredSource(app.WiFiWiredSource(s.wiredDevices, s.persistence))
	}
}

// wiredDevices lists discovery's wired registry, or nothing before it exists.
func (s *Server) wiredDevices() []*enumerate.DiscoveredDevice {
	if d := s.deviceDiscovery(); d != nil {
		return d.GetDevices()
	}
	return nil
}

// rescheduleProbeEngine reschedules the running probe engine after a settings
// save; a nil engine is a no-op (best-effort, ADR-0027).
func (s *Server) rescheduleProbeEngine(ctx context.Context) error {
	if s.probeEngine == nil {
		return nil
	}
	return s.probeEngine.Reschedule(ctx)
}

// initEngineUseCases wires the engine-status use-case (ADR-0020) from the
// composition root over the server's lazy accessor for the engine registry,
// so a nil or later-set registry (the api test harness) is honored.
func (s *Server) initEngineUseCases() {
	s.engineStatus = app.NewEngineStatus(s.engineRegistry)
}

// initIdentityUseCases wires the identity use-cases (ADR-0020, ADR-0024) from
// the composition root over the server's lazy accessors for the database and
// the license manager, so a nil or later-set collaborator (the api test
// harness) is honored.
func (s *Server) initIdentityUseCases() {
	s.identityUsers = app.NewIdentityUsers(s.persistence)
	s.identityTokens = app.NewIdentityTokens(s.persistence, s.licenseManager)
	s.identityOAuth = app.NewIdentityOAuth(s.persistence)
	s.identityMFA = app.NewMFAStore(s.persistence)
}

// initUseCases wires the ADR-0020 application use-cases that depend on the
// discovery components existing: troubleshooting + discovery + health +
// identity + engine-status.
func (s *Server) initUseCases() {
	s.initWiFiUseCases()
	s.initDiscoveryUseCases()
	s.initHealthUseCases()
	s.initEngineUseCases()
	s.initIdentityUseCases()
	s.initTFTP()
}

// passkeyConfigFromServer derives the relying-party config from the exact
// public HTTPS origin, or from the port actually bound for a local deployment.
func passkeyConfigFromServer(cfg *config.Config, listenerPort int) passkey.Config {
	const (
		devHost           = "localhost"
		defaultPort       = 8443
		standardHTTPSPort = 443
	)
	if listenerPort <= 0 {
		listenerPort = defaultPort
	}

	rpid := devHost
	originHost := devHost
	if listenerPort != standardHTTPSPort {
		originHost = net.JoinHostPort(devHost, strconv.Itoa(listenerPort))
	}
	origin := (&url.URL{Scheme: "https", Host: originHost}).String()
	if cfg != nil && cfg.Server.PublicOrigin != "" {
		if publicURL, err := url.Parse(cfg.Server.PublicOrigin); err == nil && publicURL.Hostname() != "" {
			rpid = publicURL.Hostname()
			origin = cfg.Server.PublicOrigin
		}
	}
	return passkey.Config{RPID: rpid, Origin: origin, DisplayName: "Seed"}
}

func (s *Server) initPasskeys(listenerPort int) {
	service, err := passkey.New(passkeyConfigFromServer(s.config, listenerPort))
	if err != nil {
		logging.GetLogger().Warn("Passkeys disabled: the public origin is not a valid relying party", "error", err)
	}
	s.passkeys = service
}
