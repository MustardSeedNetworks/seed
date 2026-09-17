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

// TestResolverSource carries a resolverSource across the package boundary so
// an external test can build one; the field itself stays unexported.
type TestResolverSource struct{ src resolverSource }

// ExportResolverSource builds a resolverSource from two stubs so the scoping
// rule can be exercised without a host whose resolver configuration happens to
// agree with the case under test.
func ExportResolverSource(system func() []string, forIface func(string) ([]string, bool)) TestResolverSource {
	return TestResolverSource{src: resolverSource{system: system, forIface: forIface}}
}

// ExportResolversFor applies the scoping rule to a source built above.
func ExportResolversFor(s TestResolverSource, iface string) ([]string, Scope) {
	return s.src.resolversFor(iface)
}

// ExportSetResolverSource replaces a tester's resolver source.
func ExportSetResolverSource(t *Tester, s TestResolverSource) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resolvers = s.src
}
