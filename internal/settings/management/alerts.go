package management

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/escalation"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// The alert receivers' operator half: the webhook an operator names in
// Settings (#2605), the mail relay alerts are emailed through (#2997), the
// syslog collector they are forwarded to (#3037), the secrets that go to the
// keyring on the way in, and the escalation ladders that re-send an alert
// nobody acknowledged (P-B2).

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
		"escalations": escalationsView(cfg.Alerts.Escalations),
		"syslog": map[string]any{
			"host":      cfg.Alerts.Syslog.Host,
			"port":      cfg.Alerts.Syslog.Port,
			"transport": cfg.Alerts.Syslog.Transport,
		},
	}
}

// escalationLadder is the wire shape of one ladder in the settings read
// model and in an update.
type escalationLadder struct {
	Rule          string            `json:"rule"`
	Stages        []escalationStage `json:"stages"`
	RepeatSeconds int               `json:"repeatSeconds"`
}

type escalationStage struct {
	AfterSeconds int      `json:"afterSeconds"`
	Channels     []string `json:"channels"`
}

func escalationsView(stored []config.AlertEscalationConfig) []escalationLadder {
	out := make([]escalationLadder, 0, len(stored))
	for _, l := range stored {
		v := escalationLadder{Rule: l.Rule, RepeatSeconds: l.RepeatSeconds}
		for _, st := range l.Stages {
			v.Stages = append(v.Stages, escalationStage{AfterSeconds: st.AfterSeconds, Channels: st.Channels})
		}
		out = append(out, v)
	}
	return out
}

// EscalationLadders converts the stored escalation policy to the escalator's
// ladders, refusing the first one that could not run and any rule given two.
func EscalationLadders(stored []config.AlertEscalationConfig) ([]escalation.Ladder, error) {
	out := make([]escalation.Ladder, 0, len(stored))
	seen := make(map[string]bool, len(stored))
	for _, l := range stored {
		ladder := escalation.Ladder{Rule: l.Rule, Repeat: time.Duration(l.RepeatSeconds) * time.Second}
		for _, st := range l.Stages {
			stage := escalation.Stage{After: time.Duration(st.AfterSeconds) * time.Second}
			for _, c := range st.Channels {
				stage.Channels = append(stage.Channels, alerts.Channel(c))
			}
			ladder.Stages = append(ladder.Stages, stage)
		}
		if err := ladder.Validate(); err != nil {
			return nil, err
		}
		if seen[l.Rule] {
			return nil, fmt.Errorf("%w: %s has two ladders", escalation.ErrInvalidLadder, l.Rule)
		}
		seen[l.Rule] = true
		out = append(out, ladder)
	}
	return out, nil
}

// applyEscalationUpdates applies alerts.escalations. The list replaces the
// stored one whole, and a ladder that could not run is refused with its
// reason rather than stored and silently skipped.
func applyEscalationUpdates(section map[string]any, cfg *config.Config) error {
	val, exists := section["escalations"]
	if !exists {
		return nil
	}
	raw, err := json.Marshal(val)
	if err != nil {
		return errors.New("alerts.escalations must be a list of ladders")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var ladders []escalationLadder
	if err = dec.Decode(&ladders); err != nil {
		return fmt.Errorf("alerts.escalations must be a list of ladders: %w", err)
	}
	next := make([]config.AlertEscalationConfig, 0, len(ladders))
	for _, l := range ladders {
		stored := config.AlertEscalationConfig{Rule: strings.TrimSpace(l.Rule), RepeatSeconds: l.RepeatSeconds}
		for _, st := range l.Stages {
			stored.Stages = append(stored.Stages,
				config.AlertEscalationStage{AfterSeconds: st.AfterSeconds, Channels: st.Channels})
		}
		next = append(next, stored)
	}
	if _, err = EscalationLadders(next); err != nil {
		return err
	}
	cfg.Alerts.Escalations = next
	return nil
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
		applyEscalationUpdates(section, cfg),
		applySyslogUpdates(section, cfg),
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

// applySyslogUpdates applies alerts.syslog: a collector that could never
// receive is refused with its reason, and clearing the host turns forwarding
// off. There is no secret; the collector is named, not authenticated to.
func applySyslogUpdates(section map[string]any, cfg *config.Config) error {
	val, exists := section["syslog"]
	if !exists {
		return nil
	}
	fields, ok := val.(map[string]any)
	if !ok {
		return errors.New("alerts.syslog must be an object")
	}

	const prefix = "alerts.syslog"
	next := cfg.Alerts.Syslog
	var errs []error
	for _, field := range []struct {
		key string
		dst *string
	}{{"host", &next.Host}, {"transport", &next.Transport}} {
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
	if err := errors.Join(errs...); err != nil {
		return err
	}

	if next.Host == "" {
		cfg.Alerts.Syslog = config.AlertSyslogConfig{}
		return nil
	}
	if invalid := delivery.ValidateSyslog(delivery.SyslogConfig{
		Host: next.Host, Port: next.Port, Transport: delivery.SyslogTransport(next.Transport),
	}); invalid != nil {
		return invalid
	}
	cfg.Alerts.Syslog = next
	return nil
}
