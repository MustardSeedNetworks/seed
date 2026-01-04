// Package cable exports internal functions for testing.
package cable

// TesterInterfaceName returns the interface name for testing.
func (t *Tester) TesterInterfaceName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.interfaceName
}

// IsSupportedPlatform is exported for testing.
var IsSupportedPlatform = isSupportedPlatform

// TestPlatform is exported for testing.
var TestPlatform = testPlatform
