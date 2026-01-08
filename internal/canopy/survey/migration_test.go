package survey_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
)

func TestMigrateToMultiFloor_AlreadyMigrated(t *testing.T) {
	// Survey already has floors - no migration needed
	s := &survey.Survey{
		ID: "test-survey",
		Floors: []*survey.Floor{
			{
				ID:   "floor-1",
				Name: "Floor 1",
			},
		},
	}

	migrated := survey.MigrateToMultiFloor(s)
	if migrated {
		t.Error("Expected no migration for survey with existing floors")
	}
}

func TestMigrateToMultiFloor_EmptySurvey(t *testing.T) {
	// Survey with no floor plan and no samples - should create default floor
	now := time.Now()
	s := &survey.Survey{
		ID:        "test-survey",
		CreatedAt: now,
		FloorPlan: nil,
		Samples:   nil,
	}

	migrated := survey.MigrateToMultiFloor(s)
	if !migrated {
		t.Error("Expected migration for empty survey")
	}

	if len(s.Floors) != 1 {
		t.Errorf("Expected 1 floor after migration, got %d", len(s.Floors))
	}

	if s.Floors[0].Name != "Floor 1" {
		t.Errorf("Expected floor name 'Floor 1', got '%s'", s.Floors[0].Name)
	}

	if s.Floors[0].Level != 1 {
		t.Errorf("Expected floor level 1, got %d", s.Floors[0].Level)
	}

	if s.ActiveFloorID != s.Floors[0].ID {
		t.Error("ActiveFloorID not set to new floor ID")
	}
}

func TestMigrateToMultiFloor_WithFloorPlan(t *testing.T) {
	now := time.Now()
	floorPlan := &survey.FloorPlan{
		ImageData: "base64data",
		Width:     1000,
		Height:    800,
	}

	s := &survey.Survey{
		ID:        "test-survey",
		CreatedAt: now,
		FloorPlan: floorPlan,
		Samples:   nil,
	}

	migrated := survey.MigrateToMultiFloor(s)
	if !migrated {
		t.Error("Expected migration for survey with floor plan")
	}

	if len(s.Floors) != 1 {
		t.Errorf("Expected 1 floor after migration, got %d", len(s.Floors))
	}

	if s.Floors[0].FloorPlan != floorPlan {
		t.Error("Floor plan not migrated to floor")
	}

	// Legacy field should be cleared
	if s.FloorPlan != nil {
		t.Error("Legacy FloorPlan field should be cleared after migration")
	}
}

func TestMigrateToMultiFloor_WithSamples(t *testing.T) {
	now := time.Now()
	samples := []*survey.SamplePoint{
		{X: 100, Y: 100, Timestamp: now},
		{X: 200, Y: 200, Timestamp: now},
	}

	s := &survey.Survey{
		ID:        "test-survey",
		CreatedAt: now,
		FloorPlan: nil,
		Samples:   samples,
	}

	migrated := survey.MigrateToMultiFloor(s)
	if !migrated {
		t.Error("Expected migration for survey with samples")
	}

	if len(s.Floors) != 1 {
		t.Errorf("Expected 1 floor after migration, got %d", len(s.Floors))
	}

	if len(s.Floors[0].Samples) != 2 {
		t.Errorf("Expected 2 samples in floor, got %d", len(s.Floors[0].Samples))
	}

	// Legacy field should be cleared
	if s.Samples != nil {
		t.Error("Legacy Samples field should be cleared after migration")
	}
}

func TestMigrateToMultiFloor_WithBothFloorPlanAndSamples(t *testing.T) {
	now := time.Now()
	floorPlan := &survey.FloorPlan{
		ImageData: "base64data",
		Width:     500,
		Height:    400,
	}
	samples := []*survey.SamplePoint{
		{X: 50, Y: 50, Timestamp: now},
	}

	s := &survey.Survey{
		ID:        "test-survey",
		CreatedAt: now,
		FloorPlan: floorPlan,
		Samples:   samples,
	}

	migrated := survey.MigrateToMultiFloor(s)
	if !migrated {
		t.Error("Expected migration")
	}

	floor := s.Floors[0]
	if floor.FloorPlan != floorPlan {
		t.Error("Floor plan not migrated")
	}
	if len(floor.Samples) != 1 {
		t.Errorf("Expected 1 sample, got %d", len(floor.Samples))
	}
	if s.FloorPlan != nil || s.Samples != nil {
		t.Error("Legacy fields should be cleared")
	}
}

func TestNeedsMigration(t *testing.T) {
	tests := []struct {
		name     string
		survey   *survey.Survey
		expected bool
	}{
		{
			name: "has floors - no migration needed",
			survey: &survey.Survey{
				Floors: []*survey.Floor{{ID: "f1"}},
			},
			expected: false,
		},
		{
			name: "no floors, no data - no migration needed",
			survey: &survey.Survey{
				FloorPlan: nil,
				Samples:   nil,
			},
			expected: false,
		},
		{
			name: "no floors, has floor plan - needs migration",
			survey: &survey.Survey{
				FloorPlan: &survey.FloorPlan{Width: 100},
			},
			expected: true,
		},
		{
			name: "no floors, has samples - needs migration",
			survey: &survey.Survey{
				Samples: []*survey.SamplePoint{{X: 10}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := survey.NeedsMigration(tt.survey)
			if result != tt.expected {
				t.Errorf("NeedsMigration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSurvey_GetFloorByID(t *testing.T) {
	floor1 := &survey.Floor{ID: "floor-1", Name: "Floor 1"}
	floor2 := &survey.Floor{ID: "floor-2", Name: "Floor 2"}

	s := &survey.Survey{
		Floors: []*survey.Floor{floor1, floor2},
	}

	tests := []struct {
		name     string
		floorID  string
		expected *survey.Floor
	}{
		{"existing floor 1", "floor-1", floor1},
		{"existing floor 2", "floor-2", floor2},
		{"non-existent floor", "floor-3", nil},
		{"empty ID", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.GetFloorByID(tt.floorID)
			if result != tt.expected {
				t.Errorf("GetFloorByID(%s) = %v, want %v", tt.floorID, result, tt.expected)
			}
		})
	}
}

func TestSurvey_GetActiveFloor_WithActiveFloorID(t *testing.T) {
	floor1 := &survey.Floor{ID: "floor-1", Name: "Floor 1"}
	floor2 := &survey.Floor{ID: "floor-2", Name: "Floor 2"}

	s := &survey.Survey{
		Floors:        []*survey.Floor{floor1, floor2},
		ActiveFloorID: "floor-2",
	}

	result := s.GetActiveFloor()
	if result != floor2 {
		t.Errorf("GetActiveFloor() = %v, want floor-2", result)
	}
}

func TestSurvey_GetActiveFloor_NoActiveFloorID(t *testing.T) {
	floor1 := &survey.Floor{ID: "floor-1", Name: "Floor 1"}
	floor2 := &survey.Floor{ID: "floor-2", Name: "Floor 2"}

	s := &survey.Survey{
		Floors:        []*survey.Floor{floor1, floor2},
		ActiveFloorID: "",
	}

	result := s.GetActiveFloor()
	if result != floor1 {
		t.Errorf("GetActiveFloor() with empty ActiveFloorID should return first floor")
	}
}

func TestSurvey_GetActiveFloor_InvalidActiveFloorID(t *testing.T) {
	floor1 := &survey.Floor{ID: "floor-1", Name: "Floor 1"}

	s := &survey.Survey{
		Floors:        []*survey.Floor{floor1},
		ActiveFloorID: "non-existent",
	}

	result := s.GetActiveFloor()
	if result != nil {
		t.Error("GetActiveFloor() with invalid ActiveFloorID should return nil")
	}
}

func TestSurvey_GetActiveFloor_NoFloors(t *testing.T) {
	s := &survey.Survey{
		Floors:        nil,
		ActiveFloorID: "some-id",
	}

	result := s.GetActiveFloor()
	if result != nil {
		t.Error("GetActiveFloor() with no floors should return nil")
	}
}

func TestSurvey_GetAllSamples_MultiFloor(t *testing.T) {
	sample1 := &survey.SamplePoint{X: 10, Y: 10}
	sample2 := &survey.SamplePoint{X: 20, Y: 20}
	sample3 := &survey.SamplePoint{X: 30, Y: 30}

	s := &survey.Survey{
		Floors: []*survey.Floor{
			{ID: "floor-1", Samples: []*survey.SamplePoint{sample1}},
			{ID: "floor-2", Samples: []*survey.SamplePoint{sample2, sample3}},
		},
	}

	samples := s.GetAllSamples()
	if len(samples) != 3 {
		t.Errorf("GetAllSamples() returned %d samples, want 3", len(samples))
	}
}

func TestSurvey_GetAllSamples_IncludesLegacy(t *testing.T) {
	legacySample := &survey.SamplePoint{X: 100, Y: 100}
	floorSample := &survey.SamplePoint{X: 200, Y: 200}

	s := &survey.Survey{
		Floors: []*survey.Floor{
			{ID: "floor-1", Samples: []*survey.SamplePoint{floorSample}},
		},
		Samples: []*survey.SamplePoint{legacySample}, // Legacy samples
	}

	samples := s.GetAllSamples()
	if len(samples) != 2 {
		t.Errorf("GetAllSamples() should include legacy samples, got %d", len(samples))
	}
}
