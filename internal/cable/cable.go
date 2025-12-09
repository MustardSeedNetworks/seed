// Package cable provides TDR cable testing functionality.
package cable

import (
	"runtime"
	"sync"
)

// Status represents the cable test status.
type Status string

const (
	StatusOK                Status = "ok"
	StatusOpen              Status = "open"
	StatusShort             Status = "short"
	StatusImpedanceMismatch Status = "impedance_mismatch"
	StatusUnknown           Status = "unknown"
)

// TestResult contains the cable test results.
type TestResult struct {
	Supported bool     `json:"supported"`
	Length    *float64 `json:"length,omitempty"` // meters
	Status    Status   `json:"status"`
	Faults    []string `json:"faults"`
}

// Tester performs TDR cable tests.
type Tester struct {
	interfaceName string
	mu            sync.RWMutex
}

// NewTester creates a new cable tester.
func NewTester(interfaceName string) *Tester {
	return &Tester{
		interfaceName: interfaceName,
	}
}

// SetInterface updates the interface to test.
func (t *Tester) SetInterface(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.interfaceName = name
}

// IsSupported checks if the interface supports TDR testing.
func (t *Tester) IsSupported() bool {
	t.mu.RLock()
	iface := t.interfaceName
	t.mu.RUnlock()

	return isSupportedPlatform(iface)
}

// Test performs a cable test on the interface.
func (t *Tester) Test() *TestResult {
	t.mu.RLock()
	iface := t.interfaceName
	t.mu.RUnlock()

	result := &TestResult{
		Supported: false,
		Status:    StatusUnknown,
		Faults:    make([]string, 0),
	}

	switch runtime.GOOS {
	case "linux":
		return testPlatform(iface)
	case "darwin":
		// macOS doesn't support TDR via standard tools
		return result
	default:
		return result
	}
}

// GetLastResult returns the result of the last cable test.
// This is useful for caching results since cable tests can take time.
func (t *Tester) GetLastResult() *TestResult {
	// For now, just run a new test
	// A more sophisticated implementation would cache the result
	return t.Test()
}
