// Package survey provides WiFi site survey functionality.
package survey

import (
	"bytes"
	"encoding/base64"
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
		CellSize:      10,
		Opacity:       180,
		Method:        MethodIDW,
		Power:         2.0,
		ShowGrid:      false,
		ShowSamples:   true,
		BlendWithPlan: true,
	}
}

// GenerateHeatmap creates a heatmap from survey samples.
func GenerateHeatmap(survey *Survey, config HeatmapConfig) (*HeatmapResult, error) {
	if survey == nil {
		return nil, fmt.Errorf("survey is nil")
	}

	// Determine dimensions
	width, height := getHeatmapDimensions(survey)
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("invalid dimensions: floor plan required")
	}

	// Apply defaults
	if config.CellSize <= 0 {
		config.CellSize = 10
	}
	if config.Opacity == 0 {
		config.Opacity = 180
	}
	if config.Power <= 0 {
		config.Power = 2.0
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
	renderHeatmapToImage(img, grid, config.CellSize, colorScale, config.Opacity)

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
func getHeatmapDimensions(survey *Survey) (width, height int) {
	// Check active floor first (multi-floor support)
	if activeFloor := survey.GetActiveFloor(); activeFloor != nil {
		if activeFloor.FloorPlan != nil && activeFloor.FloorPlan.Width > 0 && activeFloor.FloorPlan.Height > 0 {
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
	return maxX + 50, maxY + 50
}

// mapHeatmapTypeToValueType converts heatmap type to value extraction key.
func mapHeatmapTypeToValueType(ht HeatmapType) string {
	switch ht {
	case HeatmapRSSI, HeatmapSNR, HeatmapDensity, HeatmapInterference, HeatmapDownload, HeatmapUpload:
		return string(ht)
	default:
		return string(HeatmapRSSI)
	}
}

// getColorScaleForType returns the appropriate color scale for a heatmap type.
func getColorScaleForType(ht HeatmapType) *ColorScale {
	switch ht {
	case HeatmapRSSI:
		return &RSSIColorScale
	case HeatmapSNR:
		return &SNRColorScale
	case HeatmapDensity:
		return &APDensityColorScale
	case HeatmapInterference:
		return &InterferenceColorScale
	case HeatmapDownload, HeatmapUpload:
		// Create a throughput scale (0-500 Mbps).
		// Similar structure to other scales but with throughput-specific values.
		return &ColorScale{ //nolint:dupl // Intentional color scale data.
			Name:   "throughput",
			MinVal: 0,
			MaxVal: 500,
			Stops: []ColorStop{
				{Value: 0, Color: color.RGBA{R: 220, G: 53, B: 69, A: 255}},
				{Value: 50, Color: color.RGBA{R: 255, G: 128, B: 0, A: 255}},
				{Value: 100, Color: color.RGBA{R: 255, G: 193, B: 7, A: 255}},
				{Value: 200, Color: color.RGBA{R: 144, G: 238, B: 144, A: 255}},
				{Value: 500, Color: color.RGBA{R: 40, G: 167, B: 69, A: 255}},
			},
		}
	default:
		return &RSSIColorScale
	}
}

// renderHeatmapToImage fills the image with interpolated heatmap colors.
func renderHeatmapToImage(img *image.RGBA, grid [][]float64, cellSize int, scale *ColorScale, opacity uint8) {
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

// renderSamplePoints draws markers at sample locations.
func renderSamplePoints(img *image.RGBA, samples []SampleValue) {
	markerColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	markerSize := 4

	for _, sample := range samples {
		x := int(sample.Point.X)
		y := int(sample.Point.Y)

		// Draw a small circle/square marker
		for dy := -markerSize; dy <= markerSize; dy++ {
			for dx := -markerSize; dx <= markerSize; dx++ {
				// Circle check
				if dx*dx+dy*dy <= markerSize*markerSize {
					px, py := x+dx, y+dy
					if px >= 0 && px < img.Bounds().Max.X && py >= 0 && py < img.Bounds().Max.Y {
						img.Set(px, py, markerColor)
					}
				}
			}
		}

		// White center dot
		centerColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				px, py := x+dx, y+dy
				if px >= 0 && px < img.Bounds().Max.X && py >= 0 && py < img.Bounds().Max.Y {
					img.Set(px, py, centerColor)
				}
			}
		}
	}
}

// renderGrid draws grid lines on the image.
func renderGrid(img *image.RGBA, cellSize int) {
	gridColor := color.RGBA{R: 200, G: 200, B: 200, A: 100}
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
	case string(HeatmapRSSI), "signal":
		return HeatmapRSSI
	case string(HeatmapSNR):
		return HeatmapSNR
	case string(HeatmapDensity), "ap_density":
		return HeatmapDensity
	case string(HeatmapInterference), "cochannel":
		return HeatmapInterference
	case string(HeatmapDownload):
		return HeatmapDownload
	case string(HeatmapUpload):
		return HeatmapUpload
	default:
		return HeatmapRSSI
	}
}
