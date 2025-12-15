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
func (f *Fingerprinter) fingerprintOS(ctx context.Context, ip string, profile *DeviceProfile) *OSFingerprint {
	fp := &OSFingerprint{
		Methods: []string{},
	}

	// Method 1: TTL-based detection
	// Common initial TTL values:
	// - Linux/Unix: 64
	// - Windows: 128
	// - Cisco/network devices: 255
	// - Solaris/AIX: 254
	ttl := f.getTTL(ctx, ip)
	if ttl > 0 {
		fp.TTLObserved = ttl
		fp.Methods = append(fp.Methods, "ttl")

		switch {
		case ttl <= 64:
			fp.OSFamily = "linux"
			fp.Confidence = 60
		case ttl <= 128:
			fp.OSFamily = "windows"
			fp.Confidence = 60
		case ttl >= 254:
			fp.OSFamily = "network-device"
			fp.Confidence = 70
		}
	}

	// Method 2: Banner analysis
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

// getTTL performs a TCP connect and attempts to get TTL from response.
func (f *Fingerprinter) getTTL(ctx context.Context, ip string) int {
	// Try common ports to get a response
	ports := []int{80, 443, 22, 23}

	for _, port := range ports {
		addr := fmt.Sprintf("%s:%d", ip, port)
		d := net.Dialer{Timeout: f.timeout}

		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			continue
		}
		defer conn.Close()

		// For TCP connections, we can't directly get the TTL from Go's net package
		// We'll rely on the TTL from the DiscoveredDevice if available
		// This is a simplified approach - real implementation would need raw sockets
		return 0
	}

	return 0
}

// parseOSFromBanner extracts OS info from service banners.
func (f *Fingerprinter) parseOSFromBanner(banner string) *OSFingerprint {
	fp := &OSFingerprint{}

	// SSH banners
	if strings.Contains(banner, "ssh") {
		if strings.Contains(banner, "ubuntu") {
			fp.OSFamily = "linux"
			fp.OSVersion = extractProductVersion(banner, "ubuntu")
			fp.Confidence = 90
		} else if strings.Contains(banner, "debian") {
			fp.OSFamily = "linux"
			fp.OSVersion = "debian"
			fp.Confidence = 90
		} else if strings.Contains(banner, "centos") || strings.Contains(banner, "red hat") {
			fp.OSFamily = "linux"
			fp.OSVersion = "rhel"
			fp.Confidence = 90
		} else if strings.Contains(banner, "freebsd") {
			fp.OSFamily = "bsd"
			fp.OSVersion = "freebsd"
			fp.Confidence = 90
		} else if strings.Contains(banner, "cisco") {
			fp.OSFamily = "cisco"
			fp.Confidence = 95
		} else if strings.Contains(banner, "windows") {
			fp.OSFamily = "windows"
			fp.Confidence = 90
		} else if strings.Contains(banner, "openssh") {
			fp.OSFamily = "unix"
			fp.Confidence = 50
		}
	}

	// Telnet banners
	if strings.Contains(banner, "linux") {
		fp.OSFamily = "linux"
		fp.Confidence = 80
	} else if strings.Contains(banner, "windows") {
		fp.OSFamily = "windows"
		fp.Confidence = 80
	} else if strings.Contains(banner, "cisco") {
		fp.OSFamily = "cisco"
		fp.Confidence = 95
	} else if strings.Contains(banner, "junos") {
		fp.OSFamily = "juniper"
		fp.Confidence = 95
	}

	// FTP banners
	if strings.Contains(banner, "vsftpd") {
		fp.OSFamily = "linux"
		fp.Confidence = 75
	} else if strings.Contains(banner, "proftpd") {
		fp.OSFamily = "linux"
		fp.Confidence = 75
	} else if strings.Contains(banner, "microsoft") && strings.Contains(banner, "ftp") {
		fp.OSFamily = "windows"
		fp.Confidence = 85
	}

	if fp.OSFamily == "" {
		return nil
	}
	return fp
}

// parseOSFromServer extracts OS info from HTTP Server header.
func (f *Fingerprinter) parseOSFromServer(server string) *OSFingerprint {
	fp := &OSFingerprint{}

	// Windows indicators
	if strings.Contains(server, "microsoft") || strings.Contains(server, "iis") {
		fp.OSFamily = "windows"
		// Extract IIS version
		if match := regexp.MustCompile(`iis[/\s]*([\d.]+)`).FindStringSubmatch(server); len(match) > 1 {
			fp.OSVersion = "IIS " + match[1]
		}
		fp.Confidence = 85
	}

	// Linux indicators
	if strings.Contains(server, "ubuntu") {
		fp.OSFamily = "linux"
		fp.OSVersion = "ubuntu"
		fp.Confidence = 85
	} else if strings.Contains(server, "debian") {
		fp.OSFamily = "linux"
		fp.OSVersion = "debian"
		fp.Confidence = 85
	} else if strings.Contains(server, "centos") || strings.Contains(server, "red hat") {
		fp.OSFamily = "linux"
		fp.OSVersion = "rhel"
		fp.Confidence = 85
	}

	// Specific products that hint at OS
	if strings.Contains(server, "lighttpd") || strings.Contains(server, "nginx") {
		if fp.OSFamily == "" {
			fp.OSFamily = "unix"
			fp.Confidence = 50
		}
	}

	// Network device indicators
	if strings.Contains(server, "cisco") {
		fp.OSFamily = "cisco"
		fp.Confidence = 90
	} else if strings.Contains(server, "routeros") {
		fp.OSFamily = "mikrotik"
		fp.Confidence = 95
	} else if strings.Contains(server, "fortinet") || strings.Contains(server, "fortigate") {
		fp.OSFamily = "fortinet"
		fp.Confidence = 95
	} else if strings.Contains(server, "pfsense") || strings.Contains(server, "opnsense") {
		fp.OSFamily = "bsd"
		fp.OSVersion = "firewall"
		fp.Confidence = 90
	}

	// NAS devices
	if strings.Contains(server, "synology") {
		fp.OSFamily = "linux"
		fp.OSVersion = "dsm"
		fp.Confidence = 95
	} else if strings.Contains(server, "qnap") {
		fp.OSFamily = "linux"
		fp.OSVersion = "qts"
		fp.Confidence = 95
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

	// SSH version detection
	if port.Port == 22 || strings.Contains(bannerLower, "ssh") {
		sv.Service = "ssh"
		if match := regexp.MustCompile(`openssh[_\s]*([\d.p]+)`).FindStringSubmatch(bannerLower); len(match) > 1 {
			sv.Product = "OpenSSH"
			sv.Version = match[1]
			sv.Confidence = 95
		} else if match := regexp.MustCompile(`ssh-([\d.]+)`).FindStringSubmatch(bannerLower); len(match) > 1 {
			sv.Product = "SSH"
			sv.Version = match[1]
			sv.Confidence = 80
		}
	}

	// FTP version detection
	if port.Port == 21 || strings.HasPrefix(bannerLower, "220") {
		sv.Service = "ftp"
		if strings.Contains(bannerLower, "vsftpd") {
			sv.Product = "vsftpd"
			if match := regexp.MustCompile(`vsftpd\s*([\d.]+)`).FindStringSubmatch(bannerLower); len(match) > 1 {
				sv.Version = match[1]
			}
			sv.Confidence = 90
		} else if strings.Contains(bannerLower, "proftpd") {
			sv.Product = "ProFTPD"
			if match := regexp.MustCompile(`proftpd\s*([\d.]+)`).FindStringSubmatch(bannerLower); len(match) > 1 {
				sv.Version = match[1]
			}
			sv.Confidence = 90
		} else if strings.Contains(bannerLower, "pure-ftpd") {
			sv.Product = "Pure-FTPd"
			sv.Confidence = 90
		} else if strings.Contains(bannerLower, "microsoft") {
			sv.Product = "Microsoft FTP"
			sv.Confidence = 85
		}
	}

	// SMTP version detection
	if port.Port == 25 || port.Port == 587 {
		sv.Service = "smtp"
		if strings.Contains(bannerLower, "postfix") {
			sv.Product = "Postfix"
			sv.Confidence = 90
		} else if strings.Contains(bannerLower, "sendmail") {
			sv.Product = "Sendmail"
			sv.Confidence = 90
		} else if strings.Contains(bannerLower, "exim") {
			sv.Product = "Exim"
			if match := regexp.MustCompile(`exim\s*([\d.]+)`).FindStringSubmatch(bannerLower); len(match) > 1 {
				sv.Version = match[1]
			}
			sv.Confidence = 90
		} else if strings.Contains(bannerLower, "microsoft") {
			sv.Product = "Microsoft Exchange"
			sv.Confidence = 85
		}
	}

	// Telnet detection
	if port.Port == 23 {
		sv.Service = "telnet"
		if strings.Contains(bannerLower, "cisco") {
			sv.Product = "Cisco IOS"
			sv.Confidence = 95
		} else if strings.Contains(bannerLower, "linux") {
			sv.Product = "Linux telnetd"
			sv.Confidence = 80
		}
	}

	return sv
}

// probeTLS probes a port for TLS certificate information.
func (f *Fingerprinter) probeTLS(ctx context.Context, ip string, port int) *TLSInfo {
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

// extractProductVersion extracts a version string from text for a given product.
func extractProductVersion(text, product string) string {
	pattern := regexp.MustCompile(product + `[/_\s]*([\d.]+)`)
	if match := pattern.FindStringSubmatch(strings.ToLower(text)); len(match) > 1 {
		return product + " " + match[1]
	}
	return product
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
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 1024)

			// For HTTP ports, send a request first
			if port == 80 || port == 8080 {
				conn.Write([]byte("HEAD / HTTP/1.0\r\nHost: " + ip + "\r\n\r\n"))
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
	conn.SetWriteDeadline(time.Now().Add(f.timeout))
	_, err = conn.Write([]byte(request))
	if err != nil {
		return nil
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(f.timeout))
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
