// Package detection provides intelligent network interface auto-detection.
// Capabilities module provides cross-platform abstractions for TDR and DOM
// capability detection, with platform-specific implementations.
package detection

// Capabilities represents the detected capabilities of a network interface.
type Capabilities struct {
	TDR bool // Time Domain Reflectometry for cable testing
	DOM bool // Digital Optical Monitoring for fiber optics
}

// GetCapabilities returns the capabilities for an interface.
func GetCapabilities(name string) Capabilities {
	return Capabilities{
		TDR: hasTDRCapability(name),
		DOM: hasDOMCapability(name),
	}
}

// TDRSupport describes TDR capability levels.
type TDRSupport int

const (
	TDRNone        TDRSupport = iota // No TDR support
	TDRBasic                         // Basic cable test (pass/fail)
	TDRAdvanced                      // Distance to fault, pair status
	TDREnterprise                    // Full diagnostics with waveform
)

// DOMSupport describes DOM capability levels.
type DOMSupport int

const (
	DOMNone     DOMSupport = iota // No DOM support
	DOMBasic                      // Temperature, voltage
	DOMAdvanced                   // + Tx/Rx power, laser bias
	DOMFull                       // + Thresholds, alarms
)

// DetailedCapabilities provides more granular capability information.
type DetailedCapabilities struct {
	TDRLevel    TDRSupport `json:"tdrLevel"`
	TDRFeatures []string   `json:"tdrFeatures,omitempty"`
	DOMLevel    DOMSupport `json:"domLevel"`
	DOMFeatures []string   `json:"domFeatures,omitempty"`
}

// GetDetailedCapabilities returns detailed capability information.
// This is a placeholder for future expansion with actual driver introspection.
func GetDetailedCapabilities(name string) DetailedCapabilities {
	caps := DetailedCapabilities{}

	if hasTDRCapability(name) {
		caps.TDRLevel = TDRBasic
		caps.TDRFeatures = []string{"cable_test"}
	}

	if hasDOMCapability(name) {
		caps.DOMLevel = DOMBasic
		caps.DOMFeatures = []string{"temperature", "voltage"}
	}

	return caps
}
