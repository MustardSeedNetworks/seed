package survey_test

import (
	"image/color"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestGenerateHeatmap_MultiFloorActiveFloor(t *testing.T) {
	now := time.Now()
	samples := []*survey.SamplePoint{
		{
			X: 100, Y: 100, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -50, SNR: 30}},
			},
		},
	}

	s := &survey.Survey{
		ID: "test",
		Floors: []*survey.Floor{
			{
				ID: "floor-1",
				FloorPlan: &survey.FloorPlan{
					Width:  400,
					Height: 400,
				},
				Samples: samples,
			},
		},
		ActiveFloorID: "floor-1",
	}

	config := survey.DefaultHeatmapConfig()
	result, err := survey.GenerateHeatmap(s, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGenerateHeatmap_NearestMethod(t *testing.T) {
	now := time.Now()
	samples := []*survey.SamplePoint{
		{
			X: 50, Y: 50, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -60}},
			},
		},
	}

	s := &survey.Survey{
		ID: "test",
		Floors: []*survey.Floor{
			{
				ID: "floor-1",
				FloorPlan: &survey.FloorPlan{
					Width:  100,
					Height: 100,
				},
				Samples: samples,
			},
		},
		ActiveFloorID: "floor-1",
	}

	config := survey.HeatmapConfig{
		Type:   survey.HeatmapRSSI,
		Method: survey.MethodNearest,
	}

	result, err := survey.GenerateHeatmap(s, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestGetHeatmapDimensions_FromActiveFloorPlan(t *testing.T) {
	s := &survey.Survey{
		Floors: []*survey.Floor{
			{
				ID: "floor-1",
				FloorPlan: &survey.FloorPlan{
					Width:  1000,
					Height: 800,
				},
			},
		},
		ActiveFloorID: "floor-1",
	}

	width, height := survey.ExportGetHeatmapDimensions(s)
	if width != 1000 {
		t.Errorf("Expected width 1000, got %d", width)
	}
	if height != 800 {
		t.Errorf("Expected height 800, got %d", height)
	}
}

func TestManager_GenerateHeatmap_Success(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_ = m.StartSurvey(s.ID)

	// Add floor plan
	floorPlan := &survey.FloorPlan{Width: 200, Height: 200}
	_ = m.UpdateFloorPlan(s.ID, floorPlan)

	// Add samples
	sampleData := &survey.PassiveSample{
		Networks: []*wifi.ScannedNetwork{{Signal: -50}},
	}
	_ = m.AddSample(s.ID, 50, 50, sampleData)
	_ = m.AddSample(s.ID, 100, 100, sampleData)

	result, err := m.GenerateHeatmap(s.ID, survey.DefaultHeatmapConfig())
	if err != nil {
		t.Fatalf("Failed to generate heatmap: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestManager_GenerateHeatmap_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	_, err := m.GenerateHeatmap("non-existent", survey.DefaultHeatmapConfig())
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestColorScale_GetColor_Interpolation(t *testing.T) {
	scale := survey.GetRSSIColorScale()

	tests := []struct {
		name  string
		value float64
	}{
		{"well below minimum", -150},
		{"at minimum", -100},
		{"between stops low", -85},
		{"between stops mid", -70},
		{"between stops high", -55},
		{"at stop value", -75},
		{"at maximum", -30},
		{"above maximum", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := scale.GetColor(tt.value)
			// Just verify we get a valid color
			if c.A != 255 {
				t.Errorf("Expected alpha 255, got %d", c.A)
			}
		})
	}
}

func TestWithAlpha_AllValues(t *testing.T) {
	original := color.RGBA{R: 100, G: 150, B: 200, A: 255}

	tests := []struct {
		alpha    uint8
		expected uint8
	}{
		{0, 0},
		{1, 1},
		{127, 127},
		{128, 128},
		{254, 254},
		{255, 255},
	}

	for _, tt := range tests {
		result := survey.WithAlpha(original, tt.alpha)
		if result.A != tt.expected {
			t.Errorf("WithAlpha(_, %d).A = %d, want %d", tt.alpha, result.A, tt.expected)
		}
		// RGB should be unchanged
		if result.R != 100 || result.G != 150 || result.B != 200 {
			t.Error("WithAlpha should not modify RGB values")
		}
	}
}
