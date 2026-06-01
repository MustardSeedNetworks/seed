package survey

import (
	"image/color"
)

// Color channel constants for RGBA color definitions.
// These values represent standard 8-bit color channel intensities (0-255).
const (
	// colorChannelMidGray represents a neutral gray tone (128/255 intensity).
	colorChannelMidGray = 128

	// colorChannelDarkRed represents the red component for danger/warning colors.
	colorChannelDarkRed = 220

	// colorChannelFull represents maximum channel intensity (fully saturated).
	colorChannelFull = 255

	// colorChannelDarkGreen represents the green component for success/excellent colors.
	colorChannelDarkGreen = 167

	// colorChannelAccentGreen represents a vibrant green accent (40 out of 255).
	colorChannelAccentGreen = 40

	// colorChannelLightGreen represents the green component for light green colors.
	colorChannelLightGreen = 238

	// colorChannelMediumGreen represents a medium green intensity (144/255).
	colorChannelMediumGreen = 144

	// colorChannelWarningOrange represents the red component for orange warning colors.
	colorChannelWarningOrange = 53

	// colorChannelYellowGreen represents the green component for yellow (193/255).
	colorChannelYellowGreen = 193

	// colorChannelYellowBlue represents the blue component for yellow (7/255).
	colorChannelYellowBlue = 7

	// colorChannelCornflower represents the red component for cornflower blue (100/255).
	colorChannelCornflower = 100

	// colorChannelCornflowerGreen represents the green component for cornflower blue (149/255).
	colorChannelCornflowerGreen = 149

	// colorChannelCornflowerBlue represents the blue component for cornflower blue (237/255).
	colorChannelCornflowerBlue = 237

	// colorChannelVioletRed represents the red component for blue violet (138/255).
	colorChannelVioletRed = 138

	// colorChannelVioletGreen represents the green component for blue violet (43/255).
	colorChannelVioletGreen = 43

	// colorChannelVioletBlue represents the blue component for blue violet (226/255).
	colorChannelVioletBlue = 226

	// colorChannelLightBlue represents a very light blue tint (240/255).
	colorChannelLightBlue = 240

	// colorChannelGridGray represents the gray level for grid lines (200/255).
	colorChannelGridGray = 200

	// colorChannelGridAlpha represents the alpha for semi-transparent grid lines (100/255).
	colorChannelGridAlpha = 100

	// colorChannelOpaque represents fully opaque alpha value.
	colorChannelOpaque = 255

	// colorChannelBootstrapRed represents the blue component for Bootstrap danger red (69/255).
	colorChannelBootstrapRed = 69
)

// Signal strength scale thresholds in dBm for RSSI color scale stops.
const (
	rssiNoSignal  = -100 // Gray zone: no signal detected.
	rssiVeryPoor  = -85  // Red zone: very poor signal.
	rssiPoor      = -75  // Orange zone: poor signal.
	rssiFair      = -67  // Yellow zone: fair signal.
	rssiGood      = -55  // Light green zone: good signal.
	rssiExcellent = -30  // Green zone: excellent signal.
)

// Signal-to-noise ratio scale thresholds in dB.
const (
	snrMinimum   = 0  // Minimum SNR value (red zone).
	snrPoor      = 15 // Orange zone threshold.
	snrFair      = 25 // Yellow zone threshold.
	snrGood      = 35 // Light green zone threshold.
	snrExcellent = 50 // Maximum expected SNR (green zone).
)

// AP density scale thresholds (number of access points).
const (
	apDensityMin      = 0  // No APs visible.
	apDensityLow      = 3  // Few APs (cornflower blue).
	apDensityModerate = 8  // Moderate APs (blue violet).
	apDensityHigh     = 15 // Many APs (orange warning).
	apDensityMax      = 20 // Congested (red zone).
)

// Interference scale thresholds (number of interfering APs).
const (
	interferenceNone     = 0  // No interference (green).
	interferenceLow      = 2  // Low interference (light green).
	interferenceMild     = 4  // Mild interference (yellow).
	interferenceModerate = 6  // Moderate interference (orange).
	interferenceSevere   = 10 // Severe interference (red).
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

// GetRSSIColorScale returns the RSSI color scale.
// Maps signal strength (-100 to -30 dBm) to colors.
// Red (weak) -> Yellow (fair) -> Green (strong).
func GetRSSIColorScale() ColorScale {
	return ColorScale{
		Name:   "rssi",
		MinVal: rssiNoSignal,
		MaxVal: rssiExcellent,
		Stops: []ColorStop{
			// Gray (no signal)
			{Value: rssiNoSignal, Color: color.RGBA{
				R: colorChannelMidGray, G: colorChannelMidGray, B: colorChannelMidGray, A: colorChannelOpaque,
			}},
			// Red (very poor)
			{Value: rssiVeryPoor, Color: color.RGBA{
				R: colorChannelDarkRed,
				G: colorChannelWarningOrange,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
			// Orange (poor)
			{Value: rssiPoor, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelMidGray, B: 0, A: colorChannelOpaque,
			}},
			// Yellow (fair)
			{Value: rssiFair, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelYellowGreen, B: colorChannelYellowBlue, A: colorChannelOpaque,
			}},
			// Light green (good)
			{Value: rssiGood, Color: color.RGBA{
				R: colorChannelMediumGreen,
				G: colorChannelLightGreen,
				B: colorChannelMediumGreen,
				A: colorChannelOpaque,
			}},
			// Green (excellent)
			{Value: rssiExcellent, Color: color.RGBA{
				R: colorChannelAccentGreen,
				G: colorChannelDarkGreen,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
		},
	}
}

// GetSNRColorScale returns the SNR color scale.
// Maps signal-to-noise ratio (0 to 50 dB) to colors.
func GetSNRColorScale() ColorScale {
	return ColorScale{
		Name:   "snr",
		MinVal: snrMinimum,
		MaxVal: snrExcellent,
		Stops: []ColorStop{
			// Red
			{Value: snrMinimum, Color: color.RGBA{
				R: colorChannelDarkRed,
				G: colorChannelWarningOrange,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
			// Orange
			{Value: snrPoor, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelMidGray, B: 0, A: colorChannelOpaque,
			}},
			// Yellow
			{Value: snrFair, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelYellowGreen, B: colorChannelYellowBlue, A: colorChannelOpaque,
			}},
			// Light green
			{Value: snrGood, Color: color.RGBA{
				R: colorChannelMediumGreen,
				G: colorChannelLightGreen,
				B: colorChannelMediumGreen,
				A: colorChannelOpaque,
			}},
			// Green
			{Value: snrExcellent, Color: color.RGBA{
				R: colorChannelAccentGreen,
				G: colorChannelDarkGreen,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
		},
	}
}

// GetAPDensityColorScale returns the AP density color scale.
// Maps AP count (0 to 20+) to colors.
// Blue (few) -> Purple (moderate) -> Red (many/congested).
func GetAPDensityColorScale() ColorScale {
	return ColorScale{
		Name:   "ap_density",
		MinVal: apDensityMin,
		MaxVal: apDensityMax,
		Stops: []ColorStop{
			// Very light blue
			{Value: apDensityMin, Color: color.RGBA{
				R: colorChannelLightBlue, G: colorChannelLightBlue, B: colorChannelFull, A: colorChannelOpaque,
			}},
			// Cornflower blue
			{Value: apDensityLow, Color: color.RGBA{
				R: colorChannelCornflower,
				G: colorChannelCornflowerGreen,
				B: colorChannelCornflowerBlue,
				A: colorChannelOpaque,
			}},
			// Blue violet
			{Value: apDensityModerate, Color: color.RGBA{
				R: colorChannelVioletRed, G: colorChannelVioletGreen, B: colorChannelVioletBlue, A: colorChannelOpaque,
			}},
			// Orange
			{Value: apDensityHigh, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelMidGray, B: 0, A: colorChannelOpaque,
			}},
			// Red (congested)
			{Value: apDensityMax, Color: color.RGBA{
				R: colorChannelDarkRed,
				G: colorChannelWarningOrange,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
		},
	}
}

// GetInterferenceColorScale returns the interference color scale.
// Maps co-channel interference (0 to 10+) to colors.
// Green (none) -> Yellow -> Red (severe).
func GetInterferenceColorScale() ColorScale {
	return ColorScale{
		Name:   "interference",
		MinVal: interferenceNone,
		MaxVal: interferenceSevere,
		Stops: []ColorStop{
			// Green (no interference)
			{Value: interferenceNone, Color: color.RGBA{
				R: colorChannelAccentGreen,
				G: colorChannelDarkGreen,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
			// Light green
			{Value: interferenceLow, Color: color.RGBA{
				R: colorChannelMediumGreen,
				G: colorChannelLightGreen,
				B: colorChannelMediumGreen,
				A: colorChannelOpaque,
			}},
			// Yellow
			{Value: interferenceMild, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelYellowGreen, B: colorChannelYellowBlue, A: colorChannelOpaque,
			}},
			// Orange
			{Value: interferenceModerate, Color: color.RGBA{
				R: colorChannelFull, G: colorChannelMidGray, B: 0, A: colorChannelOpaque,
			}},
			// Red
			{Value: interferenceSevere, Color: color.RGBA{
				R: colorChannelDarkRed,
				G: colorChannelWarningOrange,
				B: colorChannelBootstrapRed,
				A: colorChannelOpaque,
			}},
		},
	}
}

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
		A: colorChannelOpaque,
	}
}

// GetColorScaleByName returns a predefined color scale by name.
// Accepts both constant values and user-friendly aliases.
func GetColorScaleByName(name string) *ColorScale {
	var scale ColorScale
	switch name {
	case string(HeatmapRSSI), HeatmapAliasSignal:
		scale = GetRSSIColorScale()
	case string(HeatmapSNR):
		scale = GetSNRColorScale()
	case string(HeatmapDensity), "ap_density":
		scale = GetAPDensityColorScale()
	case string(HeatmapInterference), HeatmapAliasCochannel:
		scale = GetInterferenceColorScale()
	default:
		scale = GetRSSIColorScale()
	}
	return &scale
}

// WithAlpha returns a copy of the color with the specified alpha value.
func WithAlpha(c color.RGBA, alpha uint8) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha}
}
