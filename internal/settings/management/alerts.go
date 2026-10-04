package management

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// The alert receivers' operator half: the webhook an operator names in
// Settings (#2605), the mail relay alerts are emailed through (#2997), and the
// secrets that go to the keyring on the way in.

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
	email := cfg.Alerts.Email
	// An unset list reads as [] rather than null, so a client iterates it
	// without a guard.
	to := email.To
	if to == nil {
		to = []string{}
	}
	return map[string]any{
		"webhook": map[string]any{
			"url":       cfg.Alerts.Webhook.URL,
			"secretSet": cfg.Alerts.Webhook.Secret != "",
		},
		"email": map[string]any{
			"host":        email.Host,
			"port":        email.Port,
			"tls":         email.TLS,
			"username":    email.Username,
			"passwordSet": email.Password != "",
			"from":        email.From,
			"to":          to,
		},
	}
}

// applyAlertsUpdates applies the alerts section: each receiver present in it.
func applyAlertsUpdates(updates map[string]any, cfg *config.Config, encrypt Encrypter) error {
	val, exists := updates["alerts"]
	if !exists {
		return nil
	}
	section, ok := val.(map[string]any)
	if !ok {
		return errors.New("alerts must be an object")
	}
	return errors.Join(
		applyWebhookUpdates(section, cfg, encrypt),
		applyEmailUpdates(section, cfg, encrypt),
	)
}

// applyWebhookUpdates applies alerts.webhook.
//
// A URL that could never receive a POST is refused here rather than stored:
// the alternative is a setting that looks accepted and silently never
// delivers, which is the failure #2606 had to make visible in the inbox.
// Signing material is required whenever a URL is set — the receiver verifies
// origin by signature, so an unsigned receiver is not a receiver — but it may
// be the material already stored, so re-pointing a URL does not make the
// operator re-type a secret they cannot read back.
func applyWebhookUpdates(section map[string]any, cfg *config.Config, encrypt Encrypter) error {
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

	if invalid := delivery.ValidateURL(next.URL); invalid != nil {
		return invalid
	}
	if next.Secret == "" {
		return errors.New("alerts.webhook.secret is required whenever a url is set; " +
			"the receiver verifies the signature to know the alert came from Seed")
	}

	cfg.Alerts.Webhook = next
	return nil
}

// applyEmailUpdates applies alerts.email. It follows the webhook's rules: a
// relay that could never deliver is refused with its reason, clearing the
// host turns email off and takes the stored password with it, and an omitted
// or empty password keeps the one already stored, which the operator cannot
// read back.
func applyEmailUpdates(section map[string]any, cfg *config.Config, encrypt Encrypter) error {
	val, exists := section["email"]
	if !exists {
		return nil
	}
	fields, ok := val.(map[string]any)
	if !ok {
		return errors.New("alerts.email must be an object")
	}

	next := cfg.Alerts.Email
	password, passwordGiven, err := readEmailFields(fields, &next)
	if err != nil {
		return err
	}

	if next.Host == "" {
		cfg.Alerts.Email = config.AlertEmailConfig{}
		return nil
	}
	if next.Username == "" {
		// A relay that admits by address has no use for a password, and a
		// stored one would silently return the next time a username is set.
		next.Password = ""
	} else if passwordGiven && password != "" {
		ciphertext, encErr := encrypt.EncryptValue(password)
		if encErr != nil {
			// The message names the field, never the value.
			return fmt.Errorf("encrypt alerts.email.password: %w", encErr)
		}
		next.Password = ciphertext
	}

	// The password is left out: what is stored is ciphertext, and it was
	// cleared above wherever it would have no username.
	if invalid := delivery.ValidateEmail(delivery.EmailConfig{
		Host: next.Host, Port: next.Port, TLS: delivery.TLSMode(next.TLS),
		Username: next.Username, From: next.From, To: next.To,
	}); invalid != nil {
		return invalid
	}
	cfg.Alerts.Email = next
	return nil
}

// readEmailFields overlays the alerts.email fields present in the update onto
// next, and returns the password separately: it is stored only as ciphertext.
func readEmailFields(fields map[string]any, next *config.AlertEmailConfig) (string, bool, error) {
	const prefix = "alerts.email"
	var errs []error
	for _, field := range []struct {
		key string
		dst *string
	}{{"host", &next.Host}, {"tls", &next.TLS}, {"username", &next.Username}, {"from", &next.From}} {
		s, given, err := extractString(fields, field.key, prefix)
		if err != nil {
			errs = append(errs, err)
		} else if given {
			*field.dst = strings.TrimSpace(s)
		}
	}
	if port, given, err := extractInt(fields, "port", prefix); err != nil {
		errs = append(errs, err)
	} else if given {
		next.Port = port
	}
	if to, given, err := extractStringList(fields, "to", prefix); err != nil {
		errs = append(errs, err)
	} else if given {
		next.To = make([]string, 0, len(to))
		for _, addr := range to {
			if addr = strings.TrimSpace(addr); addr != "" {
				next.To = append(next.To, addr)
			}
		}
	}
	password, passwordGiven, err := extractString(fields, "password", prefix)
	if err != nil {
		errs = append(errs, err)
	}
	return password, passwordGiven, errors.Join(errs...)
}
