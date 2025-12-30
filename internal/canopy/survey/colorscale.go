// Package survey provides WiFi site survey functionality.
package survey

import (
	"image/color"
)

// ColorScale defines a gradient for mapping values to colors.
type ColorScale struct {
	Name   string      // Scale name for identification.
	Stops  []ColorStop // Gradient stops (must be sorted by value).
	MinVal float64     // Minimum expected value.
	MaxVal float64     // Maximum expected value.
}

// ColorStop represents a single point in a color gradient.
type ColorStop struct {
	Value float64    // The value at this stop.
	Color color.RGBA // The color at this stop.
}

// Predefined color scales for different visualization types.
// Each scale defines gradient stops for a specific metric range.
//
//nolint:dupl // Color scale data definitions are intentionally similar structures.
var (
	// RSSIColorScale maps signal strength (-100 to -30 dBm) to colors.
	// Red (weak) -> Yellow (fair) -> Green (strong).
	RSSIColorScale = ColorScale{
		Name:   "rssi",
		MinVal: -100,
		MaxVal: -30,
		Stops: []ColorStop{
			{Value: -100, Color: color.RGBA{R: 128, G: 128, B: 128, A: 255}}, // Gray (no signal)
			{Value: -85, Color: color.RGBA{R: 220, G: 53, B: 69, A: 255}},    // Red (very poor)
			{Value: -75, Color: color.RGBA{R: 255, G: 128, B: 0, A: 255}},    // Orange (poor)
			{Value: -67, Color: color.RGBA{R: 255, G: 193, B: 7, A: 255}},    // Yellow (fair)
			{Value: -55, Color: color.RGBA{R: 144, G: 238, B: 144, A: 255}},  // Light green (good)
			{Value: -30, Color: color.RGBA{R: 40, G: 167, B: 69, A: 255}},    // Green (excellent)
		},
	}

	// SNRColorScale maps signal-to-noise ratio (0 to 50 dB) to colors.
	SNRColorScale = ColorScale{
		Name:   "snr",
		MinVal: 0,
		MaxVal: 50,
		Stops: []ColorStop{
			{Value: 0, Color: color.RGBA{R: 220, G: 53, B: 69, A: 255}},    // Red
			{Value: 15, Color: color.RGBA{R: 255, G: 128, B: 0, A: 255}},   // Orange
			{Value: 25, Color: color.RGBA{R: 255, G: 193, B: 7, A: 255}},   // Yellow
			{Value: 35, Color: color.RGBA{R: 144, G: 238, B: 144, A: 255}}, // Light green
			{Value: 50, Color: color.RGBA{R: 40, G: 167, B: 69, A: 255}},   // Green
		},
	}

	// APDensityColorScale maps AP count (0 to 20+) to colors.
	// Blue (few) -> Purple (moderate) -> Red (many/congested).
	APDensityColorScale = ColorScale{
		Name:   "ap_density",
		MinVal: 0,
		MaxVal: 20,
		Stops: []ColorStop{
			{Value: 0, Color: color.RGBA{R: 240, G: 240, B: 255, A: 255}}, // Very light blue
			{Value: 3, Color: color.RGBA{R: 100, G: 149, B: 237, A: 255}}, // Cornflower blue
			{Value: 8, Color: color.RGBA{R: 138, G: 43, B: 226, A: 255}},  // Blue violet
			{Value: 15, Color: color.RGBA{R: 255, G: 128, B: 0, A: 255}},  // Orange
			{Value: 20, Color: color.RGBA{R: 220, G: 53, B: 69, A: 255}},  // Red (congested)
		},
	}

	// InterferenceColorScale maps co-channel interference (0 to 10+) to colors.
	// Green (none) -> Yellow -> Red (severe).
	InterferenceColorScale = ColorScale{
		Name:   "interference",
		MinVal: 0,
		MaxVal: 10,
		Stops: []ColorStop{
			{Value: 0, Color: color.RGBA{R: 40, G: 167, B: 69, A: 255}},   // Green (no interference)
			{Value: 2, Color: color.RGBA{R: 144, G: 238, B: 144, A: 255}}, // Light green
			{Value: 4, Color: color.RGBA{R: 255, G: 193, B: 7, A: 255}},   // Yellow
			{Value: 6, Color: color.RGBA{R: 255, G: 128, B: 0, A: 255}},   // Orange
			{Value: 10, Color: color.RGBA{R: 220, G: 53, B: 69, A: 255}},  // Red
		},
	}
)

// GetColor returns the interpolated color for a given value.
func (cs *ColorScale) GetColor(value float64) color.RGBA {
	// Clamp value to scale range
	if value <= cs.Stops[0].Value {
		return cs.Stops[0].Color
	}
	if value >= cs.Stops[len(cs.Stops)-1].Value {
		return cs.Stops[len(cs.Stops)-1].Color
	}

	// Find the two stops to interpolate between
	for i := range len(cs.Stops) - 1 {
		if value >= cs.Stops[i].Value && value <= cs.Stops[i+1].Value {
			return interpolateColor(cs.Stops[i], cs.Stops[i+1], value)
		}
	}

	// Fallback (shouldn't reach here)
	return cs.Stops[len(cs.Stops)-1].Color
}

// interpolateColor linearly interpolates between two color stops.
func interpolateColor(stop1, stop2 ColorStop, value float64) color.RGBA {
	// Calculate interpolation factor (0.0 to 1.0)
	t := (value - stop1.Value) / (stop2.Value - stop1.Value)

	return color.RGBA{
		R: uint8(float64(stop1.Color.R) + t*(float64(stop2.Color.R)-float64(stop1.Color.R))),
		G: uint8(float64(stop1.Color.G) + t*(float64(stop2.Color.G)-float64(stop1.Color.G))),
		B: uint8(float64(stop1.Color.B) + t*(float64(stop2.Color.B)-float64(stop1.Color.B))),
		A: 255,
	}
}

// GetColorScaleByName returns a predefined color scale by name.
// Accepts both constant values and user-friendly aliases.
func GetColorScaleByName(name string) *ColorScale {
	switch name {
	case string(HeatmapRSSI), "signal":
		return &RSSIColorScale
	case string(HeatmapSNR):
		return &SNRColorScale
	case string(HeatmapDensity), "ap_density":
		return &APDensityColorScale
	case string(HeatmapInterference), "cochannel":
		return &InterferenceColorScale
	default:
		return &RSSIColorScale
	}
}

// WithAlpha returns a copy of the color with the specified alpha value.
func WithAlpha(c color.RGBA, alpha uint8) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha}
}
