// Package discovery implements multi-protocol network device discovery.
// Port scanning support enables detection of open services and their versions on discovered devices.
// Performs banner grabbing to identify service types and versions, mapping active services
// and potential vulnerabilities based on detected ports and versions.
package discovery

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// Error messages and common values for port scanning.
const (
	errNoIPv4ForTarget = "no IPv4 address found for target"
	serviceUnknown     = "unknown"
)

// ServiceInfo contains information about a detected service.
type ServiceInfo struct {
	Port     int       `json:"port"`
	State    PortState `json:"state"`
	Service  string    `json:"service"`            // Service name (http, ssh, etc.)
	Banner   string    `json:"banner,omitempty"`   // Raw banner text
	Version  string    `json:"version,omitempty"`  // Parsed version if available
	Protocol string    `json:"protocol,omitempty"` // tcp or udp
}

// PortScanResult contains the complete result of a port scan.
type PortScanResult struct {
	IP       string        `json:"ip"`
	Hostname string        `json:"hostname,omitempty"`
	Services []ServiceInfo `json:"services"`
	ScanTime time.Duration `json:"scanTime"`
	Error    string        `json:"error,omitempty"`
}

// PortScanner provides port scanning with service detection.
type PortScanner struct {
	prober        *TCPProber
	bannerTimeout time.Duration
	maxBannerLen  int
}

// NewPortScanner creates a new port scanner.
func NewPortScanner(timeout time.Duration) (*PortScanner, error) {
	prober, err := NewTCPProber(timeout)
	if err != nil {
		return nil, err
	}
	return &PortScanner{
		prober:        prober,
		bannerTimeout: 2 * time.Second,
		maxBannerLen:  512,
	}, nil
}

// Close closes the port scanner.
func (s *PortScanner) Close() error {
	return s.prober.Close()
}

// ScanWithBanners performs a port scan with service banner detection.
func (s *PortScanner) ScanWithBanners(
	ctx context.Context,
	target string,
	ports []int,
	workers int,
) *PortScanResult {
	start := time.Now()
	result := &PortScanResult{
		IP:       target,
		Services: make([]ServiceInfo, 0),
	}

	// Resolve hostname
	if ip := net.ParseIP(target); ip == nil {
		// Target is a hostname, resolve it with context
		resolver := &net.Resolver{}
		ips, err := resolver.LookupIP(ctx, "ip", target)
		if err != nil {
			result.Error = fmt.Sprintf("failed to resolve target: %v", err)
			return result
		}
		for _, ip := range ips {
			if ip4 := ip.To4(); ip4 != nil {
				result.IP = ip4.String()
				result.Hostname = target
				break
			}
		}
		if result.IP == target {
			result.Error = errNoIPv4ForTarget
			return result
		}
	}

	// Scan ports
	probeResults := s.prober.ScanPorts(ctx, result.IP, ports, workers)

	// For open ports, try to grab banners
	// Fixes #879: Check context cancellation before processing each probe
	for _, probe := range probeResults {
		select {
		case <-ctx.Done():
			result.Error = ctx.Err().Error()
			return result
		default:
		}
		if probe.State != PortOpen {
			continue
		}

		service := ServiceInfo{
			Port:     probe.Port,
			State:    probe.State,
			Protocol: "tcp",
			Service:  identifyServiceByPort(probe.Port),
		}

		// Try to grab banner for open ports
		banner, version := s.grabBanner(ctx, result.IP, probe.Port)
		if banner != "" {
			service.Banner = banner
			if version != "" {
				service.Version = version
			}
			// Try to identify service from banner
			if detected := identifyServiceFromBanner(banner); detected != "" {
				service.Service = detected
			}
		}

		result.Services = append(result.Services, service)
	}

	result.ScanTime = time.Since(start)
	return result
}

// grabBanner attempts to read a banner from an open port.
func (s *PortScanner) grabBanner(ctx context.Context, ip string, port int) (string, string) {
	ctx, cancel := context.WithTimeout(ctx, s.bannerTimeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return "", ""
	}
	defer func() { _ = conn.Close() }()

	// Set read deadline
	if deadlineErr := conn.SetReadDeadline(time.Now().Add(s.bannerTimeout)); deadlineErr != nil {
		return "", ""
	}

	// For HTTP ports, send a request
	switch {
	case isHTTPPort(port):
		_, _ = fmt.Fprintf(conn, "HEAD / HTTP/1.0\r\nHost: %s\r\n\r\n", ip)
	case port == 25 || port == 587: // SMTP sends banner on connect
	case port == 22: // SSH sends banner on connect
	}

	// Read response
	reader := bufio.NewReader(conn)
	var sb strings.Builder
	for sb.Len() < s.maxBannerLen {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			break
		}
		sb.WriteString(line)
		// Stop after first few lines for most protocols
		if strings.Count(sb.String(), "\n") >= 5 {
			break
		}
	}

	banner := strings.TrimSpace(sb.String())
	version := extractVersion(banner)
	return banner, version
}

// identifyServiceByPort returns the service name based on well-known port.
func identifyServiceByPort(port int) string {
	services := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		111:   "rpcbind",
		135:   "msrpc",
		139:   "netbios-ssn",
		143:   "imap",
		443:   "https",
		445:   "microsoft-ds",
		465:   "smtps",
		587:   "submission",
		993:   "imaps",
		995:   "pop3s",
		1433:  "mssql",
		1521:  "oracle",
		1723:  "pptp",
		3306:  "mysql",
		3389:  "ms-wbt-server",
		5432:  "postgresql",
		5900:  "vnc",
		6379:  "redis",
		8080:  "http-proxy",
		8443:  "https-alt",
		27017: "mongodb",
	}
	if svc, ok := services[port]; ok {
		return svc
	}
	return serviceUnknown
}

// identifyServiceFromBanner tries to identify the service from its banner.
func identifyServiceFromBanner(banner string) string {
	bannerLower := strings.ToLower(banner)

	patterns := []struct {
		pattern string
		service string
	}{
		{"ssh-", "ssh"},
		{"openssh", "ssh"},
		{"http/", "http"},
		{"apache", "http"},
		{"nginx", "http"},
		{"microsoft-iis", "http"},
		{"220 ", "ftp"}, // FTP greeting
		{"smtp", "smtp"},
		{"postfix", "smtp"},
		{"exim", "smtp"},
		{"sendmail", "smtp"},
		{"mysql", "mysql"},
		{"mariadb", "mysql"},
		{"postgresql", "postgresql"},
		{"redis", "redis"},
		{"mongodb", "mongodb"},
		{"imap", "imap"},
		{"pop3", "pop3"},
		{"vnc", "vnc"},
	}

	for _, p := range patterns {
		if strings.Contains(bannerLower, p.pattern) {
			return p.service
		}
	}
	return ""
}

// extractVersion tries to extract a version string from the banner.
func extractVersion(banner string) string {
	// Common version patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`SSH-[\d.]+-OpenSSH_([\d.p]+)`),
		regexp.MustCompile(`Server:\s*([^\r\n]+)`),
		regexp.MustCompile(`Apache/([\d.]+)`),
		regexp.MustCompile(`nginx/([\d.]+)`),
		regexp.MustCompile(`Microsoft-IIS/([\d.]+)`),
		regexp.MustCompile(`(\d+\.\d+\.\d+)`),
	}

	for _, re := range patterns {
		if matches := re.FindStringSubmatch(banner); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// isHTTPPort checks if the port is commonly used for HTTP.
func isHTTPPort(port int) bool {
	httpPorts := map[int]bool{
		80: true, 443: true, 8080: true, 8443: true,
		8000: true, 8888: true, 3000: true, 5000: true,
	}
	return httpPorts[port]
}

// QuickScan performs a quick scan of common ports.
func (s *PortScanner) QuickScan(ctx context.Context, target string) *PortScanResult {
	return s.ScanWithBanners(ctx, target, GetCommonPorts(), 20)
}

// WebScan scans common web ports.
func (s *PortScanner) WebScan(ctx context.Context, target string) *PortScanResult {
	return s.ScanWithBanners(ctx, target, GetWebPorts(), 10)
}

// FullScan scans the top 1000 ports.
func (s *PortScanner) FullScan(ctx context.Context, target string) *PortScanResult {
	ports := make([]int, 1000)
	for i := range ports {
		ports[i] = i + 1
	}
	return s.ScanWithBanners(ctx, target, ports, 50)
}
