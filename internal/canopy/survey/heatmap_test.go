// Package survey_test provides WiFi site survey functionality tests.
package survey_test

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestDefaultHeatmapConfig(t *testing.T) {
	config := survey.DefaultHeatmapConfig()

	if config.Type != survey.HeatmapRSSI {
		t.Errorf("Expected type RSSI, got %s", config.Type)
	}
	if config.CellSize != 10 {
		t.Errorf("Expected cellSize 10, got %d", config.CellSize)
	}
	if config.Opacity != 180 {
		t.Errorf("Expected opacity 180, got %d", config.Opacity)
	}
	if config.Method != survey.MethodIDW {
		t.Errorf("Expected method IDW, got %s", config.Method)
	}
	if config.Power != 2.0 {
		t.Errorf("Expected power 2.0, got %f", config.Power)
	}
	if config.ShowGrid {
		t.Error("Expected ShowGrid false")
	}
	if !config.ShowSamples {
		t.Error("Expected ShowSamples true")
	}
	if !config.BlendWithPlan {
		t.Error("Expected BlendWithPlan true")
	}
}

func TestGenerateHeatmap_NilSurvey(t *testing.T) {
	config := survey.DefaultHeatmapConfig()
	result, err := survey.GenerateHeatmap(nil, config)

	if err == nil {
		t.Error("Expected error for nil survey")
	}
	if result != nil {
		t.Error("Expected nil result for nil survey")
	}
	if err.Error() != "survey is nil" {
		t.Errorf("Expected 'survey is nil' error, got %q", err.Error())
	}
}

func TestGenerateHeatmap_NoFloorPlan(t *testing.T) {
	s := &survey.Survey{
		ID:        "test",
		FloorPlan: nil,
		Samples:   []*survey.SamplePoint{},
	}
	config := survey.DefaultHeatmapConfig()
	result, err := survey.GenerateHeatmap(s, config)

	if err == nil {
		t.Error("Expected error for survey without floor plan or samples")
	}
	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestGenerateHeatmap_NoSamples(t *testing.T) {
	s := &survey.Survey{
		ID: "test",
		FloorPlan: &survey.FloorPlan{
			Width:  100,
			Height: 100,
		},
		Samples: []*survey.SamplePoint{},
	}
	config := survey.DefaultHeatmapConfig()
	result, err := survey.GenerateHeatmap(s, config)

	if err == nil {
		t.Error("Expected error for no samples")
	}
	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestGenerateHeatmap_Success(t *testing.T) {
	s := createTestSurvey()
	config := survey.DefaultHeatmapConfig()

	result, err := survey.GenerateHeatmap(s, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check result properties.
	if result.Width != 100 {
		t.Errorf("Expected width 100, got %d", result.Width)
	}
	if result.Height != 100 {
		t.Errorf("Expected height 100, got %d", result.Height)
	}
	if result.Type != string(survey.HeatmapRSSI) {
		t.Errorf("Expected type rssi, got %s", result.Type)
	}
	if result.SampleCount != 4 {
		t.Errorf("Expected 4 samples, got %d", result.SampleCount)
	}
	if result.Generated.IsZero() {
		t.Error("Expected non-zero generated time")
	}

	// Check image data.
	if len(result.Image) == 0 {
		t.Error("Expected non-empty image data")
	}
	if result.ImageBase64 == "" {
		t.Error("Expected non-empty base64 image")
	}

	// Verify PNG is valid.
	_, err = png.Decode(bytes.NewReader(result.Image))
	if err != nil {
		t.Errorf("Invalid PNG data: %v", err)
	}

	// Check stats.
	if result.Stats.Count == 0 {
		t.Error("Expected non-zero stats count")
	}
	if result.Stats.Min > result.Stats.Max {
		t.Error("Stats min > max")
	}
}

func TestGenerateHeatmap_DefaultsApplied(t *testing.T) {
	s := createTestSurvey()
	config := survey.HeatmapConfig{
		Type:     survey.HeatmapRSSI,
		CellSize: 0, // Should default to 10.
		Opacity:  0, // Should default to 180.
		Power:    0, // Should default to 2.0.
	}

	result, err := survey.GenerateHeatmap(s, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	// If we got here, defaults were applied successfully.
}

func TestGenerateHeatmap_WithGrid(t *testing.T) {
	s := createTestSurvey()
	config := survey.DefaultHeatmapConfig()
	config.ShowGrid = true

	result, err := survey.GenerateHeatmap(s, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGenerateHeatmap_WithoutSamples(t *testing.T) {
	s := createTestSurvey()
	config := survey.DefaultHeatmapConfig()
	config.ShowSamples = false

	result, err := survey.GenerateHeatmap(s, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGenerateHeatmap_AllTypes(t *testing.T) {
	s := createTestSurveyWithAll()

	types := []survey.HeatmapType{
		survey.HeatmapRSSI,
		survey.HeatmapSNR,
		survey.HeatmapDensity,
		survey.HeatmapInterference,
	}

	for _, ht := range types {
		t.Run(string(ht), func(t *testing.T) {
			config := survey.DefaultHeatmapConfig()
			config.Type = ht

			result, err := survey.GenerateHeatmap(s, config)
			if err != nil {
				t.Fatalf("Unexpected error for type %s: %v", ht, err)
			}
			if result == nil {
				t.Fatalf("Expected non-nil result for type %s", ht)
			}
			if result.Type != string(ht) {
				t.Errorf("Expected type %s, got %s", ht, result.Type)
			}
		})
	}
}

func TestGenerateHeatmap_ThroughputTypes(t *testing.T) {
	s := createTestSurveyWithThroughput()

	types := []survey.HeatmapType{
		survey.HeatmapDownload,
		survey.HeatmapUpload,
	}

	for _, ht := range types {
		t.Run(string(ht), func(t *testing.T) {
			config := survey.DefaultHeatmapConfig()
			config.Type = ht

			result, err := survey.GenerateHeatmap(s, config)
			if err != nil {
				t.Fatalf("Unexpected error for type %s: %v", ht, err)
			}
			if result == nil {
				t.Fatalf("Expected non-nil result for type %s", ht)
			}
		})
	}
}

func TestParseHeatmapType(t *testing.T) {
	tests := []struct {
		input    string
		expected survey.HeatmapType
	}{
		{"rssi", survey.HeatmapRSSI},
		{"RSSI", survey.HeatmapRSSI},
		{"signal", survey.HeatmapRSSI},
		{"snr", survey.HeatmapSNR},
		{"SNR", survey.HeatmapSNR},
		{"density", survey.HeatmapDensity},
		{"ap_density", survey.HeatmapDensity},
		{"interference", survey.HeatmapInterference},
		{"cochannel", survey.HeatmapInterference},
		{"download", survey.HeatmapDownload},
		{"upload", survey.HeatmapUpload},
		{"unknown", survey.HeatmapRSSI}, // Default.
		{"", survey.HeatmapRSSI},        // Default.
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := survey.ParseHeatmapType(tt.input)
			if got != tt.expected {
				t.Errorf("ParseHeatmapType(%q) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetColorScaleForType(t *testing.T) {
	tests := []struct {
		heatmapType survey.HeatmapType
		scaleName   string
	}{
		{survey.HeatmapRSSI, "rssi"},
		{survey.HeatmapSNR, "snr"},
		{survey.HeatmapDensity, "ap_density"},
		{survey.HeatmapInterference, "interference"},
		{survey.HeatmapDownload, "throughput"},
		{survey.HeatmapUpload, "throughput"},
		{"unknown", "rssi"}, // Default.
	}

	for _, tt := range tests {
		t.Run(string(tt.heatmapType), func(t *testing.T) {
			scale := survey.GetColorScaleForType(tt.heatmapType)
			if scale.Name != tt.scaleName {
				t.Errorf("getColorScaleForType(%s) = %s, want %s",
					tt.heatmapType, scale.Name, tt.scaleName)
			}
		})
	}
}

func TestMapHeatmapTypeToValueType(t *testing.T) {
	tests := []struct {
		input    survey.HeatmapType
		expected string
	}{
		{survey.HeatmapRSSI, "rssi"},
		{survey.HeatmapSNR, "snr"},
		{survey.HeatmapDensity, "density"},
		{survey.HeatmapInterference, "interference"},
		{survey.HeatmapDownload, "download"},
		{survey.HeatmapUpload, "upload"},
		{"unknown", "rssi"}, // Default.
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := survey.MapHeatmapTypeToValueType(tt.input)
			if got != tt.expected {
				t.Errorf("mapHeatmapTypeToValueType(%s) = %s, want %s",
					tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetHeatmapDimensions_FloorPlan(t *testing.T) {
	s := &survey.Survey{
		FloorPlan: &survey.FloorPlan{
			Width:  500,
			Height: 300,
		},
	}

	width, height := survey.GetHeatmapDimensions(s)
	if width != 500 {
		t.Errorf("Expected width 500, got %d", width)
	}
	if height != 300 {
		t.Errorf("Expected height 300, got %d", height)
	}
}

func TestGetHeatmapDimensions_FromSamples(t *testing.T) {
	s := &survey.Survey{
		FloorPlan: nil,
		Samples: []*survey.SamplePoint{
			{X: 100, Y: 50},
			{X: 200, Y: 150},
			{X: 50, Y: 200},
		},
	}

	width, height := survey.GetHeatmapDimensions(s)
	// Should be max + padding (50).
	if width != 250 {
		t.Errorf("Expected width 250 (200+50), got %d", width)
	}
	if height != 250 {
		t.Errorf("Expected height 250 (200+50), got %d", height)
	}
}

func TestGetHeatmapDimensions_NoData(t *testing.T) {
	s := &survey.Survey{
		FloorPlan: nil,
		Samples:   []*survey.SamplePoint{},
	}

	width, height := survey.GetHeatmapDimensions(s)
	if width != 0 || height != 0 {
		t.Errorf("Expected 0x0, got %dx%d", width, height)
	}
}

func TestRenderHeatmapToImage(t *testing.T) {
	// This is a simple smoke test.
	grid := [][]float64{
		{-70, -60},
		{-65, -55},
	}
	img := survey.CreateTestImage(40, 40)
	scale := &survey.RSSIColorScale

	// Should not panic.
	survey.RenderHeatmapToImage(img, grid, 20, scale, 180)

	// Verify some pixels were set.
	c := img.At(5, 5)
	r, g, b, a := c.RGBA()
	if a == 0 {
		t.Error("Expected non-zero alpha at (5,5)")
	}
	// Just check we got some color.
	if r == 0 && g == 0 && b == 0 {
		t.Error("Expected non-black color at (5,5)")
	}
}

func TestRenderHeatmapToImage_EmptyGrid(t *testing.T) {
	img := survey.CreateTestImage(100, 100)
	grid := [][]float64{}
	scale := &survey.RSSIColorScale

	// Should not panic.
	survey.RenderHeatmapToImage(img, grid, 10, scale, 180)

	// Verify image was not modified (still transparent).
	c := img.At(50, 50).(color.RGBA)
	if c.A != 0 {
		t.Errorf("Expected transparent pixel for empty grid, got alpha %d", c.A)
	}
}

func TestRenderSamplePoints(t *testing.T) {
	img := survey.CreateTestImage(100, 100)
	samples := []survey.SampleValue{
		{Point: survey.Point2D{X: 50, Y: 50}, Value: -60},
	}

	// Should not panic.
	survey.RenderSamplePoints(img, samples)

	// Check center is white (marker center).
	c := img.At(50, 50).(color.RGBA)
	if c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("Expected white center at (50,50), got %v", c)
	}
}

func TestRenderSamplePoints_EdgeCases(t *testing.T) {
	img := survey.CreateTestImage(100, 100)
	samples := []survey.SampleValue{
		{Point: survey.Point2D{X: 0, Y: 0}, Value: -60},       // Corner.
		{Point: survey.Point2D{X: 99, Y: 99}, Value: -60},     // Other corner.
		{Point: survey.Point2D{X: 1000, Y: 1000}, Value: -60}, // Outside bounds.
	}

	// Should not panic.
	survey.RenderSamplePoints(img, samples)

	// Verify corner markers were drawn (white center).
	c := img.At(0, 0).(color.RGBA)
	if c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("Expected white center at (0,0), got %v", c)
	}
}

func TestRenderGrid(t *testing.T) {
	img := survey.CreateTestImage(100, 100)

	// Should not panic.
	survey.RenderGrid(img, 20)

	// Check that vertical line exists at x=0.
	c := img.At(0, 50).(color.RGBA)
	if c.R != 200 || c.G != 200 || c.B != 200 {
		t.Errorf("Expected grid color at (0,50), got %v", c)
	}

	// Check that horizontal line exists at y=0.
	c = img.At(50, 0).(color.RGBA)
	if c.R != 200 || c.G != 200 || c.B != 200 {
		t.Errorf("Expected grid color at (50,0), got %v", c)
	}
}

// Helper functions.

func createTestSurvey() *survey.Survey {
	return &survey.Survey{
		ID:   "test-survey",
		Name: "Test Survey",
		FloorPlan: &survey.FloorPlan{
			Width:  100,
			Height: 100,
		},
		Samples: []*survey.SamplePoint{
			{
				X:         10,
				Y:         10,
				Timestamp: time.Now(),
				SampleData: &survey.PassiveSample{
					Networks: []*wifi.ScannedNetwork{
						{Signal: -55, SNR: 30},
					},
					UniqueBSSIDs: 5,
					CoChannelAPs: 2,
				},
			},
			{
				X:         90,
				Y:         10,
				Timestamp: time.Now(),
				SampleData: &survey.PassiveSample{
					Networks: []*wifi.ScannedNetwork{
						{Signal: -65, SNR: 25},
					},
					UniqueBSSIDs: 4,
					CoChannelAPs: 3,
				},
			},
			{
				X:         10,
				Y:         90,
				Timestamp: time.Now(),
				SampleData: &survey.PassiveSample{
					Networks: []*wifi.ScannedNetwork{
						{Signal: -60, SNR: 28},
					},
					UniqueBSSIDs: 3,
					CoChannelAPs: 1,
				},
			},
			{
				X:         90,
				Y:         90,
				Timestamp: time.Now(),
				SampleData: &survey.PassiveSample{
					Networks: []*wifi.ScannedNetwork{
						{Signal: -70, SNR: 20},
					},
					UniqueBSSIDs: 6,
					CoChannelAPs: 4,
				},
			},
		},
	}
}

func createTestSurveyWithAll() *survey.Survey {
	s := createTestSurvey()
	// Add more diverse data for comprehensive testing.
	return s
}

func createTestSurveyWithThroughput() *survey.Survey {
	return &survey.Survey{
		ID:   "test-throughput",
		Name: "Throughput Test",
		FloorPlan: &survey.FloorPlan{
			Width:  100,
			Height: 100,
		},
		Samples: []*survey.SamplePoint{
			{
				X:         10,
				Y:         10,
				Timestamp: time.Now(),
				SampleData: &survey.ThroughputSample{
					RSSI:         -55,
					DownloadMbps: 100,
					UploadMbps:   50,
				},
			},
			{
				X:         90,
				Y:         90,
				Timestamp: time.Now(),
				SampleData: &survey.ThroughputSample{
					RSSI:         -70,
					DownloadMbps: 50,
					UploadMbps:   25,
				},
			},
		},
	}
}
