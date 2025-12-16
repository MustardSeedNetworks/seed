// Package discovery implements multi-protocol network device discovery.
// Device profiler module performs deep inspection of discovered devices through HTTP,
// SNMP, mDNS, and port scanning to gather detailed information about capabilities,
// services, and device types. Enables intelligent device identification and visualization hints.
package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
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
			161,  // SNMP
			8080, // HTTP Alt
			8443, // HTTPS Alt
		},
	}
}

// DeviceProfiler automatically profiles newly discovered devices.
type DeviceProfiler struct {
	config     *ProfilerConfig
	snmpConfig *config.SNMPConfig
	httpClient *http.Client
	mu         sync.RWMutex
	profiles   map[string]*DeviceProfile // key by IP
	profiling  map[string]bool           // track in-progress profiles
	queue      chan string               // IPs to profile
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewDeviceProfiler creates a new device profiler.
func NewDeviceProfiler(cfg *ProfilerConfig, snmpCfg *config.SNMPConfig) *DeviceProfiler {
	if cfg == nil {
		cfg = DefaultProfilerConfig()
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- Profiling internal network devices
		DialContext: (&net.Dialer{
			Timeout: cfg.Timeout,
		}).DialContext,
	}

	return &DeviceProfiler{
		config:     cfg,
		snmpConfig: snmpCfg,
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

	log.Printf("Device profiler started with %d workers", p.config.MaxConcurrent)
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
	log.Printf("Device profiler stopped")
}

// worker processes profile requests from the queue.
func (p *DeviceProfiler) worker() {
	defer p.wg.Done()

	// Capture stopCh locally to avoid race with Stop() setting it to nil
	p.mu.Lock()
	stopCh := p.stopCh
	p.mu.Unlock()

	if stopCh == nil {
		return
	}

	for {
		select {
		case <-stopCh:
			return
		case ip := <-p.queue:
			p.profileDevice(ip)
		}
	}
}

// QueueProfile adds an IP to the profiling queue.
func (p *DeviceProfiler) QueueProfile(ip string) {
	if !p.config.Enabled || ip == "" {
		return
	}

	p.mu.Lock()
	// Skip if already profiled or in progress
	if _, exists := p.profiles[ip]; exists {
		p.mu.Unlock()
		return
	}
	if p.profiling[ip] {
		p.mu.Unlock()
		return
	}
	p.profiling[ip] = true
	p.mu.Unlock()

	select {
	case p.queue <- ip:
	default:
		// Queue full, skip
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
	}
}

// profileDevice performs the actual profiling.
func (p *DeviceProfiler) profileDevice(ip string) {
	defer func() {
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	profile := &DeviceProfile{
		ProfiledAt:  time.Now(),
		OpenPorts:   []OpenPort{},
		DeviceIcons: []string{},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Check common ports in parallel
	for _, port := range p.config.QuickPorts {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			result := p.checkPort(ctx, ip, port)
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

	// Check SNMP if port 161 is open
	for _, op := range profile.OpenPorts {
		if op.Port == 161 && op.IsOpen {
			if info := p.probeSNMP(ctx, ip); info != nil {
				profile.SNMPInfo = info
			}
			break
		}
	}

	// Infer device type and icons from profile
	p.inferDeviceType(profile)

	p.mu.Lock()
	p.profiles[ip] = profile
	p.mu.Unlock()

	log.Printf("Profiled device %s: %d open ports, type=%s, icons=%v",
		ip, len(profile.OpenPorts), profile.DeviceType, profile.DeviceIcons)
}

// checkPort checks if a TCP port is open.
func (p *DeviceProfiler) checkPort(ctx context.Context, ip string, port int) OpenPort {
	result := OpenPort{
		Port:     port,
		Protocol: "tcp",
		Service:  portToService(port),
		IsOpen:   false,
	}

	address := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: p.config.Timeout}

	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return result
	}
	defer conn.Close()

	result.IsOpen = true

	// Try to grab banner for certain ports
	if port == 22 || port == 21 || port == 23 || port == 25 || port == 110 || port == 143 {
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)) //nolint:errcheck // Best-effort deadline
		banner := make([]byte, 256)
		n, _ := conn.Read(banner) //nolint:errcheck // Best-effort banner read
		if n > 0 {
			result.Banner = strings.TrimSpace(string(banner[:n]))
		}
	}

	return result
}

// probeHTTP probes an HTTP/HTTPS endpoint.
func (p *DeviceProfiler) probeHTTP(ctx context.Context, ip string, port int, isHTTPS bool) *HTTPInfo {
	scheme := "http"
	if isHTTPS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s:%d/", scheme, ip, port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "The Seed/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

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
	if p.snmpConfig == nil || len(p.snmpConfig.Communities) == 0 {
		return nil
	}

	// Query system information
	sysInfo, err := snmp.GetSystemInfo(ctx, ip, p.snmpConfig)
	if err != nil {
		return nil
	}

	return &SNMPInfo{
		SysDescr:    sysInfo.SysDescr,
		SysName:     sysInfo.SysName,
		SysContact:  sysInfo.SysContact,
		SysLocation: sysInfo.SysLocation,
	}
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
func (p *DeviceProfiler) inferFromPorts(ports []OpenPort, icons map[string]bool, deviceType string) string {
	for _, op := range ports {
		deviceType = p.setIconsForPort(op.Port, icons, deviceType)
		deviceType = p.inferFromBanner(op.Banner, deviceType)
	}
	return deviceType
}

// setIconsForPort sets icons based on port number and returns updated device type.
func (p *DeviceProfiler) setIconsForPort(port int, icons map[string]bool, deviceType string) string {
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
func (p *DeviceProfiler) inferFromHTTPInfo(httpInfo *HTTPInfo, icons map[string]bool, deviceType string) string {
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

// httpDeviceMatchers defines patterns for HTTP-based device detection.
var httpDeviceMatchers = []httpDeviceMatch{
	{[]string{"router"}, []string{"router"}, deviceTypeRouter, "router"},
	{[]string{"switch"}, nil, deviceTypeSwitch, "switch"},
	{[]string{"firewall", "pfsense", "opnsense", "fortinet"}, nil, deviceTypeFirewall, "firewall"},
	{[]string{"nas", "synology", "qnap"}, nil, deviceTypeNAS, "storage"},
	{[]string{"printer", "hp ", "canon", "epson"}, nil, deviceTypePrinter, "printer"},
	{nil, []string{"apache", "nginx"}, deviceTypeServer, "server"},
}

// matchHTTPDeviceType matches HTTP title/server against known device patterns.
func matchHTTPDeviceType(title, server string) (deviceType, icon string) {
	for _, m := range httpDeviceMatchers {
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
	for k, v := range p.profiles {
		result[k] = v
	}
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
