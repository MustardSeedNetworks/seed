package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/meraki"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
	"github.com/MustardSeedNetworks/seed/internal/paloalto"
)

func newIntegrationsServer(t *testing.T) (*Server, *config.Config, string) {
	t.Helper()
	cfg := &config.Config{}
	require.NoError(t, cfg.InitCredentialKeyring(t.TempDir()))
	path := filepath.Join(t.TempDir(), "config.json")
	merakiSettings := app.NewMeraki(cfg, path)
	paloAltoSettings := app.NewPaloAlto(cfg, path)
	discard := slog.New(slog.DiscardHandler)
	// Never started: these tests reach no network.
	s := &Server{
		meraki: merakiSettings,
		merakiReader: outbound.NewReader("meraki", merakiSettings, meraki.NewClient(nil).Devices,
			meraki.RefreshInterval, discard),
		paloAlto: paloAltoSettings,
		paloAltoReader: outbound.NewReader("paloalto", paloAltoSettings, paloalto.NewClient(nil).Devices,
			paloalto.RefreshInterval, discard),
	}
	return s, cfg, path
}

func serveIntegration(t *testing.T, handler http.HandlerFunc, vendor, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(method, APIVersionPrefix+"/integrations/"+vendor, strings.NewReader(body)))
	return w
}

func TestHandleMerakiSettings(t *testing.T) {
	t.Parallel()
	s, cfg, path := newIntegrationsServer(t)
	serve := func(method, body string) *httptest.ResponseRecorder {
		return serveIntegration(t, s.handleMerakiSettings, "meraki", method, body)
	}

	w := serve(http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"choice":"unanswered","endpoint":"`+meraki.DefaultEndpoint+`","hasApiKey":false}`,
		w.Body.String(), "a fresh install has not been asked and is off")

	w = serve(http.MethodPut, `{"choice":"on"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "needs an API key")

	w = serve(http.MethodPut, `{"choice":"on","apiKey":"hunter22hunter22"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "hunter22")
	var resp IntegrationSettingsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, outbound.ChoiceOn, resp.Choice)
	require.True(t, resp.HasAPIKey)

	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(saved), "hunter22", "the key is stored as keyring ciphertext")
	stored := cfg.Integrations.Meraki
	require.True(t, config.IsEncrypted(stored.APIKey))
	require.Empty(t, stored.Endpoint, "Meraki's endpoint is stored as the default, not copied")

	// Answering off keeps the stored key when none is re-typed.
	w = serve(http.MethodPut, `{"choice":"off","endpoint":"https://api.meraki.ca/api/v1"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, outbound.ChoiceOff, resp.Choice)
	require.Equal(t, "https://api.meraki.ca/api/v1", resp.Endpoint)
	require.True(t, resp.HasAPIKey)

	w = serve(http.MethodPut, `{"choice":"unanswered"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, "the question cannot be un-asked")
}

func TestHandlePaloAltoSettings(t *testing.T) {
	t.Parallel()
	s, cfg, _ := newIntegrationsServer(t)
	serve := func(method, body string) *httptest.ResponseRecorder {
		return serveIntegration(t, s.handlePaloAltoSettings, "paloalto", method, body)
	}

	w := serve(http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"choice":"unanswered","endpoint":"","hasApiKey":false}`, w.Body.String(),
		"PAN-OS has no default endpoint")

	w = serve(http.MethodPut, `{"choice":"on","apiKey":"hunter22hunter22"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "needs an endpoint")

	w = serve(http.MethodPut, `{"choice":"on","apiKey":"hunter22hunter22","endpoint":"https://fw1.example.net"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp IntegrationSettingsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, outbound.ChoiceOn, resp.Choice)
	require.Equal(t, "https://fw1.example.net", resp.Endpoint)
	require.True(t, resp.HasAPIKey)
	require.Equal(t, "https://fw1.example.net", cfg.Integrations.PaloAlto.Endpoint)
	require.Empty(t, cfg.Integrations.Meraki, "each integration is its own question")
}

func TestVendorAPICredentials(t *testing.T) {
	t.Parallel()
	_, cfg, path := newIntegrationsServer(t)
	settings := app.NewMeraki(cfg, path)

	_, on, err := settings.Credentials(context.Background())
	require.NoError(t, err)
	require.False(t, on, "unanswered is off")

	require.NoError(t, settings.Update(outbound.Update{Choice: outbound.ChoiceOn, APIKey: "hunter22hunter22"}))
	creds, on, err := settings.Credentials(context.Background())
	require.NoError(t, err)
	require.True(t, on)
	require.Equal(t, outbound.Credentials{Endpoint: meraki.DefaultEndpoint, APIKey: "hunter22hunter22"}, creds)
}

func TestHandleIntegrationDevicesOff(t *testing.T) {
	t.Parallel()
	s, _, _ := newIntegrationsServer(t)
	for vendor, handler := range map[string]http.HandlerFunc{
		"meraki":   s.handleMerakiDevices,
		"paloalto": s.handlePaloAltoDevices,
	} {
		w := serveIntegration(t, handler, vendor+"/devices", http.MethodGet, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.JSONEq(t, `{"devices":[]}`, w.Body.String(), vendor)
	}
}
