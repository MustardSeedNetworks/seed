// Package discovery provides device fingerprinting and OS/service detection capabilities.
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
package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"regexp"
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
		timeout = 3 * time.Second
	}
	return &Fingerprinter{timeout: timeout}
}

// ProbeDevice performs advanced probing on a device.
func (f *Fingerprinter) ProbeDevice(ctx context.Context, ip string, profile *DeviceProfile) *AdvancedProbeResult {
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
	if profile != nil {
		for _, port := range profile.OpenPorts {
			if sv := f.detectServiceVersion(port); sv != nil {
				result.ServiceVersions = append(result.ServiceVersions, *sv)
			}
		}
	}

	// TLS certificate inspection for HTTPS ports
	tlsPorts := []int{443, 8443, 8080}
	if profile != nil {
		for _, port := range profile.OpenPorts {
			if port.Port == 443 || port.Port == 8443 || strings.HasSuffix(port.Service, "s") {
				if !containsInt(tlsPorts, port.Port) {
					tlsPorts = append(tlsPorts, port.Port)
				}
			}
		}
	}

	for _, port := range tlsPorts {
		if tlsInfo := f.probeTLS(ctx, ip, port); tlsInfo != nil {
			result.TLSInfo = append(result.TLSInfo, *tlsInfo)
		}
	}

	return result
}

// fingerprintOS attempts to identify the operating system.
func (f *Fingerprinter) fingerprintOS(_ context.Context, _ string, profile *DeviceProfile) *OSFingerprint {
	fp := &OSFingerprint{
		Methods: []string{},
	}

	// Note: TTL-based detection requires raw sockets which aren't available
	// through Go's standard net package. We rely on banner and HTTP analysis instead.

	// Method 1: Banner analysis
	if profile != nil {
		for _, port := range profile.OpenPorts {
			if port.Banner != "" {
				bannerLower := strings.ToLower(port.Banner)
				if osInfo := f.parseOSFromBanner(bannerLower); osInfo != nil {
					fp.Methods = append(fp.Methods, "banner")
					// Banner is more reliable than TTL
					if osInfo.Confidence > fp.Confidence {
						fp.OSFamily = osInfo.OSFamily
						fp.OSVersion = osInfo.OSVersion
						fp.Confidence = osInfo.Confidence
					}
				}
			}
		}

		// Method 3: HTTP Server header analysis
		if profile.HTTPInfo != nil && profile.HTTPInfo.Server != "" {
			serverLower := strings.ToLower(profile.HTTPInfo.Server)
			if osInfo := f.parseOSFromServer(serverLower); osInfo != nil {
				fp.Methods = append(fp.Methods, "http")
				if osInfo.Confidence > fp.Confidence {
					fp.OSFamily = osInfo.OSFamily
					fp.OSVersion = osInfo.OSVersion
					fp.Confidence = osInfo.Confidence
				}
			}
		}
	}

	if fp.OSFamily == "" {
		return nil
	}

	return fp
}

// osMatch defines a pattern for OS detection.
type osMatch struct {
	patterns   []string // all patterns must match
	osFamily   string
	osVersion  string
	confidence int
}

// sshOSMatchers defines OS patterns for SSH banners.
var sshOSMatchers = []osMatch{
	{[]string{"ubuntu"}, osLinux, "ubuntu", 90},
	{[]string{"debian"}, osLinux, "debian", 90},
	{[]string{"centos"}, osLinux, "rhel", 90},
	{[]string{"red hat"}, osLinux, "rhel", 90},
	{[]string{"freebsd"}, "bsd", "freebsd", 90},
	{[]string{"cisco"}, osCisco, "", 95},
	{[]string{"windows"}, osWindows, "", 90},
	{[]string{"openssh"}, "unix", "", 50},
}

// genericOSMatchers defines OS patterns for generic banners.
var genericOSMatchers = []osMatch{
	{[]string{"linux"}, osLinux, "", 80},
	{[]string{"windows"}, osWindows, "", 80},
	{[]string{"cisco"}, osCisco, "", 95},
	{[]string{"junos"}, "juniper", "", 95},
	{[]string{"vsftpd"}, osLinux, "", 75},
	{[]string{"proftpd"}, osLinux, "", 75},
	{[]string{"microsoft", "ftp"}, osWindows, "", 85},
}

// parseOSFromBanner extracts OS info from service banners.
func (f *Fingerprinter) parseOSFromBanner(banner string) *OSFingerprint {
	fp := &OSFingerprint{}

	if strings.Contains(banner, "ssh") {
		f.matchOSPatterns(banner, sshOSMatchers, fp)
	}
	if fp.OSFamily == "" {
		f.matchOSPatterns(banner, genericOSMatchers, fp)
	}

	if fp.OSFamily == "" {
		return nil
	}
	return fp
}

// matchOSPatterns checks banner against patterns and sets fingerprint if matched.
func (*Fingerprinter) matchOSPatterns(banner string, matchers []osMatch, fp *OSFingerprint) {
	for _, m := range matchers {
		matched := true
		for _, p := range m.patterns {
			if !strings.Contains(banner, p) {
				matched = false
				break
			}
		}
		if matched {
			fp.OSFamily = m.osFamily
			fp.OSVersion = m.osVersion
			fp.Confidence = m.confidence
			return
		}
	}
}

// serverOSMatchers defines OS patterns for HTTP Server headers.
var serverOSMatchers = []osMatch{
	{[]string{"ubuntu"}, osLinux, "ubuntu", 85},
	{[]string{"debian"}, osLinux, "debian", 85},
	{[]string{"centos"}, osLinux, "rhel", 85},
	{[]string{"red hat"}, osLinux, "rhel", 85},
	{[]string{"cisco"}, osCisco, "", 90},
	{[]string{"routeros"}, "mikrotik", "", 95}, //nolint:misspell // RouterOS is MikroTik's product name
	{[]string{"fortinet"}, "fortinet", "", 95},
	{[]string{"fortigate"}, "fortinet", "", 95},
	{[]string{"pfsense"}, "bsd", "firewall", 90},
	{[]string{"opnsense"}, "bsd", "firewall", 90},
	{[]string{"synology"}, osLinux, "dsm", 95},
	{[]string{"qnap"}, osLinux, "qts", 95},
}

// parseOSFromServer extracts OS info from HTTP Server header.
func (f *Fingerprinter) parseOSFromServer(server string) *OSFingerprint {
	fp := &OSFingerprint{}

	// Windows indicators (special case for IIS version extraction)
	if strings.Contains(server, "microsoft") || strings.Contains(server, "iis") {
		fp.OSFamily = osWindows
		if match := regexp.MustCompile(`iis[/\s]*([\d.]+)`).FindStringSubmatch(server); len(match) > 1 {
			fp.OSVersion = "IIS " + match[1]
		}
		fp.Confidence = 85
		return fp
	}

	// Try pattern matching
	f.matchOSPatterns(server, serverOSMatchers, fp)

	// Fallback for generic web servers
	if fp.OSFamily == "" && (strings.Contains(server, "lighttpd") || strings.Contains(server, "nginx")) {
		fp.OSFamily = "unix"
		fp.Confidence = 50
	}

	if fp.OSFamily == "" {
		return nil
	}
	return fp
}

// detectServiceVersion analyzes a port's banner to determine service version.
func (f *Fingerprinter) detectServiceVersion(port OpenPort) *ServiceVersion {
	if port.Banner == "" && port.Service == "" {
		return nil
	}

	sv := &ServiceVersion{
		Port:       port.Port,
		Service:    port.Service,
		Confidence: 50,
	}

	if port.Banner == "" {
		return sv
	}

	bannerLower := strings.ToLower(port.Banner)
	f.detectSSHVersion(port.Port, bannerLower, sv)
	f.detectFTPVersion(port.Port, bannerLower, sv)
	f.detectSMTPVersion(port.Port, bannerLower, sv)
	f.detectTelnetVersion(port.Port, bannerLower, sv)

	return sv
}

// detectSSHVersion detects SSH service version from banner.
func (*Fingerprinter) detectSSHVersion(port int, banner string, sv *ServiceVersion) {
	if port != 22 && !strings.Contains(banner, "ssh") {
		return
	}
	sv.Service = "ssh"
	if match := regexp.MustCompile(`openssh[_\s]*([\d.p]+)`).FindStringSubmatch(banner); len(match) > 1 {
		sv.Product = "OpenSSH"
		sv.Version = match[1]
		sv.Confidence = 95
	} else if match := regexp.MustCompile(`ssh-([\d.]+)`).FindStringSubmatch(banner); len(match) > 1 {
		sv.Product = "SSH"
		sv.Version = match[1]
		sv.Confidence = 80
	}
}

// detectFTPVersion detects FTP service version from banner.
func (*Fingerprinter) detectFTPVersion(port int, banner string, sv *ServiceVersion) {
	if port != 21 && !strings.HasPrefix(banner, "220") {
		return
	}
	sv.Service = "ftp"
	switch {
	case strings.Contains(banner, "vsftpd"):
		sv.Product = "vsftpd"
		if match := regexp.MustCompile(`vsftpd\s*([\d.]+)`).FindStringSubmatch(banner); len(match) > 1 {
			sv.Version = match[1]
		}
		sv.Confidence = 90
	case strings.Contains(banner, "proftpd"):
		sv.Product = "ProFTPD"
		if match := regexp.MustCompile(`proftpd\s*([\d.]+)`).FindStringSubmatch(banner); len(match) > 1 {
			sv.Version = match[1]
		}
		sv.Confidence = 90
	case strings.Contains(banner, "pure-ftpd"):
		sv.Product = "Pure-FTPd"
		sv.Confidence = 90
	case strings.Contains(banner, "microsoft"):
		sv.Product = "Microsoft FTP"
		sv.Confidence = 85
	}
}

// detectSMTPVersion detects SMTP service version from banner.
func (*Fingerprinter) detectSMTPVersion(port int, banner string, sv *ServiceVersion) {
	if port != 25 && port != 587 {
		return
	}
	sv.Service = "smtp"
	switch {
	case strings.Contains(banner, "postfix"):
		sv.Product = "Postfix"
		sv.Confidence = 90
	case strings.Contains(banner, "sendmail"):
		sv.Product = "Sendmail"
		sv.Confidence = 90
	case strings.Contains(banner, "exim"):
		sv.Product = "Exim"
		if match := regexp.MustCompile(`exim\s*([\d.]+)`).FindStringSubmatch(banner); len(match) > 1 {
			sv.Version = match[1]
		}
		sv.Confidence = 90
	case strings.Contains(banner, "microsoft"):
		sv.Product = "Microsoft Exchange"
		sv.Confidence = 85
	}
}

// detectTelnetVersion detects Telnet service version from banner.
func (*Fingerprinter) detectTelnetVersion(port int, banner string, sv *ServiceVersion) {
	if port != 23 {
		return
	}
	sv.Service = "telnet"
	switch {
	case strings.Contains(banner, "cisco"):
		sv.Product = "Cisco IOS"
		sv.Confidence = 95
	case strings.Contains(banner, "linux"):
		sv.Product = "Linux telnetd"
		sv.Confidence = 80
	}
}

// probeTLS probes a port for TLS certificate information.
func (f *Fingerprinter) probeTLS(_ context.Context, ip string, port int) *TLSInfo {
	addr := fmt.Sprintf("%s:%d", ip, port)

	dialer := &net.Dialer{Timeout: f.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 -- We want to inspect any certificate
	})
	if err != nil {
		return nil
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil
	}

	cert := state.PeerCertificates[0]

	info := &TLSInfo{
		Port:              port,
		Version:           tlsVersionString(state.Version),
		CipherSuite:       tls.CipherSuiteName(state.CipherSuite),
		CommonName:        cert.Subject.CommonName,
		ValidFrom:         cert.NotBefore,
		ValidTo:           cert.NotAfter,
		DaysUntilExpiry:   int(time.Until(cert.NotAfter).Hours() / 24),
		SelfSigned:        cert.Issuer.CommonName == cert.Subject.CommonName,
		SubjectAltNames:   cert.DNSNames,
		CertificateErrors: []string{},
	}

	// Build issuer string
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	} else {
		info.Issuer = cert.Issuer.CommonName
	}

	// Check for common certificate issues
	now := time.Now()
	if now.Before(cert.NotBefore) {
		info.CertificateErrors = append(info.CertificateErrors, "not yet valid")
	}
	if now.After(cert.NotAfter) {
		info.CertificateErrors = append(info.CertificateErrors, "expired")
	}
	if info.SelfSigned {
		info.CertificateErrors = append(info.CertificateErrors, "self-signed")
	}
	if info.DaysUntilExpiry > 0 && info.DaysUntilExpiry < 30 {
		info.CertificateErrors = append(info.CertificateErrors, "expiring soon")
	}

	return info
}

// tlsVersionString converts TLS version constant to string.
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// containsInt checks if an int slice contains a value.
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// quickScan performs a fast port scan to create a minimal DeviceProfile.
// This is used when no existing profile is available.
func (f *Fingerprinter) quickScan(ctx context.Context, ip string) *DeviceProfile {
	profile := &DeviceProfile{
		ProfiledAt: time.Now(),
		OpenPorts:  []OpenPort{},
	}

	// Common ports to scan
	ports := []struct {
		port    int
		service string
	}{
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

	// Scan ports concurrently
	type scanResult struct {
		port    int
		service string
		open    bool
		banner  string
	}

	results := make(chan scanResult, len(ports))
	var wg sync.WaitGroup

	for _, p := range ports {
		wg.Add(1)
		go func(port int, service string) {
			defer wg.Done()
			result := scanResult{port: port, service: service}

			addr := fmt.Sprintf("%s:%d", ip, port)
			d := net.Dialer{Timeout: f.timeout}

			conn, err := d.DialContext(ctx, "tcp", addr)
			if err != nil {
				results <- result
				return
			}
			defer conn.Close()

			result.open = true

			// Try to grab banner with short timeout
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:errcheck // Best-effort deadline
			buf := make([]byte, 1024)

			// For HTTP ports, send a request first
			if port == 80 || port == 8080 {
				_, _ = conn.Write([]byte("HEAD / HTTP/1.0\r\nHost: " + ip + "\r\n\r\n")) //nolint:errcheck // Best-effort request
			}

			n, err := conn.Read(buf)
			if err == nil && n > 0 {
				result.banner = strings.TrimSpace(string(buf[:n]))
				// Truncate long banners
				if len(result.banner) > 256 {
					result.banner = result.banner[:256]
				}
			}

			results <- result
		}(p.port, p.service)
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		if result.open {
			profile.OpenPorts = append(profile.OpenPorts, OpenPort{
				Port:    result.port,
				Service: result.service,
				Banner:  result.banner,
			})
		}
	}

	// Try to get HTTP server header if port 80 or 8080 is open
	for _, op := range profile.OpenPorts {
		if op.Port == 80 || op.Port == 8080 {
			if httpInfo := f.getHTTPInfo(ctx, ip, op.Port); httpInfo != nil {
				profile.HTTPInfo = httpInfo
				break
			}
		}
	}

	return profile
}

// getHTTPInfo fetches HTTP server information.
func (f *Fingerprinter) getHTTPInfo(ctx context.Context, ip string, port int) *HTTPInfo {
	addr := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: f.timeout}

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil
	}
	defer conn.Close()

	// Send HTTP request
	request := fmt.Sprintf("HEAD / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", ip)
	_ = conn.SetWriteDeadline(time.Now().Add(f.timeout)) //nolint:errcheck // Best-effort deadline
	_, err = conn.Write([]byte(request))
	if err != nil {
		return nil
	}

	// Read response
	_ = conn.SetReadDeadline(time.Now().Add(f.timeout)) //nolint:errcheck // Best-effort deadline
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return nil
	}

	response := string(buf[:n])
	info := &HTTPInfo{
		Port: port,
	}

	// Parse headers
	lines := strings.Split(response, "\r\n")
	for _, line := range lines {
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
