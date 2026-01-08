package survey_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestDefaultReportOptions(t *testing.T) {
	opts := survey.DefaultReportOptions()

	if !opts.IncludeHeatmaps {
		t.Error("Expected IncludeHeatmaps to be true")
	}
	if opts.IncludeRawData {
		t.Error("Expected IncludeRawData to be false")
	}
	if !opts.IncludeRecommendations {
		t.Error("Expected IncludeRecommendations to be true")
	}
	if !opts.IncludeExecutiveSummary {
		t.Error("Expected IncludeExecutiveSummary to be true")
	}
	if opts.CompanyName != "" {
		t.Error("Expected CompanyName to be empty")
	}
	if opts.CompanyLogo != nil {
		t.Error("Expected CompanyLogo to be nil")
	}
}

func TestNewReportGenerator(t *testing.T) {
	s := &survey.Survey{
		ID:   "test-survey",
		Name: "Test Survey",
	}
	opts := survey.DefaultReportOptions()

	gen := survey.NewReportGenerator(s, opts)
	if gen == nil {
		t.Error("Expected non-nil ReportGenerator")
	}
}

func TestReportGenerator_Generate_NilSurvey(t *testing.T) {
	opts := survey.DefaultReportOptions()
	gen := survey.NewReportGenerator(nil, opts)

	_, err := gen.Generate()
	if err == nil {
		t.Error("Expected error for nil survey")
	}
	if err.Error() != "survey is nil" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestReportGenerator_Generate_EmptySurvey(t *testing.T) {
	now := time.Now()
	s := &survey.Survey{
		ID:          "test-survey",
		Name:        "Test Survey",
		Description: "A test survey for report generation",
		Status:      survey.StatusCompleted,
		CreatedAt:   now,
		UpdatedAt:   now,
		Floors:      []*survey.Floor{},
	}
	opts := survey.ReportOptions{
		IncludeHeatmaps:         false,
		IncludeRawData:          false,
		IncludeRecommendations:  true,
		IncludeExecutiveSummary: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF data")
	}
	// Verify it starts with PDF header
	if len(data) >= 4 && string(data[:4]) != "%PDF" {
		t.Error("Expected PDF data to start with '%PDF'")
	}
}

func TestReportGenerator_Generate_WithCompanyName(t *testing.T) {
	now := time.Now()
	s := &survey.Survey{
		ID:        "test-survey",
		Name:      "Test Survey",
		Status:    survey.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
		Floors:    []*survey.Floor{},
	}
	opts := survey.ReportOptions{
		IncludeExecutiveSummary: true,
		CompanyName:             "Acme Corp",
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF data")
	}
}

func TestReportGenerator_Generate_WithSamples(t *testing.T) {
	now := time.Now()

	// Create floor with samples
	samples := []*survey.SamplePoint{
		{
			X:         100,
			Y:         100,
			Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{SSID: "TestNetwork", Signal: -50, SNR: 30, Channel: 6},
				},
			},
		},
		{
			X:         200,
			Y:         200,
			Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{SSID: "TestNetwork2", Signal: -70, SNR: 20, Channel: 11},
				},
			},
		},
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Main Floor",
		Level:   1,
		Samples: samples,
		FloorPlan: &survey.FloorPlan{
			Width:  800,
			Height: 600,
		},
	}

	s := &survey.Survey{
		ID:            "test-survey",
		Name:          "Test Survey",
		Description:   "Test description",
		Status:        survey.StatusCompleted,
		CreatedAt:     now,
		UpdatedAt:     now,
		Floors:        []*survey.Floor{floor},
		ActiveFloorID: floor.ID,
	}

	opts := survey.ReportOptions{
		IncludeHeatmaps:         true,
		IncludeRawData:          true,
		IncludeRecommendations:  true,
		IncludeExecutiveSummary: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF data")
	}
}

func TestReportGenerator_Generate_AllStatuses(t *testing.T) {
	statuses := []survey.Status{
		survey.StatusCreated,
		survey.StatusInProgress,
		survey.StatusPaused,
		survey.StatusCompleted,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			s := &survey.Survey{
				ID:        "test",
				Name:      "Test",
				Status:    status,
				CreatedAt: time.Now(),
				Floors:    []*survey.Floor{},
			}

			gen := survey.NewReportGenerator(s, survey.DefaultReportOptions())
			data, err := gen.Generate()
			if err != nil {
				t.Errorf("Failed for status %s: %v", status, err)
			}
			if len(data) == 0 {
				t.Error("Expected non-empty PDF")
			}
		})
	}
}

func TestReportGenerator_Generate_MultipleFloors(t *testing.T) {
	now := time.Now()

	floors := []*survey.Floor{
		{
			ID:    "floor-1",
			Name:  "Basement",
			Level: -1,
			Samples: []*survey.SamplePoint{
				{X: 10, Y: 10, Timestamp: now},
			},
		},
		{
			ID:    "floor-2",
			Name:  "Ground Floor",
			Level: 0,
			Samples: []*survey.SamplePoint{
				{X: 20, Y: 20, Timestamp: now},
			},
		},
		{
			ID:    "floor-3",
			Name:  "First Floor",
			Level: 1,
			Samples: []*survey.SamplePoint{
				{X: 30, Y: 30, Timestamp: now},
			},
		},
	}

	s := &survey.Survey{
		ID:        "multi-floor",
		Name:      "Multi-Floor Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    floors,
	}

	opts := survey.ReportOptions{
		IncludeExecutiveSummary: true,
		IncludeRecommendations:  true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_FloorWithScale(t *testing.T) {
	now := time.Now()

	floor := &survey.Floor{
		ID:    "floor-1",
		Name:  "Test Floor",
		Level: 1,
		FloorPlan: &survey.FloorPlan{
			Width:  1000,
			Height: 800,
			ScaleM: 0.05, // 5cm per pixel
		},
		Samples: []*survey.SamplePoint{
			{X: 100, Y: 100, Timestamp: now},
		},
	}

	s := &survey.Survey{
		ID:        "test",
		Name:      "Test",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeHeatmaps: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_VariousSignalLevels(t *testing.T) {
	now := time.Now()

	// Create samples with various signal levels to test all branches
	samples := []*survey.SamplePoint{
		// Excellent signal (> -50)
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -40, Channel: 1}},
			},
		},
		// Good signal (-50 to -65)
		{
			X: 20, Y: 20, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -55, Channel: 6}},
			},
		},
		// Fair signal (-65 to -75)
		{
			X: 30, Y: 30, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -70, Channel: 11}},
			},
		},
		// Poor signal (-75 to -85)
		{
			X: 40, Y: 40, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -80, Channel: 1}},
			},
		},
		// Dead zone (< -85)
		{
			X: 50, Y: 50, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -90, Channel: 6}},
			},
		},
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "signal-test",
		Name:      "Signal Test Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeExecutiveSummary: true,
		IncludeRecommendations:  true,
		IncludeRawData:          true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_LimitedSamples(t *testing.T) {
	now := time.Now()

	// Create fewer than minimum threshold samples
	samples := make([]*survey.SamplePoint, 5)
	for i := range 5 {
		samples[i] = &survey.SamplePoint{
			X: i * 10, Y: i * 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -60}},
			},
		}
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "limited",
		Name:      "Limited Samples Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeRecommendations: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_ManySamplesForTruncation(t *testing.T) {
	now := time.Now()

	// Create more than 50 samples to test truncation
	samples := make([]*survey.SamplePoint, 100)
	for i := range 100 {
		samples[i] = &survey.SamplePoint{
			X: i * 5, Y: i * 5, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{SSID: "TestNet", Signal: -60, Channel: 6},
				},
			},
		}
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "many-samples",
		Name:      "Many Samples Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeRawData: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_LongSSIDTruncation(t *testing.T) {
	now := time.Now()

	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{SSID: "ThisIsAVeryLongSSIDNameThatShouldBeTruncated", Signal: -60, Channel: 1},
				},
			},
		},
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "long-ssid",
		Name:      "Long SSID Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeRawData: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_NoRecommendations(t *testing.T) {
	now := time.Now()

	// Create samples with excellent coverage - should result in minimal recommendations
	samples := make([]*survey.SamplePoint, 30)
	for i := range 30 {
		samples[i] = &survey.SamplePoint{
			X: i * 10, Y: i * 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -40}}, // Excellent signal
			},
		}
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "excellent",
		Name:      "Excellent Coverage Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeRecommendations: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_CriticalCoverage(t *testing.T) {
	now := time.Now()

	// Create samples with critical coverage issues
	samples := make([]*survey.SamplePoint, 30)
	for i := range 30 {
		samples[i] = &survey.SamplePoint{
			X: i * 10, Y: i * 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{{Signal: -90}}, // Dead zone signal
			},
		}
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "critical",
		Name:      "Critical Coverage Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeRecommendations: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_MultipleChannels(t *testing.T) {
	now := time.Now()

	// Create samples with multiple channel usage
	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{Signal: -50, Channel: 1},
					{Signal: -60, Channel: 6},
					{Signal: -70, Channel: 11},
				},
			},
		},
		{
			X: 20, Y: 20, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{
					{Signal: -55, Channel: 1},
					{Signal: -65, Channel: 36},
					{Signal: -75, Channel: 44},
				},
			},
		},
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "channels",
		Name:      "Channel Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeExecutiveSummary: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestReportGenerator_Generate_EmptyNetworks(t *testing.T) {
	now := time.Now()

	// Create sample with empty networks
	samples := []*survey.SamplePoint{
		{
			X: 10, Y: 10, Timestamp: now,
			SampleData: &survey.PassiveSample{
				Networks: []*wifi.ScannedNetwork{}, // Empty
			},
		},
		{
			X: 20, Y: 20, Timestamp: now,
			SampleData: nil, // Nil sample data
		},
	}

	floor := &survey.Floor{
		ID:      "floor-1",
		Name:    "Test Floor",
		Level:   1,
		Samples: samples,
	}

	s := &survey.Survey{
		ID:        "empty-networks",
		Name:      "Empty Networks Survey",
		Status:    survey.StatusCompleted,
		CreatedAt: now,
		Floors:    []*survey.Floor{floor},
	}

	opts := survey.ReportOptions{
		IncludeRawData: true,
	}

	gen := survey.NewReportGenerator(s, opts)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF")
	}
}

func TestManager_GenerateReport(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	// Create a survey
	s, err := m.CreateSurvey("Test Survey", "Description", "en0", survey.TypePassive)
	if err != nil {
		t.Fatalf("Failed to create survey: %v", err)
	}

	// Generate report
	data, err := m.GenerateReport(s.ID, survey.DefaultReportOptions())
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty PDF data")
	}
}

func TestManager_GenerateReport_SurveyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	m := survey.NewManager(tempDir, nil, nil, nil)

	_, err := m.GenerateReport("non-existent", survey.DefaultReportOptions())
	if err == nil {
		t.Error("Expected error for non-existent survey")
	}
}
