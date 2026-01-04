// Package cable_test provides TDR cable testing functionality.
// Test suite validates cable test status constants and result interpretation.
package cable_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/sap/cable"
)

func TestStatusConstants(t *testing.T) {
	if cable.StatusOK != "ok" {
		t.Errorf("expected StatusOK = 'ok', got %q", cable.StatusOK)
	}
	if cable.StatusOpen != "open" {
		t.Errorf("expected StatusOpen = 'open', got %q", cable.StatusOpen)
	}
	if cable.StatusShort != "short" {
		t.Errorf("expected StatusShort = 'short', got %q", cable.StatusShort)
	}
	if cable.StatusImpedanceMismatch != "impedance_mismatch" {
		t.Errorf("expected StatusImpedanceMismatch = 'impedance_mismatch', got %q", cable.StatusImpedanceMismatch)
	}
	if cable.StatusUnknown != "unknown" {
		t.Errorf("expected StatusUnknown = 'unknown', got %q", cable.StatusUnknown)
	}
}

func TestNewTester(t *testing.T) {
	tester := cable.NewTester("eth0")
	if tester == nil {
		t.Fatal("expected non-nil tester")
	}

	if tester.TesterInterfaceName() != "eth0" {
		t.Errorf("expected interfaceName 'eth0', got %q", tester.TesterInterfaceName())
	}
}

func TestTesterSetInterface(t *testing.T) {
	tester := cable.NewTester("eth0")

	tester.SetInterface("en0")
	if tester.TesterInterfaceName() != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", tester.TesterInterfaceName())
	}

	tester.SetInterface("bond0")
	if tester.TesterInterfaceName() != "bond0" {
		t.Errorf("expected interfaceName 'bond0', got %q", tester.TesterInterfaceName())
	}
}

func TestTesterIsSupported(_ *testing.T) {
	tester := cable.NewTester("eth0")

	// This will return false on non-Linux systems or without ethtool.
	supported := tester.IsSupported()
	// Just verify it doesn't panic.
	_ = supported
}

func TestTesterTest(t *testing.T) {
	tester := cable.NewTester("eth0")

	result := tester.Test()
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
	// On non-Linux systems, should return unsupported.
}

func TestTesterGetLastResult(t *testing.T) {
	tester := cable.NewTester("eth0")

	result := tester.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result from GetLastResult")
	}
}

func TestTestResultFields(t *testing.T) {
	length := 25.5
	result := cable.TestResult{
		Supported: true,
		Length:    &length,
		Status:    cable.StatusOK,
		Faults:    []string{"fault1", "fault2"},
	}

	if !result.Supported {
		t.Error("expected Supported to be true")
	}
	if result.Length == nil || *result.Length != 25.5 {
		t.Error("expected Length 25.5")
	}
	if result.Status != cable.StatusOK {
		t.Errorf("expected Status StatusOK, got %v", result.Status)
	}
	if len(result.Faults) != 2 {
		t.Errorf("expected 2 faults, got %d", len(result.Faults))
	}
}

func TestTestResultNoLength(t *testing.T) {
	result := cable.TestResult{
		Supported: false,
		Length:    nil,
		Status:    cable.StatusUnknown,
		Faults:    []string{},
	}

	if result.Supported {
		t.Error("expected Supported to be false")
	}
	if result.Length != nil {
		t.Error("expected nil Length")
	}
	if result.Status != cable.StatusUnknown {
		t.Errorf("expected Status StatusUnknown, got %v", result.Status)
	}
	if len(result.Faults) != 0 {
		t.Errorf("expected 0 faults, got %d", len(result.Faults))
	}
}

func TestConcurrentTesterAccess(_ *testing.T) {
	tester := cable.NewTester("eth0")

	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for range 50 {
				tester.SetInterface("eth" + string(rune('0'+id)))
				_ = tester.IsSupported()
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
}

func TestIsSupportedPlatform(_ *testing.T) {
	// Test the platform-specific function.
	result := cable.IsSupportedPlatform("eth0")
	// Just verify it doesn't panic - result depends on system.
	_ = result
}

func TestTestPlatform(t *testing.T) {
	// Test the platform-specific function.
	result := cable.TestPlatform("eth0")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// On non-Linux, should not be supported.
	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
}

func TestGetLastResultCallsTest(t *testing.T) {
	tester := cable.NewTester("eth0")

	result := tester.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Currently GetLastResult just runs a new test.
	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
}

func TestTestResultWithAllFields(t *testing.T) {
	length := 50.0
	result := cable.TestResult{
		Supported: true,
		Length:    &length,
		Status:    cable.StatusOK,
		Faults:    []string{"minor issue"},
	}

	if !result.Supported {
		t.Error("expected Supported true")
	}
	if result.Length == nil || *result.Length != 50.0 {
		t.Error("expected Length 50.0")
	}
	if result.Status != cable.StatusOK {
		t.Errorf("expected Status OK, got %v", result.Status)
	}
	if len(result.Faults) != 1 || result.Faults[0] != "minor issue" {
		t.Error("expected one fault 'minor issue'")
	}
}

func TestStatusValues(t *testing.T) {
	statuses := []cable.Status{
		cable.StatusOK,
		cable.StatusOpen,
		cable.StatusShort,
		cable.StatusImpedanceMismatch,
		cable.StatusUnknown,
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
		tester := cable.NewTester(iface)
		if tester.TesterInterfaceName() != iface {
			t.Errorf("expected interfaceName %q, got %q", iface, tester.TesterInterfaceName())
		}

		// Just verify no panics.
		_ = tester.IsSupported()
		result := tester.Test()
		if result == nil {
			t.Errorf("expected non-nil result for %s", iface)
		}
	}
}

func TestTestResultEmptyFaults(t *testing.T) {
	result := cable.TestResult{
		Supported: true,
		Status:    cable.StatusOK,
		Faults:    make([]string, 0),
	}

	if !result.Supported {
		t.Error("expected Supported to be true")
	}
	if result.Status != cable.StatusOK {
		t.Errorf("expected Status %v, got %v", cable.StatusOK, result.Status)
	}
	if len(result.Faults) != 0 {
		t.Error("expected empty faults slice")
	}

	// Verify we can append to it.
	result.Faults = append(result.Faults, "new fault")
	if len(result.Faults) != 1 {
		t.Error("expected one fault after append")
	}
}

func TestTesterTestReturnsValidResult(t *testing.T) {
	tester := cable.NewTester("lo0")

	result := tester.Test()

	// Verify all fields are initialized.
	if result.Faults == nil {
		t.Error("Faults should not be nil")
	}

	// Status should be set (even if unknown).
	validStatuses := map[cable.Status]bool{
		cable.StatusOK:                true,
		cable.StatusOpen:              true,
		cable.StatusShort:             true,
		cable.StatusImpedanceMismatch: true,
		cable.StatusCrosstalk:         true,
		cable.StatusSplitPair:         true,
		cable.StatusUnknown:           true,
	}

	if !validStatuses[result.Status] {
		t.Errorf("unexpected Status: %v", result.Status)
	}
}
