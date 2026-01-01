package canopy

import (
	"context"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/iperf"
)

// DefaultInterface is the default network interface to use when none is configured.
const DefaultInterface = "wlan0"

// WiFiService handles WiFi scanning and connections.
type WiFiService struct {
	cfg       *config.Config
	manager   *wifi.Manager
	scanner   *wifi.Scanner
	available bool
}

// NewWiFiService creates a new WiFi service.
func NewWiFiService(cfg *config.Config) *WiFiService {
	iface, ok := cfg.GetActiveInterface()
	if !ok || iface == "" {
		iface = DefaultInterface
	}

	return &WiFiService{
		cfg:     cfg,
		manager: wifi.NewManager(iface),
		scanner: wifi.NewScanner(iface),
	}
}

// Init initializes the WiFi service.
func (s *WiFiService) Init() error {
	// Check if we can access WiFi functionality
	s.available = s.manager.IsWireless()
	return nil
}

// IsAvailable returns whether WiFi scanning is available.
func (s *WiFiService) IsAvailable() bool {
	return s.available
}

// Scan performs a WiFi network scan.
func (s *WiFiService) Scan(_ context.Context) (*ScanResult, error) {
	if s.scanner == nil {
		return nil, ErrNotInitialized
	}

	startTime := time.Now()

	// Perform the scan
	networks, err := s.scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("wifi scan: %w", err)
	}

	// Get active interface name
	iface, _ := s.cfg.GetActiveInterface()
	if iface == "" {
		iface = DefaultInterface
	}

	// Convert to canopy types
	result := &ScanResult{
		Interface:  iface,
		Networks:   make([]WiFiNetwork, 0, len(networks)),
		ScanTime:   time.Since(startTime),
		ScanTimeMs: float64(time.Since(startTime).Milliseconds()),
		ScannedAt:  time.Now(),
	}

	for _, n := range networks {
		result.Networks = append(result.Networks, convertScannedNetwork(n))
	}

	return result, nil
}

// Connect connects to a WiFi network.
func (s *WiFiService) Connect(_ context.Context, _, _ string) error {
	// WiFi connection is platform-specific and typically handled by the OS
	return ErrNotImplemented
}

// Scanner returns the underlying WiFi scanner for dependency injection.
func (s *WiFiService) Scanner() *wifi.Scanner {
	return s.scanner
}

// Manager returns the underlying WiFi manager for dependency injection.
func (s *WiFiService) Manager() *wifi.Manager {
	return s.manager
}

// GetStatus returns the current WiFi connection status.
func (s *WiFiService) GetStatus(_ context.Context) (*ConnectionStatus, error) {
	if s.manager == nil {
		return nil, ErrNotInitialized
	}

	info := s.manager.GetInfo()
	if info == nil {
		return &ConnectionStatus{Connected: false}, nil
	}

	return &ConnectionStatus{
		Connected: info.SSID != "",
		SSID:      info.SSID,
		BSSID:     info.BSSID,
		Channel:   info.Channel,
		Frequency: info.Frequency,
		Band:      frequencyToBand(info.Frequency),
		Signal:    info.Signal,
		Security:  SecurityType(info.Security),
	}, nil
}

// SurveyService manages WiFi site surveys.
type SurveyService struct {
	cfg     *config.Config
	db      *database.DB
	manager *survey.Manager
	cancel  context.CancelFunc
}

// DefaultSurveyStoragePath is the default path for survey data storage.
const DefaultSurveyStoragePath = "data/surveys"

// NewSurveyService creates a new survey service.
func NewSurveyService(
	cfg *config.Config,
	db *database.DB,
	wifiScanner *wifi.Scanner,
	wifiManager *wifi.Manager,
	iperfManager *iperf.Manager,
) *SurveyService {
	storagePath := DefaultSurveyStoragePath

	return &SurveyService{
		cfg:     cfg,
		db:      db,
		manager: survey.NewManager(storagePath, wifiScanner, wifiManager, iperfManager),
	}
}

// Stop halts any active survey operations.
func (s *SurveyService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// SurveyManager returns the underlying survey manager for direct access.
func (s *SurveyService) SurveyManager() *survey.Manager {
	return s.manager
}

// Create creates a new survey.
func (s *SurveyService) Create(_ context.Context, name, description string) (*Survey, error) {
	if s.manager == nil {
		return nil, ErrNotInitialized
	}

	iface, ok := s.cfg.GetActiveInterface()
	if !ok || iface == "" {
		iface = DefaultInterface
	}

	// Create using the underlying manager
	surveyObj, err := s.manager.CreateSurvey(name, description, iface, survey.TypePassive)
	if err != nil {
		return nil, fmt.Errorf("creating survey: %w", err)
	}

	return convertSurvey(surveyObj), nil
}

// Get retrieves a survey by ID.
func (s *SurveyService) Get(_ context.Context, id string) (*Survey, error) {
	if s.manager == nil {
		return nil, ErrNotInitialized
	}

	surveyObj, err := s.manager.GetSurvey(id)
	if err != nil {
		return nil, fmt.Errorf("getting survey: %w", err)
	}

	return convertSurvey(surveyObj), nil
}

// List returns all surveys.
func (s *SurveyService) List(_ context.Context) ([]Survey, error) {
	if s.manager == nil {
		return nil, ErrNotInitialized
	}

	err := s.manager.LoadSurveys()
	if err != nil {
		return nil, fmt.Errorf("loading surveys: %w", err)
	}

	surveyList := s.manager.ListSurveys()
	result := make([]Survey, 0, len(surveyList))
	for _, surveyObj := range surveyList {
		result = append(result, *convertSurvey(surveyObj))
	}

	return result, nil
}

// AddPoint adds a measurement point to a survey.
func (s *SurveyService) AddPoint(_ context.Context, surveyID string, point *SurveyPoint) error {
	if s.manager == nil {
		return ErrNotInitialized
	}

	// Convert networks to wifi.ScannedNetwork
	networks := make([]*wifi.ScannedNetwork, 0, len(point.Networks))
	for _, n := range point.Networks {
		var security string
		if len(n.Security) > 0 {
			security = string(n.Security[0])
		}
		networks = append(networks, &wifi.ScannedNetwork{
			SSID:      n.SSID,
			BSSID:     n.BSSID,
			Signal:    n.SignalStrength,
			Channel:   n.Channel,
			Frequency: n.Frequency,
			Security:  security,
		})
	}

	// Create sample data
	sampleData := &survey.PassiveSample{
		Networks: networks,
	}

	// Use the public AddSample method which handles saving internally
	if err := s.manager.AddSample(surveyID, int(point.X), int(point.Y), sampleData); err != nil {
		return fmt.Errorf("adding sample: %w", err)
	}

	return nil
}

// ChannelService provides channel analysis.
type ChannelService struct {
	cfg     *config.Config
	scanner *wifi.Scanner
}

// NewChannelService creates a new channel service.
func NewChannelService(cfg *config.Config, scanner *wifi.Scanner) *ChannelService {
	return &ChannelService{
		cfg:     cfg,
		scanner: scanner,
	}
}

// Analyze performs channel utilization analysis.
func (s *ChannelService) Analyze(_ context.Context, band WiFiBand) (*ChannelAnalysis, error) {
	if s.scanner == nil {
		return nil, ErrNotInitialized
	}

	// Get cached networks or perform a scan
	networks := s.scanner.GetCachedNetworks()
	if len(networks) == 0 {
		scanned, err := s.scanner.Scan()
		if err != nil {
			return nil, fmt.Errorf("scanning for channel analysis: %w", err)
		}
		networks = scanned
	}

	// Group networks by channel
	channelCounts := make(map[int]int)
	for _, n := range networks {
		networkBand := frequencyToBand(n.Frequency)
		if networkBand == band {
			channelCounts[n.Channel]++
		}
	}

	// Build channel info
	channels := make([]ChannelInfo, 0)
	var recommendedChannel int
	minNetworks := 999

	for channel, count := range channelCounts {
		info := ChannelInfo{
			Number:        channel,
			CenterFreqMHz: channelToFrequency(channel),
			NetworkCount:  count,
			Utilization:   float64(count) * 10, // Simple utilization estimate
			IsDFS:         isDFSChannel(channel),
		}

		if count < minNetworks {
			minNetworks = count
			recommendedChannel = channel
			info.IsRecommended = true
		}

		channels = append(channels, info)
	}

	return &ChannelAnalysis{
		Band:               band,
		Channels:           channels,
		RecommendedChannel: recommendedChannel,
		AnalyzedAt:         time.Now(),
	}, nil
}

// AIService provides AI-assisted WiFi planning.
type AIService struct {
	cfg *config.Config
}

// NewAIService creates a new AI planning service.
func NewAIService(cfg *config.Config) *AIService {
	return &AIService{cfg: cfg}
}

// AnalyzeCoverage analyzes survey coverage and provides recommendations.
func (s *AIService) AnalyzeCoverage(_ context.Context, _ *Survey) (*CoverageAnalysis, error) {
	// AI coverage analysis requires ML models not yet implemented
	return nil, ErrNotImplemented
}

// SuggestAPPlacement suggests optimal AP placement.
func (s *AIService) SuggestAPPlacement(_ context.Context, _ *FloorPlan, _ map[string]any) ([]Recommendation, error) {
	// AI-based AP placement requires ML models not yet implemented
	return nil, ErrNotImplemented
}

// Helper functions

func convertScannedNetwork(n *wifi.ScannedNetwork) WiFiNetwork {
	return WiFiNetwork{
		SSID:           n.SSID,
		BSSID:          n.BSSID,
		Channel:        n.Channel,
		Frequency:      n.Frequency,
		Band:           frequencyToBand(n.Frequency),
		SignalStrength: n.Signal,
		NoiseFloor:     n.NoiseFloor,
		SNR:            n.SNR,
		Security:       []SecurityType{SecurityType(n.Security)},
		ChannelWidth:   n.ChannelWidth,
		LastSeen:       n.LastSeen,
	}
}

func convertSurvey(s *survey.Survey) *Survey {
	status := SurveyStatusDraft
	switch s.Status {
	case survey.StatusCreated:
		status = SurveyStatusDraft
	case survey.StatusInProgress:
		status = SurveyStatusInProgress
	case survey.StatusPaused:
		status = SurveyStatusInProgress // Paused surveys are still considered in progress
	case survey.StatusCompleted:
		status = SurveyStatusComplete
	}

	result := &Survey{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Status:      status,
		Points:      make([]SurveyPoint, 0),
	}

	// Convert floor plan if present
	if s.FloorPlan != nil {
		result.FloorPlan = &FloorPlan{
			ID:     s.ID,
			Name:   s.Name,
			Width:  float64(s.FloorPlan.Width) * s.FloorPlan.ScaleM,
			Height: float64(s.FloorPlan.Height) * s.FloorPlan.ScaleM,
			Scale:  1.0 / s.FloorPlan.ScaleM,
		}
	}

	// Convert samples from first floor if available
	if len(s.Floors) > 0 && len(s.Floors[0].Samples) > 0 {
		for _, sample := range s.Floors[0].Samples {
			point := SurveyPoint{
				X:          float64(sample.X),
				Y:          float64(sample.Y),
				MeasuredAt: sample.Timestamp,
				Networks:   make([]WiFiNetwork, 0),
			}

			// Convert sample data if it's a passive sample
			if passive, ok := sample.SampleData.(*survey.PassiveSample); ok {
				for _, n := range passive.Networks {
					point.Networks = append(point.Networks, convertScannedNetwork(n))
				}
			}

			result.Points = append(result.Points, point)
		}
	}

	return result
}

func frequencyToBand(freq int) WiFiBand {
	switch {
	case freq >= 2400 && freq < 2500:
		return Band2_4GHz
	case freq >= 5000 && freq < 5900:
		return Band5GHz
	case freq >= 5900:
		return Band6GHz
	default:
		return Band2_4GHz
	}
}

func channelToFrequency(channel int) int {
	// 2.4 GHz band
	if channel >= 1 && channel <= 13 {
		return 2407 + (channel * 5)
	}
	if channel == 14 {
		return 2484
	}

	// 5 GHz band
	if channel >= 36 && channel <= 165 {
		return 5000 + (channel * 5)
	}

	return 0
}

func isDFSChannel(channel int) bool {
	// DFS channels in 5 GHz band: 52-64, 100-144
	return (channel >= 52 && channel <= 64) || (channel >= 100 && channel <= 144)
}
