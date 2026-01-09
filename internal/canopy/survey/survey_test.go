package survey_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/iperf"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	wifiScanner := wifi.NewScanner("wlan0")
	wifiManager := wifi.NewManager("wlan0")
	iperfManager := iperf.NewManager()

	mgr := survey.NewManager(tmpDir, wifiScanner, wifiManager, iperfManager)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}

	if mgr.GetStoragePath() != tmpDir {
		t.Errorf("storagePath = %v, want %v", mgr.GetStoragePath(), tmpDir)
	}

	if mgr.GetSurveys() == nil {
		t.Error("surveys map is nil")
	}

	if mgr.GetWifiScanner() == nil {
		t.Error("wifiScanner is nil")
	}

	if mgr.GetWifiManager() == nil {
		t.Error("wifiManager is nil")
	}

	if mgr.GetIperfManager() == nil {
		t.Error("iperfManager is nil")
	}
}

// createSurveyTestCase defines a test case for TestCreateSurvey.
type createSurveyTestCase struct {
	name        string
	surveyName  string
	description string
	iface       string
	surveyType  survey.Type
	wantErr     bool
}

// assertSurveyFields validates basic survey fields match expected values.
func assertSurveyFields(t *testing.T, s *survey.Survey, tc createSurveyTestCase) {
	t.Helper()

	if s.ID == "" {
		t.Error("Survey ID is empty")
	}
	if s.Name != tc.surveyName {
		t.Errorf("Survey Name = %v, want %v", s.Name, tc.surveyName)
	}
	if s.Description != tc.description {
		t.Errorf("Survey Description = %v, want %v", s.Description, tc.description)
	}
	if s.Interface != tc.iface {
		t.Errorf("Survey Interface = %v, want %v", s.Interface, tc.iface)
	}
	if s.SurveyType != tc.surveyType {
		t.Errorf("Survey Type = %v, want %v", s.SurveyType, tc.surveyType)
	}
	if s.Status != survey.StatusCreated {
		t.Errorf("Survey Status = %v, want %v", s.Status, survey.StatusCreated)
	}
}

// assertSurveyFloors validates survey has proper floor structure.
func assertSurveyFloors(t *testing.T, s *survey.Survey) {
	t.Helper()

	if len(s.Floors) == 0 {
		t.Error("Survey has no floors")
	}

	activeFloor := s.GetActiveFloor()
	if activeFloor == nil {
		t.Fatal("Survey has no active floor")
	}
	if activeFloor.Samples == nil {
		t.Error("Active floor Samples is nil")
	}
}

// assertSurveyTimestamps validates survey timestamps are set.
func assertSurveyTimestamps(t *testing.T, s *survey.Survey) {
	t.Helper()

	if s.CreatedAt.IsZero() {
		t.Error("Survey CreatedAt is zero")
	}
	if s.UpdatedAt.IsZero() {
		t.Error("Survey UpdatedAt is zero")
	}
}

func TestCreateSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	tests := []createSurveyTestCase{
		{
			name:        "valid passive survey",
			surveyName:  "Test Passive",
			description: "Test passive survey",
			iface:       "wlan0",
			surveyType:  survey.TypePassive,
			wantErr:     false,
		},
		{
			name:        "valid active survey",
			surveyName:  "Test Active",
			description: "Test active survey",
			iface:       "wlan0",
			surveyType:  survey.TypeActive,
			wantErr:     false,
		},
		{
			name:        "valid throughput survey",
			surveyName:  "Test Throughput",
			description: "Test throughput survey",
			iface:       "wlan0",
			surveyType:  survey.TypeThroughput,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := mgr.CreateSurvey(tt.surveyName, tt.description, tt.iface, tt.surveyType)

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

			if s == nil {
				t.Fatal("CreateSurvey() returned nil survey")
			}

			assertSurveyFields(t, s, tt)
			assertSurveyFloors(t, s)
			assertSurveyTimestamps(t, s)
		})
	}
}

func TestGetSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	// Create a survey first.
	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
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
			id:      s.ID,
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
			result, getErr := mgr.GetSurvey(tt.id)

			if tt.wantErr {
				if getErr == nil {
					t.Error("GetSurvey() error = nil, want error")
				}
				return
			}

			if getErr != nil {
				t.Errorf("GetSurvey() error = %v, want nil", getErr)
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
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	// Initially empty.
	surveys := mgr.ListSurveys()
	if len(surveys) != 0 {
		t.Errorf("ListSurveys() returned %d surveys, want 0", len(surveys))
	}

	// Create surveys.
	_, err := mgr.CreateSurvey("Survey 1", "Desc 1", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	_, err = mgr.CreateSurvey("Survey 2", "Desc 2", "wlan0", survey.TypeActive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	_, err = mgr.CreateSurvey("Survey 3", "Desc 3", "wlan0", survey.TypeThroughput)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Should now have 3 surveys.
	surveys = mgr.ListSurveys()
	if len(surveys) != 3 {
		t.Errorf("ListSurveys() returned %d surveys, want 3", len(surveys))
	}

	// Verify surveys are not nil.
	for i, s := range surveys {
		if s == nil {
			t.Errorf("Survey at index %d is nil", i)
		}
	}
}

func TestDeleteSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
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
			id:      s.ID,
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
			deleteErr := mgr.DeleteSurvey(tt.id)

			if tt.wantErr {
				if deleteErr == nil {
					t.Error("DeleteSurvey() error = nil, want error")
				}
				return
			}

			if deleteErr != nil {
				t.Errorf("DeleteSurvey() error = %v, want nil", deleteErr)
			}

			// Verify survey is deleted.
			_, getErr := mgr.GetSurvey(tt.id)
			if getErr == nil {
				t.Error("GetSurvey() after delete succeeded, want error")
			}
		})
	}
}

func TestStartSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	tests := []struct {
		name        string
		id          string
		setupStatus survey.Status
		wantErr     bool
	}{
		{
			name:        "start created survey",
			id:          s.ID,
			setupStatus: survey.StatusCreated,
			wantErr:     false,
		},
		{
			name:        "start non-existent survey",
			id:          "non-existent-id",
			setupStatus: survey.StatusCreated,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startErr := mgr.StartSurvey(tt.id)

			if tt.wantErr {
				if startErr == nil {
					t.Error("StartSurvey() error = nil, want error")
				}
				return
			}

			if startErr != nil {
				t.Errorf("StartSurvey() error = %v, want nil", startErr)
				return
			}

			// Verify status changed.
			result, getErr := mgr.GetSurvey(tt.id)
			if getErr != nil {
				t.Fatalf("GetSurvey() failed: %v", getErr)
			}

			if result.Status != survey.StatusInProgress {
				t.Errorf("Survey Status = %v, want %v", result.Status, survey.StatusInProgress)
			}
		})
	}
}

func TestPauseSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Start survey first.
	err = mgr.StartSurvey(s.ID)
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
			id:      s.ID,
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
			pauseErr := mgr.PauseSurvey(tt.id)

			if tt.wantErr {
				if pauseErr == nil {
					t.Error("PauseSurvey() error = nil, want error")
				}
				return
			}

			if pauseErr != nil {
				t.Errorf("PauseSurvey() error = %v, want nil", pauseErr)
				return
			}

			// Verify status changed.
			result, getErr := mgr.GetSurvey(tt.id)
			if getErr != nil {
				t.Fatalf("GetSurvey() failed: %v", getErr)
			}

			if result.Status != survey.StatusPaused {
				t.Errorf("Survey Status = %v, want %v", result.Status, survey.StatusPaused)
			}
		})
	}
}

func TestCompleteSurvey(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Start survey first.
	err = mgr.StartSurvey(s.ID)
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
			id:      s.ID,
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
			completeErr := mgr.CompleteSurvey(tt.id)

			if tt.wantErr {
				if completeErr == nil {
					t.Error("CompleteSurvey() error = nil, want error")
				}
				return
			}

			if completeErr != nil {
				t.Errorf("CompleteSurvey() error = %v, want nil", completeErr)
				return
			}

			// Verify status changed.
			result, getErr := mgr.GetSurvey(tt.id)
			if getErr != nil {
				t.Fatalf("GetSurvey() failed: %v", getErr)
			}

			if result.Status != survey.StatusCompleted {
				t.Errorf("Survey Status = %v, want %v", result.Status, survey.StatusCompleted)
			}
		})
	}
}

func TestStateTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Test valid state transitions.
	// Created -> InProgress.
	err = mgr.StartSurvey(s.ID)
	if err != nil {
		t.Errorf("Created -> InProgress failed: %v", err)
	}

	// InProgress -> Paused.
	err = mgr.PauseSurvey(s.ID)
	if err != nil {
		t.Errorf("InProgress -> Paused failed: %v", err)
	}

	// Paused -> InProgress (resume).
	err = mgr.StartSurvey(s.ID)
	if err != nil {
		t.Errorf("Paused -> InProgress failed: %v", err)
	}

	// InProgress -> Completed.
	err = mgr.CompleteSurvey(s.ID)
	if err != nil {
		t.Errorf("InProgress -> Completed failed: %v", err)
	}

	// Verify final state.
	result, _ := mgr.GetSurvey(s.ID)
	if result.Status != survey.StatusCompleted {
		t.Errorf("Final status = %v, want %v", result.Status, survey.StatusCompleted)
	}
}

// updateFloorPlanTestCase defines a test case for TestUpdateFloorPlan.
type updateFloorPlanTestCase struct {
	name      string
	id        string
	floorPlan *survey.FloorPlan
	wantErr   bool
}

// assertFloorPlanUpdated verifies that the floor plan was correctly updated on the survey.
func assertFloorPlanUpdated(
	t *testing.T,
	mgr *survey.Manager,
	surveyID string,
	expected *survey.FloorPlan,
) {
	t.Helper()

	result, getErr := mgr.GetSurvey(surveyID)
	if getErr != nil {
		t.Fatalf("GetSurvey() failed: %v", getErr)
	}

	activeFloor := result.GetActiveFloor()
	if activeFloor == nil {
		t.Fatal("No active floor found")
	}

	if activeFloor.FloorPlan == nil {
		t.Fatal("FloorPlan is nil after update")
	}

	if activeFloor.FloorPlan.Width != expected.Width {
		t.Errorf("FloorPlan Width = %v, want %v", activeFloor.FloorPlan.Width, expected.Width)
	}

	if activeFloor.FloorPlan.Height != expected.Height {
		t.Errorf("FloorPlan Height = %v, want %v", activeFloor.FloorPlan.Height, expected.Height)
	}
}

// runUpdateFloorPlanTest executes a single update floor plan test case.
func runUpdateFloorPlanTest(
	t *testing.T,
	mgr *survey.Manager,
	tc updateFloorPlanTestCase,
) {
	t.Helper()

	updateErr := mgr.UpdateFloorPlan(tc.id, tc.floorPlan)

	if tc.wantErr {
		if updateErr == nil {
			t.Error("UpdateFloorPlan() error = nil, want error")
		}
		return
	}

	if updateErr != nil {
		t.Errorf("UpdateFloorPlan() error = %v, want nil", updateErr)
		return
	}

	assertFloorPlanUpdated(t, mgr, tc.id, tc.floorPlan)
}

func TestUpdateFloorPlan(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	floorPlan := &survey.FloorPlan{
		ImageData: "base64encodeddata",
		Width:     1000,
		Height:    800,
	}

	tests := []updateFloorPlanTestCase{
		{
			name:      "update with valid floor plan",
			id:        s.ID,
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
			runUpdateFloorPlanTest(t, mgr, tt)
		})
	}
}

// addSampleTestCase defines test parameters for TestAddSample.
type addSampleTestCase struct {
	name       string
	id         string
	x          int
	y          int
	sampleData map[string]any
	wantErr    bool
}

// addSampleTestFixture holds shared test resources for AddSample tests.
type addSampleTestFixture struct {
	mgr         *survey.Manager
	validID     string
	passiveData map[string]any
}

// setupAddSampleTest creates a test fixture with a started survey.
func setupAddSampleTest(t *testing.T) *addSampleTestFixture {
	t.Helper()
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	err = mgr.StartSurvey(s.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	passiveData := map[string]any{
		"networks": []any{
			map[string]any{
				"ssid":  "TestNetwork",
				"bssid": "00:11:22:33:44:55",
				"rssi":  -50,
			},
		},
	}

	return &addSampleTestFixture{
		mgr:         mgr,
		validID:     s.ID,
		passiveData: passiveData,
	}
}

// assertSampleAdded verifies that a sample was correctly added to the survey.
func assertSampleAdded(t *testing.T, mgr *survey.Manager, surveyID string, wantX, wantY int) {
	t.Helper()
	result, err := mgr.GetSurvey(surveyID)
	if err != nil {
		t.Fatalf("GetSurvey() failed: %v", err)
	}

	samples := result.GetAllSamples()
	if len(samples) == 0 {
		t.Fatal("No samples found after AddSample()")
	}

	lastSample := samples[len(samples)-1]
	if lastSample.X != wantX {
		t.Errorf("Sample X = %v, want %v", lastSample.X, wantX)
	}
	if lastSample.Y != wantY {
		t.Errorf("Sample Y = %v, want %v", lastSample.Y, wantY)
	}
	if lastSample.Timestamp.IsZero() {
		t.Error("Sample Timestamp is zero")
	}
}

func TestAddSample(t *testing.T) {
	fixture := setupAddSampleTest(t)

	tests := []addSampleTestCase{
		{
			name:       "add valid sample",
			id:         fixture.validID,
			x:          100,
			y:          200,
			sampleData: fixture.passiveData,
			wantErr:    false,
		},
		{
			name:       "add sample to non-existent survey",
			id:         "non-existent-id",
			x:          100,
			y:          200,
			sampleData: fixture.passiveData,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fixture.mgr.AddSample(tt.id, tt.x, tt.y, tt.sampleData)

			if tt.wantErr {
				if err == nil {
					t.Error("AddSample() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("AddSample() error = %v, want nil", err)
			}

			assertSampleAdded(t, fixture.mgr, tt.id, tt.x, tt.y)
		})
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	// Create a survey.
	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	// Verify file was created.
	filePath := filepath.Join(tmpDir, s.ID+".json")
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		t.Errorf("Survey file not created at %s", filePath)
	}

	// Create new manager to load surveys.
	mgr2 := survey.NewManager(tmpDir, nil, nil, nil)

	// Load surveys from disk.
	err = mgr2.LoadSurveys()
	if err != nil {
		t.Fatalf("LoadSurveys() failed: %v", err)
	}

	// Verify survey was loaded.
	loaded, err := mgr2.GetSurvey(s.ID)
	if err != nil {
		t.Errorf("Failed to load survey: %v", err)
	}

	if loaded == nil {
		t.Fatal("Loaded survey is nil")
	}

	if loaded.Name != s.Name {
		t.Errorf("Loaded survey Name = %v, want %v", loaded.Name, s.Name)
	}

	if loaded.Description != s.Description {
		t.Errorf("Loaded survey Description = %v, want %v", loaded.Description, s.Description)
	}

	// Delete and verify file removed.
	err = mgr2.DeleteSurvey(s.ID)
	if err != nil {
		t.Errorf("DeleteSurvey() failed: %v", err)
	}

	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Error("Survey file not deleted")
	}
}

func TestConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent creates.
	for i := range numGoroutines {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			_, err := mgr.CreateSurvey("Survey", "Desc", "wlan0", survey.TypePassive)
			if err != nil {
				t.Errorf("Concurrent CreateSurvey() failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Should have created all surveys.
	surveys := mgr.ListSurveys()
	if len(surveys) != numGoroutines {
		t.Errorf("Expected %d surveys, got %d", numGoroutines, len(surveys))
	}
}

func TestConcurrentSampleAddition(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	err = mgr.StartSurvey(s.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	var wg sync.WaitGroup
	numSamples := 50

	sampleData := map[string]any{
		"networks": []any{
			map[string]any{
				"ssid": "Test",
				"rssi": -60,
			},
		},
	}

	// Concurrent sample additions.
	for i := range numSamples {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			addErr := mgr.AddSample(s.ID, n, n, sampleData)
			if addErr != nil {
				t.Errorf("Concurrent AddSample() failed: %v", addErr)
			}
		}(i)
	}

	wg.Wait()

	// Verify all samples added (now on the active floor).
	result, err := mgr.GetSurvey(s.ID)
	if err != nil {
		t.Fatalf("GetSurvey() failed: %v", err)
	}

	samples := result.GetAllSamples()
	if len(samples) != numSamples {
		t.Errorf("Expected %d samples, got %d", numSamples, len(samples))
	}
}

func TestSurveyTimestamps(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	beforeCreate := time.Now()
	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}
	afterCreate := time.Now()

	// Verify CreatedAt is within expected time range.
	if s.CreatedAt.Before(beforeCreate) || s.CreatedAt.After(afterCreate) {
		t.Error("CreatedAt timestamp out of expected range")
	}

	// Verify UpdatedAt is set.
	if s.UpdatedAt.Before(beforeCreate) || s.UpdatedAt.After(afterCreate) {
		t.Error("UpdatedAt timestamp out of expected range")
	}

	// Complete survey.
	err = mgr.StartSurvey(s.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	err = mgr.CompleteSurvey(s.ID)
	if err != nil {
		t.Fatalf("CompleteSurvey() failed: %v", err)
	}

	result, _ := mgr.GetSurvey(s.ID)
	if result.Status != survey.StatusCompleted {
		t.Error("Survey not marked as completed")
	}
}

func TestSampleCount(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := survey.NewManager(tmpDir, nil, nil, nil)

	s, err := mgr.CreateSurvey("Test Survey", "Description", "wlan0", survey.TypePassive)
	if err != nil {
		t.Fatalf("CreateSurvey() failed: %v", err)
	}

	err = mgr.StartSurvey(s.ID)
	if err != nil {
		t.Fatalf("StartSurvey() failed: %v", err)
	}

	sampleData := map[string]any{
		"networks": []any{
			map[string]any{"ssid": "Test", "rssi": -60},
		},
	}

	// Add multiple samples.
	for i := range 5 {
		err = mgr.AddSample(s.ID, i*10, i*10, sampleData)
		if err != nil {
			t.Errorf("AddSample() failed: %v", err)
		}
	}

	result, err := mgr.GetSurvey(s.ID)
	if err != nil {
		t.Fatalf("GetSurvey() failed: %v", err)
	}

	// With multi-floor support, samples are on the active floor.
	samples := result.GetAllSamples()
	if len(samples) != 5 {
		t.Errorf("Sample count = %d, want 5", len(samples))
	}
}

func TestPassiveSampleAggregations(t *testing.T) {
	tests := []struct {
		name     string
		networks []*wifi.ScannedNetwork
		want     survey.PassiveSample
	}{
		{
			name:     "empty networks",
			networks: []*wifi.ScannedNetwork{},
			want: survey.PassiveSample{
				Networks:      []*wifi.ScannedNetwork{},
				UniqueSSIDs:   0,
				UniqueBSSIDs:  0,
				APCount2_4:    0,
				APCount5:      0,
				APCount6:      0,
				CoChannelAPs:  0,
				AdjChannelAPs: 0,
			},
		},
		{
			name:     "nil networks",
			networks: nil,
			want: survey.PassiveSample{
				Networks:      nil,
				UniqueSSIDs:   0,
				UniqueBSSIDs:  0,
				APCount2_4:    0,
				APCount5:      0,
				APCount6:      0,
				CoChannelAPs:  0,
				AdjChannelAPs: 0,
			},
		},
		{
			name: "single 2.4GHz network",
			networks: []*wifi.ScannedNetwork{
				{
					SSID:      "TestNet",
					BSSID:     "00:11:22:33:44:55",
					Channel:   6,
					Frequency: 2437,
					Signal:    -50,
				},
			},
			want: survey.PassiveSample{
				UniqueSSIDs:   1,
				UniqueBSSIDs:  1,
				APCount2_4:    1,
				APCount5:      0,
				APCount6:      0,
				CoChannelAPs:  1, // Same as strongest (itself).
				AdjChannelAPs: 0,
			},
		},
		{
			name: "multiple bands and channels",
			networks: []*wifi.ScannedNetwork{
				// Strongest AP on channel 36 (5GHz).
				{
					SSID:      "Net5G",
					BSSID:     "00:11:22:33:44:55",
					Channel:   36,
					Frequency: 5180,
					Signal:    -40,
				},
				// Co-channel AP.
				{
					SSID:      "Net5G-2",
					BSSID:     "00:11:22:33:44:66",
					Channel:   36,
					Frequency: 5180,
					Signal:    -50,
				},
				// Adjacent channel (+-1).
				{
					SSID:      "Net5G-3",
					BSSID:     "00:11:22:33:44:77",
					Channel:   37,
					Frequency: 5185,
					Signal:    -55,
				},
				// Adjacent channel (+-2).
				{
					SSID:      "Net5G-4",
					BSSID:     "00:11:22:33:44:88",
					Channel:   38,
					Frequency: 5190,
					Signal:    -60,
				},
				// 2.4GHz networks.
				{
					SSID:      "Net2.4",
					BSSID:     "AA:BB:CC:DD:EE:FF",
					Channel:   1,
					Frequency: 2412,
					Signal:    -65,
				},
				{
					SSID:      "Net2.4-2",
					BSSID:     "AA:BB:CC:DD:EE:AA",
					Channel:   6,
					Frequency: 2437,
					Signal:    -70,
				},
				// 6GHz network.
				{
					SSID:      "Net6G",
					BSSID:     "FF:EE:DD:CC:BB:AA",
					Channel:   1,
					Frequency: 5955,
					Signal:    -45,
				},
			},
			want: survey.PassiveSample{
				UniqueSSIDs:   7,
				UniqueBSSIDs:  7,
				APCount2_4:    2,
				APCount5:      4,
				APCount6:      1,
				CoChannelAPs:  2, // Two APs on channel 36.
				AdjChannelAPs: 2, // Channels 37 and 38.
			},
		},
		{
			name: "duplicate SSIDs different BSSIDs",
			networks: []*wifi.ScannedNetwork{
				{
					SSID:      "SameNet",
					BSSID:     "00:11:22:33:44:55",
					Channel:   1,
					Frequency: 2412,
					Signal:    -50,
				},
				{
					SSID:      "SameNet",
					BSSID:     "00:11:22:33:44:66",
					Channel:   1,
					Frequency: 2412,
					Signal:    -55,
				},
				{
					SSID:      "SameNet",
					BSSID:     "00:11:22:33:44:77",
					Channel:   1,
					Frequency: 2412,
					Signal:    -60,
				},
			},
			want: survey.PassiveSample{
				UniqueSSIDs:   1, // Only one unique SSID.
				UniqueBSSIDs:  3, // Three different APs.
				APCount2_4:    3,
				APCount5:      0,
				APCount6:      0,
				CoChannelAPs:  3, // All on channel 1.
				AdjChannelAPs: 0,
			},
		},
		{
			name: "hidden SSID handling",
			networks: []*wifi.ScannedNetwork{
				{SSID: "", BSSID: "00:11:22:33:44:55", Channel: 6, Frequency: 2437, Signal: -50},
				{
					SSID:      "VisibleNet",
					BSSID:     "00:11:22:33:44:66",
					Channel:   6,
					Frequency: 2437,
					Signal:    -55,
				},
			},
			want: survey.PassiveSample{
				UniqueSSIDs:   1, // Hidden SSID not counted.
				UniqueBSSIDs:  2, // Both BSSIDs counted.
				APCount2_4:    2,
				APCount5:      0,
				APCount6:      0,
				CoChannelAPs:  2,
				AdjChannelAPs: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample := survey.PassiveSample{
				Networks: tt.networks,
			}
			sample.CalculateAggregations()
			assertPassiveSampleAggregations(t, sample, tt.want)
		})
	}
}

// assertPassiveSampleAggregations compares the aggregation fields of a PassiveSample
// and reports any mismatches. This helper reduces cognitive complexity in the test.
func assertPassiveSampleAggregations(t *testing.T, got, want survey.PassiveSample) {
	t.Helper()

	if got.UniqueSSIDs != want.UniqueSSIDs {
		t.Errorf("UniqueSSIDs = %d, want %d", got.UniqueSSIDs, want.UniqueSSIDs)
	}
	if got.UniqueBSSIDs != want.UniqueBSSIDs {
		t.Errorf("UniqueBSSIDs = %d, want %d", got.UniqueBSSIDs, want.UniqueBSSIDs)
	}
	if got.APCount2_4 != want.APCount2_4 {
		t.Errorf("APCount2_4 = %d, want %d", got.APCount2_4, want.APCount2_4)
	}
	if got.APCount5 != want.APCount5 {
		t.Errorf("APCount5 = %d, want %d", got.APCount5, want.APCount5)
	}
	if got.APCount6 != want.APCount6 {
		t.Errorf("APCount6 = %d, want %d", got.APCount6, want.APCount6)
	}
	if got.CoChannelAPs != want.CoChannelAPs {
		t.Errorf("CoChannelAPs = %d, want %d", got.CoChannelAPs, want.CoChannelAPs)
	}
	if got.AdjChannelAPs != want.AdjChannelAPs {
		t.Errorf("AdjChannelAPs = %d, want %d", got.AdjChannelAPs, want.AdjChannelAPs)
	}
}
