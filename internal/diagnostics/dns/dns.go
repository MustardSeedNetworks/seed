package dns

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// Status represents the status of a DNS operation.
type Status string

// DNS operation status constants.
const (
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

// DNS timeout bounds for validation.
const (
	MinDNSTimeout = 100 * time.Millisecond
	MaxDNSTimeout = 30 * time.Second
)

// Default threshold values for DNS response times.
const (
	DefaultWarningThresholdMs  = 100 // milliseconds - response times above this trigger warning
	DefaultCriticalThresholdMs = 500 // milliseconds - response times above this trigger critical
)

// DefaultDialerTimeout is the default timeout for DNS dialer connections.
const DefaultDialerTimeout = 5 * time.Second

// ValidateDNSTimeout validates that a DNS timeout is within acceptable bounds.
func ValidateDNSTimeout(timeout time.Duration) error {
	if timeout < MinDNSTimeout || timeout > MaxDNSTimeout {
		return &TimeoutError{timeout, MinDNSTimeout, MaxDNSTimeout}
	}
	return nil
}

// TimeoutError represents an invalid timeout configuration.
type TimeoutError struct {
	Value time.Duration
	Min   time.Duration
	Max   time.Duration
}

func (e *TimeoutError) Error() string {
	return "DNS timeout " + e.Value.String() + " must be between " + e.Min.String() + " and " + e.Max.String()
}

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
	Servers          []string            `json:"servers"`     // All configured DNS servers
	ServerScope      Scope               `json:"serverScope"` // Whose resolvers Servers describes
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
		Warning:  DefaultWarningThresholdMs * time.Millisecond,
		Critical: DefaultCriticalThresholdMs * time.Millisecond,
	}
}

// ConfiguredServer represents a user-configured DNS server.
type ConfiguredServer struct {
	Address string
	Enabled bool
}

// Tester performs DNS tests with timing.
type Tester struct {
	server            string
	testHostname      string
	thresholds        Thresholds
	resolver          *net.Resolver
	configuredServers []ConfiguredServer
	// iface is the interface the operator selected. The resolvers shown and
	// tested are scoped to it (#2690); empty means no selection, and the
	// answer stays system-wide.
	iface     string
	resolvers resolverSource
	mu        sync.RWMutex
}

// NewTester creates a new DNS tester.
func NewTester(server, testHostname string, thresholds Thresholds) *Tester {
	t := &Tester{
		server:       server,
		testHostname: testHostname,
		thresholds:   thresholds,
		resolvers:    systemResolvers(),
		resolver:     resolverForServer(server),
	}

	return t
}

// NewTesterForInterface creates a tester whose resolvers are scoped to iface,
// the way the gateway tester takes the active interface at construction. An
// empty name keeps the system-wide answer, which is right when nothing has
// been selected yet.
func NewTesterForInterface(server, testHostname string, thresholds Thresholds, iface string) *Tester {
	t := NewTester(server, testHostname, thresholds)
	t.SetInterface(iface)
	return t
}

// resolverForServer returns a resolver that dials server, or the system
// resolver when no server is named.
func resolverForServer(server string) *net.Resolver {
	if server == "" {
		return net.DefaultResolver
	}
	// Capture the address in the closure so a later SetServer cannot race it.
	serverAddr := server + ":53"
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: DefaultDialerTimeout}
			return d.DialContext(ctx, "udp", serverAddr)
		},
	}
}

// SetInterface scopes the resolvers to the interface the operator selected.
func (t *Tester) SetInterface(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.iface = name
}

// GetInterface returns the interface the resolvers are scoped to.
func (t *Tester) GetInterface() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.iface
}

// SetTestHostname updates the hostname used for testing.
func (t *Tester) SetTestHostname(hostname string) {
	t.mu.Lock()
	t.testHostname = hostname
	t.mu.Unlock()
}

// SetServer updates the DNS server to use.
// Fixes #859: Lock held throughout entire resolver update via defer.
func (t *Tester) SetServer(server string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.server = server
	t.resolver = resolverForServer(server)
}

// SetConfiguredServers updates the list of user-configured DNS servers.
func (t *Tester) SetConfiguredServers(servers []ConfiguredServer) {
	t.mu.Lock()
	t.configuredServers = servers
	t.mu.Unlock()
}

// getStatus determines status based on timing and thresholds.
func (t *Tester) getStatus(duration time.Duration, hasError bool) Status {
	t.mu.RLock()
	th := t.thresholds
	t.mu.RUnlock()
	return t.getStatusWith(th, duration, hasError)
}

// ForwardLookup performs a forward DNS lookup (hostname to IP) with timing.
func (t *Tester) ForwardLookup(ctx context.Context, hostname string) *LookupResult {
	t.mu.RLock()
	if hostname == "" {
		hostname = t.testHostname
	}
	resolver := t.resolver
	thresholds := t.thresholds
	t.mu.RUnlock()

	if hostname == "" {
		hostname = t.testHostname
	}

	start := time.Now()
	addrs, err := resolver.LookupHost(ctx, hostname)
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
	result.Status = t.getStatusWith(thresholds, elapsed, false)

	return result
}

func (t *Tester) getStatusWith(th Thresholds, duration time.Duration, hasError bool) Status {
	if hasError {
		return StatusError
	}
	if duration >= th.Critical {
		return StatusError
	}
	if duration >= th.Warning {
		return StatusWarning
	}
	return StatusSuccess
}

// forwardLookupIP performs a forward DNS lookup for the specified network type.
// network should be "ip4" for A records or "ip6" for AAAA records.
func (t *Tester) forwardLookupIP(
	ctx context.Context,
	hostname, network, noRecordMsg string,
) *LookupResult {
	t.mu.RLock()
	if hostname == "" {
		hostname = t.testHostname
	}
	resolver := t.resolver
	thresholds := t.thresholds
	t.mu.RUnlock()

	start := time.Now()
	addrs, err := resolver.LookupIP(ctx, network, hostname)
	elapsed := time.Since(start)

	result := &LookupResult{
		Time:   elapsed,
		TimeMs: elapsed.Milliseconds(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Result = noRecordMsg
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
		result.Result = noRecordMsg
		result.Status = StatusWarning
		return result
	}
	result.Status = t.getStatusWith(thresholds, elapsed, false)

	return result
}

// ForwardLookupIPv4 performs an IPv4-only forward DNS lookup (A record).
func (t *Tester) ForwardLookupIPv4(ctx context.Context, hostname string) *LookupResult {
	return t.forwardLookupIP(ctx, hostname, "ip4", "No A record")
}

// ForwardLookupIPv6 performs an IPv6-only forward DNS lookup (AAAA record).
func (t *Tester) ForwardLookupIPv6(ctx context.Context, hostname string) *LookupResult {
	return t.forwardLookupIP(ctx, hostname, "ip6", "No AAAA record")
}

// ReverseLookup performs a reverse DNS lookup (IP to hostname) with timing.
func (t *Tester) ReverseLookup(ctx context.Context, ip string) *LookupResult {
	t.mu.RLock()
	resolver := t.resolver
	thresholds := t.thresholds
	t.mu.RUnlock()

	start := time.Now()
	names, err := resolver.LookupAddr(ctx, ip)
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
	result.Status = t.getStatusWith(thresholds, elapsed, false)

	return result
}

// TestServer performs a DNS test against a specific server.
func (t *Tester) TestServer(ctx context.Context, server string) *ServerTestResult {
	// Create a temporary tester for this specific server
	t.mu.RLock()
	host := t.testHostname
	th := t.thresholds
	t.mu.RUnlock()

	tempTester := NewTester(server, host, th)

	result := &ServerTestResult{
		Server: server,
	}

	// IPv4 forward lookup (A record)
	result.Forward = tempTester.ForwardLookupIPv4(ctx, host)

	// IPv6 forward lookup (AAAA record)
	result.ForwardIPv6 = tempTester.ForwardLookupIPv6(ctx, host)

	// Calculate overall status and average time
	var totalTime int64
	var count int64
	hasError := false
	hasWarning := false

	if result.Forward != nil {
		totalTime += result.Forward.TimeMs
		count++
		switch result.Forward.Status {
		case StatusError:
			hasError = true
		case StatusWarning:
			hasWarning = true
		case StatusSuccess:
			// Success is the default, no action needed
		}
	}
	if result.ForwardIPv6 != nil {
		totalTime += result.ForwardIPv6.TimeMs
		count++
		switch result.ForwardIPv6.Status {
		case StatusError:
			hasError = true
		case StatusWarning:
			hasWarning = true
		case StatusSuccess:
			// Success is the default, no action needed
		}
	}

	if count > 0 {
		result.AvgTimeMs = totalTime / count
	}

	switch {
	case hasError:
		result.Status = StatusError
	case hasWarning:
		result.Status = StatusWarning
	default:
		result.Status = StatusSuccess
	}

	return result
}

// Test performs a complete DNS test (forward and reverse) for both IPv4 and IPv6.
func (t *Tester) Test(ctx context.Context) *TestResult {
	t.mu.RLock()
	cfgServers := append([]ConfiguredServer(nil), t.configuredServers...)
	host := t.testHostname
	selectedServer := t.server
	source := t.resolvers
	iface := t.iface
	thresholds := t.thresholds
	t.mu.RUnlock()

	servers, scope := source.resolversFor(iface)

	// Add enabled configured servers to the list (avoiding duplicates). They
	// are an explicit operator choice, so they are offered whatever the scope
	// is, and they do not turn an interface-scoped answer into a host-wide one.
	serverSet := make(map[string]bool)
	for _, s := range servers {
		serverSet[s] = true
	}
	for _, cs := range cfgServers {
		if cs.Enabled && cs.Address != "" && !serverSet[cs.Address] {
			servers = append(servers, cs.Address)
			serverSet[cs.Address] = true
		}
	}

	result := &TestResult{
		Server:       selectedServer,
		TestHostname: host,
		Servers:      servers,
		ServerScope:  scope,
	}

	// An interface with no resolvers of its own is not measured through
	// another link's. The system resolver would answer — over whichever
	// interface carries the route — and the card would report a success this
	// link did not earn, which is the defect the scoping exists to remove.
	if scope == ScopeInterface && len(servers) == 0 {
		return result
	}

	// Without an explicitly selected server the lookups would go through the
	// system resolver, so a scoped list would be displayed and a host-wide
	// resolver measured. Dial the interface's own first resolver instead.
	lookups := t
	if selectedServer == "" && scope == ScopeInterface {
		lookups = NewTester(servers[0], host, thresholds)
		result.Server = servers[0]
	} else if selectedServer == "" {
		result.Server = "System Default"
	}

	// IPv4 forward lookup (A record)
	result.Forward = lookups.ForwardLookupIPv4(ctx, host)

	// IPv6 forward lookup (AAAA record)
	result.ForwardIPv6 = lookups.ForwardLookupIPv6(ctx, host)

	// Reverse lookup on the first IPv4 result
	if result.Forward.Status != StatusError && len(result.Forward.Resolved) > 0 {
		result.Reverse = lookups.ReverseLookup(ctx, result.Forward.Resolved[0])
	}

	// Reverse lookup on the first IPv6 result
	if result.ForwardIPv6.Status != StatusError && len(result.ForwardIPv6.Resolved) > 0 {
		result.ReverseIPv6 = lookups.ReverseLookup(ctx, result.ForwardIPv6.Resolved[0])
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
// Implementation is platform-specific (dns_linux.go, dns_darwin.go).
func GetSystemDNS() []string {
	return getSystemDNSPlatform()
}
