// Package survey provides WiFi site survey functionality.
package survey

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestDetectDeadZones_NilSurvey(t *testing.T) {
	_, err := DetectDeadZones(nil, DefaultThreshold)
	if err == nil {
		t.Fatal("Expected error for nil survey, got nil")
	}
	if err.Error() != "survey is nil" {
		t.Errorf("Expected 'survey is nil' error, got: %v", err)
	}
}

func TestDetectDeadZones_NoSamples(t *testing.T) {
	survey := &Survey{
		ID:      "test-survey",
		Samples: []*SamplePoint{},
	}

	_, err := DetectDeadZones(survey, DefaultThreshold)
	if err == nil {
		t.Fatal("Expected error for survey with no samples, got nil")
	}
	if err.Error() != "survey has no samples" {
		t.Errorf("Expected 'survey has no samples' error, got: %v", err)
	}
}

func TestDetectDeadZones_NoRSSISamples(t *testing.T) {
	survey := &Survey{
		ID: "test-survey",
		Samples: []*SamplePoint{
			{
				X:         100,
				Y:         100,
				Timestamp: time.Now(),
				// Use a sample type that doesn't have RSSI-like data
				SampleData: map[string]interface{}{
					"someField": "someValue",
				},
			},
		},
	}

	_, err := DetectDeadZones(survey, DefaultThreshold)
	if err == nil {
		t.Fatal("Expected error for survey with no RSSI samples, got nil")
	}
}

func TestDetectDeadZones_AllGoodSignals(t *testing.T) {
	survey := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -55),
		createPassiveSamplePoint(200, 200, -60),
		createPassiveSamplePoint(300, 300, -65),
	})

	analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis.DeadZones) != 0 {
		t.Errorf("Expected 0 dead zones, got %d", len(analysis.DeadZones))
	}

	if analysis.CoverageScore != 100.0 {
		t.Errorf("Expected coverage score 100.0, got %.2f", analysis.CoverageScore)
	}

	if analysis.SurveyID != survey.ID {
		t.Errorf("Expected survey ID %s, got %s", survey.ID, analysis.SurveyID)
	}

	if analysis.ThresholdDBm != DefaultThreshold {
		t.Errorf("Expected threshold %d, got %d", DefaultThreshold, analysis.ThresholdDBm)
	}
}

func TestDetectDeadZones_SingleDeadZone(t *testing.T) {
	survey := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -88), // Severe
		createPassiveSamplePoint(110, 110, -86), // Severe
		createPassiveSamplePoint(120, 120, -87), // Severe
	})

	analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis.DeadZones) != 1 {
		t.Fatalf("Expected 1 dead zone, got %d", len(analysis.DeadZones))
	}

	zone := analysis.DeadZones[0]
	if zone.Severity != "severe" {
		t.Errorf("Expected severity 'severe', got '%s'", zone.Severity)
	}

	if zone.SampleCount != 3 {
		t.Errorf("Expected 3 samples in zone, got %d", zone.SampleCount)
	}

	if zone.ID != "zone-1" {
		t.Errorf("Expected zone ID 'zone-1', got '%s'", zone.ID)
	}

	if analysis.CoverageScore != 0.0 {
		t.Errorf("Expected coverage score 0.0, got %.2f", analysis.CoverageScore)
	}
}

func TestDetectDeadZones_MultipleDeadZones(t *testing.T) {
	survey := createTestSurveyWithSamples([]*SamplePoint{
		// Zone 1: Severe (clustered)
		createPassiveSamplePoint(100, 100, -90),
		createPassiveSamplePoint(110, 110, -88),

		// Zone 2: Moderate (clustered, far from zone 1)
		createPassiveSamplePoint(500, 500, -82),
		createPassiveSamplePoint(510, 510, -83),

		// Good signal (not in dead zone)
		createPassiveSamplePoint(300, 300, -60),
	})

	analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis.DeadZones) != 2 {
		t.Fatalf("Expected 2 dead zones, got %d", len(analysis.DeadZones))
	}

	// Verify coverage score
	expectedCoverage := (1.0 / 5.0) * 100.0 // 1 good out of 5 total
	if analysis.CoverageScore != expectedCoverage {
		t.Errorf("Expected coverage score %.2f, got %.2f", expectedCoverage, analysis.CoverageScore)
	}
}

func TestDetectDeadZones_SeverityClassification(t *testing.T) {
	tests := []struct {
		name     string
		rssi     int
		severity string
	}{
		{"Severe", -90, "severe"},
		{"Severe boundary", -85, "severe"},
		{"Moderate", -82, "moderate"},
		{"Moderate boundary", -80, "moderate"},
		{"Minor", -77, "minor"},
		{"Minor just below threshold", -76, "minor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			survey := createTestSurveyWithSamples([]*SamplePoint{
				createPassiveSamplePoint(100, 100, tt.rssi),
			})

			analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
			if err != nil {
				return
			}

			if len(analysis.DeadZones) != 1 {
				t.Fatalf("Expected 1 dead zone, got %d", len(analysis.DeadZones))
			}

			if analysis.DeadZones[0].Severity != tt.severity {
				t.Errorf("RSSI %d: expected severity '%s', got '%s'",
					tt.rssi, tt.severity, analysis.DeadZones[0].Severity)
			}
		})
	}
}

func TestDetectDeadZones_CustomThreshold(t *testing.T) {
	survey := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -68), // Above -75, below -65
		createPassiveSamplePoint(200, 200, -60), // Good signal
	})

	// With default threshold (-75), first sample is good
	analysis1, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis1.DeadZones) != 0 {
		t.Errorf("With threshold -75: expected 0 dead zones, got %d", len(analysis1.DeadZones))
	}

	// With threshold -65, first sample is weak
	analysis2, err := detectDeadZonesHelper(t, survey, -65)
	if err != nil {
		return
	}

	if len(analysis2.DeadZones) != 1 {
		t.Errorf("With threshold -65: expected 1 dead zone, got %d", len(analysis2.DeadZones))
	}
}

func TestDetectDeadZones_FloorPlanScaling(t *testing.T) {
	floorPlan := &FloorPlan{
		Width:  1000,
		Height: 1000,
		ScaleM: 0.1, // 1 pixel = 0.1 meters
	}

	survey := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -88),
		createPassiveSamplePoint(120, 120, -86), // ~28 pixels away diagonally, within ClusterRadius
	})
	survey.FloorPlan = floorPlan

	analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis.DeadZones) != 1 {
		t.Fatalf("Expected 1 dead zone, got %d", len(analysis.DeadZones))
	}

	zone := analysis.DeadZones[0]

	// Radius should be scaled to meters
	if zone.RadiusM <= 0 {
		t.Errorf("Expected positive radius in meters, got %.2f", zone.RadiusM)
	}

	// With scale of 0.1m per pixel and ~28 pixel distance between points,
	// the radius from center should be around 1-2 meters
	if zone.RadiusM < 1.0 || zone.RadiusM > 3.0 {
		t.Errorf("Expected radius between 1-3m, got %.2fm", zone.RadiusM)
	}
}

func TestDetectDeadZones_Recommendations(t *testing.T) {
	tests := []struct {
		name            string
		samples         []*SamplePoint
		expectedKeyword string
	}{
		{
			name: "Excellent coverage",
			samples: []*SamplePoint{
				createPassiveSamplePoint(100, 100, -55),
				createPassiveSamplePoint(200, 200, -60),
			},
			expectedKeyword: "Excellent",
		},
		{
			name: "Severe dead zones",
			samples: []*SamplePoint{
				createPassiveSamplePoint(100, 100, -90),
				createPassiveSamplePoint(110, 110, -88),
			},
			expectedKeyword: "severe",
		},
		{
			name: "Limited samples",
			samples: []*SamplePoint{
				createPassiveSamplePoint(100, 100, -70),
			},
			expectedKeyword: "Limited sample",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			survey := createTestSurveyWithSamples(tt.samples)
			analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
			if err != nil {
				return
			}

			if len(analysis.Recommendations) == 0 {
				t.Error("Expected recommendations, got none")
				return
			}

			found := false
			for _, rec := range analysis.Recommendations {
				if containsIgnoreCase(rec, tt.expectedKeyword) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected recommendation containing '%s', got: %v",
					tt.expectedKeyword, analysis.Recommendations)
			}
		})
	}
}

func TestDetectDeadZones_ClusteringDistance(t *testing.T) {
	// Two samples within cluster radius should be one zone
	survey1 := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -88),
		createPassiveSamplePoint(130, 130, -86), // ~42 pixels away, within ClusterRadius
	})

	analysis1, err := detectDeadZonesHelper(t, survey1, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis1.DeadZones) != 1 {
		t.Errorf("Close samples: expected 1 zone, got %d", len(analysis1.DeadZones))
	}

	// Two samples beyond cluster radius should be two zones
	survey2 := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -88),
		createPassiveSamplePoint(200, 200, -86), // ~141 pixels away, beyond ClusterRadius
	})

	analysis2, err := detectDeadZonesHelper(t, survey2, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis2.DeadZones) != 2 {
		t.Errorf("Far samples: expected 2 zones, got %d", len(analysis2.DeadZones))
	}
}

func TestDetectDeadZones_MinMaxRSSI(t *testing.T) {
	survey := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -90),
		createPassiveSamplePoint(110, 110, -80),
		createPassiveSamplePoint(120, 120, -85),
	})

	analysis, err := detectDeadZonesHelper(t, survey, DefaultThreshold)
	if err != nil {
		return
	}

	if len(analysis.DeadZones) != 1 {
		t.Fatalf("Expected 1 dead zone, got %d", len(analysis.DeadZones))
	}

	zone := analysis.DeadZones[0]

	if zone.MinRSSI != -90 {
		t.Errorf("Expected min RSSI -90, got %d", zone.MinRSSI)
	}

	expectedAvg := (-90 + -80 + -85) / 3
	if zone.AvgRSSI != expectedAvg {
		t.Errorf("Expected avg RSSI %d, got %d", expectedAvg, zone.AvgRSSI)
	}
}

func TestManager_DetectDeadZones(t *testing.T) {
	manager := NewManager(t.TempDir(), nil, nil, nil)

	// Create a survey
	survey := createTestSurveyWithSamples([]*SamplePoint{
		createPassiveSamplePoint(100, 100, -88),
	})

	manager.surveys[survey.ID] = survey

	// Test through manager
	analysis, err := manager.DetectDeadZones(survey.ID, DefaultThreshold)
	if err != nil {
		t.Fatalf("Manager.DetectDeadZones failed: %v", err)
	}

	if analysis.SurveyID != survey.ID {
		t.Errorf("Expected survey ID %s, got %s", survey.ID, analysis.SurveyID)
	}
}

func TestManager_DetectDeadZones_NotFound(t *testing.T) {
	manager := NewManager(t.TempDir(), nil, nil, nil)

	_, err := manager.DetectDeadZones("nonexistent-id", DefaultThreshold)
	if err == nil {
		t.Error("Expected error for nonexistent survey, got nil")
	}
}

func TestFilterWeakSamples(t *testing.T) {
	samples := []SampleValue{
		{Point: Point2D{X: 100, Y: 100}, Value: -60}, // Good
		{Point: Point2D{X: 200, Y: 200}, Value: -80}, // Weak
		{Point: Point2D{X: 300, Y: 300}, Value: -55}, // Good
		{Point: Point2D{X: 400, Y: 400}, Value: -90}, // Weak
	}

	weak := filterWeakSamples(samples, -75)

	if len(weak) != 2 {
		t.Errorf("Expected 2 weak samples, got %d", len(weak))
	}

	for _, s := range weak {
		if s.Value >= -75 {
			t.Errorf("Found non-weak sample with value %.2f", s.Value)
		}
	}
}

func TestCalculateCoverageScore(t *testing.T) {
	tests := []struct {
		name            string
		totalSamples    int
		weakSamples     int
		expectedScore   float64
		expectZeroScore bool
	}{
		{"All good", 10, 0, 100.0, false},
		{"All weak", 10, 10, 0.0, false},
		{"Half weak", 10, 5, 50.0, false},
		{"No samples", 0, 0, 0.0, true},
		{"Most good", 100, 20, 80.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allSamples := make([]SampleValue, tt.totalSamples)
			weakSamples := make([]SampleValue, tt.weakSamples)

			score := calculateCoverageScore(allSamples, weakSamples)

			if tt.expectZeroScore {
				if score != 0.0 {
					t.Errorf("Expected 0.0 for empty samples, got %.2f", score)
				}
			} else if score != tt.expectedScore {
				t.Errorf("Expected score %.2f, got %.2f", tt.expectedScore, score)
			}
		})
	}
}

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		rssi     float64
		severity string
	}{
		{-100, "severe"},
		{-86, "severe"},
		{-85, "severe"},
		{-84.99, "moderate"},
		{-82, "moderate"},
		{-80, "moderate"},
		{-79.99, "minor"},
		{-77, "minor"},
		{-75, "minor"},
	}

	for _, tt := range tests {
		result := determineSeverity(tt.rssi)
		if result != tt.severity {
			t.Errorf("RSSI %.2f: expected '%s', got '%s'", tt.rssi, tt.severity, result)
		}
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name          string
		deadZones     []DeadZone
		coverageScore float64
		totalSamples  int
		minRecCount   int
	}{
		{
			name:          "Perfect coverage",
			deadZones:     []DeadZone{},
			coverageScore: 98.0,
			totalSamples:  100,
			minRecCount:   2,
		},
		{
			name: "Severe zones",
			deadZones: []DeadZone{
				{Severity: "severe"},
				{Severity: "severe"},
			},
			coverageScore: 50.0,
			totalSamples:  50,
			minRecCount:   2,
		},
		{
			name:          "Limited samples",
			deadZones:     []DeadZone{},
			coverageScore: 90.0,
			totalSamples:  5,
			minRecCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := generateRecommendations(tt.deadZones, tt.coverageScore, tt.totalSamples)

			if len(recs) < tt.minRecCount {
				t.Errorf("Expected at least %d recommendations, got %d", tt.minRecCount, len(recs))
			}

			for _, rec := range recs {
				if rec == "" {
					t.Error("Found empty recommendation")
				}
			}
		})
	}
}

// Helper functions

func createTestSurveyWithSamples(samples []*SamplePoint) *Survey {
	return &Survey{
		ID:         "test-survey-id",
		Name:       "Test Survey",
		SurveyType: TypePassive,
		Status:     StatusCompleted,
		Samples:    samples,
	}
}

func createPassiveSamplePoint(x, y, rssi int) *SamplePoint {
	return &SamplePoint{
		X:         x,
		Y:         y,
		Timestamp: time.Now(),
		SampleData: &PassiveSample{
			Networks: []*wifi.ScannedNetwork{
				{
					Signal: rssi,
				},
			},
		},
	}
}

func detectDeadZonesHelper(t *testing.T, survey *Survey, threshold int) (*DeadZoneAnalysis, error) {
	t.Helper()
	analysis, err := DetectDeadZones(survey, threshold)
	if err != nil {
		t.Errorf("DetectDeadZones failed: %v", err)
		return nil, err
	}
	return analysis, nil
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexString(s, substr) >= 0)
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
