package sap_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/sap"
	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
)

// =============================================================================
// convertCableStatus Tests - Using Actual Internal Function
// =============================================================================

// TestConvertCableStatusActualOK tests conversion of cable.StatusOK.
func TestConvertCableStatusActualOK(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusOKValue)
	if result != sap.CableStatusOK {
		t.Errorf("expected CableStatusOK, got %q", result)
	}
}

// TestConvertCableStatusActualOpen tests conversion of cable.StatusOpen.
func TestConvertCableStatusActualOpen(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusOpenValue)
	if result != sap.CableStatusOpen {
		t.Errorf("expected CableStatusOpen, got %q", result)
	}
}

// TestConvertCableStatusActualShort tests conversion of cable.StatusShort.
func TestConvertCableStatusActualShort(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusShortValue)
	if result != sap.CableStatusShort {
		t.Errorf("expected CableStatusShort, got %q", result)
	}
}

// TestConvertCableStatusActualImpedance tests conversion of cable.StatusImpedanceMismatch.
func TestConvertCableStatusActualImpedance(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusImpedanceMismatchValue)
	if result != sap.CableStatusImpedance {
		t.Errorf("expected CableStatusImpedance, got %q", result)
	}
}

// TestConvertCableStatusActualCrosstalk tests conversion of cable.StatusCrosstalk.
func TestConvertCableStatusActualCrosstalk(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusCrosstalkValue)
	if result != sap.CableStatusUnknown {
		t.Errorf("expected CableStatusUnknown for crosstalk, got %q", result)
	}
}

// TestConvertCableStatusActualSplitPair tests conversion of cable.StatusSplitPair.
func TestConvertCableStatusActualSplitPair(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusSplitPairValue)
	if result != sap.CableStatusUnknown {
		t.Errorf("expected CableStatusUnknown for split pair, got %q", result)
	}
}

// TestConvertCableStatusActualUnknown tests conversion of cable.StatusUnknown.
func TestConvertCableStatusActualUnknown(t *testing.T) {
	t.Parallel()
	result := sap.ConvertCableStatusActual(sap.CableStatusUnknownValue)
	if result != sap.CableStatusUnknown {
		t.Errorf("expected CableStatusUnknown, got %q", result)
	}
}

// =============================================================================
// convertPairResults Tests - Using Actual Internal Function
// =============================================================================

// TestConvertPairResultsActualEmpty tests conversion of empty slice.
func TestConvertPairResultsActualEmpty(t *testing.T) {
	t.Parallel()
	result := sap.ConvertPairResultsActual(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

// TestConvertPairResultsActualEmptySlice tests conversion of empty slice.
func TestConvertPairResultsActualEmptySlice(t *testing.T) {
	t.Parallel()
	result := sap.ConvertPairResultsActual([]cable.PairResult{})
	if result != nil {
		t.Errorf("expected nil result for empty slice, got %v", result)
	}
}

// TestConvertPairResultsActualSingleWithLength tests conversion of single pair with length.
func TestConvertPairResultsActualSingleWithLength(t *testing.T) {
	t.Parallel()
	length := 25.5
	pairs := []cable.PairResult{
		sap.MakeCablePairResult(sap.CableStatusOKValue, &length),
	}

	result := sap.ConvertPairResultsActual(pairs)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Pair != 1 {
		t.Errorf("expected pair 1, got %d", result[0].Pair)
	}
	if result[0].Length != 25.5 {
		t.Errorf("expected length 25.5, got %f", result[0].Length)
	}
	if result[0].Status != sap.CableStatusOK {
		t.Errorf("expected status OK, got %q", result[0].Status)
	}
}

// TestConvertPairResultsActualFourPairs tests conversion of four pairs.
func TestConvertPairResultsActualFourPairs(t *testing.T) {
	t.Parallel()
	len25 := 25.0
	len10 := 10.0

	pairs := []cable.PairResult{
		sap.MakeCablePairResult(sap.CableStatusOKValue, &len25),
		sap.MakeCablePairResult(sap.CableStatusOpenValue, &len10),
		sap.MakeCablePairResult(sap.CableStatusShortValue, nil),
		sap.MakeCablePairResult(sap.CableStatusUnknownValue, nil),
	}

	result := sap.ConvertPairResultsActual(pairs)

	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	// Check pair numbers are 1-indexed
	for i := range 4 {
		if result[i].Pair != i+1 {
			t.Errorf("expected pair %d, got %d", i+1, result[i].Pair)
		}
	}

	// Check statuses
	if result[0].Status != sap.CableStatusOK {
		t.Errorf("expected first pair status OK, got %q", result[0].Status)
	}
	if result[1].Status != sap.CableStatusOpen {
		t.Errorf("expected second pair status Open, got %q", result[1].Status)
	}
	if result[2].Status != sap.CableStatusShort {
		t.Errorf("expected third pair status Short, got %q", result[2].Status)
	}
	if result[3].Status != sap.CableStatusUnknown {
		t.Errorf("expected fourth pair status Unknown, got %q", result[3].Status)
	}

	// Check lengths
	if result[0].Length != 25.0 {
		t.Errorf("expected first pair length 25.0, got %f", result[0].Length)
	}
	if result[1].Length != 10.0 {
		t.Errorf("expected second pair length 10.0, got %f", result[1].Length)
	}
	if result[2].Length != 0 {
		t.Errorf("expected third pair length 0 (nil), got %f", result[2].Length)
	}
	if result[3].Length != 0 {
		t.Errorf("expected fourth pair length 0 (nil), got %f", result[3].Length)
	}
}

// =============================================================================
// convertGatewayStatus Tests - Using Actual Internal Function
// =============================================================================

// TestConvertGatewayStatusActualSuccess tests conversion of gateway.StatusSuccess.
func TestConvertGatewayStatusActualSuccess(t *testing.T) {
	t.Parallel()
	result := sap.ConvertGatewayStatusActual(sap.GatewayStatusSuccessValue)
	if result != sap.HealthStatusHealthy {
		t.Errorf("expected HealthStatusHealthy, got %q", result)
	}
}

// TestConvertGatewayStatusActualWarning tests conversion of gateway.StatusWarning.
func TestConvertGatewayStatusActualWarning(t *testing.T) {
	t.Parallel()
	result := sap.ConvertGatewayStatusActual(sap.GatewayStatusWarningValue)
	if result != sap.HealthStatusDegraded {
		t.Errorf("expected HealthStatusDegraded, got %q", result)
	}
}

// TestConvertGatewayStatusActualError tests conversion of gateway.StatusError.
func TestConvertGatewayStatusActualError(t *testing.T) {
	t.Parallel()
	result := sap.ConvertGatewayStatusActual(sap.GatewayStatusErrorValue)
	if result != sap.HealthStatusUnhealthy {
		t.Errorf("expected HealthStatusUnhealthy, got %q", result)
	}
}

// TestConvertGatewayStatusActualUnknown tests conversion of gateway.StatusUnknown.
func TestConvertGatewayStatusActualUnknown(t *testing.T) {
	t.Parallel()
	result := sap.ConvertGatewayStatusActual(sap.GatewayStatusUnknownValue)
	if result != sap.HealthStatusUnknown {
		t.Errorf("expected HealthStatusUnknown, got %q", result)
	}
}

// =============================================================================
// Table-Driven Tests for Actual Internal Functions
// =============================================================================

// TestConvertCableStatusActualTableDriven tests all cable status conversions.
func TestConvertCableStatusActualTableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    cable.Status
		expected sap.CableStatus
	}{
		{"StatusOK", sap.CableStatusOKValue, sap.CableStatusOK},
		{"StatusOpen", sap.CableStatusOpenValue, sap.CableStatusOpen},
		{"StatusShort", sap.CableStatusShortValue, sap.CableStatusShort},
		{"StatusImpedanceMismatch", sap.CableStatusImpedanceMismatchValue, sap.CableStatusImpedance},
		{"StatusCrosstalk", sap.CableStatusCrosstalkValue, sap.CableStatusUnknown},
		{"StatusSplitPair", sap.CableStatusSplitPairValue, sap.CableStatusUnknown},
		{"StatusUnknown", sap.CableStatusUnknownValue, sap.CableStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sap.ConvertCableStatusActual(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertCableStatusActual(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConvertGatewayStatusActualTableDriven tests all gateway status conversions.
func TestConvertGatewayStatusActualTableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    gateway.Status
		expected sap.HealthStatus
	}{
		{"StatusSuccess", sap.GatewayStatusSuccessValue, sap.HealthStatusHealthy},
		{"StatusWarning", sap.GatewayStatusWarningValue, sap.HealthStatusDegraded},
		{"StatusError", sap.GatewayStatusErrorValue, sap.HealthStatusUnhealthy},
		{"StatusUnknown", sap.GatewayStatusUnknownValue, sap.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sap.ConvertGatewayStatusActual(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertGatewayStatusActual(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Direct Conversion Tests
// =============================================================================

// TestConvertCableStatusActualAllCases directly tests convertCableStatus.
func TestConvertCableStatusActualAllCases(t *testing.T) {
	t.Parallel()

	// OK
	ok := sap.ConvertCableStatusActual(sap.CableStatusOKValue)
	if ok != sap.CableStatusOK {
		t.Errorf("OK: expected %q, got %q", sap.CableStatusOK, ok)
	}

	// Open
	open := sap.ConvertCableStatusActual(sap.CableStatusOpenValue)
	if open != sap.CableStatusOpen {
		t.Errorf("Open: expected %q, got %q", sap.CableStatusOpen, open)
	}

	// Short
	short := sap.ConvertCableStatusActual(sap.CableStatusShortValue)
	if short != sap.CableStatusShort {
		t.Errorf("Short: expected %q, got %q", sap.CableStatusShort, short)
	}

	// Impedance
	impedance := sap.ConvertCableStatusActual(sap.CableStatusImpedanceMismatchValue)
	if impedance != sap.CableStatusImpedance {
		t.Errorf("Impedance: expected %q, got %q", sap.CableStatusImpedance, impedance)
	}

	// Crosstalk -> Unknown
	crosstalk := sap.ConvertCableStatusActual(sap.CableStatusCrosstalkValue)
	if crosstalk != sap.CableStatusUnknown {
		t.Errorf("Crosstalk: expected %q, got %q", sap.CableStatusUnknown, crosstalk)
	}

	// SplitPair -> Unknown
	splitPair := sap.ConvertCableStatusActual(sap.CableStatusSplitPairValue)
	if splitPair != sap.CableStatusUnknown {
		t.Errorf("SplitPair: expected %q, got %q", sap.CableStatusUnknown, splitPair)
	}

	// Unknown
	unknown := sap.ConvertCableStatusActual(sap.CableStatusUnknownValue)
	if unknown != sap.CableStatusUnknown {
		t.Errorf("Unknown: expected %q, got %q", sap.CableStatusUnknown, unknown)
	}
}

// TestConvertGatewayStatusActualAllCases directly tests convertGatewayStatus.
func TestConvertGatewayStatusActualAllCases(t *testing.T) {
	t.Parallel()

	// Success
	success := sap.ConvertGatewayStatusActual(sap.GatewayStatusSuccessValue)
	if success != sap.HealthStatusHealthy {
		t.Errorf("Success: expected %q, got %q", sap.HealthStatusHealthy, success)
	}

	// Warning
	warning := sap.ConvertGatewayStatusActual(sap.GatewayStatusWarningValue)
	if warning != sap.HealthStatusDegraded {
		t.Errorf("Warning: expected %q, got %q", sap.HealthStatusDegraded, warning)
	}

	// Error
	errStatus := sap.ConvertGatewayStatusActual(sap.GatewayStatusErrorValue)
	if errStatus != sap.HealthStatusUnhealthy {
		t.Errorf("Error: expected %q, got %q", sap.HealthStatusUnhealthy, errStatus)
	}

	// Unknown
	unknown := sap.ConvertGatewayStatusActual(sap.GatewayStatusUnknownValue)
	if unknown != sap.HealthStatusUnknown {
		t.Errorf("Unknown: expected %q, got %q", sap.HealthStatusUnknown, unknown)
	}
}

// TestConvertPairResultsActualAllCases tests convertPairResults with various inputs.
func TestConvertPairResultsActualAllCases(t *testing.T) {
	t.Parallel()

	// Nil input
	nilResult := sap.ConvertPairResultsActual(nil)
	if nilResult != nil {
		t.Errorf("nil input: expected nil, got %v", nilResult)
	}

	// Single pair with length
	len25 := 25.5
	singlePair := []cable.PairResult{
		sap.MakeCablePairResult(sap.CableStatusOKValue, &len25),
	}
	singleResult := sap.ConvertPairResultsActual(singlePair)
	if len(singleResult) != 1 {
		t.Fatalf("single pair: expected 1 result, got %d", len(singleResult))
	}
	if singleResult[0].Length != 25.5 {
		t.Errorf("single pair: expected length 25.5, got %f", singleResult[0].Length)
	}

	// Single pair without length
	noLenPair := []cable.PairResult{
		sap.MakeCablePairResult(sap.CableStatusOKValue, nil),
	}
	noLenResult := sap.ConvertPairResultsActual(noLenPair)
	if len(noLenResult) != 1 {
		t.Fatalf("no length pair: expected 1 result, got %d", len(noLenResult))
	}
	if noLenResult[0].Length != 0 {
		t.Errorf("no length pair: expected length 0, got %f", noLenResult[0].Length)
	}
}
