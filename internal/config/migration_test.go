package config_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"gopkg.in/yaml.v3"
)

func TestMigrationManager_Migrate_NoMigrationsNeeded(t *testing.T) {
	mgr := NewMigrationManager()

	data := []byte("version: 1\nserver:\n  port: 8080\n")

	// Same version - no migration needed
	result, err := mgr.Migrate(data, 1, 1)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if !bytes.Equal(result, data) {
		t.Errorf("Migrate() modified data when no migration needed")
	}
}

func TestMigrationManager_Migrate_UnversionedConfig(t *testing.T) {
	mgr := NewMigrationManager()

	data := []byte("version: 0\nserver:\n  port: 8080\n")

	// Version 0 (unversioned) should be allowed and migrate to target version
	result, err := mgr.Migrate(data, 0, 1)
	if err != nil {
		t.Fatalf("Migrate() error = %v; version 0 (unversioned) should be allowed", err)
	}

	// Verify version was updated
	var partial struct {
		Version int `yaml:"version"`
	}
	if unmarshalErr := yaml.Unmarshal(result, &partial); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal result: %v", unmarshalErr)
	}

	if partial.Version != 1 {
		t.Errorf("Migrate() version = %d, want 1", partial.Version)
	}
}

func TestMigrationManager_Migrate_UpdateVersionOnly(t *testing.T) {
	mgr := NewMigrationManager()

	data := []byte("version: 1\nserver:\n  port: 8080\n")

	// No migrations registered, but version should be updated
	result, err := mgr.Migrate(data, 1, 2)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Parse result and check version
	var partial struct {
		Version int `yaml:"version"`
	}
	if unmarshalErr := yaml.Unmarshal(result, &partial); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal result: %v", unmarshalErr)
	}

	if partial.Version != 2 {
		t.Errorf("Migrate() version = %d, want 2", partial.Version)
	}
}

func TestMigrationManager_RegisterMigration(t *testing.T) {
	mgr := &MigrationManager{}

	// Register migrations out of order
	mgr.RegisterMigration(Migration{
		FromVersion: 3,
		ToVersion:   4,
		Description: "Migration 3->4",
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})
	mgr.RegisterMigration(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "Migration 1->2",
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})
	mgr.RegisterMigration(Migration{
		FromVersion: 2,
		ToVersion:   3,
		Description: "Migration 2->3",
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})

	// Should be sorted by FromVersion
	migrations := mgr.ListMigrations()
	if len(migrations) != 3 {
		t.Fatalf("Expected 3 migrations, got %d", len(migrations))
	}

	expectedOrder := []int{1, 2, 3}
	for i, m := range migrations {
		if m.FromVersion != expectedOrder[i] {
			t.Errorf("Migration %d has FromVersion = %d, want %d",
				i, m.FromVersion, expectedOrder[i])
		}
	}
}

func TestMigrationManager_GetMigrationPath(t *testing.T) {
	mgr := &MigrationManager{}

	mgr.RegisterMigration(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "Migration 1->2",
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})
	mgr.RegisterMigration(Migration{
		FromVersion: 2,
		ToVersion:   3,
		Description: "Migration 2->3",
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})
	mgr.RegisterMigration(Migration{
		FromVersion: 3,
		ToVersion:   4,
		Description: "Migration 3->4",
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})

	tests := []struct {
		name      string
		from      int
		to        int
		wantLen   int
		wantFirst int
		wantLast  int
	}{
		{
			name:      "single step",
			from:      1,
			to:        2,
			wantLen:   1,
			wantFirst: 1,
			wantLast:  1,
		},
		{
			name:      "multiple steps",
			from:      1,
			to:        4,
			wantLen:   3,
			wantFirst: 1,
			wantLast:  3,
		},
		{
			name:      "partial path",
			from:      2,
			to:        4,
			wantLen:   2,
			wantFirst: 2,
			wantLast:  3,
		},
		{
			name:    "no path needed",
			from:    4,
			to:      4,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := mgr.GetMigrationPath(tt.from, tt.to)

			if len(path) != tt.wantLen {
				t.Errorf("GetMigrationPath() returned %d migrations, want %d",
					len(path), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				if path[0].FromVersion != tt.wantFirst {
					t.Errorf("First migration FromVersion = %d, want %d",
						path[0].FromVersion, tt.wantFirst)
				}
				if path[len(path)-1].FromVersion != tt.wantLast {
					t.Errorf("Last migration FromVersion = %d, want %d",
						path[len(path)-1].FromVersion, tt.wantLast)
				}
			}
		})
	}
}

func TestMigrationManager_Migrate_ChainedMigrations(t *testing.T) {
	mgr := &MigrationManager{}

	// Register migrations that modify data
	mgr.RegisterMigration(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "Add field_a",
		Migrate: func(data []byte) ([]byte, error) {
			var raw map[string]any
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return nil, err
			}
			raw["field_a"] = "added_in_v2"
			raw["version"] = 2
			return yaml.Marshal(raw)
		},
	})
	mgr.RegisterMigration(Migration{
		FromVersion: 2,
		ToVersion:   3,
		Description: "Add field_b",
		Migrate: func(data []byte) ([]byte, error) {
			var raw map[string]any
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return nil, err
			}
			raw["field_b"] = "added_in_v3"
			raw["version"] = 3
			return yaml.Marshal(raw)
		},
	})

	data := []byte("version: 1\noriginal: value\n")

	result, err := mgr.Migrate(data, 1, 3)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	var final map[string]any
	if unmarshalErr := yaml.Unmarshal(result, &final); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal result: %v", unmarshalErr)
	}

	// Check all fields present
	if final["original"] != "value" {
		t.Error("Original field missing or modified")
	}
	if final["field_a"] != "added_in_v2" {
		t.Error("field_a not added by v1->v2 migration")
	}
	if final["field_b"] != "added_in_v3" {
		t.Error("field_b not added by v2->v3 migration")
	}
	if final["version"] != 3 {
		t.Errorf("Final version = %v, want 3", final["version"])
	}
}

func TestMigrationManager_HasMigrations(t *testing.T) {
	mgr := &MigrationManager{}

	if mgr.HasMigrations() {
		t.Error("Empty manager should not have migrations")
	}

	mgr.RegisterMigration(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Migrate:     func(data []byte) ([]byte, error) { return data, nil },
	})

	if !mgr.HasMigrations() {
		t.Error("Manager with migration should report HasMigrations")
	}
}

func TestLoadWithMigration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Create unversioned config
	data := []byte("server:\n  port: 9999\n")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	mgr := NewMigrationManager()

	cfg, migrated, err := LoadWithMigration(configPath, mgr)
	if err != nil {
		t.Fatalf("LoadWithMigration() error = %v", err)
	}

	if !migrated {
		t.Error("LoadWithMigration() should report migration for unversioned config")
	}

	if cfg.Version != ConfigVersion {
		t.Errorf("Config version = %d, want %d", cfg.Version, ConfigVersion)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("Config port = %d, want 9999", cfg.Server.Port)
	}
}
