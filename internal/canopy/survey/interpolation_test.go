// Package survey provides WiFi site survey functionality.
package survey

import (
	"math"
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestNewInterpolator(t *testing.T) {
	samples := []SampleValue{
		{Point: Point2D{X: 0, Y: 0}, Value: 10},
		{Point: Point2D{X: 100, Y: 100}, Value: 20},
	}

	interp := NewInterpolator(samples)

	if len(interp.Samples) != 2 {
		t.Errorf("Expected 2 samples, got %d", len(interp.Samples))
	}
	if interp.Method != MethodIDW {
		t.Errorf("Expected method IDW, got %s", interp.Method)
	}
	if interp.Power != 2.0 {
		t.Errorf("Expected power 2.0, got %f", interp.Power)
	}
	if interp.MaxDist != 0 {
		t.Errorf("Expected maxDist 0, got %f", interp.MaxDist)
	}
	if interp.MinCount != 1 {
		t.Errorf("Expected minCount 1, got %d", interp.MinCount)
	}
}

func TestInterpolator_Interpolate_Empty(t *testing.T) {
	interp := NewInterpolator([]SampleValue{})

	result := interp.Interpolate(50, 50)
	if result != 0 {
		t.Errorf("Expected 0 for empty samples, got %f", result)
	}
}

func TestInterpolator_Interpolate_IDW(t *testing.T) {
	samples := []SampleValue{
		{Point: Point2D{X: 0, Y: 0}, Value: -70},
		{Point: Point2D{X: 100, Y: 0}, Value: -50},
		{Point: Point2D{X: 0, Y: 100}, Value: -60},
		{Point: Point2D{X: 100, Y: 100}, Value: -40},
	}

	interp := NewInterpolator(samples)
	interp.Method = MethodIDW

	// Test at a sample point - should return exact value
	result := interp.Interpolate(0, 0)
	if result != -70 {
		t.Errorf("Expected -70 at sample point, got %f", result)
	}

	// Test at center - should be weighted average
	result = interp.Interpolate(50, 50)
	// All corners are equidistant, so result should be average
	expected := (-70 + -50 + -60 + -40) / 4.0
	if math.Abs(result-expected) > 0.1 {
		t.Errorf("Expected ~%f at center, got %f", expected, result)
	}

	// Test closer to one corner
	result = interp.Interpolate(10, 10)
	// Should be closer to -70 (nearest corner)
	if result > -65 || result < -75 {
		t.Errorf("Expected value near -70 at (10,10), got %f", result)
	}
}

func TestInterpolator_Interpolate_Nearest(t *testing.T) {
	samples := []SampleValue{
		{Point: Point2D{X: 0, Y: 0}, Value: -70},
		{Point: Point2D{X: 100, Y: 100}, Value: -40},
	}

	interp := NewInterpolator(samples)
	interp.Method = MethodNearest

	// Test at first sample
	result := interp.Interpolate(0, 0)
	if result != -70 {
		t.Errorf("Expected -70, got %f", result)
	}

	// Test closer to first sample
	result = interp.Interpolate(10, 10)
	if result != -70 {
		t.Errorf("Expected -70 (nearest), got %f", result)
	}

	// Test closer to second sample
	result = interp.Interpolate(90, 90)
	if result != -40 {
		t.Errorf("Expected -40 (nearest), got %f", result)
	}
}

func TestInterpolator_Interpolate_MaxDist(t *testing.T) {
	samples := []SampleValue{
		{Point: Point2D{X: 0, Y: 0}, Value: -70},
		{Point: Point2D{X: 1000, Y: 1000}, Value: -40},
	}

	interp := NewInterpolator(samples)
	interp.Method = MethodIDW
	interp.MaxDist = 50 // Only consider samples within 50 units

	// At origin, should use first sample
	result := interp.Interpolate(0, 0)
	if result != -70 {
		t.Errorf("Expected -70, got %f", result)
	}

	// At (100, 100), first sample is out of range, should fall back to nearest
	result = interp.Interpolate(100, 100)
	if result != -70 {
		t.Errorf("Expected -70 (nearest fallback), got %f", result)
	}
}

func TestInterpolator_InterpolateGrid(t *testing.T) {
	samples := []SampleValue{
		{Point: Point2D{X: 0, Y: 0}, Value: -70},
		{Point: Point2D{X: 100, Y: 0}, Value: -50},
		{Point: Point2D{X: 0, Y: 100}, Value: -60},
		{Point: Point2D{X: 100, Y: 100}, Value: -40},
	}

	interp := NewInterpolator(samples)

	// 100x100 with 50px cells = 2x2 grid
	grid := interp.InterpolateGrid(100, 100, 50)

	if len(grid) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(grid))
	}
	if len(grid[0]) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(grid[0]))
	}

	// Values should be reasonable (between -70 and -40)
	for row, rowData := range grid {
		for col, val := range rowData {
			if val > -40 || val < -70 {
				t.Errorf("Value at [%d][%d] = %f out of range [-70, -40]",
					row, col, val)
			}
		}
	}
}

func TestDistance(t *testing.T) {
	tests := []struct {
		name     string
		p1       Point2D
		p2       Point2D
		expected float64
	}{
		{
			name:     "same point",
			p1:       Point2D{X: 0, Y: 0},
			p2:       Point2D{X: 0, Y: 0},
			expected: 0,
		},
		{
			name:     "horizontal",
			p1:       Point2D{X: 0, Y: 0},
			p2:       Point2D{X: 10, Y: 0},
			expected: 10,
		},
		{
			name:     "vertical",
			p1:       Point2D{X: 0, Y: 0},
			p2:       Point2D{X: 0, Y: 10},
			expected: 10,
		},
		{
			name:     "diagonal 3-4-5 triangle",
			p1:       Point2D{X: 0, Y: 0},
			p2:       Point2D{X: 3, Y: 4},
			expected: 5,
		},
		{
			name:     "negative coordinates",
			p1:       Point2D{X: -5, Y: -5},
			p2:       Point2D{X: -5, Y: 5},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distance(tt.p1, tt.p2)
			if math.Abs(got-tt.expected) > 0.0001 {
				t.Errorf("distance(%v, %v) = %f, want %f", tt.p1, tt.p2, got, tt.expected)
			}
		})
	}
}

func TestCalculateGridStats(t *testing.T) {
	tests := []struct {
		name     string
		grid     [][]float64
		expected GridStats
	}{
		{
			name:     "empty grid",
			grid:     [][]float64{},
			expected: GridStats{},
		},
		{
			name:     "single value",
			grid:     [][]float64{{-50}},
			expected: GridStats{Min: -50, Max: -50, Average: -50, Count: 1},
		},
		{
			name: "2x2 grid",
			grid: [][]float64{
				{-70, -50},
				{-60, -40},
			},
			expected: GridStats{Min: -70, Max: -40, Average: -55, Count: 4},
		},
		{
			name: "3x3 grid",
			grid: [][]float64{
				{10, 20, 30},
				{40, 50, 60},
				{70, 80, 90},
			},
			expected: GridStats{Min: 10, Max: 90, Average: 50, Count: 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateGridStats(tt.grid)
			if got.Count != tt.expected.Count {
				t.Errorf("Count = %d, want %d", got.Count, tt.expected.Count)
			}
			if got.Count > 0 {
				if got.Min != tt.expected.Min {
					t.Errorf("Min = %f, want %f", got.Min, tt.expected.Min)
				}
				if got.Max != tt.expected.Max {
					t.Errorf("Max = %f, want %f", got.Max, tt.expected.Max)
				}
				if math.Abs(got.Average-tt.expected.Average) > 0.0001 {
					t.Errorf("Average = %f, want %f", got.Average, tt.expected.Average)
				}
			}
		})
	}
}

func TestExtractSamplesFromSurvey(t *testing.T) {
	survey := &Survey{
		Samples: []*SamplePoint{
			{
				X: 10,
				Y: 20,
				SampleData: &PassiveSample{
					Networks: []*wifi.ScannedNetwork{
						{Signal: -55, SNR: 30},
					},
					UniqueBSSIDs: 5,
					CoChannelAPs: 2,
				},
			},
			{
				X: 30,
				Y: 40,
				SampleData: &PassiveSample{
					Networks: []*wifi.ScannedNetwork{
						{Signal: -65, SNR: 25},
					},
					UniqueBSSIDs: 3,
					CoChannelAPs: 1,
				},
			},
		},
	}

	tests := []struct {
		name      string
		valueType string
		expected  []float64
	}{
		{
			name:      "rssi",
			valueType: "rssi",
			expected:  []float64{-55, -65},
		},
		{
			name:      "snr",
			valueType: "snr",
			expected:  []float64{30, 25},
		},
		{
			name:      "density",
			valueType: "density",
			expected:  []float64{5, 3},
		},
		{
			name:      "interference",
			valueType: "interference",
			expected:  []float64{2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			samples := ExtractSamplesFromSurvey(survey, tt.valueType)
			if len(samples) != len(tt.expected) {
				t.Fatalf("Expected %d samples, got %d", len(tt.expected), len(samples))
			}
			for i, sample := range samples {
				if sample.Value != tt.expected[i] {
					t.Errorf("Sample[%d].Value = %f, want %f", i, sample.Value, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractPassiveValue(t *testing.T) {
	sample := &PassiveSample{
		Networks: []*wifi.ScannedNetwork{
			{Signal: -55, SNR: 30},
		},
		UniqueBSSIDs: 5,
		CoChannelAPs: 2,
		APCount2_4:   3,
		APCount5:     2,
		APCount6:     1,
	}

	tests := []struct {
		name      string
		valueType string
		expected  float64
	}{
		{"rssi", "rssi", -55},
		{"signal alias", "signal", -55},
		{"snr", "snr", 30},
		{"density", "density", 5},
		{"ap_count alias", "ap_count", 5},
		{"interference", "interference", 2},
		{"cochannel alias", "cochannel", 2},
		{"ap_2_4", "ap_2_4", 3},
		{"ap_5", "ap_5", 2},
		{"ap_6", "ap_6", 1},
		{"default", "unknown", -55}, // Defaults to signal
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPassiveValue(sample, tt.valueType)
			if got != tt.expected {
				t.Errorf("extractPassiveValue(%q) = %f, want %f", tt.valueType, got, tt.expected)
			}
		})
	}
}

func TestExtractPassiveValue_Empty(t *testing.T) {
	// Nil sample
	result := extractPassiveValue(nil, "rssi")
	if !math.IsNaN(result) {
		t.Errorf("Expected NaN for nil sample, got %f", result)
	}

	// Empty networks
	sample := &PassiveSample{Networks: []*wifi.ScannedNetwork{}}
	result = extractPassiveValue(sample, "rssi")
	if !math.IsNaN(result) {
		t.Errorf("Expected NaN for empty networks, got %f", result)
	}
}

func TestExtractActiveValue(t *testing.T) {
	sample := &ActiveSample{
		RSSI:     -60,
		DataRate: 100.5,
	}

	tests := []struct {
		name      string
		valueType string
		expected  float64
	}{
		{"rssi", "rssi", -60},
		{"signal alias", "signal", -60},
		{"datarate", "datarate", 100.5},
		{"speed alias", "speed", 100.5},
		{"default", "unknown", -60}, // Defaults to RSSI
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractActiveValue(sample, tt.valueType)
			if got != tt.expected {
				t.Errorf("extractActiveValue(%q) = %f, want %f", tt.valueType, got, tt.expected)
			}
		})
	}
}

func TestExtractActiveValue_Nil(t *testing.T) {
	result := extractActiveValue(nil, "rssi")
	if !math.IsNaN(result) {
		t.Errorf("Expected NaN for nil sample, got %f", result)
	}
}

func TestExtractThroughputValue(t *testing.T) {
	sample := &ThroughputSample{
		RSSI:         -65,
		DownloadMbps: 100.0,
		UploadMbps:   50.0,
		Latency:      10.5,
		Jitter:       2.3,
	}

	tests := []struct {
		name      string
		valueType string
		expected  float64
	}{
		{"rssi", "rssi", -65},
		{"signal alias", "signal", -65},
		{"download", "download", 100.0},
		{"upload", "upload", 50.0},
		{"latency", "latency", 10.5},
		{"jitter", "jitter", 2.3},
		{"default", "unknown", -65}, // Defaults to RSSI
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractThroughputValue(sample, tt.valueType)
			if got != tt.expected {
				t.Errorf("extractThroughputValue(%q) = %f, want %f", tt.valueType, got, tt.expected)
			}
		})
	}
}

func TestExtractThroughputValue_Nil(t *testing.T) {
	result := extractThroughputValue(nil, "rssi")
	if !math.IsNaN(result) {
		t.Errorf("Expected NaN for nil sample, got %f", result)
	}
}

func TestExtractMapValue(t *testing.T) {
	data := map[string]any{
		"networks": []any{
			map[string]any{
				"rssi": float64(-55),
			},
		},
		"uniqueBSSIDs": float64(5),
		"coChannelAPs": float64(2),
	}

	tests := []struct {
		name      string
		valueType string
		expected  float64
	}{
		{"rssi from networks", "rssi", -55},
		{"signal alias", "signal", -55},
		{"density", "density", 5},
		{"interference", "interference", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMapValue(data, tt.valueType)
			if got != tt.expected {
				t.Errorf("extractMapValue(%q) = %f, want %f", tt.valueType, got, tt.expected)
			}
		})
	}
}

func TestExtractMapValue_IntValue(t *testing.T) {
	data := map[string]any{
		"rssi": int(-60),
	}

	result := extractMapValue(data, "rssi")
	if result != -60 {
		t.Errorf("Expected -60 for int value, got %f", result)
	}
}

func TestExtractMapValue_Missing(t *testing.T) {
	data := map[string]any{}

	result := extractMapValue(data, "rssi")
	if !math.IsNaN(result) {
		t.Errorf("Expected NaN for missing key, got %f", result)
	}
}

func TestExtractValue_UnsupportedType(t *testing.T) {
	result := extractValue("string data", "rssi")
	if !math.IsNaN(result) {
		t.Errorf("Expected NaN for unsupported type, got %f", result)
	}
}

func TestExtractValue_PassiveSampleDirect(t *testing.T) {
	// Test non-pointer PassiveSample
	sample := PassiveSample{
		Networks: []*wifi.ScannedNetwork{
			{Signal: -55},
		},
	}

	result := extractValue(sample, "rssi")
	if result != -55 {
		t.Errorf("Expected -55, got %f", result)
	}
}

func TestExtractValue_ActiveSampleDirect(t *testing.T) {
	// Test non-pointer ActiveSample
	sample := ActiveSample{
		RSSI: -60,
	}

	result := extractValue(sample, "rssi")
	if result != -60 {
		t.Errorf("Expected -60, got %f", result)
	}
}

func TestExtractValue_ThroughputSampleDirect(t *testing.T) {
	// Test non-pointer ThroughputSample
	sample := ThroughputSample{
		RSSI: -65,
	}

	result := extractValue(sample, "rssi")
	if result != -65 {
		t.Errorf("Expected -65, got %f", result)
	}
}
