package app

// settings.go wires the composition root to the settings-persistence and
// settings-management application (use-case) services (ADR-0020). The adapters
// below implement the narrow ports declared in internal/settings/persistence
// and internal/settings/management over the concrete collaborators (config,
// database), so the API handlers depend on use-cases instead of reaching into
// the server's service fields directly.

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
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

// NewSettingsManagement builds the settings-management use-case (ADR-0020,
// WS-A2) from the live config and the on-disk config path. cfg and path are
// fixed for the process lifetime.
func NewSettingsManagement(cfg *config.Config, path string) *management.Service {
	return management.NewService(managementStore{cfg: cfg, path: path})
}

// managementStore implements management.Store over the live config, owning
// the lock + on-disk save the port abstracts away.
type managementStore struct {
	cfg  *config.Config
	path string
}

// Read calls fn with the live config held under the config RLock.
func (s managementStore) Read(fn func(*config.Config)) {
	s.cfg.RLock()
	defer s.cfg.RUnlock()
	fn(s.cfg)
}

// Write calls fn with the live config held under the config Lock. The lock is
// released before Save acquires its own RLock to avoid the historic deadlock
// (fixes #783). If fn returns a non-nil error the config is not saved.
func (s managementStore) Write(fn func(*config.Config) error) error {
	s.cfg.Lock()
	err := fn(s.cfg)
	s.cfg.Unlock()
	if err != nil {
		return err
	}
	return s.cfg.Save(s.path)
}
