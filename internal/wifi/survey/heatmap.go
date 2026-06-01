package survey

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"
)

// HeatmapType defines the type of heatmap to generate.
type HeatmapType string

// Heatmap type constants for different visualization modes.
const (
	HeatmapRSSI         HeatmapType = "rssi"         // Signal strength.
	HeatmapSNR          HeatmapType = "snr"          // Signal-to-noise ratio.
	HeatmapDensity      HeatmapType = "density"      // AP density.
	HeatmapInterference HeatmapType = "interference" // Co-channel interference.
	HeatmapDownload     HeatmapType = "download"     // Download speed.
	HeatmapUpload       HeatmapType = "upload"       // Upload speed.
)

// Heatmap type alias constants for user-friendly names.
const (
	HeatmapAliasSignal    = "signal"    // Alias for RSSI.
	HeatmapAliasCochannel = "cochannel" // Alias for interference.
)

// Heatmap rendering constants.
const (
	// defaultHeatmapCellSize is the default size of each grid cell in pixels.
	defaultHeatmapCellSize = 10

	// defaultHeatmapOpacity is the default opacity for heatmap overlay (0-255).
	defaultHeatmapOpacity = 180

	// defaultIDWPowerHeatmap is the default IDW power parameter for heatmap interpolation.
	defaultIDWPowerHeatmap = 2.0

	// heatmapPaddingPixels is the padding added to calculated dimensions from sample points.
	heatmapPaddingPixels = 50
)

// Throughput color scale thresholds in Mbps.
const (
	throughputMin       = 0   // Minimum throughput (red).
	throughputSlow      = 50  // Slow throughput (orange).
	throughputModerate  = 100 // Moderate throughput (yellow).
	throughputGood      = 200 // Good throughput (light green).
	throughputExcellent = 500 // Excellent throughput (green).
)

// Sample point marker rendering constants.
const (
	// markerSizePixels is the radius of sample point markers.
	markerSizePixels = 4

	// markerCenterPixels is the radius of the center dot in sample markers.
	markerCenterPixels = 1
)

// HeatmapConfig contains configuration for heatmap generation.
type HeatmapConfig struct {
	Type          HeatmapType         // Type of heatmap to generate
	CellSize      int                 // Size of each grid cell in pixels (default: 10)
	Opacity       uint8               // Heatmap opacity 0-255 (default: 180)
	Method        InterpolationMethod // Interpolation method (default: IDW)
	Power         float64             // IDW power parameter (default: 2.0)
	ShowGrid      bool                // Overlay grid lines
	ShowSamples   bool                // Show sample point markers
	BlendWithPlan bool                // Blend with floor plan image
}

// HeatmapResult contains the generated heatmap.
type HeatmapResult struct {
	Image       []byte    `json:"image"`        // PNG image data
	ImageBase64 string    `json:"image_base64"` // Base64-encoded PNG
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Type        string    `json:"type"`
	Stats       GridStats `json:"stats"`
	Generated   time.Time `json:"generated"`
	SampleCount int       `json:"sample_count"`
}

// DefaultHeatmapConfig returns default configuration.
func DefaultHeatmapConfig() HeatmapConfig {
	return HeatmapConfig{
		Type:          HeatmapRSSI,
		CellSize:      defaultHeatmapCellSize,
		Opacity:       defaultHeatmapOpacity,
		Method:        MethodIDW,
		Power:         defaultIDWPowerHeatmap,
		ShowGrid:      false,
		ShowSamples:   true,
		BlendWithPlan: true,
	}
}

// GenerateHeatmap creates a heatmap from survey samples.
func GenerateHeatmap(survey *Survey, config HeatmapConfig) (*HeatmapResult, error) {
	if survey == nil {
		return nil, errors.New("survey is nil")
	}

	// Determine dimensions
	width, height := getHeatmapDimensions(survey)
	if width == 0 || height == 0 {
		return nil, errors.New("invalid dimensions: floor plan required")
	}

	// Apply defaults
	if config.CellSize <= 0 {
		config.CellSize = defaultHeatmapCellSize
	}
	if config.Opacity == 0 {
		config.Opacity = defaultHeatmapOpacity
	}
	if config.Power <= 0 {
		config.Power = defaultIDWPowerHeatmap
	}

	// Extract sample values for the requested type
	valueType := mapHeatmapTypeToValueType(config.Type)
	samples := ExtractSamplesFromSurvey(survey, valueType)

	if len(samples) == 0 {
		return nil, fmt.Errorf("no samples found for heatmap type: %s", config.Type)
	}

	// Create interpolator
	interpolator := NewInterpolator(samples)
	interpolator.Method = config.Method
	interpolator.Power = config.Power

	// Generate interpolated grid
	grid := interpolator.InterpolateGrid(width, height, config.CellSize)
	stats := CalculateGridStats(grid)

	// Get color scale
	colorScale := getColorScaleForType(config.Type)

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with heatmap colors
	renderHeatmapToImage(img, grid, config.CellSize, &colorScale, config.Opacity)

	// Optionally show sample points
	if config.ShowSamples {
		renderSamplePoints(img, samples)
	}

	// Optionally show grid
	if config.ShowGrid {
		renderGrid(img, config.CellSize)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	imageData := buf.Bytes()

	return &HeatmapResult{
		Image:       imageData,
		ImageBase64: base64.StdEncoding.EncodeToString(imageData),
		Width:       width,
		Height:      height,
		Type:        string(config.Type),
		Stats:       stats,
		Generated:   time.Now(),
		SampleCount: len(samples),
	}, nil
}

// getHeatmapDimensions returns the dimensions for the heatmap.
// Uses the active floor's floor plan in multi-floor surveys.
func getHeatmapDimensions(survey *Survey) (int, int) {
	// Check active floor first (multi-floor support)
	if activeFloor := survey.GetActiveFloor(); activeFloor != nil {
		if activeFloor.FloorPlan != nil && activeFloor.FloorPlan.Width > 0 &&
			activeFloor.FloorPlan.Height > 0 {
			return activeFloor.FloorPlan.Width, activeFloor.FloorPlan.Height
		}
	}

	// Legacy fallback: check survey-level floor plan
	if survey.FloorPlan != nil && survey.FloorPlan.Width > 0 && survey.FloorPlan.Height > 0 {
		return survey.FloorPlan.Width, survey.FloorPlan.Height
	}

	// Fallback: calculate from sample points
	allSamples := survey.GetAllSamples()
	if len(allSamples) == 0 {
		return 0, 0
	}

	var maxX, maxY int
	for _, s := range allSamples {
		if s.X > maxX {
			maxX = s.X
		}
		if s.Y > maxY {
			maxY = s.Y
		}
	}

	// Add padding
	return maxX + heatmapPaddingPixels, maxY + heatmapPaddingPixels
}

// mapHeatmapTypeToValueType converts heatmap type to value extraction key.
func mapHeatmapTypeToValueType(ht HeatmapType) string {
	switch ht {
	case HeatmapRSSI,
		HeatmapSNR,
		HeatmapDensity,
		HeatmapInterference,
		HeatmapDownload,
		HeatmapUpload:
		return string(ht)
	default:
		return string(HeatmapRSSI)
	}
}

// getColorScaleForType returns the appropriate color scale for a heatmap type.
func getColorScaleForType(ht HeatmapType) ColorScale {
	switch ht {
	case HeatmapRSSI:
		return GetRSSIColorScale()
	case HeatmapSNR:
		return GetSNRColorScale()
	case HeatmapDensity:
		return GetAPDensityColorScale()
	case HeatmapInterference:
		return GetInterferenceColorScale()
	case HeatmapDownload, HeatmapUpload:
		// Create a throughput scale (0-500 Mbps).
		// Similar structure to other scales but with throughput-specific values.
		return ColorScale{
			Name:   "throughput",
			MinVal: throughputMin,
			MaxVal: throughputExcellent,
			Stops: []ColorStop{
				// Red (very slow)
				{Value: throughputMin, Color: color.RGBA{
					R: colorChannelDarkRed,
					G: colorChannelWarningOrange,
					B: colorChannelBootstrapRed,
					A: colorChannelOpaque,
				}},
				// Orange (slow)
				{Value: throughputSlow, Color: color.RGBA{
					R: colorChannelFull, G: colorChannelMidGray, B: 0, A: colorChannelOpaque,
				}},
				// Yellow (moderate)
				{Value: throughputModerate, Color: color.RGBA{
					R: colorChannelFull, G: colorChannelYellowGreen, B: colorChannelYellowBlue, A: colorChannelOpaque,
				}},
				// Light green (good)
				{Value: throughputGood, Color: color.RGBA{
					R: colorChannelMediumGreen,
					G: colorChannelLightGreen,
					B: colorChannelMediumGreen,
					A: colorChannelOpaque,
				}},
				// Green (excellent)
				{Value: throughputExcellent, Color: color.RGBA{
					R: colorChannelAccentGreen,
					G: colorChannelDarkGreen,
					B: colorChannelBootstrapRed,
					A: colorChannelOpaque,
				}},
			},
		}
	default:
		return GetRSSIColorScale()
	}
}

// renderHeatmapToImage fills the image with interpolated heatmap colors.
func renderHeatmapToImage(
	img *image.RGBA,
	grid [][]float64,
	cellSize int,
	scale *ColorScale,
	opacity uint8,
) {
	rows := len(grid)
	if rows == 0 {
		return
	}
	cols := len(grid[0])

	for row := range rows {
		for col := range cols {
			value := grid[row][col]
			baseColor := scale.GetColor(value)
			c := WithAlpha(baseColor, opacity)

			// Fill the cell
			for dy := range cellSize {
				for dx := range cellSize {
					x := col*cellSize + dx
					y := row*cellSize + dy
					if x < img.Bounds().Max.X && y < img.Bounds().Max.Y {
						img.Set(x, y, c)
					}
				}
			}
		}
	}
}

// isWithinBounds checks if a pixel coordinate is within the image bounds.
func isWithinBounds(img *image.RGBA, x, y int) bool {
	return x >= 0 && x < img.Bounds().Max.X && y >= 0 && y < img.Bounds().Max.Y
}

// setPixelSafe sets a pixel if it's within bounds.
func setPixelSafe(img *image.RGBA, x, y int, c color.Color) {
	if isWithinBounds(img, x, y) {
		img.Set(x, y, c)
	}
}

// drawFilledCircle draws a filled circle at the given center point.
func drawFilledCircle(img *image.RGBA, centerX, centerY, radius int, c color.Color) {
	radiusSq := radius * radius
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radiusSq {
				setPixelSafe(img, centerX+dx, centerY+dy, c)
			}
		}
	}
}

// renderSamplePoints draws markers at sample locations.
func renderSamplePoints(img *image.RGBA, samples []SampleValue) {
	markerColor := color.RGBA{R: 0, G: 0, B: 0, A: colorChannelOpaque}
	centerColor := color.RGBA{R: colorChannelFull, G: colorChannelFull, B: colorChannelFull, A: colorChannelOpaque}
	markerSize := markerSizePixels
	centerSize := markerCenterPixels

	for _, sample := range samples {
		x := int(sample.Point.X)
		y := int(sample.Point.Y)

		// Draw outer marker circle (black)
		drawFilledCircle(img, x, y, markerSize, markerColor)

		// Draw center dot (white)
		drawFilledCircle(img, x, y, centerSize, centerColor)
	}
}

// renderGrid draws grid lines on the image.
func renderGrid(img *image.RGBA, cellSize int) {
	gridColor := color.RGBA{
		R: colorChannelGridGray,
		G: colorChannelGridGray,
		B: colorChannelGridGray,
		A: colorChannelGridAlpha,
	}
	maxX := img.Bounds().Max.X
	maxY := img.Bounds().Max.Y

	// Vertical lines
	for x := 0; x < maxX; x += cellSize {
		for y := range maxY {
			img.Set(x, y, gridColor)
		}
	}

	// Horizontal lines
	for y := 0; y < maxY; y += cellSize {
		for x := range maxX {
			img.Set(x, y, gridColor)
		}
	}
}

// GenerateHeatmap generates a heatmap for the specified survey.
func (m *Manager) GenerateHeatmap(surveyID string, config HeatmapConfig) (*HeatmapResult, error) {
	survey, err := m.GetSurvey(surveyID)
	if err != nil {
		return nil, err
	}

	return GenerateHeatmap(survey, config)
}

// ParseHeatmapType parses a string to HeatmapType.
// Accepts both constant values and user-friendly aliases.
func ParseHeatmapType(s string) HeatmapType {
	switch strings.ToLower(s) {
	case string(HeatmapRSSI), HeatmapAliasSignal:
		return HeatmapRSSI
	case string(HeatmapSNR):
		return HeatmapSNR
	case string(HeatmapDensity), "ap_density":
		return HeatmapDensity
	case string(HeatmapInterference), HeatmapAliasCochannel:
		return HeatmapInterference
	case string(HeatmapDownload):
		return HeatmapDownload
	case string(HeatmapUpload):
		return HeatmapUpload
	default:
		return HeatmapRSSI
	}
}
