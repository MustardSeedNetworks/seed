package survey

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/iperf"
	"github.com/krisarmstrong/luminetiq/internal/wifi"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	wifiScanner := wifi.NewScanner("wlan0")
	wifiManager := wifi.NewManager("wlan0")
	iperfManager := iperf.NewManager()

	mgr := NewManager(tmpDir, wifiScanner, wifiManager, iperfManager)

	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}

	if mgr.storagePath != tmpDir {
		t.Errorf("storagePath = %v, want %v", mgr.storagePath, tmpDir)
	}

	if mgr.surveys == nil {
		t.Error("surveys map is nil")
	}

	if mgr.wifiScanner == nil {
		t.Error("wifiScanner is nil")
	}

	if mgr.wifiManager == nil {
		t.Error("wifiManager is nil")
	}

	if mgr.iperfManager == nil {
		t.Error("iperfManager is nil")
	}
}

func TestCreateSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	tests := []struct {
		name        string
		surveyName  string
		description string
		iface       string
		surveyType  SurveyType
		wantErr     bool
	}{
		{
			name:        "valid passive survey",
			surveyName:  "Test Passive",
			description: "Test passive survey",
			iface:       "wlan0",
			surveyType:  SurveyTypePassive,
			wantErr:     false,
		},
		{
			name:        "valid active survey",
			surveyName:  "Test Active",
			description: "Test active survey",
			iface:       "wlan0",
			surveyType:  SurveyTypeActive,
			wantErr:     false,
		},
		{
			name:        "valid throughput survey",
			surveyName:  "Test Throughput",
			description: "Test throughput survey",
			iface:       "wlan0",
			surveyType:  SurveyTypeThroughput,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			survey, err := mgr.CreateSurvey(tt.surveyName, tt.description, tt.iface, tt.surveyType)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateSurvey() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateSurvey() error = %v, want nil", err)
				return
			}

			if survey == nil {
				t.Fatal("CreateSurvey() returned nil survey")
			}

			if survey.ID == "" {
				t.Error("Survey ID is empty")
			}

			if survey.Name != tt.surveyName {
				t.Errorf("Survey Name = %v, want %v", survey.Name, tt.surveyName)
			}

			if survey.Description != tt.description {
				t.Errorf("Survey Description = %v, want %v", survey.Description, tt.description)
			}

			if survey.Interface != tt.iface {
				t.Errorf("Survey Interface = %v, want %v", survey.Interface, tt.iface)
			}

			if survey.SurveyType != tt.surveyType {
				t.Errorf("Survey Type = %v, want %v", survey.SurveyType, tt.surveyType)
			}

			if survey.Status != StatusCreated {
				t.Errorf("Survey Status = %v, want %v", survey.Status, StatusCreated)
			}

			if survey.Samples == nil {
				t.Error("Survey Samples is nil")
			}

			if survey.CreatedAt.IsZero() {
				t.Error("Survey CreatedAt is zero")
			}

			if survey.UpdatedAt.IsZero() {
				t.Error("Survey UpdatedAt is zero")
			}
		})
	}
}

func TestGetSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	// Create a survey first
	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing survey",
			id:      survey.ID,
			wantErr: false,
		},
		{
			name:    "non-existent survey",
			id:      "non-existent-id",
			wantErr: true,
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mgr.GetSurvey(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("GetSurvey() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("GetSurvey() error = %v, want nil", err)
				return
			}

			if result == nil {
				t.Fatal("GetSurvey() returned nil")
			}

			if result.ID != tt.id {
				t.Errorf("Survey ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}

func TestListSurveys(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	// Initially empty
	surveys := mgr.ListSurveys()
	if len(surveys) != 0 {
		t.Errorf("ListSurveys() returned %d surveys, want 0", len(surveys))
	}

	// Create surveys
	_, err := mgr.CreateSurvey("Survey 1", "Desc 1", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	_, err = mgr.CreateSurvey("Survey 2", "Desc 2", "wlan0", SurveyTypeActive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	_, err = mgr.CreateSurvey("Survey 3", "Desc 3", "wlan0", SurveyTypeThroughput)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Should now have 3 surveys
	surveys = mgr.ListSurveys()
	if len(surveys) != 3 {
		t.Errorf("ListSurveys() returned %d surveys, want 3", len(surveys))
	}

	// Verify surveys are not nil
	for i, s := range surveys {
		if s == nil {
			t.Errorf("Survey at index %d is nil", i)
		}
	}
}

func TestDeleteSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "delete existing survey",
			id:      survey.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existent survey",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.DeleteSurvey(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("DeleteSurvey() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteSurvey() error = %v, want nil", err)
			}

			// Verify survey is deleted
			_, err = mgr.GetSurvey(tt.id)
			if err == nil {
				t.Error("GetSurvey() after delete succeeded, want error")
			}
		})
	}
}

func TestStartSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	tests := []struct {
		name        string
		id          string
		setupStatus SurveyStatus
		wantErr     bool
	}{
		{
			name:        "start created survey",
			id:          survey.ID,
			setupStatus: StatusCreated,
			wantErr:     false,
		},
		{
			name:        "start non-existent survey",
			id:          "non-existent-id",
			setupStatus: StatusCreated,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.StartSurvey(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("StartSurvey() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("StartSurvey() error = %v, want nil", err)
				return
			}

			// Verify status changed
			result, err := mgr.GetSurvey(tt.id)
			if err != nil {
				t.Fatalf("GetSurvey() failed: %v", err)
			}

			if result.Status != StatusInProgress {
				t.Errorf("Survey Status = %v, want %v", result.Status, StatusInProgress)
			}
		})
	}
}

func TestPauseSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Start survey first
	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "pause in-progress survey",
			id:      survey.ID,
			wantErr: false,
		},
		{
			name:    "pause non-existent survey",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.PauseSurvey(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("PauseSurvey() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("PauseSurvey() error = %v, want nil", err)
				return
			}

			// Verify status changed
			result, err := mgr.GetSurvey(tt.id)
			if err != nil {
				t.Fatalf("GetSurvey() failed: %v", err)
			}

			if result.Status != StatusPaused {
				t.Errorf("Survey Status = %v, want %v", result.Status, StatusPaused)
			}
		})
	}
}

func TestCompleteSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Start survey first
	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "complete in-progress survey",
			id:      survey.ID,
			wantErr: false,
		},
		{
			name:    "complete non-existent survey",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.CompleteSurvey(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("CompleteSurvey() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("CompleteSurvey() error = %v, want nil", err)
				return
			}

			// Verify status changed
			result, err := mgr.GetSurvey(tt.id)
			if err != nil {
				t.Fatalf("GetSurvey() failed: %v", err)
			}

			if result.Status != StatusCompleted {
				t.Errorf("Survey Status = %v, want %v", result.Status, StatusCompleted)
			}
		})
	}
}

func TestStateTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Test valid state transitions
	// Created -> InProgress
	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Errorf("Created -> InProgress failed: %v", err)
	}

	// InProgress -> Paused
	err = mgr.PauseSurvey(survey.ID)
	if err != nil {
		t.Errorf("InProgress -> Paused failed: %v", err)
	}

	// Paused -> InProgress (resume)
	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Errorf("Paused -> InProgress failed: %v", err)
	}

	// InProgress -> Completed
	err = mgr.CompleteSurvey(survey.ID)
	if err != nil {
		t.Errorf("InProgress -> Completed failed: %v", err)
	}

	// Verify final state
	result, _ := mgr.GetSurvey(survey.ID)
	if result.Status != StatusCompleted {
		t.Errorf("Final status = %v, want %v", result.Status, StatusCompleted)
	}
}

func TestUpdateFloorPlan(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	floorPlan := &FloorPlan{
		ImageData: "base64encodeddata",
		Width:     1000,
		Height:    800,
	}

	tests := []struct {
		name      string
		id        string
		floorPlan *FloorPlan
		wantErr   bool
	}{
		{
			name:      "update with valid floor plan",
			id:        survey.ID,
			floorPlan: floorPlan,
			wantErr:   false,
		},
		{
			name:      "update non-existent survey",
			id:        "non-existent-id",
			floorPlan: floorPlan,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.UpdateFloorPlan(tt.id, tt.floorPlan)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdateFloorPlan() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateFloorPlan() error = %v, want nil", err)
				return
			}

			// Verify floor plan updated
			result, err := mgr.GetSurvey(tt.id)
			if err != nil {
				t.Fatalf("GetSurvey() failed: %v", err)
			}

			if result.FloorPlan == nil {
				t.Fatal("FloorPlan is nil after update")
			}

			if result.FloorPlan.Width != floorPlan.Width {
				t.Errorf("FloorPlan Width = %v, want %v", result.FloorPlan.Width, floorPlan.Width)
			}

			if result.FloorPlan.Height != floorPlan.Height {
				t.Errorf("FloorPlan Height = %v, want %v", result.FloorPlan.Height, floorPlan.Height)
			}
		})
	}
}

func TestAddSample(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Start survey to allow samples
	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	passiveData := map[string]interface{}{
		"networks": []interface{}{
			map[string]interface{}{
				"ssid":  "TestNetwork",
				"bssid": "00:11:22:33:44:55",
				"rssi":  -50,
			},
		},
	}

	tests := []struct {
		name       string
		id         string
		x          int
		y          int
		sampleData map[string]interface{}
		wantErr    bool
	}{
		{
			name:       "add valid sample",
			id:         survey.ID,
			x:          100,
			y:          200,
			sampleData: passiveData,
			wantErr:    false,
		},
		{
			name:       "add sample to non-existent survey",
			id:         "non-existent-id",
			x:          100,
			y:          200,
			sampleData: passiveData,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.AddSample(tt.id, tt.x, tt.y, tt.sampleData)

			if tt.wantErr {
				if err == nil {
					t.Error("AddSample() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("AddSample() error = %v, want nil", err)
				return
			}

			// Verify sample was added
			result, err := mgr.GetSurvey(tt.id)
			if err != nil {
				t.Fatalf("GetSurvey() failed: %v", err)
			}

			if len(result.Samples) == 0 {
				t.Error("No samples found after AddSample()")
				return
			}

			lastSample := result.Samples[len(result.Samples)-1]
			if lastSample.X != tt.x {
				t.Errorf("Sample X = %v, want %v", lastSample.X, tt.x)
			}

			if lastSample.Y != tt.y {
				t.Errorf("Sample Y = %v, want %v", lastSample.Y, tt.y)
			}

			if lastSample.Timestamp.IsZero() {
				t.Error("Sample Timestamp is zero")
			}
		})
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	// Create a survey
	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, survey.ID+".json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Survey file not created at %s", filePath)
	}

	// Create new manager to load surveys
	mgr2 := NewManager(tmpDir, nil, nil, nil)

	// Load surveys from disk
	err = mgr2.LoadSurveys()
	if err != nil {
		t.Fatalf("LoadSurveys() failed: %v", err)
	}

	// Verify survey was loaded
	loaded, err := mgr2.GetSurvey(survey.ID)
	if err != nil {
		t.Errorf("Failed to load survey: %v", err)
	}

	if loaded == nil {
		t.Fatal("Loaded survey is nil")
	}

	if loaded.Name != survey.Name {
		t.Errorf("Loaded survey Name = %v, want %v", loaded.Name, survey.Name)
	}

	if loaded.Description != survey.Description {
		t.Errorf("Loaded survey Description = %v, want %v", loaded.Description, survey.Description)
	}

	// Delete and verify file removed
	err = mgr2.DeleteSurvey(survey.ID)
	if err != nil {
		t.Errorf("DeleteSurvey() failed: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Survey file not deleted")
	}
}

func TestConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := mgr.CreateSurvey("Survey", "Desc", "wlan0", SurveyTypePassive)
			if err != nil {
				t.Errorf("Concurrent CreateSurvey() failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Should have created all surveys
	surveys := mgr.ListSurveys()
	if len(surveys) != numGoroutines {
		t.Errorf("Expected %d surveys, got %d", numGoroutines, len(surveys))
	}
}

func TestConcurrentSampleAddition(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	var wg sync.WaitGroup
	numSamples := 50

	sampleData := map[string]interface{}{
		"networks": []interface{}{
			map[string]interface{}{
				"ssid": "Test",
				"rssi": -60,
			},
		},
	}

	// Concurrent sample additions
	for i := 0; i < numSamples; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := mgr.AddSample(survey.ID, n, n, sampleData)
			if err != nil {
				t.Errorf("Concurrent AddSample() failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all samples added
	result, err := mgr.GetSurvey(survey.ID)
	if err != nil {
		t.Fatalf("GetSurvey() failed: %v", err)
	}

	if len(result.Samples) != numSamples {
		t.Errorf("Expected %d samples, got %d", numSamples, len(result.Samples))
	}
}

func TestSurveyTimestamps(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	beforeCreate := time.Now()
	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}
	afterCreate := time.Now()

	// Verify CreatedAt is within expected time range
	if survey.CreatedAt.Before(beforeCreate) || survey.CreatedAt.After(afterCreate) {
		t.Error("CreatedAt timestamp out of expected range")
	}

	// Verify UpdatedAt is set
	if survey.UpdatedAt.Before(beforeCreate) || survey.UpdatedAt.After(afterCreate) {
		t.Error("UpdatedAt timestamp out of expected range")
	}

	// Complete survey
	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	err = mgr.CompleteSurvey(survey.ID)
	if err != nil {
		t.Fatalf("CompleteSurvey() failed: %v", err)
	}

	result, _ := mgr.GetSurvey(survey.ID)
	if result.Status != StatusCompleted {
		t.Error("Survey not marked as completed")
	}
}

func TestSampleCount(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, nil, nil, nil)

	survey, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", SurveyTypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	err = mgr.StartSurvey(survey.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	sampleData := map[string]interface{}{
		"networks": []interface{}{
			map[string]interface{}{"ssid": "Test", "rssi": -60},
		},
	}

	// Add multiple samples
	for i := 0; i < 5; i++ {
		err = mgr.AddSample(survey.ID, i*10, i*10, sampleData)
		if err != nil {
			t.Errorf("AddSample() failed: %v", err)
		}
	}

	result, err := mgr.GetSurvey(survey.ID)
	if err != nil {
		t.Fatalf("GetSurvey() failed: %v", err)
	}

	if len(result.Samples) != 5 {
		t.Errorf("Sample count = %d, want 5", len(result.Samples))
	}
}
