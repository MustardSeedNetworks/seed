package services_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/services"
	"github.com/krisarmstrong/seed/internal/services/cable"
	"github.com/krisarmstrong/seed/internal/services/gateway"
)

// =============================================================================
// convertCableStatus Tests - Using Actual Internal Function
// =============================================================================

// TestConvertCableStatusActualOK tests conversion of cable.StatusOK.
func TestConvertCableStatusActualOK(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusOKValue)
	if result != services.CableStatusOK {
		t.Errorf("expected CableStatusOK, got %q", result)
	}
}

// TestConvertCableStatusActualOpen tests conversion of cable.StatusOpen.
func TestConvertCableStatusActualOpen(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusOpenValue)
	if result != services.CableStatusOpen {
		t.Errorf("expected CableStatusOpen, got %q", result)
	}
}

// TestConvertCableStatusActualShort tests conversion of cable.StatusShort.
func TestConvertCableStatusActualShort(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusShortValue)
	if result != services.CableStatusShort {
		t.Errorf("expected CableStatusShort, got %q", result)
	}
}

// TestConvertCableStatusActualImpedance tests conversion of cable.StatusImpedanceMismatch.
func TestConvertCableStatusActualImpedance(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusImpedanceMismatchValue)
	if result != services.CableStatusImpedance {
		t.Errorf("expected CableStatusImpedance, got %q", result)
	}
}

// TestConvertCableStatusActualCrosstalk tests conversion of cable.StatusCrosstalk.
func TestConvertCableStatusActualCrosstalk(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusCrosstalkValue)
	if result != services.CableStatusUnknown {
		t.Errorf("expected CableStatusUnknown for crosstalk, got %q", result)
	}
}

// TestConvertCableStatusActualSplitPair tests conversion of cable.StatusSplitPair.
func TestConvertCableStatusActualSplitPair(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusSplitPairValue)
	if result != services.CableStatusUnknown {
		t.Errorf("expected CableStatusUnknown for split pair, got %q", result)
	}
}

// TestConvertCableStatusActualUnknown tests conversion of cable.StatusUnknown.
func TestConvertCableStatusActualUnknown(t *testing.T) {
	t.Parallel()
	result := services.ConvertCableStatusActual(services.CableStatusUnknownValue)
	if result != services.CableStatusUnknown {
		t.Errorf("expected CableStatusUnknown, got %q", result)
	}
}

// =============================================================================
// convertPairResults Tests - Using Actual Internal Function
// =============================================================================

// TestConvertPairResultsActualEmpty tests conversion of empty slice.
func TestConvertPairResultsActualEmpty(t *testing.T) {
	t.Parallel()
	result := services.ConvertPairResultsActual(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

// TestConvertPairResultsActualEmptySlice tests conversion of empty slice.
func TestConvertPairResultsActualEmptySlice(t *testing.T) {
	t.Parallel()
	result := services.ConvertPairResultsActual([]cable.PairResult{})
	if result != nil {
		t.Errorf("expected nil result for empty slice, got %v", result)
	}
}

// TestConvertPairResultsActualSingleWithLength tests conversion of single pair with length.
func TestConvertPairResultsActualSingleWithLength(t *testing.T) {
	t.Parallel()
	length := 25.5
	pairs := []cable.PairResult{
		services.MakeCablePairResult(services.CableStatusOKValue, &length),
	}

	result := services.ConvertPairResultsActual(pairs)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Pair != 1 {
		t.Errorf("expected pair 1, got %d", result[0].Pair)
	}
	if result[0].Length != 25.5 {
		t.Errorf("expected length 25.5, got %f", result[0].Length)
	}
	if result[0].Status != services.CableStatusOK {
		t.Errorf("expected status OK, got %q", result[0].Status)
	}
}

// TestConvertPairResultsActualFourPairs tests conversion of four pairs.
func TestConvertPairResultsActualFourPairs(t *testing.T) {
	t.Parallel()
	len25 := 25.0
	len10 := 10.0

	pairs := []cable.PairResult{
		services.MakeCablePairResult(services.CableStatusOKValue, &len25),
		services.MakeCablePairResult(services.CableStatusOpenValue, &len10),
		services.MakeCablePairResult(services.CableStatusShortValue, nil),
		services.MakeCablePairResult(services.CableStatusUnknownValue, nil),
	}

	result := services.ConvertPairResultsActual(pairs)

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
	if result[0].Status != services.CableStatusOK {
		t.Errorf("expected first pair status OK, got %q", result[0].Status)
	}
	if result[1].Status != services.CableStatusOpen {
		t.Errorf("expected second pair status Open, got %q", result[1].Status)
	}
	if result[2].Status != services.CableStatusShort {
		t.Errorf("expected third pair status Short, got %q", result[2].Status)
	}
	if result[3].Status != services.CableStatusUnknown {
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
	result := services.ConvertGatewayStatusActual(services.GatewayStatusSuccessValue)
	if result != services.HealthStatusHealthy {
		t.Errorf("expected HealthStatusHealthy, got %q", result)
	}
}

// TestConvertGatewayStatusActualWarning tests conversion of gateway.StatusWarning.
func TestConvertGatewayStatusActualWarning(t *testing.T) {
	t.Parallel()
	result := services.ConvertGatewayStatusActual(services.GatewayStatusWarningValue)
	if result != services.HealthStatusDegraded {
		t.Errorf("expected HealthStatusDegraded, got %q", result)
	}
}

// TestConvertGatewayStatusActualError tests conversion of gateway.StatusError.
func TestConvertGatewayStatusActualError(t *testing.T) {
	t.Parallel()
	result := services.ConvertGatewayStatusActual(services.GatewayStatusErrorValue)
	if result != services.HealthStatusUnhealthy {
		t.Errorf("expected HealthStatusUnhealthy, got %q", result)
	}
}

// TestConvertGatewayStatusActualUnknown tests conversion of gateway.StatusUnknown.
func TestConvertGatewayStatusActualUnknown(t *testing.T) {
	t.Parallel()
	result := services.ConvertGatewayStatusActual(services.GatewayStatusUnknownValue)
	if result != services.HealthStatusUnknown {
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
		expected services.CableStatus
	}{
		{"StatusOK", services.CableStatusOKValue, services.CableStatusOK},
		{"StatusOpen", services.CableStatusOpenValue, services.CableStatusOpen},
		{"StatusShort", services.CableStatusShortValue, services.CableStatusShort},
		{"StatusImpedanceMismatch", services.CableStatusImpedanceMismatchValue, services.CableStatusImpedance},
		{"StatusCrosstalk", services.CableStatusCrosstalkValue, services.CableStatusUnknown},
		{"StatusSplitPair", services.CableStatusSplitPairValue, services.CableStatusUnknown},
		{"StatusUnknown", services.CableStatusUnknownValue, services.CableStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := services.ConvertCableStatusActual(tt.input)
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
		expected services.HealthStatus
	}{
		{"StatusSuccess", services.GatewayStatusSuccessValue, services.HealthStatusHealthy},
		{"StatusWarning", services.GatewayStatusWarningValue, services.HealthStatusDegraded},
		{"StatusError", services.GatewayStatusErrorValue, services.HealthStatusUnhealthy},
		{"StatusUnknown", services.GatewayStatusUnknownValue, services.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := services.ConvertGatewayStatusActual(tt.input)
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
	ok := services.ConvertCableStatusActual(services.CableStatusOKValue)
	if ok != services.CableStatusOK {
		t.Errorf("OK: expected %q, got %q", services.CableStatusOK, ok)
	}

	// Open
	open := services.ConvertCableStatusActual(services.CableStatusOpenValue)
	if open != services.CableStatusOpen {
		t.Errorf("Open: expected %q, got %q", services.CableStatusOpen, open)
	}

	// Short
	short := services.ConvertCableStatusActual(services.CableStatusShortValue)
	if short != services.CableStatusShort {
		t.Errorf("Short: expected %q, got %q", services.CableStatusShort, short)
	}

	// Impedance
	impedance := services.ConvertCableStatusActual(services.CableStatusImpedanceMismatchValue)
	if impedance != services.CableStatusImpedance {
		t.Errorf("Impedance: expected %q, got %q", services.CableStatusImpedance, impedance)
	}

	// Crosstalk -> Unknown
	crosstalk := services.ConvertCableStatusActual(services.CableStatusCrosstalkValue)
	if crosstalk != services.CableStatusUnknown {
		t.Errorf("Crosstalk: expected %q, got %q", services.CableStatusUnknown, crosstalk)
	}

	// SplitPair -> Unknown
	splitPair := services.ConvertCableStatusActual(services.CableStatusSplitPairValue)
	if splitPair != services.CableStatusUnknown {
		t.Errorf("SplitPair: expected %q, got %q", services.CableStatusUnknown, splitPair)
	}

	// Unknown
	unknown := services.ConvertCableStatusActual(services.CableStatusUnknownValue)
	if unknown != services.CableStatusUnknown {
		t.Errorf("Unknown: expected %q, got %q", services.CableStatusUnknown, unknown)
	}
}

// TestConvertGatewayStatusActualAllCases directly tests convertGatewayStatus.
func TestConvertGatewayStatusActualAllCases(t *testing.T) {
	t.Parallel()

	// Success
	success := services.ConvertGatewayStatusActual(services.GatewayStatusSuccessValue)
	if success != services.HealthStatusHealthy {
		t.Errorf("Success: expected %q, got %q", services.HealthStatusHealthy, success)
	}

	// Warning
	warning := services.ConvertGatewayStatusActual(services.GatewayStatusWarningValue)
	if warning != services.HealthStatusDegraded {
		t.Errorf("Warning: expected %q, got %q", services.HealthStatusDegraded, warning)
	}

	// Error
	errStatus := services.ConvertGatewayStatusActual(services.GatewayStatusErrorValue)
	if errStatus != services.HealthStatusUnhealthy {
		t.Errorf("Error: expected %q, got %q", services.HealthStatusUnhealthy, errStatus)
	}

	// Unknown
	unknown := services.ConvertGatewayStatusActual(services.GatewayStatusUnknownValue)
	if unknown != services.HealthStatusUnknown {
		t.Errorf("Unknown: expected %q, got %q", services.HealthStatusUnknown, unknown)
	}
}

// TestConvertPairResultsActualAllCases tests convertPairResults with various inputs.
func TestConvertPairResultsActualAllCases(t *testing.T) {
	t.Parallel()

	// Nil input
	nilResult := services.ConvertPairResultsActual(nil)
	if nilResult != nil {
		t.Errorf("nil input: expected nil, got %v", nilResult)
	}

	// Single pair with length
	len25 := 25.5
	singlePair := []cable.PairResult{
		services.MakeCablePairResult(services.CableStatusOKValue, &len25),
	}
	singleResult := services.ConvertPairResultsActual(singlePair)
	if len(singleResult) != 1 {
		t.Fatalf("single pair: expected 1 result, got %d", len(singleResult))
	}
	if singleResult[0].Length != 25.5 {
		t.Errorf("single pair: expected length 25.5, got %f", singleResult[0].Length)
	}

	// Single pair without length
	noLenPair := []cable.PairResult{
		services.MakeCablePairResult(services.CableStatusOKValue, nil),
	}
	noLenResult := services.ConvertPairResultsActual(noLenPair)
	if len(noLenResult) != 1 {
		t.Fatalf("no length pair: expected 1 result, got %d", len(noLenResult))
	}
	if noLenResult[0].Length != 0 {
		t.Errorf("no length pair: expected length 0, got %f", noLenResult[0].Length)
	}
}
