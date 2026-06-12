// Package backups is the config backup/restore use-case (ADR-0020, WS-A9): the
// /api/config/backup* endpoints' application service. It owns the BackupManager
// (constructed once here instead of per-request in the transport layer) and the
// restore reload-and-apply sequence, so the handlers depend on a use-case instead
// of constructing infrastructure and mutating the live config inline.
package backups

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// DefaultMaxCount is the number of configuration backups retained.
const DefaultMaxCount = 10

var (
	// ErrRestoreFailed wraps a failure to restore the backup file.
	ErrRestoreFailed = errors.New("backups: restore failed")
	// ErrReloadFailed wraps a failure to reload the config after a restore.
	ErrReloadFailed = errors.New("backups: reload after restore failed")
)

// Service is the config backup/restore use-case.
type Service struct {
	mgr  *config.BackupManager
	cfg  *config.Config
	path string
}

// NewService builds the use-case over the live config and its on-disk path,
// constructing the BackupManager for the path's directory.
func NewService(cfg *config.Config, path string) *Service {
	return &Service{
		mgr:  config.NewBackupManager(path, filepath.Dir(path), DefaultMaxCount),
		cfg:  cfg,
		path: path,
	}
}

// List returns the available config backups.
func (s *Service) List() ([]config.BackupInfo, error) { return s.mgr.ListBackups() }

// Create writes a new backup of the current config.
func (s *Service) Create() (*config.BackupInfo, error) { return s.mgr.CreateBackup() }

// Delete removes the named backup.
func (s *Service) Delete(name string) error { return s.mgr.DeleteBackup(name) }

// Restore restores the named backup, reloads the config from disk, and copies the
// reloaded fields into the live config under its write lock. A restore-file
// failure returns ErrRestoreFailed; a reload failure returns ErrReloadFailed (the
// transport layer maps the two to distinct messages).
func (s *Service) Restore(name string) error {
	if err := s.mgr.RestoreBackup(name); err != nil {
		return fmt.Errorf("%w: %w", ErrRestoreFailed, err)
	}
	newCfg, err := config.Load(s.path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrReloadFailed, err)
	}
	// CopyFieldsFrom uses struct literals for compile-time checking that no fields
	// are missed; the lock guards the live config against concurrent readers.
	s.cfg.Lock()
	defer s.cfg.Unlock()
	s.cfg.CopyFieldsFrom(newCfg)
	return nil
}

// Version returns the live config's schema version and the latest known version.
func (s *Service) Version() (int, int) {
	s.cfg.RLock()
	current := s.cfg.Version
	s.cfg.RUnlock()
	return current, config.ConfigVersion
}
