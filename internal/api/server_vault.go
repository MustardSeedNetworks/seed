package api

import (
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

// initVaultUseCases wires the use-cases that need the keyring owning the
// credential DEK: the vault itself, and configuration backup, which decrypts
// the vault's SSH passwords to log in. Without a config there is no keyring, so
// both stay nil and their handlers report 503 — the alternative is a CRUD
// surface that would persist plaintext.
func (s *Server) initVaultUseCases() {
	if s.config == nil {
		return
	}
	keyring, err := s.config.CredentialKeyring()
	if err != nil {
		return
	}
	if svc, credErr := app.NewDeviceCredentials(s.db, keyring); credErr == nil {
		s.deviceCredentials = svc
	}
	if svc, backupErr := app.NewDeviceConfigBackups(s.db, keyring, deviceconfig.SSHFetcher{}); backupErr == nil {
		s.deviceConfigs = svc
	}
}
