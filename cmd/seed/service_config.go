package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/paths"
)

// anchorServicePaths makes every storage path in cfg absolute. A relative path
// resolves against sp.Root, so the stock "data/seed.db" lands in sp.DataDir and
// an operator's "backups/x" lands beside it, never in the process's working
// directory. An unset database or log path takes the service default.
func anchorServicePaths(cfg *config.Config, sp *paths.Paths) {
	anchor := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(sp.Root, p)
	}
	cfg.Database.Path = anchor(firstSet(cfg.Database.Path, filepath.FromSlash(paths.DefaultDatabaseFile)))
	cfg.Logging.File = anchor(firstSet(cfg.Logging.File, filepath.FromSlash(paths.DefaultLogFile)))
	cfg.NetworkDiscovery.OUIFilePath = anchor(cfg.NetworkDiscovery.OUIFilePath)
	cfg.Server.CertFile = anchor(cfg.Server.CertFile)
	cfg.Server.KeyFile = anchor(cfg.Server.KeyFile)
}

func loadAndConfigureConfigForService(configPath string, opts securedir.Options) (*config.Config, error) {
	cfg, _, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		return nil, fmt.Errorf("load service config: %w", err)
	}

	if cfg.Auth.JWTSecret == "" {
		cfg.UpdateJWTSecret(auth.GenerateJWTSecret())
		if saveErr := savePrivateConfig(cfg, configPath, opts); saveErr != nil {
			return nil, fmt.Errorf("persist service JWT secret: %w", saveErr)
		}
	}

	if errors.Is(err, config.ErrInsecureCredentials) {
		cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	}
	if validateErr := cfg.Validate(); validateErr != nil {
		return nil, fmt.Errorf("validate service config: %w", validateErr)
	}
	return cfg, nil
}
