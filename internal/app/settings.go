package app

// settings.go wires the composition root to the settings-persistence
// application (use-case) service (ADR-0020). The adapters below implement the
// narrow ports declared in internal/settings/persistence over the concrete
// database repositories and the live config, so the settings handlers depend on
// a use-case instead of reaching into ServiceContainer / s.db() directly.

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/settings/persistence"
)

// NewSettings builds the settings-persistence use-case (ADR-0020) from the lazy
// database accessor and the live config. The database is resolved through db on
// each call so a nil or later-assigned db is tolerated, preserving the handlers'
// historic "no db -> persist to config file only" behavior.
func NewSettings(db func() *database.DB, cfg *config.Config) *persistence.Service {
	return persistence.NewService(
		settingsProfileStore{db: db},
		settingsConfigSource{cfg: cfg},
	)
}

// settingsProfileStore implements persistence.ProfileStore over the database
// repositories. It resolves the *database.DB lazily (via the accessor) so it
// tolerates a nil or later-assigned database.
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
		return "", persistence.ErrNoProfile
	}
	profile, err := db.Profiles().GetDefault(ctx)
	if err != nil {
		// No default profile exists — nothing to persist to (not an error).
		return "", persistence.ErrNoProfile
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

// settingsConfigSource implements persistence.ConfigSource over the live config,
// serializing it with the single-source-of-truth profile encoder.
type settingsConfigSource struct {
	cfg *config.Config
}

func (c settingsConfigSource) ProfileJSON() (string, error) {
	return c.cfg.ToProfileJSON()
}
