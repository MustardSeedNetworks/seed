// Package config handles application configuration.
package config

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// Migration represents a single config schema migration.
type Migration struct {
	FromVersion int
	ToVersion   int
	Description string
	Migrate     func(data []byte) ([]byte, error)
}

// MigrationManager handles config schema migrations.
type MigrationManager struct {
	migrations []Migration
}

// NewMigrationManager creates a new migration manager with the default migrations.
func NewMigrationManager() *MigrationManager {
	return &MigrationManager{
		migrations: defaultMigrations,
	}
}

// RegisterMigration adds a migration to the manager.
func (m *MigrationManager) RegisterMigration(migration Migration) {
	m.migrations = append(m.migrations, migration)
	// Keep migrations sorted by FromVersion
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].FromVersion < m.migrations[j].FromVersion
	})
}

// Migrate applies all necessary migrations to transform config data from one version to another.
// Version 0 is treated as "unversioned" and is automatically allowed.
func (m *MigrationManager) Migrate(data []byte, fromVersion, toVersion int) ([]byte, error) {
	if fromVersion >= toVersion {
		return data, nil // No migration needed
	}

	// Version 0 means unversioned config - treat as version 1 for migration purposes
	effectiveFromVersion := fromVersion
	if fromVersion == 0 {
		effectiveFromVersion = 1
	}

	if effectiveFromVersion < MinSupportedVersion {
		return nil, fmt.Errorf("config version %d is too old; minimum supported version is %d",
			fromVersion, MinSupportedVersion)
	}

	// Get migration path
	path := m.GetMigrationPath(effectiveFromVersion, toVersion)
	if len(path) == 0 {
		// No explicit migrations, but we need to update version
		// This handles the case where config structure is compatible but version is old
		return m.updateVersion(data, toVersion)
	}

	// Apply migrations in order
	currentData := data
	for _, migration := range path {
		var err error
		currentData, err = migration.Migrate(currentData)
		if err != nil {
			return nil, fmt.Errorf("migration from v%d to v%d failed: %w",
				migration.FromVersion, migration.ToVersion, err)
		}
	}

	return currentData, nil
}

// GetMigrationPath returns the sequence of migrations needed to go from one version to another.
func (m *MigrationManager) GetMigrationPath(fromVersion, toVersion int) []Migration {
	var path []Migration

	currentVersion := fromVersion
	for currentVersion < toVersion {
		found := false
		for _, migration := range m.migrations {
			if migration.FromVersion == currentVersion {
				path = append(path, migration)
				currentVersion = migration.ToVersion
				found = true
				break
			}
		}
		if !found {
			// No migration found for this version, stop
			break
		}
	}

	return path
}

// HasMigrations returns true if there are migrations available.
func (m *MigrationManager) HasMigrations() bool {
	return len(m.migrations) > 0
}

// ListMigrations returns all registered migrations.
func (m *MigrationManager) ListMigrations() []Migration {
	return m.migrations
}

// updateVersion updates the version field in YAML data without other changes.
func (m *MigrationManager) updateVersion(data []byte, newVersion int) ([]byte, error) {
	// Parse, update version, and re-marshal
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	cfg.Version = newVersion

	newData, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated config: %w", err)
	}

	return newData, nil
}

// defaultMigrations contains the built-in migrations.
// Add new migrations here when making breaking config changes.
var defaultMigrations = []Migration{
	// Example migration (commented out - uncomment and modify when needed):
	// {
	// 	FromVersion: 1,
	// 	ToVersion:   2,
	// 	Description: "Rename network_discovery.enabled to network_discovery.active",
	// 	Migrate: func(data []byte) ([]byte, error) {
	// 		// Parse as generic map for field renaming
	// 		var raw map[string]interface{}
	// 		if err := yaml.Unmarshal(data, &raw); err != nil {
	// 			return nil, err
	// 		}
	//
	// 		// Perform the rename
	// 		if nd, ok := raw["network_discovery"].(map[string]interface{}); ok {
	// 			if enabled, exists := nd["enabled"]; exists {
	// 				nd["active"] = enabled
	// 				delete(nd, "enabled")
	// 			}
	// 		}
	//
	// 		// Update version
	// 		raw["version"] = 2
	//
	// 		return yaml.Marshal(raw)
	// 	},
	// },
}
