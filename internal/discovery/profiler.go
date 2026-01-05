// Package discovery implements multi-protocol network device discovery.
// Device profiler module performs deep inspection of discovered devices through HTTP,
// SNMP, mDNS, and port scanning to gather detailed information about capabilities,
// services, and device types. Enables intelligent device identification and visualization hints.
package discovery

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
}

// DefaultProfilerConfig returns sensible defaults.
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:       true,
		Timeout:       2 * time.Second,
		MaxConcurrent: 10,
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
		ProbeDelay:        50 * time.Millisecond,
		HostDelay:         20 * time.Millisecond,
		ConnectTimeout:    2 * time.Second,
		SkipTLSVerify:     false, // Set to true for internal network devices with self-signed certs
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

// DeviceProfiler automatically profiles newly discovered devices.
type DeviceProfiler struct {
	config     *ProfilerConfig
	snmpConfig *config.SNMPConfig
	httpClient *http.Client
	transport  *http.Transport // Store transport for cleanup (fixes #825)
	mu         sync.RWMutex
	profiles   map[string]*DeviceProfile // key by IP
	profiling  map[string]bool           // track in-progress profiles
	queue      chan string               // IPs to profile
	stopCh     chan struct{}
	wg         sync.WaitGroup
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

	return &DeviceProfiler{
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
		profiles:  make(map[string]*DeviceProfile),
		profiling: make(map[string]bool),
		queue:     make(chan string, 100),
	}
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
	logging.GetLogger().Info("Updated profiler scan config", "intensity", intensity, "timing", timing)
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

// profileDevice performs the actual profiling.
func (p *DeviceProfiler) profileDevice(ip string) {
	defer func() {
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
	}()

	// Get stopCh under lock to check for shutdown (fixes #828)
	p.mu.RLock()
	stopCh := p.stopCh
	p.mu.RUnlock()

	// Check if we're shutting down before starting work
	if stopCh == nil {
		return
	}
	select {
	case <-stopCh:
		return
	default:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create cancellable context that respects shutdown (fixes #828)
	ctx, cancelWithShutdown := context.WithCancel(ctx)
	go func() {
		select {
		case <-stopCh:
			cancelWithShutdown()
		case <-ctx.Done():
		}
	}()
	defer cancelWithShutdown()

	profile := &DeviceProfile{
		ProfiledAt:  time.Now(),
		OpenPorts:   []OpenPort{},
		DeviceIcons: []string{},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Determine which ports to scan based on intensity
	portsToScan := p.config.GetPortsForIntensity()
	if len(portsToScan) == 0 {
		// Fall back to QuickPorts if intensity is off but profiler is enabled
		portsToScan = p.config.QuickPorts
	}

	// Rate limiting semaphore for IDS-friendly scanning
	sem := make(chan struct{}, p.config.MaxConcurrent)

	// Check ports with rate limiting based on timing profile
	for _, port := range portsToScan {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			// Check for context cancellation before acquiring semaphore (fixes #834)
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			// Apply probe delay for IDS-friendly scanning with context check (fixes #834)
			if p.config.ProbeDelay > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(p.config.ProbeDelay):
				}
			}

			result := p.checkPortWithConfig(ctx, ip, port)
			if result.IsOpen {
				mu.Lock()
				profile.OpenPorts = append(profile.OpenPorts, result)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	// Check HTTP/HTTPS if ports are open
	for _, op := range profile.OpenPorts {
		if op.Port == 80 || op.Port == 8080 {
			if info := p.probeHTTP(ctx, ip, op.Port, false); info != nil {
				profile.HTTPInfo = info
				break
			}
		}
		if op.Port == 443 || op.Port == 8443 {
			if info := p.probeHTTP(ctx, ip, op.Port, true); info != nil {
				profile.HTTPInfo = info
				break
			}
		}
	}

	// Always try SNMP probing - SNMP uses UDP port 161, not TCP,
	// so we can't detect it via TCP port scanning. Just try to query.
	if info := p.probeSNMP(ctx, ip); info != nil {
		profile.SNMPInfo = info
		logging.GetLogger().DebugContext(ctx, "Got SNMP info from device", "ip", ip, "sysName", info.SysName)
	}

	// Infer device type and icons from profile
	p.inferDeviceType(profile)

	p.mu.Lock()
	p.profiles[ip] = profile
	p.mu.Unlock()

	logging.GetLogger().InfoContext(ctx,
		"Profiled device",
		"ip",
		ip,
		"open_ports",
		len(profile.OpenPorts),
		"type",
		profile.DeviceType,
		"icons",
		profile.DeviceIcons,
	)
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
		if port == 22 || port == 21 || port == 23 || port == 25 || port == 110 || port == 143 ||
			port == 3306 || port == 5432 || port == 6379 || port == 27017 {
			_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			banner := make([]byte, 256)
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
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
		logging.GetLogger().DebugContext(ctx, "SNMP probe skipped - no communities or v3 credentials configured", "ip", ip)
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
		truncateString(sysInfo.SysDescr, 50),
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
	if maxLen < 3 {
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
		if len(title) > 100 {
			title = title[:100] + "..."
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
	case 22:
		icons["ssh"] = true
	case 23:
		icons["telnet"] = true
	case 80, 8080:
		icons["web"] = true
	case 443, 8443:
		icons["web-secure"] = true
	case 21:
		icons["ftp"] = true
	case 25, 587:
		icons["mail"] = true
	case 53:
		icons["dns"] = true
	case 161:
		icons["snmp"] = true
	case 3306, 5432:
		icons["database"] = true
	case 6379:
		icons["cache"] = true
	case 9100, 515, 631:
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
		{[]string{"firewall", "pfsense", "opnsense", "fortinet"}, nil, deviceTypeFirewall, "firewall"},
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
