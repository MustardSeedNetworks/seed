package survey

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UpdateFloorPlan updates the floor plan for the active floor (or specified floor).
func (m *Manager) UpdateFloorPlan(id string, floorPlan *FloorPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[id]
	if !exists {
		return fmt.Errorf("survey not found: %s", id)
	}

	// Find the active floor
	floor := survey.GetActiveFloor()
	if floor == nil {
		return fmt.Errorf("no active floor set for survey: %s", id)
	}

	floor.FloorPlan = floorPlan
	floor.UpdatedAt = time.Now()
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// UpdateFloorPlanByFloorID updates the floor plan for a specific floor.
func (m *Manager) UpdateFloorPlanByFloorID(surveyID, floorID string, floorPlan *FloorPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return fmt.Errorf("survey not found: %s", surveyID)
	}

	floor := survey.GetFloorByID(floorID)
	if floor == nil {
		return fmt.Errorf("floor not found: %s", floorID)
	}

	floor.FloorPlan = floorPlan
	floor.UpdatedAt = time.Now()
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// AddFloor adds a new floor to a survey.
func (m *Manager) AddFloor(surveyID, name string, level int) (*Floor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return nil, fmt.Errorf("survey not found: %s", surveyID)
	}

	now := time.Now()
	floor := &Floor{
		ID:        uuid.New().String(),
		Name:      name,
		Level:     level,
		Samples:   make([]*SamplePoint, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}

	survey.Floors = append(survey.Floors, floor)
	survey.UpdatedAt = now

	if err := m.saveSurvey(survey); err != nil {
		return nil, err
	}

	return floor, nil
}

// UpdateFloor updates floor metadata (name, level).
func (m *Manager) UpdateFloor(surveyID, floorID, name string, level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return fmt.Errorf("survey not found: %s", surveyID)
	}

	floor := survey.GetFloorByID(floorID)
	if floor == nil {
		return fmt.Errorf("floor not found: %s", floorID)
	}

	floor.Name = name
	floor.Level = level
	floor.UpdatedAt = time.Now()
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// DeleteFloor removes a floor from a survey.
func (m *Manager) DeleteFloor(surveyID, floorID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return fmt.Errorf("survey not found: %s", surveyID)
	}

	// Don't allow deletion of the last floor
	if len(survey.Floors) <= 1 {
		return errors.New("cannot delete the last floor")
	}

	// Find and remove the floor
	for i, floor := range survey.Floors {
		if floor.ID == floorID {
			survey.Floors = append(survey.Floors[:i], survey.Floors[i+1:]...)

			// If we deleted the active floor, switch to the first remaining floor
			if survey.ActiveFloorID == floorID {
				survey.ActiveFloorID = survey.Floors[0].ID
			}

			survey.UpdatedAt = time.Now()
			return m.saveSurvey(survey)
		}
	}

	return fmt.Errorf("floor not found: %s", floorID)
}

// SetActiveFloor sets the active floor for data collection.
func (m *Manager) SetActiveFloor(surveyID, floorID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return fmt.Errorf("survey not found: %s", surveyID)
	}

	// Verify floor exists
	floor := survey.GetFloorByID(floorID)
	if floor == nil {
		return fmt.Errorf("floor not found: %s", floorID)
	}

	survey.ActiveFloorID = floorID
	survey.UpdatedAt = time.Now()

	return m.saveSurvey(survey)
}

// GetFloors returns all floors for a survey.
func (m *Manager) GetFloors(surveyID string) ([]*Floor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return nil, fmt.Errorf("survey not found: %s", surveyID)
	}

	return survey.Floors, nil
}

// GetFloor returns a specific floor.
func (m *Manager) GetFloor(surveyID, floorID string) (*Floor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	survey, exists := m.surveys[surveyID]
	if !exists {
		return nil, fmt.Errorf("survey not found: %s", surveyID)
	}

	floor := survey.GetFloorByID(floorID)
	if floor == nil {
		return nil, fmt.Errorf("floor not found: %s", floorID)
	}

	return floor, nil
}
