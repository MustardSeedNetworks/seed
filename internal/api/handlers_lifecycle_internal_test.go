package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
)

func newLifecycleServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{}
	s.persist = app.NewPersistence(newTestDB(t))
	s.lifecycleEoX = app.NewLifecycleStore(s.persistence)
	return s
}

func serveLifecycleEoX(
	t *testing.T,
	s *Server,
	method, body string,
) (*httptest.ResponseRecorder, LifecycleEoXResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleLifecycleEoX(w, httptest.NewRequest(method, APIVersionPrefix+"/lifecycle/eox", strings.NewReader(body)))
	var resp LifecycleEoXResponse
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	}
	return w, resp
}

// The acceptance of P-D9, first half: lifecycle data loads from a local
// source. The operator's downloaded EoX response is imported, kept, and
// cleared by importing a response with no records.
func TestLifecycleEoXImportAndClear(t *testing.T) {
	t.Parallel()
	s := newLifecycleServer(t)

	w, resp := serveLifecycleEoX(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []lifecycle.EoXRecord{}, resp.Records)

	fixture, err := os.ReadFile("../lifecycle/testdata/eox-response.json")
	require.NoError(t, err)
	w, resp = serveLifecycleEoX(t, s, http.MethodPut, string(fixture))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, resp.Records, 2)
	require.Equal(t, "C9300-48P", resp.Records[0].ProductID)
	require.Equal(t, "WS-C3750X-48P-S", resp.Records[1].ProductID)

	w, resp = serveLifecycleEoX(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, resp.Records, 2, "the import survives a fresh read from settings")

	w, resp = serveLifecycleEoX(t, s, http.MethodPut, `{"EOXRecord":[]}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []lifecycle.EoXRecord{}, resp.Records)
}

func TestLifecycleEoXRejectsAndKeepsTheTable(t *testing.T) {
	t.Parallel()
	s := newLifecycleServer(t)
	w, _ := serveLifecycleEoX(t, s, http.MethodPut,
		`{"EOXRecord":[{"EOLProductID":"WS-C3750X-48P-S","EndOfSaleDate":{"value":"2016-10-30"}}]}`)
	require.Equal(t, http.StatusOK, w.Code)

	for name, body := range map[string]string{
		"csv":      "EOLProductID,EndOfSaleDate\nWS-C3750X-48P-S,2016-10-30",
		"not eox":  `{"records":[]}`,
		"bad date": `{"EOXRecord":[{"EOLProductID":"A-1","EndOfSaleDate":{"value":"30/10/2016"}}]}`,
	} {
		rejected, _ := serveLifecycleEoX(t, s, http.MethodPut, body)
		require.Equal(t, http.StatusBadRequest, rejected.Code, name)
	}
	_, resp := serveLifecycleEoX(t, s, http.MethodGet, "")
	require.Len(t, resp.Records, 1)
}

func TestLifecycleDevices(t *testing.T) {
	table, err := lifecycle.New([]lifecycle.EoXRecord{
		{ProductID: "WS-C3750X-48P-S", EndOfSale: "2016-10-30", LastDateOfSupport: "2021-10-31"},
		{ProductID: "C3KX-PWR-715WAC", EndOfSale: "2026-12-01"},
	})
	require.NoError(t, err)
	devices := []*discovery.DiscoveredDevice{
		{IP: "10.44.10.20", DisplayName: "access-2", SNMPData: &discovery.SNMPFullData{
			Inventory: []discovery.SNMPEntity{
				{Index: 1001, Class: "powerSupply", Name: "Switch 1 - Power Supply A", ModelName: "C3KX-PWR-715WAC"},
				{Index: 1, Class: "chassis", Name: "1", ModelName: " WS-C3750X-48P-S ", SerialNum: "FDO1234X0AB"},
				{Index: 1010, Class: "port", Name: "Gi1/0/1"},
				{Index: 1100, Class: "module", Name: "GigabitEthernet1/1/1", ModelName: "GLC-TE"},
			},
		}},
		{IP: "10.44.10.3", SNMPData: &discovery.SNMPFullData{
			Inventory: []discovery.SNMPEntity{{Index: 1, Class: "chassis"}},
		}},
		{IP: "10.44.10.2"},
	}

	got := lifecycleDevices(table, devices, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))

	chassis, psu := table.Records()[1], table.Records()[0]
	require.Equal(t, []LifecycleDevice{{
		IP:   "10.44.10.20",
		Name: "access-2",
		Components: []LifecycleComponent{
			{
				ProductID: "WS-C3750X-48P-S", Name: "1", Class: "chassis", SerialNum: "FDO1234X0AB",
				Status: lifecycle.StatusEndOfSupport, Record: &chassis,
			},
			{
				ProductID: "C3KX-PWR-715WAC", Name: "Switch 1 - Power Supply A", Class: "powerSupply",
				Status: lifecycle.StatusAnnounced, Record: &psu,
			},
			{
				ProductID: "GLC-TE", Name: "GigabitEthernet1/1/1", Class: "module",
				Status: lifecycle.StatusNotListed,
			},
		},
	}}, got)
}

func TestHandleLifecycleDevicesWithoutDiscovery(t *testing.T) {
	t.Parallel()
	s := newLifecycleServer(t)
	w := httptest.NewRecorder()
	s.handleLifecycleDevices(w, httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/lifecycle/devices", http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"records":0,"devices":[]}`, w.Body.String())
}

func newLifecycleFeedServer(t *testing.T) (*Server, *config.Config, string) {
	t.Helper()
	cfg := &config.Config{}
	require.NoError(t, cfg.InitCredentialKeyring(t.TempDir()))
	path := filepath.Join(t.TempDir(), "config.json")
	s := &Server{}
	s.lifecycleFeed = app.NewLifecycleFeed(cfg, path)
	return s, cfg, path
}

func serveLifecycleFeed(t *testing.T, s *Server, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleLifecycleFeed(w, httptest.NewRequest(method, APIVersionPrefix+"/lifecycle/feed", strings.NewReader(body)))
	return w
}

func TestHandleLifecycleFeed(t *testing.T) {
	t.Parallel()
	s, cfg, path := newLifecycleFeedServer(t)

	w := serveLifecycleFeed(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"choice":"unanswered","endpoint":"`+lifecycle.DefaultEndpoint+
		`","tokenUrl":"`+lifecycle.DefaultOAuthURL+`","clientId":"","hasClientSecret":false,"fetched":0}`,
		w.Body.String(), "a fresh install has not been asked and is off")

	w = serveLifecycleFeed(t, s, http.MethodPut, `{"choice":"on"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "needs a client ID and secret")

	w = serveLifecycleFeed(t, s, http.MethodPut,
		`{"choice":"on","clientId":"seed-eox","clientSecret":"hunter22"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "hunter22")
	var resp LifecycleFeedResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, lifecycle.ChoiceOn, resp.Choice)
	require.True(t, resp.HasClientSecret)

	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(saved), "hunter22", "the secret is stored as keyring ciphertext")
	feed := cfg.Lifecycle.EoXFeed
	require.True(t, config.IsEncrypted(feed.ClientSecret))
	require.Empty(t, feed.Endpoint, "Cisco's endpoint is stored as the default, not copied")

	// Answering off keeps the stored secret when none is re-typed.
	w = serveLifecycleFeed(t, s, http.MethodPut, `{"choice":"off","clientId":"seed-eox"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, lifecycle.ChoiceOff, resp.Choice)
	require.True(t, resp.HasClientSecret)

	w = serveLifecycleFeed(t, s, http.MethodPut, `{"choice":"unanswered"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, "the question cannot be un-asked")
}

func TestLifecycleFeedCredentials(t *testing.T) {
	t.Parallel()
	_, cfg, path := newLifecycleFeedServer(t)
	feed := app.NewLifecycleFeed(cfg, path)

	_, on, err := feed.Credentials(context.Background())
	require.NoError(t, err)
	require.False(t, on, "unanswered is off")

	require.NoError(t, feed.Update(lifecycle.FeedUpdate{
		Choice: lifecycle.ChoiceOn, ClientID: "seed-eox", ClientSecret: "hunter22",
	}))
	creds, on, err := feed.Credentials(context.Background())
	require.NoError(t, err)
	require.True(t, on)
	require.Equal(t, lifecycle.Credentials{
		Endpoint: lifecycle.DefaultEndpoint, TokenURL: lifecycle.DefaultOAuthURL,
		ClientID: "seed-eox", ClientSecret: "hunter22",
	}, creds)
}
