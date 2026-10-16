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
)

func newMerakiServer(t *testing.T) (*Server, *config.Config, string) {
	t.Helper()
	cfg := &config.Config{}
	require.NoError(t, cfg.InitCredentialKeyring(t.TempDir()))
	path := filepath.Join(t.TempDir(), "config.json")
	settings := app.NewMeraki(cfg, path)
	s := &Server{meraki: settings}
	// Never started: these tests reach no network.
	s.merakiRefresher = meraki.NewRefresher(settings, meraki.NewClient(nil), meraki.RefreshInterval,
		slog.New(slog.DiscardHandler))
	return s, cfg, path
}

func serveMeraki(t *testing.T, s *Server, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleMerakiSettings(w, httptest.NewRequest(method, APIVersionPrefix+"/integrations/meraki",
		strings.NewReader(body)))
	return w
}

func TestHandleMerakiSettings(t *testing.T) {
	t.Parallel()
	s, cfg, path := newMerakiServer(t)

	w := serveMeraki(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"choice":"unanswered","endpoint":"`+meraki.DefaultEndpoint+`","hasApiKey":false}`,
		w.Body.String(), "a fresh install has not been asked and is off")

	w = serveMeraki(t, s, http.MethodPut, `{"choice":"on"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "needs an API key")

	w = serveMeraki(t, s, http.MethodPut, `{"choice":"on","apiKey":"hunter22hunter22"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "hunter22")
	var resp MerakiSettingsResponse
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
	w = serveMeraki(t, s, http.MethodPut, `{"choice":"off","endpoint":"https://api.meraki.ca/api/v1"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, outbound.ChoiceOff, resp.Choice)
	require.Equal(t, "https://api.meraki.ca/api/v1", resp.Endpoint)
	require.True(t, resp.HasAPIKey)

	w = serveMeraki(t, s, http.MethodPut, `{"choice":"unanswered"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, "the question cannot be un-asked")
}

func TestMerakiCredentials(t *testing.T) {
	t.Parallel()
	_, cfg, path := newMerakiServer(t)
	settings := app.NewMeraki(cfg, path)

	_, on, err := settings.Credentials(context.Background())
	require.NoError(t, err)
	require.False(t, on, "unanswered is off")

	require.NoError(t, settings.Update(meraki.Update{Choice: outbound.ChoiceOn, APIKey: "hunter22hunter22"}))
	creds, on, err := settings.Credentials(context.Background())
	require.NoError(t, err)
	require.True(t, on)
	require.Equal(t, meraki.Credentials{Endpoint: meraki.DefaultEndpoint, APIKey: "hunter22hunter22"}, creds)
}

func TestHandleMerakiDevicesOff(t *testing.T) {
	t.Parallel()
	s, _, _ := newMerakiServer(t)
	w := httptest.NewRecorder()
	s.handleMerakiDevices(w, httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/integrations/meraki/devices",
		http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"devices":[]}`, w.Body.String())
}
