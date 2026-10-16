package app

// integrations.go wires the opt-in vendor API integrations (P-D12) to the
// config file, so an installer can answer each question before first start.

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/meraki"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

// VendorAPI is one vendor API integration's settings over the live config.
type VendorAPI struct {
	name            string
	store           managementStore
	field           func(*config.Config) *config.VendorAPIConfig
	defaultEndpoint string
}

// NewMeraki builds the Meraki Dashboard API settings over the live config,
// saved to path on every write.
func NewMeraki(cfg *config.Config, path string) *VendorAPI {
	return &VendorAPI{
		name:            "Meraki",
		store:           managementStore{cfg: cfg, path: path},
		field:           func(c *config.Config) *config.VendorAPIConfig { return &c.Integrations.Meraki },
		defaultEndpoint: meraki.DefaultEndpoint,
	}
}

// NewPaloAlto builds the PAN-OS XML API settings over the live config, saved
// to path on every write. PAN-OS has no default endpoint.
func NewPaloAlto(cfg *config.Config, path string) *VendorAPI {
	return &VendorAPI{
		name:  "Palo Alto",
		store: managementStore{cfg: cfg, path: path},
		field: func(c *config.Config) *config.VendorAPIConfig { return &c.Integrations.PaloAlto },
	}
}

// Settings returns the integration settings, with the vendor's endpoint
// filled in for an empty one and the key reported only as present or not.
func (v *VendorAPI) Settings() outbound.Settings {
	var out outbound.Settings
	v.store.Read(func(cfg *config.Config) {
		stored := v.field(cfg)
		out = outbound.Settings{
			Choice:    optInChoice(stored.Choice),
			Endpoint:  orDefault(stored.Endpoint, v.defaultEndpoint),
			HasAPIKey: stored.APIKey != "",
		}
	})
	return out
}

// Update validates and persists the operator's answer and settings. An
// endpoint equal to the vendor's is stored empty, so it follows the default.
func (v *VendorAPI) Update(u outbound.Update) error {
	return v.store.Write(func(cfg *config.Config) error {
		stored := v.field(cfg)
		if err := u.Validate(stored.APIKey != "", v.defaultEndpoint); err != nil {
			return err
		}
		next := config.VendorAPIConfig{
			Choice:   optInAnswer(u.Choice),
			Endpoint: emptyIfDefault(u.Endpoint, v.defaultEndpoint),
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

// Credentials implements outbound.Source: false unless the operator turned
// the integration on.
func (v *VendorAPI) Credentials(context.Context) (outbound.Credentials, bool, error) {
	var (
		stored config.VendorAPIConfig
		key    string
		err    error
	)
	v.store.Read(func(cfg *config.Config) {
		stored = *v.field(cfg)
		if stored.Choice == config.OptInOn {
			key, err = decryptSecret(cfg, stored.APIKey)
		}
	})
	if stored.Choice != config.OptInOn {
		return outbound.Credentials{}, false, nil
	}
	if err != nil {
		return outbound.Credentials{}, false, fmt.Errorf("decrypt %s API key: %w", v.name, err)
	}
	return outbound.Credentials{
		Endpoint: orDefault(stored.Endpoint, v.defaultEndpoint),
		APIKey:   key,
	}, true, nil
}
