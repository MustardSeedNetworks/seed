// Package dns provides DNS testing and lookup functionality with timing.
package dns

import (
	"context"
	"net"
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

// TestResult contains the complete DNS test results.
type TestResult struct {
	Server       string        `json:"server"`
	TestHostname string        `json:"testHostname"`
	Forward      *LookupResult `json:"forward"`
	Reverse      *LookupResult `json:"reverse"`
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

// Test performs a complete DNS test (forward and reverse).
func (t *Tester) Test(ctx context.Context) *TestResult {
	result := &TestResult{
		Server:       t.server,
		TestHostname: t.testHostname,
	}

	if t.server == "" {
		result.Server = "System Default"
	}

	// Forward lookup
	result.Forward = t.ForwardLookup(ctx, t.testHostname)

	// Reverse lookup on the first result
	if result.Forward.Status != StatusError && len(result.Forward.Resolved) > 0 {
		result.Reverse = t.ReverseLookup(ctx, result.Forward.Resolved[0])
	}

	return result
}

// GetSystemDNS attempts to get the system's configured DNS servers.
func GetSystemDNS() []string {
	// This is a simplified version - a full implementation would
	// parse /etc/resolv.conf on Linux or use scutil on macOS
	servers := []string{}

	// Try to resolve a known hostname and observe where the query goes
	// For now, return empty and let the UI show "System Default"
	return servers
}
