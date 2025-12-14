// Package survey provides WiFi site survey functionality.
package survey

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/krisarmstrong/luminetiq/internal/iperf"
	"github.com/krisarmstrong/luminetiq/internal/wifi"
)

// SurveyType indicates the type of survey being conducted.
type SurveyType string

const (
	SurveyTypePassive    SurveyType = "passive"    // Passive scan (all visible networks)
	SurveyTypeActive     SurveyType = "active"     // Active monitoring (current connection)
	SurveyTypeThroughput SurveyType = "throughput" // Throughput testing with iperf3
)

// SurveyStatus indicates the current status of a survey.
type SurveyStatus string

const (
	StatusCreated    SurveyStatus = "created"
	StatusInProgress SurveyStatus = "in_progress"
	StatusPaused     SurveyStatus = "paused"
	StatusCompleted  SurveyStatus = "completed"
)

// FloorPlan contains floor plan image and metadata.
type FloorPlan struct {
	ImageData string `json:"imageData"` // Base64-encoded image
	Width     int    `json:"width"`     // Image width in pixels
	Height    int    `json:"height"`    // Image height in pixels
	ScaleM    float64 `json:"scaleM"`    // Meters per pixel
}

// PassiveSample contains data from a passive WiFi scan.
type PassiveSample struct {
	Networks []*wifi.ScannedNetwork `json:"networks"` // All visible APs
}

// ActiveSample contains data from active connection monitoring.
type ActiveSample struct {
	SSID         string  `json:"ssid"`
	BSSID        string  `json:"bssid"`
	RSSI         int     `json:"rssi"`         // Signal strength in dBm
	DataRate     float64 `json:"dataRate"`     // Mbps
	RoamingEvent bool    `json:"roamingEvent"` // true if BSSID changed since last sample
}

// ThroughputSample contains data from iperf3 throughput testing.
type ThroughputSample struct {
	SSID         string  `json:"ssid"`
	BSSID        string  `json:"bssid"`
	RSSI         int     `json:"rssi"`
	DownloadMbps float64 `json:"downloadMbps"`
	UploadMbps   float64 `json:"uploadMbps"`
	Latency      float64 `json:"latency"`      // milliseconds
	Jitter       float64 `json:"jitter"`       // milliseconds
	PacketLoss   float64 `json:"packetLoss"`   // percentage
}

// SamplePoint represents a measurement at a specific location.
type SamplePoint struct {
	X          int         `json:"x"`          // Pixel X coordinate on floor plan
	Y          int         `json:"y"`          // Pixel Y coordinate on floor plan
	Timestamp  time.Time   `json:"timestamp"`
	SampleData interface{} `json:"sampleData"` // PassiveSample | ActiveSample | ThroughputSample
}

// Survey represents a WiFi site survey.
type Survey struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	FloorPlan   *FloorPlan     `json:"floorPlan,omitempty"`
	SurveyType  SurveyType     `json:"surveyType"`
	Status      SurveyStatus   `json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	Samples     []*SamplePoint `json:"samples"`

	// Configuration
	Interface    string `json:"interface"`              // WiFi interface to use
	IperfServer  string `json:"iperfServer,omitempty"`  // For throughput surveys
	TestDuration int    `json:"testDuration,omitempty"` // seconds, for throughput tests
}

// Manager manages WiFi site surveys.
type Manager struct {
	mu            sync.RWMutex
	surveys       map[string]*Survey // key is survey ID
	storagePath   string
	wifiScanner   *wifi.Scanner
	wifiManager   *wifi.Manager
	iperfManager  *iperf.Manager
}

// NewManager creates a new survey manager.
func NewManager(storagePath string, wifiScanner *wifi.Scanner, wifiManager *wifi.Manager, iperfManager *iperf.Manager) *Manager {
	return &Manager{
		surveys:      make(map[string]*Survey),
		storagePath:  storagePath,
		wifiScanner:  wifiScanner,
		wifiManager:  wifiManager,
		iperfManager: iperfManager,
	}
}

// CreateSurvey creates a new survey.
func (m *Manager) CreateSurvey(name, description, iface string, surveyType SurveyType) (*Survey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey := &Survey{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		SurveyType:  surveyType,
		Status:      StatusCreated,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Interface:   iface,
		Samples:     make([]*SamplePoint, 0),
	}

	m.surveys[survey.ID] = survey

	// Persist to disk
	if err := m.saveSurvey(survey); err != nil {
		return nil, fmt.Errorf("failed to save survey: %w", err)
	}

	return survey, nil
}

// GetSurvey retrieves a survey by ID.
func (m *Manager) GetSurvey(id string) (*Survey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	survey, exists := m.surveys[id]
	if !exists {
		return nil, fmt.Errorf("survey not found: %s", id)
	}

	return survey, nil
}

// ListSurveys returns all surveys.
func (m *Manager) ListSurveys() []*Survey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	surveys := make([]*Survey, 0, len(m.surveys))
	for _, survey := range m.surveys {
		surveys = append(surveys, survey)
	}

	return surveys
}

// DeleteSurvey deletes a survey.
func (m *Manager) DeleteSurvey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.surveys[id]; !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	// Delete from disk
	filename := filepath.Join(m.storagePath, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete survey file: %w", err)
	}

	delete(m.surveys, id)

	return nil
}

// UpdateFloorPlan updates the floor plan for a survey.
func (m *Manager) UpdateFloorPlan(id string, floorPlan *FloorPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	survey.FloorPlan = floorPlan
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// StartSurvey starts a survey.
func (m *Manager) StartSurvey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	if survey.Status == StatusInProgress {
		return fmt.Errorf("survey already in progress")
	}

	survey.Status = StatusInProgress
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// PauseSurvey pauses a survey.
func (m *Manager) PauseSurvey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	survey.Status = StatusPaused
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// CompleteSurvey completes a survey.
func (m *Manager) CompleteSurvey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	survey.Status = StatusCompleted
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// AddSample adds a measurement sample to a survey.
func (m *Manager) AddSample(id string, x, y int, sampleData interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	if survey.Status != StatusInProgress {
		return fmt.Errorf("survey not in progress")
	}

	sample := &SamplePoint{
		X:          x,
		Y:          y,
		Timestamp:  time.Now(),
		SampleData: sampleData,
	}

	survey.Samples = append(survey.Samples, sample)
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// saveSurvey persists a survey to disk.
func (m *Manager) saveSurvey(survey *Survey) error {
	// Ensure storage directory exists
	if err := os.MkdirAll(m.storagePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	filename := filepath.Join(m.storagePath, fmt.Sprintf("%s.json", survey.ID))
	data, err := json.MarshalIndent(survey, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal survey: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write survey file: %w", err)
	}

	return nil
}

// LoadSurveys loads all surveys from disk.
func (m *Manager) LoadSurveys() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(m.storagePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Read all JSON files
	files, err := filepath.Glob(filepath.Join(m.storagePath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list survey files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var survey Survey
		if err := json.Unmarshal(data, &survey); err != nil {
			continue // Skip invalid JSON
		}

		m.surveys[survey.ID] = &survey
	}

	return nil
}
