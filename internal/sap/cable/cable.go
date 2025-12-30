// Package cable provides TDR cable testing functionality.
package cable

import (
	"runtime"
	"sync"
)

// Status represents the cable test status.
type Status string

// Cable test status constants indicating test results.
const (
	StatusOK                Status = "ok"
	StatusOpen              Status = "open"
	StatusShort             Status = "short"
	StatusImpedanceMismatch Status = "impedance_mismatch"
	StatusCrosstalk         Status = "crosstalk"
	StatusSplitPair         Status = "split_pair"
	StatusUnknown           Status = "unknown"
)

// WiringStandard represents the Ethernet wiring standard.
type WiringStandard string

// Ethernet wiring standards for RJ45 connectors.
const (
	// Wiring568A is the T568A standard, used in residential and crossover cables.
	Wiring568A WiringStandard = "568A"
	// Wiring568B is the T568B standard, most common in commercial installations.
	Wiring568B WiringStandard = "568B"
)

// PairResult contains TDR test results for a single twisted pair.
type PairResult struct {
	Pair       string   `json:"pair"`               // "1-2", "3-6", "4-5", "7-8"
	PairLetter string   `json:"pairLetter"`         // "A", "B", "C", "D" (per TIA standard)
	Status     Status   `json:"status"`             // ok, open, short, etc.
	LengthM    *float64 `json:"lengthM,omitempty"`  // Length or distance to fault in meters
	LengthFt   *float64 `json:"lengthFt,omitempty"` // Length or distance to fault in feet
}

// WirePinout represents the color-to-pin mapping for a wiring standard.
type WirePinout struct {
	Pin   int    `json:"pin"`
	Color string `json:"color"`
	Pair  string `json:"pair"` // Which pair this pin belongs to
}

// TestResult contains the cable test results.
type TestResult struct {
	Supported   bool           `json:"supported"`
	Status      Status         `json:"status"`                // Overall status
	Length      *float64       `json:"length,omitempty"`      // meters (overall or shortest)
	LengthFt    *float64       `json:"lengthFt,omitempty"`    // feet
	Pairs       []PairResult   `json:"pairs,omitempty"`       // Per-pair results
	Faults      []string       `json:"faults"`                // Detected faults
	WiringStd   WiringStandard `json:"wiringStandard"`        // Selected display standard
	Pinout      []WirePinout   `json:"pinout,omitempty"`      // Pin-to-color mapping
	IsCrossover bool           `json:"isCrossover,omitempty"` // True if crossover cable detected
	DriverName  string         `json:"driverName,omitempty"`  // NIC driver for reference
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

// Get568APinout returns the T568A wiring standard pin-to-color mapping.
// T568A is primarily used in residential installations and crossover cables.
func Get568APinout() []WirePinout {
	return []WirePinout{
		{Pin: 1, Color: "White/Green", Pair: "3-6"},
		{Pin: 2, Color: "Green", Pair: "3-6"},
		{Pin: 3, Color: "White/Orange", Pair: "1-2"},
		{Pin: 4, Color: "Blue", Pair: "4-5"},
		{Pin: 5, Color: "White/Blue", Pair: "4-5"},
		{Pin: 6, Color: "Orange", Pair: "1-2"},
		{Pin: 7, Color: "White/Brown", Pair: "7-8"},
		{Pin: 8, Color: "Brown", Pair: "7-8"},
	}
}

// Get568BPinout returns the T568B wiring standard pin-to-color mapping.
// T568B is the most common standard in commercial installations.
func Get568BPinout() []WirePinout {
	return []WirePinout{
		{Pin: 1, Color: "White/Orange", Pair: "1-2"},
		{Pin: 2, Color: "Orange", Pair: "1-2"},
		{Pin: 3, Color: "White/Green", Pair: "3-6"},
		{Pin: 4, Color: "Blue", Pair: "4-5"},
		{Pin: 5, Color: "White/Blue", Pair: "4-5"},
		{Pin: 6, Color: "Green", Pair: "3-6"},
		{Pin: 7, Color: "White/Brown", Pair: "7-8"},
		{Pin: 8, Color: "Brown", Pair: "7-8"},
	}
}

// GetPinout returns the pinout for the specified wiring standard.
func GetPinout(std WiringStandard) []WirePinout {
	switch std {
	case Wiring568A:
		return Get568APinout()
	case Wiring568B:
		return Get568BPinout()
	default:
		return Get568BPinout() // Default to 568B (most common)
	}
}

// MetersToFeet converts meters to feet.
func MetersToFeet(m float64) float64 {
	return m * 3.28084
}

// GetPairInfo returns pair identification info for Ethernet.
// Pairs are named A-D per TIA standard, with pin numbers per 10/100 and GigE usage.
func GetPairInfo() []struct {
	Letter   string
	Pins     string
	Usage10  string
	UsageGig string
} {
	return []struct {
		Letter   string
		Pins     string
		Usage10  string
		UsageGig string
	}{
		{"A", "1-2", "TX+/TX-", "BI_DA+/BI_DA-"},
		{"B", "3-6", "RX+/RX-", "BI_DB+/BI_DB-"},
		{"C", "4-5", "Unused", "BI_DC+/BI_DC-"},
		{"D", "7-8", "Unused", "BI_DD+/BI_DD-"},
	}
}
