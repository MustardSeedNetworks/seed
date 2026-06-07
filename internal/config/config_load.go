package config

// config_load.go contains the JSON Load/Save lifecycle: file reads, migration
// orchestration, backup-on-save, and first-boot credential bootstrap.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Load reads configuration from a JSON file.
// If the config has no version or an older version, it will be updated.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
		return nil, fmt.Errorf("parse config JSON: %w", unmarshalErr)
	}

	// Handle unversioned configs (version 0 means unversioned)
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		logging.GetLogger().
			Info("Upgraded unversioned config to current version", "version", ConfigVersion)
	}

	return cfg, nil
}

// LoadWithMigration reads configuration from a JSON file and applies any necessary migrations.
// It creates a backup before applying migrations.
func LoadWithMigration(path string, migrator *MigrationManager) (*Config, bool, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil // Use defaults if file doesn't exist
		}
		return nil, false, fmt.Errorf("read config file: %w", err)
	}

	// Check current version in file
	var partial struct {
		Version int `json:"version"`
	}
	if partialErr := json.Unmarshal(data, &partial); partialErr != nil {
		return nil, false, fmt.Errorf("failed to parse config version: %w", partialErr)
	}

	migrated := false
	if partial.Version < ConfigVersion && migrator != nil {
		// Create backup before migration
		backupMgr := NewBackupManager(path, "", defaultBackupMaxCount)
		if _, backupErr := backupMgr.CreateBackup(); backupErr != nil {
			logging.GetLogger().Warn("Failed to create backup before migration", "error", backupErr)
		}

		// Apply migrations
		migratedData, migrateErr := migrator.Migrate(data, partial.Version, ConfigVersion)
		if migrateErr != nil {
			return nil, false, fmt.Errorf("failed to migrate config from v%d to v%d: %w",
				partial.Version, ConfigVersion, migrateErr)
		}
		data = migratedData
		migrated = true
		logging.GetLogger().
			Info("Migrated config", "from_version", partial.Version, "to_version", ConfigVersion)
	}

	if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
		return nil, false, fmt.Errorf("parse config JSON: %w", unmarshalErr)
	}

	// Ensure version is set
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		migrated = true
	}

	// Save migrated config
	if migrated {
		if saveErr := cfg.Save(path); saveErr != nil {
			logging.GetLogger().Warn("Failed to save migrated config", "error", saveErr)
		}
	}

	return cfg, migrated, nil
}

// Save writes the configuration to a JSON file at the specified path.
// This method acquires a read lock to prevent data races during marshaling.
func (c *Config) Save(path string) error {
	c.mu.RLock()
	data, err := json.MarshalIndent(c, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal config JSON: %w", err)
	}
	if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
		return fmt.Errorf("write config file: %w", writeErr)
	}
	return nil
}

// SaveWithBackup writes the configuration to a JSON file, creating a backup first.
// This method acquires a read lock to prevent data races during marshaling.
// Returns the backup info if a backup was created, or nil if the file didn't exist.
func (c *Config) SaveWithBackup(path, backupDir string, maxBackups int) (*BackupInfo, error) {
	// Create backup if file exists
	var backup *BackupInfo
	if _, err := os.Stat(path); err == nil {
		backupMgr := NewBackupManager(path, backupDir, maxBackups)
		backup, err = backupMgr.CreateBackup()
		if err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Save the config
	if err := c.Save(path); err != nil {
		return backup, err
	}

	return backup, nil
}

// EnsureConfig handles first-boot setup and credential security.
// It checks for insecure default credentials and generates secure ones if needed.
// Returns SetupResult with credentials to display if they were generated.
//
// The function will:
// 1. Create config directory if it doesn't exist.
// 2. Load existing config or create default.
// 3. Check if using insecure default credentials (admin/seed).
// 4. Generate and persist secure credentials if needed.
// 5. Ensure JWT secret is persisted.
func EnsureConfig(
	path string,
	checkDefaultPassword func(hash string) bool,
) (*Config, *SetupResult, error) {
	result := &SetupResult{}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Check if config file exists
	_, err := os.Stat(path)
	isFirstBoot := os.IsNotExist(err)
	result.IsFirstBoot = isFirstBoot

	// Load or create config
	cfg, err := Load(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	needsSave := false

	// Check for insecure or missing credentials
	// Empty password hash = first boot, needs credential generation
	// Default password hash = insecure, needs credential generation
	if cfg.Auth.DefaultPasswordHash == "" ||
		(checkDefaultPassword != nil && checkDefaultPassword(cfg.Auth.DefaultPasswordHash)) {
		// Generate new secure credentials
		result.GeneratedCreds = true
		result.Username = cfg.Auth.DefaultUsername

		// Return error to signal caller needs to generate credentials
		return cfg, result, ErrInsecureCredentials
	}

	// Ensure JWT secret is set and persisted
	if cfg.Auth.JWTSecret == "" {
		needsSave = true
		result.JWTSecretStored = true
	}

	if needsSave && !isFirstBoot {
		if saveErr := cfg.Save(path); saveErr != nil {
			return nil, nil, fmt.Errorf("failed to save config: %w", saveErr)
		}
	}

	return cfg, result, nil
}
