package dhcp

import (
	"sync"
	"time"
)

// TesterInterfaceName returns the interface name for testing.
func (t *Tester) TesterInterfaceName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.interfaceName
}

// TesterThresholds returns the thresholds for testing.
func (t *Tester) TesterThresholds() Thresholds {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.thresholds
}

// TesterTestTimeout returns the test timeout for testing.
func (t *Tester) TesterTestTimeout() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.testTimeout
}

// TesterSetTestTimeout sets the test timeout for testing (bypasses validation).
func (t *Tester) TesterSetTestTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.testTimeout = timeout
}

// TesterMu exposes the mutex for testing.
func (t *Tester) TesterMu() *sync.RWMutex {
	return &t.mu
}

// GetStatus is exported for testing.
func (t *Tester) GetStatus(duration time.Duration, hasError bool) Status {
	return t.getStatus(duration, hasError)
}

// ExportIsContiguousMask is exported for testing.
func ExportIsContiguousMask(mask []byte) bool {
	if len(mask) != ipv4Len {
		return false
	}
	return isContiguousMask(mask)
}
