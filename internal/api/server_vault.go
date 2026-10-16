package api

import (
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

// initVaultUseCases wires the use-cases that need the keyring owning the
// credential DEK: the vault itself, configuration backup, which decrypts the
// vault's SSH passwords to log in, and Windows inventory, which decrypts its
// WinRM passwords. Without a config there is no keyring, so
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
	if svc, credErr := app.NewDeviceCredentials(s.persistence, keyring); credErr == nil {
		s.deviceCredentials = svc
	}
	alertWriter := func() deviceconfig.AlertWriter { return s.alertWriter }
	if svc, backupErr := app.NewDeviceConfigBackups(
		s.persistence,
		keyring,
		deviceconfig.SSHFetcher{},
		alertWriter,
	); backupErr == nil {
		s.deviceConfigs = svc
	}
	if svc, wmiErr := app.NewWMIInventory(s.persistence, keyring, wmi.WSManCollector{}); wmiErr == nil {
		s.wmiInventory = svc
	}
}
