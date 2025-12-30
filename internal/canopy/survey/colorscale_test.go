// Package survey provides WiFi site survey functionality.
package survey

import (
	"image/color"
	"testing"
)

func TestColorScale_GetColor(t *testing.T) {
	tests := []struct {
		name     string
		scale    *ColorScale
		value    float64
		expected color.RGBA
	}{
		{
			name:     "RSSI at minimum",
			scale:    &RSSIColorScale,
			value:    -100,
			expected: color.RGBA{R: 128, G: 128, B: 128, A: 255}, // Gray
		},
		{
			name:     "RSSI below minimum (clamped)",
			scale:    &RSSIColorScale,
			value:    -120,
			expected: color.RGBA{R: 128, G: 128, B: 128, A: 255}, // Gray
		},
		{
			name:     "RSSI at maximum",
			scale:    &RSSIColorScale,
			value:    -30,
			expected: color.RGBA{R: 40, G: 167, B: 69, A: 255}, // Green
		},
		{
			name:     "RSSI above maximum (clamped)",
			scale:    &RSSIColorScale,
			value:    0,
			expected: color.RGBA{R: 40, G: 167, B: 69, A: 255}, // Green
		},
		{
			name:  "RSSI interpolated between stops",
			scale: &RSSIColorScale,
			value: -70, // Between -75 (orange) and -67 (yellow)
			// Should be somewhere between orange and yellow
		},
		{
			name:     "SNR at zero",
			scale:    &SNRColorScale,
			value:    0,
			expected: color.RGBA{R: 220, G: 53, B: 69, A: 255}, // Red
		},
		{
			name:     "SNR at max",
			scale:    &SNRColorScale,
			value:    50,
			expected: color.RGBA{R: 40, G: 167, B: 69, A: 255}, // Green
		},
		{
			name:     "AP density at zero",
			scale:    &APDensityColorScale,
			value:    0,
			expected: color.RGBA{R: 240, G: 240, B: 255, A: 255}, // Very light blue
		},
		{
			name:     "Interference at zero",
			scale:    &InterferenceColorScale,
			value:    0,
			expected: color.RGBA{R: 40, G: 167, B: 69, A: 255}, // Green
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scale.GetColor(tt.value)

			// For exact matches, compare exactly
			if tt.expected != (color.RGBA{}) {
				if got != tt.expected {
					t.Errorf("GetColor(%v) = %v, want %v", tt.value, got, tt.expected)
				}
			} else {
				// For interpolated values, just check it's valid
				if got.A != 255 {
					t.Errorf("GetColor(%v) alpha = %d, want 255", tt.value, got.A)
				}
			}
		})
	}
}

func TestInterpolateColor(t *testing.T) {
	stop1 := ColorStop{Value: 0, Color: color.RGBA{R: 0, G: 0, B: 0, A: 255}}
	stop2 := ColorStop{Value: 100, Color: color.RGBA{R: 100, G: 200, B: 50, A: 255}}

	tests := []struct {
		name     string
		value    float64
		expected color.RGBA
	}{
		{
			name:     "at start",
			value:    0,
			expected: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name:     "at end",
			value:    100,
			expected: color.RGBA{R: 100, G: 200, B: 50, A: 255},
		},
		{
			name:     "midpoint",
			value:    50,
			expected: color.RGBA{R: 50, G: 100, B: 25, A: 255},
		},
		{
			name:     "quarter",
			value:    25,
			expected: color.RGBA{R: 25, G: 50, B: 12, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interpolateColor(stop1, stop2, tt.value)
			if got != tt.expected {
				t.Errorf("interpolateColor(%v) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestGetColorScaleByName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ColorScale
	}{
		{
			name:     "rssi",
			input:    "rssi",
			expected: &RSSIColorScale,
		},
		{
			name:     "signal alias",
			input:    "signal",
			expected: &RSSIColorScale,
		},
		{
			name:     "snr",
			input:    "snr",
			expected: &SNRColorScale,
		},
		{
			name:     "density",
			input:    "density",
			expected: &APDensityColorScale,
		},
		{
			name:     "ap_density alias",
			input:    "ap_density",
			expected: &APDensityColorScale,
		},
		{
			name:     "interference",
			input:    "interference",
			expected: &InterferenceColorScale,
		},
		{
			name:     "cochannel alias",
			input:    "cochannel",
			expected: &InterferenceColorScale,
		},
		{
			name:     "unknown defaults to RSSI",
			input:    "unknown",
			expected: &RSSIColorScale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetColorScaleByName(tt.input)
			if got.Name != tt.expected.Name {
				t.Errorf("GetColorScaleByName(%q) = %s, want %s", tt.input, got.Name, tt.expected.Name)
			}
		})
	}
}

func TestWithAlpha(t *testing.T) {
	original := color.RGBA{R: 100, G: 150, B: 200, A: 255}

	tests := []struct {
		name     string
		alpha    uint8
		expected color.RGBA
	}{
		{
			name:     "full alpha",
			alpha:    255,
			expected: color.RGBA{R: 100, G: 150, B: 200, A: 255},
		},
		{
			name:     "half alpha",
			alpha:    128,
			expected: color.RGBA{R: 100, G: 150, B: 200, A: 128},
		},
		{
			name:     "zero alpha",
			alpha:    0,
			expected: color.RGBA{R: 100, G: 150, B: 200, A: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithAlpha(original, tt.alpha)
			if got != tt.expected {
				t.Errorf("WithAlpha(%v, %d) = %v, want %v", original, tt.alpha, got, tt.expected)
			}
		})
	}
}

func TestColorScaleProperties(t *testing.T) {
	scales := []*ColorScale{
		&RSSIColorScale,
		&SNRColorScale,
		&APDensityColorScale,
		&InterferenceColorScale,
	}

	for _, scale := range scales {
		t.Run(scale.Name, func(t *testing.T) {
			// Check that stops are sorted by value
			for i := 1; i < len(scale.Stops); i++ {
				if scale.Stops[i].Value <= scale.Stops[i-1].Value {
					t.Errorf("Stops not sorted: %v at %d, %v at %d",
						scale.Stops[i-1].Value, i-1, scale.Stops[i].Value, i)
				}
			}

			// Check min/max match first/last stops
			if scale.MinVal > scale.Stops[0].Value {
				t.Errorf("MinVal %v > first stop %v", scale.MinVal, scale.Stops[0].Value)
			}
			if scale.MaxVal < scale.Stops[len(scale.Stops)-1].Value {
				t.Errorf("MaxVal %v < last stop %v", scale.MaxVal, scale.Stops[len(scale.Stops)-1].Value)
			}
		})
	}
}
