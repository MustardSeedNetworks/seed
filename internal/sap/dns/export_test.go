package dns

import (
	"net"
	"sync"
	"time"
)

// TesterServer returns the server for testing.
func (t *Tester) TesterServer() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.server
}

// TesterTestHostname returns the test hostname for testing.
func (t *Tester) TesterTestHostname() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.testHostname
}

// TesterResolver returns whether resolver is non-nil for testing.
func (t *Tester) TesterHasResolver() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.resolver != nil
}

// TesterConfiguredServersCount returns the count of configured servers for testing.
func (t *Tester) TesterConfiguredServersCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.configuredServers)
}

// TesterMu exposes the mutex for testing.
func (t *Tester) TesterMu() *sync.RWMutex {
	return &t.mu
}

// GetStatus is exported for testing.
func (t *Tester) GetStatus(duration time.Duration, hasError bool) Status {
	return t.getStatus(duration, hasError)
}

// ExportGetSystemDNSPlatform is exported for testing.
func ExportGetSystemDNSPlatform() []string {
	return getSystemDNSPlatform()
}

// ExportIsPrivateIP is exported for testing.
func ExportIsPrivateIP(ip net.IP) bool {
	return isPrivateIP(ip)
}

// ExportCalculateSeverity is exported for testing.
func ExportCalculateSeverity(result *SecurityScanResult) string {
	scanner := NewSecurityScanner(DefaultSecurityScanConfig())
	return scanner.calculateSeverity(result)
}

// ExportParseResolvConfDarwin is exported for testing (darwin only).
func ExportParseResolvConfDarwin(path string) []string {
	return parseResolvConfDarwin(path)
}

// ExportGetDNSFromInterfaces is exported for testing (darwin only).
func ExportGetDNSFromInterfaces() []string {
	return getDNSFromInterfaces()
}
