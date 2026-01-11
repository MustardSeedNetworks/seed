package httpapi

import (
	"github.com/krisarmstrong/seed/internal/alerts"
	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/health"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/oauth"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
	"github.com/krisarmstrong/seed/internal/update"
)

// ServiceContainer holds all application services organized by domain.
// This reduces the Server struct's field count and enables dependency injection.
// Related issue: #888.
type ServiceContainer struct {
	Auth      *AuthServices
	RateLimit *RateLimitServices
	Network   *NetworkServices
	Discovery *DiscoveryServices
	Sap       *SapServices
	Canopy    *CanopyServices
	Roots     *RootsServices
	RealTime  *RealTimeServices
	Database  *DatabaseServices
	Health    *HealthServices
	Update    *update.Service
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
}

// RateLimitServices groups rate limiting services.
type RateLimitServices struct {
	Login    *RateLimiter
	Endpoint *EndpointRateLimiter
}

// NetworkServices groups core network management services.
type NetworkServices struct {
	Manager     *network.Manager
	LinkMonitor *network.LinkMonitor
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
	Unified          *discovery.UnifiedDiscoveryService // Correlates wired/WiFi/Bluetooth (deprecated)
	Engine           *discovery.DiscoveryEngine         // New unified discovery engine
}

// SapServices groups SAP module services (live telemetry).
type SapServices struct {
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

// CanopyServices groups Canopy module services (Wi-Fi planning).
type CanopyServices struct {
	WiFi    *wifi.Manager
	Scanner *wifi.Scanner
	Survey  *survey.Manager
}

// RootsServices groups Roots module services (path analysis).
// Currently minimal - PublicIP moved to SapServices as it's telemetry-focused.
type RootsServices struct {
	// Traceroute and path analysis are handled directly in handlers
	// as they don't require persistent state
}

// RealTimeServices groups real-time communication services.
type RealTimeServices struct {
	WSHub          *Hub                    // WebSocket hub (deprecated)
	SSEHub         *SSEHub                 // SSE hub for real-time updates
	LogBroadcaster *logging.LogBroadcaster // Log streaming
}

// DatabaseServices groups database-related services.
type DatabaseServices struct {
	DB              *database.DB
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

// NewServiceContainer creates a new empty ServiceContainer.
func NewServiceContainer() *ServiceContainer {
	return &ServiceContainer{
		Auth:      &AuthServices{},
		RateLimit: &RateLimitServices{},
		Network:   &NetworkServices{},
		Discovery: &DiscoveryServices{},
		Sap:       &SapServices{},
		Canopy:    &CanopyServices{},
		Roots:     &RootsServices{},
		RealTime:  &RealTimeServices{},
		Database:  &DatabaseServices{},
		Health:    &HealthServices{},
	}
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
	if sc.RealTime.WSHub != nil {
		sc.RealTime.WSHub.Shutdown()
	}
	if sc.RealTime.SSEHub != nil {
		sc.RealTime.SSEHub.Shutdown()
	}

	// Stop network services
	if sc.Network.LinkMonitor != nil {
		sc.Network.LinkMonitor.Stop()
	}

	// Stop discovery services
	if sc.Discovery.Engine != nil {
		sc.Discovery.Engine.Stop()
	}
	if sc.Discovery.Service != nil {
		sc.Discovery.Service.Stop()
	}

	// Stop SAP services
	if sc.Sap.VLANTraffic != nil {
		sc.Sap.VLANTraffic.Stop()
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
