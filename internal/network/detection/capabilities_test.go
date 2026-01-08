package detection_test

// Test suite for capabilities detection module.

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/network/detection"
)

func TestGetCapabilities(t *testing.T) {
	tests := []struct {
		name        string
		ifaceName   string
		wantTDR     bool
		wantDOM     bool
		description string
	}{
		{
			name:        "ethernet interface",
			ifaceName:   "eth0",
			wantTDR:     false, // macOS always returns false for TDR
			wantDOM:     false, // macOS always returns false for DOM
			description: "standard ethernet interface",
		},
		{
			name:        "wifi interface",
			ifaceName:   "wlan0",
			wantTDR:     false,
			wantDOM:     false,
			description: "wifi interface",
		},
		{
			name:        "virtual interface",
			ifaceName:   "docker0",
			wantTDR:     false,
			wantDOM:     false,
			description: "virtual interface",
		},
		{
			name:        "nonexistent interface",
			ifaceName:   "nonexistent99",
			wantTDR:     false,
			wantDOM:     false,
			description: "interface that does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := detection.GetCapabilities(tt.ifaceName)

			if caps.TDR != tt.wantTDR {
				t.Errorf("GetCapabilities(%q).TDR = %v, want %v (%s)",
					tt.ifaceName, caps.TDR, tt.wantTDR, tt.description)
			}

			if caps.DOM != tt.wantDOM {
				t.Errorf("GetCapabilities(%q).DOM = %v, want %v (%s)",
					tt.ifaceName, caps.DOM, tt.wantDOM, tt.description)
			}
		})
	}
}

func TestGetDetailedCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		ifaceName    string
		wantTDRLevel detection.TDRSupport
		wantDOMLevel detection.DOMSupport
	}{
		{
			name:         "standard interface without TDR/DOM",
			ifaceName:    "eth0",
			wantTDRLevel: detection.TDRNone,
			wantDOMLevel: detection.DOMNone,
		},
		{
			name:         "virtual interface",
			ifaceName:    "utun0",
			wantTDRLevel: detection.TDRNone,
			wantDOMLevel: detection.DOMNone,
		},
		{
			name:         "nonexistent interface",
			ifaceName:    "fake_iface_xyz",
			wantTDRLevel: detection.TDRNone,
			wantDOMLevel: detection.DOMNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := detection.GetDetailedCapabilities(tt.ifaceName)

			if caps.TDRLevel != tt.wantTDRLevel {
				t.Errorf("GetDetailedCapabilities(%q).TDRLevel = %v, want %v",
					tt.ifaceName, caps.TDRLevel, tt.wantTDRLevel)
			}

			if caps.DOMLevel != tt.wantDOMLevel {
				t.Errorf("GetDetailedCapabilities(%q).DOMLevel = %v, want %v",
					tt.ifaceName, caps.DOMLevel, tt.wantDOMLevel)
			}
		})
	}
}

func TestTDRSupportConstants(t *testing.T) {
	// Verify the TDR support level constants have expected ordinal values.
	if detection.TDRNone != 0 {
		t.Errorf("TDRNone = %d, want 0", detection.TDRNone)
	}
	if detection.TDRBasic != 1 {
		t.Errorf("TDRBasic = %d, want 1", detection.TDRBasic)
	}
	if detection.TDRAdvanced != 2 {
		t.Errorf("TDRAdvanced = %d, want 2", detection.TDRAdvanced)
	}
	if detection.TDREnterprise != 3 {
		t.Errorf("TDREnterprise = %d, want 3", detection.TDREnterprise)
	}
}

func TestDOMSupportConstants(t *testing.T) {
	// Verify the DOM support level constants have expected ordinal values.
	if detection.DOMNone != 0 {
		t.Errorf("DOMNone = %d, want 0", detection.DOMNone)
	}
	if detection.DOMBasic != 1 {
		t.Errorf("DOMBasic = %d, want 1", detection.DOMBasic)
	}
	if detection.DOMAdvanced != 2 {
		t.Errorf("DOMAdvanced = %d, want 2", detection.DOMAdvanced)
	}
	if detection.DOMFull != 3 {
		t.Errorf("DOMFull = %d, want 3", detection.DOMFull)
	}
}

func TestCapabilitiesStruct(t *testing.T) {
	// Test that Capabilities struct can be created with expected values.
	caps := detection.Capabilities{
		TDR: true,
		DOM: true,
	}

	if !caps.TDR {
		t.Error("Capabilities.TDR should be true")
	}
	if !caps.DOM {
		t.Error("Capabilities.DOM should be true")
	}

	// Test zero value.
	var zeroCaps detection.Capabilities
	if zeroCaps.TDR {
		t.Error("Zero Capabilities.TDR should be false")
	}
	if zeroCaps.DOM {
		t.Error("Zero Capabilities.DOM should be false")
	}
}

func TestDetailedCapabilitiesStruct(t *testing.T) {
	// Test that DetailedCapabilities struct can be created with expected values.
	caps := detection.DetailedCapabilities{
		TDRLevel:    detection.TDRAdvanced,
		TDRFeatures: []string{"cable_test", "distance_to_fault"},
		DOMLevel:    detection.DOMFull,
		DOMFeatures: []string{"temperature", "voltage", "tx_power", "rx_power"},
	}

	if caps.TDRLevel != detection.TDRAdvanced {
		t.Errorf("TDRLevel = %v, want TDRAdvanced", caps.TDRLevel)
	}

	if len(caps.TDRFeatures) != 2 {
		t.Errorf("TDRFeatures length = %d, want 2", len(caps.TDRFeatures))
	}

	if caps.DOMLevel != detection.DOMFull {
		t.Errorf("DOMLevel = %v, want DOMFull", caps.DOMLevel)
	}

	if len(caps.DOMFeatures) != 4 {
		t.Errorf("DOMFeatures length = %d, want 4", len(caps.DOMFeatures))
	}
}
