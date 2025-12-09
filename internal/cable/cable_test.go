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

func TestParseTDROutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedStatus Status
		hasLength      bool
		expectedLength float64
	}{
		{
			name:           "ok status",
			output:         "Cable status: ok\nCable length: 25 meters",
			expectedStatus: StatusOK,
			hasLength:      true,
			expectedLength: 25.0,
		},
		{
			name:           "open status",
			output:         "Cable test result: open circuit detected",
			expectedStatus: StatusOpen,
			hasLength:      false,
		},
		{
			name:           "short status",
			output:         "Cable test result: short circuit detected",
			expectedStatus: StatusShort,
			hasLength:      false,
		},
		{
			name:           "impedance mismatch",
			output:         "Cable test: impedance mismatch",
			expectedStatus: StatusImpedanceMismatch,
			hasLength:      false,
		},
		{
			name:           "pass status",
			output:         "Test result: PASS",
			expectedStatus: StatusOK,
			hasLength:      false,
		},
		{
			name:           "length with decimal",
			output:         "length: 15.5m",
			expectedStatus: StatusUnknown,
			hasLength:      true,
			expectedLength: 15.5,
		},
		{
			name:           "empty output",
			output:         "",
			expectedStatus: StatusUnknown,
			hasLength:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{
				Status: StatusUnknown,
				Faults: make([]string, 0),
			}
			result = parseTDROutput(tt.output, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected Status %v, got %v", tt.expectedStatus, result.Status)
			}
			if tt.hasLength {
				if result.Length == nil {
					t.Error("expected non-nil Length")
				} else if *result.Length != tt.expectedLength {
					t.Errorf("expected Length %v, got %v", tt.expectedLength, *result.Length)
				}
			}
		})
	}
}

func TestParseCableTestOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedStatus Status
	}{
		{
			name:           "pair ok",
			output:         "Pair A: ok\nPair B: ok\nPair C: ok\nPair D: ok",
			expectedStatus: StatusOK,
		},
		{
			name:           "pair terminated",
			output:         "Pair A: terminated, 25m",
			expectedStatus: StatusOK,
		},
		{
			name:           "pair open",
			output:         "Pair A: open, 15m",
			expectedStatus: StatusOpen,
		},
		{
			name:           "pair short",
			output:         "Pair B: short, 5m",
			expectedStatus: StatusShort,
		},
		{
			name:           "empty output",
			output:         "",
			expectedStatus: StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{
				Status: StatusUnknown,
				Faults: make([]string, 0),
			}
			result = parseCableTestOutput(tt.output, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected Status %v, got %v", tt.expectedStatus, result.Status)
			}
		})
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

func TestIsSupportedLinux(t *testing.T) {
	// Test the Linux-specific function
	result := isSupportedLinux("eth0")
	// Just verify it doesn't panic - result depends on system
	_ = result
}

func TestTestLinux(t *testing.T) {
	// Test the Linux-specific function
	result := testLinux("eth0")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// On non-Linux, should not be supported
	if result.Faults == nil {
		t.Error("expected non-nil Faults slice")
	}
}

func TestParseTDROutputMoreCases(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedStatus Status
		hasLength      bool
	}{
		{
			name:           "terminated cable",
			output:         "Cable properly terminated",
			expectedStatus: StatusUnknown,
			hasLength:      false,
		},
		{
			name:           "cable length with meters",
			output:         "Cable length: 50 meters",
			expectedStatus: StatusUnknown,
			hasLength:      true,
		},
		{
			name:           "multiple status indicators",
			output:         "Test: ok\nlength: 25m\nStatus: pass",
			expectedStatus: StatusOK,
			hasLength:      true,
		},
		{
			name:           "impedance mismatch at distance",
			output:         "Fault: impedance mismatch\nlength: 15.5m",
			expectedStatus: StatusImpedanceMismatch,
			hasLength:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{
				Status: StatusUnknown,
				Faults: make([]string, 0),
			}
			result = parseTDROutput(tt.output, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected Status %v, got %v", tt.expectedStatus, result.Status)
			}
			if tt.hasLength && result.Length == nil {
				t.Error("expected non-nil Length")
			}
		})
	}
}

func TestParseCableTestOutputMoreCases(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedStatus Status
	}{
		{
			name:           "all pairs ok",
			output:         "Pair A: ok, 25m\nPair B: ok, 25m\nPair C: ok, 25m\nPair D: ok, 25m",
			expectedStatus: StatusOK,
		},
		{
			name:           "mixed status",
			output:         "Pair A: ok\nPair B: open, 10m",
			expectedStatus: StatusOpen,
		},
		{
			name:           "short takes precedence",
			output:         "Pair A: ok\nPair C: short, 5m",
			expectedStatus: StatusShort,
		},
		{
			name:           "multiple faults",
			output:         "Pair A: open, 10m\nPair B: short, 5m",
			expectedStatus: StatusShort, // Last fault overwrites
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{
				Status: StatusUnknown,
				Faults: make([]string, 0),
			}
			result = parseCableTestOutput(tt.output, result)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected Status %v, got %v", tt.expectedStatus, result.Status)
			}
		})
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

func TestParseLengthFormats(t *testing.T) {
	tests := []struct {
		output   string
		expected float64
	}{
		{"length: 25m", 25.0},
		{"length: 25 m", 25.0},
		{"length: 25 meters", 25.0},
		{"Cable length: 100.5 meters", 100.5},
		{"length: 0.5m", 0.5},
	}

	for _, tt := range tests {
		result := &TestResult{Status: StatusUnknown, Faults: make([]string, 0)}
		result = parseTDROutput(tt.output, result)

		if result.Length == nil {
			t.Errorf("expected non-nil Length for %q", tt.output)
		} else if *result.Length != tt.expected {
			t.Errorf("for %q: expected Length %v, got %v", tt.output, tt.expected, *result.Length)
		}
	}
}
