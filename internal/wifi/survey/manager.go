package survey

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// Manager manages WiFi site surveys.
type Manager struct {
	mu           sync.RWMutex
	surveys      map[string]*Survey // key is survey ID
	storagePath  string
	wifiScanner  *wifi.Scanner
	wifiManager  *wifi.Manager
	iperfManager *iperf.Manager
}

// NewManager creates a new survey manager.
func NewManager(
	storagePath string,
	wifiScanner *wifi.Scanner,
	wifiManager *wifi.Manager,
	iperfManager *iperf.Manager,
) *Manager {
	return &Manager{
		surveys:      make(map[string]*Survey),
		storagePath:  storagePath,
		wifiScanner:  wifiScanner,
		wifiManager:  wifiManager,
		iperfManager: iperfManager,
	}
}

// CreateSurvey creates a new survey with a default floor.
func (m *Manager) CreateSurvey(name, description, iface string, surveyType Type) (*Survey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Create default first floor
	defaultFloor := &Floor{
		ID:        uuid.New().String(),
		Name:      "Floor 1",
		Level:     1,
		Samples:   make([]*SamplePoint, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}

	survey := &Survey{
		ID:            uuid.New().String(),
		Name:          name,
		Description:   description,
		SurveyType:    surveyType,
		Status:        StatusCreated,
		CreatedAt:     now,
		UpdatedAt:     now,
		Interface:     iface,
		Floors:        []*Floor{defaultFloor},
		ActiveFloorID: defaultFloor.ID,
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

// StartSurvey starts a survey.
func (m *Manager) StartSurvey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	if survey.Status == StatusInProgress {
		return errors.New("survey already in progress")
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

// UpdateSurveySettings updates survey settings (only when survey is in created state).
func (m *Manager) UpdateSurveySettings(
	id string,
	surveyType Type,
	iperfServer string,
	testDuration int,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	if survey.Status != StatusCreated {
		return errors.New("cannot update settings after survey has started")
	}

	// Validate survey type
	switch surveyType {
	case TypePassive, TypeActive, TypeThroughput:
		survey.SurveyType = surveyType
	default:
		return fmt.Errorf("invalid survey type: %s", surveyType)
	}

	// Validate test duration
	if testDuration < 1 {
		testDuration = defaultTestDurationSec
	}
	if testDuration > maxTestDurationSec {
		testDuration = maxTestDurationSec
	}
	survey.TestDuration = testDuration
	survey.IperfServer = iperfServer
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// ImportedDataUpdate carries the slices a caller wants to replace on a survey.
// Each field is optional; nil means "leave as-is". An empty (non-nil) slice
// clears the corresponding survey field, which is the right behaviour when an
// importer wants to fully replace the existing list.
type ImportedDataUpdate struct {
	APLocations      []APLocation
	ClientLocations  []ClientLocation
	PassFailCriteria []PassFailCriterion
}

// UpdateImportedData replaces the survey's imported-data fields (#727).
// Callers (most notably the AirMapper import flow) use this to persist the
// AP/client placements and pass-fail criteria that come back from the parser
// so the data survives a page reload.
func (m *Manager) UpdateImportedData(id string, update ImportedDataUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	if update.APLocations != nil {
		survey.APLocations = update.APLocations
	}
	if update.ClientLocations != nil {
		survey.ClientLocations = update.ClientLocations
	}
	if update.PassFailCriteria != nil {
		survey.PassFailCriteria = update.PassFailCriteria
	}
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// AddSample adds a measurement sample to the active floor of a survey.
func (m *Manager) AddSample(id string, x, y int, sampleData any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	if survey.Status != StatusInProgress {
		return errors.New("survey not in progress")
	}

	floor := survey.GetActiveFloor()
	if floor == nil {
		return fmt.Errorf("no active floor set for survey: %s", id)
	}

	sample := &SamplePoint{
		X:          x,
		Y:          y,
		Timestamp:  time.Now(),
		SampleData: sampleData,
	}

	floor.Samples = append(floor.Samples, sample)
	floor.UpdatedAt = time.Now()
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// AddSampleToFloor adds a measurement sample to a specific floor.
func (m *Manager) AddSampleToFloor(surveyID, floorID string, x, y int, sampleData any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return fmt.Errorf("survey not found: %s", surveyID)
	}

	if survey.Status != StatusInProgress {
		return errors.New("survey not in progress")
	}

	floor := survey.GetFloorByID(floorID)
	if floor == nil {
		return fmt.Errorf("floor not found: %s", floorID)
	}

	sample := &SamplePoint{
		X:          x,
		Y:          y,
		Timestamp:  time.Now(),
		SampleData: sampleData,
	}

	floor.Samples = append(floor.Samples, sample)
	floor.UpdatedAt = time.Now()
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}
