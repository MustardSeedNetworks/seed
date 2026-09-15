package management

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// The alert webhook's operator half (#2605): the receiver an operator names in
// Settings, and the signing material that goes to the keyring on the way in.

// Encrypter turns plaintext into keyring ciphertext. Satisfied by
// config.Keyring; declared here so the service holds only the seam and
// plaintext signing material never reaches the config file.
type Encrypter interface {
	EncryptValue(plaintext string) (string, error)
}

// buildAlertSettings is the read model. It reports whether signing material is
// stored, never the material: a secret served back is a secret readable by
// anyone who can read the settings, which is the whole reason it is in the
// keyring.
func buildAlertSettings(cfg *config.Config) map[string]any {
	return map[string]any{
		"webhook": map[string]any{
			"url":       cfg.Alerts.Webhook.URL,
			"secretSet": cfg.Alerts.Webhook.Secret != "",
		},
	}
}

// applyAlertsUpdates applies the alert-webhook section.
//
// A URL that could never receive a POST is refused here rather than stored:
// the alternative is a setting that looks accepted and silently never
// delivers, which is the failure #2606 had to make visible in the inbox.
// Signing material is required whenever a URL is set — the receiver verifies
// origin by signature, so an unsigned receiver is not a receiver — but it may
// be the material already stored, so re-pointing a URL does not make the
// operator re-type a secret they cannot read back.
func applyAlertsUpdates(updates map[string]any, cfg *config.Config, encrypt Encrypter) error {
	val, exists := updates["alerts"]
	if !exists {
		return nil
	}
	section, ok := val.(map[string]any)
	if !ok {
		return errors.New("alerts must be an object")
	}
	hookVal, exists := section["webhook"]
	if !exists {
		return nil
	}
	webhook, ok := hookVal.(map[string]any)
	if !ok {
		return errors.New("alerts.webhook must be an object")
	}

	url, urlGiven, err := extractString(webhook, "url", "alerts.webhook")
	if err != nil {
		return err
	}
	secret, secretGiven, err := extractString(webhook, "secret", "alerts.webhook")
	if err != nil {
		return err
	}
	url = strings.TrimSpace(url)
	secret = strings.TrimSpace(secret)

	next := cfg.Alerts.Webhook
	if urlGiven {
		next.URL = url
	}
	if secretGiven && secret != "" {
		ciphertext, encErr := encrypt.EncryptValue(secret)
		if encErr != nil {
			// The message names the field, never the value.
			return fmt.Errorf("encrypt alerts.webhook.secret: %w", encErr)
		}
		next.Secret = ciphertext
	}

	if next.URL == "" {
		// Clearing the URL turns delivery off. The signing material goes with
		// it: leaving it behind would silently re-arm the old secret the next
		// time any URL was set.
		cfg.Alerts.Webhook = config.AlertWebhookConfig{}
		return nil
	}

	if err := delivery.ValidateURL(next.URL); err != nil {
		return err
	}
	if next.Secret == "" {
		return errors.New("alerts.webhook.secret is required whenever a url is set; " +
			"the receiver verifies the signature to know the alert came from Seed")
	}

	cfg.Alerts.Webhook = next
	return nil
}
