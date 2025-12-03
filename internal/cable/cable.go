// Package cable provides TDR cable testing functionality.
package cable

import (
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Status represents the cable test status.
type Status string

const (
	StatusOK               Status = "ok"
	StatusOpen             Status = "open"
	StatusShort            Status = "short"
	StatusImpedanceMismatch Status = "impedance_mismatch"
	StatusUnknown          Status = "unknown"
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

	switch runtime.GOOS {
	case "linux":
		return isSupportedLinux(iface)
	default:
		// TDR is typically only available on Linux via ethtool
		return false
	}
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
		return testLinux(iface)
	case "darwin":
		// macOS doesn't support TDR via standard tools
		return result
	default:
		return result
	}
}

// isSupportedLinux checks if the NIC supports TDR on Linux.
func isSupportedLinux(iface string) bool {
	// Check if ethtool supports cable-test for this interface
	cmd := exec.Command("ethtool", "--show-features", iface)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for TDR support indications
	outStr := string(output)
	return strings.Contains(outStr, "cable-test") || strings.Contains(outStr, "tdr")
}

// testLinux performs a cable test on Linux using ethtool.
func testLinux(iface string) *TestResult {
	result := &TestResult{
		Supported: false,
		Status:    StatusUnknown,
		Faults:    make([]string, 0),
	}

	// First, try to initiate a cable test
	cmd := exec.Command("ethtool", "--cable-test", iface)
	output, err := cmd.CombinedOutput()

	// Check if command is supported
	outStr := string(output)
	if strings.Contains(outStr, "not supported") ||
		strings.Contains(outStr, "Operation not supported") ||
		strings.Contains(outStr, "unknown command") {
		return result
	}

	result.Supported = true

	if err != nil {
		// Some errors are okay - the test might still have run
		if strings.Contains(outStr, "link test") {
			result.Faults = append(result.Faults, "Link must be down to test")
		}
	}

	// Try to get cable test results
	cmd = exec.Command("ethtool", "--cable-test-tdr", iface)
	output, err = cmd.CombinedOutput()
	if err == nil {
		outStr = string(output)
		result = parseTDROutput(outStr, result)
	}

	// If TDR didn't work, try the basic cable test output
	if result.Status == StatusUnknown && len(output) > 0 {
		result = parseCableTestOutput(string(output), result)
	}

	return result
}

// parseTDROutput parses the output of ethtool --cable-test-tdr.
func parseTDROutput(output string, result *TestResult) *TestResult {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse cable length
		if strings.Contains(line, "Cable length:") || strings.Contains(line, "length:") {
			re := regexp.MustCompile(`(\d+\.?\d*)\s*(m|meters?)`)
			if matches := re.FindStringSubmatch(line); len(matches) >= 2 {
				if length, err := strconv.ParseFloat(matches[1], 64); err == nil {
					result.Length = &length
				}
			}
		}

		// Parse status
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "open") {
			result.Status = StatusOpen
			result.Faults = append(result.Faults, "Open circuit detected")
		} else if strings.Contains(lineLower, "short") {
			result.Status = StatusShort
			result.Faults = append(result.Faults, "Short circuit detected")
		} else if strings.Contains(lineLower, "impedance") {
			result.Status = StatusImpedanceMismatch
			result.Faults = append(result.Faults, "Impedance mismatch")
		} else if strings.Contains(lineLower, "ok") || strings.Contains(lineLower, "pass") {
			result.Status = StatusOK
		}
	}

	return result
}

// parseCableTestOutput parses the output of ethtool --cable-test.
func parseCableTestOutput(output string, result *TestResult) *TestResult {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		// Look for pair status (Pair A, B, C, D for gigabit Ethernet)
		if strings.Contains(lineLower, "pair") {
			if strings.Contains(lineLower, "ok") || strings.Contains(lineLower, "terminated") {
				if result.Status == StatusUnknown {
					result.Status = StatusOK
				}
			} else if strings.Contains(lineLower, "open") {
				result.Status = StatusOpen
				result.Faults = append(result.Faults, line)
			} else if strings.Contains(lineLower, "short") {
				result.Status = StatusShort
				result.Faults = append(result.Faults, line)
			}

			// Try to extract distance
			re := regexp.MustCompile(`(\d+\.?\d*)\s*(m|meters?)`)
			if matches := re.FindStringSubmatch(line); len(matches) >= 2 {
				if length, err := strconv.ParseFloat(matches[1], 64); err == nil {
					// Use the maximum length found
					if result.Length == nil || length > *result.Length {
						result.Length = &length
					}
				}
			}
		}
	}

	return result
}

// GetLastResult returns the result of the last cable test.
// This is useful for caching results since cable tests can take time.
func (t *Tester) GetLastResult() *TestResult {
	// For now, just run a new test
	// A more sophisticated implementation would cache the result
	return t.Test()
}
