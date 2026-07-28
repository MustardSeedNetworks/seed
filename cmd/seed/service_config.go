package main

import (
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

func loadAndConfigureConfigForService(configPath string) (*config.Config, error) {
	cfg, _, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		return nil, fmt.Errorf("load service config: %w", err)
	}

	if cfg.Auth.JWTSecret == "" {
		cfg.UpdateJWTSecret(auth.GenerateJWTSecret())
		if saveErr := cfg.Save(configPath); saveErr != nil {
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
