package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// backupFileMode is the permission set for a written backup or restored
// config file: owner read/write only. mnd's ignored-functions list covers
// [os.WriteFile] but not the identically-named [os.Root] method used here,
// so this is named rather than repeated as a literal.
const backupFileMode = 0o600

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
	CreatedAt time.Time `json:"createdAt"`
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

	// Generate backup filename with timestamp (including nanoseconds for uniqueness).
	// filepath.Base on configPath strips any directory traversal attempts to
	// just the last element.
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000000")
	baseName := filepath.Base(m.configPath)
	backupName := fmt.Sprintf("%s.backup.%s", baseName, timestamp)
	backupPath := filepath.Join(m.backupDir, backupName)

	// backupName is written through an [os.Root] scoped to m.backupDir, so it
	// cannot resolve outside the backup directory even via a symlink; this
	// replaces the previous manual HasPrefix containment check.
	root, err := os.OpenRoot(m.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	if writeErr := root.WriteFile(backupName, data, backupFileMode); writeErr != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", writeErr)
	}

	// Get file info for metadata
	info, err := root.Stat(backupName)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	// Extract version from backup if possible
	version := m.ExtractVersion(data)

	backup := &BackupInfo{
		Name:      backupName,
		Path:      backupPath,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
		Version:   version,
	}

	// Prune old backups
	if pruneErr := m.PruneOldBackups(); pruneErr != nil {
		// Log but don't fail - the backup was created successfully
		logging.GetLogger().Warn("Failed to prune old backups", "error", pruneErr)
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
		matched, matchErr := filepath.Match(pattern, entry.Name())
		if matchErr != nil || !matched {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		backupPath := filepath.Join(m.backupDir, entry.Name())

		// Try to extract version from file
		version := 0

		if data, readErr := os.ReadFile(backupPath); readErr == nil {
			version = m.ExtractVersion(data)
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
//
// backupName arrives from the /api/config/restore request body: it is
// opened through an [os.Root] scoped to m.backupDir, so a name that steps
// outside the backup directory (e.g. via "../") is rejected instead of the
// previous HasPrefix check, which matched a sibling directory sharing
// m.backupDir's name as a prefix (e.g. "backupsXXX").
func (m *BackupManager) RestoreBackup(backupName string) error {
	backupRoot, err := os.OpenRoot(m.backupDir)
	if err != nil {
		return fmt.Errorf("failed to open backup directory: %w", err)
	}
	defer func() { _ = backupRoot.Close() }()

	data, err := backupRoot.ReadFile(backupName)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Validate the backup is valid JSON config
	cfg := DefaultConfig()
	if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
		return fmt.Errorf("backup file contains invalid configuration: %w", unmarshalErr)
	}

	// Create backup of current config before restoring
	if _, statErr := os.Stat(m.configPath); statErr == nil {
		if _, backupErr := m.CreateBackup(); backupErr != nil {
			return fmt.Errorf("failed to backup current config before restore: %w", backupErr)
		}
	}

	// Write restored config back to m.configPath, the exact file it is
	// configured against — scoped via fsutil.RootAt rather than a bare
	// os.WriteFile.
	cfgRoot, cfgName, err := fsutil.RootAt(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to open config directory: %w", err)
	}
	defer func() { _ = cfgRoot.Close() }()

	if writeErr := cfgRoot.WriteFile(cfgName, data, backupFileMode); writeErr != nil {
		return fmt.Errorf("failed to write restored config: %w", writeErr)
	}

	return nil
}

// DeleteBackup removes a backup file.
//
// backupName is opened through an [os.Root] scoped to m.backupDir for the
// same reason as RestoreBackup above.
func (m *BackupManager) DeleteBackup(backupName string) error {
	// Verify it's a backup file (not the main config)
	baseName := filepath.Base(m.configPath)
	if !strings.HasPrefix(backupName, baseName+".backup.") {
		return fmt.Errorf("invalid backup file: %s", backupName)
	}

	root, err := os.OpenRoot(m.backupDir)
	if err != nil {
		return fmt.Errorf("failed to open backup directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	if removeErr := root.Remove(backupName); removeErr != nil {
		return fmt.Errorf("failed to delete backup: %w", removeErr)
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
		if removeErr := os.Remove(backup.Path); removeErr != nil {
			// Continue trying to delete others
			logging.GetLogger().
				Warn("Failed to delete old backup", "name", backup.Name, "error", removeErr)
		}
	}

	return nil
}

// GetBackupDir returns the backup directory path.
func (m *BackupManager) GetBackupDir() string {
	return m.backupDir
}

// ExtractVersion attempts to extract the version number from config JSON data.
// Returns 0 if version cannot be determined.
func (m *BackupManager) ExtractVersion(data []byte) int {
	// Quick parse to extract just the version
	var partial struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return 0
	}
	return partial.Version
}
