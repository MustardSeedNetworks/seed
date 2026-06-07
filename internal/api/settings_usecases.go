package api

// settings_usecases.go wires the API layer to the settings application
// (use-case) service (ADR-0016 strangle phase 3). The adapters below implement
// the narrow ports declared in internal/settings/app over the concrete database
// repositories and the live config, so the settings handlers depend on a
// use-case instead of reaching into ServiceContainer / s.db() directly.

import (
	"context"
	"fmt"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	settingsapp "github.com/krisarmstrong/seed/internal/settings/app"
)

// settingsProfileStore implements settingsapp.ProfileStore over the database
// repositories. It resolves the *database.DB lazily (via the accessor) so it
// tolerates a nil or later-assigned database, preserving the handlers' historic
// "no db -> persist to config file only" behavior.
type settingsProfileStore struct {
	db func() *database.DB
}

func (s settingsProfileStore) ActiveProfileID(ctx context.Context) (string, error) {
	db := s.db()
	if db == nil {
		return "", nil // no database; the (also empty) default path makes this a no-op
	}
	return db.Settings().GetValue(ctx, database.SettingKeyActiveProfile)
}

func (s settingsProfileStore) DefaultProfileID(ctx context.Context) (string, error) {
	db := s.db()
	if db == nil {
		return "", settingsapp.ErrNoProfile
	}
	profile, err := db.Profiles().GetDefault(ctx)
	if err != nil {
		// No default profile exists — nothing to persist to (not an error).
		return "", settingsapp.ErrNoProfile
	}
	return profile.ID, nil
}

func (s settingsProfileStore) SaveProfileConfig(ctx context.Context, id, configJSON string) error {
	db := s.db()
	if db == nil {
		return nil
	}
	profile, err := db.Profiles().Get(ctx, id)
	if err != nil {
		return fmt.Errorf("load profile %s: %w", id, err)
	}
	profile.ConfigJSON = configJSON
	return db.Profiles().Update(ctx, profile)
}

// settingsConfigSource implements settingsapp.ConfigSource over the live config,
// serializing it with the single-source-of-truth profile encoder.
type settingsConfigSource struct {
	cfg *config.Config
}

func (c settingsConfigSource) ProfileJSON() (string, error) {
	return c.cfg.ToProfileJSON()
}

// initSettingsUseCase builds the settings-persistence use-case (ADR-0016 phase
// 3) from the lazy database accessor and the live config.
func (s *Server) initSettingsUseCase() {
	s.settingsStore = settingsapp.NewPersistence(
		settingsProfileStore{db: s.db},
		settingsConfigSource{cfg: s.config},
	)
}
