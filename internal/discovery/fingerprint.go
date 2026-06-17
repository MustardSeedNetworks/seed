package discovery

//
// This file implements active fingerprinting to identify operating systems, service versions,
// and TLS configurations of discovered network devices. It uses multiple techniques including
// TTL analysis, banner grabbing, HTTP headers, and TLS certificate inspection.
//
// Detection methods:
//   - OS fingerprinting via TCP/IP TTL values and TCP window sizes
//   - Service version detection through banner grabbing on common ports
//   - TLS/SSL certificate analysis and cipher suite detection
//   - HTTP server identification via headers and response patterns
//
// The fingerprinter combines results from multiple methods and assigns confidence scores
// to provide accurate device identification.

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"
)

// OS family constants for fingerprint identification.
const (
	osLinux   = "linux"
	osWindows = "windows"
	osCisco   = "cisco"
)

// Product name constants.
const productCiscoIOS = "Cisco IOS"

// Fingerprinting constants.
const (
	hoursPerDay              = 24   // Hours in a day for certificate expiry calculation
	bannerReadBufferSize     = 1024 // Buffer size for reading service banners
	httpReadBufferSize       = 2048 // Buffer size for reading HTTP responses
	defaultFingerprintTimeS  = 3    // Default timeout in seconds for fingerprinting operations
	telnetPort               = 23   // Standard Telnet port number
	defaultServiceConfidence = 50   // Default confidence score for service detection
)

// OSFingerprint contains OS detection results.
type OSFingerprint struct {
	OSFamily    string   `json:"osFamily,omitempty"`    // linux, windows, bsd, cisco, etc.
	OSVersion   string   `json:"osVersion,omitempty"`   // Specific version if detected
	Confidence  int      `json:"confidence"`            // 0-100 confidence score
	Methods     []string `json:"methods,omitempty"`     // How detected: ttl, banner, http
	TTLObserved int      `json:"ttlObserved,omitempty"` // Observed TTL value
}

// ServiceVersion contains service version detection results.
type ServiceVersion struct {
	Port       int    `json:"port"`
	Service    string `json:"service"`
	Product    string `json:"product,omitempty"`
	Version    string `json:"version,omitempty"`
	ExtraInfo  string `json:"extraInfo,omitempty"`
	Confidence int    `json:"confidence"` // 0-100
}

// TLSInfo contains TLS certificate and configuration details.
type TLSInfo struct {
	Port              int       `json:"port"`
	Version           string    `json:"version"`         // TLS 1.2, TLS 1.3, etc.
	CipherSuite       string    `json:"cipherSuite"`     // Negotiated cipher
	CommonName        string    `json:"commonName"`      // Certificate CN
	Issuer            string    `json:"issuer"`          // Certificate issuer
	ValidFrom         time.Time `json:"validFrom"`       // Certificate start
	ValidTo           time.Time `json:"validTo"`         // Certificate expiry
	DaysUntilExpiry   int       `json:"daysUntilExpiry"` // Days until cert expires
	SelfSigned        bool      `json:"selfSigned"`      // Is self-signed?
	SubjectAltNames   []string  `json:"subjectAltNames"` // SANs
	CertificateErrors []string  `json:"certErrors"`      // Any validation errors
}

// AdvancedProbeResult contains results from advanced probing.
type AdvancedProbeResult struct {
	IP              string           `json:"ip"`
	ProbedAt        time.Time        `json:"probedAt"`
	OSFingerprint   *OSFingerprint   `json:"osFingerprint,omitempty"`
	ServiceVersions []ServiceVersion `json:"serviceVersions,omitempty"`
	TLSInfo         []TLSInfo        `json:"tlsInfo,omitempty"`
}

// Fingerprinter performs advanced probing and fingerprinting.
type Fingerprinter struct {
	timeout time.Duration
}

// NewFingerprinter creates a new fingerprinter.
func NewFingerprinter(timeout time.Duration) *Fingerprinter {
	if timeout == 0 {
		timeout = defaultFingerprintTimeS * time.Second
	}
	return &Fingerprinter{timeout: timeout}
}

// ProbeDevice performs advanced probing on a device.
func (f *Fingerprinter) ProbeDevice(
	ctx context.Context,
	ip string,
	profile *DeviceProfile,
) *AdvancedProbeResult {
	result := &AdvancedProbeResult{
		IP:              ip,
		ProbedAt:        time.Now(),
		ServiceVersions: []ServiceVersion{},
		TLSInfo:         []TLSInfo{},
	}

	// If no profile, do active port scanning first
	if profile == nil {
		profile = f.quickScan(ctx, ip)
	}

	// OS fingerprinting from TTL and banners
	result.OSFingerprint = f.fingerprintOS(ctx, ip, profile)

	// Service version detection from banners
	result.ServiceVersions = f.detectAllServiceVersions(profile)

	// TLS certificate inspection for HTTPS ports
	tlsPorts := f.collectTLSPorts(profile)
	result.TLSInfo = f.probeAllTLS(ctx, ip, tlsPorts)

	return result
}

// detectAllServiceVersions detects service versions for all open ports in a profile.
func (f *Fingerprinter) detectAllServiceVersions(profile *DeviceProfile) []ServiceVersion {
	versions := []ServiceVersion{}
	if profile == nil {
		return versions
	}
	for _, port := range profile.OpenPorts {
		if sv := f.detectServiceVersion(port); sv != nil {
			versions = append(versions, *sv)
		}
	}
	return versions
}

// collectTLSPorts gathers ports that should be probed for TLS certificates.
// Fixes #980: Cap tlsPorts to prevent unbounded growth from malicious profiles.
func (f *Fingerprinter) collectTLSPorts(profile *DeviceProfile) []int {
	const maxTLSPorts = 20
	tlsPorts := []int{443, 8443, 8080}

	if profile == nil {
		return tlsPorts
	}

	for _, port := range profile.OpenPorts {
		if len(tlsPorts) >= maxTLSPorts {
			break
		}
		if f.isTLSPort(port) && !containsInt(tlsPorts, port.Port) {
			tlsPorts = append(tlsPorts, port.Port)
		}
	}
	return tlsPorts
}

// isTLSPort determines if a port should be probed for TLS.
func (*Fingerprinter) isTLSPort(port OpenPort) bool {
	return port.Port == 443 || port.Port == 8443 || strings.HasSuffix(port.Service, "s")
}

// probeAllTLS probes all specified ports for TLS information.
func (f *Fingerprinter) probeAllTLS(ctx context.Context, ip string, ports []int) []TLSInfo {
	tlsInfos := []TLSInfo{}
	for _, port := range ports {
		if tlsInfo := f.probeTLS(ctx, ip, port); tlsInfo != nil {
			tlsInfos = append(tlsInfos, *tlsInfo)
		}
	}
	return tlsInfos
}

// fingerprintOS attempts to identify the operating system.
func (f *Fingerprinter) fingerprintOS(
	_ context.Context,
	_ string,
	profile *DeviceProfile,
) *OSFingerprint {
	fp := &OSFingerprint{
		Methods: []string{},
	}

	if profile == nil {
		return nil
	}

	// Note: TTL-based detection requires raw sockets which aren't available
	// through Go's standard net package. We rely on banner and HTTP analysis instead.

	// Method 1: Banner analysis
	f.fingerprintFromBanners(profile, fp)

	// Method 2: HTTP Server header analysis
	f.fingerprintFromHTTP(profile, fp)

	if fp.OSFamily == "" {
		return nil
	}

	return fp
}

// containsInt checks if an int slice contains a value.
func containsInt(slice []int, val int) bool {
	return slices.Contains(slice, val)
}

// portSpec defines a port and its associated service name.
type portSpec struct {
	port    int
	service string
}

// scanResult holds the result of scanning a single port.
type scanResult struct {
	port    int
	service string
	open    bool
	banner  string
}

// getCommonPorts returns the list of common ports to scan.
func getCommonPorts() []portSpec {
	return []portSpec{
		{22, "ssh"},
		{23, "telnet"},
		{80, "http"},
		{443, "https"},
		{21, "ftp"},
		{25, "smtp"},
		{53, "dns"},
		{110, "pop3"},
		{143, "imap"},
		{3306, "mysql"},
		{5432, "postgresql"},
		{6379, "redis"},
		{8080, "http-alt"},
		{8443, "https-alt"},
		{3389, "rdp"},
		{5900, "vnc"},
	}
}

// quickScan performs a fast port scan to create a minimal DeviceProfile.
// This is used when no existing profile is available.
func (f *Fingerprinter) quickScan(ctx context.Context, ip string) *DeviceProfile {
	profile := &DeviceProfile{
		ProfiledAt: time.Now(),
		OpenPorts:  []OpenPort{},
	}

	ports := getCommonPorts()
	results := f.scanPortsConcurrently(ctx, ip, ports)
	profile.OpenPorts = f.collectOpenPorts(results)
	profile.HTTPInfo = f.findHTTPInfo(ctx, ip, profile.OpenPorts)

	return profile
}

// scanPortsConcurrently scans all ports in parallel and returns results channel.
func (f *Fingerprinter) scanPortsConcurrently(
	ctx context.Context,
	ip string,
	ports []portSpec,
) <-chan scanResult {
	results := make(chan scanResult, len(ports))
	var wg sync.WaitGroup

	for _, p := range ports {
		wg.Add(1)
		go func(port int, service string) {
			defer wg.Done()
			results <- f.scanSinglePort(ctx, ip, port, service)
		}(p.port, p.service)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// scanSinglePort probes a single port and returns the result.
func (f *Fingerprinter) scanSinglePort(
	ctx context.Context,
	ip string,
	port int,
	service string,
) scanResult {
	result := scanResult{port: port, service: service}

	addr := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: f.timeout}

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return result
	}
	defer func() { _ = conn.Close() }()

	result.open = true
	result.banner = f.grabBanner(conn, ip, port)

	return result
}

// grabBanner attempts to read a service banner from a connection.
func (*Fingerprinter) grabBanner(conn net.Conn, ip string, port int) string {
	const bannerTimeout = 2 * time.Second
	const maxBannerLen = 256

	_ = conn.SetReadDeadline(time.Now().Add(bannerTimeout))
	buf := make([]byte, bannerReadBufferSize)

	// For HTTP ports, send a request first
	if port == 80 || port == 8080 {
		_, _ = conn.Write([]byte("HEAD / HTTP/1.0\r\nHost: " + ip + "\r\n\r\n"))
	}

	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return ""
	}

	banner := strings.TrimSpace(string(buf[:n]))
	if len(banner) > maxBannerLen {
		banner = banner[:maxBannerLen]
	}
	return banner
}

// collectOpenPorts converts scan results into OpenPort slice.
func (*Fingerprinter) collectOpenPorts(results <-chan scanResult) []OpenPort {
	openPorts := []OpenPort{}
	for result := range results {
		if result.open {
			openPorts = append(openPorts, OpenPort{
				Port:    result.port,
				Service: result.service,
				Banner:  result.banner,
			})
		}
	}
	return openPorts
}

// findHTTPInfo attempts to get HTTP server info from open HTTP ports.
func (f *Fingerprinter) findHTTPInfo(
	ctx context.Context,
	ip string,
	openPorts []OpenPort,
) *HTTPInfo {
	for _, op := range openPorts {
		if op.Port == 80 || op.Port == 8080 {
			if httpInfo := f.getHTTPInfo(ctx, ip, op.Port); httpInfo != nil {
				return httpInfo
			}
		}
	}
	return nil
}

// getHTTPInfo fetches HTTP server information.
func (f *Fingerprinter) getHTTPInfo(ctx context.Context, ip string, port int) *HTTPInfo {
	addr := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: f.timeout}

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil
	}
	defer func() { _ = conn.Close() }()

	// Send HTTP request
	request := fmt.Sprintf("HEAD / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", ip)
	_ = conn.SetWriteDeadline(time.Now().Add(f.timeout))
	_, err = conn.Write([]byte(request))
	if err != nil {
		return nil
	}

	// Read response
	_ = conn.SetReadDeadline(time.Now().Add(f.timeout))
	buf := make([]byte, httpReadBufferSize)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return nil
	}

	response := string(buf[:n])
	info := &HTTPInfo{
		Port: port,
	}

	// Parse headers
	lines := strings.SplitSeq(response, "\r\n")
	for line := range lines {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "server:") {
			info.Server = strings.TrimSpace(line[7:])
		}
	}

	if info.Server == "" {
		return nil
	}

	return info
}
