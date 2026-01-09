package discovery

// Device profiler module performs deep inspection of discovered devices through HTTP,
// SNMP, mDNS, and port scanning to gather detailed information about capabilities,
// services, and device types. Enables intelligent device identification and visualization hints.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// Device type constants for profiler classification.
const (
	deviceTypePrinter       = "printer"
	deviceTypeNetworkDevice = "network-device"
	deviceTypeServer        = "server"
	deviceTypeRouter        = "router"
	deviceTypeSwitch        = "switch"
	deviceTypeFirewall      = "firewall"
	deviceTypeNAS           = "nas"
)

// Profiler timing and buffer constants.
const (
	profilerQueueSize         = 100  // Size of profiling queue channel
	profilerDefaultTimeoutS   = 10   // Default timeout for profiling operations in seconds
	profilerBannerReadMs      = 500  // Timeout for reading service banners in milliseconds
	profilerBannerBufferSize  = 256  // Buffer size for reading service banners
	profilerHTTPBodyLimit     = 8192 // Maximum HTTP response body to read
	profilerLogTruncateLen    = 50   // Maximum length for log message truncation
	profilerMinTruncateLen    = 3    // Minimum length for truncation with ellipsis
	profilerTitleMaxLen       = 100  // Maximum length for extracted HTML titles
	profilerTimeoutS          = 2    // Default profiler timeout in seconds
	profilerMaxConcurrent     = 10   // Default max concurrent profiling operations
	profilerProbeDelayMs      = 50   // Default probe delay in milliseconds
	profilerHostDelayMs       = 20   // Default host delay in milliseconds
	profilerNameResolveTimeMs = 500  // Default name resolution timeout in milliseconds
)

// Common port numbers for service classification.
const (
	portFTP        = 21
	portSSHProf    = 22
	portTelnet     = 23
	portSMTP       = 25
	portDNS        = 53
	portHTTPProf   = 80
	portPOP3       = 110
	portIMAP       = 143
	portSNMP       = 161
	portSMTPSubmit = 587
	portMySQL      = 3306
	portPostgreSQL = 5432
	portRedis      = 6379
	portHTTPAltP   = 8080
	portHTTPSProf  = 443
	portHTTPSAltP  = 8443
	portJetDirect  = 9100
	portLPD        = 515
	portIPP        = 631
	portMongoDB    = 27017
)

// DeviceProfile contains auto-discovered profile information about a device.
type DeviceProfile struct {
	ProfiledAt   time.Time     `json:"profiledAt"`
	OpenPorts    []OpenPort    `json:"openPorts,omitempty"`
	HTTPInfo     *HTTPInfo     `json:"httpInfo,omitempty"`
	SNMPInfo     *SNMPInfo     `json:"snmpInfo,omitempty"`
	MDNSServices []MDNSService `json:"mdnsServices,omitempty"`
	DeviceType   string        `json:"deviceType,omitempty"`  // Inferred type: router, switch, printer, server, etc.
	DeviceIcons  []string      `json:"deviceIcons,omitempty"` // Icon hints for UI: web, ssh, snmp, printer, etc.
}

// OpenPort represents an open port found during profiling.
type OpenPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp or udp
	Service  string `json:"service,omitempty"`
	Banner   string `json:"banner,omitempty"`
	IsOpen   bool   `json:"isOpen"`
}

// HTTPInfo contains HTTP/HTTPS probe results.
type HTTPInfo struct {
	Port       int    `json:"port"`
	StatusCode int    `json:"statusCode"`
	Title      string `json:"title,omitempty"`
	Server     string `json:"server,omitempty"`
	IsHTTPS    bool   `json:"isHttps"`
}

// SNMPInfo contains SNMP probe results.
type SNMPInfo struct {
	SysDescr    string `json:"sysDescr,omitempty"`
	SysName     string `json:"sysName,omitempty"`
	SysContact  string `json:"sysContact,omitempty"`
	SysLocation string `json:"sysLocation,omitempty"`
}

// MDNSService represents an mDNS/Bonjour advertised service.
type MDNSService struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	Port int               `json:"port,omitempty"`
	TXT  map[string]string `json:"txt,omitempty"`
}

// ProfilerConfig holds configuration for the device profiler.
type ProfilerConfig struct {
	Enabled       bool
	Timeout       time.Duration
	MaxConcurrent int
	QuickPorts    []int // Ports to check during quick profile

	// Enhanced settings for pipeline integration
	PortScanIntensity PortScanIntensity // Intensity level for port scanning
	TimingProfile     ScanTimingProfile // Timing profile for rate limiting
	CustomPorts       []int             // Custom port list when intensity is PortScanCustom
	BannerGrab        bool              // Whether to attempt banner grabbing
	ProbeDelay        time.Duration     // Delay between probes to same host
	HostDelay         time.Duration     // Delay between starting different hosts
	ConnectTimeout    time.Duration     // Timeout for TCP connections

	// TLS configuration for HTTPS probing
	// SkipTLSVerify allows connecting to devices with self-signed certificates.
	// This is common for network devices, printers, and other internal infrastructure.
	// Default: true (required for profiling internal network devices)
	SkipTLSVerify bool

	// EnableSNMPCollection enables automatic full SNMP MIB collection during profiling.
	// When true and SNMP credentials are configured, the profiler will collect full
	// interface MIB data (IF-MIB, IP-MIB, BRIDGE-MIB, etc.) from each device.
	EnableSNMPCollection bool

	// SNMPMIBs specifies which MIBs to collect. If nil, defaults to interface MIBs.
	SNMPMIBs *SNMPMIBSelection

	// EnableNameResolution enables automatic DNS, NetBIOS, and mDNS name resolution.
	EnableNameResolution bool

	// ResolveDNS enables reverse DNS PTR lookups.
	ResolveDNS bool

	// ResolveNetBIOS enables NetBIOS name queries for Windows devices.
	ResolveNetBIOS bool

	// ResolveMDNS enables mDNS queries for Apple/Linux devices.
	ResolveMDNS bool

	// NameResolutionTimeout is the timeout for each name resolution query.
	NameResolutionTimeout time.Duration
}

// DefaultProfilerConfig returns sensible defaults.
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:       true,
		Timeout:       profilerTimeoutS * time.Second,
		MaxConcurrent: profilerMaxConcurrent,
		QuickPorts: []int{
			22,   // SSH
			23,   // Telnet
			80,   // HTTP
			443,  // HTTPS
			8080, // HTTP Alt
			8443, // HTTPS Alt
			// Note: SNMP (port 161) is UDP, probed separately via probeSNMP()
		},
		PortScanIntensity: PortScanOff, // Default: OFF for security
		TimingProfile:     ScanProfileNormal,
		BannerGrab:        true,
		ProbeDelay:        profilerProbeDelayMs * time.Millisecond,
		HostDelay:         profilerHostDelayMs * time.Millisecond,
		ConnectTimeout:    profilerTimeoutS * time.Second,
		SkipTLSVerify:     false, // Set to true for internal network devices with self-signed certs
		// Enable automatic SNMP collection when credentials are configured
		EnableSNMPCollection: true,
		// Default to interface MIBs for network device discovery
		SNMPMIBs: &SNMPMIBSelection{
			System:      true,  // sysDescr, sysName, sysLocation, etc.
			Interfaces:  true,  // IF-MIB (interface speeds, MACs, status)
			IPAddresses: true,  // IP-MIB (device IPs)
			Bridge:      true,  // BRIDGE-MIB (MAC table for switches)
			VLAN:        true,  // Q-BRIDGE-MIB (VLAN info)
			LLDP:        true,  // LLDP-MIB (neighbor discovery)
			Routing:     false, // IP-FORWARD-MIB (disable by default - can be large)
			Entity:      false, // ENTITY-MIB (disable by default - not always useful)
		},
		// Enable automatic name resolution
		EnableNameResolution:  true,
		ResolveDNS:            true, // DNS PTR lookups for all IPs
		ResolveNetBIOS:        true, // NetBIOS for Windows devices
		ResolveMDNS:           true, // mDNS for Apple/Linux devices
		NameResolutionTimeout: profilerNameResolveTimeMs * time.Millisecond,
	}
}

// NewProfilerConfigFromPipeline creates a ProfilerConfig from pipeline settings.
func NewProfilerConfigFromPipeline(pipelineConfig *PipelineConfig) *ProfilerConfig {
	cfg := DefaultProfilerConfig()
	cfg.PortScanIntensity = pipelineConfig.PortScan.Intensity
	cfg.CustomPorts = pipelineConfig.PortScan.CustomPorts
	cfg.BannerGrab = pipelineConfig.PortScan.BannerGrab
	cfg.ConnectTimeout = pipelineConfig.PortScan.ConnectTimeout
	cfg.TimingProfile = pipelineConfig.Timing.Profile
	cfg.ProbeDelay = pipelineConfig.Timing.ProbeDelay
	cfg.HostDelay = pipelineConfig.Timing.HostDelay
	cfg.MaxConcurrent = pipelineConfig.Timing.MaxConcurrentHosts
	cfg.Timeout = pipelineConfig.Timing.PhaseTimeout
	return cfg
}

// GetPortsForIntensity returns the appropriate port list based on intensity level.
func (c *ProfilerConfig) GetPortsForIntensity() []int {
	switch c.PortScanIntensity {
	case PortScanOff:
		return nil
	case PortScanQuick:
		return GetQuickPorts()
	case PortScanStandard:
		return GetStandardPorts()
	case PortScanComprehensive:
		return GetComprehensivePorts()
	case PortScanCustom:
		return c.CustomPorts
	default:
		return nil
	}
}

// ResolvedNames holds resolved names for a device.
type ResolvedNames struct {
	Hostname    string // DNS PTR resolved name
	NetBIOSName string // Windows NetBIOS name
	MDNSName    string // mDNS/Bonjour .local name
}

// DeviceProfiler automatically profiles newly discovered devices.
type DeviceProfiler struct {
	config          *ProfilerConfig
	snmpConfig      *config.SNMPConfig
	snmpCollector   *SNMPCollector   // For full MIB collection
	netbiosResolver *NetBIOSResolver // For NetBIOS name queries
	mdnsResolver    *MDNSResolver    // For mDNS name queries
	httpClient      *http.Client
	transport       *http.Transport // Store transport for cleanup (fixes #825)
	mu              sync.RWMutex
	profiles        map[string]*DeviceProfile // key by IP
	snmpData        map[string]*SNMPFullData  // Full SNMP MIB data by IP
	resolvedNames   map[string]*ResolvedNames // Resolved names by IP
	profiling       map[string]bool           // track in-progress profiles
	queue           chan string               // IPs to profile
	stopCh          chan struct{}
	wg              sync.WaitGroup
	interfaceName   string // For mDNS resolver
}

// newProfilerTLSConfig creates a TLS config for the device profiler.
// When insecure is true, the config uses a custom verification function
// that accepts all certificates (required for internal network devices
// with self-signed certificates).
func newProfilerTLSConfig(insecure bool) *tls.Config {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if insecure {
		// Use custom verification that accepts all certificates.
		// This is required for profiling internal network devices with self-signed certs.
		cfg.VerifyPeerCertificate = func(_ [][]byte, _ [][]*x509.Certificate) error {
			return nil // Accept all certificates
		}
		cfg.VerifyConnection = func(_ tls.ConnectionState) error {
			return nil // Accept all connections
		}
	}
	return cfg
}

// NewDeviceProfiler creates a new device profiler.
func NewDeviceProfiler(cfg *ProfilerConfig, snmpCfg *config.SNMPConfig) *DeviceProfiler {
	if cfg == nil {
		cfg = DefaultProfilerConfig()
	}

	transport := &http.Transport{
		TLSClientConfig: newProfilerTLSConfig(cfg.SkipTLSVerify),
		DialContext: (&net.Dialer{
			Timeout: cfg.Timeout,
		}).DialContext,
	}

	p := &DeviceProfiler{
		config:     cfg,
		snmpConfig: snmpCfg,
		transport:  transport, // Store for cleanup (fixes #825)
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
		profiles:      make(map[string]*DeviceProfile),
		snmpData:      make(map[string]*SNMPFullData),
		resolvedNames: make(map[string]*ResolvedNames),
		profiling:     make(map[string]bool),
		queue:         make(chan string, profilerQueueSize),
	}

	// Initialize SNMP collector if credentials are configured and collection is enabled
	if cfg.EnableSNMPCollection && snmpCfg != nil && p.hasSNMPCredentials() {
		mibConfig := SNMPMIBSelection{
			System:      true,
			Interfaces:  true,
			IPAddresses: true,
			Bridge:      true,
			VLAN:        true,
			LLDP:        true,
		}
		if cfg.SNMPMIBs != nil {
			mibConfig = *cfg.SNMPMIBs
		}
		p.snmpCollector = NewSNMPCollector(snmpCfg, mibConfig)
		logging.GetLogger().Info("SNMP collector initialized for automatic MIB polling",
			"communities", len(snmpCfg.Communities),
			"v3creds", len(snmpCfg.V3Credentials))
	}

	// Initialize name resolvers if enabled
	if cfg.EnableNameResolution {
		if cfg.ResolveNetBIOS {
			p.netbiosResolver = NewNetBIOSResolver()
		}
		if cfg.ResolveMDNS {
			p.mdnsResolver = NewMDNSResolver("") // Interface set later via SetInterface
		}
		logging.GetLogger().Info("Name resolution initialized",
			"dns", cfg.ResolveDNS,
			"netbios", cfg.ResolveNetBIOS,
			"mdns", cfg.ResolveMDNS)
	}

	return p
}

// UpdateScanConfig updates the port scanning configuration.
// This allows Pipeline to set the scan intensity without recreating the profiler.
// Thread-safe: can be called while profiler is running.
func (p *DeviceProfiler) UpdateScanConfig(
	intensity PortScanIntensity,
	customPorts []int,
	timing ScanTimingProfile,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.PortScanIntensity = intensity
	p.config.CustomPorts = customPorts
	p.config.TimingProfile = timing
	logging.GetLogger().
		Info("Updated profiler scan config", "intensity", intensity, "timing", timing)
}

// Start begins the profiler worker pool.
func (p *DeviceProfiler) Start() {
	p.mu.Lock()
	if p.stopCh != nil {
		p.mu.Unlock()
		return // Already running
	}
	p.stopCh = make(chan struct{})

	// Add to WaitGroup before unlocking to prevent race with Stop()
	p.wg.Add(p.config.MaxConcurrent)
	p.mu.Unlock()

	// Start worker goroutines
	for range p.config.MaxConcurrent {
		go p.worker()
	}

	logging.GetLogger().Info("Device profiler started", "workers", p.config.MaxConcurrent)
}

// Stop stops the profiler.
func (p *DeviceProfiler) Stop() {
	p.mu.Lock()
	if p.stopCh == nil {
		p.mu.Unlock()
		return
	}
	close(p.stopCh)
	p.stopCh = nil
	p.mu.Unlock()

	p.wg.Wait()

	// Close idle connections to prevent resource leak (fixes #825)
	if p.transport != nil {
		p.transport.CloseIdleConnections()
	}

	logging.GetLogger().Info("Device profiler stopped")
}

// worker processes profile requests from the queue.
// Fixes #981: Added nil check and defensive handling for stopCh.
func (p *DeviceProfiler) worker() {
	defer p.wg.Done()

	// Capture stopCh locally to avoid race with Stop() setting it to nil
	p.mu.Lock()
	stopCh := p.stopCh
	queue := p.queue
	p.mu.Unlock()

	// Fixes #981: If stopCh is nil, worker should exit immediately
	if stopCh == nil {
		return
	}

	for {
		select {
		case <-stopCh:
			return
		case ip, ok := <-queue:
			// Fixes #981: Handle queue close (ok=false means channel closed)
			if !ok {
				return
			}
			p.profileDevice(ip)
		}
	}
}

// QueueProfile adds an IP to the profiling queue.
// Fixes #888: Returns error to distinguish between skip reasons.
// Fixes #930: Check if profiler is started before accepting queue items.
func (p *DeviceProfiler) QueueProfile(ip string) error {
	if !p.config.Enabled {
		logging.GetLogger().Debug("QueueProfile skipped - profiler disabled", "ip", ip)
		return errors.New("profiler disabled")
	}
	if ip == "" {
		return errors.New("empty IP address")
	}

	p.mu.Lock()
	// Fixes #930: Check if profiler is started
	if p.stopCh == nil {
		p.mu.Unlock()
		logging.GetLogger().Debug("QueueProfile skipped - profiler not started", "ip", ip)
		return errors.New("profiler not started")
	}
	// Skip if already profiled or in progress
	if _, exists := p.profiles[ip]; exists {
		p.mu.Unlock()
		logging.GetLogger().Debug("QueueProfile skipped - already profiled", "ip", ip)
		return nil // Not an error, just already done
	}
	if p.profiling[ip] {
		p.mu.Unlock()
		logging.GetLogger().Debug("QueueProfile skipped - in progress", "ip", ip)
		return nil // Not an error, already in progress
	}
	p.profiling[ip] = true
	p.mu.Unlock()

	select {
	case p.queue <- ip:
		logging.GetLogger().Info("Queued device for profiling", "ip", ip)
		return nil
	default:
		// Queue full, skip (fixes #888)
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
		logging.GetLogger().Warn("Profile queue full, skipped", "ip", ip)
		return errors.New("profile queue full")
	}
}

// checkShutdown checks if the profiler is shutting down.
// Returns the stopCh if active, or nil if shutdown is in progress/complete.
func (p *DeviceProfiler) checkShutdown() chan struct{} {
	p.mu.RLock()
	stopCh := p.stopCh
	p.mu.RUnlock()

	if stopCh == nil {
		return nil
	}

	select {
	case <-stopCh:
		return nil
	default:
		return stopCh
	}
}

// createProfilingContext creates a context for profiling that respects shutdown signals.
// Returns the context, a cancel function, and a cleanup function that must be called when done.
func (p *DeviceProfiler) createProfilingContext(
	stopCh chan struct{},
) (context.Context, context.CancelFunc, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), profilerDefaultTimeoutS*time.Second)

	// Create cancellable context that respects shutdown (fixes #828)
	ctx, cancelWithShutdown := context.WithCancel(ctx)
	go func() {
		select {
		case <-stopCh:
			cancelWithShutdown()
		case <-ctx.Done():
		}
	}()

	cleanup := func() {
		cancelWithShutdown()
		cancel()
	}

	return ctx, cancelWithShutdown, cleanup
}

// scanPorts scans the configured ports and returns open ports found.
func (p *DeviceProfiler) scanPorts(ctx context.Context, ip string) []OpenPort {
	portsToScan := p.config.GetPortsForIntensity()
	if len(portsToScan) == 0 {
		portsToScan = p.config.QuickPorts
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	openPorts := []OpenPort{}

	sem := make(chan struct{}, p.config.MaxConcurrent)

	for _, port := range portsToScan {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			if result := p.scanSinglePort(ctx, ip, port, sem); result != nil {
				mu.Lock()
				openPorts = append(openPorts, *result)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()
	return openPorts
}

// scanSinglePort scans a single port with rate limiting and returns the result if open.
func (p *DeviceProfiler) scanSinglePort(
	ctx context.Context,
	ip string,
	port int,
	sem chan struct{},
) *OpenPort {
	// Check for context cancellation before acquiring semaphore (fixes #834)
	select {
	case <-ctx.Done():
		return nil
	case sem <- struct{}{}:
	}
	defer func() { <-sem }()

	// Apply probe delay for IDS-friendly scanning with context check (fixes #834)
	if p.config.ProbeDelay > 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(p.config.ProbeDelay):
		}
	}

	result := p.checkPortWithConfig(ctx, ip, port)
	if result.IsOpen {
		return &result
	}
	return nil
}

// probeHTTPFromOpenPorts probes HTTP/HTTPS on open ports and returns the first successful result.
func (p *DeviceProfiler) probeHTTPFromOpenPorts(
	ctx context.Context,
	ip string,
	openPorts []OpenPort,
) *HTTPInfo {
	for _, op := range openPorts {
		if op.Port == 80 || op.Port == 8080 {
			if info := p.probeHTTP(ctx, ip, op.Port, false); info != nil {
				return info
			}
		}
		if op.Port == 443 || op.Port == 8443 {
			if info := p.probeHTTP(ctx, ip, op.Port, true); info != nil {
				return info
			}
		}
	}
	return nil
}

// probeSNMPAndLog probes SNMP and logs the result if successful.
func (p *DeviceProfiler) probeSNMPAndLog(ctx context.Context, ip string) *SNMPInfo {
	info := p.probeSNMP(ctx, ip)
	if info != nil {
		logging.GetLogger().
			DebugContext(ctx, "Got SNMP info from device", "ip", ip, "sysName", info.SysName)
	}
	return info
}

// storeProfileAndLog stores the profile and logs the result.
func (p *DeviceProfiler) storeProfileAndLog(
	ctx context.Context,
	ip string,
	profile *DeviceProfile,
) {
	p.mu.Lock()
	p.profiles[ip] = profile
	p.mu.Unlock()

	logging.GetLogger().InfoContext(ctx,
		"Profiled device",
		"ip", ip,
		"open_ports", len(profile.OpenPorts),
		"type", profile.DeviceType,
		"icons", profile.DeviceIcons,
	)
}

// profileDevice performs the actual profiling.
func (p *DeviceProfiler) profileDevice(ip string) {
	defer func() {
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
	}()

	stopCh := p.checkShutdown()
	if stopCh == nil {
		return
	}

	ctx, _, cleanup := p.createProfilingContext(stopCh)
	defer cleanup()

	profile := &DeviceProfile{
		ProfiledAt:  time.Now(),
		OpenPorts:   p.scanPorts(ctx, ip),
		DeviceIcons: []string{},
	}

	profile.HTTPInfo = p.probeHTTPFromOpenPorts(ctx, ip, profile.OpenPorts)
	profile.SNMPInfo = p.probeSNMPAndLog(ctx, ip)

	// Collect full SNMP MIB data if SNMP collector is available
	p.collectFullSNMPData(ctx, ip)

	// Perform name resolution (DNS, NetBIOS, mDNS)
	p.resolveDeviceNames(ctx, ip)

	p.inferDeviceType(profile)
	p.storeProfileAndLog(ctx, ip, profile)
}

// collectFullSNMPData performs full SNMP MIB collection for a device.
// This is called automatically when SNMP credentials are configured.
func (p *DeviceProfiler) collectFullSNMPData(ctx context.Context, ip string) {
	if p.snmpCollector == nil {
		return
	}

	logging.GetLogger().DebugContext(ctx, "Starting full SNMP MIB collection", "ip", ip)

	data, err := p.snmpCollector.Collect(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "SNMP MIB collection failed", "ip", ip, "error", err)
		return
	}

	// Store the SNMP data
	p.mu.Lock()
	p.snmpData[ip] = data
	p.mu.Unlock()

	// Log summary of what was collected
	ifCount := len(data.Interfaces)
	macCount := len(data.MACTable)
	vlanCount := len(data.VLANs)
	lldpCount := len(data.LLDPNeighbors)
	errCount := len(data.Errors)

	if ifCount > 0 || macCount > 0 || vlanCount > 0 || lldpCount > 0 {
		logging.GetLogger().InfoContext(ctx, "SNMP MIB data collected",
			"ip", ip,
			"interfaces", ifCount,
			"macs", macCount,
			"vlans", vlanCount,
			"lldp", lldpCount,
			"errors", errCount)
	} else if errCount > 0 {
		logging.GetLogger().DebugContext(ctx, "SNMP MIB collection had errors",
			"ip", ip,
			"errors", data.Errors)
	}
}

// resolveDeviceNames performs DNS, NetBIOS, and mDNS name resolution for a device.
func (p *DeviceProfiler) resolveDeviceNames(ctx context.Context, ip string) {
	if !p.config.EnableNameResolution {
		return
	}

	names := &ResolvedNames{}
	resolved := false

	// DNS PTR lookup
	if p.config.ResolveDNS {
		if hostname := p.resolveDNS(ctx, ip); hostname != "" {
			names.Hostname = hostname
			resolved = true
		}
	}

	// NetBIOS name query
	if p.config.ResolveNetBIOS && p.netbiosResolver != nil {
		if nbName := p.resolveNetBIOS(ctx, ip); nbName != "" {
			names.NetBIOSName = nbName
			resolved = true
		}
	}

	// mDNS name query
	if p.config.ResolveMDNS && p.mdnsResolver != nil {
		if mdnsName := p.resolveMDNS(ctx, ip); mdnsName != "" {
			names.MDNSName = mdnsName
			resolved = true
		}
	}

	// Only store if we resolved at least one name
	if resolved {
		p.mu.Lock()
		p.resolvedNames[ip] = names
		p.mu.Unlock()

		logging.GetLogger().DebugContext(ctx, "Name resolution completed",
			"ip", ip,
			"hostname", names.Hostname,
			"netbios", names.NetBIOSName,
			"mdns", names.MDNSName)
	}
}

// resolveDNS performs reverse DNS lookup for an IP.
func (p *DeviceProfiler) resolveDNS(ctx context.Context, ip string) string {
	timeout := p.config.NameResolutionTimeout
	if timeout == 0 {
		timeout = profilerNameResolveTimeMs * time.Millisecond
	}

	lookupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(lookupCtx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "DNS lookup failed", "ip", ip, "error", err)
		return ""
	}

	if len(names) > 0 {
		hostname := strings.TrimSuffix(names[0], ".")
		logging.GetLogger().DebugContext(ctx, "DNS resolved", "ip", ip, "hostname", hostname)
		return hostname
	}
	return ""
}

// resolveNetBIOS performs NetBIOS name query for an IP.
func (p *DeviceProfiler) resolveNetBIOS(ctx context.Context, ip string) string {
	if p.netbiosResolver == nil {
		return ""
	}

	name, err := p.netbiosResolver.ResolveIP(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "NetBIOS lookup failed", "ip", ip, "error", err)
		return ""
	}
	if name != "" {
		logging.GetLogger().DebugContext(ctx, "NetBIOS resolved", "ip", ip, "name", name)
	}
	return name
}

// resolveMDNS performs mDNS name query for an IP.
func (p *DeviceProfiler) resolveMDNS(ctx context.Context, ip string) string {
	if p.mdnsResolver == nil {
		return ""
	}

	name, err := p.mdnsResolver.ResolveIP(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "mDNS lookup failed", "ip", ip, "error", err)
		return ""
	}
	if name != "" {
		logging.GetLogger().DebugContext(ctx, "mDNS resolved", "ip", ip, "name", name)
	}
	return name
}

// GetResolvedNames returns the resolved names for an IP address.
func (p *DeviceProfiler) GetResolvedNames(ip string) *ResolvedNames {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.resolvedNames[ip]
}

// SetInterface sets the interface name for mDNS resolver.
func (p *DeviceProfiler) SetInterface(interfaceName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interfaceName = interfaceName
	if p.mdnsResolver != nil {
		p.mdnsResolver = NewMDNSResolver(interfaceName)
	}
}

// ClearResolvedNames removes all stored resolved names.
func (p *DeviceProfiler) ClearResolvedNames() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resolvedNames = make(map[string]*ResolvedNames)
}

// checkPortWithConfig checks if a TCP port is open using configurable settings.
// Uses ConnectTimeout from config and respects BannerGrab setting.
func (p *DeviceProfiler) checkPortWithConfig(ctx context.Context, ip string, port int) OpenPort {
	result := OpenPort{
		Port:     port,
		Protocol: "tcp",
		Service:  portToService(port),
		IsOpen:   false,
	}

	// Use configured connect timeout
	timeout := p.config.ConnectTimeout
	if timeout == 0 {
		timeout = p.config.Timeout
	}

	address := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: timeout}

	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return result
	}
	defer func() { _ = conn.Close() }()

	result.IsOpen = true

	// Only grab banner if enabled in config
	if p.config.BannerGrab {
		// Fixes #902: Check context before attempting banner read to avoid blocking on cancelled context
		select {
		case <-ctx.Done():
			return result
		default:
		}
		// Try to grab banner for certain ports that typically send banners
		if port == portSSHProf || port == portFTP || port == portTelnet || port == portSMTP ||
			port == portPOP3 ||
			port == portIMAP ||
			port == portMySQL ||
			port == portPostgreSQL ||
			port == portRedis ||
			port == portMongoDB {
			_ = conn.SetReadDeadline(time.Now().Add(profilerBannerReadMs * time.Millisecond))
			banner := make([]byte, profilerBannerBufferSize)
			n, _ := conn.Read(banner)
			if n > 0 {
				result.Banner = strings.TrimSpace(string(banner[:n]))
			}
		}
	}

	return result
}

// probeHTTP probes an HTTP/HTTPS endpoint.
func (p *DeviceProfiler) probeHTTP(
	ctx context.Context,
	ip string,
	port int,
	isHTTPS bool,
) *HTTPInfo {
	scheme := "http"
	if isHTTPS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s:%d/", scheme, ip, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "The Seed/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	info := &HTTPInfo{
		Port:       port,
		StatusCode: resp.StatusCode,
		Server:     resp.Header.Get("Server"),
		IsHTTPS:    isHTTPS,
	}

	// Read limited body to extract title
	body, err := io.ReadAll(io.LimitReader(resp.Body, profilerHTTPBodyLimit))
	if err == nil {
		info.Title = extractHTMLTitle(string(body))
	}

	return info
}

// probeSNMP attempts to retrieve SNMP information from the device.
func (p *DeviceProfiler) probeSNMP(ctx context.Context, ip string) *SNMPInfo {
	if p.snmpConfig == nil {
		logging.GetLogger().DebugContext(ctx, "SNMP probe skipped - no SNMP config", "ip", ip)
		return nil
	}
	if len(p.snmpConfig.Communities) == 0 && len(p.snmpConfig.V3Credentials) == 0 {
		logging.GetLogger().DebugContext(
			ctx,
			"SNMP probe skipped - no communities or v3 credentials configured",
			"ip", ip,
		)
		return nil
	}

	logging.GetLogger().DebugContext(ctx,
		"Attempting SNMP probe",
		"ip",
		ip,
		"communities",
		len(p.snmpConfig.Communities),
		"v3creds",
		len(p.snmpConfig.V3Credentials),
	)

	// Query system information
	sysInfo, err := snmp.GetSystemInfo(ctx, ip, p.snmpConfig)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "SNMP probe failed", "ip", ip, "error", err)
		return nil
	}

	logging.GetLogger().InfoContext(ctx,
		"SNMP probe succeeded",
		"ip",
		ip,
		"sysName",
		sysInfo.SysName,
		"sysDescr",
		truncateString(sysInfo.SysDescr, profilerLogTruncateLen),
	)

	return &SNMPInfo{
		SysDescr:    sysInfo.SysDescr,
		SysName:     sysInfo.SysName,
		SysContact:  sysInfo.SysContact,
		SysLocation: sysInfo.SysLocation,
	}
}

// truncateString truncates a string to maxLen with ellipsis.
// Fixes #982: Guard against maxLen < 3 to prevent negative slice index panic.
func truncateString(s string, maxLen int) string {
	if maxLen < profilerMinTruncateLen {
		// Can't fit ellipsis, just truncate to maxLen (or return empty for 0 or negative)
		if maxLen <= 0 {
			return ""
		}
		if len(s) <= maxLen {
			return s
		}
		return s[:maxLen]
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// extractHTMLTitle extracts the <title> from HTML content.
func extractHTMLTitle(html string) string {
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Truncate long titles
		if len(title) > profilerTitleMaxLen {
			title = title[:profilerTitleMaxLen] + "..."
		}
		return title
	}
	return ""
}

// inferDeviceType infers device type and icons from the profile.
func (p *DeviceProfiler) inferDeviceType(profile *DeviceProfile) {
	icons := make(map[string]bool)
	deviceType := "unknown"

	deviceType = p.inferFromPorts(profile.OpenPorts, icons, deviceType)
	deviceType = p.inferFromHTTPInfo(profile.HTTPInfo, icons, deviceType)

	for icon := range icons {
		profile.DeviceIcons = append(profile.DeviceIcons, icon)
	}

	if deviceType == "unknown" && len(profile.OpenPorts) > 0 {
		deviceType = "host"
	}

	profile.DeviceType = deviceType
}

// inferFromPorts infers device type and icons from open ports.
func (p *DeviceProfiler) inferFromPorts(
	ports []OpenPort,
	icons map[string]bool,
	deviceType string,
) string {
	for _, op := range ports {
		deviceType = p.setIconsForPort(op.Port, icons, deviceType)
		deviceType = p.inferFromBanner(op.Banner, deviceType)
	}
	return deviceType
}

// setIconsForPort sets icons based on port number and returns updated device type.
func (p *DeviceProfiler) setIconsForPort(
	port int,
	icons map[string]bool,
	deviceType string,
) string {
	switch port {
	case portSSHProf:
		icons["ssh"] = true
	case portTelnet:
		icons["telnet"] = true
	case portHTTPProf, portHTTPAltP:
		icons["web"] = true
	case portHTTPSProf, portHTTPSAltP:
		icons["web-secure"] = true
	case portFTP:
		icons["ftp"] = true
	case portSMTP, portSMTPSubmit:
		icons["mail"] = true
	case portDNS:
		icons["dns"] = true
	case portSNMP:
		icons["snmp"] = true
	case portMySQL, portPostgreSQL:
		icons["database"] = true
	case portRedis:
		icons["cache"] = true
	case portJetDirect, portLPD, portIPP:
		icons["printer"] = true
		deviceType = deviceTypePrinter
	}
	return deviceType
}

// inferFromBanner infers device type from service banner.
func (p *DeviceProfiler) inferFromBanner(banner, deviceType string) string {
	bannerLower := strings.ToLower(banner)
	if !strings.Contains(bannerLower, "ssh") {
		return deviceType
	}
	if strings.Contains(bannerLower, "cisco") {
		return deviceTypeNetworkDevice
	}
	if strings.Contains(bannerLower, "ubuntu") || strings.Contains(bannerLower, "debian") {
		return deviceTypeServer
	}
	return deviceType
}

// inferFromHTTPInfo infers device type and icons from HTTP response.
func (p *DeviceProfiler) inferFromHTTPInfo(
	httpInfo *HTTPInfo,
	icons map[string]bool,
	deviceType string,
) string {
	if httpInfo == nil {
		return deviceType
	}

	titleLower := strings.ToLower(httpInfo.Title)
	serverLower := strings.ToLower(httpInfo.Server)

	if t, icon := matchHTTPDeviceType(titleLower, serverLower); t != "" {
		icons[icon] = true
		return t
	}

	return deviceType
}

// httpDeviceMatch defines a pattern match for HTTP-based device detection.
type httpDeviceMatch struct {
	titlePatterns  []string
	serverPatterns []string
	deviceType     string
	icon           string
}

// getHTTPDeviceMatchers returns the patterns for HTTP-based device detection.
func getHTTPDeviceMatchers() []httpDeviceMatch {
	return []httpDeviceMatch{
		{[]string{"router"}, []string{"router"}, deviceTypeRouter, "router"},
		{[]string{"switch"}, nil, deviceTypeSwitch, "switch"},
		{
			[]string{"firewall", "pfsense", "opnsense", "fortinet"},
			nil,
			deviceTypeFirewall,
			"firewall",
		},
		{[]string{"nas", "synology", "qnap"}, nil, deviceTypeNAS, "storage"},
		{[]string{"printer", "hp ", "canon", "epson"}, nil, deviceTypePrinter, "printer"},
		{nil, []string{"apache", "nginx"}, deviceTypeServer, "server"},
	}
}

// matchHTTPDeviceType matches HTTP title/server against known device patterns.
func matchHTTPDeviceType(title, server string) (string, string) {
	for _, m := range getHTTPDeviceMatchers() {
		for _, p := range m.titlePatterns {
			if strings.Contains(title, p) {
				return m.deviceType, m.icon
			}
		}
		for _, p := range m.serverPatterns {
			if strings.Contains(server, p) {
				return m.deviceType, m.icon
			}
		}
	}
	return "", ""
}

// GetProfile returns the profile for an IP address.
func (p *DeviceProfiler) GetProfile(ip string) *DeviceProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.profiles[ip]
}

// GetAllProfiles returns all collected profiles.
func (p *DeviceProfiler) GetAllProfiles() map[string]*DeviceProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*DeviceProfile, len(p.profiles))
	maps.Copy(result, p.profiles)
	return result
}

// ClearProfiles removes all stored profiles.
func (p *DeviceProfiler) ClearProfiles() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profiles = make(map[string]*DeviceProfile)
}

// IsProfiled returns true if the IP has been profiled.
func (p *DeviceProfiler) IsProfiled(ip string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.profiles[ip]
	return exists
}

// IsProfiling returns true if the IP is currently being profiled.
func (p *DeviceProfiler) IsProfiling(ip string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.profiling[ip]
}

// hasSNMPCredentials returns true if SNMP credentials are configured.
func (p *DeviceProfiler) hasSNMPCredentials() bool {
	if p.snmpConfig == nil {
		return false
	}
	return len(p.snmpConfig.Communities) > 0 || len(p.snmpConfig.V3Credentials) > 0
}

// GetSNMPData returns the full SNMP MIB data for an IP address.
func (p *DeviceProfiler) GetSNMPData(ip string) *SNMPFullData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.snmpData[ip]
}

// GetAllSNMPData returns all collected SNMP MIB data.
func (p *DeviceProfiler) GetAllSNMPData() map[string]*SNMPFullData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*SNMPFullData, len(p.snmpData))
	maps.Copy(result, p.snmpData)
	return result
}

// ClearSNMPData removes all stored SNMP data.
func (p *DeviceProfiler) ClearSNMPData() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snmpData = make(map[string]*SNMPFullData)
}

// ProfilingStatus represents the current state of the device profiler.
type ProfilingStatus struct {
	TotalProfiled int      `json:"totalProfiled"` // Number of devices successfully profiled
	InProgress    int      `json:"inProgress"`    // Number of devices currently being profiled
	QueueLength   int      `json:"queueLength"`   // Number of devices waiting to be profiled
	ProfilingIPs  []string `json:"profilingIps"`  // IPs currently being profiled
	Enabled       bool     `json:"enabled"`       // Whether profiling is enabled
	MaxConcurrent int      `json:"maxConcurrent"` // Maximum concurrent profiling operations
	PortsToScan   int      `json:"portsToScan"`   // Number of ports being scanned per device
	ScanIntensity string   `json:"scanIntensity"` // Current port scan intensity level
}

// GetProfilingStatus returns comprehensive status about profiling operations.
func (p *DeviceProfiler) GetProfilingStatus() *ProfilingStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profilingIPs := make([]string, 0, len(p.profiling))
	for ip := range p.profiling {
		profilingIPs = append(profilingIPs, ip)
	}

	// Get port count based on intensity
	ports := p.config.GetPortsForIntensity()
	portsCount := len(ports)
	if ports == nil {
		portsCount = len(p.config.QuickPorts) // Fallback to quick ports
	}

	return &ProfilingStatus{
		TotalProfiled: len(p.profiles),
		InProgress:    len(p.profiling),
		QueueLength:   len(p.queue),
		ProfilingIPs:  profilingIPs,
		Enabled:       p.config.Enabled,
		MaxConcurrent: p.config.MaxConcurrent,
		PortsToScan:   portsCount,
		ScanIntensity: string(p.config.PortScanIntensity),
	}
}

// GetDeviceProfilingState returns the profiling state for a specific device IP.
func (p *DeviceProfiler) GetDeviceProfilingState(ip string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.profiling[ip] {
		return "profiling"
	}
	if _, exists := p.profiles[ip]; exists {
		return "completed"
	}
	return "pending"
}

// portToService maps common ports to service names.
func portToService(port int) string {
	services := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		161:   "snmp",
		443:   "https",
		445:   "smb",
		515:   "lpd",
		587:   "submission",
		631:   "ipp",
		993:   "imaps",
		995:   "pop3s",
		3306:  "mysql",
		3389:  "rdp",
		5432:  "postgresql",
		5900:  "vnc",
		6379:  "redis",
		8080:  "http-alt",
		8443:  "https-alt",
		9100:  "jetdirect",
		27017: "mongodb",
	}

	if s, ok := services[port]; ok {
		return s
	}
	return ""
}
