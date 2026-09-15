package management_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
)

// #2605's four acceptance clauses as tests: a receiver an operator can set, a
// secret that never reaches config.json in plaintext or a response at all, a
// URL that can be cleared, and a URL that is refused with a reason rather than
// accepted and silently dead.

// fakeEncrypter stands in for config.Keyring. The prefix is the real one, so a
// test that asserts "the stored value is ciphertext" asserts the same shape
// production writes.
type fakeEncrypter struct{ err error }

func (f fakeEncrypter) EncryptValue(plaintext string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "enc:v1:" + plaintext, nil
}

// countingReconfigurer records that the running webhook was told to re-read.
type countingReconfigurer struct{ calls int }

func (c *countingReconfigurer) ReconfigureAlerts() { c.calls++ }

func webhookUpdate(fields map[string]any) map[string]any {
	return map[string]any{"alerts": map[string]any{"webhook": fields}}
}

func TestUpdateStoresWebhookSecretAsCiphertext(t *testing.T) {
	cfg := config.DefaultConfig()
	reconfig := &countingReconfigurer{}
	svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, reconfig)

	err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://receiver.example.com/hook", "secret": "s3cret",
	}), "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := cfg.Alerts.Webhook.URL; got != "https://receiver.example.com/hook" {
		t.Errorf("stored url = %q", got)
	}
	if got := cfg.Alerts.Webhook.Secret; got == "s3cret" || !config.IsEncrypted(got) {
		t.Errorf("stored secret = %q, want keyring ciphertext (enc: prefix)", got)
	}
	if reconfig.calls != 1 {
		t.Errorf("reconfigure called %d times, want 1 — the receiver must change "+
			"without a daemon restart", reconfig.calls)
	}
}

func TestGetNeverServesTheWebhookSecret(t *testing.T) {
	cfg := config.DefaultConfig()
	svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, nil)
	if err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://receiver.example.com/hook", "secret": "s3cret",
	}), ""); err != nil {
		t.Fatalf("Update: %v", err)
	}

	settings, _ := svc.Get()
	section, ok := settings["alerts"].(map[string]any)
	if !ok {
		t.Fatalf("settings has no alerts section: %+v", settings)
	}
	webhook, ok := section["webhook"].(map[string]any)
	if !ok {
		t.Fatalf("alerts section has no webhook: %+v", section)
	}
	if _, present := webhook["secret"]; present {
		t.Error("the read model serves the signing material; it must report only that one is set")
	}
	if webhook["secretSet"] != true {
		t.Errorf("secretSet = %v, want true", webhook["secretSet"])
	}
	if webhook["url"] != "https://receiver.example.com/hook" {
		t.Errorf("url = %v", webhook["url"])
	}
}

func TestUpdateRepointsURLWithoutRetypingTheSecret(t *testing.T) {
	cfg := config.DefaultConfig()
	svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, nil)
	if err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://first.example.com/hook", "secret": "s3cret",
	}), ""); err != nil {
		t.Fatalf("first Update: %v", err)
	}
	stored := cfg.Alerts.Webhook.Secret

	// The operator cannot read the secret back, so a URL-only edit must not
	// demand one.
	if err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://second.example.com/hook",
	}), ""); err != nil {
		t.Fatalf("url-only Update: %v", err)
	}
	if cfg.Alerts.Webhook.URL != "https://second.example.com/hook" {
		t.Errorf("url = %q", cfg.Alerts.Webhook.URL)
	}
	if cfg.Alerts.Webhook.Secret != stored {
		t.Errorf("secret changed on a url-only edit: %q", cfg.Alerts.Webhook.Secret)
	}
}

func TestUpdateClearingTheURLDropsTheSecretToo(t *testing.T) {
	cfg := config.DefaultConfig()
	svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, nil)
	if err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://receiver.example.com/hook", "secret": "s3cret",
	}), ""); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := svc.Update(webhookUpdate(map[string]any{"url": ""}), ""); err != nil {
		t.Fatalf("clearing Update: %v", err)
	}
	if cfg.Alerts.Webhook != (config.AlertWebhookConfig{}) {
		t.Errorf("clearing the url left %+v; the old secret must not silently "+
			"re-arm the next time any url is set", cfg.Alerts.Webhook)
	}
}

func TestUpdateRefusesAWebhookThatCouldNeverDeliver(t *testing.T) {
	for name, fields := range map[string]map[string]any{
		"unparseable url":     {"url": "ht tp://receiver", "secret": "s3cret"},
		"not http":            {"url": "ftp://receiver.example.com/hook", "secret": "s3cret"},
		"no host":             {"url": "https:///hook", "secret": "s3cret"},
		"carries userinfo":    {"url": "https://user:pw@receiver.example.com/h", "secret": "s3cret"},
		"no secret":           {"url": "https://receiver.example.com/hook"},
		"url is not a string": {"url": 42},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, nil)
			err := svc.Update(webhookUpdate(fields), "")
			if !errors.Is(err, management.ErrValidation) {
				t.Fatalf("Update error = %v, want ErrValidation", err)
			}
			if cfg.Alerts.Webhook.URL != "" {
				t.Errorf("a refused webhook was stored: %+v", cfg.Alerts.Webhook)
			}
		})
	}
}

func TestUpdateRefusesTheSecretWithNoKeyring(t *testing.T) {
	cfg := config.DefaultConfig()
	svc := management.NewService(&fakeStore{cfg: cfg}, nil, nil)
	err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://receiver.example.com/hook", "secret": "s3cret",
	}), "")
	if !errors.Is(err, management.ErrValidation) {
		t.Fatalf("Update error = %v, want ErrValidation", err)
	}
	if strings.Contains(cfg.Alerts.Webhook.Secret, "s3cret") {
		t.Error("the signing material was written in plaintext when no keyring was available")
	}
}

func TestWebhookChangeMovesTheETag(t *testing.T) {
	cfg := config.DefaultConfig()
	svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, nil)
	_, before := svc.Get()

	if err := svc.Update(webhookUpdate(map[string]any{
		"url": "https://receiver.example.com/hook", "secret": "s3cret",
	}), ""); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, after := svc.Get(); after == before {
		t.Error("the ETag did not move when the webhook changed; a conditional " +
			"write to this section would be blind to a concurrent edit")
	}
}
