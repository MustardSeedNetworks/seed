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
		t.Errorf(
			"expected StatusImpedanceMismatch = 'impedance_mismatch', got %q",
			cable.StatusImpedanceMismatch,
		)
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
	result := cable.ExportIsSupportedPlatform("eth0")
	// Just verify it doesn't panic - result depends on system.
	_ = result
}

func TestTestPlatform(t *testing.T) {
	// Test the platform-specific function.
	result := cable.ExportTestPlatform("eth0")
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

// TestGet568APinout verifies the T568A wiring standard pinout.
func TestGet568APinout(t *testing.T) {
	pinout := cable.Get568APinout()

	if len(pinout) != 8 {
		t.Fatalf("expected 8 pins, got %d", len(pinout))
	}

	// Verify each pin mapping according to T568A standard.
	expected := []struct {
		pin   int
		color string
		pair  string
	}{
		{1, "White/Green", "3-6"},
		{2, "Green", "3-6"},
		{3, "White/Orange", "1-2"},
		{4, "Blue", "4-5"},
		{5, "White/Blue", "4-5"},
		{6, "Orange", "1-2"},
		{7, "White/Brown", "7-8"},
		{8, "Brown", "7-8"},
	}

	for i, exp := range expected {
		if pinout[i].Pin != exp.pin {
			t.Errorf("pin %d: expected Pin %d, got %d", i+1, exp.pin, pinout[i].Pin)
		}
		if pinout[i].Color != exp.color {
			t.Errorf("pin %d: expected Color %q, got %q", i+1, exp.color, pinout[i].Color)
		}
		if pinout[i].Pair != exp.pair {
			t.Errorf("pin %d: expected Pair %q, got %q", i+1, exp.pair, pinout[i].Pair)
		}
	}
}

// TestGet568BPinout verifies the T568B wiring standard pinout.
func TestGet568BPinout(t *testing.T) {
	pinout := cable.Get568BPinout()

	if len(pinout) != 8 {
		t.Fatalf("expected 8 pins, got %d", len(pinout))
	}

	// Verify each pin mapping according to T568B standard.
	expected := []struct {
		pin   int
		color string
		pair  string
	}{
		{1, "White/Orange", "1-2"},
		{2, "Orange", "1-2"},
		{3, "White/Green", "3-6"},
		{4, "Blue", "4-5"},
		{5, "White/Blue", "4-5"},
		{6, "Green", "3-6"},
		{7, "White/Brown", "7-8"},
		{8, "Brown", "7-8"},
	}

	for i, exp := range expected {
		if pinout[i].Pin != exp.pin {
			t.Errorf("pin %d: expected Pin %d, got %d", i+1, exp.pin, pinout[i].Pin)
		}
		if pinout[i].Color != exp.color {
			t.Errorf("pin %d: expected Color %q, got %q", i+1, exp.color, pinout[i].Color)
		}
		if pinout[i].Pair != exp.pair {
			t.Errorf("pin %d: expected Pair %q, got %q", i+1, exp.pair, pinout[i].Pair)
		}
	}
}

// TestGetPinout tests the GetPinout function with all wiring standards.
func TestGetPinout(t *testing.T) {
	tests := []struct {
		name     string
		standard cable.WiringStandard
		wantPin1 string // Color for pin 1
	}{
		{
			name:     "568A standard",
			standard: cable.Wiring568A,
			wantPin1: "White/Green",
		},
		{
			name:     "568B standard",
			standard: cable.Wiring568B,
			wantPin1: "White/Orange",
		},
		{
			name:     "unknown standard defaults to 568B",
			standard: cable.WiringStandard("unknown"),
			wantPin1: "White/Orange",
		},
		{
			name:     "empty standard defaults to 568B",
			standard: cable.WiringStandard(""),
			wantPin1: "White/Orange",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pinout := cable.GetPinout(tt.standard)
			if len(pinout) != 8 {
				t.Fatalf("expected 8 pins, got %d", len(pinout))
			}
			if pinout[0].Color != tt.wantPin1 {
				t.Errorf("expected pin 1 color %q, got %q", tt.wantPin1, pinout[0].Color)
			}
		})
	}
}

// TestMetersToFeet tests the meter to feet conversion function.
func TestMetersToFeet(t *testing.T) {
	tests := []struct {
		name    string
		meters  float64
		wantFt  float64
		epsilon float64
	}{
		{
			name:    "zero meters",
			meters:  0,
			wantFt:  0,
			epsilon: 0.0001,
		},
		{
			name:    "one meter",
			meters:  1.0,
			wantFt:  3.28084,
			epsilon: 0.0001,
		},
		{
			name:    "ten meters",
			meters:  10.0,
			wantFt:  32.8084,
			epsilon: 0.001,
		},
		{
			name:    "100 meters (Cat5e max length)",
			meters:  100.0,
			wantFt:  328.084,
			epsilon: 0.01,
		},
		{
			name:    "fractional meters",
			meters:  2.5,
			wantFt:  8.2021,
			epsilon: 0.001,
		},
		{
			name:    "negative meters (edge case)",
			meters:  -5.0,
			wantFt:  -16.4042,
			epsilon: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cable.MetersToFeet(tt.meters)
			diff := got - tt.wantFt
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.epsilon {
				t.Errorf("MetersToFeet(%v) = %v, want %v (within %v)", tt.meters, got, tt.wantFt, tt.epsilon)
			}
		})
	}
}

// TestMetersToFeetFactor verifies the conversion factor constant.
func TestMetersToFeetFactor(t *testing.T) {
	if cable.MetersToFeetFactor != 3.28084 {
		t.Errorf("expected MetersToFeetFactor = 3.28084, got %v", cable.MetersToFeetFactor)
	}
}

// TestGetPairInfo verifies the Ethernet pair information.
func TestGetPairInfo(t *testing.T) {
	pairInfo := cable.GetPairInfo()

	if len(pairInfo) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairInfo))
	}

	// Verify pair information.
	expected := []struct {
		letter   string
		pins     string
		usage10  string
		usageGig string
	}{
		{"A", "1-2", "TX+/TX-", "BI_DA+/BI_DA-"},
		{"B", "3-6", "RX+/RX-", "BI_DB+/BI_DB-"},
		{"C", "4-5", "Unused", "BI_DC+/BI_DC-"},
		{"D", "7-8", "Unused", "BI_DD+/BI_DD-"},
	}

	for i, exp := range expected {
		if pairInfo[i].Letter != exp.letter {
			t.Errorf("pair %d: expected Letter %q, got %q", i, exp.letter, pairInfo[i].Letter)
		}
		if pairInfo[i].Pins != exp.pins {
			t.Errorf("pair %d: expected Pins %q, got %q", i, exp.pins, pairInfo[i].Pins)
		}
		if pairInfo[i].Usage10 != exp.usage10 {
			t.Errorf("pair %d: expected Usage10 %q, got %q", i, exp.usage10, pairInfo[i].Usage10)
		}
		if pairInfo[i].UsageGig != exp.usageGig {
			t.Errorf("pair %d: expected UsageGig %q, got %q", i, exp.usageGig, pairInfo[i].UsageGig)
		}
	}
}

// TestPinConstants verifies the RJ45 pin number constants.
func TestPinConstants(t *testing.T) {
	if cable.Pin1 != 1 {
		t.Errorf("expected Pin1 = 1, got %d", cable.Pin1)
	}
	if cable.Pin2 != 2 {
		t.Errorf("expected Pin2 = 2, got %d", cable.Pin2)
	}
	if cable.Pin3 != 3 {
		t.Errorf("expected Pin3 = 3, got %d", cable.Pin3)
	}
	if cable.Pin4 != 4 {
		t.Errorf("expected Pin4 = 4, got %d", cable.Pin4)
	}
	if cable.Pin5 != 5 {
		t.Errorf("expected Pin5 = 5, got %d", cable.Pin5)
	}
	if cable.Pin6 != 6 {
		t.Errorf("expected Pin6 = 6, got %d", cable.Pin6)
	}
	if cable.Pin7 != 7 {
		t.Errorf("expected Pin7 = 7, got %d", cable.Pin7)
	}
	if cable.Pin8 != 8 {
		t.Errorf("expected Pin8 = 8, got %d", cable.Pin8)
	}
}

// TestWiringStandardConstants verifies the wiring standard constants.
func TestWiringStandardConstants(t *testing.T) {
	if cable.Wiring568A != "568A" {
		t.Errorf("expected Wiring568A = '568A', got %q", cable.Wiring568A)
	}
	if cable.Wiring568B != "568B" {
		t.Errorf("expected Wiring568B = '568B', got %q", cable.Wiring568B)
	}
}

// TestPairResultStruct verifies the PairResult structure.
func TestPairResultStruct(t *testing.T) {
	lengthM := 25.5
	lengthFt := cable.MetersToFeet(lengthM)

	result := cable.PairResult{
		Pair:       "1-2",
		PairLetter: "A",
		Status:     cable.StatusOK,
		LengthM:    &lengthM,
		LengthFt:   &lengthFt,
	}

	if result.Pair != "1-2" {
		t.Errorf("expected Pair '1-2', got %q", result.Pair)
	}
	if result.PairLetter != "A" {
		t.Errorf("expected PairLetter 'A', got %q", result.PairLetter)
	}
	if result.Status != cable.StatusOK {
		t.Errorf("expected Status OK, got %v", result.Status)
	}
	if result.LengthM == nil || *result.LengthM != 25.5 {
		t.Error("expected LengthM 25.5")
	}
	if result.LengthFt == nil || (*result.LengthFt-83.6614) > 0.01 {
		t.Errorf("expected LengthFt ~83.6614, got %v", *result.LengthFt)
	}
}

// TestPairResultNilLengths verifies PairResult with nil lengths.
func TestPairResultNilLengths(t *testing.T) {
	result := cable.PairResult{
		Pair:       "3-6",
		PairLetter: "B",
		Status:     cable.StatusOpen,
		LengthM:    nil,
		LengthFt:   nil,
	}

	// Verify all fields are set correctly.
	if result.Pair != "3-6" {
		t.Errorf("expected Pair '3-6', got %q", result.Pair)
	}
	if result.PairLetter != "B" {
		t.Errorf("expected PairLetter 'B', got %q", result.PairLetter)
	}
	if result.Status != cable.StatusOpen {
		t.Errorf("expected Status Open, got %v", result.Status)
	}
	if result.LengthM != nil {
		t.Error("expected nil LengthM")
	}
	if result.LengthFt != nil {
		t.Error("expected nil LengthFt")
	}
}

// TestWirePinoutStruct verifies the WirePinout structure.
func TestWirePinoutStruct(t *testing.T) {
	pinout := cable.WirePinout{
		Pin:   1,
		Color: "White/Orange",
		Pair:  "1-2",
	}

	if pinout.Pin != 1 {
		t.Errorf("expected Pin 1, got %d", pinout.Pin)
	}
	if pinout.Color != "White/Orange" {
		t.Errorf("expected Color 'White/Orange', got %q", pinout.Color)
	}
	if pinout.Pair != "1-2" {
		t.Errorf("expected Pair '1-2', got %q", pinout.Pair)
	}
}

// TestTestResultAllFields verifies TestResult with all optional fields.
func TestTestResultAllFields(t *testing.T) {
	length := 50.0
	lengthFt := cable.MetersToFeet(length)

	result := cable.TestResult{
		Supported: true,
		Status:    cable.StatusOK,
		Length:    &length,
		LengthFt:  &lengthFt,
		Pairs: []cable.PairResult{
			{Pair: "1-2", PairLetter: "A", Status: cable.StatusOK},
			{Pair: "3-6", PairLetter: "B", Status: cable.StatusOK},
			{Pair: "4-5", PairLetter: "C", Status: cable.StatusOK},
			{Pair: "7-8", PairLetter: "D", Status: cable.StatusOK},
		},
		Faults:      []string{},
		WiringStd:   cable.Wiring568B,
		Pinout:      cable.Get568BPinout(),
		IsCrossover: false,
		DriverName:  "e1000e",
	}

	if !result.Supported {
		t.Error("expected Supported true")
	}
	if result.Status != cable.StatusOK {
		t.Errorf("expected Status OK, got %v", result.Status)
	}
	if result.Length == nil || *result.Length != 50.0 {
		t.Error("expected Length 50.0")
	}
	if result.LengthFt == nil || (*result.LengthFt-164.042) > 0.01 {
		t.Errorf("expected LengthFt ~164.042, got %v", result.LengthFt)
	}
	if len(result.Pairs) != 4 {
		t.Errorf("expected 4 pairs, got %d", len(result.Pairs))
	}
	if len(result.Faults) != 0 {
		t.Errorf("expected 0 faults, got %d", len(result.Faults))
	}
	if result.WiringStd != cable.Wiring568B {
		t.Errorf("expected WiringStd 568B, got %v", result.WiringStd)
	}
	if len(result.Pinout) != 8 {
		t.Errorf("expected 8 pinout entries, got %d", len(result.Pinout))
	}
	if result.IsCrossover {
		t.Error("expected IsCrossover false")
	}
	if result.DriverName != "e1000e" {
		t.Errorf("expected DriverName 'e1000e', got %q", result.DriverName)
	}
}

// TestTestResultCrossoverCable verifies TestResult for crossover cable.
func TestTestResultCrossoverCable(t *testing.T) {
	result := cable.TestResult{
		Supported:   true,
		Status:      cable.StatusOK,
		Faults:      []string{},
		WiringStd:   cable.Wiring568A,
		Pinout:      cable.Get568APinout(),
		IsCrossover: true,
	}

	if !result.Supported {
		t.Error("expected Supported true")
	}
	if result.Status != cable.StatusOK {
		t.Errorf("expected Status OK, got %v", result.Status)
	}
	if len(result.Faults) != 0 {
		t.Errorf("expected 0 faults, got %d", len(result.Faults))
	}
	if !result.IsCrossover {
		t.Error("expected IsCrossover true")
	}
	if result.WiringStd != cable.Wiring568A {
		t.Errorf("expected WiringStd 568A, got %v", result.WiringStd)
	}
	if len(result.Pinout) != 8 {
		t.Errorf("expected 8 pinout entries, got %d", len(result.Pinout))
	}
}

// TestAllStatusCrosstalkAndSplitPair verifies the additional status constants.
func TestAllStatusCrosstalkAndSplitPair(t *testing.T) {
	if cable.StatusCrosstalk != "crosstalk" {
		t.Errorf("expected StatusCrosstalk = 'crosstalk', got %q", cable.StatusCrosstalk)
	}
	if cable.StatusSplitPair != "split_pair" {
		t.Errorf("expected StatusSplitPair = 'split_pair', got %q", cable.StatusSplitPair)
	}
}

// Test568ADiffers568B ensures T568A and T568B are actually different.
func Test568ADiffers568B(t *testing.T) {
	pinoutA := cable.Get568APinout()
	pinoutB := cable.Get568BPinout()

	// At least pins 1, 2, 3, and 6 should differ between standards.
	if pinoutA[0].Color == pinoutB[0].Color {
		t.Error("pin 1 should differ between 568A and 568B")
	}
	if pinoutA[1].Color == pinoutB[1].Color {
		t.Error("pin 2 should differ between 568A and 568B")
	}
	if pinoutA[2].Color == pinoutB[2].Color {
		t.Error("pin 3 should differ between 568A and 568B")
	}
	if pinoutA[5].Color == pinoutB[5].Color {
		t.Error("pin 6 should differ between 568A and 568B")
	}

	// Pins 4, 5, 7, 8 should be the same (blue and brown pairs).
	if pinoutA[3].Color != pinoutB[3].Color {
		t.Error("pin 4 should be same between 568A and 568B")
	}
	if pinoutA[4].Color != pinoutB[4].Color {
		t.Error("pin 5 should be same between 568A and 568B")
	}
	if pinoutA[6].Color != pinoutB[6].Color {
		t.Error("pin 7 should be same between 568A and 568B")
	}
	if pinoutA[7].Color != pinoutB[7].Color {
		t.Error("pin 8 should be same between 568A and 568B")
	}
}

// TestEmptyInterfaceName tests tester with empty interface name.
func TestEmptyInterfaceName(t *testing.T) {
	tester := cable.NewTester("")

	if tester.TesterInterfaceName() != "" {
		t.Errorf("expected empty interfaceName, got %q", tester.TesterInterfaceName())
	}

	// Should not panic.
	_ = tester.IsSupported()
	result := tester.Test()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestSpecialCharacterInterfaceName tests interface names with special characters.
func TestSpecialCharacterInterfaceName(t *testing.T) {
	interfaces := []string{
		"eth0.100",   // VLAN interface
		"br-docker0", // Docker bridge
		"veth123abc", // Container veth
		"tap0",       // TAP interface
	}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			tester := cable.NewTester(iface)
			if tester.TesterInterfaceName() != iface {
				t.Errorf("expected %q, got %q", iface, tester.TesterInterfaceName())
			}

			// Should not panic even with unusual names.
			_ = tester.IsSupported()
			result := tester.Test()
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}
