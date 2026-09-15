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

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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
// fixed for the process lifetime. webhook resolves the running alert-delivery
// Manager, which the service re-points after a write so a Settings edit takes
// effect without a daemon restart (#2605). It is a getter rather than a value
// because the Manager is built later than this service, when the database (its
// delivery recorder) is up; nil, or a getter returning nil, means there is no
// webhook to re-point.
func NewSettingsManagement(
	cfg *config.Config,
	path string,
	webhook func() *alertdelivery.Manager,
) *management.Service {
	keyring, err := cfg.CredentialKeyring()
	if err != nil {
		// The settings service refuses the webhook section without a keyring,
		// which is the honest outcome: there is nowhere safe to put a secret.
		logging.GetLogger().Error("credential keyring unavailable; "+
			"the alert webhook secret cannot be stored", "error", err)
		keyring = nil
	}
	var reconfig management.Reconfigurer
	if webhook != nil {
		reconfig = alertReconfigurer{manager: webhook, cfg: cfg}
	}
	return management.NewService(managementStore{cfg: cfg, path: path}, keyringOrNil(keyring), reconfig)
}

// keyringOrNil keeps a nil *config.Keyring from becoming a non-nil interface
// holding a nil pointer, which would pass the service's nil check and then
// panic on first use.
func keyringOrNil(kr *config.Keyring) management.Encrypter {
	if kr == nil {
		return nil
	}
	return kr
}

// alertReconfigurer re-points the running alert webhook at what was just
// written. It is the one place plaintext signing material exists outside the
// operator's browser: decrypted here, handed to the notifier, never stored.
type alertReconfigurer struct {
	manager func() *alertdelivery.Manager
	cfg     *config.Config
}

func (a alertReconfigurer) ReconfigureAlerts() {
	if m := a.manager(); m != nil {
		ApplyAlertWebhook(a.cfg, m)
	}
}

// ApplyAlertWebhook points m at the receiver the config names. It is the one
// place plaintext signing material exists outside the operator's browser:
// decrypted here, handed to the notifier, never stored. Startup and every
// later settings write both come through it, so there is one decision about
// what a stored webhook means.
func ApplyAlertWebhook(cfg *config.Config, m *alertdelivery.Manager) {
	cfg.RLock()
	webhook := cfg.Alerts.Webhook
	cfg.RUnlock()

	logger := logging.GetLogger()
	if webhook.URL == "" {
		m.Apply(alertdelivery.Config{})
		return
	}
	secret := webhook.Secret
	if config.IsEncrypted(secret) {
		keyring, err := cfg.CredentialKeyring()
		if err == nil {
			secret, err = keyring.DecryptValue(webhook.Secret)
		}
		if err != nil {
			// A secret this install cannot decrypt — an imported profile from
			// another deployment, whose keyring is not this one — disables
			// delivery rather than signing with ciphertext no receiver expects.
			logger.Error("alert webhook secret could not be decrypted; "+
				"delivery is off until it is set again", "error", err)
			m.Apply(alertdelivery.Config{})
			return
		}
	}
	m.Apply(alertdelivery.Config{URL: webhook.URL, Secret: secret})
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
