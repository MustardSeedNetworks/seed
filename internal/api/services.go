package api

import (
	"github.com/krisarmstrong/seed/internal/alerts"
	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/diagnostics/cable"
	"github.com/krisarmstrong/seed/internal/diagnostics/dns"
	"github.com/krisarmstrong/seed/internal/diagnostics/gateway"
	"github.com/krisarmstrong/seed/internal/diagnostics/iperf"
	"github.com/krisarmstrong/seed/internal/diagnostics/speedtest"
	"github.com/krisarmstrong/seed/internal/diagnostics/vlan"
	"github.com/krisarmstrong/seed/internal/engine"
	"github.com/krisarmstrong/seed/internal/health"
	"github.com/krisarmstrong/seed/internal/license"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/mibdb"
	"github.com/krisarmstrong/seed/internal/netif"
	"github.com/krisarmstrong/seed/internal/oauth"
	"github.com/krisarmstrong/seed/internal/pipeline/publicip"
	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/scheduler"
	"github.com/krisarmstrong/seed/internal/services/discovery"
	"github.com/krisarmstrong/seed/internal/timeseries/retention"
	"github.com/krisarmstrong/seed/internal/update"
	"github.com/krisarmstrong/seed/internal/wifi"
	"github.com/krisarmstrong/seed/internal/wifi/survey"
)

// ServiceContainer holds all application services organized by domain.
// This reduces the Server struct's field count and enables dependency injection.
// Related issue: #888.
type ServiceContainer struct {
	Auth        *AuthServices
	RateLimit   *RateLimitServices
	Network     *NetworkServices
	Discovery   *DiscoveryServices
	Diagnostics *DiagnosticsServices
	Probe       *ProbeServices
	Wireless    *WiFiServices
	RealTime    *RealTimeServices
	Database    *DatabaseServices
	Health      *HealthServices
	Update      *update.Service

	// Engines is the lifecycle registry every long-running engine
	// registers with (probe, retention, snmp-poller, listeners,
	// discovery). Server.Start drives Registry.Start; Server.Shutdown
	// drives Registry.Stop in reverse registration order.
	// V1.0 NMS expansion — Stage A3.5d.
	Engines *engine.Registry
}

// NewServiceContainer creates a new empty ServiceContainer.
func NewServiceContainer() *ServiceContainer {
	return &ServiceContainer{
		Auth:        &AuthServices{},
		RateLimit:   &RateLimitServices{},
		Network:     &NetworkServices{},
		Discovery:   &DiscoveryServices{},
		Diagnostics: &DiagnosticsServices{},
		Probe:       &ProbeServices{},
		Wireless:    &WiFiServices{},
		RealTime:    &RealTimeServices{},
		Database:    &DatabaseServices{},
		Health:      &HealthServices{},
		Engines:     engine.NewRegistry(nil),
	}
}

// GetUpdateService returns the update service.
func (sc *ServiceContainer) GetUpdateService() *update.Service {
	return sc.Update
}

// AuthServices groups authentication and security-related services.
type AuthServices struct {
	Manager        *auth.Manager
	CSRF           *auth.CSRFManager
	SetupToken     *SetupTokenManager
	Recovery       *auth.RecoveryTokenManager
	OAuth          *oauth.Manager
	TrustedProxies *TrustedProxies
	// WebAuthn is the optional WebAuthn (passkeys) manager, populated
	// in server_init.go. Wave 3 (#85).
	WebAuthn *auth.WebAuthnManager
	// License is the offline license manager. Populated lazily in
	// server_init.go (Phase D-2); may be nil in tests that don't
	// exercise license-gated endpoints.
	License *license.Manager
	// APITokens persists personal-access tokens (Phase D-2).
	APITokens *database.APITokenRepository
}

// RateLimitServices groups rate limiting services.
type RateLimitServices struct {
	Login    *RateLimiter
	Endpoint *EndpointRateLimiter
}

// NetworkServices groups core network management services.
//
// LinkMonitor watches the single "primary" interface (cfg.Interface.Default)
// — it stays as-is for the existing card surfaces that bind to one
// interface at a time. LinkMonitorPool watches every interface in the
// Pro multi_interface fan-out (cfg.Interface.AllEthernet() ∪
// cfg.Interface.AllWiFi()). The pool is reconciled on profile change so
// it tracks the operator's current configuration.
type NetworkServices struct {
	Manager         *netif.Manager
	LinkMonitor     *netif.LinkMonitor
	LinkMonitorPool *netif.LinkMonitorPool
}

// DiscoveryServices groups device and network discovery services.
type DiscoveryServices struct {
	Device           *discovery.DeviceDiscovery
	Service          *discovery.Service
	Pipeline         *discovery.Pipeline
	Vulnerability    *discovery.VulnerabilityScanner
	ProblemDetector  *discovery.ProblemDetector
	BluetoothScanner *discovery.BluetoothScanner
	WiFiBridge       *discovery.WiFiBridge
	Profiler         *discovery.DeviceProfiler // Shared profiler for SNMP/ports/fingerprinting
	PortScanner      *discovery.PortScanner    // TCP port scanner
	Engine           *discovery.Engine         // Unified discovery engine (primary)
}

// DiagnosticsServices groups the on-demand network diagnostic testers.
type DiagnosticsServices struct {
	DNS           *dns.Tester
	DNSSecurity   *dns.SecurityScanner
	DHCP          *dhcp.Monitor
	RogueDetector *dhcp.RogueDetector
	Gateway       *gateway.Tester
	VLAN          *vlan.Manager
	VLANTraffic   *vlan.TrafficMonitor
	Speedtest     *speedtest.Tester
	Iperf         *iperf.Manager
	Cable         *cable.Tester
	PublicIP      *publicip.Checker
}

// ProbeServices groups the unified probe engine + its substrate.
// The engine schedules and dispatches every probe-style observation
// (DNS, TLS, PING, TCP, UDP, HTTP, HTTPS, RTSP, DICOM, HL7, FHIR,
// LTI, LDAP, OPCUA, MODBUS, NTP, SIP, 802.1X, cable, transaction)
// and emits ResultEvents to subscribers (alerts pipeline).
//
// V1.0 NMS expansion — Stage A1.8 wires the engine into the
// production server lifecycle. Stage A1.7 will absorb the
// remaining 11 internal/api/health_checks_*.go checkers; until
// then the engine runs in parallel and handles only the kinds
// registered (DNS, TLS in V1.0 baseline).
type ProbeServices struct {
	Engine    *probe.Engine
	Scheduler *scheduler.Scheduler
	Retention *retention.Engine
}

// WiFiServices groups the Wi-Fi visibility services (scan, manage, survey).
type WiFiServices struct {
	WiFi    *wifi.Manager
	Scanner *wifi.Scanner
	Survey  *survey.Manager
}

// RealTimeServices groups real-time communication services.
type RealTimeServices struct {
	SSEHub         *SSEHub                 // SSE hub for real-time updates
	LogBroadcaster *logging.LogBroadcaster // Log streaming
}

// DatabaseServices groups database-related services.
type DatabaseServices struct {
	DB              *database.DB
	MibDB           *mibdb.DB // MIB database for SNMP OID resolution
	RetentionStopCh chan struct{}
}

// HealthServices groups health check monitoring services.
type HealthServices struct {
	Repository      *database.HealthCheckRepository
	Scorer          *health.ScoringService
	SLATracker      *health.SLATracker
	AnomalyDetector *health.AnomalyDetector
	DependencyMgr   *health.DependencyManager
	AlertManager    *alerts.AlertManager
}

// Stop gracefully stops all services in the container.
func (sc *ServiceContainer) Stop() {
	// Stop rate limiters
	if sc.RateLimit.Login != nil {
		sc.RateLimit.Login.Stop()
	}
	if sc.RateLimit.Endpoint != nil {
		sc.RateLimit.Endpoint.Stop()
	}

	// Stop auth services
	if sc.Auth.CSRF != nil {
		sc.Auth.CSRF.Stop()
	}

	// Stop real-time services
	if sc.RealTime.SSEHub != nil {
		sc.RealTime.SSEHub.Shutdown()
	}

	// Stop network services
	if sc.Network.LinkMonitor != nil {
		sc.Network.LinkMonitor.Stop()
	}
	if sc.Network.LinkMonitorPool != nil {
		sc.Network.LinkMonitorPool.Stop()
	}

	// Stop discovery services
	if sc.Discovery.Engine != nil {
		sc.Discovery.Engine.Stop()
	}
	if sc.Discovery.Service != nil {
		sc.Discovery.Service.Stop()
	}

	// Stop telemetry services
	if sc.Diagnostics.VLANTraffic != nil {
		sc.Diagnostics.VLANTraffic.Stop()
	}

	// Stop update service
	if sc.Update != nil {
		sc.Update.Stop()
	}

	// Stop database retention
	if sc.Database.RetentionStopCh != nil {
		close(sc.Database.RetentionStopCh)
		sc.Database.RetentionStopCh = nil
	}

	// Close database
	if sc.Database.DB != nil {
		_ = sc.Database.DB.Close()
	}
}
