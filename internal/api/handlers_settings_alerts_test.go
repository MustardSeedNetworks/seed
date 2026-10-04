package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// The alert receiver's transport half (#2605), through the real handler and the
// real settings service.
//
// The clause worth a test at this level is the fourth acceptance bullet: an
// invalid URL is refused *with a reason*. Asserting the service returns
// ErrValidation does not prove that — the handler used to map every validation
// failure to "Invalid settings format. Check server logs for details.", so the
// reason never left the daemon and the operator saw a refusal they could not
// act on.

func putSettings(t *testing.T, server *api.Server, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.HandleSettings(w, req)
	return w
}

func webhookBody(fields map[string]any) map[string]any {
	return map[string]any{"alerts": map[string]any{"webhook": fields}}
}

// errorText reads the field the UI's api client actually shows the operator
// (`body.error`), not the one a reader might assume.
func errorText(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error   string `json:"error"`
		Details string `json:"details"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return body.Error
}

func TestPutSettingsRefusesAnUndeliverableWebhookWithItsReason(t *testing.T) {
	for name, tc := range map[string]struct {
		fields map[string]any
		want   string
	}{
		"wrong scheme": {
			webhookBody(map[string]any{"url": "ftp://receiver.example.com/h", "secret": "s3cret"})["alerts"].(map[string]any),
			"not http or https",
		},
		"carries userinfo": {
			webhookBody(map[string]any{"url": "https://u:p@receiver.example.com/h", "secret": "s3cret"})["alerts"].(map[string]any),
			"userinfo",
		},
		"no signing material": {
			webhookBody(map[string]any{"url": "https://receiver.example.com/h"})["alerts"].(map[string]any),
			"secret is required",
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			w := putSettings(t, server, map[string]any{"alerts": tc.fields})

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, http.StatusBadRequest, w.Body)
			}
			if got := errorText(t, w); !strings.Contains(got, tc.want) {
				t.Errorf("error = %q, want it to contain %q — a refusal the operator "+
					"cannot act on is indistinguishable from one that silently did nothing", got, tc.want)
			}
		})
	}
}

func TestPutSettingsStoresTheWebhookAndGetNeverServesTheSecret(t *testing.T) {
	cfg := config.DefaultConfig()
	server := api.NewTestServerWithConfig(cfg)
	defer server.Close()

	w := putSettings(t, server, webhookBody(map[string]any{
		"url": "https://receiver.example.com/hook", "secret": "s3cret",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body)
	}

	if got := cfg.Alerts.Webhook.Secret; got == "" || got == "s3cret" || !config.IsEncrypted(got) {
		t.Errorf("stored secret = %q, want keyring ciphertext", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
	read := httptest.NewRecorder()
	server.HandleSettings(read, req)
	if read.Code != http.StatusOK {
		t.Fatalf("GET status = %d", read.Code)
	}
	if strings.Contains(read.Body.String(), "s3cret") {
		t.Fatal("GET /api/v1/settings served the signing material back")
	}
	var settings struct {
		Alerts struct {
			Webhook struct {
				URL       string `json:"url"`
				SecretSet bool   `json:"secretSet"`
			} `json:"webhook"`
		} `json:"alerts"`
	}
	if err := json.NewDecoder(read.Body).Decode(&settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings.Alerts.Webhook.URL != "https://receiver.example.com/hook" {
		t.Errorf("url = %q", settings.Alerts.Webhook.URL)
	}
	if !settings.Alerts.Webhook.SecretSet {
		t.Error("secretSet = false after a secret was stored")
	}
}

func emailBody(fields map[string]any) map[string]any {
	return map[string]any{"alerts": map[string]any{"email": fields}}
}

// The mail relay (#2997) follows the webhook's rules: refused with a reason,
// the password stored as ciphertext and never served back.
func TestPutSettingsStoresTheMailRelayAndGetNeverServesThePassword(t *testing.T) {
	cfg := config.DefaultConfig()
	server := api.NewTestServerWithConfig(cfg)
	defer server.Close()

	w := putSettings(t, server, emailBody(map[string]any{
		"host": " mail.example.com ", "port": 587, "tls": "starttls",
		"username": "seed-alerts", "password": "relay-pass",
		"from": "Seed <seed@example.com>", "to": []any{"noc@example.com", " "},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body)
	}
	if got := cfg.Alerts.Email.Password; got == "relay-pass" || !config.IsEncrypted(got) {
		t.Errorf("stored password = %q, want keyring ciphertext", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
	read := httptest.NewRecorder()
	server.HandleSettings(read, req)
	if strings.Contains(read.Body.String(), "relay-pass") {
		t.Fatal("GET /api/v1/settings served the relay password back")
	}
	var settings struct {
		Alerts struct {
			Email struct {
				Host        string   `json:"host"`
				Port        int      `json:"port"`
				TLS         string   `json:"tls"`
				Username    string   `json:"username"`
				PasswordSet bool     `json:"passwordSet"`
				From        string   `json:"from"`
				To          []string `json:"to"`
			} `json:"email"`
		} `json:"alerts"`
	}
	if err := json.NewDecoder(read.Body).Decode(&settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	got := settings.Alerts.Email
	if got.Host != "mail.example.com" || got.Port != 587 || got.TLS != "starttls" ||
		got.Username != "seed-alerts" || !got.PasswordSet || got.From != "Seed <seed@example.com>" ||
		len(got.To) != 1 || got.To[0] != "noc@example.com" {
		t.Errorf("GET alerts.email = %+v", got)
	}

	// Re-pointing the relay keeps the password the operator cannot read back;
	// clearing the host takes it with it.
	if repoint := putSettings(
		t,
		server,
		emailBody(map[string]any{"host": "relay2.example.com"}),
	); repoint.Code != http.StatusOK {
		t.Fatalf("re-point status = %d (body %s)", repoint.Code, repoint.Body)
	}
	if !config.IsEncrypted(cfg.Alerts.Email.Password) {
		t.Error("re-pointing the host dropped the stored password")
	}
	if cleared := putSettings(t, server, emailBody(map[string]any{"host": ""})); cleared.Code != http.StatusOK {
		t.Fatalf("clear status = %d (body %s)", cleared.Code, cleared.Body)
	}
	if cfg.Alerts.Email.Password != "" || len(cfg.Alerts.Email.To) != 0 {
		t.Errorf("clearing the host left %+v behind", cfg.Alerts.Email)
	}
}

func TestPutSettingsRefusesAnUndeliverableMailRelayWithItsReason(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{
			"host": "mail.example.com", "from": "seed@example.com", "to": []any{"noc@example.com"},
		}
	}
	for name, tc := range map[string]struct {
		mutate func(map[string]any)
		want   string
	}{
		"plaintext":          {func(f map[string]any) { f["tls"] = "none" }, `tls mode "none"`},
		"sender":             {func(f map[string]any) { f["from"] = "seed" }, `sender "seed"`},
		"no recipients":      {func(f map[string]any) { f["to"] = []any{} }, "at least one recipient"},
		"recipients as text": {func(f map[string]any) { f["to"] = "noc@example.com" }, "alerts.email.to must be a list"},
		"host with port":     {func(f map[string]any) { f["host"] = "mail.example.com:25" }, "bare host name"},
	} {
		t.Run(name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()
			fields := valid()
			tc.mutate(fields)

			w := putSettings(t, server, emailBody(fields))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body)
			}
			if got := errorText(t, w); !strings.Contains(got, tc.want) {
				t.Errorf("error = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

func syslogBody(fields map[string]any) map[string]any {
	return map[string]any{"alerts": map[string]any{"syslog": fields}}
}

// The syslog collector (#3037): stored as given, served back whole (it holds
// no secret), refused with a reason when it could never receive, and off when
// the host is cleared.
func TestPutSettingsStoresTheSyslogCollector(t *testing.T) {
	cfg := config.DefaultConfig()
	server := api.NewTestServerWithConfig(cfg)
	defer server.Close()

	w := putSettings(t, server, syslogBody(map[string]any{
		"host": " siem.example.com ", "port": 6514, "transport": "tls",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
	read := httptest.NewRecorder()
	server.HandleSettings(read, req)
	var settings struct {
		Alerts struct {
			Syslog config.AlertSyslogConfig `json:"syslog"`
		} `json:"alerts"`
	}
	if err := json.NewDecoder(read.Body).Decode(&settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	want := config.AlertSyslogConfig{Host: "siem.example.com", Port: 6514, Transport: "tls"}
	if settings.Alerts.Syslog != want {
		t.Errorf("GET alerts.syslog = %+v, want %+v", settings.Alerts.Syslog, want)
	}

	for name, tc := range map[string]struct {
		fields map[string]any
		want   string
	}{
		"unknown transport": {map[string]any{"transport": "relp"}, `syslog transport "relp"`},
		"host with port":    {map[string]any{"host": "siem.example.com:514"}, "bare host name"},
		"port as text":      {map[string]any{"port": "514"}, "alerts.syslog.port"},
	} {
		t.Run(name, func(t *testing.T) {
			refused := putSettings(t, server, syslogBody(tc.fields))
			if refused.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %s)", refused.Code, refused.Body)
			}
			if got := errorText(t, refused); !strings.Contains(got, tc.want) {
				t.Errorf("error = %q, want it to contain %q", got, tc.want)
			}
			if cfg.Alerts.Syslog != want {
				t.Errorf("a refused update changed the stored collector to %+v", cfg.Alerts.Syslog)
			}
		})
	}

	if cleared := putSettings(t, server, syslogBody(map[string]any{"host": ""})); cleared.Code != http.StatusOK {
		t.Fatalf("clear status = %d (body %s)", cleared.Code, cleared.Body)
	}
	if cfg.Alerts.Syslog != (config.AlertSyslogConfig{}) {
		t.Errorf("clearing the host left %+v behind", cfg.Alerts.Syslog)
	}
}
