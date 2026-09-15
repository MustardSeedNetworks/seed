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
