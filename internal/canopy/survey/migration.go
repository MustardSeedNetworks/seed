// Package survey provides WiFi site survey functionality.
package survey

import (
	"time"

	"github.com/google/uuid"
)

// MigrateToMultiFloor migrates a legacy single-floor survey to multi-floor format.
// This function is idempotent - calling it multiple times on an already migrated
// survey will have no effect.
//
// Migration behavior:
//   - If the survey already has floors, no migration is performed
//   - If the survey has legacy FloorPlan or Samples, they are moved to a new "Floor 1"
//   - The legacy fields are cleared after migration
//   - A new ActiveFloorID is set to the migrated floor
func MigrateToMultiFloor(survey *Survey) bool {
	// Already migrated - has floors
	if len(survey.Floors) > 0 {
		return false
	}

	// Nothing to migrate
	if survey.FloorPlan == nil && len(survey.Samples) == 0 {
		// Still create a default floor for surveys without any data
		now := time.Now()
		floor := &Floor{
			ID:        uuid.New().String(),
			Name:      "Floor 1",
			Level:     1,
			Samples:   make([]*SamplePoint, 0),
			CreatedAt: survey.CreatedAt,
			UpdatedAt: now,
		}
		survey.Floors = []*Floor{floor}
		survey.ActiveFloorID = floor.ID
		return true
	}

	// Migrate legacy data to a new floor
	now := time.Now()
	floor := &Floor{
		ID:        uuid.New().String(),
		Name:      "Floor 1",
		Level:     1,
		FloorPlan: survey.FloorPlan,
		Samples:   survey.Samples,
		CreatedAt: survey.CreatedAt,
		UpdatedAt: now,
	}

	// Ensure samples array is not nil
	if floor.Samples == nil {
		floor.Samples = make([]*SamplePoint, 0)
	}

	// Add the migrated floor
	survey.Floors = []*Floor{floor}
	survey.ActiveFloorID = floor.ID

	// Clear legacy fields (they're now in the floor)
	survey.FloorPlan = nil
	survey.Samples = nil

	return true
}

// NeedsMigration checks if a survey needs to be migrated to multi-floor format.
func NeedsMigration(survey *Survey) bool {
	// Already has floors - no migration needed
	if len(survey.Floors) > 0 {
		return false
	}

	// Has legacy data that needs migration
	return survey.FloorPlan != nil || len(survey.Samples) > 0
}
