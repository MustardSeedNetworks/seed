package config

// config_load.go contains the JSON Load/Save lifecycle: file reads, migration
// orchestration, backup-on-save, and first-boot credential bootstrap.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Load reads configuration from a JSON file.
// If the config has no version or an older version, it will be updated.
func Load(path string) (*Config, error) {
	if err := rejectRemovedTLSEnvironment(); err != nil {
		return nil, err
	}
	cfg := DefaultConfig()

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return nil, fmt.Errorf("read config file: %w", readErr)
		}
	} else {
		if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
			return nil, fmt.Errorf("parse config JSON: %w", unmarshalErr)
		}
	}

	// Handle unversioned configs (version 0 means unversioned)
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		logging.GetLogger().
			Info("Upgraded unversioned config to current version", "version", ConfigVersion)
	}
	if envErr := applyConfigServerEnv(cfg); envErr != nil {
		return nil, envErr
	}

	return cfg, nil
}

func readHTTPSPortEnvironment() (int, bool, error) {
	value, exists := os.LookupEnv("SEED_HTTPS_PORT")
	if !exists || value == "" {
		return 0, false, nil
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("SEED_HTTPS_PORT must be an integer: %w", err)
	}
	return port, true, nil
}

func applyConfigServerEnv(cfg *Config) error {
	port, exists, err := readHTTPSPortEnvironment()
	if err != nil {
		return err
	}
	if exists {
		cfg.Server.Port = port
	}
	if value := os.Getenv("SEED_PUBLIC_ORIGIN"); value != "" {
		cfg.Server.PublicOrigin = value
	}
	if value := os.Getenv("SEED_TLS_CERT_FILE"); value != "" {
		cfg.Server.CertFile = value
	}
	if value := os.Getenv("SEED_TLS_KEY_FILE"); value != "" {
		cfg.Server.KeyFile = value
	}
	return nil
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
