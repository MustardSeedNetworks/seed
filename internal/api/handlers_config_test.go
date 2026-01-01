package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/testutil"
)

func TestHandleConfigVersion(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create server with minimal setup
	s := &Server{
		config:     cfg,
		configPath: configPath,
	}

	// Test GET /api/config/version
	req := httptest.NewRequest(http.MethodGet, "/api/config/version", http.NoBody)
	w := httptest.NewRecorder()
	s.handleConfigVersion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleConfigVersion() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ConfigVersionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Current != config.ConfigVersion {
		t.Errorf("Current version = %d, want %d", resp.Current, config.ConfigVersion)
	}
	if resp.Latest != config.ConfigVersion {
		t.Errorf("Latest version = %d, want %d", resp.Latest, config.ConfigVersion)
	}
	if resp.NeedsMigration {
		t.Error("NeedsMigration should be false for current version")
	}
}

func TestHandleConfigBackups(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create a backup first
	backupMgr := config.NewBackupManager(configPath, tmpDir, 10)
	if _, err := backupMgr.CreateBackup(); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	s := &Server{
		config:     cfg,
		configPath: configPath,
	}

	// Test GET /api/config/backups
	req := httptest.NewRequest(http.MethodGet, "/api/config/backups", http.NoBody)
	w := httptest.NewRecorder()
	s.handleConfigBackups(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleConfigBackups() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp BackupListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Backups) == 0 {
		t.Error("Expected at least one backup")
	}
}

func TestHandleConfigBackupCreate(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	s := &Server{
		config:     cfg,
		configPath: configPath,
	}

	// Test POST /api/config/backup
	req := httptest.NewRequest(http.MethodPost, "/api/config/backup", http.NoBody)
	w := httptest.NewRecorder()
	s.handleConfigBackupCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("handleConfigBackupCreate() status = %d, want %d; body: %s",
			w.Code, http.StatusCreated, w.Body.String())
	}

	var backup config.BackupInfo
	if err := json.NewDecoder(w.Body).Decode(&backup); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if backup.Name == "" {
		t.Error("Backup name should not be empty")
	}

	// Verify backup file exists
	if _, err := os.Stat(backup.Path); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}
}

func TestHandleConfigRestore(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().
		WithPort(8080).
		Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create a backup
	backupMgr := config.NewBackupManager(configPath, tmpDir, 10)
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify config
	cfg.Server.Port = 9999
	if saveErr := cfg.Save(configPath); saveErr != nil {
		t.Fatalf("Failed to save modified config: %v", saveErr)
	}

	s := &Server{
		config:     cfg,
		configPath: configPath,
	}

	// Test POST /api/config/restore
	reqBody := RestoreRequest{BackupName: backup.Name}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/config/restore", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleConfigRestore(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleConfigRestore() status = %d, want %d; body: %s",
			w.Code, http.StatusOK, w.Body.String())
	}

	// Verify config was restored
	if s.config.Server.Port != 8080 {
		t.Errorf("Config port = %d, want 8080 (restored value)", s.config.Server.Port)
	}
}

func TestHandleConfigBackupDelete(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create a backup
	backupMgr := config.NewBackupManager(configPath, tmpDir, 10)
	backup, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	s := &Server{
		config:     cfg,
		configPath: configPath,
	}

	// Test DELETE /api/config/backup/delete?name=...
	req := httptest.NewRequest(http.MethodDelete, "/api/config/backup/delete?name="+backup.Name, http.NoBody)
	w := httptest.NewRecorder()
	s.handleConfigBackupDelete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleConfigBackupDelete() status = %d, want %d; body: %s",
			w.Code, http.StatusOK, w.Body.String())
	}

	// Verify backup was deleted
	if _, statErr := os.Stat(backup.Path); !os.IsNotExist(statErr) {
		t.Error("Backup file should be deleted")
	}
}

func TestHandleConfigVersion_MethodNotAllowed(t *testing.T) {
	// Use testutil for consistent test configuration
	s := &Server{
		config:     testutil.NewConfigBuilder().Build(),
		configPath: "/tmp/config.yaml",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config/version", http.NoBody)
	w := httptest.NewRecorder()
	s.handleConfigVersion(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleConfigVersion(POST) status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleConfigRestore_MissingBackupName(t *testing.T) {
	// Use testutil for consistent test configuration
	s := &Server{
		config:     testutil.NewConfigBuilder().Build(),
		configPath: "/tmp/config.yaml",
	}

	body, _ := json.Marshal(RestoreRequest{BackupName: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/config/restore", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleConfigRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("handleConfigRestore() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
