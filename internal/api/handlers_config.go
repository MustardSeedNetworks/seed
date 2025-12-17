// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Config Backup/Restore Handlers (implements #494)
// ============================================================================

// ConfigVersionResponse contains config version information.
type ConfigVersionResponse struct {
	Current        int  `json:"current"`
	Latest         int  `json:"latest"`
	NeedsMigration bool `json:"needs_migration"`
}

// BackupListResponse contains a list of config backups.
type BackupListResponse struct {
	Backups []config.BackupInfo `json:"backups"`
}

// RestoreRequest contains the backup name to restore from.
type RestoreRequest struct {
	BackupName string `json:"backup_name"`
}

// handleConfigBackups handles GET /api/config/backups - list all backups.
func (s *Server) handleConfigBackups(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backupDir := filepath.Dir(s.configPath)
	backupMgr := config.NewBackupManager(s.configPath, backupDir, 10)

	backups, err := backupMgr.ListBackups()
	if err != nil {
		logger.Error("Failed to list backups", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, BackupListResponse{Backups: backups})
}

// handleConfigBackupCreate handles POST /api/config/backup - create a new backup.
func (s *Server) handleConfigBackupCreate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backupDir := filepath.Dir(s.configPath)
	backupMgr := config.NewBackupManager(s.configPath, backupDir, 10)

	backup, err := backupMgr.CreateBackup()
	if err != nil {
		http.Error(w, "Failed to create backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Security audit log: config backup created (fixes #697)
	clientIP := GetClientIP(r)
	logger.Info("Configuration backup created",
		"client_ip", clientIP,
		"backup_name", backup.Name,
		"event", "config.backup.create")

	sendJSONResponse(w, logger, http.StatusCreated, backup)
}

// handleConfigRestore handles POST /api/config/restore - restore from a backup.
func (s *Server) handleConfigRestore(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeConfig)

	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.BackupName == "" {
		http.Error(w, "backup_name is required", http.StatusBadRequest)
		return
	}

	backupDir := filepath.Dir(s.configPath)
	backupMgr := config.NewBackupManager(s.configPath, backupDir, 10)

	if err := backupMgr.RestoreBackup(req.BackupName); err != nil {
		http.Error(w, "Failed to restore backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload config after restore
	newCfg, err := config.Load(s.configPath)
	if err != nil {
		http.Error(w, "Backup restored but failed to reload config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update server config using CopyFieldsFrom to prevent race conditions (fixes #691)
	// CopyFieldsFrom uses struct literals for compile-time checking that no fields are missed
	s.config.Lock()
	s.config.CopyFieldsFrom(newCfg)
	s.config.Unlock()

	// Security audit log: config restored from backup (fixes #697)
	clientIP := GetClientIP(r)
	logger.Info("Configuration restored from backup",
		"client_ip", clientIP,
		"backup_name", req.BackupName,
		"event", "config.backup.restore")

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "restored", "backup": req.BackupName})
}

// handleConfigBackupDelete handles DELETE /api/config/backup - delete a backup.
func (s *Server) handleConfigBackupDelete(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backupName := r.URL.Query().Get("name")
	if backupName == "" {
		http.Error(w, "name parameter is required", http.StatusBadRequest)
		return
	}

	// Prevent path traversal attacks (fixes #683)
	// Only allow alphanumeric, dash, underscore, and dot characters
	if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(backupName) {
		http.Error(w, "Invalid backup name", http.StatusBadRequest)
		return
	}
	// Strip any directory components as additional safety measure
	backupName = filepath.Base(backupName)

	backupDir := filepath.Dir(s.configPath)
	backupMgr := config.NewBackupManager(s.configPath, backupDir, 10)

	if err := backupMgr.DeleteBackup(backupName); err != nil {
		http.Error(w, "Failed to delete backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Security audit log: config backup deleted (fixes #697)
	clientIP := GetClientIP(r)
	logger.Info("Configuration backup deleted",
		"client_ip", clientIP,
		"backup_name", backupName,
		"event", "config.backup.delete")

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "deleted", "backup": backupName})
}

// handleConfigVersion handles GET /api/config/version - get config version info.
func (s *Server) handleConfigVersion(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.config.RLock()
	currentVersion := s.config.Version
	s.config.RUnlock()

	resp := ConfigVersionResponse{
		Current:        currentVersion,
		Latest:         config.ConfigVersion,
		NeedsMigration: currentVersion < config.ConfigVersion,
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}
