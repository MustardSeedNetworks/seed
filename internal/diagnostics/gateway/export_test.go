package gateway

import (
	"errors"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
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
func (t *Tester) TesterSetPinger(p *discovery.ICMPPinger) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pinger = p
}

// TesterGetPinger gets the pinger for testing.
func (t *Tester) TesterGetPinger() *discovery.ICMPPinger {
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
