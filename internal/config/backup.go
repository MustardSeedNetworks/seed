// Package config handles application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BackupManager handles configuration file backups.
type BackupManager struct {
	configPath string
	backupDir  string
	maxBackups int
}

// BackupInfo contains metadata about a configuration backup.
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Version   int       `json:"version"`
}

// NewBackupManager creates a new backup manager.
// If backupDir is empty, backups are stored alongside the config file.
func NewBackupManager(configPath, backupDir string, maxBackups int) *BackupManager {
	if maxBackups <= 0 {
		maxBackups = 10 // Default to keeping 10 backups
	}
	if backupDir == "" {
		backupDir = filepath.Dir(configPath)
	}
	return &BackupManager{
		configPath: configPath,
		backupDir:  backupDir,
		maxBackups: maxBackups,
	}
}

// CreateBackup creates a backup of the current configuration file.
// Returns the backup info or an error if the backup fails.
func (m *BackupManager) CreateBackup() (*BackupInfo, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(m.backupDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Read current config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file does not exist: %s", m.configPath)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Generate backup filename with timestamp (including nanoseconds for uniqueness)
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000000")
	baseName := filepath.Base(m.configPath)
	backupName := fmt.Sprintf("%s.backup.%s", baseName, timestamp)
	backupPath := filepath.Join(m.backupDir, backupName)

	// Write backup file with restricted permissions
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	// Get file info for metadata
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	// Extract version from backup if possible
	version := m.extractVersion(data)

	backup := &BackupInfo{
		Name:      backupName,
		Path:      backupPath,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
		Version:   version,
	}

	// Prune old backups
	if err := m.PruneOldBackups(); err != nil {
		// Log but don't fail - the backup was created successfully
		fmt.Printf("Warning: failed to prune old backups: %v\n", err)
	}

	return backup, nil
}

// ListBackups returns a list of available configuration backups.
// Backups are sorted by creation time, newest first.
func (m *BackupManager) ListBackups() ([]BackupInfo, error) {
	// Get config filename pattern
	baseName := filepath.Base(m.configPath)
	pattern := fmt.Sprintf("%s.backup.*", baseName)

	// Read backup directory
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if filename matches backup pattern
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil || !matched {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backupPath := filepath.Join(m.backupDir, entry.Name())

		// Try to extract version from file
		version := 0
		//nolint:gosec // G304: backupPath is constructed from known-safe directory
		if data, err := os.ReadFile(backupPath); err == nil {
			version = m.extractVersion(data)
		}

		backups = append(backups, BackupInfo{
			Name:      entry.Name(),
			Path:      backupPath,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Version:   version,
		})
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// RestoreBackup restores configuration from a backup file.
// Creates a backup of the current config before restoring.
func (m *BackupManager) RestoreBackup(backupName string) error {
	// Construct full path and validate
	backupPath := filepath.Join(m.backupDir, backupName)

	// Security: ensure the backup is in the expected directory
	cleanPath := filepath.Clean(backupPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(m.backupDir)) {
		return fmt.Errorf("invalid backup path: must be within backup directory")
	}

	// Read backup file
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Validate the backup is valid YAML config
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("backup file contains invalid configuration: %w", err)
	}

	// Create backup of current config before restoring
	if _, err := os.Stat(m.configPath); err == nil {
		if _, err := m.CreateBackup(); err != nil {
			return fmt.Errorf("failed to backup current config before restore: %w", err)
		}
	}

	// Write restored config
	if err := os.WriteFile(m.configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write restored config: %w", err)
	}

	return nil
}

// DeleteBackup removes a backup file.
func (m *BackupManager) DeleteBackup(backupName string) error {
	// Construct full path and validate
	backupPath := filepath.Join(m.backupDir, backupName)

	// Security: ensure the backup is in the expected directory
	cleanPath := filepath.Clean(backupPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(m.backupDir)) {
		return fmt.Errorf("invalid backup path: must be within backup directory")
	}

	// Verify it's a backup file (not the main config)
	baseName := filepath.Base(m.configPath)
	if !strings.HasPrefix(backupName, baseName+".backup.") {
		return fmt.Errorf("invalid backup file: %s", backupName)
	}

	if err := os.Remove(cleanPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// PruneOldBackups removes backups exceeding the maximum retention count.
func (m *BackupManager) PruneOldBackups() error {
	backups, err := m.ListBackups()
	if err != nil {
		return err
	}

	// Keep only maxBackups, delete the rest (oldest first since list is sorted newest first)
	if len(backups) <= m.maxBackups {
		return nil
	}

	for _, backup := range backups[m.maxBackups:] {
		if err := os.Remove(backup.Path); err != nil {
			// Continue trying to delete others
			fmt.Printf("Warning: failed to delete old backup %s: %v\n", backup.Name, err)
		}
	}

	return nil
}

// GetBackupDir returns the backup directory path.
func (m *BackupManager) GetBackupDir() string {
	return m.backupDir
}

// extractVersion attempts to extract the version number from config YAML data.
// Returns 0 if version cannot be determined.
func (m *BackupManager) extractVersion(data []byte) int {
	// Quick parse to extract just the version
	var partial struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &partial); err != nil {
		return 0
	}
	return partial.Version
}
