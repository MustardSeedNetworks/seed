// Package cable provides TDR cable testing functionality.
// Test suite validates cable test status constants and result interpretation.
package cable

import (
	"testing"
)

func TestStatusConstants(t *testing.T) {
	if StatusOK != "ok" {
		t.Errorf("expected StatusOK = 'ok', got %q", StatusOK)
	}
	if StatusOpen != "open" {
		t.Errorf("expected StatusOpen = 'open', got %q", StatusOpen)
	}
	if StatusShort != "short" {
		t.Errorf("expected StatusShort = 'short', got %q", StatusShort)
	}
	if StatusImpedanceMismatch != "impedance_mismatch" {
		t.Errorf("expected StatusImpedanceMismatch = 'impedance_mismatch', got %q", StatusImpedanceMismatch)
	}
	if StatusUnknown != "unknown" {
		t.Errorf("expected StatusUnknown = 'unknown', got %q", StatusUnknown)
	}
}

func TestNewTester(t *testing.T) {
	tester := NewTester("eth0")
	if tester == nil {
		t.Fatal("expected non-nil tester")
	}

	if tester.interfaceName != "eth0" {
		t.Errorf("expected interfaceName 'eth0', got %q", tester.interfaceName)
	}
}

func TestTesterSetInterface(t *testing.T) {
	tester := NewTester("eth0")

	tester.SetInterface("en0")
	if tester.interfaceName != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", tester.interfaceName)
	}

	tester.SetInterface("bond0")
	if tester.interfaceName != "bond0" {
		t.Errorf("expected interfaceName 'bond0', got %q", tester.interfaceName)
	}
}

func TestTesterIsSupported(t *testing.T) {
	tester := NewTester("eth0")

	// This will return false on non-Linux systems or without ethtool
	supported := tester.IsSupported()
	// Just verify it doesn't panic
	_ = supported
}

func TestTesterTest(t *testing.T) {
	tester := NewTester("eth0")

	result := tester.Test()
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
	// On non-Linux systems, should return unsupported
}

func TestTesterGetLastResult(t *testing.T) {
	tester := NewTester("eth0")

	result := tester.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result from GetLastResult")
	}
}

func TestTestResultFields(t *testing.T) {
	length := 25.5
	result := TestResult{
		Supported: true,
		Length:    &length,
		Status:    StatusOK,
		Faults:    []string{"fault1", "fault2"},
	}

	if !result.Supported {
		t.Error("expected Supported to be true")
	}
	if result.Length == nil || *result.Length != 25.5 {
		t.Error("expected Length 25.5")
	}
	if result.Status != StatusOK {
		t.Errorf("expected Status StatusOK, got %v", result.Status)
	}
	if len(result.Faults) != 2 {
		t.Errorf("expected 2 faults, got %d", len(result.Faults))
	}
}

func TestTestResultNoLength(t *testing.T) {
	result := TestResult{
		Supported: false,
		Length:    nil,
		Status:    StatusUnknown,
		Faults:    []string{},
	}

	if result.Supported {
		t.Error("expected Supported to be false")
	}
	if result.Length != nil {
		t.Error("expected nil Length")
	}
	if result.Status != StatusUnknown {
		t.Errorf("expected Status StatusUnknown, got %v", result.Status)
	}
	if len(result.Faults) != 0 {
		t.Errorf("expected 0 faults, got %d", len(result.Faults))
	}
}

func TestConcurrentTesterAccess(t *testing.T) {
	tester := NewTester("eth0")

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				tester.SetInterface("eth" + string(rune('0'+id)))
				_ = tester.IsSupported()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestIsSupportedPlatform(t *testing.T) {
	// Test the platform-specific function
	result := isSupportedPlatform("eth0")
	// Just verify it doesn't panic - result depends on system
	_ = result
}

func TestTestPlatform(t *testing.T) {
	// Test the platform-specific function
	result := testPlatform("eth0")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// On non-Linux, should not be supported
	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
}

func TestGetLastResultCallsTest(t *testing.T) {
	tester := NewTester("eth0")

	result := tester.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Currently GetLastResult just runs a new test
	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
}

func TestTestResultWithAllFields(t *testing.T) {
	length := 50.0
	result := TestResult{
		Supported: true,
		Length:    &length,
		Status:    StatusOK,
		Faults:    []string{"minor issue"},
	}

	if !result.Supported {
		t.Error("expected Supported true")
	}
	if result.Length == nil || *result.Length != 50.0 {
		t.Error("expected Length 50.0")
	}
	if result.Status != StatusOK {
		t.Errorf("expected Status OK, got %v", result.Status)
	}
	if len(result.Faults) != 1 || result.Faults[0] != "minor issue" {
		t.Error("expected one fault 'minor issue'")
	}
}

func TestStatusValues(t *testing.T) {
	statuses := []Status{
		StatusOK,
		StatusOpen,
		StatusShort,
		StatusImpedanceMismatch,
		StatusUnknown,
	}

	expectedStrings := []string{
		"ok",
		"open",
		"short",
		"impedance_mismatch",
		"unknown",
	}

	for i, s := range statuses {
		if string(s) != expectedStrings[i] {
			t.Errorf("expected %q, got %q", expectedStrings[i], string(s))
		}
	}
}

func TestTesterWithDifferentInterfaces(t *testing.T) {
	interfaces := []string{"eth0", "en0", "wlan0", "bond0", "lo0"}

	for _, iface := range interfaces {
		tester := NewTester(iface)
		if tester.interfaceName != iface {
			t.Errorf("expected interfaceName %q, got %q", iface, tester.interfaceName)
		}

		// Just verify no panics
		_ = tester.IsSupported()
		result := tester.Test()
		if result == nil {
			t.Errorf("expected non-nil result for %s", iface)
		}
	}
}

func TestTestResultEmptyFaults(t *testing.T) {
	result := TestResult{
		Supported: true,
		Status:    StatusOK,
		Faults:    make([]string, 0),
	}

	if len(result.Faults) != 0 {
		t.Error("expected empty faults slice")
	}

	// Verify we can append to it
	result.Faults = append(result.Faults, "new fault")
	if len(result.Faults) != 1 {
		t.Error("expected one fault after append")
	}
}

func TestTesterTestReturnsValidResult(t *testing.T) {
	tester := NewTester("lo0")

	result := tester.Test()

	// Verify all fields are initialized
	if result.Faults == nil {
		t.Error("Faults should not be nil")
	}

	// Status should be set (even if unknown)
	validStatuses := map[Status]bool{
		StatusOK:                true,
		StatusOpen:              true,
		StatusShort:             true,
		StatusImpedanceMismatch: true,
		StatusUnknown:           true,
	}

	if !validStatuses[result.Status] {
		t.Errorf("unexpected Status: %v", result.Status)
	}
}
