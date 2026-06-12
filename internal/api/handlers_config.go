package api

import (
	"errors"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/config/backups"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// ============================================================================
// Config Backup/Restore Handlers (implements #494)
// ============================================================================

// ConfigVersionResponse contains config version information.
type ConfigVersionResponse struct {
	Current        int  `json:"current"`
	Latest         int  `json:"latest"`
	NeedsMigration bool `json:"needsMigration"`
}

// BackupListResponse contains a list of config backups.
type BackupListResponse struct {
	Backups []config.BackupInfo `json:"backups"`
}

// RestoreRequest contains the backup name to restore from.
type RestoreRequest struct {
	BackupName string `json:"backupName"`
}

// handleConfigBackups handles GET /api/config/backups - list all backups.
func (s *Server) handleConfigBackups(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	backupList, err := s.configBackups.List()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to list backups", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.api.internalError"),
			"",
		) // fixes #694
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, BackupListResponse{Backups: backupList})
}

// handleConfigBackupCreate handles POST /api/config/backup - create a new backup.
func (s *Server) handleConfigBackupCreate(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	backup, err := s.configBackups.Create()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to create backup", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.config.failedToCreateBackup"),
			"",
		) // fixes #694, #H7
		return
	}

	// Security audit log: config backup created (fixes #697)
	clientIP := s.getClientIP(r)
	logger.InfoContext(r.Context(), "Configuration backup created",
		"client_ip", clientIP,
		"backup_name", backup.Name,
		"event", "config.backup.create")

	sendJSONResponse(w, logger, http.StatusCreated, backup)
}

// handleConfigRestore handles POST /api/config/restore - restore from a backup.
func (s *Server) handleConfigRestore(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req RestoreRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	if req.BackupName == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.config.backupNameRequired"),
			"",
		) // fixes #694
		return
	}

	if err := s.configBackups.Restore(req.BackupName); err != nil {
		msgKey := "errors.config.failedToRestoreBackup"
		if errors.Is(err, backups.ErrReloadFailed) {
			msgKey = "errors.config.failedToReloadAfterRestore"
		}
		logger.ErrorContext(r.Context(), "Failed to restore backup", "error", err, "backup_name", req.BackupName)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T(msgKey), "",
		)
		return
	}

	// Security audit log: config restored from backup (fixes #697)
	clientIP := s.getClientIP(r)
	logger.InfoContext(r.Context(), "Configuration restored from backup",
		"client_ip", clientIP,
		"backup_name", req.BackupName,
		"event", "config.backup.restore")

	sendJSONResponse(
		w,
		logger,
		http.StatusOK,
		map[string]string{"status": "restored", "backup": req.BackupName},
	)
}

// handleConfigBackupDelete handles DELETE /api/config/backup - delete a backup.
func (s *Server) handleConfigBackupDelete(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	backupName := r.URL.Query().Get("name")
	if backupName == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.config.nameParamRequired"),
			"",
		) // fixes #694
		return
	}

	// Prevent path traversal attacks (fixes #683)
	// Only allow alphanumeric, dash, underscore, and dot characters
	if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(backupName) {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.config.invalidBackupName"),
			"",
		) // fixes #694
		return
	}
	// Strip any directory components as additional safety measure
	backupName = filepath.Base(backupName)

	if err := s.configBackups.Delete(backupName); err != nil {
		logger.ErrorContext(r.Context(), "Failed to delete backup", "error", err, "backup_name", backupName)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.config.failedToDeleteBackup"),
			"",
		) // fixes #694, #H7
		return
	}

	// Security audit log: config backup deleted (fixes #697)
	clientIP := s.getClientIP(r)
	logger.InfoContext(r.Context(), "Configuration backup deleted",
		"client_ip", clientIP,
		"backup_name", backupName,
		"event", "config.backup.delete")

	sendJSONResponse(
		w,
		logger,
		http.StatusOK,
		map[string]string{"status": "deleted", "backup": backupName},
	)
}

// handleConfigVersion handles GET /api/config/version - get config version info.
func (s *Server) handleConfigVersion(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	current, latest := s.configBackups.Version()
	resp := ConfigVersionResponse{
		Current:        current,
		Latest:         latest,
		NeedsMigration: current < latest,
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}
