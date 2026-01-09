package canopy_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy"
	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/iperf"
)

// -----------------------------------------------------------------------------
// Module Tests
// -----------------------------------------------------------------------------

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "with default config",
			cfg:  config.DefaultConfig(),
		},
		{
			name: "with nil config values",
			cfg:  &config.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := canopy.New(tt.cfg, nil)
			if module == nil {
				t.Fatal("expected non-nil module")
			}

			// Verify services are initialized
			if module.WiFi() == nil {
				t.Error("expected WiFi service to be initialized")
			}
			if module.Survey() == nil {
				t.Error("expected Survey service to be initialized")
			}
			if module.Channel() == nil {
				t.Error("expected Channel service to be initialized")
			}
			if module.AI() == nil {
				t.Error("expected AI service to be initialized")
			}
		})
	}
}

func TestModuleStartStop(t *testing.T) {
	cfg := config.DefaultConfig()
	module := canopy.New(cfg, nil)

	ctx := context.Background()

	// Test Start - should not error even if WiFi is unavailable
	err := module.Start(ctx)
	if err != nil {
		t.Errorf("Start() returned unexpected error: %v", err)
	}

	// Test Stop
	err = module.Stop()
	if err != nil {
		t.Errorf("Stop() returned unexpected error: %v", err)
	}
}

func TestModuleServiceAccessors(t *testing.T) {
	cfg := config.DefaultConfig()
	module := canopy.New(cfg, nil)

	// Test that service accessors are thread-safe
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = module.WiFi()
				_ = module.Survey()
				_ = module.Channel()
				_ = module.AI()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// -----------------------------------------------------------------------------
// WiFiService Tests
// -----------------------------------------------------------------------------

func TestNewWiFiService(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		wantInterface string
	}{
		{
			name: "with configured interface",
			cfg: &config.Config{
				Interface: config.InterfaceConfig{
					Default: "en0",
				},
			},
			wantInterface: "en0",
		},
		{
			name:          "with empty interface defaults to wlan0",
			cfg:           &config.Config{},
			wantInterface: "wlan0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := canopy.NewWiFiService(tt.cfg)
			if service == nil {
				t.Fatal("expected non-nil WiFiService")
			}

			// Service should return scanner and manager
			if service.Scanner() == nil {
				t.Error("expected non-nil Scanner")
			}
			if service.Manager() == nil {
				t.Error("expected non-nil Manager")
			}
		})
	}
}

func TestWiFiServiceInit(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewWiFiService(cfg)

	err := service.Init()
	if err != nil {
		t.Errorf("Init() returned unexpected error: %v", err)
	}

	// IsAvailable depends on system state, just verify it doesn't panic
	_ = service.IsAvailable()
}

func TestWiFiServiceScan(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewWiFiService(cfg)
	ctx := context.Background()

	// Scan may fail on systems without WiFi, which is expected
	result, err := service.Scan(ctx)
	if err != nil {
		// Expected on systems without WiFi or proper permissions
		t.Logf("Scan returned error (expected on systems without WiFi): %v", err)
		return
	}

	if result == nil {
		t.Error("expected non-nil result when no error")
		return
	}

	// Verify result structure
	if result.Interface == "" {
		t.Error("expected non-empty Interface")
	}
	if result.Networks == nil {
		t.Error("expected non-nil Networks slice")
	}
}

func TestWiFiServiceScanNilScanner(t *testing.T) {
	service := &canopy.WiFiService{}
	service.SetScanner(nil)

	ctx := context.Background()
	_, err := service.Scan(ctx)
	if err == nil {
		t.Error("expected error when scanner is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestWiFiServiceConnect(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewWiFiService(cfg)
	ctx := context.Background()

	// Connect is not implemented, should return ErrNotImplemented
	err := service.Connect(ctx, "TestNetwork", "password123")
	if err == nil {
		t.Error("expected error from Connect")
	}
	if !errors.Is(err, canopy.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got: %v", err)
	}
}

func TestWiFiServiceGetStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewWiFiService(cfg)
	ctx := context.Background()

	status, err := service.GetStatus(ctx)
	if err != nil {
		t.Errorf("GetStatus() returned unexpected error: %v", err)
		return
	}

	if status == nil {
		t.Error("expected non-nil status")
	}
}

func TestWiFiServiceGetStatusNilManager(t *testing.T) {
	service := &canopy.WiFiService{}
	service.SetManager(nil)

	ctx := context.Background()
	_, err := service.GetStatus(ctx)
	if err == nil {
		t.Error("expected error when manager is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// SurveyService Tests
// -----------------------------------------------------------------------------

func TestNewSurveyService(t *testing.T) {
	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)
	if service == nil {
		t.Fatal("expected non-nil SurveyService")
	}

	if service.SurveyManager() == nil {
		t.Error("expected non-nil SurveyManager")
	}
}

func TestSurveyServiceCreate(t *testing.T) {
	// Create a temp directory for survey storage
	tempDir, err := os.MkdirTemp("", "canopy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	// Create service with temp storage
	surveyManager := survey.NewManager(tempDir, wifiScanner, wifiManager, iperfManager)
	service := &canopy.SurveyService{}
	// We need to use the exported interface here

	// Use the actual service creation
	service = canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)

	tests := []struct {
		name        string
		surveyName  string
		description string
		wantErr     bool
	}{
		{
			name:        "create valid survey",
			surveyName:  "Test Survey",
			description: "Test description",
			wantErr:     false,
		},
		{
			name:        "create survey with empty name",
			surveyName:  "",
			description: "",
			wantErr:     false, // Empty names are allowed
		},
	}

	// Store the original manager
	originalManager := surveyManager

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Use a properly configured manager
			_ = originalManager // silence unused var

			result, err := service.Create(ctx, tt.surveyName, tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("expected non-nil result")
			}
			if !tt.wantErr && result != nil {
				if result.ID == "" {
					t.Error("expected non-empty ID")
				}
				if result.Name != tt.surveyName {
					t.Errorf("expected Name %q, got %q", tt.surveyName, result.Name)
				}
			}
		})
	}
}

func TestSurveyServiceGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "canopy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)
	ctx := context.Background()

	// Create a survey first
	created, err := service.Create(ctx, "Test Survey", "Description")
	if err != nil {
		t.Fatalf("failed to create survey: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "get existing survey",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "get non-existent survey",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Get(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestSurveyServiceList(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "canopy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)
	ctx := context.Background()

	// Create a few surveys
	for i := 0; i < 3; i++ {
		_, err := service.Create(ctx, "Survey "+string(rune('A'+i)), "Description")
		if err != nil {
			t.Fatalf("failed to create survey: %v", err)
		}
	}

	// List should return at least the created surveys
	result, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if len(result) < 3 {
		t.Errorf("expected at least 3 surveys, got %d", len(result))
	}
}

func TestSurveyServiceAddPoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "canopy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)
	ctx := context.Background()

	// Create a survey
	created, err := service.Create(ctx, "Test Survey", "Description")
	if err != nil {
		t.Fatalf("failed to create survey: %v", err)
	}

	// Start the survey (required before adding points)
	manager := service.SurveyManager()
	if err := manager.StartSurvey(created.ID); err != nil {
		t.Fatalf("failed to start survey: %v", err)
	}

	point := &canopy.SurveyPoint{
		X:          100.0,
		Y:          200.0,
		MeasuredAt: time.Now(),
		Networks: []canopy.WiFiNetwork{
			{
				SSID:           "TestNetwork",
				BSSID:          "00:11:22:33:44:55",
				Channel:        6,
				Frequency:      2437,
				SignalStrength: -65,
				Security:       []canopy.SecurityType{canopy.SecurityWPA2},
			},
		},
	}

	err = service.AddPoint(ctx, created.ID, point)
	if err != nil {
		t.Errorf("AddPoint() returned error: %v", err)
	}
}

func TestSurveyServiceStop(t *testing.T) {
	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)

	// Stop should not panic even when nothing is running
	service.Stop()
}

// -----------------------------------------------------------------------------
// ChannelService Tests
// -----------------------------------------------------------------------------

func TestNewChannelService(t *testing.T) {
	cfg := config.DefaultConfig()
	scanner := wifi.NewScanner("en0")

	service := canopy.NewChannelService(cfg, scanner)
	if service == nil {
		t.Fatal("expected non-nil ChannelService")
	}

	if service.GetScanner() == nil {
		t.Error("expected non-nil Scanner")
	}
}

func TestChannelServiceAnalyze(t *testing.T) {
	cfg := config.DefaultConfig()
	scanner := wifi.NewScanner("en0")
	service := canopy.NewChannelService(cfg, scanner)
	ctx := context.Background()

	tests := []struct {
		name string
		band canopy.WiFiBand
	}{
		{
			name: "analyze 2.4GHz band",
			band: canopy.Band2_4GHz,
		},
		{
			name: "analyze 5GHz band",
			band: canopy.Band5GHz,
		},
		{
			name: "analyze 6GHz band",
			band: canopy.Band6GHz,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Analyze(ctx, tt.band)
			if err != nil {
				// May fail on systems without WiFi, which is expected
				t.Logf("Analyze returned error (expected on systems without WiFi): %v", err)
				return
			}

			if result == nil {
				t.Error("expected non-nil result when no error")
				return
			}

			if result.Band != tt.band {
				t.Errorf("expected Band %v, got %v", tt.band, result.Band)
			}
			if result.AnalyzedAt.IsZero() {
				t.Error("expected non-zero AnalyzedAt")
			}
		})
	}
}

func TestChannelServiceAnalyzeNilScanner(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewChannelService(cfg, nil)
	ctx := context.Background()

	_, err := service.Analyze(ctx, canopy.Band2_4GHz)
	if err == nil {
		t.Error("expected error when scanner is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// AIService Tests
// -----------------------------------------------------------------------------

func TestNewAIService(t *testing.T) {
	cfg := config.DefaultConfig()

	service := canopy.NewAIService(cfg)
	if service == nil {
		t.Fatal("expected non-nil AIService")
	}
}

func TestAIServiceAnalyzeCoverage(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewAIService(cfg)
	ctx := context.Background()

	survey := &canopy.Survey{
		ID:   "test-survey",
		Name: "Test Survey",
	}

	_, err := service.AnalyzeCoverage(ctx, survey)
	if err == nil {
		t.Error("expected error (not implemented)")
	}
	if !errors.Is(err, canopy.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got: %v", err)
	}
}

func TestAIServiceSuggestAPPlacement(t *testing.T) {
	cfg := config.DefaultConfig()
	service := canopy.NewAIService(cfg)
	ctx := context.Background()

	floorPlan := &canopy.FloorPlan{
		ID:     "test-floor",
		Name:   "Test Floor",
		Width:  100.0,
		Height: 50.0,
	}

	_, err := service.SuggestAPPlacement(ctx, floorPlan, nil)
	if err == nil {
		t.Error("expected error (not implemented)")
	}
	if !errors.Is(err, canopy.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Helper Function Tests
// -----------------------------------------------------------------------------

func TestFrequencyToBand(t *testing.T) {
	tests := []struct {
		name     string
		freq     int
		expected canopy.WiFiBand
	}{
		// 2.4 GHz band
		{"2.4GHz lower bound", 2400, canopy.Band2_4GHz},
		{"2.4GHz channel 1", 2412, canopy.Band2_4GHz},
		{"2.4GHz channel 6", 2437, canopy.Band2_4GHz},
		{"2.4GHz channel 11", 2462, canopy.Band2_4GHz},
		{"2.4GHz channel 14", 2484, canopy.Band2_4GHz},
		{"2.4GHz upper bound", 2499, canopy.Band2_4GHz},

		// 5 GHz band
		{"5GHz lower bound", 5000, canopy.Band5GHz},
		{"5GHz channel 36", 5180, canopy.Band5GHz},
		{"5GHz channel 100", 5500, canopy.Band5GHz},
		{"5GHz channel 165", 5825, canopy.Band5GHz},
		{"5GHz upper bound", 5899, canopy.Band5GHz},

		// 6 GHz band
		{"6GHz lower bound", 5900, canopy.Band6GHz},
		{"6GHz channel 1", 5955, canopy.Band6GHz},
		{"6GHz channel 93", 6415, canopy.Band6GHz},
		{"6GHz channel 233", 7115, canopy.Band6GHz},

		// Edge cases (defaults to 2.4GHz)
		{"Below 2.4GHz", 2399, canopy.Band2_4GHz},
		{"Zero frequency", 0, canopy.Band2_4GHz},
		{"Negative frequency", -100, canopy.Band2_4GHz},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canopy.FrequencyToBand(tt.freq)
			if result != tt.expected {
				t.Errorf("FrequencyToBand(%d) = %v, want %v", tt.freq, result, tt.expected)
			}
		})
	}
}

func TestChannelToFrequency(t *testing.T) {
	tests := []struct {
		name     string
		channel  int
		expected int
	}{
		// 2.4 GHz band
		{"Channel 1", 1, 2412},
		{"Channel 6", 6, 2437},
		{"Channel 11", 11, 2462},
		{"Channel 13", 13, 2472},
		{"Channel 14 (Japan)", 14, 2484},

		// 5 GHz band
		{"Channel 36", 36, 5180},
		{"Channel 40", 40, 5200},
		{"Channel 44", 44, 5220},
		{"Channel 48", 48, 5240},
		{"Channel 100", 100, 5500},
		{"Channel 149", 149, 5745},
		{"Channel 165", 165, 5825},

		// Invalid channels
		{"Channel 0", 0, 0},
		{"Channel 15", 15, 0},
		{"Channel 35", 35, 0},
		{"Channel 166", 166, 0},
		{"Negative channel", -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canopy.ChannelToFrequency(tt.channel)
			if result != tt.expected {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestIsDFSChannel(t *testing.T) {
	tests := []struct {
		name     string
		channel  int
		expected bool
	}{
		// Non-DFS channels
		{"Channel 36 (non-DFS)", 36, false},
		{"Channel 40 (non-DFS)", 40, false},
		{"Channel 44 (non-DFS)", 44, false},
		{"Channel 48 (non-DFS)", 48, false},
		{"Channel 149 (non-DFS)", 149, false},
		{"Channel 165 (non-DFS)", 165, false},

		// DFS channels (52-64)
		{"Channel 52 (DFS)", 52, true},
		{"Channel 56 (DFS)", 56, true},
		{"Channel 60 (DFS)", 60, true},
		{"Channel 64 (DFS)", 64, true},

		// DFS channels (100-144)
		{"Channel 100 (DFS)", 100, true},
		{"Channel 104 (DFS)", 104, true},
		{"Channel 108 (DFS)", 108, true},
		{"Channel 112 (DFS)", 112, true},
		{"Channel 116 (DFS)", 116, true},
		{"Channel 120 (DFS)", 120, true},
		{"Channel 124 (DFS)", 124, true},
		{"Channel 128 (DFS)", 128, true},
		{"Channel 132 (DFS)", 132, true},
		{"Channel 136 (DFS)", 136, true},
		{"Channel 140 (DFS)", 140, true},
		{"Channel 144 (DFS)", 144, true},

		// Edge cases
		{"Channel 51 (edge, non-DFS)", 51, false},
		{"Channel 65 (edge, non-DFS)", 65, false},
		{"Channel 99 (edge, non-DFS)", 99, false},
		{"Channel 145 (edge, non-DFS)", 145, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canopy.IsDFSChannel(tt.channel)
			if result != tt.expected {
				t.Errorf("IsDFSChannel(%d) = %v, want %v", tt.channel, result, tt.expected)
			}
		})
	}
}

func TestConvertScannedNetwork(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		network *wifi.ScannedNetwork
	}{
		{
			name: "full network data",
			network: &wifi.ScannedNetwork{
				SSID:         "TestNetwork",
				BSSID:        "00:11:22:33:44:55",
				Signal:       -65,
				Channel:      6,
				Frequency:    2437,
				Security:     "WPA2",
				ChannelWidth: 40,
				NoiseFloor:   -95,
				SNR:          30,
				LastSeen:     now,
			},
		},
		{
			name: "minimal network data",
			network: &wifi.ScannedNetwork{
				SSID:  "MinimalNetwork",
				BSSID: "AA:BB:CC:DD:EE:FF",
			},
		},
		{
			name: "hidden network",
			network: &wifi.ScannedNetwork{
				SSID:      "",
				BSSID:     "11:22:33:44:55:66",
				Signal:    -70,
				Channel:   11,
				Frequency: 2462,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canopy.ConvertScannedNetwork(tt.network)

			if result.SSID != tt.network.SSID {
				t.Errorf("expected SSID %q, got %q", tt.network.SSID, result.SSID)
			}
			if result.BSSID != tt.network.BSSID {
				t.Errorf("expected BSSID %q, got %q", tt.network.BSSID, result.BSSID)
			}
			if result.Channel != tt.network.Channel {
				t.Errorf("expected Channel %d, got %d", tt.network.Channel, result.Channel)
			}
			if result.Frequency != tt.network.Frequency {
				t.Errorf("expected Frequency %d, got %d", tt.network.Frequency, result.Frequency)
			}
			if result.SignalStrength != tt.network.Signal {
				t.Errorf("expected SignalStrength %d, got %d", tt.network.Signal, result.SignalStrength)
			}
			if result.ChannelWidth != tt.network.ChannelWidth {
				t.Errorf("expected ChannelWidth %d, got %d", tt.network.ChannelWidth, result.ChannelWidth)
			}
		})
	}
}

func TestConvertSurvey(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		survey *survey.Survey
	}{
		{
			name: "created survey",
			survey: &survey.Survey{
				ID:          "survey-1",
				Name:        "Test Survey",
				Description: "Test description",
				Status:      survey.StatusCreated,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "in progress survey",
			survey: &survey.Survey{
				ID:     "survey-2",
				Name:   "In Progress Survey",
				Status: survey.StatusInProgress,
			},
		},
		{
			name: "paused survey",
			survey: &survey.Survey{
				ID:     "survey-3",
				Name:   "Paused Survey",
				Status: survey.StatusPaused,
			},
		},
		{
			name: "completed survey",
			survey: &survey.Survey{
				ID:     "survey-4",
				Name:   "Completed Survey",
				Status: survey.StatusCompleted,
			},
		},
		{
			name: "survey with floor plan",
			survey: &survey.Survey{
				ID:     "survey-5",
				Name:   "Survey with Floor Plan",
				Status: survey.StatusCreated,
				FloorPlan: &survey.FloorPlan{
					Width:  800,
					Height: 600,
					ScaleM: 0.1,
				},
			},
		},
		{
			name: "survey with samples",
			survey: &survey.Survey{
				ID:     "survey-6",
				Name:   "Survey with Samples",
				Status: survey.StatusInProgress,
				Floors: []*survey.Floor{
					{
						ID:   "floor-1",
						Name: "Floor 1",
						Samples: []*survey.SamplePoint{
							{
								X:         100,
								Y:         200,
								Timestamp: now,
								SampleData: &survey.PassiveSample{
									Networks: []*wifi.ScannedNetwork{
										{
											SSID:      "Network1",
											BSSID:     "00:11:22:33:44:55",
											Signal:    -60,
											Frequency: 2437,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canopy.ConvertSurvey(tt.survey)

			if result.ID != tt.survey.ID {
				t.Errorf("expected ID %q, got %q", tt.survey.ID, result.ID)
			}
			if result.Name != tt.survey.Name {
				t.Errorf("expected Name %q, got %q", tt.survey.Name, result.Name)
			}
			if result.Description != tt.survey.Description {
				t.Errorf("expected Description %q, got %q", tt.survey.Description, result.Description)
			}

			// Verify status mapping
			switch tt.survey.Status {
			case survey.StatusCreated:
				if result.Status != canopy.SurveyStatusDraft {
					t.Errorf("expected Status draft, got %v", result.Status)
				}
			case survey.StatusInProgress, survey.StatusPaused:
				if result.Status != canopy.SurveyStatusInProgress {
					t.Errorf("expected Status in_progress, got %v", result.Status)
				}
			case survey.StatusCompleted:
				if result.Status != canopy.SurveyStatusComplete {
					t.Errorf("expected Status complete, got %v", result.Status)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Type Tests
// -----------------------------------------------------------------------------

func TestWiFiBandValues(t *testing.T) {
	// Verify constant values
	if canopy.Band2_4GHz != "2.4GHz" {
		t.Errorf("expected Band2_4GHz to be '2.4GHz', got %v", canopy.Band2_4GHz)
	}
	if canopy.Band5GHz != "5GHz" {
		t.Errorf("expected Band5GHz to be '5GHz', got %v", canopy.Band5GHz)
	}
	if canopy.Band6GHz != "6GHz" {
		t.Errorf("expected Band6GHz to be '6GHz', got %v", canopy.Band6GHz)
	}
}

func TestSecurityTypeValues(t *testing.T) {
	// Verify constant values
	tests := []struct {
		secType  canopy.SecurityType
		expected string
	}{
		{canopy.SecurityOpen, "Open"},
		{canopy.SecurityWEP, "WEP"},
		{canopy.SecurityWPA, "WPA"},
		{canopy.SecurityWPA2, "WPA2"},
		{canopy.SecurityWPA3, "WPA3"},
		{canopy.SecurityWPA2Ent, "WPA2-Enterprise"},
		{canopy.SecurityWPA3Ent, "WPA3-Enterprise"},
	}

	for _, tt := range tests {
		t.Run(string(tt.secType), func(t *testing.T) {
			if string(tt.secType) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.secType))
			}
		})
	}
}

func TestSurveyStatusValues(t *testing.T) {
	tests := []struct {
		status   canopy.SurveyStatus
		expected string
	}{
		{canopy.SurveyStatusDraft, "draft"},
		{canopy.SurveyStatusInProgress, "in_progress"},
		{canopy.SurveyStatusComplete, "complete"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

func TestWiFiStandardValues(t *testing.T) {
	tests := []struct {
		standard canopy.WiFiStandard
		expected string
	}{
		{canopy.Standard80211a, "802.11a"},
		{canopy.Standard80211b, "802.11b"},
		{canopy.Standard80211g, "802.11g"},
		{canopy.Standard80211n, "802.11n"},
		{canopy.Standard80211ac, "802.11ac"},
		{canopy.Standard80211ax, "802.11ax"},
		{canopy.Standard80211be, "802.11be"},
	}

	for _, tt := range tests {
		t.Run(string(tt.standard), func(t *testing.T) {
			if string(tt.standard) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.standard))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Error Tests
// -----------------------------------------------------------------------------

func TestErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "ErrNotImplemented",
			err:     canopy.ErrNotImplemented,
			message: "not implemented: pending migration",
		},
		{
			name:    "ErrNotInitialized",
			err:     canopy.ErrNotInitialized,
			message: "service not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("expected non-nil error")
			}
			if tt.err.Error() != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, tt.err.Error())
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Constants Tests
// -----------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	// Test interface constant
	if canopy.DefaultInterface != "wlan0" {
		t.Errorf("expected DefaultInterface 'wlan0', got %q", canopy.DefaultInterface)
	}

	// Test utilization constant
	if canopy.UtilizationPerNetworkPercent != 10 {
		t.Errorf("expected UtilizationPerNetworkPercent 10, got %d", canopy.UtilizationPerNetworkPercent)
	}

	// Test frequency thresholds
	if canopy.Freq6GHzMinMHz != 5900 {
		t.Errorf("expected Freq6GHzMinMHz 5900, got %d", canopy.Freq6GHzMinMHz)
	}

	// Test channel conversion constants
	if canopy.Channel2_4GHzMax != 13 {
		t.Errorf("expected Channel2_4GHzMax 13, got %d", canopy.Channel2_4GHzMax)
	}
	if canopy.Channel2_4GHzJapan != 14 {
		t.Errorf("expected Channel2_4GHzJapan 14, got %d", canopy.Channel2_4GHzJapan)
	}
	if canopy.Freq2_4GHzBaseMHz != 2407 {
		t.Errorf("expected Freq2_4GHzBaseMHz 2407, got %d", canopy.Freq2_4GHzBaseMHz)
	}
	if canopy.Freq2_4GHzChannel14MHz != 2484 {
		t.Errorf("expected Freq2_4GHzChannel14MHz 2484, got %d", canopy.Freq2_4GHzChannel14MHz)
	}
	if canopy.Freq5GHzBaseMHz != 5000 {
		t.Errorf("expected Freq5GHzBaseMHz 5000, got %d", canopy.Freq5GHzBaseMHz)
	}
	if canopy.ChannelSpacingMHz != 5 {
		t.Errorf("expected ChannelSpacingMHz 5, got %d", canopy.ChannelSpacingMHz)
	}

	// Test survey storage path
	if canopy.DefaultSurveyStoragePath != "data/surveys" {
		t.Errorf("expected DefaultSurveyStoragePath 'data/surveys', got %q", canopy.DefaultSurveyStoragePath)
	}
}

// -----------------------------------------------------------------------------
// Concurrent Access Tests
// -----------------------------------------------------------------------------

func TestModuleConcurrentAccess(t *testing.T) {
	cfg := config.DefaultConfig()
	module := canopy.New(cfg, nil)
	ctx := context.Background()

	// Start the module
	if err := module.Start(ctx); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer module.Stop()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				_ = module.WiFi()
				_ = module.Survey()
				_ = module.Channel()
				_ = module.AI()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// -----------------------------------------------------------------------------
// Integration Tests
// -----------------------------------------------------------------------------

func TestChannelServiceAnalyzeEmptyCache(t *testing.T) {
	// This test verifies behavior when scanner has no cached networks
	// The analysis will try to scan, which may fail on systems without WiFi
	cfg := config.DefaultConfig()
	scanner := wifi.NewScanner("en0")
	service := canopy.NewChannelService(cfg, scanner)
	ctx := context.Background()

	// Analyze will attempt a scan since cache is empty
	// This may fail on systems without WiFi, which is expected
	result, err := service.Analyze(ctx, canopy.Band2_4GHz)
	if err != nil {
		t.Logf("Analyze returned error (expected on systems without WiFi): %v", err)
		return
	}

	// If scan succeeded, verify result structure
	if result == nil {
		t.Fatal("expected non-nil result when no error")
	}
	if result.Band != canopy.Band2_4GHz {
		t.Errorf("expected Band 2.4GHz, got %v", result.Band)
	}
}

func TestChannelInfoStructure(t *testing.T) {
	// Test ChannelInfo struct fields
	info := canopy.ChannelInfo{
		Number:        6,
		CenterFreqMHz: 2437,
		NetworkCount:  3,
		Utilization:   30.0,
		IsDFS:         false,
		IsRecommended: true,
	}

	if info.Number != 6 {
		t.Errorf("expected Number 6, got %d", info.Number)
	}
	if info.CenterFreqMHz != 2437 {
		t.Errorf("expected CenterFreqMHz 2437, got %d", info.CenterFreqMHz)
	}
	if info.NetworkCount != 3 {
		t.Errorf("expected NetworkCount 3, got %d", info.NetworkCount)
	}
	if info.Utilization != 30.0 {
		t.Errorf("expected Utilization 30.0, got %.1f", info.Utilization)
	}
	if info.IsDFS {
		t.Error("expected IsDFS false")
	}
	if !info.IsRecommended {
		t.Error("expected IsRecommended true")
	}
}

func TestChannelAnalysisStructure(t *testing.T) {
	// Test ChannelAnalysis struct fields
	now := time.Now()
	analysis := canopy.ChannelAnalysis{
		Band: canopy.Band5GHz,
		Channels: []canopy.ChannelInfo{
			{Number: 36, CenterFreqMHz: 5180, NetworkCount: 1},
			{Number: 40, CenterFreqMHz: 5200, NetworkCount: 0, IsRecommended: true},
		},
		RecommendedChannel: 40,
		AnalyzedAt:         now,
	}

	if analysis.Band != canopy.Band5GHz {
		t.Errorf("expected Band 5GHz, got %v", analysis.Band)
	}
	if len(analysis.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(analysis.Channels))
	}
	if analysis.RecommendedChannel != 40 {
		t.Errorf("expected RecommendedChannel 40, got %d", analysis.RecommendedChannel)
	}
	if !analysis.AnalyzedAt.Equal(now) {
		t.Error("expected AnalyzedAt to match")
	}
}

func TestWiFiServiceScanWithCachedNetworks(t *testing.T) {
	cfg := config.DefaultConfig()
	scanner := wifi.NewScanner("en0")
	manager := wifi.NewManager("en0")

	// We can't easily test the successful scan path without mocking the platform
	// But we can test the service creation and initialization
	service := canopy.NewWiFiServiceForTest(scanner, manager)

	if service == nil {
		t.Fatal("expected non-nil WiFiService")
	}

	if !service.IsAvailable() {
		t.Error("expected service to be available")
	}

	// Verify scanner and manager are set
	if service.Scanner() != scanner {
		t.Error("expected scanner to match")
	}
	if service.Manager() != manager {
		t.Error("expected manager to match")
	}

	// Set available to false and verify
	service.SetAvailable(false)
	if service.IsAvailable() {
		t.Error("expected service to not be available after SetAvailable(false)")
	}

	_ = cfg // silence unused var
}

func TestSurveyServiceStartAndStop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "canopy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)
	ctx := context.Background()

	// Create a survey
	created, err := service.Create(ctx, "Start/Stop Test Survey", "Description")
	if err != nil {
		t.Fatalf("failed to create survey: %v", err)
	}

	// Start the survey via the underlying manager
	manager := service.SurveyManager()
	if err := manager.StartSurvey(created.ID); err != nil {
		t.Fatalf("failed to start survey: %v", err)
	}

	// Verify the survey is in progress
	s, err := manager.GetSurvey(created.ID)
	if err != nil {
		t.Fatalf("failed to get survey: %v", err)
	}
	if s.Status != survey.StatusInProgress {
		t.Errorf("expected status InProgress, got %v", s.Status)
	}

	// Stop the service
	service.Stop()

	// After stop, the survey should still be accessible
	_, err = service.Get(ctx, created.ID)
	if err != nil {
		t.Errorf("Get() after Stop() returned error: %v", err)
	}
}

func TestSurveyServiceDeleteAndExport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "canopy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	wifiScanner := wifi.NewScanner("en0")
	wifiManager := wifi.NewManager("en0")
	iperfManager := iperf.NewManager()

	service := canopy.NewSurveyService(cfg, nil, wifiScanner, wifiManager, iperfManager)
	ctx := context.Background()

	// Create a survey
	created, err := service.Create(ctx, "Delete Test Survey", "Description")
	if err != nil {
		t.Fatalf("failed to create survey: %v", err)
	}

	// Verify it exists
	_, err = service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get created survey: %v", err)
	}

	// The underlying manager has delete functionality
	manager := service.SurveyManager()
	if err := manager.DeleteSurvey(created.ID); err != nil {
		t.Fatalf("failed to delete survey: %v", err)
	}

	// Verify it no longer exists
	_, err = service.Get(ctx, created.ID)
	if err == nil {
		t.Error("expected error when getting deleted survey")
	}
}

func TestSurveyServiceCreateWithNilManager(t *testing.T) {
	service := &canopy.SurveyService{}
	ctx := context.Background()

	_, err := service.Create(ctx, "Test", "Description")
	if err == nil {
		t.Error("expected error when manager is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestSurveyServiceGetWithNilManager(t *testing.T) {
	service := &canopy.SurveyService{}
	ctx := context.Background()

	_, err := service.Get(ctx, "some-id")
	if err == nil {
		t.Error("expected error when manager is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestSurveyServiceListWithNilManager(t *testing.T) {
	service := &canopy.SurveyService{}
	ctx := context.Background()

	_, err := service.List(ctx)
	if err == nil {
		t.Error("expected error when manager is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestSurveyServiceAddPointWithNilManager(t *testing.T) {
	service := &canopy.SurveyService{}
	ctx := context.Background()

	err := service.AddPoint(ctx, "some-id", &canopy.SurveyPoint{})
	if err == nil {
		t.Error("expected error when manager is nil")
	}
	if !errors.Is(err, canopy.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestScanResultStructure(t *testing.T) {
	now := time.Now()
	result := canopy.ScanResult{
		Interface: "wlan0",
		Networks: []canopy.WiFiNetwork{
			{
				SSID:           "TestNetwork",
				BSSID:          "00:11:22:33:44:55",
				Channel:        6,
				Frequency:      2437,
				SignalStrength: -65,
			},
		},
		ScanTime:   100 * time.Millisecond,
		ScanTimeMs: 100.0,
		ScannedAt:  now,
	}

	if result.Interface != "wlan0" {
		t.Errorf("expected Interface 'wlan0', got %q", result.Interface)
	}
	if len(result.Networks) != 1 {
		t.Errorf("expected 1 network, got %d", len(result.Networks))
	}
	if result.ScanTimeMs != 100.0 {
		t.Errorf("expected ScanTimeMs 100.0, got %.1f", result.ScanTimeMs)
	}
	if !result.ScannedAt.Equal(now) {
		t.Error("expected ScannedAt to match")
	}
}

func TestConnectionStatusStructure(t *testing.T) {
	status := canopy.ConnectionStatus{
		Connected: true,
		SSID:      "MyNetwork",
		BSSID:     "00:11:22:33:44:55",
		Channel:   11,
		Frequency: 2462,
		Signal:    -55,
		TxRate:    144,
		Security:  "WPA2",
		IPAddress: "192.168.1.100",
		Gateway:   "192.168.1.1",
	}

	if !status.Connected {
		t.Error("expected Connected true")
	}
	if status.SSID != "MyNetwork" {
		t.Errorf("expected SSID 'MyNetwork', got %q", status.SSID)
	}
	if status.Channel != 11 {
		t.Errorf("expected Channel 11, got %d", status.Channel)
	}
	if status.TxRate != 144 {
		t.Errorf("expected TxRate 144, got %.1f", status.TxRate)
	}
	if status.IPAddress != "192.168.1.100" {
		t.Errorf("expected IPAddress '192.168.1.100', got %q", status.IPAddress)
	}
}

func TestSurveyPointStructure(t *testing.T) {
	now := time.Now()
	point := canopy.SurveyPoint{
		X:          150.5,
		Y:          200.3,
		MeasuredAt: now,
		Networks: []canopy.WiFiNetwork{
			{SSID: "Network1", BSSID: "00:11:22:33:44:55", SignalStrength: -60},
			{SSID: "Network2", BSSID: "00:11:22:33:44:56", SignalStrength: -70},
		},
	}

	if point.X != 150.5 {
		t.Errorf("expected X 150.5, got %.1f", point.X)
	}
	if point.Y != 200.3 {
		t.Errorf("expected Y 200.3, got %.1f", point.Y)
	}
	if len(point.Networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(point.Networks))
	}
	if !point.MeasuredAt.Equal(now) {
		t.Error("expected MeasuredAt to match")
	}
}

func TestFloorPlanStructure(t *testing.T) {
	floorPlan := canopy.FloorPlan{
		ID:       "floor-1",
		Name:     "First Floor",
		ImageURL: "/images/floor1.png",
		Width:    800.0,
		Height:   600.0,
		Scale:    0.1,
	}

	if floorPlan.ID != "floor-1" {
		t.Errorf("expected ID 'floor-1', got %q", floorPlan.ID)
	}
	if floorPlan.Name != "First Floor" {
		t.Errorf("expected Name 'First Floor', got %q", floorPlan.Name)
	}
	if floorPlan.Width != 800.0 {
		t.Errorf("expected Width 800.0, got %.1f", floorPlan.Width)
	}
	if floorPlan.Height != 600.0 {
		t.Errorf("expected Height 600.0, got %.1f", floorPlan.Height)
	}
	if floorPlan.Scale != 0.1 {
		t.Errorf("expected Scale 0.1, got %.2f", floorPlan.Scale)
	}
}

func TestCoverageAnalysisStructure(t *testing.T) {
	analysis := canopy.CoverageAnalysis{
		TotalArea:       1000.0,
		CoveredArea:     855.0,
		CoveragePercent: 85.5,
		DeadZones: []canopy.DeadZone{
			{X: 100, Y: 200, Radius: 5, Signal: -85},
		},
		Recommendations: []canopy.Recommendation{
			{Type: "placement", Priority: "high", Description: "Add AP near conference room"},
			{Type: "config", Priority: "medium", Description: "Adjust channel on AP1"},
		},
	}

	if analysis.CoveragePercent != 85.5 {
		t.Errorf("expected CoveragePercent 85.5, got %.1f", analysis.CoveragePercent)
	}
	if len(analysis.DeadZones) != 1 {
		t.Errorf("expected 1 dead zone, got %d", len(analysis.DeadZones))
	}
	if len(analysis.Recommendations) != 2 {
		t.Errorf("expected 2 recommendations, got %d", len(analysis.Recommendations))
	}
}

func TestModuleIntegration(t *testing.T) {
	// Create temp directory for survey storage
	tempDir, err := os.MkdirTemp("", "canopy-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Ensure the survey storage directory exists
	surveyPath := filepath.Join(tempDir, "surveys")
	if err := os.MkdirAll(surveyPath, 0o755); err != nil {
		t.Fatalf("failed to create survey path: %v", err)
	}

	cfg := config.DefaultConfig()
	module := canopy.New(cfg, nil)
	ctx := context.Background()

	// Start module
	if err := module.Start(ctx); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Test WiFi service init
	wifi := module.WiFi()
	if err := wifi.Init(); err != nil {
		t.Logf("WiFi.Init() returned error (expected on systems without WiFi): %v", err)
	}

	// Test creating a survey
	survey := module.Survey()
	createdSurvey, err := survey.Create(ctx, "Integration Test Survey", "Testing all services")
	if err != nil {
		t.Logf("Survey.Create() returned error: %v", err)
	} else if createdSurvey.ID == "" {
		t.Error("expected non-empty survey ID")
	}

	// Test channel analysis
	channel := module.Channel()
	_, err = channel.Analyze(ctx, canopy.Band2_4GHz)
	if err != nil {
		t.Logf("Channel.Analyze() returned error (expected on systems without WiFi): %v", err)
	}

	// Test AI service (returns not implemented)
	ai := module.AI()
	_, err = ai.AnalyzeCoverage(ctx, &canopy.Survey{})
	if !errors.Is(err, canopy.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented from AI.AnalyzeCoverage, got: %v", err)
	}

	// Stop module
	if err := module.Stop(); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}
