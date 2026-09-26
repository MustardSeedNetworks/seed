package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
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

// TestBackupInfoWireKeysAreCamelCase pins the JSON the backup list emits.
//
// BackupInfo is never written to disk — it is derived from the backup
// directory on every call and serialized straight onto the API — so its tags
// are wire tags and ADR-0010 makes them camelCase. `created_at` shipped
// instead, and the Config Backups settings panel reads `backup.createdAt`, so
// every backup's timestamp rendered from `undefined`. scripts/check-json-casing.sh
// did not catch it: that gate scans internal/api and internal/discovery only.
func TestBackupInfoWireKeysAreCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(config.BackupInfo{
		Name:      "seed.yaml.20260908-120000.bak",
		Path:      "/var/lib/seed/backups/seed.yaml.20260908-120000.bak",
		Size:      2048,
		CreatedAt: time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		Version:   3,
	})
	if err != nil {
		t.Fatalf("marshal BackupInfo: %v", err)
	}

	var wire map[string]any
	if err = json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("unmarshal wire: %v", err)
	}

	for _, key := range []string{"name", "path", "size", "createdAt", "version"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("wire is missing %q; got %v", key, wire)
		}
	}
	if _, ok := wire["created_at"]; ok {
		t.Error(`wire still carries snake_case "created_at" (ADR-0010)`)
	}
}

// TestBackupManager_SiblingDirectoryEscape proves RestoreBackup and
// DeleteBackup reject a name that steps out of backupDir into a sibling
// directory whose name happens to share backupDir's name as a string
// prefix (e.g. "backups" vs "backupsXXX") — the case a bare
// [strings.HasPrefix](cleanPath, cleanBackupDir) check (without a trailing
// separator) let through before both methods moved to [os.Root] confinement.
func TestBackupManager_SiblingDirectoryEscape(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("mkdir backupDir: %v", err)
	}

	// A sibling directory whose name has backupDir's name as a prefix.
	siblingDir := filepath.Join(tmpDir, "backupsXXX")
	if err := os.MkdirAll(siblingDir, 0o750); err != nil {
		t.Fatalf("mkdir siblingDir: %v", err)
	}
	evilPath := filepath.Join(siblingDir, "evil.json")
	if err := os.WriteFile(evilPath, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatalf("write evil file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, backupDir, 10)
	escapeName := filepath.Join("..", "backupsXXX", "evil.json")

	tests := []struct {
		name string
		call func() error
	}{
		{"RestoreBackup", func() error { return backupMgr.RestoreBackup(escapeName) }},
		{"DeleteBackup", func() error { return backupMgr.DeleteBackup(escapeName) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Error("expected an error confining the name to backupDir, got nil")
			}
			if _, statErr := os.Stat(evilPath); statErr != nil {
				t.Errorf("sibling file was affected: %v", statErr)
			}
		})
	}
}

// TestBackupManager_RestoreBackup_NormalPath proves a legitimate backup
// name (produced by CreateBackup, living directly under backupDir) is
// still accepted by RestoreBackup's [os.Root] confinement.
func TestBackupManager_RestoreBackup_NormalPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := config.DefaultConfig()
	cfg.Server.Port = 5555
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	if restoreErr := backupMgr.RestoreBackup(backup.Name); restoreErr != nil {
		t.Fatalf("RestoreBackup(%q) = %v, want nil", backup.Name, restoreErr)
	}
}

// TestBackupManager_RestoreBackup_ConfigPathIsSymlink proves RestoreBackup
// still writes through m.configPath when it is itself a symlink to a file
// elsewhere (an alternatives-style config layout) — the case a bare
// [os.Root] scoped to configPath's own (unresolved) directory would
// reject, since [os.Root] refuses to follow a symlink that leaves its
// scope. The write-back goes through fsutil.RootAt, which resolves
// symlinks first.
func TestBackupManager_RestoreBackup_ConfigPathIsSymlink(t *testing.T) {
	realDir := t.TempDir()
	realConfigPath := filepath.Join(realDir, "config.json")

	cfg := config.DefaultConfig()
	cfg.Server.Port = 6666
	if err := cfg.Save(realConfigPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	linkDir := t.TempDir()
	configPath := filepath.Join(linkDir, "config.json")
	if err := os.Symlink(realConfigPath, configPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	backupMgr := config.NewBackupManager(configPath, "", 10)
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	if restoreErr := backupMgr.RestoreBackup(backup.Name); restoreErr != nil {
		t.Fatalf("RestoreBackup(%q) = %v, want nil", backup.Name, restoreErr)
	}

	// The write-back should have landed on the real file behind the
	// symlink, not silently failed or written somewhere else.
	restored, err := os.ReadFile(realConfigPath)
	if err != nil {
		t.Fatalf("read real config file: %v", err)
	}
	var wire map[string]any
	if unmarshalErr := json.Unmarshal(restored, &wire); unmarshalErr != nil {
		t.Fatalf("unmarshal restored config: %v", unmarshalErr)
	}
}
