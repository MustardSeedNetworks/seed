package app

// meraki.go wires the opt-in Meraki Dashboard API integration (P-D12) to the
// config file, so an installer can answer its question before first start.

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/meraki"
)

// Meraki is the integration's settings over the live config.
type Meraki struct {
	store managementStore
}

// NewMeraki builds the integration settings over the live config, saved to
// path on every write.
func NewMeraki(cfg *config.Config, path string) *Meraki {
	return &Meraki{store: managementStore{cfg: cfg, path: path}}
}

// Settings returns the integration settings, with Meraki's endpoint filled in
// for an empty one and the key reported only as present or not.
func (m *Meraki) Settings() meraki.Settings {
	var out meraki.Settings
	m.store.Read(func(cfg *config.Config) {
		stored := cfg.Integrations.Meraki
		out = meraki.Settings{
			Choice:    optInChoice(stored.Choice),
			Endpoint:  orDefault(stored.Endpoint, meraki.DefaultEndpoint),
			HasAPIKey: stored.APIKey != "",
		}
	})
	return out
}

// Update validates and persists the operator's answer and settings. An
// endpoint equal to Meraki's is stored empty, so it follows the default.
func (m *Meraki) Update(u meraki.Update) error {
	return m.store.Write(func(cfg *config.Config) error {
		stored := &cfg.Integrations.Meraki
		if err := u.Validate(stored.APIKey != ""); err != nil {
			return err
		}
		next := config.MerakiConfig{
			Choice:   optInAnswer(u.Choice),
			Endpoint: emptyIfDefault(u.Endpoint, meraki.DefaultEndpoint),
			APIKey:   stored.APIKey,
		}
		switch {
		case u.ClearAPIKey:
			next.APIKey = ""
		case u.APIKey != "":
			ciphertext, err := configKeyring{cfg: cfg}.EncryptValue(u.APIKey)
			if err != nil {
				// The message names the field, never the value.
				return fmt.Errorf("encrypt apiKey: %w", err)
			}
			next.APIKey = ciphertext
		}
		*stored = next
		return nil
	})
}

// Credentials implements meraki.Source: false unless the operator turned the
// integration on.
func (m *Meraki) Credentials(context.Context) (meraki.Credentials, bool, error) {
	var (
		stored config.MerakiConfig
		key    string
		err    error
	)
	m.store.Read(func(cfg *config.Config) {
		stored = cfg.Integrations.Meraki
		if stored.Choice == config.OptInOn {
			key, err = decryptSecret(cfg, stored.APIKey)
		}
	})
	if stored.Choice != config.OptInOn {
		return meraki.Credentials{}, false, nil
	}
	if err != nil {
		return meraki.Credentials{}, false, fmt.Errorf("decrypt Meraki API key: %w", err)
	}
	return meraki.Credentials{
		Endpoint: orDefault(stored.Endpoint, meraki.DefaultEndpoint),
		APIKey:   key,
	}, true, nil
}
