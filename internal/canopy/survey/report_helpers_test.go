package survey_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.ExportTruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestGetCoverageGrade(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"excellent A", 95, "A"},
		{"good A threshold", 90, "A"},
		{"good B", 85, "B"},
		{"fair C", 75, "C"},
		{"poor D", 65, "D"},
		{"failing F", 50, "F"},
		{"failing F low", 30, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.ExportGetCoverageGrade(tt.score)
			if result != tt.expected {
				t.Errorf("getCoverageGrade(%v) = %q, want %q", tt.score, result, tt.expected)
			}
		})
	}
}

func TestGetGradeColor(t *testing.T) {
	tests := []struct {
		name  string
		grade string
	}{
		{"grade A", "A"},
		{"grade B", "B"},
		{"grade C", "C"},
		{"grade D", "D"},
		{"grade F", "F"},
		{"unknown grade", "X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.ExportGetGradeColor(tt.grade)
			if len(result) != 3 {
				t.Errorf("getGradeColor(%q) returned %d values, want 3", tt.grade, len(result))
			}
			// Verify RGB values are in valid range
			for i, v := range result {
				if v < 0 || v > 255 {
					t.Errorf("getGradeColor(%q)[%d] = %d, want 0-255", tt.grade, i, v)
				}
			}
		})
	}
}

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		name   string
		status survey.Status
	}{
		{"completed", survey.StatusCompleted},
		{"in progress", survey.StatusInProgress},
		{"paused", survey.StatusPaused},
		{"created", survey.StatusCreated},
		{"unknown", survey.Status("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.ExportGetStatusColor(tt.status)
			if len(result) != 3 {
				t.Errorf("getStatusColor(%q) returned %d values, want 3", tt.status, len(result))
			}
			// Verify RGB values are in valid range
			for i, v := range result {
				if v < 0 || v > 255 {
					t.Errorf("getStatusColor(%q)[%d] = %d, want 0-255", tt.status, i, v)
				}
			}
		})
	}
}

func TestGetPriorityLabel(t *testing.T) {
	tests := []struct {
		name     string
		priority survey.RecommendationPriority
		expected string
	}{
		{"high priority", survey.PriorityHigh, "HIGH"},
		{"medium priority", survey.PriorityMedium, "MED"},
		{"low priority", survey.PriorityLow, "LOW"},
		{"unknown priority", survey.RecommendationPriority(99), "LOW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.ExportGetPriorityLabel(tt.priority)
			if result != tt.expected {
				t.Errorf("getPriorityLabel(%d) = %q, want %q", tt.priority, result, tt.expected)
			}
		})
	}
}

func TestGenerateSurveyRecommendations(t *testing.T) {
	tests := []struct {
		name                  string
		stats                 *survey.SurveyStats
		expectRecommendations bool
		minCount              int
	}{
		{
			name: "critical coverage",
			stats: &survey.SurveyStats{
				CoverageScore: 30,
				DeadZones:     5,
				WeakAreas:     10,
				TotalSamples:  100,
			},
			expectRecommendations: true,
			minCount:              2,
		},
		{
			name: "poor coverage",
			stats: &survey.SurveyStats{
				CoverageScore: 60,
				DeadZones:     2,
				WeakAreas:     5,
				TotalSamples:  100,
			},
			expectRecommendations: true,
			minCount:              2,
		},
		{
			name: "moderate coverage",
			stats: &survey.SurveyStats{
				CoverageScore: 80,
				DeadZones:     0,
				WeakAreas:     2,
				TotalSamples:  100,
			},
			expectRecommendations: true,
			minCount:              1,
		},
		{
			name: "excellent coverage",
			stats: &survey.SurveyStats{
				CoverageScore: 95,
				DeadZones:     0,
				WeakAreas:     0,
				TotalSamples:  100,
			},
			expectRecommendations: true,
			minCount:              1, // Should have "meets quality standards" recommendation
		},
		{
			name: "limited samples",
			stats: &survey.SurveyStats{
				CoverageScore: 90,
				DeadZones:     0,
				WeakAreas:     0,
				TotalSamples:  10, // Below 20 threshold
			},
			expectRecommendations: true,
			minCount:              1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := survey.ExportGenerateSurveyRecommendations(tt.stats)
			if tt.expectRecommendations && len(recs) < tt.minCount {
				t.Errorf("Expected at least %d recommendations, got %d", tt.minCount, len(recs))
			}
		})
	}
}

func TestCalculateSurveyStats_EmptySamples(t *testing.T) {
	stats := survey.ExportCalculateSurveyStats(nil)

	if stats.TotalSamples != 0 {
		t.Errorf("Expected TotalSamples 0, got %d", stats.TotalSamples)
	}
	if stats.CoverageScore != 0 {
		t.Errorf("Expected CoverageScore 0, got %f", stats.CoverageScore)
	}
}

func TestCalculateSurveyStats_WithSamples(t *testing.T) {
	now := time.Now()
	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -40}}, // Excellent
			},
		},
		{
			X: 20, Y: 20, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -60}}, // Good
			},
		},
		{
			X: 30, Y: 30, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -70}}, // Fair
			},
		},
		{
			X: 40, Y: 40, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -80}}, // Poor
			},
		},
		{
			X: 50, Y: 50, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -90}}, // Dead
			},
		},
	}

	stats := survey.ExportCalculateSurveyStats(samples)

	if stats.TotalSamples != 5 {
		t.Errorf("Expected TotalSamples 5, got %d", stats.TotalSamples)
	}
	if stats.MaxRSSI != -40 {
		t.Errorf("Expected MaxRSSI -40, got %d", stats.MaxRSSI)
	}
	if stats.MinRSSI != -90 {
		t.Errorf("Expected MinRSSI -90, got %d", stats.MinRSSI)
	}
	if stats.DeadZones != 1 {
		t.Errorf("Expected 1 dead zone, got %d", stats.DeadZones)
	}
	if stats.WeakAreas != 1 {
		t.Errorf("Expected 1 weak area, got %d", stats.WeakAreas)
	}
}

func TestCalculateSurveyStats_NilPassiveSample(t *testing.T) {
	now := time.Now()
	samples := []*survey.SamplePoint{
		{X: 10, Y: 10, Timestamp: now, SampleData: nil},
		{X: 20, Y: 20, Timestamp: now, SampleData: &survey.PassiveSample{Networks: nil}},
		{X: 30, Y: 30, Timestamp: now, SampleData: &survey.PassiveSample{Networks: []*wifi.ScannedNetwork{}}},
	}

	stats := survey.ExportCalculateSurveyStats(samples)

	// Should handle nil gracefully
	if stats.TotalSamples != 3 {
		t.Errorf("Expected TotalSamples 3, got %d", stats.TotalSamples)
	}
}

func TestGetChannelUsage(t *testing.T) {
	now := time.Now()
	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{Channel: 1},
					{Channel: 6},
					{Channel: 11},
				},
			},
		},
		{
			X: 20, Y: 20, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{Channel: 1},
					{Channel: 1}, // Same channel
					{Channel: 6},
				},
			},
		},
	}

	channels := survey.ExportGetChannelUsage(samples)

	if len(channels) != 3 {
		t.Errorf("Expected 3 channels, got %d", len(channels))
	}

	// Should be sorted by count (channel 1 has highest count)
	if channels[0].Channel != 1 {
		t.Errorf("Expected channel 1 to be first (most used), got channel %d", channels[0].Channel)
	}
}

func TestGetChannelUsage_EmptySamples(t *testing.T) {
	channels := survey.ExportGetChannelUsage(nil)
	if len(channels) != 0 {
		t.Errorf("Expected 0 channels for nil samples, got %d", len(channels))
	}
}

func TestGetChannelUsage_LimitToTopChannels(t *testing.T) {
	now := time.Now()
	// Create samples with more than 5 channels
	networks := make([]*wifi.ScannedNetwork, 10)
	for i := range 10 {
		networks[i] = &wifi.ScannedNetwork{Channel: i + 1}
	}

	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{Networks: networks},
		},
	}

	channels := survey.ExportGetChannelUsage(samples)

	// Should be limited to top 5
	if len(channels) > 5 {
		t.Errorf("Expected at most 5 channels, got %d", len(channels))
	}
}

func TestGetPassiveSampleFromPoint(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		sp         *survey.SamplePoint
		expectNil  bool
		expectSSID string
	}{
		{
			name:      "nil sample point",
			sp:        nil,
			expectNil: true,
		},
		{
			name: "nil sample data",
			sp: &survey.SamplePoint{
				X: 10, Y: 10, Timestamp: now,
				SampleData: nil,
			},
			expectNil: true,
		},
		{
			name: "pointer to PassiveSample",
			sp: &survey.SamplePoint{
				X: 10, Y: 10, Timestamp: now,
				SampleData: &survey.PassiveSample{
					Networks: []*wifi.ScannedNetwork{{SSID: "TestNet"}},
				},
			},
			expectNil:  false,
			expectSSID: "TestNet",
		},
		{
			name: "value PassiveSample",
			sp: &survey.SamplePoint{
				X: 10, Y: 10, Timestamp: now,
				SampleData: survey.PassiveSample{
					Networks: []*wifi.ScannedNetwork{{SSID: "ValueNet"}},
				},
			},
			expectNil:  false,
			expectSSID: "ValueNet",
		},
		{
			name: "ActiveSample returns nil",
			sp: &survey.SamplePoint{
				X: 10, Y: 10, Timestamp: now,
				SampleData: &survey.ActiveSample{SSID: "Active"},
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.ExportGetPassiveSampleFromPoint(tt.sp)
			if tt.expectNil && result != nil {
				t.Error("Expected nil result")
			}
			if !tt.expectNil && result == nil {
				t.Error("Expected non-nil result")
			}
			if !tt.expectNil && result != nil && len(result.Networks) > 0 {
				if result.Networks[0].SSID != tt.expectSSID {
					t.Errorf("Expected SSID %q, got %q", tt.expectSSID, result.Networks[0].SSID)
				}
			}
		})
	}
}

func TestCalculateSurveyStats_MultipleNetworks(t *testing.T) {
	now := time.Now()
	// Test that stats use best RSSI from multiple networks
	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{Signal: -80}, // Poor
					{Signal: -50}, // Excellent - should use this
					{Signal: -70}, // Fair
				},
			},
		},
	}

	stats := survey.ExportCalculateSurveyStats(samples)

	if stats.MaxRSSI != -50 {
		t.Errorf("Expected MaxRSSI -50 (best of multiple networks), got %d", stats.MaxRSSI)
	}
}
