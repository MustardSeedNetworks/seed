package config

// config_load.go contains the JSON Load/Save lifecycle: file reads, migration
// orchestration, backup-on-save, and first-boot credential bootstrap.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// configFileMode is the permission the config is written with: owner-only,
// because it carries credentials.
const configFileMode = 0o600

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
	} else if decodeErr := decodeAndMigrate(data, cfg); decodeErr != nil {
		return nil, decodeErr
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

// decodeAndMigrate brings the document forward to the current schema version
// and decodes it.
//
// Deliberately in memory only. Writing back would give Load a side effect on
// the file it was asked to read, and it is not needed: the migration is
// idempotent and costs a map walk, and the first Save after this writes the
// new spelling anyway. So an install that never changes a setting keeps its
// file as the operator left it, and one that does self-heals.
func decodeAndMigrate(data []byte, cfg *Config) error {
	migrated, stripped, _, err := migrateJSON(data)
	if err != nil {
		return err
	}
	for _, removed := range stripped {
		// One line per key, naming where the setting went. Without this the
		// operator would find the setting quietly absent after an upgrade.
		logging.GetLogger().Warn("Dropped a setting this version no longer has",
			"key", removed.String(),
			"replacement", removed.replacement)
	}
	if unmarshalErr := json.Unmarshal(migrated, cfg); unmarshalErr != nil {
		return fmt.Errorf("parse config JSON: %w", unmarshalErr)
	}

	return nil
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
	return writeFileAtomically(path, data, configFileMode)
}

// writeFileAtomically replaces path with data through a scratch file in the
// same directory, so a crash or a full disk mid-save leaves the previous
// config exactly as it was (seed#2747). [os.WriteFile] truncates first, which
// both loses the old document before the new one is complete and lets a
// concurrent reader — Load, or an operator's editor — see an empty file.
//
// The cost is that the save now needs a writable directory rather than only a
// writable file. Every path Seed writes is one the writer owns: the packaged
// daemon owns /etc/seed (0750 seed:seed, listed in the unit's ReadWritePaths)
// and the CLI writes under the user's own XDG directory.
func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	// A symlinked config is replaced through its target, so the scratch file
	// lands on the same filesystem as the file rename will replace and the
	// operator's link survives — behaviour [os.WriteFile] had for free.
	target := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		target = resolved
	}

	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Both are no-ops once the rename has succeeded: the handle is closed
		// and tmpPath no longer names a file.
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, writeErr := tmp.Write(data); writeErr != nil {
		return fmt.Errorf("write config file: %w", writeErr)
	}
	if chmodErr := tmp.Chmod(mode); chmodErr != nil {
		return fmt.Errorf("set config file mode: %w", chmodErr)
	}
	// Without this the rename can be durable while the bytes it points at are
	// not, which is the crash this change exists to survive.
	if syncErr := tmp.Sync(); syncErr != nil {
		return fmt.Errorf("sync config file: %w", syncErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		return fmt.Errorf("close config file: %w", closeErr)
	}
	if renameErr := os.Rename(tmpPath, target); renameErr != nil {
		return fmt.Errorf("replace config file: %w", renameErr)
	}
	return syncDir(dir)
}

// syncDir makes the rename itself durable.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		// Windows cannot open a directory as a file to fsync it; NTFS orders
		// the rename's metadata itself.
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open config directory: %w", err)
	}
	defer func() { _ = d.Close() }()
	if syncErr := d.Sync(); syncErr != nil {
		return fmt.Errorf("sync config directory: %w", syncErr)
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
