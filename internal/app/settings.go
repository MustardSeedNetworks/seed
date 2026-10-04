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
	"slices"

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/config/backups"
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
	var reconfig management.Reconfigurer
	if webhook != nil {
		reconfig = alertReconfigurer{manager: webhook, cfg: cfg}
	}
	return management.NewService(managementStore{cfg: cfg, path: path}, configKeyring{cfg: cfg}, reconfig)
}

// NewConfigBackups builds the config backup/restore use-case (ADR-0020) over the
// live config and its on-disk path. A restore re-points the running alert
// webhook through webhook, the same lazy getter NewSettingsManagement takes, so
// the restored receiver is the one that delivers (#2928).
func NewConfigBackups(cfg *config.Config, path string, webhook func() *alertdelivery.Manager) *backups.Service {
	return backups.NewService(cfg, path, alertReconfigurer{manager: webhook, cfg: cfg})
}

// configKeyring encrypts through whatever keyring the config holds *at the
// time of the write*, rather than capturing one at construction. Capturing
// would be a latent bug wherever this service is built before
// InitCredentialKeyring runs: ensureKeyring installs an ephemeral keyring on
// first use, init later replaces it, and every secret written in between would
// be encrypted with a DEK that dies at restart — decrypting to an error on the
// next boot, which for the webhook means delivery silently off.
type configKeyring struct{ cfg *config.Config }

func (k configKeyring) EncryptValue(plaintext string) (string, error) {
	keyring, err := k.cfg.CredentialKeyring()
	if err != nil {
		return "", err
	}
	return keyring.EncryptValue(plaintext)
}

// alertReconfigurer re-points the running alert receivers at what was just
// written.
type alertReconfigurer struct {
	manager func() *alertdelivery.Manager
	cfg     *config.Config
}

func (a alertReconfigurer) ReconfigureAlerts() {
	if m := a.manager(); m != nil {
		ApplyAlertReceivers(a.cfg, m)
	}
}

// ApplyAlertReceivers points m at the receivers and escalation ladders the
// config names. It is the
// one place plaintext receiver secrets exist outside the operator's browser:
// decrypted here, handed to the notifier, never stored. Startup and every
// later settings write both come through it, so there is one decision about
// what a stored receiver means.
//
// A secret this install cannot decrypt — an imported profile from another
// deployment, whose keyring is not this one — turns that one channel off
// rather than authenticating with ciphertext no receiver expects.
func ApplyAlertReceivers(cfg *config.Config, m *alertdelivery.Manager) {
	cfg.RLock()
	webhook := cfg.Alerts.Webhook
	email := cfg.Alerts.Email
	email.To = slices.Clone(email.To)
	ladders, ladderErr := management.EscalationLadders(cfg.Alerts.Escalations)
	syslog := cfg.Alerts.Syslog
	cfg.RUnlock()

	// The settings write refuses a ladder that could not run, so this is a
	// hand-edited or restored config. Running part of a policy would page
	// the wrong people on the wrong schedule; none of it runs until fixed.
	if ladderErr != nil {
		logging.GetLogger().Error("alert escalation ladders are invalid; "+
			"no alert is escalated until they are fixed", "error", ladderErr)
	}
	m.ApplyEscalations(ladders)

	if webhook.URL == "" {
		m.ApplyWebhook(alertdelivery.WebhookConfig{})
	} else if secret, err := decryptSecret(cfg, webhook.Secret); err != nil {
		logging.GetLogger().Error("alert webhook secret could not be decrypted; "+
			"delivery is off until it is set again", "error", err)
		m.ApplyWebhook(alertdelivery.WebhookConfig{})
	} else {
		m.ApplyWebhook(alertdelivery.WebhookConfig{URL: webhook.URL, Secret: secret})
	}

	if email.Host == "" {
		m.ApplyEmail(alertdelivery.EmailConfig{})
	} else if password, err := decryptSecret(cfg, email.Password); err != nil {
		logging.GetLogger().Error("alert email password could not be decrypted; "+
			"email is off until it is set again", "error", err)
		m.ApplyEmail(alertdelivery.EmailConfig{})
	} else {
		m.ApplyEmail(alertdelivery.EmailConfig{
			Host:     email.Host,
			Port:     email.Port,
			TLS:      alertdelivery.TLSMode(email.TLS),
			Username: email.Username,
			Password: password,
			From:     email.From,
			To:       email.To,
		})
	}

	m.ApplySyslog(alertdelivery.SyslogConfig{
		Host:      syslog.Host,
		Port:      syslog.Port,
		Transport: alertdelivery.SyslogTransport(syslog.Transport),
	})
}

// decryptSecret returns stored keyring ciphertext as plaintext. A value
// without the `enc:` prefix is returned as it is.
func decryptSecret(cfg *config.Config, stored string) (string, error) {
	if !config.IsEncrypted(stored) {
		return stored, nil
	}
	keyring, err := cfg.CredentialKeyring()
	if err != nil {
		return "", err
	}
	return keyring.DecryptValue(stored)
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
