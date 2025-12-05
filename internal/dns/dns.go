// Package dns provides DNS testing and lookup functionality with timing.
package dns

import (
	"bufio"
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Status represents the status of a DNS operation.
type Status string

const (
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

// LookupResult contains the result of a DNS lookup with timing.
type LookupResult struct {
	Result   string        `json:"result"`
	Time     time.Duration `json:"time"`
	TimeMs   int64         `json:"timeMs"`
	Status   Status        `json:"status"`
	Error    string        `json:"error,omitempty"`
	Resolved []string      `json:"resolved,omitempty"`
}

// ServerTestResult contains test results for a specific DNS server.
type ServerTestResult struct {
	Server      string        `json:"server"`
	Forward     *LookupResult `json:"forward"`     // IPv4 forward lookup (A record)
	ForwardIPv6 *LookupResult `json:"forwardIpv6"` // IPv6 forward lookup (AAAA record)
	Status      Status        `json:"status"`      // Overall status for this server
	AvgTimeMs   int64         `json:"avgTimeMs"`   // Average response time
}

// TestResult contains the complete DNS test results.
type TestResult struct {
	Server           string              `json:"server"`
	Servers          []string            `json:"servers"`          // All configured DNS servers
	TestHostname     string              `json:"testHostname"`
	Forward          *LookupResult       `json:"forward"`          // IPv4 forward lookup (A record)
	ForwardIPv6      *LookupResult       `json:"forwardIpv6"`      // IPv6 forward lookup (AAAA record)
	Reverse          *LookupResult       `json:"reverse"`          // Reverse lookup (PTR record)
	ReverseIPv6      *LookupResult       `json:"reverseIpv6"`      // IPv6 reverse lookup
	PerServerResults []*ServerTestResult `json:"perServerResults"` // Results for each DNS server
}

// Thresholds defines timing thresholds for DNS lookups.
type Thresholds struct {
	Warning  time.Duration
	Critical time.Duration
}

// DefaultThresholds returns reasonable default thresholds for DNS.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}
}

// Tester performs DNS tests with timing.
type Tester struct {
	server       string
	testHostname string
	thresholds   Thresholds
	resolver     *net.Resolver
}

// NewTester creates a new DNS tester.
func NewTester(server, testHostname string, thresholds Thresholds) *Tester {
	t := &Tester{
		server:       server,
		testHostname: testHostname,
		thresholds:   thresholds,
	}

	// Create custom resolver if server is specified
	if server != "" {
		t.resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second,
				}
				// Use the specified DNS server
				return d.DialContext(ctx, "udp", server+":53")
			},
		}
	} else {
		t.resolver = net.DefaultResolver
	}

	return t
}

// SetTestHostname updates the hostname used for testing.
func (t *Tester) SetTestHostname(hostname string) {
	t.testHostname = hostname
}

// SetServer updates the DNS server to use.
func (t *Tester) SetServer(server string) {
	t.server = server
	if server != "" {
		t.resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second,
				}
				return d.DialContext(ctx, "udp", server+":53")
			},
		}
	} else {
		t.resolver = net.DefaultResolver
	}
}

// getStatus determines status based on timing and thresholds.
func (t *Tester) getStatus(duration time.Duration, hasError bool) Status {
	if hasError {
		return StatusError
	}
	if duration >= t.thresholds.Critical {
		return StatusError
	}
	if duration >= t.thresholds.Warning {
		return StatusWarning
	}
	return StatusSuccess
}

// ForwardLookup performs a forward DNS lookup (hostname to IP) with timing.
func (t *Tester) ForwardLookup(ctx context.Context, hostname string) *LookupResult {
	if hostname == "" {
		hostname = t.testHostname
	}

	start := time.Now()
	addrs, err := t.resolver.LookupHost(ctx, hostname)
	elapsed := time.Since(start)

	result := &LookupResult{
		Time:   elapsed,
		TimeMs: elapsed.Milliseconds(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Result = "Failed"
		result.Status = StatusError
		return result
	}

	if len(addrs) > 0 {
		result.Result = addrs[0]
		result.Resolved = addrs
	} else {
		result.Result = "No results"
	}
	result.Status = t.getStatus(elapsed, false)

	return result
}

// ForwardLookupIPv4 performs an IPv4-only forward DNS lookup (A record).
func (t *Tester) ForwardLookupIPv4(ctx context.Context, hostname string) *LookupResult {
	if hostname == "" {
		hostname = t.testHostname
	}

	start := time.Now()
	addrs, err := t.resolver.LookupIP(ctx, "ip4", hostname)
	elapsed := time.Since(start)

	result := &LookupResult{
		Time:   elapsed,
		TimeMs: elapsed.Milliseconds(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Result = "No A record"
		result.Status = StatusWarning
		return result
	}

	resolved := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		resolved = append(resolved, addr.String())
	}

	if len(resolved) > 0 {
		result.Result = resolved[0]
		result.Resolved = resolved
	} else {
		result.Result = "No A record"
		result.Status = StatusWarning
		return result
	}
	result.Status = t.getStatus(elapsed, false)

	return result
}

// ForwardLookupIPv6 performs an IPv6-only forward DNS lookup (AAAA record).
func (t *Tester) ForwardLookupIPv6(ctx context.Context, hostname string) *LookupResult {
	if hostname == "" {
		hostname = t.testHostname
	}

	start := time.Now()
	addrs, err := t.resolver.LookupIP(ctx, "ip6", hostname)
	elapsed := time.Since(start)

	result := &LookupResult{
		Time:   elapsed,
		TimeMs: elapsed.Milliseconds(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Result = "No AAAA record"
		result.Status = StatusWarning
		return result
	}

	resolved := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		resolved = append(resolved, addr.String())
	}

	if len(resolved) > 0 {
		result.Result = resolved[0]
		result.Resolved = resolved
	} else {
		result.Result = "No AAAA record"
		result.Status = StatusWarning
		return result
	}
	result.Status = t.getStatus(elapsed, false)

	return result
}

// ReverseLookup performs a reverse DNS lookup (IP to hostname) with timing.
func (t *Tester) ReverseLookup(ctx context.Context, ip string) *LookupResult {
	start := time.Now()
	names, err := t.resolver.LookupAddr(ctx, ip)
	elapsed := time.Since(start)

	result := &LookupResult{
		Time:   elapsed,
		TimeMs: elapsed.Milliseconds(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Result = "Failed"
		result.Status = StatusError
		return result
	}

	if len(names) > 0 {
		result.Result = names[0]
		result.Resolved = names
	} else {
		result.Result = "No PTR record"
		result.Status = StatusWarning
		return result
	}
	result.Status = t.getStatus(elapsed, false)

	return result
}

// TestServer performs a DNS test against a specific server.
func (t *Tester) TestServer(ctx context.Context, server string) *ServerTestResult {
	// Create a temporary tester for this specific server
	tempTester := NewTester(server, t.testHostname, t.thresholds)

	result := &ServerTestResult{
		Server: server,
	}

	// IPv4 forward lookup (A record)
	result.Forward = tempTester.ForwardLookupIPv4(ctx, t.testHostname)

	// IPv6 forward lookup (AAAA record)
	result.ForwardIPv6 = tempTester.ForwardLookupIPv6(ctx, t.testHostname)

	// Calculate overall status and average time
	var totalTime int64
	var count int64
	hasError := false
	hasWarning := false

	if result.Forward != nil {
		totalTime += result.Forward.TimeMs
		count++
		if result.Forward.Status == StatusError {
			hasError = true
		} else if result.Forward.Status == StatusWarning {
			hasWarning = true
		}
	}
	if result.ForwardIPv6 != nil {
		totalTime += result.ForwardIPv6.TimeMs
		count++
		if result.ForwardIPv6.Status == StatusError {
			hasError = true
		} else if result.ForwardIPv6.Status == StatusWarning {
			hasWarning = true
		}
	}

	if count > 0 {
		result.AvgTimeMs = totalTime / count
	}

	if hasError {
		result.Status = StatusError
	} else if hasWarning {
		result.Status = StatusWarning
	} else {
		result.Status = StatusSuccess
	}

	return result
}

// Test performs a complete DNS test (forward and reverse) for both IPv4 and IPv6.
func (t *Tester) Test(ctx context.Context) *TestResult {
	servers := GetSystemDNS()
	result := &TestResult{
		Server:       t.server,
		TestHostname: t.testHostname,
		Servers:      servers,
	}

	if t.server == "" {
		result.Server = "System Default"
	}

	// IPv4 forward lookup (A record)
	result.Forward = t.ForwardLookupIPv4(ctx, t.testHostname)

	// IPv6 forward lookup (AAAA record)
	result.ForwardIPv6 = t.ForwardLookupIPv6(ctx, t.testHostname)

	// Reverse lookup on the first IPv4 result
	if result.Forward.Status != StatusError && len(result.Forward.Resolved) > 0 {
		result.Reverse = t.ReverseLookup(ctx, result.Forward.Resolved[0])
	}

	// Reverse lookup on the first IPv6 result
	if result.ForwardIPv6.Status != StatusError && len(result.ForwardIPv6.Resolved) > 0 {
		result.ReverseIPv6 = t.ReverseLookup(ctx, result.ForwardIPv6.Resolved[0])
	}

	// Per-server testing (only for IPv4 servers to avoid duplicate long tests)
	for _, server := range servers {
		// Skip IPv6 servers for per-server testing if they'd be duplicates
		if strings.Contains(server, ":") {
			continue // Skip IPv6 for now, test only IPv4 DNS servers individually
		}
		serverResult := t.TestServer(ctx, server)
		result.PerServerResults = append(result.PerServerResults, serverResult)
	}

	return result
}

// GetSystemDNS attempts to get the system's configured DNS servers.
func GetSystemDNS() []string {
	switch runtime.GOOS {
	case "darwin":
		return getSystemDNSDarwin()
	case "linux":
		return getSystemDNSLinux()
	default:
		return []string{}
	}
}

// getSystemDNSDarwin reads DNS servers from macOS using scutil.
func getSystemDNSDarwin() []string {
	servers := []string{}

	cmd := exec.Command("scutil", "--dns")
	output, err := cmd.Output()
	if err != nil {
		return servers
	}

	// Parse scutil output for nameserver entries
	lines := strings.Split(string(output), "\n")
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver[") {
			// Format: "nameserver[0] : 192.168.1.1"
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				server := strings.TrimSpace(parts[1])
				if server != "" && !seen[server] {
					seen[server] = true
					servers = append(servers, server)
				}
			}
		}
	}

	return servers
}

// getSystemDNSLinux reads DNS servers from /etc/resolv.conf or resolvectl.
func getSystemDNSLinux() []string {
	servers := []string{}

	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return servers
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Parse nameserver lines
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}

	// If only systemd-resolved stub is found, try to get real servers
	if len(servers) == 1 && servers[0] == "127.0.0.53" {
		if realServers := getSystemDNSResolvectl(); len(realServers) > 0 {
			return realServers
		}
	}

	return servers
}

// getSystemDNSResolvectl gets DNS servers from systemd-resolved via resolvectl.
func getSystemDNSResolvectl() []string {
	servers := []string{}
	seen := make(map[string]bool)

	// Try resolvectl first, then fall back to systemd-resolve
	cmd := exec.Command("resolvectl", "status")
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("systemd-resolve", "--status")
		output, err = cmd.Output()
		if err != nil {
			return servers
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "DNS Servers:" lines
		if strings.HasPrefix(line, "DNS Servers:") {
			// Format: "DNS Servers: 192.168.64.1 fe80::1c57:dcff:fea5:2564"
			parts := strings.TrimPrefix(line, "DNS Servers:")
			for _, server := range strings.Fields(parts) {
				server = strings.TrimSpace(server)
				if server != "" && !seen[server] {
					seen[server] = true
					servers = append(servers, server)
				}
			}
		}
	}

	return servers
}
