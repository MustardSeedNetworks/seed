// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/krisarmstrong/seed/internal/config"
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backupDir := filepath.Dir(s.configPath)
	backupMgr := config.NewBackupManager(s.configPath, backupDir, 10)

	backups, err := backupMgr.ListBackups()
	if err != nil {
		http.Error(w, "Failed to list backups: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(BackupListResponse{Backups: backups}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleConfigBackupCreate handles POST /api/config/backup - create a new backup.
func (s *Server) handleConfigBackupCreate(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(backup); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleConfigRestore handles POST /api/config/restore - restore from a backup.
func (s *Server) handleConfigRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	// Update server config (need to be careful with concurrent access)
	// Note: We copy individual fields to preserve the mutex in s.config
	s.config.Lock()
	s.config.Version = newCfg.Version
	s.config.Server = newCfg.Server
	s.config.Interface = newCfg.Interface
	s.config.VLAN = newCfg.VLAN
	s.config.IP = newCfg.IP
	s.config.Discovery = newCfg.Discovery
	s.config.NetworkDiscovery = newCfg.NetworkDiscovery
	s.config.DNS = newCfg.DNS
	s.config.Tests = newCfg.Tests
	s.config.Speedtest = newCfg.Speedtest
	s.config.Iperf = newCfg.Iperf
	s.config.Thresholds = newCfg.Thresholds
	s.config.Auth = newCfg.Auth
	s.config.Security = newCfg.Security
	s.config.DHCP = newCfg.DHCP
	s.config.SNMP = newCfg.SNMP
	s.config.FABOptions = newCfg.FABOptions
	s.config.DisplayOptions = newCfg.DisplayOptions
	s.config.Logging = newCfg.Logging
	s.config.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "restored", "backup": req.BackupName}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleConfigBackupDelete handles DELETE /api/config/backup - delete a backup.
func (s *Server) handleConfigBackupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backupName := r.URL.Query().Get("name")
	if backupName == "" {
		http.Error(w, "name parameter is required", http.StatusBadRequest)
		return
	}

	backupDir := filepath.Dir(s.configPath)
	backupMgr := config.NewBackupManager(s.configPath, backupDir, 10)

	if err := backupMgr.DeleteBackup(backupName); err != nil {
		http.Error(w, "Failed to delete backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "backup": backupName}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleConfigVersion handles GET /api/config/version - get config version info.
func (s *Server) handleConfigVersion(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
