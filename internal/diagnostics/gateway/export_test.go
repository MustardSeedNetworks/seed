package gateway

import (
	"errors"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// TesterPingCount returns the ping count for testing.
func (t *Tester) TesterPingCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pingCount
}

// TesterPingTimeout returns the ping timeout for testing.
func (t *Tester) TesterPingTimeout() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pingTimeout
}

// TesterStats returns the stats for testing.
func (t *Tester) TesterStats() *PingStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}

// TesterSetStats sets the stats for testing.
func (t *Tester) TesterSetStats(stats *PingStats) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats = stats
}

// TesterSetPingCount sets the ping count for testing.
func (t *Tester) TesterSetPingCount(count int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pingCount = count
}

// TesterSetPingTimeout sets the ping timeout for testing.
func (t *Tester) TesterSetPingTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pingTimeout = timeout
}

// TesterMu exposes the mutex for testing.
func (t *Tester) TesterMu() *sync.RWMutex {
	return &t.mu
}

// DetermineStatus is exported for testing.
func (t *Tester) DetermineStatus(stats *PingStats) Status {
	return t.determineStatus(stats)
}

// TesterSetPinger sets the pinger for testing.
func (t *Tester) TesterSetPinger(p *enumerate.ICMPPinger) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pinger = p
}

// TesterGetPinger gets the pinger for testing.
func (t *Tester) TesterGetPinger() *enumerate.ICMPPinger {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pinger
}

// TesterRunning returns the running status for testing.
func (t *Tester) TesterRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

// PingErrorMessage exposes pingErrorMessage for testing.
func PingErrorMessage(err error) string {
	return pingErrorMessage(err)
}

// ErrTest is a sentinel error for testing.
var ErrTest = errors.New("test error")

// RoutingReads stands in for the host's routing table and DHCP leases, so the
// interface-scoping rule is testable without a host that says the right thing.
type RoutingReads struct {
	Routes            []RouteInfo
	RoutesErr         error
	SystemGateway     string
	SystemGatewayIPv6 string
	// LeaseRouters is the router each interface's lease names.
	LeaseRouters map[string]string
	LeaseErr     error
}

func (r RoutingReads) reader() routingReader {
	return routingReader{
		routes:            func() ([]RouteInfo, error) { return r.Routes, r.RoutesErr },
		systemGateway:     func() (string, error) { return r.SystemGateway, nil },
		systemGatewayIPv6: func() (string, error) { return r.SystemGatewayIPv6, nil },
		leaseRouter:       func(iface string) (string, error) { return r.LeaseRouters[iface], r.LeaseErr },
	}
}

// GatewayForInterfaceWithReads applies the IPv4 interface-scoping rule to r.
func GatewayForInterfaceWithReads(iface string, r RoutingReads) (string, Source, error) {
	return r.reader().gatewayForInterface(iface)
}

// IPv6GatewayForInterfaceWithReads applies the IPv6 interface-scoping rule to r.
func IPv6GatewayForInterfaceWithReads(iface string, r RoutingReads) (string, error) {
	return r.reader().ipv6GatewayForInterface(iface)
}

// RouterOfLease exposes routerOfLease for testing.
func RouterOfLease(lease *dhcp.LeaseInfo, err error, now time.Time) (string, error) {
	return routerOfLease(lease, err, now)
}

// SetRoutingForTesting points the tester's detection at r.
func (t *Tester) SetRoutingForTesting(r RoutingReads) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.routing = r.reader()
}
