package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestBackupManager_CreateBackup(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create initial config file
	cfg := config.DefaultConfig()
	cfg.Server.Port = 9999
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Create backup manager
	backupMgr := config.NewBackupManager(configPath, "", 10)

	// Create backup
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Verify backup exists
	if _, statErr := os.Stat(backup.Path); os.IsNotExist(statErr) {
		t.Errorf("Backup file does not exist: %s", backup.Path)
	}

	// Verify backup contains correct data
	data, readErr := os.ReadFile(backup.Path)
	if readErr != nil {
		t.Fatalf("Failed to read backup: %v", readErr)
	}

	var loadedCfg config.Config
	if unmarshalErr := json.Unmarshal(data, &loadedCfg); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal backup: %v", unmarshalErr)
	}

	if loadedCfg.Server.Port != 9999 {
		t.Errorf("Backup has wrong port = %d, want 9999", loadedCfg.Server.Port)
	}
}

func TestBackupManager_ListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config file
	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)

	// Create multiple backups with small delay
	for range 3 {
		if _, err := backupMgr.CreateBackup(); err != nil {
			t.Fatalf("CreateBackup() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List backups
	backups, err := backupMgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("ListBackups() returned %d backups, want 3", len(backups))
	}

	// Verify sorted newest first
	for i := 1; i < len(backups); i++ {
		if backups[i-1].CreatedAt.Before(backups[i].CreatedAt) {
			t.Errorf("Backups not sorted by time: %v before %v",
				backups[i-1].CreatedAt, backups[i].CreatedAt)
		}
	}
}

func TestBackupManager_RestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create and save original config
	cfg := config.DefaultConfig()
	cfg.Server.Port = 8080
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)

	// Create backup
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Modify config
	cfg.Server.Port = 9999
	if saveErr := cfg.Save(configPath); saveErr != nil {
		t.Fatalf("Failed to save modified config: %v", saveErr)
	}

	// Restore from backup
	if restoreErr := backupMgr.RestoreBackup(backup.Name); restoreErr != nil {
		t.Fatalf("RestoreBackup() error = %v", restoreErr)
	}

	// Verify restored config
	restored, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load restored config: %v", err)
	}

	if restored.Server.Port != 8080 {
		t.Errorf("Restored config has port = %d, want 8080", restored.Server.Port)
	}
}

func TestBackupManager_PruneOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config file
	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create manager with max 3 backups
	backupMgr := config.NewBackupManager(configPath, "", 3)

	// Create 5 backups
	for range 5 {
		if _, err := backupMgr.CreateBackup(); err != nil {
			t.Fatalf("CreateBackup() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// List should return only 3 (pruning happens during CreateBackup)
	backups, err := backupMgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("After pruning, got %d backups, want 3", len(backups))
	}
}

func TestBackupManager_DeleteBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config file
	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)

	// Create backup
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Delete backup
	if deleteErr := backupMgr.DeleteBackup(backup.Name); deleteErr != nil {
		t.Fatalf("DeleteBackup() error = %v", deleteErr)
	}

	// Verify deleted
	if _, statErr := os.Stat(backup.Path); !os.IsNotExist(statErr) {
		t.Errorf("Backup file still exists after delete")
	}
}

func TestBackupManager_DeleteBackup_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config file
	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)

	// Try to delete with path traversal
	err := backupMgr.DeleteBackup("../../../etc/passwd")
	if err == nil {
		t.Error("DeleteBackup() should reject path traversal")
	}

	// Try to delete non-backup file
	err = backupMgr.DeleteBackup("config.json")
	if err == nil {
		t.Error("DeleteBackup() should reject non-backup files")
	}
}

func TestBackupManager_RestoreBackup_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)

	// Try to restore with path traversal
	err := backupMgr.RestoreBackup("../../../etc/passwd")
	if err == nil {
		t.Error("RestoreBackup() should reject path traversal")
	}
}

func TestBackupManager_CreateBackup_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	backupMgr := config.NewBackupManager(configPath, "", 10)

	_, err := backupMgr.CreateBackup()
	if err == nil {
		t.Error("CreateBackup() should fail when config file doesn't exist")
	}
}

func TestBackupManager_ExtractVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	backupMgr := config.NewBackupManager(configPath, "", 10)

	tests := []struct {
		name string
		data string
		want int
	}{
		{
			name: "versioned config",
			data: `{"version": 5, "server": {"port": 8080}}`,
			want: 5,
		},
		{
			name: "unversioned config",
			data: `{"server": {"port": 8080}}`,
			want: 0,
		},
		{
			name: "invalid json",
			data: "not valid json",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backupMgr.ExtractVersion([]byte(tt.data))
			if got != tt.want {
				t.Errorf("ExtractVersion() = %d, want %d", got, tt.want)
			}
		})
	}
}
