package survey_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestManager_AddFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	// Create a survey first
	s, err := m.CreateSurvey("Test Survey", "Description", "en0", survey.TypePassive)
	if err != nil {
		t.Fatalf("Failed to create survey: %v", err)
	}

	// Add a floor
	floor, err := m.AddFloor(s.ID, "Second Floor", 2)
	if err != nil {
		t.Fatalf("Failed to add floor: %v", err)
	}

	if floor == nil {
		t.Fatal("Expected non-nil floor")
	}
	if floor.Name != "Second Floor" {
		t.Errorf("Expected floor name 'Second Floor', got '%s'", floor.Name)
	}
	if floor.Level != 2 {
		t.Errorf("Expected floor level 2, got %d", floor.Level)
	}
	if floor.ID == "" {
		t.Error("Expected non-empty floor ID")
	}

	// Verify survey has 2 floors now (default + new)
	updated, _ := m.GetSurvey(s.ID)
	if len(updated.Floors) != 2 {
		t.Errorf("Expected 2 floors, got %d", len(updated.Floors))
	}
}

func TestManager_AddFloor_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	_, err := m.AddFloor("non-existent", "Floor", 1)
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_UpdateFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	floorID := s.Floors[0].ID

	err := m.UpdateFloor(s.ID, floorID, "Updated Name", 3)
	if err != nil {
		t.Fatalf("Failed to update floor: %v", err)
	}

	floor, err := m.GetFloor(s.ID, floorID)
	if err != nil {
		t.Fatalf("Failed to get floor: %v", err)
	}

	if floor.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", floor.Name)
	}
	if floor.Level != 3 {
		t.Errorf("Expected level 3, got %d", floor.Level)
	}
}

func TestManager_UpdateFloor_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	err := m.UpdateFloor("non-existent", "floor", "Name", 1)
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_UpdateFloor_FloorNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	err := m.UpdateFloor(s.ID, "non-existent-floor", "Name", 1)
	if err == nil {
		t.Error("Expected error for non-existent floor")
	}
}

func TestManager_DeleteFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	// Add a second floor
	floor2, _ := m.AddFloor(s.ID, "Floor 2", 2)

	// Delete the second floor
	err := m.DeleteFloor(s.ID, floor2.ID)
	if err != nil {
		t.Fatalf("Failed to delete floor: %v", err)
	}

	updated, _ := m.GetSurvey(s.ID)
	if len(updated.Floors) != 1 {
		t.Errorf("Expected 1 floor after deletion, got %d", len(updated.Floors))
	}
}

func TestManager_DeleteFloor_LastFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	floorID := s.Floors[0].ID

	// Try to delete the last floor - should fail
	err := m.DeleteFloor(s.ID, floorID)
	if err == nil {
		t.Error("Expected error when deleting last floor")
	}
	if err.Error() != "cannot delete the last floor" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestManager_DeleteFloor_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	err := m.DeleteFloor("non-existent", "floor")
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_DeleteFloor_FloorNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	// Add a second floor so we can try to delete a non-existent one
	_, _ = m.AddFloor(s.ID, "Floor 2", 2)

	err := m.DeleteFloor(s.ID, "non-existent-floor")
	if err == nil {
		t.Error("Expected error for non-existent floor")
	}
}

func TestManager_DeleteFloor_ActiveFloorReset(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	floor1ID := s.Floors[0].ID

	// Add a second floor and make it active
	floor2, _ := m.AddFloor(s.ID, "Floor 2", 2)
	_ = m.SetActiveFloor(s.ID, floor2.ID)

	// Delete the active floor
	err := m.DeleteFloor(s.ID, floor2.ID)
	if err != nil {
		t.Fatalf("Failed to delete floor: %v", err)
	}

	// Active floor should switch to first remaining floor
	updated, _ := m.GetSurvey(s.ID)
	if updated.ActiveFloorID != floor1ID {
		t.Error("Expected active floor to switch to remaining floor")
	}
}

func TestManager_SetActiveFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	floor2, _ := m.AddFloor(s.ID, "Floor 2", 2)

	err := m.SetActiveFloor(s.ID, floor2.ID)
	if err != nil {
		t.Fatalf("Failed to set active floor: %v", err)
	}

	updated, _ := m.GetSurvey(s.ID)
	if updated.ActiveFloorID != floor2.ID {
		t.Error("Active floor not updated")
	}
}

func TestManager_SetActiveFloor_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	err := m.SetActiveFloor("non-existent", "floor")
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_SetActiveFloor_FloorNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	err := m.SetActiveFloor(s.ID, "non-existent-floor")
	if err == nil {
		t.Error("Expected error for non-existent floor")
	}
}

func TestManager_GetFloors(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_, _ = m.AddFloor(s.ID, "Floor 2", 2)

	floors, err := m.GetFloors(s.ID)
	if err != nil {
		t.Fatalf("Failed to get floors: %v", err)
	}

	if len(floors) != 2 {
		t.Errorf("Expected 2 floors, got %d", len(floors))
	}
}

func TestManager_GetFloors_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	_, err := m.GetFloors("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_GetFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	floorID := s.Floors[0].ID

	floor, err := m.GetFloor(s.ID, floorID)
	if err != nil {
		t.Fatalf("Failed to get floor: %v", err)
	}

	if floor.ID != floorID {
		t.Error("Retrieved wrong floor")
	}
}

func TestManager_GetFloor_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	_, err := m.GetFloor("non-existent", "floor")
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_GetFloor_FloorNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	_, err := m.GetFloor(s.ID, "non-existent-floor")
	if err == nil {
		t.Error("Expected error for non-existent floor")
	}
}

func TestManager_AddSampleToFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_ = m.StartSurvey(s.ID)

	floorID := s.Floors[0].ID
	sampleData := &survey.PassiveSample{
		Networks: []*wifi.ScannedNetwork{{SSID: "Test", Signal: -50}},
	}

	err := m.AddSampleToFloor(s.ID, floorID, 100, 200, sampleData)
	if err != nil {
		t.Fatalf("Failed to add sample to floor: %v", err)
	}

	floor, _ := m.GetFloor(s.ID, floorID)
	if len(floor.Samples) != 1 {
		t.Errorf("Expected 1 sample, got %d", len(floor.Samples))
	}
	if floor.Samples[0].X != 100 || floor.Samples[0].Y != 200 {
		t.Error("Sample coordinates incorrect")
	}
}

func TestManager_AddSampleToFloor_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	err := m.AddSampleToFloor("non-existent", "floor", 0, 0, nil)
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_AddSampleToFloor_SurveyNotInProgress(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	// Don't start the survey

	err := m.AddSampleToFloor(s.ID, s.Floors[0].ID, 0, 0, nil)
	if err == nil {
		t.Error("Expected error when survey not in progress")
	}
}

func TestManager_AddSampleToFloor_FloorNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_ = m.StartSurvey(s.ID)

	err := m.AddSampleToFloor(s.ID, "non-existent-floor", 0, 0, nil)
	if err == nil {
		t.Error("Expected error for non-existent floor")
	}
}

func TestManager_UpdateFloorPlanByFloorID(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	floorID := s.Floors[0].ID

	floorPlan := &survey.FloorPlan{
		ImageData: "base64data",
		Width:     800,
		Height:    600,
	}

	err := m.UpdateFloorPlanByFloorID(s.ID, floorID, floorPlan)
	if err != nil {
		t.Fatalf("Failed to update floor plan: %v", err)
	}

	floor, _ := m.GetFloor(s.ID, floorID)
	if floor.FloorPlan == nil {
		t.Error("Floor plan not set")
	}
	if floor.FloorPlan.Width != 800 {
		t.Errorf("Expected width 800, got %d", floor.FloorPlan.Width)
	}
}

func TestManager_UpdateFloorPlanByFloorID_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	err := m.UpdateFloorPlanByFloorID("non-existent", "floor", nil)
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_UpdateFloorPlanByFloorID_FloorNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	err := m.UpdateFloorPlanByFloorID(s.ID, "non-existent-floor", nil)
	if err == nil {
		t.Error("Expected error for non-existent floor")
	}
}

func TestManager_UpdateSurveySettings(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	// Create initial survey for baseline
	_, _ = m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	tests := []struct {
		name         string
		surveyType   survey.Type
		iperfServer  string
		testDuration int
		expectError  bool
	}{
		{
			name:         "valid passive type",
			surveyType:   survey.TypePassive,
			iperfServer:  "",
			testDuration: 5,
			expectError:  false,
		},
		{
			name:         "valid active type",
			surveyType:   survey.TypeActive,
			iperfServer:  "",
			testDuration: 5,
			expectError:  false,
		},
		{
			name:         "valid throughput type",
			surveyType:   survey.TypeThroughput,
			iperfServer:  "192.168.1.100",
			testDuration: 10,
			expectError:  false,
		},
		{
			name:         "invalid survey type",
			surveyType:   survey.Type("invalid"),
			iperfServer:  "",
			testDuration: 5,
			expectError:  true,
		},
		{
			name:         "zero duration uses default",
			surveyType:   survey.TypePassive,
			iperfServer:  "",
			testDuration: 0,
			expectError:  false,
		},
		{
			name:         "exceed max duration gets clamped",
			surveyType:   survey.TypePassive,
			iperfServer:  "",
			testDuration: 100,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Need to recreate survey for each test to reset state
			s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

			err := m.UpdateSurveySettings(s.ID, tt.surveyType, tt.iperfServer, tt.testDuration)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestManager_UpdateSurveySettings_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	err := m.UpdateSurveySettings("non-existent", survey.TypePassive, "", 5)
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}

func TestManager_UpdateSurveySettings_SurveyAlreadyStarted(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_ = m.StartSurvey(s.ID)

	err := m.UpdateSurveySettings(s.ID, survey.TypeActive, "", 5)
	if err == nil {
		t.Error("Expected error when survey already started")
	}
}

func TestManager_UpdateFloorPlan_NoActiveFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	// Create survey and remove all floors to test no active floor case
	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)

	// Set active floor to non-existent ID
	m.SetSurvey(&survey.Survey{
		ID:            s.ID,
		Name:          s.Name,
		Floors:        []*survey.Floor{},
		ActiveFloorID: "non-existent",
	})

	err := m.UpdateFloorPlan(s.ID, &survey.FloorPlan{Width: 100})
	if err == nil {
		t.Error("Expected error when no active floor")
	}
}

func TestManager_AddSample_NoActiveFloor(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_ = m.StartSurvey(s.ID)

	// Override to have invalid active floor
	m.SetSurvey(&survey.Survey{
		ID:            s.ID,
		Name:          s.Name,
		Status:        survey.StatusInProgress,
		Floors:        []*survey.Floor{},
		ActiveFloorID: "non-existent",
	})

	err := m.AddSample(s.ID, 100, 100, nil)
	if err == nil {
		t.Error("Expected error when no active floor")
	}
}

func TestManager_StartSurvey_AlreadyInProgress(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	s, _ := m.CreateSurvey("Test", "Desc", "en0", survey.TypePassive)
	_ = m.StartSurvey(s.ID)

	err := m.StartSurvey(s.ID)
	if err == nil {
		t.Error("Expected error when survey already in progress")
	}
}
