// Package gateway exports internal functions for testing.
package gateway

import (
	"sync"
	"time"
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
