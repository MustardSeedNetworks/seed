package survey

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// saveSurvey persists a survey to disk.
func (m *Manager) saveSurvey(survey *Survey) error {
	// Ensure storage directory exists with restrictive permissions
	if err := os.MkdirAll(m.storagePath, 0o750); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	filename := filepath.Join(m.storagePath, fmt.Sprintf("%s.json", survey.ID))
	data, err := json.MarshalIndent(survey, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal survey: %w", err)
	}

	if writeErr := os.WriteFile(filename, data, 0o600); writeErr != nil {
		return fmt.Errorf("failed to write survey file: %w", writeErr)
	}

	return nil
}

// LoadSurveys loads all surveys from disk and auto-migrates legacy single-floor surveys.
// Fixes #872: Added timeout protection for file operations (30 second limit).
func (m *Manager) LoadSurveys() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create storage directory if it doesn't exist with restrictive permissions
	if err := os.MkdirAll(m.storagePath, 0o750); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Read all JSON files
	files, err := filepath.Glob(filepath.Join(m.storagePath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list survey files: %w", err)
	}

	// Fixes #872: Track start time to prevent hanging on slow filesystem
	startTime := time.Now()
	const loadTimeout = 30 * time.Second

	for _, file := range files {
		// Check timeout to prevent hanging on slow/unresponsive filesystem (fixes #872)
		if time.Since(startTime) > loadTimeout {
			logging.GetLogger().
				Warn("Survey loading timeout reached, some surveys may not be loaded",
					"loaded_count", len(m.surveys),
					"elapsed", time.Since(startTime))
			break
		}

		// file comes from filepath.Glob with our controlled pattern, not user input
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			continue // Skip files that can't be read
		}

		var survey Survey
		if unmarshalErr := json.Unmarshal(data, &survey); unmarshalErr != nil {
			continue // Skip invalid JSON
		}

		// Auto-migrate legacy single-floor surveys to multi-floor format
		if MigrateToMultiFloor(&survey) {
			// Save the migrated survey back to disk
			if saveErr := m.saveSurveyUnlocked(&survey); saveErr != nil {
				logging.GetLogger().
					Warn("Failed to save migrated survey", "survey_id", survey.ID, "error", saveErr)
			}
		}

		m.surveys[survey.ID] = &survey
	}

	return nil
}

// saveSurveyUnlocked persists a survey to disk without acquiring a lock.
// This is used internally when the lock is already held.
func (m *Manager) saveSurveyUnlocked(survey *Survey) error {
	// Ensure storage directory exists with restrictive permissions
	if err := os.MkdirAll(m.storagePath, 0o750); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	filename := filepath.Join(m.storagePath, fmt.Sprintf("%s.json", survey.ID))
	data, err := json.MarshalIndent(survey, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal survey: %w", err)
	}

	if writeErr := os.WriteFile(filename, data, 0o600); writeErr != nil {
		return fmt.Errorf("failed to write survey file: %w", writeErr)
	}

	return nil
}
